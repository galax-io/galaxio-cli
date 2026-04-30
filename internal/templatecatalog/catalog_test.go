package templatecatalog

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRegistryFromLocalSource(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, registryFileName, `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: gatling
    source: local:/tmp/templates-gatling
    description: Gatling templates
`)

	registry, err := SourceFetcher{}.LoadRegistry(context.Background(), root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(registry.Packs) != 1 {
		t.Fatalf("expected one pack, got %d", len(registry.Packs))
	}
	if registry.Packs[0].Name != "gatling" {
		t.Fatalf("expected gatling pack, got %q", registry.Packs[0].Name)
	}
}

func TestListTemplatesFromRegistry(t *testing.T) {
	packRoot := t.TempDir()
	writeGatlingPack(t, packRoot)

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, fmt.Sprintf(`apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: gatling
    source: local:%s
`, packRoot))

	templates, err := SourceFetcher{}.ListTemplates(context.Background(), registryRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected one template, got %d", len(templates))
	}
	if templates[0].Name != "gatling/scala-sbt" {
		t.Fatalf("expected gatling/scala-sbt, got %q", templates[0].Name)
	}
	if templates[0].PackVersion != "0.1.0" {
		t.Fatalf("expected pack version 0.1.0, got %q", templates[0].PackVersion)
	}
	if templates[0].Version != "" {
		t.Fatalf("expected placeholder template version to be empty, got %q", templates[0].Version)
	}
	if !templates[0].Placeholder {
		t.Fatal("expected placeholder template")
	}
}

func TestFindTemplateFromRegistry(t *testing.T) {
	packRoot := t.TempDir()
	writeGatlingPack(t, packRoot)

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, fmt.Sprintf(`apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: gatling
    source: local:%s
`, packRoot))

	template, err := SourceFetcher{}.FindTemplate(context.Background(), registryRoot, "gatling/scala-sbt")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if template.Name != "gatling/scala-sbt" {
		t.Fatalf("expected gatling/scala-sbt, got %q", template.Name)
	}
	if template.Source != "local:"+packRoot {
		t.Fatalf("expected source local:%s, got %q", packRoot, template.Source)
	}
}

func TestFindTemplateReportsMissingTemplate(t *testing.T) {
	packRoot := t.TempDir()
	writeGatlingPack(t, packRoot)

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, fmt.Sprintf(`apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: gatling
    source: local:%s
`, packRoot))

	_, err := SourceFetcher{}.FindTemplate(context.Background(), registryRoot, "missing/template")
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestListTemplatesUsesTemplateVersionWhenPresent(t *testing.T) {
	packRoot := t.TempDir()
	writeFile(t, packRoot, packFileName, `apiVersion: galaxio.io/v1
kind: TemplatePack
name: examples
version: 0.1.0
templates:
  - name: renderable
    version: 1.2.3
    path: renderable
`)
	writeFile(t, filepath.Join(packRoot, "renderable"), templateFileName, `apiVersion: galaxio.io/v1
kind: Template
name: renderable
engine: go-template
inputs:
  Name:
    type: string
files:
  - from: files
    to: .
`)

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, fmt.Sprintf(`apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: examples
    source: local:%s
`, packRoot))

	templates, err := SourceFetcher{}.ListTemplates(context.Background(), registryRoot)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if templates[0].PackVersion != "0.1.0" {
		t.Fatalf("expected pack version 0.1.0, got %q", templates[0].PackVersion)
	}
	if templates[0].Version != "1.2.3" {
		t.Fatalf("expected template version 1.2.3, got %q", templates[0].Version)
	}
	if templates[0].Placeholder {
		t.Fatal("expected renderable template")
	}
}

