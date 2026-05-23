package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateIfExistsStrategy(t *testing.T) {
	t.Parallel()

	for _, strategy := range []string{IfExistsSuffix, IfExistsMerge, IfExistsSkip, IfExistsOverwrite} {
		if err := ValidateIfExistsStrategy(strategy); err != nil {
			t.Fatalf("ValidateIfExistsStrategy(%q) error = %v", strategy, err)
		}
	}

	if err := ValidateIfExistsStrategy("bad"); err == nil {
		t.Fatal("expected invalid strategy to fail")
	}
}

func TestWriteFileWithStrategyWritesNewFile(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "cases", "PetsActions.scala")
	result, err := WriteFileWithStrategy(target, []byte("generated"), IfExistsSuffix)
	if err != nil {
		t.Fatalf("WriteFileWithStrategy() error = %v", err)
	}
	if result.Status != "written" {
		t.Fatalf("expected written status, got %#v", result)
	}
	assertFileEquals(t, target, "generated")
}

func TestWriteFileWithStrategySuffixCreatesGeneratedSibling(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "cases", "PetsActions.scala")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing) error = %v", err)
	}

	result, err := WriteFileWithStrategy(target, []byte("generated"), IfExistsSuffix)
	if err != nil {
		t.Fatalf("WriteFileWithStrategy() error = %v", err)
	}
	if result.WrittenPath != filepath.Join(filepath.Dir(target), "PetsActions.generated.scala") {
		t.Fatalf("unexpected written path %#v", result)
	}
	assertFileEquals(t, target, "existing")
	assertFileEquals(t, result.WrittenPath, "generated")
}

func TestWriteFileWithStrategyMergeWritesConflictMarkers(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "cases", "PetsActions.scala")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing) error = %v", err)
	}

	result, err := WriteFileWithStrategy(target, []byte("generated"), IfExistsMerge)
	if err != nil {
		t.Fatalf("WriteFileWithStrategy() error = %v", err)
	}
	if result.Status != "conflict" {
		t.Fatalf("expected conflict status, got %#v", result)
	}

	payload, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	for _, want := range []string{"<<<< generated", "generated", "====", "existing", ">>>> existing"} {
		if !strings.Contains(string(payload), want) {
			t.Fatalf("expected merged content to contain %q, got %q", want, payload)
		}
	}
}

func TestWriteFileWithStrategySkipLeavesOriginalUntouched(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "cases", "PetsActions.scala")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing) error = %v", err)
	}

	result, err := WriteFileWithStrategy(target, []byte("generated"), IfExistsSkip)
	if err != nil {
		t.Fatalf("WriteFileWithStrategy() error = %v", err)
	}
	if result.Status != "skipped" {
		t.Fatalf("expected skipped status, got %#v", result)
	}
	assertFileEquals(t, target, "existing")
}

func TestWriteFileWithStrategyOverwriteReplacesExistingFile(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "cases", "PetsActions.scala")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing) error = %v", err)
	}

	result, err := WriteFileWithStrategy(target, []byte("generated"), IfExistsOverwrite)
	if err != nil {
		t.Fatalf("WriteFileWithStrategy() error = %v", err)
	}
	if result.Status != "overwritten" {
		t.Fatalf("expected overwritten status, got %#v", result)
	}
	assertFileEquals(t, target, "generated")
}

func assertFileEquals(t *testing.T, path string, want string) {
	t.Helper()

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(payload) != want {
		t.Fatalf("expected %q, got %q", want, payload)
	}
}
