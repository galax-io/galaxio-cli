package featureflags

import (
	"os"
	"strconv"
)

// Flag describes an environment-gated feature.
type Flag struct {
	Command string
	EnvVar  string
}

// Enabled reports whether a feature flag is enabled through its environment variable.
func Enabled(flag Flag) bool {
	value := os.Getenv(flag.EnvVar)
	enabled, err := strconv.ParseBool(value)
	return err == nil && enabled
}
