package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCommandPrintsHelpWhenEnabled(t *testing.T) {

	code, stdout, stderr := runCLI("generate")

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d", exitOK, code)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{"Generate Gatling load-test source files", "swagger", "har", "postman", "--from", "--template", "--dest", "--package", "--if-exists"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, stdout)
		}
	}
}

func TestGenerateSwaggerWritesFilesAndPrintsSummary(t *testing.T) {

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")

	code, stdout, stderr := runCLI(
		"generate",
		"swagger",
		"--from", source,
		"--template", "scala-sbt",
		"--dest", dest,
		"--package", "org.example.performance",
	)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "Generated 4 action files, 3 scenario files, 1 body files") {
		t.Fatalf("expected summary output, got %q", stdout)
	}
	if !strings.Contains(stdout, "Files written: 8, skipped: 0, conflicts: 0, overwritten: 0") {
		t.Fatalf("expected conflict summary output, got %q", stdout)
	}

	for _, file := range []string{
		"cases/AuthActions.scala",
		"cases/PetsActions.scala",
		"cases/OrdersActions.scala",
		"cases/StatusActions.scala",
		"scenarios/PetsScenario.scala",
		"scenarios/OrdersScenario.scala",
		"scenarios/StatusScenario.scala",
		"resources/bodies/createPet.json",
	} {
		if _, err := os.Stat(filepath.Join(dest, file)); err != nil {
			t.Fatalf("expected generated file %s: %v", file, err)
		}
	}

	petsActions, err := os.ReadFile(filepath.Join(dest, "cases", "PetsActions.scala"))
	if err != nil {
		t.Fatalf("read generated actions: %v", err)
	}
	if !strings.Contains(string(petsActions), `package org.example.performance.cases`) {
		t.Fatalf("expected package declaration in actions file, got %q", petsActions)
	}
	if !strings.Contains(string(petsActions), `.body(ElFileBody("bodies/createPet.json")).asJson`) {
		t.Fatalf("expected body reference in actions file, got %q", petsActions)
	}
}

func TestGenerateSwaggerSupportsJSONOutput(t *testing.T) {

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")

	code, stdout, stderr := runCLI(
		"generate",
		"swagger",
		"--from", source,
		"--dest", dest,
		"--output", "json",
	)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var result generateSwaggerOutput
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode json output: %v; output %q", err, stdout)
	}
	if result.Template != "scala-sbt" {
		t.Fatalf("expected template scala-sbt, got %q", result.Template)
	}
	if result.IfExists != "suffix" {
		t.Fatalf("expected ifExists suffix, got %q", result.IfExists)
	}
	if result.Actions != 4 || result.Scenarios != 3 || result.BodyFiles != 1 {
		t.Fatalf("unexpected counts: %#v", result)
	}
	if result.FilesWritten != 8 {
		t.Fatalf("expected 8 files written, got %d", result.FilesWritten)
	}
	if result.FilesSkipped != 0 || result.FilesConflicted != 0 || result.FilesOverwritten != 0 {
		t.Fatalf("unexpected conflict stats: %#v", result)
	}
}

func TestGenerateSwaggerSupportsOpenAPI3Input(t *testing.T) {

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-openapi3.yaml")

	code, stdout, stderr := runCLI(
		"generate",
		"swagger",
		"--from", source,
		"--template", "scala-sbt",
		"--dest", dest,
		"--package", "org.example.oas3",
	)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "Generated") {
		t.Fatalf("expected generation summary, got %q", stdout)
	}

	for _, file := range []string{
		"cases/AuthActions.scala",
		"cases/PetsActions.scala",
		"scenarios/PetsScenario.scala",
		"resources/bodies/createPet.json",
	} {
		if _, err := os.Stat(filepath.Join(dest, file)); err != nil {
			t.Fatalf("expected generated file %s: %v", file, err)
		}
	}

	petsActions, err := os.ReadFile(filepath.Join(dest, "cases", "PetsActions.scala"))
	if err != nil {
		t.Fatalf("read generated actions: %v", err)
	}
	if !strings.Contains(string(petsActions), `package org.example.oas3.cases`) {
		t.Fatalf("expected package declaration in actions file, got %q", petsActions)
	}
	if !strings.Contains(string(petsActions), `.body(ElFileBody("bodies/createPet.json")).asJson`) {
		t.Fatalf("expected body reference in actions file, got %q", petsActions)
	}
}

func TestGenerateSwaggerRejectsMissingSourceFile(t *testing.T) {

	code, stdout, stderr := runCLI("generate", "swagger", "--from", filepath.Join(t.TempDir(), "missing.yaml"))

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d", exitRuntime, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	for _, want := range []string{"read swagger input", "no such file"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected error to contain %q, got %q", want, stderr)
		}
	}
}

