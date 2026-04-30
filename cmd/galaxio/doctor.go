package main

import (
	"fmt"

	"github.com/galax-io/galaxio-cli/internal/templatecatalog"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	var registry string
	var output string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check galaxio configuration and dependencies.",
		Long:  "Check galaxio configuration, template registry access, and template pack manifests.",
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

			fetcher := templatecatalog.SourceFetcher{}
			templates, err := fetcher.ListPacks(cmd.Context(), registry)
			if err != nil {
				return RuntimeError{Err: err}
			}

			if output == outputJSON {
				payload, err := encodeJSON(map[string]interface{}{
					"registry":        registry,
					"status":          "ok",
					"templateEntries": len(templates),
				})
				if err != nil {
					return RuntimeError{Err: err}
				}
				_, err = cmd.OutOrStdout().Write(payload)
				if err != nil {
					return RuntimeError{Err: err}
				}
				return nil
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "OK template registry: %s\n", registry)
			if err != nil {
				return RuntimeError{Err: err}
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "OK template entries: %d\n", len(templates))
			if err != nil {
				return RuntimeError{Err: err}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&registry, "registry", templatecatalog.DefaultRegistrySource, "template registry source")
	cmd.Flags().StringVarP(&output, "output", "o", outputText, "output format: text or json")

	return cmd
}
