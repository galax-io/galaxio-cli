// Package templatecatalog reads and validates Galaxio template registries and
// packs.
package templatecatalog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultRegistrySource = "github:galax-io/galaxio-template-registry"
	registryFileName      = "galaxio-registry.yaml"
	packFileName          = "galaxio-pack.yaml"
	templateFileName      = "galaxio-template.yaml"
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

// Template describes one renderable template.
type Template struct {
	APIVersion  string            `yaml:"apiVersion"`
	Kind        string            `yaml:"kind"`
	Name        string            `yaml:"name"`
	DisplayName string            `yaml:"displayName"`
	Description string            `yaml:"description"`
	Engine      string            `yaml:"engine"`
	Tags        []string          `yaml:"tags"`
	Inputs      map[string]any    `yaml:"inputs"`
	Computed    map[string]string `yaml:"computed"`
	Files       []TemplateFile    `yaml:"files"`
}

// TemplateFile describes one file mapping in a template.
type TemplateFile struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// TemplateRef is a resolved template available to users.
type TemplateRef struct {
	Name        string `json:"name"`
	Pack        string `json:"pack"`
	PackVersion string `json:"packVersion"`
	Version     string `json:"version,omitempty"`
	Source      string `json:"source"`
	Description string `json:"description,omitempty"`
	Templates   int    `json:"templates"`
	Placeholder bool   `json:"placeholder"`
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
				Source:      registryPack.Source,
				Description: template.Description,
				Templates:   len(pack.Templates),
				Placeholder: template.Path == "",
			})
		}
	}

	return result, nil
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
	parts := strings.Split(strings.Trim(repo, "/"), "/")
	if len(parts) < 2 {
		return "https://raw.githubusercontent.com/" + strings.Trim(repo, "/") + "/main/" + manifest
	}

	path := ""
	if len(parts) > 2 {
		path = strings.Join(parts[2:], "/") + "/"
	}

	return "https://raw.githubusercontent.com/" + parts[0] + "/" + parts[1] + "/main/" + path + manifest
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
