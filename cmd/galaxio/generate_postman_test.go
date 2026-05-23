package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/featureflags"
)

func TestGeneratePostmanWritesFilesAndPrintsSummary(t *testing.T) {
	t.Setenv(featureflags.Generate.EnvVar, "true")

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "sample-collection.json")

	code, stdout, stderr := runCLI(
		"generate",
		"postman",
		"--from", source,
		"--template", "scala-sbt",
		"--dest", dest,
		"--package", "org.example.postman",
	)

	if code != exitOK {
		t.Fatalf("expected exit code %d, got %d, stderr %q", exitOK, code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{
		"Generated 3 action files, 2 scenario files, 1 body files",
		"Files written: 6, skipped: 0, conflicts: 0, overwritten: 0",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected summary output to contain %q, got %q", want, stdout)
		}
	}

	for _, file := range []string{
		"cases/AuthActions.scala",
		"cases/OrdersActions.scala",
		"cases/DefaultActions.scala",
		"scenarios/OrdersScenario.scala",
		"scenarios/DefaultScenario.scala",
		"resources/bodies/createOrder.json",
	} {
		if _, err := os.Stat(filepath.Join(dest, file)); err != nil {
			t.Fatalf("expected generated file %s: %v", file, err)
		}
	}

	actions, err := os.ReadFile(filepath.Join(dest, "cases", "OrdersActions.scala"))
	if err != nil {
		t.Fatalf("read generated actions: %v", err)
	}
	payload := string(actions)
	for _, want := range []string{
		`package org.example.postman.cases`,
		`.post("/api/v1/orders")`,
		`.header("X-Trace-Id", "${traceId}")`,
		`.get("/api/v1/orders/${orderId}")`,
		`.queryParam("expand", "${expand}")`,
		`.body(ElFileBody("bodies/createOrder.json")).asJson`,
		`TODO prerequest script: pm.variables.set("traceId", "trace-456");`,
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("expected actions file to contain %q, got %q", want, payload)
		}
	}

	bodyPayload, err := os.ReadFile(filepath.Join(dest, "resources", "bodies", "createOrder.json"))
	if err != nil {
		t.Fatalf("read generated body: %v", err)
	}
	if !strings.Contains(string(bodyPayload), `"sku": "${sku}"`) {
		t.Fatalf("expected variable mapping in generated body, got %q", bodyPayload)
	}
}

func TestGeneratePostmanSupportsJSONOutput(t *testing.T) {
	t.Setenv(featureflags.Generate.EnvVar, "true")

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "sample-collection.json")

	code, stdout, stderr := runCLI(
		"generate",
		"postman",
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

	var result generatePostmanOutput
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode json output: %v; output %q", err, stdout)
	}
	if result.Template != "scala-sbt" {
		t.Fatalf("expected template scala-sbt, got %q", result.Template)
	}
	if result.Actions != 3 || result.Scenarios != 2 || result.BodyFiles != 1 {
		t.Fatalf("unexpected counts: %#v", result)
	}
	if result.FilesWritten != 6 {
		t.Fatalf("expected 6 files written, got %d", result.FilesWritten)
	}
}

func TestGeneratePostmanRejectsInvalidCollection(t *testing.T) {
	t.Setenv(featureflags.Generate.EnvVar, "true")

	source := filepath.Join(t.TempDir(), "invalid-postman.json")
	if err := os.WriteFile(source, []byte(`{"info":{}}`), 0o644); err != nil {
		t.Fatalf("write invalid collection: %v", err)
	}

	code, stdout, stderr := runCLI("generate", "postman", "--from", source)

	if code != exitRuntime {
		t.Fatalf("expected exit code %d, got %d", exitRuntime, code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "parse postman input") {
		t.Fatalf("expected parse error, got %q", stderr)
	}
}

func TestGeneratePostmanSupportsZitadelFixture(t *testing.T) {
	t.Setenv(featureflags.Generate.EnvVar, "true")

	dest := t.TempDir()
	source := filepath.Join("..", "..", "internal", "codegen", "testdata", "zitadel.postman_collection.json")

	code, stdout, stderr := runCLI(
		"generate",
		"postman",
		"--from", source,
		"--template", "scala-sbt",
		"--dest", dest,
		"--package", "org.example.zitadel",
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
		"cases/DefaultActions.scala",
		"scenarios/DefaultScenario.scala",
		"resources/bodies/addZitadelProject.json",
	} {
		if _, err := os.Stat(filepath.Join(dest, file)); err != nil {
			t.Fatalf("expected generated file %s: %v", file, err)
		}
	}

	actions, err := os.ReadFile(filepath.Join(dest, "cases", "DefaultActions.scala"))
	if err != nil {
		t.Fatalf("read generated actions: %v", err)
	}
	payload := string(actions)
	for _, want := range []string{
		`.post("${yourZitadelDomain}/management/v1/projects")`,
		`.get("http://localhost:3000/protected-get")`,
		`.body(ElFileBody("bodies/addZitadelProject.json")).asJson`,
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("expected actions file to contain %q, got %q", want, payload)
		}
	}
}
