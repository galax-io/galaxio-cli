// Package templatecatalog reads and validates Galaxio template registries and
// packs.
package templatecatalog

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

const (
	DefaultRegistrySource = "github:galax-io/galaxio-template-registry"
	registryFileName      = "galaxio-registry.yaml"
	packFileName          = "galaxio-pack.yaml"
	templateFileName      = "galaxio-template.yaml"
	cacheEnv              = "GALAXIO_CACHE_DIR"
)

// ErrTemplateNotFound is returned when a registry does not contain a requested
// template reference.
var ErrTemplateNotFound = errors.New("template not found")

// Registry is the root index of available template packs.
type Registry struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Packs      []RegistryPack `yaml:"packs"`
}

// RegistryPack describes one template pack source.
type RegistryPack struct {
	Name        string `yaml:"name"`
	Source      string `yaml:"source"`
	Description string `yaml:"description"`
}

// Pack describes templates available in one template pack repository.
type Pack struct {
	APIVersion  string         `yaml:"apiVersion"`
	Kind        string         `yaml:"kind"`
	Name        string         `yaml:"name"`
	Version     string         `yaml:"version"`
	Description string         `yaml:"description"`
	Templates   []PackTemplate `yaml:"templates"`
}

// PackTemplate points to one template manifest inside a pack.
type PackTemplate struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// TemplateInput describes one configurable template input.
type TemplateInput struct {
	Type        string `yaml:"type"`
	Default     any    `yaml:"default"`
	Description string `yaml:"description"`
}

// Template describes one renderable template.
type Template struct {
	APIVersion  string                   `yaml:"apiVersion"`
	Kind        string                   `yaml:"kind"`
	Name        string                   `yaml:"name"`
	DisplayName string                   `yaml:"displayName"`
	Description string                   `yaml:"description"`
	Engine      string                   `yaml:"engine"`
	Tags        []string                 `yaml:"tags"`
	Inputs      map[string]TemplateInput `yaml:"inputs"`
	Computed    map[string]string        `yaml:"computed"`
	Files       []TemplateFile           `yaml:"files"`
}

// TemplateFile describes one file mapping in a template.
type TemplateFile struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
	If   string `yaml:"if"`
}

// TemplateRef is a resolved template available to users.
type TemplateRef struct {
	Name        string `json:"name"`
	Pack        string `json:"pack"`
	PackVersion string `json:"packVersion"`
	Version     string `json:"version,omitempty"`
	Path        string `json:"-"`
	Source      string `json:"source"`
	Description string `json:"description,omitempty"`
	Templates   int    `json:"templates"`
	Placeholder bool   `json:"placeholder"`
}

// RenderOptions configures template rendering.
type RenderOptions struct {
	RegistrySource string
	TemplateName   string
	Destination    string
	Values         map[string]string
}

