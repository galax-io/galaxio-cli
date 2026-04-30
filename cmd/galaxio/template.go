package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/galax-io/galaxio-cli/internal/templatecatalog"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	cmd.AddCommand(newTemplateConfigureCommand())
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
			registrySource, err := resolveTemplateRegistry(registry)
			if err != nil {
				return RuntimeError{Err: err}
			}

			templates, err := (templatecatalog.SourceFetcher{}).ListTemplates(cmd.Context(), registrySource)
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

	cmd.Flags().StringVar(&registry, "registry", "", "template registry source")
	cmd.Flags().StringVarP(&output, "output", "o", outputText, "output format: text or json")

	return cmd
}

func newTemplateConfigureCommand() *cobra.Command {
	var registry string
	var show bool

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure template registry settings.",
		Long:  "Configure persistent template registry settings for galaxio template commands.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return UsageError{Err: err}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if registry != "" {
				cfg, err := loadConfig()
				if err != nil {
					return RuntimeError{Err: fmt.Errorf("load config: %w", err)}
				}
				cfg.Template.Registry = registry
				if err := saveConfig(cfg); err != nil {
					return RuntimeError{Err: fmt.Errorf("save config: %w", err)}
				}
			}

			if show || registry != "" {
				return printTemplateConfig(cmd, registry)
			}

			return printTemplateConfig(cmd, "")
		},
	}

	cmd.Flags().StringVar(&registry, "registry", "", "template registry source")
	cmd.Flags().BoolVar(&show, "show", false, "show current template configuration")

	return cmd
}

func newTemplateInitCommand() *cobra.Command {
	var registry string
	var destination string
	var output string
	var valuesFile string
	var values []string

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
			registrySource, err := resolveTemplateRegistry(registry)
			if err != nil {
				return RuntimeError{Err: err}
			}
			renderValues, err := loadTemplateValues(valuesFile)
			if err != nil {
				return err
			}
			setValues, err := parseTemplateValues(values)
			if err != nil {
				return err
			}
			for key, value := range setValues {
				renderValues[key] = value
			}

			template, err := (templatecatalog.SourceFetcher{}).FindTemplate(cmd.Context(), registrySource, args[0])
			if err != nil {
				if errors.Is(err, templatecatalog.ErrTemplateNotFound) {
					return UsageError{Err: err}
				}
				return RuntimeError{Err: err}
			}
			if !template.Placeholder {
				result, err := (templatecatalog.SourceFetcher{}).Render(cmd.Context(), templatecatalog.RenderOptions{
					RegistrySource: registrySource,
					TemplateName:   args[0],
					Destination:    destination,
					Values:         renderValues,
				})
				if err != nil {
					return RuntimeError{Err: err}
				}
				return printTemplateInitResult(cmd, result, output)
			}

			result := templatecatalog.RenderResult{
				Template:    template.Name,
				Version:     template.Version,
				PackVersion: template.PackVersion,
				Source:      template.Source,
				Status:      templateInitStatusComingSoon,
			}
			if output == outputJSON {
				if err := writeJSON(cmd.OutOrStdout(), result); err != nil {
					return RuntimeError{Err: err}
				}
				return nil
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Template %s is coming soon\n", template.Name)
			if err != nil {
				return RuntimeError{Err: err}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&registry, "registry", "", "template registry source")
	cmd.Flags().StringVarP(&destination, "destination", "d", ".", "directory to render the template into")
	cmd.Flags().StringVarP(&output, "output", "o", outputText, "output format: text or json")
	cmd.Flags().StringVar(&valuesFile, "values", "", "YAML file with template values")
	cmd.Flags().StringArrayVar(&values, "set", nil, "template value in Key=Value form")

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
	Template    string `json:"template"`
	Version     string `json:"version,omitempty"`
	PackVersion string `json:"packVersion"`
	Source      string `json:"source"`
	Status      string `json:"status"`
}

type templateValidateOutput struct {
	Source string `json:"source"`
	Status string `json:"status"`
}

func printTemplateConfig(cmd *cobra.Command, override string) error {
	path, err := configPath()
	if err != nil {
		return RuntimeError{Err: err}
	}
	registry, err := resolveTemplateRegistry(override)
	if err != nil {
		return RuntimeError{Err: err}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "config: %s\n", path); err != nil {
		return RuntimeError{Err: err}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "registry: %s\n", registry); err != nil {
		return RuntimeError{Err: err}
	}
	return nil
}

func printTemplateInitResult(cmd *cobra.Command, result templatecatalog.RenderResult, output string) error {
	if output == outputJSON {
		return writeJSON(cmd.OutOrStdout(), result)
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Rendered %s to %s (%d files)\n", result.Template, result.Destination, result.Files)
	return err
}

func loadTemplateValues(path string) (map[string]string, error) {
	if path == "" {
		return map[string]string{}, nil
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, UsageError{Err: fmt.Errorf("read values file %q: %w", path, err)}
	}

	values := map[string]any{}
	if err := yaml.Unmarshal(payload, &values); err != nil {
		return nil, UsageError{Err: fmt.Errorf("decode values file %q: %w", path, err)}
	}

	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = fmt.Sprint(value)
	}
	return result, nil
}

func parseTemplateValues(values []string) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for _, value := range values {
		key, raw, ok := strings.Cut(value, "=")
		if !ok || key == "" {
			return nil, UsageError{Err: fmt.Errorf("--set must use Key=Value, got %q", value)}
		}
		result[key] = raw
	}
	return result, nil
}

func writeTemplateList(writer io.Writer, templates []templatecatalog.TemplateRef) error {
	for _, template := range templates {
		version := templateListVersion(template)
		if template.Description == "" {
			if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\n", template.Name, version, template.Source); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", template.Name, version, template.Source, template.Description); err != nil {
			return err
		}
	}

	return nil
}

func templateListVersion(template templatecatalog.TemplateRef) string {
	if template.Version != "" {
		return template.Version
	}
	if template.Placeholder {
		return "coming soon"
	}
	return template.PackVersion
}
