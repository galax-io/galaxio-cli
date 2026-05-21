package featureflags

import (
	"fmt"
	"os"
	"strconv"
)

// Flag describes an environment-gated experimental feature.
type Flag struct {
	Command string
	EnvVar  string
}

var (
	// Generate gates the experimental generate command.
	Generate = Flag{
		Command: "generate",
		EnvVar:  "GALAXIO_FEATURE_GENERATE",
	}

	commandFlags = map[string]Flag{
		Generate.Command: Generate,
	}
)

// Enabled reports whether a feature flag is enabled through its environment variable.
func Enabled(flag Flag) bool {
	value := os.Getenv(flag.EnvVar)
	enabled, err := strconv.ParseBool(value)
	return err == nil && enabled
}

// LookupCommandFlag returns the feature flag assigned to an experimental command.
func LookupCommandFlag(command string) (Flag, bool) {
	flag, ok := commandFlags[command]
	return flag, ok
}

// DisabledCommandError explains how to enable a gated command.
func DisabledCommandError(flag Flag) error {
	return fmt.Errorf("%q is an experimental command and is currently disabled; enable it with %s=true", flag.Command, flag.EnvVar)
}
