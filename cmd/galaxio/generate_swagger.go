package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/galax-io/galaxio-cli/internal/codegen"
	"github.com/galax-io/galaxio-cli/internal/codegen/parser"
	"github.com/galax-io/galaxio-cli/internal/codegen/renderer"
	"github.com/galax-io/galaxio-cli/internal/templatecatalog"
	"github.com/spf13/cobra"
)

const generateTemplateScalaSBT = "scala-sbt"

type generateSwaggerOutput struct {
	Template         string   `json:"template"`
	Source           string   `json:"source"`
	Destination      string   `json:"destination"`
	Package          string   `json:"package"`
	IfExists         string   `json:"ifExists"`
	Warnings         []string `json:"warnings,omitempty"`
	Init             bool     `json:"init"`
	Actions          int      `json:"actions"`
	Scenarios        int      `json:"scenarios"`
	BodyFiles        int      `json:"bodyFiles"`
	FilesWritten     int      `json:"filesWritten"`
	FilesSkipped     int      `json:"filesSkipped"`
	FilesConflicted  int      `json:"filesConflicted"`
	FilesOverwritten int      `json:"filesOverwritten"`
}

type generateExecutionSummary struct {
	destination string
	actions     int
	scenarios   int
	bodyFiles   int
	written     int
	skipped     int
	conflicted  int
	overwritten int
	warnings    []string
}

func newGenerateSwaggerCommand(opts *generateOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "swagger",
		Short: "Generate Gatling scripts from Swagger 2.0.",
		Long:  "Generate Scala/sbt Gatling source files from a Swagger 2.0 specification.",
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
			if err := validateGenerateTemplateOptions(*opts); err != nil {
				return err
			}
			opts.ifExistsSet = cmd.Flags().Changed("if-exists")

			result, err := runGenerateSwagger(cmd.Context(), *opts)
			if err != nil {
				return err
			}
			if err := writeGenerateWarnings(cmd.ErrOrStderr(), result.Warnings); err != nil {
				return RuntimeError{Err: err}
			}

			if output == outputJSON {
				if err := writeJSON(cmd.OutOrStdout(), result); err != nil {
					return RuntimeError{Err: err}
				}
				return nil
			}

			if isQuiet(cmd) {
				return nil
			}

			if _, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"Generated %d action files, %d scenario files, %d body files in %s\nFiles written: %d, skipped: %d, conflicts: %d, overwritten: %d\n",
				result.Actions,
				result.Scenarios,
				result.BodyFiles,
				result.Destination,
				result.FilesWritten,
				result.FilesSkipped,
				result.FilesConflicted,
				result.FilesOverwritten,
			); err != nil {
				return RuntimeError{Err: err}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", outputText, "output format: text or json")

	return cmd
}

func validateGenerateTemplateOptions(opts generateOptions) error {
	if strings.TrimSpace(opts.from) == "" {
		return UsageError{Err: fmt.Errorf("required flag(s) \"from\" not set")}
	}
	if strings.TrimSpace(opts.template) != generateTemplateScalaSBT {
		return UsageError{Err: fmt.Errorf("unknown generate template %q", opts.template)}
	}
	if err := codegen.ValidateIfExistsStrategy(opts.ifExists); err != nil {
		return UsageError{Err: err}
	}
	return nil
}

func runGenerateSwagger(ctx context.Context, opts generateOptions) (generateSwaggerOutput, error) {
	payload, err := os.ReadFile(opts.from)
	if err != nil {
		return generateSwaggerOutput{}, RuntimeError{Err: fmt.Errorf("read swagger input: %w", err)}
	}

	spec, err := parser.NewSwaggerParser().Parse(ctx, payload)
	if err != nil {
		return generateSwaggerOutput{}, RuntimeError{Err: fmt.Errorf("parse swagger input: %w", err)}
	}

	summary, err := renderAndWriteSpec(ctx, opts, spec)
	if err != nil {
		return generateSwaggerOutput{}, err
	}

	return generateSwaggerOutput{
		Template:         opts.template,
		Source:           opts.from,
		Destination:      summary.destination,
		Package:          opts.pkg,
		IfExists:         opts.ifExists,
		Warnings:         summary.warnings,
		Init:             opts.init,
		Actions:          summary.actions,
		Scenarios:        summary.scenarios,
		BodyFiles:        summary.bodyFiles,
		FilesWritten:     summary.written,
		FilesSkipped:     summary.skipped,
		FilesConflicted:  summary.conflicted,
		FilesOverwritten: summary.overwritten,
	}, nil
}

func renderAndWriteSpec(ctx context.Context, opts generateOptions, spec *codegen.Spec) (*generateExecutionSummary, error) {
	renderPackage := opts.pkg
	if opts.init {
		nameWord := codegen.DeriveNameWord(filepath.Base(opts.dest))
		if templateValues, err := resolveGenerateTemplateValues(opts); err == nil {
			if value := strings.TrimSpace(templateValues["NameWord"]); value != "" {
				nameWord = value
			}
		}
		renderPackage = opts.pkg + "." + nameWord
	}

	files, err := renderer.NewRenderer().Render(spec, renderer.RenderOptions{Package: renderPackage})
	if err != nil {
		return nil, RuntimeError{Err: fmt.Errorf("render generated files: %w", err)}
	}

	conflictSummary, err := writeGeneratedProject(ctx, opts, files)
	if err != nil {
		return nil, RuntimeError{Err: fmt.Errorf("write generated files: %w", err)}
	}

	absDest, err := filepath.Abs(opts.dest)
	if err != nil {
		return nil, RuntimeError{Err: fmt.Errorf("resolve destination: %w", err)}
	}

	actions, scenarios, bodyFiles := countGeneratedFiles(files)
	return &generateExecutionSummary{
		destination: absDest,
		actions:     actions,
		scenarios:   scenarios,
		bodyFiles:   bodyFiles,
		written:     conflictSummary.Written,
		skipped:     conflictSummary.Skipped,
		conflicted:  conflictSummary.Conflicted,
		overwritten: conflictSummary.Overwritten,
		warnings:    conflictSummary.Warnings,
	}, nil
}