func TestValidateSourceAcceptsPackWithoutTemplates(t *testing.T) {
	root := t.TempDir()
	writeGatlingPack(t, root)

	err := SourceFetcher{}.ValidateSource(context.Background(), root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateSourceFailsForMissingTemplateManifest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, packFileName, `apiVersion: galaxio.io/v1
kind: TemplatePack
name: gatling
version: 0.1.0
templates:
  - name: fake
    path: fake
`)

	err := SourceFetcher{}.ValidateSource(context.Background(), root)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRegistryFromGitHubSource(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: gatling
    source: github:galax-io/templates-gatling
`))
	}))
	defer server.Close()

	fetcher := SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}
	_, err := fetcher.LoadRegistry(context.Background(), "github:galax-io/galaxio-template-registry")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotPath != "/galax-io/galaxio-template-registry/main/galaxio-registry.yaml" {
		t.Fatalf("unexpected raw path %q", gotPath)
	}
}

func TestValidateSourceReadsGitHubTemplateSubpath(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/galax-io/templates-gatling/main/galaxio-pack.yaml":
			_, _ = w.Write([]byte(`apiVersion: galaxio.io/v1
kind: TemplatePack
name: gatling
version: 0.1.0
templates:
  - name: fake
    path: fake
`))
		case "/galax-io/templates-gatling/main/fake/galaxio-template.yaml":
			_, _ = w.Write([]byte(`apiVersion: galaxio.io/v1
kind: Template
name: fake
engine: go-template
inputs:
  Name:
    type: string
files:
  - from: files
    to: .
`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	fetcher := SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}
	err := fetcher.ValidateSource(context.Background(), "github:galax-io/templates-gatling")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []string{
		"/galax-io/templates-gatling/main/galaxio-pack.yaml",
		"/galax-io/templates-gatling/main/fake/galaxio-template.yaml",
	}
	if fmt.Sprint(paths) != fmt.Sprint(want) {
		t.Fatalf("expected paths %v, got %v", want, paths)
	}
}

func TestValidateRegistryRejectsMissingPacks(t *testing.T) {
	err := ValidateRegistry(Registry{
		APIVersion: "galaxio.io/v1",
		Kind:       "TemplateRegistry",
	})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRegistryWrapsDecodeErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, registryFileName, `apiVersion: [`)

	_, err := SourceFetcher{}.LoadRegistry(context.Background(), root)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "decode template registry") {
		t.Fatalf("expected decode context, got %v", err)
	}
}

func TestLoadPackWrapsMissingManifest(t *testing.T) {
	root := t.TempDir()

	_, err := SourceFetcher{}.LoadPack(context.Background(), root)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"read template pack", "galaxio-pack.yaml", "not found"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestLoadTemplateWrapsValidationErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fake"), templateFileName, `apiVersion: galaxio.io/v1
kind: Template
name: fake
engine: wrong
inputs:
  Name:
    type: string
files:
  - from: files
    to: .
`)

	_, err := SourceFetcher{}.LoadTemplate(context.Background(), root, "fake")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"validate template manifest", "template engine must be go-template"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestValidatePackRejectsParentPath(t *testing.T) {
	err := ValidatePack(Pack{
		APIVersion: "galaxio.io/v1",
		Kind:       "TemplatePack",
		Name:       "gatling",
		Version:    "0.1.0",
		Templates: []PackTemplate{
			{Name: "fake", Path: "../fake"},
		},
	})

	if err == nil {
		t.Fatal("expected error")
	}
}

func writeGatlingPack(t *testing.T, root string) {
	t.Helper()

	writeFile(t, root, packFileName, `apiVersion: galaxio.io/v1
kind: TemplatePack
name: gatling
version: 0.1.0
templates:
  - name: scala-sbt
    description: Gatling Scala project with sbt
`)
}

func writeFile(t *testing.T, root string, name string, content string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func rewriteClient(t *testing.T, baseURL string) *http.Client {
	t.Helper()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		rewritten, err := http.NewRequestWithContext(req.Context(), req.Method, baseURL+req.URL.Path, nil)
		if err != nil {
			return nil, err
		}
		rewritten.Header = req.Header.Clone()
		return http.DefaultTransport.RoundTrip(rewritten)
	})

	return &http.Client{Transport: transport}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
