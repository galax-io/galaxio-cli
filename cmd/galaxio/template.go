package main

import (
	"fmt"

	"github.com/galax-io/galaxio-cli/internal/templatecatalog"
	"github.com/spf13/cobra"
)

func newTemplateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Discover and validate project templates.",
		Long:  "Discover and validate project templates from Galaxio template registries.",
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

	cmd.AddCommand(newTemplateListCommand())
	cmd.AddCommand(newTemplateInitCommand())
	cmd.AddCommand(newTemplateValidateCommand())

	return cmd
}

func newTemplateListCommand() *cobra.Command {
	var registry string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates.",
		Long:  "List templates from a Galaxio template registry.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return UsageError{Err: err}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			packs, err := (templatecatalog.SourceFetcher{}).ListPacks(cmd.Context(), registry)
			if err != nil {
				return RuntimeError{Err: err}
			}

			for _, pack := range packs {
				state := "available"
				if pack.Placeholder {
					state = "coming soon"
				}
				if pack.Description == "" {
					_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", pack.Name, state, pack.Source)
				} else {
					_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", pack.Name, state, pack.Source, pack.Description)
				}
				if err != nil {
					return RuntimeError{Err: err}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&registry, "registry", templatecatalog.DefaultRegistrySource, "template registry source")

	return cmd
}

func newTemplateInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <template>",
		Short: "Initialize a project from a template.",
		Long:  "Initialize a project from a template. Rendering support is not available yet.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return UsageError{Err: err}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Template %s is coming soon\n", args[0])
			if err != nil {
				return RuntimeError{Err: err}
			}
			return nil
		},
	}

	return cmd
}

func newTemplateValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <source>",
		Short: "Validate a template pack.",
		Long:  "Validate a Galaxio template pack and its template manifests.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return UsageError{Err: err}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			if err := (templatecatalog.SourceFetcher{}).ValidateSource(cmd.Context(), source); err != nil {
				return RuntimeError{Err: err}
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Template pack %s is valid\n", source)
			if err != nil {
				return RuntimeError{Err: err}
			}

			return nil
		},
	}

	return cmd
}
