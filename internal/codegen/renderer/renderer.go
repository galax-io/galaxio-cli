package renderer

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"text/template"

	"github.com/galax-io/galaxio-cli/internal/codegen"
	cgtemplates "github.com/galax-io/galaxio-cli/internal/codegen/templates"
)

// OutputFile describes one generated file.
type OutputFile struct {
	Path    string
	Content []byte
}

// RenderOptions configures the generated source layout.
type RenderOptions struct {
	Package string
}

// Renderer renders codegen IR into language-specific output files.
type Renderer struct {
	templates fs.FS
	funcMap   template.FuncMap
}

// NewRenderer constructs a renderer backed by the embedded template pack.
func NewRenderer() *Renderer {
	return &Renderer{
		templates: cgtemplates.Files,
		funcMap: template.FuncMap{
			"renderBody": renderBodyTemplate,
		},
	}
}

// Render emits Scala/sbt Gatling files for the supplied IR spec.
func (r *Renderer) Render(spec *codegen.Spec, opts RenderOptions) ([]OutputFile, error) {
	if spec == nil {
		return nil, fmt.Errorf("render spec: spec is nil")
	}

	pkg := strings.TrimSpace(opts.Package)
	if pkg == "" {
		return nil, fmt.Errorf("render spec: package is required")
	}

	capHint := len(spec.Groups) * 2
	if len(spec.AuthSchemes) > 0 {
		capHint++
	}
	for _, group := range spec.Groups {
		for _, request := range group.Requests {
			if request.Body != nil {
				capHint++
			}
		}
	}

	files := make([]OutputFile, 0, capHint)
	for _, group := range spec.Groups {
		groupFiles, err := r.renderGroup(spec, group, pkg)
		if err != nil {
			return nil, err
		}
		files = append(files, groupFiles...)
	}

	if len(spec.AuthSchemes) > 0 {
		content, err := r.renderTemplate("scala-sbt/auth_actions.scala.tmpl", authActionsTemplateData{
			Package: pkg + ".cases",
			Schemes: spec.AuthSchemes,
		})
		if err != nil {
			return nil, fmt.Errorf("render auth actions: %w", err)
		}
		files = append(files, OutputFile{
			Path:    path.Join("cases", "AuthActions.scala"),
			Content: content,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

func (r *Renderer) renderGroup(spec *codegen.Spec, group codegen.Group, pkg string) ([]OutputFile, error) {
	groupName := pascalCase(group.Name)
	if groupName == "" {
		groupName = "Default"
	}

	requests := make([]actionsRequestTemplateData, 0, len(group.Requests))
	bodyFiles := make([]OutputFile, 0)
	execCalls := make([]string, 0, len(group.Requests))
	for _, request := range group.Requests {
		requestView := buildRequestTemplateData(spec, request)
		requests = append(requests, requestView)
		execCalls = append(execCalls, groupName+"Actions."+requestView.Name)

		if request.Body != nil {
			content, err := r.renderTemplate("scala-sbt/body.json.tmpl", bodyTemplateData{Body: request.Body})
			if err != nil {
				return nil, fmt.Errorf("render body %s: %w", request.Name, err)
			}
			bodyFiles = append(bodyFiles, OutputFile{
				Path:    path.Join("resources", "bodies", requestView.BodyFile),
				Content: content,
			})
		}
	}

	actionsContent, err := r.renderTemplate("scala-sbt/actions.scala.tmpl", actionsTemplateData{
		Package:    pkg + ".cases",
		ObjectName: groupName + "Actions",
		Requests:   requests,
	})
	if err != nil {
		return nil, fmt.Errorf("render actions for group %s: %w", group.Name, err)
	}

	scenarioContent, err := r.renderTemplate("scala-sbt/scenario.scala.tmpl", scenarioTemplateData{
		Package:      pkg + ".scenarios",
		CasesImport:  pkg + ".cases._",
		ObjectName:   groupName + "Scenario",
		ScenarioName: groupName + " Scenario",
		ExecCalls:    execCalls,
	})
	if err != nil {
		return nil, fmt.Errorf("render scenario for group %s: %w", group.Name, err)
	}

	files := []OutputFile{
		{
			Path:    path.Join("cases", groupName+"Actions.scala"),
			Content: actionsContent,
		},
		{
			Path:    path.Join("scenarios", groupName+"Scenario.scala"),
			Content: scenarioContent,
		},
	}
	files = append(files, bodyFiles...)

	return files, nil
}

func (r *Renderer) renderTemplate(name string, data any) ([]byte, error) {
	tmpl, err := template.New(path.Base(name)).Funcs(r.funcMap).ParseFS(r.templates, name)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	content := buf.Bytes()
	if len(content) == 0 || content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}

	return content, nil
}
