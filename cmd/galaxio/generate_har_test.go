package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateHARWritesFilesAndPrintsSummary(t *testing.T) {

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "har-application-json.json")

	code, stdout, stderr := runCLI(
		"generate",
		"har",
		"--from", source,
		"--template", "scala-sbt",
		"--dest", dest,
		"--package", "org.example.har",
	)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{
		"Generated 1 action files, 1 scenario files, 1 body files",
		"Files written: 3, skipped: 0, conflicts: 0, overwritten: 0",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected summary output to contain %q, got %q", want, stdout)
		}
	}

	for _, file := range []string{
		"cases/HttpbinOrgActions.scala",
		"scenarios/HttpbinOrgScenario.scala",
		"resources/bodies/postPost.json",
	} {
		if _, err := os.Stat(filepath.Join(dest, file)); err != nil {
			t.Fatalf("expected generated file %s: %v", file, err)
		}
	}

	actions, err := os.ReadFile(filepath.Join(dest, "cases", "HttpbinOrgActions.scala"))
	if err != nil {
		t.Fatalf("read generated actions: %v", err)
	}
	payload := string(actions)
	for _, want := range []string{
		`package org.example.har.cases`,
		`.body(ElFileBody("bodies/postPost.json")).asJson`,
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("expected actions file to contain %q, got %q", want, payload)
		}
	}
	if strings.Contains(payload, "/assets/app.js") {
		t.Fatalf("expected static asset to be filtered by default, got %q", payload)
	}

	bodyPayload, err := os.ReadFile(filepath.Join(dest, "resources", "bodies", "postPost.json"))
	if err != nil {
		t.Fatalf("read generated body: %v", err)
	}
	if !strings.Contains(string(bodyPayload), `"arr_mix_nested": {}`) {
		t.Fatalf("expected recorded JSON payload to be preserved, got %q", bodyPayload)
	}
}

func TestGenerateHARIncludeStaticKeepsAssetRequests(t *testing.T) {

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "har-static-asset.json")

	code, stdout, stderr := runCLI(
		"generate",
		"har",
		"--from", source,
		"--dest", dest,
		"--include-static",
	)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "Generated 1 action files, 1 scenario files, 0 body files") {
		t.Fatalf("expected summary output, got %q", stdout)
	}

	actions, err := os.ReadFile(filepath.Join(dest, "cases", "HttpbinOrgActions.scala"))
	if err != nil {
		t.Fatalf("read generated actions: %v", err)
	}
	if !strings.Contains(string(actions), "/assets/app.js") {
		t.Fatalf("expected static asset request in actions file, got %q", actions)
	}
}

func TestGenerateHARSupportsJSONOutput(t *testing.T) {

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "har-application-json.json")

	code, stdout, stderr := runCLI(
		"generate",
		"har",
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

	var result generateHAROutput
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode json output: %v; output %q", err, stdout)
	}
	if result.Template != "scala-sbt" {
		t.Fatalf("expected template scala-sbt, got %q", result.Template)
	}
	if result.IncludeStatic {
		t.Fatalf("expected includeStatic false by default, got %#v", result)
	}
	if result.Actions != 1 || result.Scenarios != 1 || result.BodyFiles != 1 {
		t.Fatalf("unexpected counts: %#v", result)
	}
	if result.FilesWritten != 3 {
		t.Fatalf("expected 3 files written, got %d", result.FilesWritten)
	}
}

func TestGenerateHARRejectsInvalidHAR(t *testing.T) {

	source := filepath.Join(t.TempDir(), "invalid.har")
	if err := os.WriteFile(source, []byte(`{"log":{}}`), 0o644); err != nil {
		t.Fatalf("write invalid har: %v", err)
	}

	code, stdout, stderr := runCLI("generate", "har", "--from", source)

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d", exitRuntime, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "parse har input") {
		t.Fatalf("expected parse error, got %q", stderr)
	}
}
