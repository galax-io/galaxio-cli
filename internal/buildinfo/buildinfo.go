// Package buildinfo exposes metadata embedded into the galaxio binary at build
// time.
package buildinfo

import "fmt"
import "strings"

// These values are overwritten by release builds through -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info contains version metadata for the running binary.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Current returns the build metadata compiled into the binary.
func Current() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}

// String formats build metadata for human-readable CLI output.
func (i Info) String() string {
	return fmt.Sprintf("galaxio %s (commit %s, built %s)", i.CleanVersion(), i.Commit, i.Date)
}

// CleanVersion returns Version without the common git tag prefix.
func (i Info) CleanVersion() string {
	return strings.TrimPrefix(i.Version, "v")
}
