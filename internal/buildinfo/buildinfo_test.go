package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestCurrentReturnsBuildVariables(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	Version = "v1.2.3"
	Commit = "abc1234"
	Date = "2026-04-29T12:00:00Z"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, false
	}

	got := Current()

	if got.Version != Version {
		t.Fatalf("expected version %q, got %q", Version, got.Version)
	}
	if got.Commit != Commit {
		t.Fatalf("expected commit %q, got %q", Commit, got.Commit)
	}
	if got.Date != Date {
		t.Fatalf("expected date %q, got %q", Date, got.Date)
	}
}

func TestCurrentFallsBackToModuleVersion(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	Version = "dev"
	Commit = "none"
	Date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v1.2.3"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef1234567890"},
				{Key: "vcs.time", Value: "2026-04-29T12:00:00Z"},
			},
		}, true
	}

	got := Current()

	if got.Version != "v1.2.3" {
		t.Fatalf("expected module version fallback, got %q", got.Version)
	}
	if got.Commit != "abcdef123456" {
		t.Fatalf("expected shortened vcs revision, got %q", got.Commit)
	}
	if got.Date != "2026-04-29T12:00:00Z" {
		t.Fatalf("expected vcs time fallback, got %q", got.Date)
	}
	if want := "galaxio 1.2.3 (commit abcdef123456, built 2026-04-29T12:00:00Z)"; got.String() != want {
		t.Fatalf("expected %q, got %q", want, got.String())
	}
}

func TestCurrentKeepsDevVersionForLocalBuilds(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
		readBuildInfo = originalReadBuildInfo
	})

	Version = "dev"
	Commit = "none"
	Date = "unknown"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "(devel)"},
		}, true
	}

	got := Current()

	if got.Version != "dev" {
		t.Fatalf("expected dev version for local builds, got %q", got.Version)
	}
}

func TestInfoString(t *testing.T) {
	info := Info{
		Version: "1.2.3",
		Commit:  "abc1234",
		Date:    "2026-04-29T12:00:00Z",
	}

	want := "galaxio 1.2.3 (commit abc1234, built 2026-04-29T12:00:00Z)"
	if got := info.String(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestInfoStringTrimsVersionPrefix(t *testing.T) {
	info := Info{
		Version: "v1.2.3",
		Commit:  "abc1234",
		Date:    "2026-04-29T12:00:00Z",
	}

	want := "galaxio 1.2.3 (commit abc1234, built 2026-04-29T12:00:00Z)"
	if got := info.String(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