func writeGeneratedProject(ctx context.Context, opts generateOptions, files []renderer.OutputFile) (writeSummary, error) {
	if !opts.init {
		return writeRenderedFiles(opts.dest, files, opts.ifExists)
	}

	destExists, err := dirHasEntries(opts.dest)
	if err != nil {
		return writeSummary{}, err
	}
	if destExists && !opts.ifExistsSet {
		return writeSummary{}, fmt.Errorf("destination %q already contains files; rerun with --if-exists to allow project regeneration", opts.dest)
	}

	templateValues, err := resolveGenerateTemplateValues(opts)
	if err != nil {
		return writeSummary{}, err
	}
	registrySource, err := resolveTemplateRegistry(opts.registry)
	if err != nil {
		return writeSummary{}, err
	}

	scaffoldDir, err := os.MkdirTemp("", "galaxio-generate-init-*")
	if err != nil {
		return writeSummary{}, err
	}
	defer os.RemoveAll(scaffoldDir)

	_, err = (templatecatalog.SourceFetcher{}).Render(ctx, templatecatalog.RenderOptions{
		RegistrySource: registrySource,
		TemplateName:   templateRefForGenerateTemplate(opts.template),
		Destination:    scaffoldDir,
		Values:         templateValues,
	})
	if err != nil {
		return writeSummary{}, err
	}

	projectFiles, err := collectInitProjectFiles(scaffoldDir, files, templateValues)
	if err != nil {
		return writeSummary{}, err
	}

	return writeRenderedFiles(opts.dest, projectFiles, opts.ifExists)
}

type writeSummary struct {
	Written     int
	Skipped     int
	Conflicted  int
	Overwritten int
	Warnings    []string
}

func writeRenderedFiles(dest string, files []renderer.OutputFile, strategy string) (writeSummary, error) {
	var summary writeSummary
	for _, file := range files {
		target := filepath.Join(dest, file.Path)
		result, err := codegen.WriteFileWithStrategy(target, file.Content, strategy)
		if err != nil {
			return writeSummary{}, err
		}
		switch result.Status {
		case codegen.ConflictStatusWritten:
			summary.Written++
		case codegen.ConflictStatusSkipped:
			summary.Skipped++
			if strategy == codegen.IfExistsSkip {
				summary.Warnings = append(summary.Warnings, fmt.Sprintf("warning: skipped existing file %s", result.Path))
			}
		case codegen.ConflictStatusConflict:
			summary.Conflicted++
		case codegen.ConflictStatusOverwritten:
			summary.Overwritten++
		}
	}
	return summary, nil
}

func writeGenerateWarnings(stderr io.Writer, warnings []string) error {
	for _, warning := range warnings {
		if _, err := fmt.Fprintln(stderr, warning); err != nil {
			return err
		}
	}
	return nil
}

func resolveGenerateTemplateValues(opts generateOptions) (map[string]string, error) {
	values, err := loadTemplateValues(opts.valuesFile)
	if err != nil {
		return nil, err
	}
	setValues, err := parseTemplateValues(opts.values)
	if err != nil {
		return nil, err
	}
	for key, value := range setValues {
		values[key] = value
	}

	if strings.TrimSpace(values["Name"]) == "" {
		values["Name"] = filepath.Base(opts.dest)
	}
	if strings.TrimSpace(values["NameWord"]) == "" {
		values["NameWord"] = codegen.DeriveNameWord(values["Name"])
	}
	values["Package"] = opts.pkg
	values["PackagePath"] = strings.ReplaceAll(opts.pkg, ".", "/")

	return values, nil
}

func templateRefForGenerateTemplate(templateName string) string {
	if strings.Contains(templateName, "/") {
		return templateName
	}
	return "gatling/" + templateName
}

func dirHasEntries(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return len(entries) > 0, nil
}

func collectInitProjectFiles(scaffoldDir string, generated []renderer.OutputFile, values map[string]string) ([]renderer.OutputFile, error) {
	files := make([]renderer.OutputFile, 0, len(generated))

	if err := filepath.WalkDir(scaffoldDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(scaffoldDir, path)
		if err != nil {
			return err
		}
		files = append(files, renderer.OutputFile{
			Path:    filepath.ToSlash(rel),
			Content: payload,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	packagePath := values["PackagePath"]
	nameWord := values["NameWord"]
	for _, file := range generated {
		files = append(files, renderer.OutputFile{
			Path:    overlayGeneratedPath(file.Path, packagePath, nameWord),
			Content: file.Content,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func overlayGeneratedPath(path string, packagePath string, nameWord string) string {
	switch {
	case strings.HasPrefix(path, "cases/"), strings.HasPrefix(path, "scenarios/"):
		return filepath.ToSlash(filepath.Join("src", "test", "scala", packagePath, nameWord, path))
	case strings.HasPrefix(path, "resources/"):
		return filepath.ToSlash(filepath.Join("src", "test", path))
	default:
		return path
	}
}

func countGeneratedFiles(files []renderer.OutputFile) (actions int, scenarios int, bodyFiles int) {
	for _, file := range files {
		switch {
		case strings.HasPrefix(file.Path, "cases/"):
			actions++
		case strings.HasPrefix(file.Path, "scenarios/"):
			scenarios++
		case strings.HasPrefix(file.Path, "resources/bodies/"):
			bodyFiles++
		}
	}
	return actions, scenarios, bodyFiles
}
