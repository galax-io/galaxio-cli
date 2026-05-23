package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/codegen"
	"github.com/galax-io/galaxio-cli/internal/codegen/renderer"
)

func TestWriteRenderedFilesCountsStatuses(t *testing.T) {
	dest := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dest, "cases"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "cases", "Skip.scala"), []byte("same"), 0o644); err != nil {
		t.Fatalf("WriteFile(skip) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "cases", "Overwrite.scala"), []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile(overwrite) error = %v", err)
	}

	files := []renderer.OutputFile{
		{Path: "cases/New.scala", Content: []byte("new")},
		{Path: "cases/Skip.scala", Content: []byte("same")},
		{Path: "cases/Overwrite.scala", Content: []byte("fresh")},
	}

	written, err := writeRenderedFiles(dest, files[:1], codegen.IfExistsOverwrite)
	if err != nil {
		t.Fatalf("writeRenderedFiles(new) error = %v", err)
	}
	if written.Written != 1 || written.Skipped != 0 || written.Overwritten != 0 {
		t.Fatalf("unexpected new-file summary: %#v", written)
	}

	skipped, err := writeRenderedFiles(dest, files[1:2], codegen.IfExistsOverwrite)
	if err != nil {
		t.Fatalf("writeRenderedFiles(skip) error = %v", err)
	}
	if skipped.Skipped != 1 || skipped.Written != 0 || skipped.Overwritten != 0 {
		t.Fatalf("unexpected skipped summary: %#v", skipped)
	}

	overwritten, err := writeRenderedFiles(dest, files[2:], codegen.IfExistsOverwrite)
	if err != nil {
		t.Fatalf("writeRenderedFiles(overwrite) error = %v", err)
	}
	if overwritten.Overwritten != 1 || overwritten.Written != 0 || overwritten.Skipped != 0 {
		t.Fatalf("unexpected overwritten summary: %#v", overwritten)
	}
}

func TestResolveGenerateTemplateValuesDerivesNameWord(t *testing.T) {
	values, err := resolveGenerateTemplateValues(generateOptions{
		dest: filepath.Join(t.TempDir(), "Orders API"),
		pkg:  "org.example.perf",
	})
	if err != nil {
		t.Fatalf("resolveGenerateTemplateValues() error = %v", err)
	}

	if values["NameWord"] != "ordersapi" {
		t.Fatalf("expected ordersapi, got %q", values["NameWord"])
	}
	if values["PackagePath"] != "org/example/perf" {
		t.Fatalf("expected package path org/example/perf, got %q", values["PackagePath"])
	}
}
