// Package main contains the galaxio command-line application.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

type globalOptions struct {
	noColor bool
	verbose bool
	quiet   bool
}

type contextKey struct{}

func globalOptsFromCmd(cmd *cobra.Command) *globalOptions {
	if ctx := cmd.Context(); ctx != nil {
		if opts, ok := ctx.Value(contextKey{}).(*globalOptions); ok {
			return opts
		}
	}
	return &globalOptions{}
}

func isQuiet(cmd *cobra.Command) bool {
	return globalOptsFromCmd(cmd).quiet
}

func verboseLog(cmd *cobra.Command, format string, args ...any) {
	opts := globalOptsFromCmd(cmd)
	if !opts.verbose {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "[verbose] "+format+"\n", args...)
}

// newRootCommand builds the root command for the galaxio CLI.
func newRootCommand() *cobra.Command {
	opts := &globalOptions{}

	cmd := &cobra.Command{
		Use:           "galaxio",
		Short:         "Enterprise command-line toolkit for Galaxio.",
		Long:          "Enterprise command-line toolkit for Galaxio platform workflows.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       buildVersionString(),
		// cobra reports an unknown subcommand through the root's argument
		// validator, so this is where "galaxio bogus" becomes a usage error.
		// wrapUsageArgs below gives this validator, and every other one in the
		// tree, the type the exit code is read from.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.verbose && opts.quiet {
				return UsageError{Err: fmt.Errorf("--verbose and --quiet cannot be used together")}
			}
			if opts.noColor || os.Getenv("NO_COLOR") != "" {
				opts.noColor = true
			}
			cmd.SetContext(context.WithValue(cmd.Context(), contextKey{}, opts))
			return nil
		},
	}

	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return UsageError{Err: err}
	})

	cmd.PersistentFlags().BoolVar(&opts.noColor, "no-color", false, "disable colored output")
	cmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "enable verbose diagnostic output")
	cmd.PersistentFlags().BoolVarP(&opts.quiet, "quiet", "q", false, "suppress non-essential output")

	cmd.AddCommand(newDoctorCommand())
	cmd.AddCommand(newGenerateCommand())
	cmd.AddCommand(newReportCommand())
	cmd.AddCommand(newTemplateCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newVersionCommand())

	// cobra builds the completion commands itself, during Execute, and gives
	// them a bare cobra.NoArgs whose error no code here would otherwise see.
	// Materialising them now is what lets wrapUsageArgs reach them: without it
	// "galaxio completion bash extra" is a usage error that exits like a
	// runtime failure. Both calls are idempotent and cobra repeats them.
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultCompletionCmd()
	wrapUsageArgs(cmd)

	return cmd
}

// usageArgs gives an argument validator's error the type the exit code is read
// from. A validator that already reported a usage error is left as it is, so
// that wrapping a command twice cannot nest one inside another.
func usageArgs(validate cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		err := validate(cmd, args)
		if err == nil {
			return nil
		}

		var usage UsageError
		if errors.As(err, &usage) {
			return err
		}

		return UsageError{Err: err}
	}
}

// wrapUsageArgs applies usageArgs to cmd and everything beneath it, so that a
// bad argument list exits 2 whoever wrote the command — this package or cobra.
//
// A command that sets no validator is left alone: cobra then applies its own
// default, which accepts what the command accepts and reports nothing to
// classify.
func wrapUsageArgs(cmd *cobra.Command) {
	if cmd.Args != nil {
		cmd.Args = usageArgs(cmd.Args)
	}

	for _, sub := range cmd.Commands() {
		wrapUsageArgs(sub)
	}
}

func execute(args []string, stdout io.Writer, stderr io.Writer) int {
	return executeContext(context.Background(), args, stdout, stderr)
}

// executeContext runs the CLI under ctx. Cancelling ctx stops the work a
// command is doing: reading a run is the long one, and it polls ctx as it
// walks. Without a context reaching this far, that poll could never fire and
// the only way to stop a multi-gigabyte read would be to kill the process.
func executeContext(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	cmd := newRootCommand()
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	err := cmd.ExecuteContext(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %s\n", err)
	}

	return exitCode(err)
}

func buildVersionString() string {
	return versionInfo().CleanVersion()
}