func TestGenerateSwaggerRejectsInvalidSwagger(t *testing.T) {

	source := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(source, []byte("swagger: ["), 0o644); err != nil {
		t.Fatalf("write invalid swagger: %v", err)
	}

	code, stdout, stderr := runCLI("generate", "swagger", "--from", source)

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d", exitRuntime, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "parse swagger input") {
		t.Fatalf("expected parse error, got %q", stderr)
	}
}

func TestGenerateSwaggerRejectsUnknownTemplate(t *testing.T) {

	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")
	code, stdout, stderr := runCLI("generate", "swagger", "--from", source, "--template", "java-gradle")

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, `unknown generate template "java-gradle"`) {
		t.Fatalf("expected unknown template error, got %q", stderr)
	}
}

func TestGenerateSwaggerRequiresFromFlag(t *testing.T) {

	code, stdout, stderr := runCLI("generate", "swagger")

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, `required flag(s) "from" not set`) {
		t.Fatalf("expected missing flag error, got %q", stderr)
	}
}

func TestGenerateSwaggerSuffixWritesGeneratedSiblingOnSecondRun(t *testing.T) {

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")

	firstCode, _, firstStderr := runCLI("generate", "swagger", "--from", source, "--dest", dest)
	if firstCode != exitOK || firstStderr != "" {
		t.Fatalf("first generation failed: code=%d stderr=%q", firstCode, firstStderr)
	}

	code, stdout, stderr := runCLI("generate", "swagger", "--from", source, "--dest", dest, "--if-exists", "suffix")
	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "Files written: 8, skipped: 0, conflicts: 0, overwritten: 0") {
		t.Fatalf("expected suffix summary, got %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(dest, "cases", "PetsActions.generated.scala")); err != nil {
		t.Fatalf("expected generated sibling file: %v", err)
	}
}

func TestGenerateSwaggerMergeWritesConflictMarkers(t *testing.T) {

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")

	firstCode, _, firstStderr := runCLI("generate", "swagger", "--from", source, "--dest", dest)
	if firstCode != exitOK || firstStderr != "" {
		t.Fatalf("first generation failed: code=%d stderr=%q", firstCode, firstStderr)
	}

	target := filepath.Join(dest, "cases", "PetsActions.scala")
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	code, stdout, stderr := runCLI("generate", "swagger", "--from", source, "--dest", dest, "--if-exists", "merge")
	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "conflicts: 8") {
		t.Fatalf("expected merge conflict summary, got %q", stdout)
	}

	payload, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read merged file: %v", err)
	}
	for _, want := range []string{"<<<< generated", "====", ">>>> existing"} {
		if !strings.Contains(string(payload), want) {
			t.Fatalf("expected merged file to contain %q, got %q", want, payload)
		}
	}
}

func TestGenerateSwaggerSkipLeavesOriginalUntouched(t *testing.T) {

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")

	firstCode, _, firstStderr := runCLI("generate", "swagger", "--from", source, "--dest", dest)
	if firstCode != exitOK || firstStderr != "" {
		t.Fatalf("first generation failed: code=%d stderr=%q", firstCode, firstStderr)
	}

	target := filepath.Join(dest, "cases", "PetsActions.scala")
	if err := os.WriteFile(target, []byte("keep-me"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	code, stdout, stderr := runCLI("generate", "swagger", "--from", source, "--dest", dest, "--if-exists", "skip")
	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if !strings.Contains(stderr, "warning: skipped existing file") {
		t.Fatalf("expected skip warning in stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, filepath.ToSlash(target)) && !strings.Contains(stderr, target) {
		t.Fatalf("expected skipped file path in stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "skipped: 8") {
		t.Fatalf("expected skip summary, got %q", stdout)
	}

	payload, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read skipped file: %v", err)
	}
	if string(payload) != "keep-me" {
		t.Fatalf("expected original file untouched, got %q", payload)
	}
}

func TestGenerateSwaggerOverwriteReplacesExistingFile(t *testing.T) {

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")

	firstCode, _, firstStderr := runCLI("generate", "swagger", "--from", source, "--dest", dest)
	if firstCode != exitOK || firstStderr != "" {
		t.Fatalf("first generation failed: code=%d stderr=%q", firstCode, firstStderr)
	}

	target := filepath.Join(dest, "cases", "PetsActions.scala")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	code, stdout, stderr := runCLI("generate", "swagger", "--from", source, "--dest", dest, "--if-exists", "overwrite")
	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "skipped: ") || !strings.Contains(stdout, "overwritten: ") || strings.Contains(stdout, "overwritten: 8") {
		t.Fatalf("expected overwrite summary, got %q", stdout)
	}

	payload, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read overwritten file: %v", err)
	}
	if strings.Contains(string(payload), "old") {
		t.Fatalf("expected overwritten file content, got %q", payload)
	}
}

func TestGenerateSwaggerRejectsUnknownIfExistsStrategy(t *testing.T) {

	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")
	code, stdout, stderr := runCLI("generate", "swagger", "--from", source, "--if-exists", "explode")

	if code != exitUsage {
		t.Fatalf("expected exit code %d, got %d", exitUsage, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, `unknown if-exists strategy "explode"`) {
		t.Fatalf("expected if-exists error, got %q", stderr)
	}
}

func TestGenerateSwaggerInitRendersProjectAndOverlay(t *testing.T) {

	registryRoot := writeGenerateInitTemplateCatalogFixture(t)
	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")

	code, stdout, stderr := runCLI(
		"generate",
		"swagger",
		"--from", source,
		"--dest", dest,
		"--init",
		"--registry", "local:"+registryRoot,
		"--set", "Name=orders-api",
	)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{
		"Generated 4 action files, 3 scenario files, 1 body files",
		"Files written: 11, skipped: 0, conflicts: 0, overwritten: 0",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected init summary to contain %q, got %q", want, stdout)
		}
	}

	for _, file := range []string{
		"build.sbt",
		"src/test/resources/gatling.conf",
		"src/test/scala/com/example/perf/ordersapi/Debug.scala",
		"src/test/scala/com/example/perf/ordersapi/cases/PetsActions.scala",
		"src/test/scala/com/example/perf/ordersapi/scenarios/PetsScenario.scala",
		"src/test/resources/bodies/createPet.json",
	} {
		if _, err := os.Stat(filepath.Join(dest, file)); err != nil {
			t.Fatalf("expected init output file %s: %v", file, err)
		}
	}

	payload, err := os.ReadFile(filepath.Join(dest, "src", "test", "scala", "com", "example", "perf", "ordersapi", "cases", "PetsActions.scala"))
	if err != nil {
		t.Fatalf("read overlaid actions file: %v", err)
	}
	if !strings.Contains(string(payload), "package com.example.perf.ordersapi.cases") {
		t.Fatalf("expected init overlay package, got %q", payload)
	}
}

func TestGenerateSwaggerInitRejectsExistingDestinationWithoutExplicitIfExists(t *testing.T) {

	registryRoot := writeGenerateInitTemplateCatalogFixture(t)
	dest := t.TempDir()
	writeGenerateFile(t, dest, "existing.txt", "keep\n")
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")

	code, stdout, stderr := runCLI(
		"generate",
		"swagger",
		"--from", source,
		"--dest", dest,
		"--init",
		"--registry", "local:"+registryRoot,
	)

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d", exitRuntime, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "already contains files; rerun with --if-exists") {
		t.Fatalf("expected existing destination error, got %q", stderr)
	}
}

