package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/galax-io/galaxio-cli/internal/report"
	"github.com/galax-io/parsec/gatling"
	"github.com/galax-io/parsec/gatling/run"
	"github.com/galax-io/parsec/model"
	"github.com/spf13/cobra"
)

// reportTools is the load-testing tools galaxio report can read, in the order
// a usage error lists them. Each later tool is one more entry here.
//
// The name is the library's own constant, not a second spelling of it: it is
// the same string parsec puts in model.Run.Tool, which the report prints, so
// the accepted name and the reported one cannot drift apart.
var reportTools = []string{gatling.Tool}

// reportFormat is a name -o may take and why it cannot be produced yet.
type reportFormat struct {
	name    string
	pending string
}

// reportFormatFlag is the -o value, checked as it is parsed.
//
// Checking here is what puts -o before everything else, which is the whole
// point of the rule: cobra answers --help before it reaches RunE, so a check
// stated there accepts and silently discards the flag on any invocation
// carrying help, reporting success for a format the command cannot produce.
// Flag parsing is the only stage that runs first, whatever else was asked for.
type reportFormatFlag struct {
	value string
}

func (f *reportFormatFlag) String() string { return f.value }

func (f *reportFormatFlag) Type() string { return "formats" }

func (f *reportFormatFlag) Set(value string) error {
	if err := validateReportFormats(value); err != nil {
		return err
	}

	f.value = value

	return nil
}

// reportFormats is what -o may name, in the order a usage error lists them.
// Every one is reserved: the command writes no machine-readable format yet, and
// the first it publishes will be Gatling's own stats.json. An entry leaves this
// table when its milestone implements it.
var reportFormats = []reportFormat{
	{"stats", "it arrives with milestone v0.15.0 Legacy stats.json"},
	{"global_stats", "it arrives with milestone v0.15.0 Legacy stats.json"},
	{"yml", "the OpenNFR YAML report is postponed"},
}

// timeLayout renders an instant the way the report prints it: to the
// millisecond, which is the resolution every source here records. The zone is
// spelled Z07:00 rather than a bare Z so that the layout states the zone rather
// than asserting one: a bare Z is a literal, and would stamp a local time as
// UTC if a caller ever forgot the .UTC() every call site makes.
const timeLayout = "2006-01-02T15:04:05.000Z07:00"

// reportKeyWidth is the width of the report's key column.
const reportKeyWidth = 11

// reportOptions is what runReport needs: the tool the user named, the path they
// gave, the root flags that shape the output, and where to write.
//
// It carries no -o value: every name the flag accepts is reserved, so the flag
// is answered as it is parsed and no format reaches the work.
type reportOptions struct {
	Tool string

	// Path is the path the user gave and PathSet says whether they gave one at
	// all. Both are needed because an argument that is present and empty names
	// no run and is a usage error, while an absent one asks for the default
	// results root — and the empty string cannot say which happened.
	Path    string
	PathSet bool

	Quiet   bool
	Verbose bool
	Stdout  io.Writer
	Stderr  io.Writer
}

// reportOutput is what runReport read: where the run was, what the source said
// about it, the log format, and the summary of what the log held. Every field
// but the summary is a value parsec produced.
type reportOutput struct {
	Location run.Location
	Run      model.Run
	Format   gatling.Format
	Summary  report.Summary
}

func newReportCommand() *cobra.Command {
	var output reportFormatFlag

	cmd := &cobra.Command{
		Use:   "report <tool> [PATH]",
		Short: "Report on finished load-test runs.",
		Long: fmt.Sprintf(`Report on finished load-test runs.

Names the tool that produced the run, reads the run's log once, and reports what
it holds: the tool and version, the log format, the run's identity and start, the
span it covers, and how many requests, groups, virtual-user events and errors it
recorded, requests split into successes and failures.

PATH is a run directory, its simulation.log, or a results root holding run
directories. Without it the Maven and sbt results root target/gatling is
searched, taking the run lastRun.txt names or else the most recently modified.

Tools: %s

The report formats -o can name are reserved for later releases: stats and
global_stats (Gatling's own stats.json and global_stats.json) and yml (the
OpenNFR YAML report).`, strings.Join(reportTools, ", ")),
		// wrapUsageArgs gives this the type the exit code is read from, as it
		// does for every other validator in the tree.
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// -o needs no check here: every name it accepts is reserved and the
			// flag refused the value as it was parsed, before this or the help
			// a bare invocation prints.
			if len(args) == 0 {
				return cmd.Help()
			}

			opts := reportOptions{
				Tool:    args[0],
				Quiet:   isQuiet(cmd),
				Verbose: globalOptsFromCmd(cmd).verbose,
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
			}
			if len(args) == 2 {
				opts.Path, opts.PathSet = args[1], true
			}

			_, err := runReport(cmd.Context(), opts)

			return err
		},
	}

	cmd.Flags().VarP(&output, "output", "o", "report format(s) to produce, comma-separated: stats, global_stats, yml (reserved for later releases)")

	return cmd
}

