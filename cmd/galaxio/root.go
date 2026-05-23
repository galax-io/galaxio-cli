// Package main contains the galaxio command-line application.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

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
		Args:          cobra.NoArgs,
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
	cmd.AddCommand(newTemplateCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newVersionCommand())

	return cmd
}

func execute(args []string, stdout io.Writer, stderr io.Writer) int {
	cmd := newRootCommand()
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	err := cmd.Execute()
	err = normalizeCLIError(err)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %s\n", err)
	}

	return exitCode(err)
}

func buildVersionString() string {
	return versionInfo().CleanVersion()
}

func normalizeCLIError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "unknown command") {
		return UsageError{Err: err}
	}
	return err
}
