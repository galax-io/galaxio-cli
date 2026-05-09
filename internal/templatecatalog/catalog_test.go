package templatecatalog

import (
	"archive/zip"
	"bytes"
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

func TestRenderLocalTemplate(t *testing.T) {
	registryRoot, packRoot := writeRenderablePack(t)
	destination := t.TempDir()

	result, err := SourceFetcher{}.Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "examples/renderable",
		Destination:    destination,
		Values: map[string]string{
			"Name": "orders",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Template != "examples/renderable" {
		t.Fatalf("expected examples/renderable, got %q", result.Template)
	}
	if result.Version != "1.2.3" {
		t.Fatalf("expected template version 1.2.3, got %q", result.Version)
	}
	if result.PackVersion != "0.1.0" {
		t.Fatalf("expected pack version 0.1.0, got %q", result.PackVersion)
	}
	if result.Source != "local:"+packRoot {
		t.Fatalf("expected source local:%s, got %q", packRoot, result.Source)
	}
	if result.Files != 2 {
		t.Fatalf("expected two rendered files, got %d", result.Files)
	}
	if result.Status != "rendered" {
		t.Fatalf("expected rendered status, got %q", result.Status)
	}

	payload, err := os.ReadFile(filepath.Join(destination, "orders.txt"))
	if err != nil {
		t.Fatalf("read rendered file: %v", err)
	}
	if string(payload) != "hello orders\n" {
		t.Fatalf("expected rendered file payload, got %q", payload)
	}
}

func TestRenderSkipsConditionalMappings(t *testing.T) {
	registryRoot, packRoot := writeConditionalRenderablePack(t)
	destination := t.TempDir()

	result, err := SourceFetcher{}.Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "examples/renderable",
		Destination:    destination,
		Values: map[string]string{
			"Name":   "orders",
			"Flavor": "pro",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Files != 2 {
		t.Fatalf("expected two rendered files, got %d", result.Files)
	}
	for _, want := range []string{
		filepath.Join(destination, "orders.txt"),
		filepath.Join(destination, "overrides", "pro.txt"),
	} {
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("expected rendered file %s: %v", want, err)
		}
	}
	for _, unwanted := range []string{
		filepath.Join(destination, "overrides", "enterprise.txt"),
	} {
		if _, err := os.Stat(unwanted); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %s to be skipped, got %v", unwanted, err)
		}
	}
	if result.Source != "local:"+packRoot {
		t.Fatalf("expected source local:%s, got %q", packRoot, result.Source)
	}
}

func TestRenderPreservesExecutableMode(t *testing.T) {
	registryRoot, _ := writeRenderablePack(t)
	destination := t.TempDir()

	result, err := SourceFetcher{}.Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "examples/renderable",
		Destination:    destination,
		Values: map[string]string{
			"Name": "orders",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Files != 2 {
		t.Fatalf("expected two rendered files, got %d", result.Files)
	}
	info, err := os.Stat(filepath.Join(destination, "bin", "run.sh"))
	if err != nil {
		t.Fatalf("stat rendered script: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("expected executable mode 0755, got %o", info.Mode().Perm())
	}
}

func TestRenderRejectsPlaceholderTemplate(t *testing.T) {
	packRoot := t.TempDir()
	writeGatlingPack(t, packRoot)

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, fmt.Sprintf(`apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: gatling
    source: local:%s
`, packRoot))

	_, err := SourceFetcher{}.Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "gatling/scala-sbt",
		Destination:    t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "coming soon") {
		t.Fatalf("expected coming soon error, got %v", err)
	}
}

func TestRenderRejectsEscapingPath(t *testing.T) {
	registryRoot, _ := writeRenderablePack(t)

	_, err := SourceFetcher{}.Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "examples/renderable",
		Destination:    t.TempDir(),
		Values: map[string]string{
			"Name": "../escape",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "escapes destination") {
		t.Fatalf("expected safe path error, got %v", err)
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

func TestRenderFromGitHubSourceUsesPackVersionTag(t *testing.T) {
	archive := zipSource(t, "templates-gatling-0.3.0/service/files/{{ .Name }}.txt", "hello {{ .Name }}\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/galax-io/templates-gatling/main/galaxio-pack.yaml":
			_, _ = w.Write([]byte(`apiVersion: galaxio.io/v1
kind: TemplatePack
name: examples
version: 0.3.0
templates:
  - name: service
    version: service-local-version
    path: service
`))
		case "/galax-io/templates-gatling/v0.3.0/service/galaxio-template.yaml":
			_, _ = w.Write([]byte(`apiVersion: galaxio.io/v1
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
`))
		case "/galax-io/templates-gatling/zip/refs/tags/v0.3.0":
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: examples
    source: github:galax-io/templates-gatling
`)

	destination := t.TempDir()
	result, err := (SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}).Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "examples/service",
		Destination:    destination,
		Values:         map[string]string{"Name": "orders"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.PackVersion != "0.3.0" || result.Files != 1 {
		t.Fatalf("unexpected render result %#v", result)
	}

	payload, err := os.ReadFile(filepath.Join(destination, "orders.txt"))
	if err != nil {
		t.Fatalf("read rendered file: %v", err)
	}
	if string(payload) != "hello orders\n" {
		t.Fatalf("unexpected rendered payload %q", payload)
	}
}

func TestRenderFromGitHubSourceUsesCacheOnSecondRun(t *testing.T) {
	t.Setenv(cacheEnv, t.TempDir())

	archive := zipSource(t, "templates-gatling-0.3.0/service/files/{{ .Name }}.txt", "hello {{ .Name }}\n")
	var archiveRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/galax-io/templates-gatling/main/galaxio-pack.yaml":
			_, _ = w.Write([]byte(`apiVersion: galaxio.io/v1
kind: TemplatePack
name: examples
version: 0.3.0
templates:
  - name: service
    version: service-local-version
    path: service
`))
		case "/galax-io/templates-gatling/v0.3.0/service/galaxio-template.yaml":
			_, _ = w.Write([]byte(`apiVersion: galaxio.io/v1
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
`))
		case "/galax-io/templates-gatling/zip/refs/tags/v0.3.0":
			archiveRequests++
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, `apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: examples
    source: github:galax-io/templates-gatling
`)

	fetcher := SourceFetcher{HTTPClient: rewriteClient(t, server.URL)}
	for i := 0; i < 2; i++ {
		_, err := fetcher.Render(context.Background(), RenderOptions{
			RegistrySource: registryRoot,
			TemplateName:   "examples/service",
			Destination:    t.TempDir(),
			Values:         map[string]string{"Name": "orders"},
		})
		if err != nil {
			t.Fatalf("render %d failed: %v", i+1, err)
		}
	}

	if archiveRequests != 1 {
		t.Fatalf("expected one archive request, got %d", archiveRequests)
	}
}

func TestClearCacheRemovesTemplateCacheDirectory(t *testing.T) {
	cacheRoot := t.TempDir()
	t.Setenv(cacheEnv, cacheRoot)

	dir, err := CacheDir()
	if err != nil {
		t.Fatalf("cache dir: %v", err)
	}
	path := filepath.Join(dir, "sources", "cached", "file.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cleared, err := ClearCache()
	if err != nil {
		t.Fatalf("clear cache: %v", err)
	}
	if cleared != dir {
		t.Fatalf("expected cleared path %q, got %q", dir, cleared)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected cache dir removed, got %v", err)
	}
}

func TestClearCacheReturnsPathWhenDirectoryDoesNotExist(t *testing.T) {
	cacheRoot := t.TempDir()
	t.Setenv(cacheEnv, cacheRoot)

	dir, err := CacheDir()
	if err != nil {
		t.Fatalf("cache dir: %v", err)
	}
	cleared, err := ClearCache()
	if err != nil {
		t.Fatalf("clear cache: %v", err)
	}
	if cleared != dir {
		t.Fatalf("expected cleared path %q, got %q", dir, cleared)
	}
}

func TestCacheReadyHandlesMissingMarkerAndDirectory(t *testing.T) {
	root := t.TempDir()

	ready, err := cacheReady(filepath.Join(root, "cache"), filepath.Join(root, "cache", ".ready"))
	if err != nil {
		t.Fatalf("cache ready returned error: %v", err)
	}
	if ready {
		t.Fatal("expected cache to be not ready")
	}
}

func TestCacheReadyRejectsNonRegularMarker(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "cache")
	marker := filepath.Join(cacheDir, ".ready")
	if err := os.MkdirAll(marker, 0o755); err != nil {
		t.Fatalf("mkdir marker dir: %v", err)
	}

	ready, err := cacheReady(cacheDir, marker)
	if err != nil {
		t.Fatalf("cache ready returned error: %v", err)
	}
	if ready {
		t.Fatal("expected cache to be not ready for non-regular marker")
	}
}

func TestSourceOrDefaultUsesFallback(t *testing.T) {
	if got := sourceOrDefault(""); got != DefaultRegistrySource {
		t.Fatalf("expected default source, got %q", got)
	}
	if got := sourceOrDefault("local:/tmp/registry"); got != "local:/tmp/registry" {
		t.Fatalf("expected explicit source, got %q", got)
	}
}

func TestMaterializeSourceRejectsURLSources(t *testing.T) {
	_, _, err := SourceFetcher{}.materializeSource(context.Background(), "https://example.com/templates")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "URL sources is not supported") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestRenderRejectsEscapingTemplatePath(t *testing.T) {
	registryRoot, _ := writeRenderablePack(t)
	_, err := SourceFetcher{}.Render(context.Background(), RenderOptions{
		RegistrySource: registryRoot,
		TemplateName:   "examples/renderable",
		Destination:    t.TempDir(),
		Values:         map[string]string{"Name": "../escape"},
	})
	if err == nil {
		t.Fatal("expected escaping path error")
	}
	if !strings.Contains(err.Error(), "template path escapes destination") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestParseGitHubSourceRejectsMissingRepoName(t *testing.T) {
	_, err := parseGitHubSource("galax-io/", "main")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "owner/repo") {
		t.Fatalf("expected owner/repo error, got %v", err)
	}
}

func TestSourceErrorUnwrap(t *testing.T) {
	cause := errors.New("boom")
	err := SourceError{Source: "local:/tmp/missing", What: "read", Err: cause}
	if !errors.Is(err, cause) {
		t.Fatalf("expected source error to unwrap cause")
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

func TestValidateRegistryRejectsMissingPackNameAndSource(t *testing.T) {
	tests := []struct {
		name     string
		registry Registry
		want     string
	}{
		{
			name: "missing name",
			registry: Registry{
				APIVersion: "galaxio.io/v1",
				Kind:       "TemplateRegistry",
				Packs:      []RegistryPack{{Source: "github:galax-io/templates-gatling"}},
			},
			want: "registry pack name is required",
		},
		{
			name: "missing source",
			registry: Registry{
				APIVersion: "galaxio.io/v1",
				Kind:       "TemplateRegistry",
				Packs:      []RegistryPack{{Name: "gatling"}},
			},
			want: `registry pack "gatling" source is required`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistry(tt.registry)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}
}

func TestValidatePackRejectsInvalidTemplateEntries(t *testing.T) {
	tests := []struct {
		name string
		pack Pack
		want string
	}{
		{
			name: "missing template name",
			pack: Pack{
				APIVersion: "galaxio.io/v1",
				Kind:       "TemplatePack",
				Name:       "gatling",
				Version:    "0.1.0",
				Templates:  []PackTemplate{{Path: "scala-sbt"}},
			},
			want: "template name is required",
		},
		{
			name: "template path escapes",
			pack: Pack{
				APIVersion: "galaxio.io/v1",
				Kind:       "TemplatePack",
				Name:       "gatling",
				Version:    "0.1.0",
				Templates:  []PackTemplate{{Name: "scala-sbt", Path: "../scala-sbt"}},
			},
			want: `template "scala-sbt" path must not contain '..'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePack(tt.pack)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}
}

func TestValidateTemplateRejectsMissingInputsAndFiles(t *testing.T) {
	tests := []struct {
		name     string
		template Template
		want     string
	}{
		{
			name: "missing inputs",
			template: Template{
				APIVersion: "galaxio.io/v1",
				Kind:       "Template",
				Name:       "scala-sbt",
				Engine:     "go-template",
				Files:      []TemplateFile{{From: "files", To: "."}},
			},
			want: "template inputs are required",
		},
		{
			name: "missing files",
			template: Template{
				APIVersion: "galaxio.io/v1",
				Kind:       "Template",
				Name:       "scala-sbt",
				Engine:     "go-template",
				Inputs:     map[string]TemplateInput{"Name": {Type: "string"}},
			},
			want: "template files are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.template)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
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

func writeRenderablePack(t *testing.T) (string, string) {
	t.Helper()

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
    default: default
files:
  - from: files
    to: .
`)
	writeFile(t, filepath.Join(packRoot, "renderable", "files"), "{{ .Name }}.txt", "hello {{ .Name }}\n")
	writeFileMode(t, filepath.Join(packRoot, "renderable", "files", "bin"), "run.sh", "#!/bin/sh\necho hi\n", 0o755)

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, fmt.Sprintf(`apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: examples
    source: local:%s
`, packRoot))

	return registryRoot, packRoot
}

func writeConditionalRenderablePack(t *testing.T) (string, string) {
	t.Helper()

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
    default: default
  Flavor:
    type: string
    default: basic
files:
  - from: files/base
    to: .
  - from: files/overrides/pro
    to: overrides
    if: '{{ eq .Flavor "pro" }}'
  - from: files/overrides/enterprise
    to: overrides
    if: '{{ eq .Flavor "enterprise" }}'
`)
	writeFile(t, filepath.Join(packRoot, "renderable", "files", "base"), "{{ .Name }}.txt", "hello {{ .Name }}\n")
	writeFile(t, filepath.Join(packRoot, "renderable", "files", "overrides", "pro"), "pro.txt", "pro\n")
	writeFile(t, filepath.Join(packRoot, "renderable", "files", "overrides", "enterprise"), "enterprise.txt", "enterprise\n")

	registryRoot := t.TempDir()
	writeFile(t, registryRoot, registryFileName, fmt.Sprintf(`apiVersion: galaxio.io/v1
kind: TemplateRegistry
packs:
  - name: examples
    source: local:%s
`, packRoot))

	return registryRoot, packRoot
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

func writeFileMode(t *testing.T, root string, name string, content string, mode os.FileMode) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
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

func zipSource(t *testing.T, name string, body string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := file.Write([]byte(body)); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	return buffer.Bytes()
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