func TestGenerateSwaggerInitAllowsExistingDestinationWithExplicitIfExists(t *testing.T) {

	registryRoot := writeGenerateInitTemplateCatalogFixture(t)
	dest := t.TempDir()
	writeGenerateFile(t, dest, "build.sbt", "old\n")
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "petstore-swagger2.yaml")

	code, stdout, stderr := runCLI(
		"generate",
		"swagger",
		"--from", source,
		"--dest", dest,
		"--init",
		"--registry", "local:"+registryRoot,
		"--if-exists", "overwrite",
	)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "overwritten:") {
		t.Fatalf("expected overwrite summary, got %q", stdout)
	}

	payload, err := os.ReadFile(filepath.Join(dest, "build.sbt"))
	if err != nil {
		t.Fatalf("read scaffold file: %v", err)
	}
	if strings.Contains(string(payload), "old") {
		t.Fatalf("expected scaffold file to be overwritten, got %q", payload)
	}
}

func writeGenerateInitTemplateCatalogFixture(t *testing.T) string {
	t.Helper()

	packRoot := t.TempDir()
	writeGenerateFile(t, packRoot, "galaxio-pack.yaml", `apiVersion: galaxio.io/v1
kind: TemplatePack
name: gatling
version: 0.1.0
templates:
  - name: scala-sbt
    version: 1.0.0
    path: scala-sbt
`)
	writeGenerateFile(t, packRoot, "scala-sbt/galaxio-template.yaml", `apiVersion: galaxio.io/v1
kind: Template
name: scala-sbt
engine: go-template
inputs:
  Name:
    type: string
    default: myservice
  NameWord:
    type: string
    default: myservice
  Package:
    type: string
    default: org.galaxio.performance
  PackagePath:
    type: string
    default: org/galaxio/performance
files:
  - from: files
    to: .
`)
	writeGenerateFile(t, packRoot, "scala-sbt/files/build.sbt", `name := "{{ .Name }}"
`)
	writeGenerateFile(t, packRoot, "scala-sbt/files/src/test/resources/gatling.conf", "gatling {}\n")
	writeGenerateFile(t, packRoot, "scala-sbt/files/src/test/scala/{{ .PackagePath }}/{{ .NameWord }}/Debug.scala", `package {{ .Package }}.{{ .NameWord }}

object Debug
`)

	registryRoot := t.TempDir()
	writeGenerateFile(t, registryRoot, "galaxio-registry.yaml", `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: gatling
    source: local:`+packRoot+`
`)

	return registryRoot
}

func writeGenerateFile(t *testing.T, root string, name string, content string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
