package main

import (
	"fmt"

	"github.com/galax-io/galaxio-cli/internal/buildinfo"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	var output string

	cmd := &cobra.Command{
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
			if err := validateOutputFormat(output); err != nil {
				return err
			}

			info := versionInfo()
			if output == outputJSON {
				if err := writeJSON(cmd.OutOrStdout(), versionOutput{
					Version: info.CleanVersion(),
					Commit:  info.Commit,
					Date:    info.Date,
				}); err != nil {
					return RuntimeError{Err: err}
				}
				return nil
			}

			_, err := fmt.Fprintln(cmd.OutOrStdout(), info.String())
			if err != nil {
				return RuntimeError{Err: err}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", outputText, "output format: text or json")

	return cmd
}

type versionOutput struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func versionInfo() buildinfo.Info {
	return buildinfo.Current()
}
