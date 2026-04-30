package main

import (
	"errors"
	"fmt"
	"io"

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
	var output string

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
			if err := validateOutputFormat(output); err != nil {
				return err
			}

			templates, err := (templatecatalog.SourceFetcher{}).ListTemplates(cmd.Context(), registry)
			if err != nil {
				return RuntimeError{Err: err}
			}

			if output == outputJSON {
				if err := writeJSON(cmd.OutOrStdout(), templates); err != nil {
					return RuntimeError{Err: err}
				}
				return nil
			}

			if err := writeTemplateList(cmd.OutOrStdout(), templates); err != nil {
				return RuntimeError{Err: err}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&registry, "registry", templatecatalog.DefaultRegistrySource, "template registry source")
	cmd.Flags().StringVarP(&output, "output", "o", outputText, "output format: text or json")

	return cmd
}

func newTemplateInitCommand() *cobra.Command {
	var registry string
	var output string

	cmd := &cobra.Command{
		Use:   "init <template>",
		Short: "Initialize a project from a template.",
		Long:  "Initialize a project from a template resolved from a Galaxio template registry.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return UsageError{Err: err}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputFormat(output); err != nil {
				return err
			}
			template, err := (templatecatalog.SourceFetcher{}).FindTemplate(cmd.Context(), registry, args[0])
			if err != nil {
				if errors.Is(err, templatecatalog.ErrTemplateNotFound) {
					return UsageError{Err: err}
				}
				return RuntimeError{Err: err}
			}
			if !template.Placeholder {
				return RuntimeError{Err: fmt.Errorf("template %q is available, but rendering is not implemented yet", template.Name)}
			}

			if output == outputJSON {
				if err := writeJSON(cmd.OutOrStdout(), templateInitOutput{
					Template: template.Name,
					Version:  template.Version,
					Source:   template.Source,
					Status:   templateInitStatusComingSoon,
				}); err != nil {
					return RuntimeError{Err: err}
				}
				return nil
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Template %s %s is coming soon\n", template.Name, template.Version)
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

func newTemplateValidateCommand() *cobra.Command {
	var output string

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
			if err := validateOutputFormat(output); err != nil {
				return err
			}

			source := args[0]
			if err := (templatecatalog.SourceFetcher{}).ValidateSource(cmd.Context(), source); err != nil {
				return RuntimeError{Err: err}
			}

			if output == outputJSON {
				if err := writeJSON(cmd.OutOrStdout(), templateValidateOutput{
					Source: source,
					Status: "valid",
				}); err != nil {
					return RuntimeError{Err: err}
				}
				return nil
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Template pack %s is valid\n", source)
			if err != nil {
				return RuntimeError{Err: err}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", outputText, "output format: text or json")

	return cmd
}

const templateInitStatusComingSoon = "coming_soon"

type templateInitOutput struct {
	Template string `json:"template"`
	Version  string `json:"version"`
	Source   string `json:"source"`
	Status   string `json:"status"`
}

type templateValidateOutput struct {
	Source string `json:"source"`
	Status string `json:"status"`
}

func writeTemplateList(writer io.Writer, templates []templatecatalog.TemplateRef) error {
	for _, template := range templates {
		if template.Description == "" {
			if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\n", template.Name, template.Version, template.Source); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", template.Name, template.Version, template.Source, template.Description); err != nil {
			return err
		}
	}

	return nil
}
