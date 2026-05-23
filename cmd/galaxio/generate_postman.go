package main

import (
	"context"
	"fmt"
	"os"

	"github.com/galax-io/galaxio-cli/internal/codegen/parser"
	"github.com/spf13/cobra"
)

type generatePostmanOutput struct {
	Template         string `json:"template"`
	Source           string `json:"source"`
	Destination      string `json:"destination"`
	Package          string `json:"package"`
	IfExists         string `json:"ifExists"`
	Init             bool   `json:"init"`
	Actions          int    `json:"actions"`
	Scenarios        int    `json:"scenarios"`
	BodyFiles        int    `json:"bodyFiles"`
	FilesWritten     int    `json:"filesWritten"`
	FilesSkipped     int    `json:"filesSkipped"`
	FilesConflicted  int    `json:"filesConflicted"`
	FilesOverwritten int    `json:"filesOverwritten"`
}

func newGeneratePostmanCommand(opts *generateOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "postman",
		Short: "Generate Gatling scripts from Postman collections.",
		Long:  "Generate Scala/sbt Gatling source files from a Postman Collection v2.1 export.",
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

			result, err := runGeneratePostman(cmd.Context(), *opts)
			if err != nil {
				return err
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

func runGeneratePostman(ctx context.Context, opts generateOptions) (generatePostmanOutput, error) {
	payload, err := os.ReadFile(opts.from)
	if err != nil {
		return generatePostmanOutput{}, RuntimeError{Err: fmt.Errorf("read postman input: %w", err)}
	}

	spec, err := parser.NewPostmanParser().Parse(ctx, payload)
	if err != nil {
		return generatePostmanOutput{}, RuntimeError{Err: fmt.Errorf("parse postman input: %w", err)}
	}

	summary, err := renderAndWriteSpec(ctx, opts, spec)
	if err != nil {
		return generatePostmanOutput{}, err
	}

	return generatePostmanOutput{
		Template:         opts.template,
		Source:           opts.from,
		Destination:      summary.destination,
		Package:          opts.pkg,
		IfExists:         opts.ifExists,
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
