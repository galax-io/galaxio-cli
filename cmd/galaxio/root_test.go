package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/buildinfo"
	"github.com/galax-io/galaxio-cli/internal/selfupdate"
	"github.com/galax-io/galaxio-cli/internal/templatecatalog"
	"github.com/spf13/cobra"
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
	for _, want := range []string{"Enterprise command-line toolkit", "Usage:", "galaxio [flags]", "completion", "doctor", "template", "update", "version", "generate"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, stdout)
		}
	}
}

func TestGenerateCommandIsRegistered(t *testing.T) {
	code, stdout, stderr := runCLI("--help")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "generate") {
		t.Fatalf("expected generate in help output, got %q", stdout)
	}
}

func TestGenerateCommandPrintsHelp(t *testing.T) {
	code, stdout, stderr := runCLI("generate")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if want := "Generate Gatling load-test source files"; !strings.Contains(stdout, want) {
		t.Fatalf("expected generate help output, got %q", stdout)
	}
}

func TestGenerateSubcommandsAreShown(t *testing.T) {
	code, stdout, stderr := runCLI("generate", "--help")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{"swagger", "har", "postman", "--from", "--template", "--dest", "--package"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected generate help to contain %q, got %q", want, stdout)
		}
	}
}

func TestDoctorRejectsArguments(t *testing.T) {
	code, stdout, stderr := runCLI("doctor", "unexpected")

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "unknown command") && !strings.Contains(stderr, "accepts 0 arg") {
		t.Fatalf("expected argument validation error, got %q", stderr)
	}
}

func TestTemplateCommandPrintsHelp(t *testing.T) {
	code, stdout, stderr := runCLI("template", "--help")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{
		"Discover, render, and validate",
		"configure",
		"init",
		"list",
		"validate",
		"clear-cache",
		"galaxio template init gatling/scala-sbt",
		"--set Name=orders",
		"--values ./template-values.yaml",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected template help to contain %q, got %q", want, stdout)
		}
	}
}

func TestTemplateInitHelpShowsExamples(t *testing.T) {
	code, stdout, stderr := runCLI("template", "init", "--help")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{
		"Initialize a project from a template",
		"galaxio template init gatling/scala-sbt",
		"--destination ./perf",
		"--set Name=orders",
		"--values ./template-values.yaml",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected init help to contain %q, got %q", want, stdout)
		}
	}
}

func TestTemplateInitPrintsComingSoon(t *testing.T) {
	registryRoot, _ := writeTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("template", "init", "gatling/scala-sbt", "--registry", "local:"+registryRoot)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if want := "Template gatling/scala-sbt is coming soon"; !strings.Contains(stdout, want) {
		t.Fatalf("expected coming soon output %q, got %q", want, stdout)
	}
}

func TestTemplateValidateRequiresSource(t *testing.T) {
	code, stdout, stderr := runCLI("template", "validate")

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "accepts 1 arg") {
		t.Fatalf("expected argument validation error, got %q", stderr)
	}
}

func TestUpdateCommandRejectsArguments(t *testing.T) {
	code, stdout, stderr := runCLI("update", "unexpected")

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Fatalf("expected argument validation error, got %q", stderr)
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

func TestVersionCommandPrintsJSONOutput(t *testing.T) {
	originalVersion := buildinfo.Version
	buildinfo.Version = "v1.2.3"
	t.Cleanup(func() {
		buildinfo.Version = originalVersion
	})

	code, stdout, stderr := runCLI("version", "--output", "json")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var result struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode json output: %v; output %q", err, stdout)
	}
	if result.Version != "1.2.3" {
		t.Fatalf("expected clean version 1.2.3 in json output, got %#v", result)
	}
	if result.Commit == "" {
		t.Fatalf("expected commit in json output, got %#v", result)
	}
	if result.Date == "" {
		t.Fatalf("expected build date in json output, got %#v", result)
	}
}

