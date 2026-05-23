package templatecatalog

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZipCopiesFiles(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)

	file, err := writer.Create("templates/example.txt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := file.Write([]byte("hello")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}

	dest := t.TempDir()
	if err := extractZip(reader, dest); err != nil {
		t.Fatalf("extractZip() error = %v", err)
	}

	payload, err := os.ReadFile(filepath.Join(dest, "templates", "example.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(payload) != "hello" {
		t.Fatalf("expected hello, got %q", payload)
	}
}

func TestExtractZipRejectsEscapingEntry(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)

	file, err := writer.Create("../escape.txt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := file.Write([]byte("bad")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}

	if err := extractZip(reader, t.TempDir()); err == nil {
		t.Fatal("expected escaping zip entry to fail")
	}
}

func TestRenderTreeRendersFileNamesAndBodies(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()

	sourceDir := filepath.Join(source, "files")
	if err := os.MkdirAll(filepath.Join(sourceDir, "{{ .Name }}"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "{{ .Name }}", "hello-{{ .Name }}.txt"), []byte("hello {{ .Name }}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	rendered, err := renderTree(sourceDir, target, map[string]any{"Name": "orders"})
	if err != nil {
		t.Fatalf("renderTree() error = %v", err)
	}
	if rendered != 1 {
		t.Fatalf("expected one rendered file, got %d", rendered)
	}

	payload, err := os.ReadFile(filepath.Join(target, "orders", "hello-orders.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(payload) != "hello orders" {
		t.Fatalf("expected rendered template body, got %q", payload)
	}
}
