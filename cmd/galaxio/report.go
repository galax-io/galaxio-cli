package main

import "github.com/spf13/cobra"

func newReportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Report on finished load-test runs.",
		Long: `Report on finished load-test runs.

This command reserves the reporting namespace. Operational subcommands are introduced separately.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return UsageError{Err: err}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}
