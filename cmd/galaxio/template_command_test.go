package main

import (
	"encoding/json"
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
	if strings.Contains(stdout, "\t0.1.0\t") {
		t.Fatalf("expected placeholder template list to omit pack version, got %q", stdout)
	}
}

func TestTemplateListWithJSONOutput(t *testing.T) {
	registryRoot, packRoot := writeTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("template", "list", "--registry", "local:"+registryRoot, "--output", "json")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var refs []struct {
		Name        string `json:"name"`
		Pack        string `json:"pack"`
		PackVersion string `json:"packVersion"`
		Version     string `json:"version"`
		Source      string `json:"source"`
		Description string `json:"description"`
		Templates   int    `json:"templates"`
		Placeholder bool   `json:"placeholder"`
	}
	if err := json.Unmarshal([]byte(stdout), &refs); err != nil {
		t.Fatalf("decode json output: %v; output %q", err, stdout)
	}
	if len(refs) != 1 {
		t.Fatalf("expected one template ref, got %#v", refs)
	}
	ref := refs[0]
	if ref.Name != "gatling/scala-sbt" {
		t.Fatalf("expected template name gatling/scala-sbt, got %q", ref.Name)
	}
	if ref.Pack != "gatling" {
		t.Fatalf("expected pack gatling, got %q", ref.Pack)
	}
	if ref.PackVersion != "0.1.0" {
		t.Fatalf("expected pack version 0.1.0, got %q", ref.PackVersion)
	}
	if ref.Version != "" {
		t.Fatalf("expected placeholder template version to be empty, got %q", ref.Version)
	}
	if ref.Source != "local:"+packRoot {
		t.Fatalf("expected source local:%s, got %q", packRoot, ref.Source)
	}
	if ref.Description != "Gatling Scala project with sbt" {
		t.Fatalf("expected template description, got %q", ref.Description)
	}
	if ref.Templates != 1 {
		t.Fatalf("expected one template in pack, got %d", ref.Templates)
	}
	if !ref.Placeholder {
		t.Fatalf("expected placeholder template, got %#v", ref)
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

func TestTemplateListRejectsUnsupportedOutput(t *testing.T) {
	registryRoot, _ := writeTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("template", "list", "--registry", "local:"+registryRoot, "--output", "xml")

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

func TestTemplateValidateWithJSONOutput(t *testing.T) {
	_, packRoot := writeTemplateCatalogFixture(t)

	source := "local:" + packRoot
	code, stdout, stderr := runCLI("template", "validate", source, "--output", "json")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var result struct {
		Source string `json:"source"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode json output: %v; output %q", err, stdout)
	}
	if result.Source != source {
		t.Fatalf("expected source %q, got %q", source, result.Source)
	}
	if result.Status != "valid" {
		t.Fatalf("expected status valid, got %q", result.Status)
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

func TestTemplateInitWithJSONOutput(t *testing.T) {
	registryRoot, packRoot := writeTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("template", "init", "gatling/scala-sbt", "--registry", "local:"+registryRoot, "--output", "json")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var result struct {
		Template    string `json:"template"`
		Version     string `json:"version"`
		PackVersion string `json:"packVersion"`
		Source      string `json:"source"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode json output: %v; output %q", err, stdout)
	}
	if result.Template != "gatling/scala-sbt" {
		t.Fatalf("expected template gatling/scala-sbt, got %q", result.Template)
	}
	if result.Version != "" {
		t.Fatalf("expected placeholder template version to be empty, got %q", result.Version)
	}
	if result.PackVersion != "0.1.0" {
		t.Fatalf("expected pack version 0.1.0, got %q", result.PackVersion)
	}
	if result.Source != "local:"+packRoot {
		t.Fatalf("expected source local:%s, got %q", packRoot, result.Source)
	}
	if result.Status != "coming_soon" {
		t.Fatalf("expected status coming_soon, got %q", result.Status)
	}
}

func TestTemplateInitRendersLocalTemplate(t *testing.T) {
	registryRoot, _ := writeRenderableTemplateCatalogFixture(t)
	destination := t.TempDir()

	code, stdout, stderr := runCLI(
		"template",
		"init",
		"examples/service",
		"--registry",
		"local:"+registryRoot,
		"--destination",
		destination,
		"--set",
		"Name=orders",
	)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "Rendered examples/service to ") {
		t.Fatalf("expected render summary, got %q", stdout)
	}

	renderedPath := filepath.Join(destination, "orders.txt")
	payload, err := os.ReadFile(renderedPath)
	if err != nil {
		t.Fatalf("read rendered file: %v", err)
	}
	if string(payload) != "hello orders\n" {
		t.Fatalf("expected rendered payload, got %q", payload)
	}
}

func TestTemplateInitRendersJSONOutput(t *testing.T) {
	registryRoot, _ := writeRenderableTemplateCatalogFixture(t)
	destination := t.TempDir()

	code, stdout, stderr := runCLI(
		"template",
		"init",
		"examples/service",
		"--registry",
		"local:"+registryRoot,
		"--destination",
		destination,
		"--output",
		"json",
	)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var result struct {
		Template    string `json:"template"`
		Version     string `json:"version"`
		PackVersion string `json:"packVersion"`
		Destination string `json:"destination"`
		Files       int    `json:"files"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode json output: %v; output %q", err, stdout)
	}
	if result.Template != "examples/service" {
		t.Fatalf("expected template examples/service, got %q", result.Template)
	}
	if result.Version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %q", result.Version)
	}
	if result.PackVersion != "0.1.0" {
		t.Fatalf("expected pack version 0.1.0, got %q", result.PackVersion)
	}
	if result.Destination == "" {
		t.Fatalf("expected destination in result, got %#v", result)
	}
	if result.Files != 1 {
		t.Fatalf("expected one rendered file, got %d", result.Files)
	}
	if result.Status != "rendered" {
		t.Fatalf("expected rendered status, got %q", result.Status)
	}
}

func TestTemplateInitRejectsInvalidSetValue(t *testing.T) {
	registryRoot, _ := writeRenderableTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("template", "init", "examples/service", "--registry", "local:"+registryRoot, "--set", "Name")

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--set must use Key=Value") {
		t.Fatalf("expected invalid set error, got %q", stderr)
	}
}

func TestTemplateInitReportsUnknownTemplate(t *testing.T) {
	registryRoot, _ := writeTemplateCatalogFixture(t)

	code, stdout, stderr := runCLI("template", "init", "missing/template", "--registry", "local:"+registryRoot)

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, want := range []string{"template not found", "missing/template"} {
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

func TestDoctorWithJSONOutput(t *testing.T) {
	registryRoot, _ := writeTemplateCatalogFixture(t)

	registry := "local:" + registryRoot
	code, stdout, stderr := runCLI("doctor", "--registry", registry, "--output", "json")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var result struct {
		Registry        string `json:"registry"`
		Status          string `json:"status"`
		TemplateEntries int    `json:"templateEntries"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode json output: %v; output %q", err, stdout)
	}
	if result.Registry != registry {
		t.Fatalf("expected registry %q, got %q", registry, result.Registry)
	}
	if result.Status != "ok" {
		t.Fatalf("expected status ok, got %q", result.Status)
	}
	if result.TemplateEntries != 1 {
		t.Fatalf("expected one template entry, got %d", result.TemplateEntries)
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

func writeRenderableTemplateCatalogFixture(t *testing.T) (string, string) {
	t.Helper()

	packRoot := t.TempDir()
	writeTestFile(t, packRoot, "galaxio-pack.yaml", `apiVersion: galaxio.io/v1
kind: TemplatePack
name: examples
version: 0.1.0
templates:
  - name: service
    version: 1.2.3
    path: service
    description: Example service
`)
	writeTestFile(t, packRoot, "service/galaxio-template.yaml", `apiVersion: galaxio.io/v1
kind: Template
name: service
engine: go-template
inputs:
  Name:
    type: string
    default: default
files:
  - from: files
    to: .
`)
	writeTestFile(t, packRoot, "service/files/{{ .Name }}.txt", "hello {{ .Name }}\n")

	registryRoot := t.TempDir()
	writeTestFile(t, registryRoot, "galaxio-registry.yaml", `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: examples
    source: local:`+packRoot+`
`)

	return registryRoot, packRoot
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