func TestVersionCommandTrimsTagPrefix(t *testing.T) {
	originalVersion := buildinfo.Version
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

func TestUpdateResultPrintsJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	err := printUpdateResult(cmd, selfupdate.Result{
		CurrentVersion: "0.1.0",
		TargetVersion:  "0.2.0",
		DryRun:         true,
		AssetName:      "galaxio_0.2.0_darwin_arm64.tar.gz",
	}, outputJSON)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var result struct {
		CurrentVersion string `json:"currentVersion"`
		TargetVersion  string `json:"targetVersion"`
		Updated        bool   `json:"updated"`
		DryRun         bool   `json:"dryRun"`
		AssetName      string `json:"assetName"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode json output: %v; output %q", err, stdout.String())
	}
	if result.CurrentVersion != "0.1.0" {
		t.Fatalf("expected current version 0.1.0, got %q", result.CurrentVersion)
	}
	if result.TargetVersion != "0.2.0" {
		t.Fatalf("expected target version 0.2.0, got %q", result.TargetVersion)
	}
	if !result.DryRun {
		t.Fatalf("expected dry run result, got %#v", result)
	}
	if result.Updated {
		t.Fatalf("expected update not installed during dry run, got %#v", result)
	}
	if result.AssetName != "galaxio_0.2.0_darwin_arm64.tar.gz" {
		t.Fatalf("expected asset name, got %q", result.AssetName)
	}
}

func TestUpdateResultPrintsTextBranches(t *testing.T) {
	tests := []struct {
		name   string
		result selfupdate.Result
		want   string
	}{
		{
			name: "updated",
			result: selfupdate.Result{
				CurrentVersion: "0.1.0",
				TargetVersion:  "0.2.0",
				Updated:        true,
			},
			want: "Updated galaxio from 0.1.0 to 0.2.0",
		},
		{
			name: "dry run no update",
			result: selfupdate.Result{
				CurrentVersion: "0.2.0",
				DryRun:         true,
			},
			want: "galaxio is already up to date (0.2.0)",
		},
		{
			name: "no update",
			result: selfupdate.Result{
				CurrentVersion: "0.2.0",
			},
			want: "galaxio is already up to date (0.2.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&stdout)

			err := printUpdateResult(cmd, tt.result, outputText)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !strings.Contains(stdout.String(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, stdout.String())
			}
		})
	}
}

func TestConfigPathUsesEnvironmentOverride(t *testing.T) {
	t.Setenv(configEnv, "/tmp/galaxio-test-config.yaml")

	path, err := configPath()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path != "/tmp/galaxio-test-config.yaml" {
		t.Fatalf("expected env config path, got %q", path)
	}
}

func TestConfigPathUsesHomeDirectory(t *testing.T) {
	t.Setenv(configEnv, "")
	t.Setenv("HOME", "/tmp/galaxio-home")

	path, err := configPath()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path != "/tmp/galaxio-home/.galaxio/config.yaml" {
		t.Fatalf("expected home config path, got %q", path)
	}
}

func TestWriteTemplateListCoversVersionFallbacks(t *testing.T) {
	var stdout bytes.Buffer
	err := writeTemplateList(&stdout, []templatecatalog.TemplateRef{
		{Name: "examples/service", Pack: "examples", PackVersion: "0.1.0", Version: "1.2.3", Source: "local:/tmp/templates"},
		{Name: "gatling/scala-sbt", Pack: "gatling", PackVersion: "0.2.0", Placeholder: true, Source: "github:galax-io/templates-gatling"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	for _, want := range []string{
		"Pack: examples",
		"Pack version: 0.1.0",
		"Source: local:/tmp/templates",
		"service",
		"1.2.3",
		"Pack: gatling",
		"Pack version: 0.2.0",
		"Source: github:galax-io/templates-gatling",
		"scala-sbt",
		"coming soon",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected output to contain %q, got %q", want, stdout.String())
		}
	}
}

func TestGroupTemplatesByPackKeepsOrderAndVersions(t *testing.T) {
	groups := groupTemplatesByPack([]templatecatalog.TemplateRef{
		{Name: "examples/service", Pack: "examples", PackVersion: "0.1.0", Source: "local:/tmp/examples"},
		{Name: "examples/worker", Pack: "examples", PackVersion: "0.1.0", Source: "local:/tmp/examples"},
		{Name: "gatling/scala-sbt", Pack: "gatling", PackVersion: "0.2.0", Source: "github:galax-io/templates-gatling"},
	})

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Pack != "examples" || groups[0].PackVersion != "0.1.0" {
		t.Fatalf("unexpected first group %#v", groups[0])
	}
	if len(groups[0].Templates) != 2 {
		t.Fatalf("expected 2 templates in first group, got %d", len(groups[0].Templates))
	}
	if groups[1].Pack != "gatling" || groups[1].Source != "github:galax-io/templates-gatling" {
		t.Fatalf("unexpected second group %#v", groups[1])
	}
}

func TestTemplateDisplayNameAndVersionFallbacks(t *testing.T) {
	if got := templateDisplayName(templatecatalog.TemplateRef{Name: "gatling/scala-sbt"}); got != "scala-sbt" {
		t.Fatalf("expected short template name, got %q", got)
	}
	if got := templateDisplayName(templatecatalog.TemplateRef{Name: "standalone"}); got != "standalone" {
		t.Fatalf("expected unchanged template name, got %q", got)
	}

	if got := templateListVersion(templatecatalog.TemplateRef{Version: "1.2.3", PackVersion: "0.1.0"}); got != "1.2.3" {
		t.Fatalf("expected template version, got %q", got)
	}
	if got := templateListVersion(templatecatalog.TemplateRef{Placeholder: true}); got != "coming soon" {
		t.Fatalf("expected coming soon placeholder, got %q", got)
	}
	if got := templateListVersion(templatecatalog.TemplateRef{PackVersion: "0.1.0"}); got != "0.1.0" {
		t.Fatalf("expected pack version fallback, got %q", got)
	}
}

func TestEncodeJSONRejectsUnsupportedValue(t *testing.T) {
	_, err := encodeJSON(func() {})
	if err == nil {
		t.Fatal("expected json error")
	}
}

func TestUpdateRejectsUnsupportedOutputBeforeRunning(t *testing.T) {
	code, stdout, stderr := runCLI("update", "--output", "xml")

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, `unsupported output format "xml"`) {
		t.Fatalf("expected unsupported output error, got %q", stderr)
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

func TestVerboseEmitsLogToStderr(t *testing.T) {
	registryRoot, _ := writeTemplateCatalogFixture(t)

	code, _, stderr := runCLI("--verbose", "template", "list", "--registry", "local:"+registryRoot)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if !strings.Contains(stderr, "[verbose]") {
		t.Fatalf("expected verbose log on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, "using registry source:") {
		t.Fatalf("expected registry source in verbose log, got %q", stderr)
	}
}

func TestQuietSuppressesTextOutput(t *testing.T) {
	registryRoot, _ := writeTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("--quiet", "doctor", "--registry", "local:"+registryRoot)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected quiet mode to suppress text output, got %q", stdout)
	}
}

func TestQuietDoesNotSuppressJSONOutput(t *testing.T) {
	registryRoot, _ := writeTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("--quiet", "doctor", "--registry", "local:"+registryRoot, "--output", "json")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if !strings.Contains(stdout, `"status":"ok"`) {
		t.Fatalf("expected JSON output even in quiet mode, got %q", stdout)
	}
}

func TestNoColorFlagIsAccepted(t *testing.T) {
	code, _, stderr := runCLI("--no-color", "version")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
}

func TestNoColorEnvironmentVariable(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	code, _, stderr := runCLI("version")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
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
		{name: "unknown defaults to runtime", err: errors.New("unknown command"), want: exitRuntime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCode(tt.err); got != tt.want {
				t.Fatalf("expected exit code %d, got %d", tt.want, got)
			}
		})
	}
}
