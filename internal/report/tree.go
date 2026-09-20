package report

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/galax-io/parsec/model"
)

// Tree holds the root request summary and one owned node per recorded position.
// Nodes are in first-observation order, with implicit ancestors before children.
// Its memory depends on distinct positions, not the number of log records.
type Tree struct {
	Summary   Summary
	Nodes     []*Node
	positions map[model.Position]*Node
	options   Options
}

// Node summarizes only its own requests or group traversals, never descendants.
// The zero Parent denotes the root. Position owns all retained name storage.
type Node struct {
	Position model.Position
	Parent   model.Position
	stats    Summary
}

// Node looks up a structured position without conflating names or kinds.
func (t *Tree) Node(position model.Position) (*Node, bool) {
	node, ok := t.positions[position]
	return node, ok
}

// Export validates this population using the same whole-run rate denominator
// as the root. Implicit groups have empty columns, not invented traversals.
func (n *Node) Export(bounds model.Bounds) (ExportStats, error) {
	summary := n.stats
	summary.Tally.Bounds = bounds
	view, err := summary.Export()
	if err != nil {
		return ExportStats{}, fmt.Errorf("position %s: %w", n.Position, err)
	}
	return view, nil
}

func newTree(opts Options) (*Tree, error) {
	opts, err := opts.Normalize()
	if err != nil {
		return nil, err
	}
	if len(opts.Percentiles) != 4 {
		return nil, errors.New("legacy statistics require exactly four normalized percentile ranks")
	}
	return &Tree{positions: make(map[model.Position]*Node), options: opts}, nil
}

func (t *Tree) ensure(position, parent model.Position) *Node {
	if node, ok := t.positions[position]; ok {
		return node
	}
	node := &Node{
		Position: position, Parent: parent,
		stats: Summary{Options: t.options},
	}
	t.positions[position] = node
	t.Nodes = append(t.Nodes, node)
	return node
}

func (t *Tree) ancestors(groups []string) model.Position {
	var parent model.Position
	for depth := range groups {
		position := model.NewGroupPosition(groups[:depth+1])
		t.ensure(position, parent)
		parent = position
	}
	return parent
}

func (t *Tree) collect(item *model.Item) error {
	var position model.Position
	var groups []string
	var duration, wall model.Opt[time.Duration]
	var outcome model.Outcome
	switch item.Kind {
	case model.ItemSample:
		sample := &item.Sample
		position, groups = sample.Position(), sample.Groups
		duration, wall, outcome = sample.Duration, sample.Duration, sample.Outcome
	case model.ItemGroup:
		group := &item.Group
		position, groups = group.Position(), group.Groups
		duration, wall, outcome = group.CumulatedDuration, group.Duration, group.Outcome
		if len(groups) == 0 {
			return errors.New("group path is missing")
		}
	default:
		return nil
	}
	if outcome != model.OutcomeSuccess && outcome != model.OutcomeFailure {
		return fmt.Errorf("position %s: outcome is unknown", position)
	}
	if value, ok := duration.Get(); !ok || value < 0 {
		label := "request duration"
		if item.Kind == model.ItemGroup {
			label = "group cumulated duration"
		}
		return fmt.Errorf("position %s: %s is missing or negative", position, label)
	}
	wallValue, ok := wall.Get()
	if !ok || wallValue < 0 {
		return fmt.Errorf("position %s: group wall duration is missing or negative", position)
	}

	node, exists := t.positions[position]
	if !exists {
		parent := t.ancestors(groups)
		if item.Kind == model.ItemGroup {
			node = t.positions[position]
		} else {
			node = t.ensure(position, parent)
		}
	}
	s := &node.stats
	if s.Tally.Requests == math.MaxInt {
		return fmt.Errorf("position %s: population count overflows", position)
	}
	s.Tally.Requests++
	var ms int64
	if outcome == model.OutcomeSuccess {
		s.Tally.Successes++
		ms, _ = s.OK.add(duration)
		s.band(wallValue.Milliseconds())
	} else {
		s.Tally.Failures++
		ms, _ = s.Failed.add(duration)
	}
	s.estimate(ms)
	return nil
}
