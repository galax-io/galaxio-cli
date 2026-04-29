package buildinfo

import "testing"

func TestCurrentReturnsBuildVariables(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
	})

	Version = "v1.2.3"
	Commit = "abc1234"
	Date = "2026-04-29T12:00:00Z"

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