// RenderResult describes rendered template output.
type RenderResult struct {
	Template    string `json:"template"`
	Version     string `json:"version,omitempty"`
	PackVersion string `json:"packVersion"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Files       int    `json:"files"`
	Status      string `json:"status"`
}

// SourceFetcher loads manifest files from local paths or remote sources.
type SourceFetcher struct {
	HTTPClient *http.Client
}

// SourceError adds user-facing context to source loading failures.
type SourceError struct {
	Source string
	What   string
	Err    error
}

func (e SourceError) Error() string {
	return fmt.Sprintf("%s %q: %v", e.What, e.Source, e.Err)
}

func (e SourceError) Unwrap() error {
	return e.Err
}

// LoadRegistry loads and validates a registry manifest from source.
func (f SourceFetcher) LoadRegistry(ctx context.Context, source string) (Registry, error) {
	if source == "" {
		source = DefaultRegistrySource
	}

	payload, err := f.readManifest(ctx, source, registryFileName)
	if err != nil {
		return Registry{}, SourceError{Source: sourceOrDefault(source), What: "read template registry", Err: err}
	}

	var registry Registry
	if err := yaml.Unmarshal(payload, &registry); err != nil {
		return Registry{}, SourceError{Source: sourceOrDefault(source), What: "decode template registry", Err: err}
	}

	if err := ValidateRegistry(registry); err != nil {
		return Registry{}, SourceError{Source: sourceOrDefault(source), What: "validate template registry", Err: err}
	}

	return registry, nil
}

// LoadPack loads and validates a template pack from source.
func (f SourceFetcher) LoadPack(ctx context.Context, source string) (Pack, error) {
	payload, err := f.readManifest(ctx, source, packFileName)
	if err != nil {
		return Pack{}, SourceError{Source: source, What: "read template pack", Err: err}
	}

	var pack Pack
	if err := yaml.Unmarshal(payload, &pack); err != nil {
		return Pack{}, SourceError{Source: source, What: "decode template pack", Err: err}
	}

	if err := ValidatePack(pack); err != nil {
		return Pack{}, SourceError{Source: source, What: "validate template pack", Err: err}
	}

	return pack, nil
}

// LoadTemplate loads and validates a template manifest from a pack source.
func (f SourceFetcher) LoadTemplate(ctx context.Context, packSource string, templatePath string) (Template, error) {
	payload, err := f.readManifest(ctx, joinSource(packSource, templatePath), templateFileName)
	if err != nil {
		return Template{}, SourceError{Source: joinSource(packSource, templatePath), What: "read template manifest", Err: err}
	}

	var template Template
	if err := yaml.Unmarshal(payload, &template); err != nil {
		return Template{}, SourceError{Source: joinSource(packSource, templatePath), What: "decode template manifest", Err: err}
	}

	if err := ValidateTemplate(template); err != nil {
		return Template{}, SourceError{Source: joinSource(packSource, templatePath), What: "validate template manifest", Err: err}
	}

	return template, nil
}

// ListTemplates reads the registry and resolves templates from referenced packs.
func (f SourceFetcher) ListTemplates(ctx context.Context, registrySource string) ([]TemplateRef, error) {
	registry, err := f.LoadRegistry(ctx, registrySource)
	if err != nil {
		return nil, err
	}

	var result []TemplateRef
	for _, registryPack := range registry.Packs {
		pack, err := f.LoadPack(ctx, registryPack.Source)
		if err != nil {
			return nil, fmt.Errorf("load pack %q: %w", registryPack.Name, err)
		}
		for _, template := range pack.Templates {
			result = append(result, TemplateRef{
				Name:        pack.Name + "/" + template.Name,
				Pack:        pack.Name,
				PackVersion: pack.Version,
				Version:     template.Version,
				Path:        template.Path,
				Source:      registryPack.Source,
				Description: template.Description,
				Templates:   len(pack.Templates),
				Placeholder: template.Path == "",
			})
		}
	}

	return result, nil
}

// Render resolves a template from a registry and renders it into a destination.
func (f SourceFetcher) Render(ctx context.Context, opts RenderOptions) (RenderResult, error) {
	ref, err := f.FindTemplate(ctx, opts.RegistrySource, opts.TemplateName)
	if err != nil {
		return RenderResult{}, err
	}
	if ref.Path == "" {
		return RenderResult{}, fmt.Errorf("template %q is coming soon", ref.Name)
	}

	renderSource := ref.renderSource()
	manifest, err := f.LoadTemplate(ctx, renderSource, ref.Path)
	if err != nil {
		return RenderResult{}, err
	}

	sourceRoot, cleanup, err := f.materializeSource(ctx, renderSource)
	if err != nil {
		return RenderResult{}, err
	}
	defer cleanup()

	destination := opts.Destination
	if destination == "" {
		destination = "."
	}
	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return RenderResult{}, err
	}

	data := templateData(manifest, opts.Values)
	files, err := renderTemplateFiles(filepath.Join(sourceRoot, filepath.FromSlash(ref.Path)), absDestination, manifest, data)
	if err != nil {
		return RenderResult{}, err
	}

	return RenderResult{
		Template:    ref.Name,
		Version:     ref.Version,
		PackVersion: ref.PackVersion,
		Source:      ref.Source,
		Destination: absDestination,
		Files:       files,
		Status:      "rendered",
	}, nil
}

// FindTemplate resolves a template reference from a registry.
func (f SourceFetcher) FindTemplate(ctx context.Context, registrySource string, name string) (TemplateRef, error) {
	templates, err := f.ListTemplates(ctx, registrySource)
	if err != nil {
		return TemplateRef{}, err
	}

	for _, template := range templates {
		if template.Name == name {
			return template, nil
		}
	}

	return TemplateRef{}, fmt.Errorf("%w: %s", ErrTemplateNotFound, name)
}

// ValidateSource validates a template pack source and its template manifests.
func (f SourceFetcher) ValidateSource(ctx context.Context, source string) error {
	pack, err := f.LoadPack(ctx, source)
	if err != nil {
		return err
	}

	for _, template := range pack.Templates {
		if template.Path == "" {
			continue
		}
		if _, err := f.LoadTemplate(ctx, source, template.Path); err != nil {
			return fmt.Errorf("validate template %q: %w", template.Name, err)
		}
	}

	return nil
}

// ValidateRegistry validates registry-level invariants.
func ValidateRegistry(registry Registry) error {
	if registry.APIVersion != "galaxio.io/v1" {
		return fmt.Errorf("registry apiVersion must be galaxio.io/v1")
	}
	if registry.Kind != "TemplateRegistry" {
		return fmt.Errorf("registry kind must be TemplateRegistry")
	}
	if len(registry.Packs) == 0 {
		return fmt.Errorf("registry must contain at least one pack")
	}

	for _, pack := range registry.Packs {
		if pack.Name == "" {
			return fmt.Errorf("registry pack name is required")
		}
		if pack.Source == "" {
			return fmt.Errorf("registry pack %q source is required", pack.Name)
		}
	}

	return nil
}

// ValidatePack validates pack-level invariants.
func ValidatePack(pack Pack) error {
	if pack.APIVersion != "galaxio.io/v1" {
		return fmt.Errorf("pack apiVersion must be galaxio.io/v1")
	}
	if pack.Kind != "TemplatePack" {
		return fmt.Errorf("pack kind must be TemplatePack")
	}
	if pack.Name == "" {
		return fmt.Errorf("pack name is required")
	}
	if pack.Version == "" {
		return fmt.Errorf("pack version is required")
	}
	for _, template := range pack.Templates {
		if template.Name == "" {
			return fmt.Errorf("template name is required")
		}
		if template.Path == "" {
			continue
		}
		if strings.Contains(template.Path, "..") {
			return fmt.Errorf("template %q path must not contain '..'", template.Name)
		}
	}

	return nil
}

// ValidateTemplate validates template-level invariants.
func ValidateTemplate(template Template) error {
	if template.APIVersion != "galaxio.io/v1" {
		return fmt.Errorf("template apiVersion must be galaxio.io/v1")
	}
	if template.Kind != "Template" {
		return fmt.Errorf("template kind must be Template")
	}
	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if template.Engine != "go-template" {
		return fmt.Errorf("template engine must be go-template")
	}
	if len(template.Inputs) == 0 {
		return fmt.Errorf("template inputs are required")
	}
	if len(template.Files) == 0 {
		return fmt.Errorf("template files are required")
	}

	return nil
}

func (f SourceFetcher) readManifest(ctx context.Context, source string, manifest string) ([]byte, error) {
	switch {
	case strings.HasPrefix(source, "github:"):
		return f.readURL(ctx, githubRawURL(strings.TrimPrefix(source, "github:"), manifest))
	case strings.HasPrefix(source, "local:"):
		return readLocalManifest(strings.TrimPrefix(source, "local:"), manifest)
	case strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://"):
		return f.readURL(ctx, source)
	default:
		return readLocalManifest(source, manifest)
	}
}

func (f SourceFetcher) readURL(ctx context.Context, url string) ([]byte, error) {
	client := f.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "galaxio-cli")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("GET %s returned %s", url, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (f SourceFetcher) materializeSource(ctx context.Context, source string) (string, func(), error) {
	switch {
	case strings.HasPrefix(source, "local:"):
		return strings.TrimPrefix(source, "local:"), func() {}, nil
	case strings.HasPrefix(source, "github:"):
		return f.materializeGitHubSource(ctx, strings.TrimPrefix(source, "github:"))
	case strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://"):
		return "", nil, fmt.Errorf("rendering from URL sources is not supported yet")
	default:
		return source, func() {}, nil
	}
}

func (f SourceFetcher) materializeGitHubSource(ctx context.Context, repo string) (string, func(), error) {
	ref, err := parseGitHubSource(repo, "main")
	if err != nil {
		return "", nil, err
	}
	cacheDir, err := templateCacheArchiveDir(repo)
	if err != nil {
		return "", nil, err
	}
	markerPath := filepath.Join(cacheDir, ".ready")
	if ready, err := cacheReady(cacheDir, markerPath); err == nil && ready {
		return filepath.Join(cacheDir, filepath.FromSlash(ref.Subpath)), func() {}, nil
	}

	payload, err := f.readURL(ctx, ref.archiveURL())
	if err != nil {
		return "", nil, err
	}

	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return "", nil, err
	}

	tempDir, err := os.MkdirTemp(filepath.Dir(cacheDir), "template-extract-*")
	if err != nil {
		return "", nil, err
	}
	cleanupTemp := func() {
		_ = os.RemoveAll(tempDir)
	}
	if err := extractZip(reader, tempDir); err != nil {
		cleanupTemp()
		return "", nil, err
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		cleanupTemp()
		return "", nil, err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		cleanupTemp()
		return "", nil, fmt.Errorf("unexpected GitHub archive layout for %q", repo)
	}

	if err := os.RemoveAll(cacheDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		cleanupTemp()
		return "", nil, err
	}
	if err := os.Rename(filepath.Join(tempDir, entries[0].Name()), cacheDir); err != nil {
		cleanupTemp()
		return "", nil, err
	}
	if err := os.WriteFile(markerPath, []byte("ok\n"), 0o644); err != nil {
		_ = os.RemoveAll(cacheDir)
		return "", nil, err
	}
	cleanupTemp()

	return filepath.Join(cacheDir, filepath.FromSlash(ref.Subpath)), func() {}, nil
}

func readLocalManifest(root string, manifest string) ([]byte, error) {
	if root == "" {
		return nil, errors.New("local source path is required")
	}

	path := filepath.Join(root, manifest)
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s not found; check the source path and expected manifest name", path)
		}
		return nil, err
	}

	return payload, nil
}

func githubRawURL(repo string, manifest string) string {
	ref, err := parseGitHubSource(repo, "main")
	if err != nil {
		return "https://raw.githubusercontent.com/" + strings.Trim(repo, "/") + "/main/" + manifest
	}

	path := ""
	if ref.Subpath != "" {
		path = strings.Trim(ref.Subpath, "/") + "/"
	}

	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s%s", ref.Owner, ref.Repo, ref.Ref, path, manifest)
}

type githubSource struct {
	Owner   string
	Repo    string
	Ref     string
	Subpath string
}

func parseGitHubSource(repo string, fallbackRef string) (githubSource, error) {
	repo, ref, hasRef := strings.Cut(repo, "#")
	if !hasRef || ref == "" {
		ref = fallbackRef
	}

	parts := strings.Split(strings.Trim(repo, "/"), "/")
	if len(parts) < 2 {
		return githubSource{}, fmt.Errorf("github source must be owner/repo, got %q", repo)
	}
	if parts[1] == "" {
		return githubSource{}, fmt.Errorf("github source must include repository name, got %q", repo)
	}

	subpath := ""
	if len(parts) > 2 {
		subpath = strings.Join(parts[2:], "/")
	}
	return githubSource{
		Owner:   parts[0],
		Repo:    parts[1],
		Ref:     ref,
		Subpath: subpath,
	}, nil
}

func (s githubSource) archiveURL() string {
	refPath := s.Ref
	if s.Ref == "main" || s.Ref == "master" {
		refPath = "refs/heads/" + s.Ref
	} else if strings.HasPrefix(s.Ref, "v") {
		refPath = "refs/tags/" + s.Ref
	}
	return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/%s", s.Owner, s.Repo, refPath)
}

func (r TemplateRef) renderSource() string {
	if !strings.HasPrefix(r.Source, "github:") {
		return r.Source
	}
	if r.PackVersion == "" {
		return r.Source
	}
	return r.Source + "#v" + strings.TrimPrefix(r.PackVersion, "v")
}

func CacheDir() (string, error) {
	if path := os.Getenv(cacheEnv); path != "" {
		return filepath.Join(path, "templates"), nil
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "galaxio", "templates"), nil
}

func ClearCache() (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func templateCacheArchiveDir(source string) (string, error) {
	root, err := CacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(source))
	return filepath.Join(root, "sources", hex.EncodeToString(sum[:])), os.MkdirAll(filepath.Join(root, "sources"), 0o755)
}

func cacheReady(dir string, markerPath string) (bool, error) {
	info, err := os.Stat(markerPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}
	entryInfo, err := os.Stat(dir)
	if err != nil {
		return false, err
	}
	return entryInfo.IsDir(), nil
}

func extractZip(reader *zip.Reader, destination string) error {
	for _, file := range reader.File {
		target := filepath.Join(destination, filepath.Clean(file.Name))
		if !strings.HasPrefix(target, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("zip entry escapes destination: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		if err := copyFile(target, source, file.FileInfo().Mode()); err != nil {
			_ = source.Close()
			return err
		}
		if err := source.Close(); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(path string, source io.Reader, mode os.FileMode) error {
	target, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer target.Close()
	_, err = io.Copy(target, source)
	return err
}

func templateData(manifest Template, overrides map[string]string) map[string]any {
	data := make(map[string]any, len(manifest.Inputs))
	for name, input := range manifest.Inputs {
		data[name] = input.Default
	}
	for name, value := range overrides {
		data[name] = value
	}
	return data
}

func renderTemplateFiles(templateRoot string, destination string, manifest Template, data map[string]any) (int, error) {
	var rendered int
	for _, mapping := range manifest.Files {
		ok, err := shouldRenderMapping(mapping, data)
		if err != nil {
			return 0, err
		}
		if !ok {
			continue
		}
		sourceRoot := filepath.Join(templateRoot, filepath.FromSlash(mapping.From))
		targetRoot := filepath.Join(destination, filepath.FromSlash(mapping.To))
		count, err := renderTree(sourceRoot, targetRoot, data)
		if err != nil {
			return 0, err
		}
		rendered += count
	}
	return rendered, nil
}

func renderTree(sourceRoot string, targetRoot string, data map[string]any) (int, error) {
	var rendered int
	err := filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		targetRelative, err := renderString("path", filepath.ToSlash(relative), data)
		if err != nil {
			return err
		}
		target, err := safeJoin(targetRoot, filepath.FromSlash(targetRelative))
		if err != nil {
			return err
		}

		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		body, err := renderString(relative, string(payload), data)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(body), info.Mode()); err != nil {
			return err
		}
		rendered++
		return nil
	})
	if err != nil {
		return 0, err
	}
	return rendered, nil
}

func renderString(name string, value string, data map[string]any) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(value)
	if err != nil {
		return "", err
	}
	var output strings.Builder
	if err := tmpl.Execute(&output, data); err != nil {
		return "", err
	}
	return output.String(), nil
}

func shouldRenderMapping(mapping TemplateFile, data map[string]any) (bool, error) {
	if strings.TrimSpace(mapping.If) == "" {
		return true, nil
	}
	value, err := renderString("mapping if", mapping.If, data)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no", "off":
		return false, nil
	default:
		return true, nil
	}
}

func safeJoin(root string, relative string) (string, error) {
	target := filepath.Join(root, relative)
	cleanRoot := filepath.Clean(root)
	cleanTarget := filepath.Clean(target)
	if cleanTarget != cleanRoot && !strings.HasPrefix(cleanTarget, cleanRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("template path escapes destination: %s", relative)
	}
	return target, nil
}

func joinSource(source string, child string) string {
	child = strings.Trim(child, "/")
	switch {
	case strings.HasPrefix(source, "github:"):
		return strings.TrimRight(source, "/") + "/" + child
	case strings.HasPrefix(source, "local:"):
		return "local:" + filepath.Join(strings.TrimPrefix(source, "local:"), child)
	case strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://"):
		return strings.TrimRight(source, "/") + "/" + child + "/" + templateFileName
	default:
		return filepath.Join(source, child)
	}
}

func sourceOrDefault(source string) string {
	if source == "" {
		return DefaultRegistrySource
	}
	return source
}
