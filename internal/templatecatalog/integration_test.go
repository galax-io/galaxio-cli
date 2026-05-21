//go:build integration

package templatecatalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, packYAML, templateYAML string, archive []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/galax-io/templates-test/main/galaxio-pack.yaml":
			_, _ = w.Write([]byte(packYAML))
		case "/galax-io/templates-test/v0.1.0/svc/galaxio-template.yaml":
			_, _ = w.Write([]byte(templateYAML))
		case "/galax-io/templates-test/zip/refs/tags/v0.1.0":
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
}

const testPackYAML = `apiVersion: galaxio.io/v1
kind: TemplatePack
name: mypack
version: 0.1.0
templates:
  - name: svc
    version: 1.0.0
    path: svc
    description: A service template
  - name: placeholder
    description: Coming soon
`

const testTemplateYAML = `apiVersion: galaxio.io/v1
kind: Template
name: svc
engine: go-template
inputs:
  Name:
    type: string
    default: myapp
files:
  - from: files
    to: .
`

func TestIntegration_ListTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/galax-io/templates-test/main/galaxio-pack.yaml":
			_, _ = w.Write([]byte(testPackYAML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: mypack
    source: github:galax-io/templates-test
`)

	fetcher := SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}
	refs, err := fetcher.ListTemplates(context.Background(), registryRoot)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 template refs, got %d", len(refs))
	}

	svc := refs[0]
	if svc.Name != "mypack/svc" {
		t.Fatalf("expected name mypack/svc, got %q", svc.Name)
	}
	if svc.Version != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %q", svc.Version)
	}
	if svc.Placeholder {
		t.Fatal("expected svc to not be placeholder")
	}

	placeholder := refs[1]
	if placeholder.Name != "mypack/placeholder" {
		t.Fatalf("expected name mypack/placeholder, got %q", placeholder.Name)
	}
	if !placeholder.Placeholder {
		t.Fatal("expected placeholder to be true")
	}
}

func TestIntegration_RenderTemplate(t *testing.T) {
	t.Setenv(cacheEnv, t.TempDir())

	archive := zipSource(t, "templates-test-0.1.0/svc/files/{{ .Name }}.txt", "hello {{ .Name }}\n")
	server := newTestServer(t, testPackYAML, testTemplateYAML, archive)
	defer server.Close()

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: mypack
    source: github:galax-io/templates-test
`)

	destination := t.TempDir()
	fetcher := SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}
	result, err := fetcher.Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "mypack/svc",
		Destination:    destination,
		Values:         map[string]string{"Name": "orders"},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if result.Files != 1 {
		t.Fatalf("expected 1 file, got %d", result.Files)
	}
	if result.Status != "rendered" {
		t.Fatalf("expected status rendered, got %q", result.Status)
	}
	if result.PackVersion != "0.1.0" {
		t.Fatalf("expected pack version 0.1.0, got %q", result.PackVersion)
	}

	payload, err := os.ReadFile(filepath.Join(destination, "orders.txt"))
	if err != nil {
		t.Fatalf("read rendered file: %v", err)
	}
	if string(payload) != "hello orders\n" {
		t.Fatalf("unexpected content %q", payload)
	}
}

func TestIntegration_MalformedRegistry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not: valid: yaml: [[[`))
	}))
	defer server.Close()

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: broken
    source: github:galax-io/templates-test
`)

	fetcher := SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}
	_, err := fetcher.ListTemplates(context.Background(), registryRoot)
	if err == nil {
		t.Fatal("expected error for malformed pack")
	}
}

func TestIntegration_MissingArchive(t *testing.T) {
	t.Setenv(cacheEnv, t.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/galax-io/templates-test/main/galaxio-pack.yaml":
			_, _ = w.Write([]byte(testPackYAML))
		case "/galax-io/templates-test/v0.1.0/svc/galaxio-template.yaml":
			_, _ = w.Write([]byte(testTemplateYAML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: mypack
    source: github:galax-io/templates-test
`)

	fetcher := SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}
	_, err := fetcher.Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "mypack/svc",
		Destination:    t.TempDir(),
		Values:         map[string]string{"Name": "test"},
	})
	if err == nil {
		t.Fatal("expected error for missing archive")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected 404 in error, got %v", err)
	}
}

func TestIntegration_CacheIsolation(t *testing.T) {
	t.Setenv(cacheEnv, t.TempDir())

	archive := zipSource(t, "templates-test-0.1.0/svc/files/{{ .Name }}.txt", "hello {{ .Name }}\n")
	server := newTestServer(t, testPackYAML, testTemplateYAML, archive)
	defer server.Close()

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: mypack
    source: github:galax-io/templates-test
`)

	fetcher := SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}

	for i, name := range []string{"alpha", "beta"} {
		dest := t.TempDir()
		result, err := fetcher.Render(context.Background(), RenderOptions{
			RegistrySource: registryRoot,
			TemplateName:   "mypack/svc",
			Destination:    dest,
			Values:         map[string]string{"Name": name},
		})
		if err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
		if result.Files != 1 {
			t.Fatalf("render %d: expected 1 file, got %d", i, result.Files)
		}

		payload, err := os.ReadFile(filepath.Join(dest, fmt.Sprintf("%s.txt", name)))
		if err != nil {
			t.Fatalf("render %d: read file: %v", i, err)
		}
		expected := fmt.Sprintf("hello %s\n", name)
		if string(payload) != expected {
			t.Fatalf("render %d: expected %q, got %q", i, expected, payload)
		}

		other := map[string]string{"alpha": "beta", "beta": "alpha"}[name]
		otherFile := filepath.Join(dest, fmt.Sprintf("%s.txt", other))
		if _, err := os.Stat(otherFile); err == nil {
			t.Fatalf("render %d: found file from other render: %s", i, otherFile)
		}
	}
}

func TestIntegration_PlaceholderTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/galax-io/templates-test/main/galaxio-pack.yaml":
			_, _ = w.Write([]byte(testPackYAML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: mypack
    source: github:galax-io/templates-test
`)

	fetcher := SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}
	_, err := fetcher.Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "mypack/placeholder",
		Destination:    t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for placeholder template")
	}
	if !strings.Contains(err.Error(), "coming soon") {
		t.Fatalf("expected 'coming soon' error, got %v", err)
	}
}

func TestIntegration_InvalidRegistryManifest(t *testing.T) {
	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, `apiVersion: galaxio.io/v1
kind: WrongKind
packs:
  - name: mypack
    source: local:/tmp/nope
`)

	_, err := SourceFetcher{}.ListTemplates(context.Background(), registryRoot)
	if err == nil {
		t.Fatal("expected error for invalid registry kind")
	}
	if !strings.Contains(err.Error(), "TemplateRegistry") {
		t.Fatalf("expected kind validation error, got %v", err)
	}
}

func TestIntegration_TemplateNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/galax-io/templates-test/main/galaxio-pack.yaml":
			_, _ = w.Write([]byte(testPackYAML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: mypack
    source: github:galax-io/templates-test
`)

	fetcher := SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}
	_, err := fetcher.Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "mypack/nonexistent",
		Destination:    t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}