// pendingFor returns why a report format cannot be produced yet, and whether
// the name is one this command knows at all.
func pendingFor(name string) (string, bool) {
	for _, f := range reportFormats {
		if f.name == name {
			return f.pending, true
		}
	}

	return "", false
}

// validateReportFormats rejects the whole -o list. Every known name is reserved,
// so any value fails, and a script that asks for stats today must fail loudly
// rather than receive something else.
//
// An unknown name outranks a reserved one: it is a typo, and the milestone that
// will deliver the rest of the list does not matter until it is fixed. An empty
// name — from -o "" or from an empty element — is unknown like any other, so
// that -o "" and -o " " cannot take opposite branches.
//
// The error is plain: the flag layer is what turns it into a usage error, so
// that the type is applied once, where every flag failure passes.
func validateReportFormats(list string) error {
	names := strings.Split(list, ",")

	for i, name := range names {
		name = strings.TrimSpace(name)
		names[i] = name

		if _, ok := pendingFor(name); !ok {
			known := make([]string, len(reportFormats))
			for j, f := range reportFormats {
				known[j] = f.name
			}

			return fmt.Errorf("unknown report format %q: known formats: %s", name, strings.Join(known, ", "))
		}
	}

	// Every name is known, and every known name is reserved, so the list fails
	// on the first of them.
	pending, _ := pendingFor(names[0])

	return fmt.Errorf("report format %q is not available yet: %s", names[0], pending)
}

// runReport locates, opens and reads one run. An unsupported tool or any -o
// value is a UsageError; everything that fails while reading a valid invocation
// is a RuntimeError whose message names the path, directory or version at fault.
func runReport(ctx context.Context, opts reportOptions) (reportOutput, error) {
	if !slices.Contains(reportTools, opts.Tool) {
		return reportOutput{}, UsageError{Err: fmt.Errorf("unsupported tool %q: accepted tools: %s", opts.Tool, strings.Join(reportTools, ", "))}
	}

	// An argument that is there and empty names no run. The zero value of every
	// configuration field and every unset shell variable is the empty string,
	// and reading the default results root for one would answer confidently
	// about a run nobody asked for.
	if opts.PathSet && strings.TrimSpace(opts.Path) == "" {
		return reportOutput{}, UsageError{Err: report.ErrNoPath}
	}

	loc, err := report.Locate(opts.Path)
	if err != nil {
		return reportOutput{}, RuntimeError{Err: err}
	}

	src, err := report.Open(loc)
	if err != nil {
		return reportOutput{}, RuntimeError{Err: err}
	}
	defer func() { _ = src.Close() }()

	out := reportOutput{Location: loc, Run: src.Reader.Run(), Format: src.Format}

	// A version newer than any recording is read, and the warning travels in
	// the run description too. It is printed even under --quiet because it is
	// not informational: it qualifies every number below it.
	for _, w := range out.Run.Warnings {
		fmt.Fprintf(opts.Stderr, "report: warning: %s\n", printable(w.String()))
	}

	summary, scanErr := src.Scan(ctx, report.DefaultOptions())
	out.Summary = summary

	// The log is named here, once, so that the failure reads the same whether
	// it travels alone or beside a write that also failed. Wrapping keeps the
	// chain, so a caller can still ask whether the run was merely cut short.
	if scanErr != nil {
		scanErr = fmt.Errorf("%s: %w", loc.Log, scanErr)
	}

	// A log cut short still has a report worth printing: the counts of what it
	// did hold, which is what the run recorded before it was killed.
	var cutShort *gatling.TruncationError
	if !opts.Quiet && (scanErr == nil || errors.As(scanErr, &cutShort)) {
		if _, err := io.WriteString(opts.Stdout, formatReport(out, opts.Verbose)); err != nil {
			// Both failures matter: the write is why the user sees nothing,
			// and scanErr is why the run is not a complete one.
			return out, RuntimeError{Err: errors.Join(fmt.Errorf("writing report: %w", err), scanErr)}
		}
	}

	if scanErr != nil {
		return out, RuntimeError{Err: scanErr}
	}

	return out, nil
}

