package main

import (
	"fmt"
	"os"

	"github.com/galax-io/galaxio-cli/internal/selfupdate"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	repo    string
	version string
	dryRun  bool
	output  string
}

func newUpdateCommand() *cobra.Command {
	opts := &updateOptions{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update galaxio to a newer release.",
		Long:  "Update galaxio by downloading a verified binary from GitHub Releases.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return UsageError{Err: err}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputFormat(opts.output); err != nil {
				return err
			}

			executable, err := os.Executable()
			if err != nil {
				return RuntimeError{Err: fmt.Errorf("resolve executable path: %w", err)}
			}

			result, err := selfupdate.Updater{}.Run(cmd.Context(), selfupdate.Options{
				Repo:           opts.repo,
				TargetVersion:  opts.version,
				CurrentVersion: versionInfo().CleanVersion(),
				Executable:     executable,
				DryRun:         opts.dryRun,
			})
			if err != nil {
				return RuntimeError{Err: err}
			}

			if err := printUpdateResult(cmd, result, opts.output); err != nil {
				return RuntimeError{Err: err}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.repo, "repo", "galax-io/galaxio-cli", "GitHub repository to update from")
	cmd.Flags().StringVar(&opts.version, "version", "", "target version to install")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "check for an update without installing it")
	cmd.Flags().StringVarP(&opts.output, "output", "o", outputText, "output format: text or json")

	return cmd
}

func printUpdateResult(cmd *cobra.Command, result selfupdate.Result, output string) error {
	if err := validateOutputFormat(output); err != nil {
		return err
	}
	if output == outputJSON {
		return writeJSON(cmd.OutOrStdout(), result)
	}

	if isQuiet(cmd) {
		return nil
	}
	switch {
	case result.Updated:
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Updated galaxio from %s to %s\n", result.CurrentVersion, result.TargetVersion)
		return err
	case result.DryRun && result.AssetName != "":
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Update available: %s -> %s (%s)\n", result.CurrentVersion, result.TargetVersion, result.AssetName)
		return err
	case result.DryRun:
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "galaxio is already up to date (%s)\n", result.CurrentVersion)
		return err
	default:
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "galaxio is already up to date (%s)\n", result.CurrentVersion)
		return err
	}
}
