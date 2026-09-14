package main

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/galax-io/galaxio-cli/internal/report"
	"github.com/galax-io/parsec/gatling/run"
	"github.com/spf13/cobra"
)

// reportTools is the load-testing tools galaxio report can read, in the order
// a usage error lists them. Each later tool is one more entry here.
var reportTools = []string{"gatling"}

// reportOptions is what runReport needs: the tool the user named, the path
// they gave (empty for the default results root), the root flags that shape
// the diagnostics, and where to write.
type reportOptions struct {
	Tool    string
	Path    string
	Quiet   bool
	Verbose bool
	Stdout  io.Writer
	Stderr  io.Writer
}

// reportOutput is what runReport learned: which run it read, how that run was
// chosen, and what it wrote. The records themselves went to Stdout.
type reportOutput struct {
	Dir     string
	Log     string
	Found   run.FoundBy
	Summary report.Summary
}

func newReportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report <tool> [PATH]",
		Short: "Report on finished load-test runs.",
		Long: `Report on finished load-test runs.

Names the tool that produced the run and writes the run as records: one JSON
object per line on standard output, the run header first, then every request,
group, virtual-user event and error in the order the log recorded them.

PATH is a run directory, its simulation.log, or a results root holding run
directories. Without it the Maven and sbt results root target/gatling is
searched, taking the run lastRun.txt names or else the most recently modified.

Tools: gatling`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MaximumNArgs(2)(cmd, args); err != nil {
				return UsageError{Err: err}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
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
				opts.Path = args[1]
			}
			_, err := runReport(cmd.Context(), opts)
			return err
		},
	}

	return cmd
}

// runReport locates, opens and writes one run. An unsupported tool is a
// UsageError; everything that fails while reading a valid invocation is a
// RuntimeError whose message names the path, directory or version at fault.
func runReport(ctx context.Context, opts reportOptions) (reportOutput, error) {
	if !slices.Contains(reportTools, opts.Tool) {
		return reportOutput{}, UsageError{Err: fmt.Errorf("unsupported tool %q: accepted tools: %s", opts.Tool, strings.Join(reportTools, ", "))}
	}

	loc, err := report.Locate(opts.Path)
	if err != nil {
		return reportOutput{}, RuntimeError{Err: err}
	}
	// Which run was read, and by which rule: "newest" is a guess from
	// modification times, and a user who did not name the run should know
	// that is how it was picked.
	if !opts.Quiet {
		fmt.Fprintf(opts.Stderr, "report: reading %s (found by %s)\n", loc.Dir, loc.Found)
	}

	src, err := report.Open(loc)
	if err != nil {
		return reportOutput{}, RuntimeError{Err: err}
	}
	defer src.Close()

	// A version newer than any recording is read, and the warning travels in
	// the header too; it is printed even under --quiet because it is not
	// informational — the numbers it qualifies are on stdout.
	for _, w := range src.Reader.Run().Warnings {
		fmt.Fprintf(opts.Stderr, "report: warning: %s\n", w)
	}

	sum, err := report.Write(ctx, src.Reader, opts.Stdout)
	out := reportOutput{Dir: loc.Dir, Log: loc.Log, Found: loc.Found, Summary: sum}
	if err != nil {
		return out, RuntimeError{Err: err}
	}
	if opts.Verbose {
		fmt.Fprintf(opts.Stderr, "report: %s log, Gatling %s: %d requests, %d groups, %d user events, %d errors\n",
			src.Format, src.Reader.Run().ToolVersion, sum.Requests, sum.Groups, sum.Users, sum.Errors)
	}
	return out, nil
}
