// Package buildinfo exposes metadata embedded into the galaxio binary.
package buildinfo

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// These values are overwritten by release builds through -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var readBuildInfo = debug.ReadBuildInfo

// Info contains version metadata for the running binary.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Current returns the build metadata compiled into the binary.
func Current() Info {
	info := Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}

	build, ok := readBuildInfo()
	if !ok {
		return info
	}

	if isDefaultVersion(info.Version) && build.Main.Version != "" && build.Main.Version != "(devel)" {
		info.Version = build.Main.Version
	}

	settings := buildSettings(build.Settings)
	if info.Commit == "none" {
		info.Commit = settings["vcs.revision"]
		if len(info.Commit) > 12 {
			info.Commit = info.Commit[:12]
		}
		if info.Commit == "" {
			info.Commit = "none"
		}
	}

	if info.Date == "unknown" && settings["vcs.time"] != "" {
		info.Date = settings["vcs.time"]
	}

	return info
}

// String formats build metadata for human-readable CLI output.
func (i Info) String() string {
	return fmt.Sprintf("galaxio %s (commit %s, built %s)", i.CleanVersion(), i.Commit, i.Date)
}

// CleanVersion returns Version without the common git tag prefix.
func (i Info) CleanVersion() string {
	return strings.TrimPrefix(i.Version, "v")
}

func isDefaultVersion(version string) bool {
	return version == "" || version == "dev"
}

func buildSettings(settings []debug.BuildSetting) map[string]string {
	result := make(map[string]string, len(settings))
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}
	return result
}
