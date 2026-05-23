package main

import (
	"context"
	"fmt"
	"os"

	"github.com/galax-io/galaxio-cli/internal/codegen/parser"
	"github.com/spf13/cobra"
)

type generateHAROutput struct {
	Template         string   `json:"template"`
	Source           string   `json:"source"`
	Destination      string   `json:"destination"`
	Package          string   `json:"package"`
	IfExists         string   `json:"ifExists"`
	Warnings         []string `json:"warnings,omitempty"`
	Init             bool     `json:"init"`
	IncludeStatic    bool     `json:"includeStatic"`
	Actions          int      `json:"actions"`
	Scenarios        int      `json:"scenarios"`
	BodyFiles        int      `json:"bodyFiles"`
	FilesWritten     int      `json:"filesWritten"`
	FilesSkipped     int      `json:"filesSkipped"`
	FilesConflicted  int      `json:"filesConflicted"`
	FilesOverwritten int      `json:"filesOverwritten"`
}

func newGenerateHARCommand(opts *generateOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "har",
		Short: "Generate Gatling scripts from HAR 1.2 recordings.",
		Long:  "Generate Scala/sbt Gatling source files from a browser-exported HAR 1.2 recording.",
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

			result, err := runGenerateHAR(cmd.Context(), *opts)
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
	cmd.Flags().BoolVar(&opts.includeStatic, "include-static", false, "include static resources such as images, CSS, JS, and fonts")

	return cmd
}

func runGenerateHAR(ctx context.Context, opts generateOptions) (generateHAROutput, error) {
	payload, err := os.ReadFile(opts.from)
	if err != nil {
		return generateHAROutput{}, RuntimeError{Err: fmt.Errorf("read har input: %w", err)}
	}

	summary, err := runGenerateSpec(ctx, opts, payload, func(ctx context.Context, payload []byte) (*generateExecutionSummary, error) {
		spec, err := parser.NewHARParser(opts.includeStatic).Parse(ctx, payload)
		if err != nil {
			return nil, RuntimeError{Err: fmt.Errorf("parse har input: %w", err)}
		}
		return renderAndWriteSpec(ctx, opts, spec)
	})
	if err != nil {
		return generateHAROutput{}, err
	}

	return generateHAROutput{
		Template:         opts.template,
		Source:           opts.from,
		Destination:      summary.destination,
		Package:          opts.pkg,
		IfExists:         opts.ifExists,
		Warnings:         summary.warnings,
		Init:             opts.init,
		IncludeStatic:    opts.includeStatic,
		Actions:          summary.actions,
		Scenarios:        summary.scenarios,
		BodyFiles:        summary.bodyFiles,
		FilesWritten:     summary.written,
		FilesSkipped:     summary.skipped,
		FilesConflicted:  summary.conflicted,
		FilesOverwritten: summary.overwritten,
	}, nil
}

func runGenerateSpec(ctx context.Context, opts generateOptions, payload []byte, run func(context.Context, []byte) (*generateExecutionSummary, error)) (*generateExecutionSummary, error) {
	return run(ctx, payload)
}
