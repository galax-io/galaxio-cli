package main

import (
	"fmt"

	"github.com/galax-io/galaxio-cli/internal/featureflags"
	"github.com/spf13/cobra"
)

func newGenerateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Experimental project generation commands.",
		Long:  "Experimental project generation commands.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RuntimeError{Err: fmt.Errorf("generate is enabled but not implemented yet")}
		},
	}
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