// printable renders free text a log supplied. A value that is valid UTF-8 and
// carries no control character is printed as it stands; anything else is
// quoted, which is what parsec does with the bytes it quotes back.
//
// The log is rarely the reader's own — it arrives as a CI artifact or a shared
// results archive — and its identity fields are free text the tool wrote
// verbatim. A carriage return in one erases the line already written and shows
// a name the file does not contain; an escape sequence colours or moves the
// terminal; and the binary format length-prefixes its strings, so a newline
// there ends the line and forges another inside a block that is read by
// position.
func printable(s string) string {
	if utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0 {
		return s
	}

	return strconv.Quote(s)
}

// plural renders a count with its noun, so that a report never says
// "1 requests" about a run that recorded one.
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}

	return fmt.Sprintf("%d %ss", n, noun)
}

// formatReport renders what was read as an aligned block, one fact per line. A
// fact the source did not record is left out rather than shown empty.
func formatReport(out reportOutput, verbose bool) string {
	var b strings.Builder

	tally := out.Summary.Tally

	// The column is one wider than the longest key above, so a key added later
	// keeps the block aligned; TestReportLineWidth pins the two together.
	line := func(key, format string, args ...any) {
		fmt.Fprintf(&b, "%-*s ", reportKeyWidth, key)
		fmt.Fprintf(&b, format, args...)
		fmt.Fprintln(&b)
	}

	// A source that recorded no name or no id is not a source that recorded
	// an empty one: the line goes, as every other absent fact's does.
	if out.Run.Name != "" {
		line("run", "%s", printable(out.Run.Name))
	}

	if out.Run.ID != "" {
		line("id", "%s", printable(out.Run.ID))
	}

	if !out.Run.Start.IsZero() {
		line("started", "%s", out.Run.Start.UTC().Format(timeLayout))
	}

	line("tool", "%s %s", printable(out.Run.Tool), printable(out.Run.ToolVersion))
	line("log", "%s, %s", out.Format, printable(out.Location.Log))
	line("found by", "%s", out.Location.Found)

	if start, ok := tally.Bounds.Start(); ok {
		if end, endOK := tally.Bounds.End(); endOK {
			line("span", "%s .. %s (%s)", start.UTC().Format(timeLayout), end.UTC().Format(timeLayout), end.Sub(start))
		}
	}

	line("requests", "%d (%d ok, %d ko)", tally.Requests, tally.Successes, tally.Failures)

	if tally.Unknown > 0 {
		line("unknown", "%s whose outcome the source lost", plural(tally.Unknown, "request"))
	}

	line("groups", "%s", plural(tally.Groups, "traversal"))
	line("users", "%s", plural(tally.Users, "event"))
	line("errors", "%d", tally.Errors)

	// A declared assertion reaches a reader in one of two places, and both are
	// the run's: a source that writes its payloads ahead of the events puts
	// them on the run description, one that writes them among the events yields
	// them in the stream. Counting only the second said nothing whatever about
	// a Gatling run, which always uses the first.
	if declared := len(out.Run.Assertions) + tally.Assertions; declared > 0 {
		line("assertions", "%s the run declared", plural(declared, "payload"))
	}

	if tally.Other > 0 {
		line("other", "%s of a kind this release does not know", plural(tally.Other, "record"))
	}

	if verbose {
		absent := out.Run.Capabilities.Absent()
		names := make([]string, len(absent))

		for i, f := range absent {
			names[i] = f.String()
		}

		if len(names) > 0 {
			line("absent", "%s", strings.Join(names, ", "))
		}
	}

	return b.String()
}
