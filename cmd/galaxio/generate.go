package main

import (
	"strings"

	"github.com/galax-io/galaxio-cli/internal/featureflags"
	"github.com/spf13/cobra"
)

type generateOptions struct {
	from        string
	template    string
	dest        string
	pkg         string
	ifExists    string
	ifExistsSet bool
	registry    string
	valuesFile  string
	values      []string
	init        bool
	// includeStatic is only used by `generate har`.
	includeStatic bool
}

func newGenerateCommand() *cobra.Command {
	opts := &generateOptions{
		template: "scala-sbt",
		dest:     ".",
		pkg:      "com.example.perf",
		ifExists: "suffix",
	}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate code from API specifications.",
		Long:  "Generate Gatling load-test source files from supported API specifications.",
		Example: strings.Join([]string{
			"  galaxio generate swagger --from ./petstore.yaml",
			"  galaxio generate har --from ./recording.har",
			"  galaxio generate postman --from ./collection.json",
			"  galaxio generate swagger --from ./petstore.yaml --dest ./out",
			"  galaxio generate swagger --from ./petstore.yaml --package org.example.performance --output json",
		}, "\n"),
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

	cmd.PersistentFlags().StringVar(&opts.from, "from", "", "path to the source Swagger file")
	cmd.PersistentFlags().StringVar(&opts.template, "template", opts.template, "Gatling DSL template pack")
	cmd.PersistentFlags().StringVar(&opts.dest, "dest", opts.dest, "output directory")
	cmd.PersistentFlags().StringVar(&opts.pkg, "package", opts.pkg, "base package for generated code")
	cmd.PersistentFlags().StringVar(&opts.ifExists, "if-exists", opts.ifExists, "conflict strategy: suffix, merge, skip, overwrite")
	cmd.PersistentFlags().StringVar(&opts.registry, "registry", "", "template registry source used by --init")
	cmd.PersistentFlags().StringVar(&opts.valuesFile, "values", "", "YAML file with template values used by --init")
	cmd.PersistentFlags().StringArrayVar(&opts.values, "set", nil, "template value in Key=Value form used by --init")
	cmd.PersistentFlags().BoolVar(&opts.init, "init", false, "initialize a full project scaffold before overlaying generated files")

	cmd.AddCommand(newGenerateSwaggerCommand(opts))
	cmd.AddCommand(newGenerateHARCommand(opts))
	cmd.AddCommand(newGeneratePostmanCommand(opts))

	return cmd
}

func validateExperimentalCommand(args []string) error {
	command := firstCommandArg(args)
	if command == "" {
		return nil
	}

	flag, ok := featureflags.LookupCommandFlag(command)
	if !ok || featureflags.Enabled(flag) {
		return nil
	}

	return UsageError{Err: featureflags.DisabledCommandError(flag)}
}

func firstCommandArg(args []string) string {
	for _, arg := range args {
		if arg == "--" {
			return ""
		}
		if arg == "help" {
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			continue
		}
		return arg
	}
	return ""
}
