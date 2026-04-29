package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/buildinfo"
)

func runCLI(args ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := execute(args, &stdout, &stderr)

	return code, stdout.String(), stderr.String()
}

func TestHelpPrintsMinimalUsage(t *testing.T) {
	code, stdout, stderr := runCLI("--help")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{"Enterprise command-line toolkit", "Usage:", "galaxio [flags]", "completion", "version"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, stdout)
		}
	}
}

func TestRootWithoutArgsPrintsHelp(t *testing.T) {
	code, stdout, stderr := runCLI()

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("expected help output, got %q", stdout)
	}
}

func TestVersionCommandPrintsBuildInfo(t *testing.T) {
	code, stdout, stderr := runCLI("version")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if want := "galaxio dev (commit none, built unknown)"; !strings.Contains(stdout, want) {
		t.Fatalf("expected version output to contain %q, got %q", want, stdout)
	}
}

func TestVersionCommandTrimsTagPrefix(t *testing.T) {
	originalVersion := versionInfo().Version
	buildinfo.Version = "v1.2.3"
	t.Cleanup(func() {
		buildinfo.Version = originalVersion
	})

	code, stdout, stderr := runCLI("version")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if want := "galaxio 1.2.3"; !strings.Contains(stdout, want) {
		t.Fatalf("expected version output to contain %q, got %q", want, stdout)
	}
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	code, stdout, stderr := runCLI("--version")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if want := "galaxio version dev"; !strings.Contains(stdout, want) {
		t.Fatalf("expected version flag output to contain %q, got %q", want, stdout)
	}
}

func TestUnknownCommandReturnsUsageExitCode(t *testing.T) {
	code, stdout, stderr := runCLI("missing")

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Fatalf("expected unknown command error, got %q", stderr)
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("expected no usage dump on error, got %q", stderr)
	}
}

func TestVerboseAndQuietAreMutuallyExclusive(t *testing.T) {
	code, stdout, stderr := runCLI("--verbose", "--quiet", "version")

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--verbose and --quiet cannot be used together") {
		t.Fatalf("expected mutual exclusion error, got %q", stderr)
	}
}

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "ok", err: nil, want: exitOK},
		{name: "usage", err: UsageError{Err: errors.New("bad args")}, want: exitUsage},
		{name: "runtime", err: RuntimeError{Err: errors.New("boom")}, want: exitRuntime},
		{name: "unknown defaults to usage", err: errors.New("unknown command"), want: exitUsage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCode(tt.err); got != tt.want {
				t.Fatalf("expected exit code %d, got %d", tt.want, got)
			}
		})
	}
}
