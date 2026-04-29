package main

import (
	"fmt"

	"github.com/galax-io/galaxio-cli/internal/buildinfo"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information.",
		Long:  "Print version, commit, and build date information for the galaxio binary.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return UsageError{Err: err}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), versionInfo().String())
			if err != nil {
				return RuntimeError{Err: err}
			}
			return nil
		},
	}
}

func versionInfo() buildinfo.Info {
	return buildinfo.Current()
}
