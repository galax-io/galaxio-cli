package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateListWithLocalRegistry(t *testing.T) {
	registryRoot, packRoot := writeTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("template", "list", "--registry", "local:"+registryRoot)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{"gatling/scala-sbt", "coming soon", "local:" + packRoot, "Gatling Scala project with sbt"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected output to contain %q, got %q", want, stdout)
		}
	}
}

func TestTemplateListReportsMissingRegistry(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	code, stdout, stderr := runCLI("template", "list", "--registry", "local:"+missing)

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d", exitRuntime, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, want := range []string{"read template registry", "galaxio-registry.yaml", "not found"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected error to contain %q, got %q", want, stderr)
		}
	}
}

func TestTemplateValidateWithLocalPack(t *testing.T) {
	_, packRoot := writeTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("template", "validate", "local:"+packRoot)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if want := "Template pack local:" + packRoot + " is valid"; !strings.Contains(stdout, want) {
		t.Fatalf("expected output to contain %q, got %q", want, stdout)
	}
}

func TestTemplateValidateReportsInvalidPack(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "galaxio-pack.yaml", `apiVersion: galaxio.io/v1
kind: WrongKind
name: gatling
version: 0.1.0
templates: []
`)

	code, stdout, stderr := runCLI("template", "validate", "local:"+root)

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d", exitRuntime, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, want := range []string{"validate template pack", "pack kind must be TemplatePack"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected error to contain %q, got %q", want, stderr)
		}
	}
}

func TestDoctorWithLocalRegistry(t *testing.T) {
	registryRoot, _ := writeTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("doctor", "--registry", "local:"+registryRoot)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{"OK template registry: local:" + registryRoot, "OK template entries: 1"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected output to contain %q, got %q", want, stdout)
		}
	}
}

func TestDoctorReportsRegistryErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	code, stdout, stderr := runCLI("doctor", "--registry", "local:"+missing)

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d", exitRuntime, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, want := range []string{"read template registry", "not found"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected error to contain %q, got %q", want, stderr)
		}
	}
}

func TestTemplateInitRequiresTemplateName(t *testing.T) {
	code, stdout, stderr := runCLI("template", "init")

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

func writeTemplateCatalogFixture(t *testing.T) (string, string) {
	t.Helper()

	packRoot := t.TempDir()
	writeTestFile(t, packRoot, "galaxio-pack.yaml", `apiVersion: galaxio.io/v1
kind: TemplatePack
name: gatling
version: 0.1.0
description: Gatling performance testing templates
templates:
  - name: scala-sbt
    description: Gatling Scala project with sbt
`)

	registryRoot := t.TempDir()
	writeTestFile(t, registryRoot, "galaxio-registry.yaml", `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: gatling
    source: local:`+packRoot+`
    description: Gatling performance testing templates
`)

	return registryRoot, packRoot
}

func writeTestFile(t *testing.T, root string, name string, content string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
