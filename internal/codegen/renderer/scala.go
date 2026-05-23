package renderer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/galax-io/galaxio-cli/internal/codegen"
)

type actionsTemplateData struct {
	Package    string
	ObjectName string
	Requests   []actionsRequestTemplateData
}

type actionsRequestTemplateData struct {
	Name     string
	Builder  string
	BodyFile string
	Comments []string
}

type scenarioTemplateData struct {
	Package      string
	CasesImport  string
	ObjectName   string
	ScenarioName string
	ExecCalls    []string
}

type authActionsTemplateData struct {
	Package string
	Schemes []codegen.AuthScheme
}

type bodyTemplateData struct {
	Body *codegen.BodySchema
}

func buildRequestTemplateData(spec *codegen.Spec, request codegen.Request) actionsRequestTemplateData {
	name := codegen.LowerCamel(request.Name)
	if name == "" {
		name = "request"
	}

	lines := []string{
		fmt.Sprintf("http(%q)", requestLabel(request)),
		fmt.Sprintf(".%s(%q)", strings.ToLower(request.Method), joinURLPath(spec.BasePath, request.Path)),
	}
	lines = append(lines, renderParamLines(request.Params)...)
	lines = append(lines, renderHeaderLines(request.Headers)...)

	bodyFile := ""
	if request.Body != nil {
		bodyFile = codegen.LowerCamel(request.Name) + ".json"
		lines = append(lines, fmt.Sprintf(".body(ElFileBody(%q)).asJson", "bodies/"+bodyFile))
	}

	if statusCode := primaryStatusCode(request.Responses); statusCode != "" {
		lines = append(lines, fmt.Sprintf(".check(status is %s)", statusCode))
	}

	return actionsRequestTemplateData{
		Name:     name,
		Builder:  strings.Join(lines, "\n    "),
		BodyFile: bodyFile,
		Comments: request.Comments,
	}
}

func renderParamLines(params []codegen.Param) []string {
	lines := make([]string, 0, len(params))
	for _, param := range params {
		placeholder := scalaPlaceholder(codegen.LowerCamel(param.Name))
		switch strings.ToLower(param.In) {
		case "query":
			lines = append(lines, fmt.Sprintf(".queryParam(%q, %q)", param.Name, placeholder))
		case "formdata":
			lines = append(lines, fmt.Sprintf(".formParam(%q, %q)", param.Name, placeholder))
		}
	}
	return lines
}

func renderHeaderLines(headers []codegen.Header) []string {
	lines := make([]string, 0, len(headers))
	for _, header := range headers {
		value := header.Value
		if strings.TrimSpace(value) == "" {
			value = scalaPlaceholder(codegen.LowerCamel(header.Name))
		}
		lines = append(lines, fmt.Sprintf(".header(%q, %q)", header.Name, value))
	}
	return lines
}

func primaryStatusCode(responses []codegen.Response) string {
	for _, response := range responses {
		if _, err := strconv.Atoi(response.StatusCode); err == nil {
			return response.StatusCode
		}
	}
	return ""
}

func joinURLPath(basePath string, requestPath string) string {
	basePath = strings.TrimSpace(basePath)
	requestPath = strings.TrimSpace(requestPath)

	switch {
	case basePath == "":
		return rewritePathPlaceholders(requestPath)
	case requestPath == "":
		return rewritePathPlaceholders(basePath)
	default:
		return rewritePathPlaceholders(strings.TrimRight(basePath, "/") + "/" + strings.TrimLeft(requestPath, "/"))
	}
}

func rewritePathPlaceholders(value string) string {
	return pathParamReplacer(strings.TrimSpace(value))
}

func pathParamReplacer(value string) string {
	var builder strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '{' {
			if i > 0 && value[i-1] == '$' {
				builder.WriteByte(value[i])
				continue
			}
			end := strings.IndexByte(value[i:], '}')
			if end > 0 {
				name := value[i+1 : i+end]
				builder.WriteString(scalaPlaceholder(codegen.LowerCamel(name)))
				i += end
				continue
			}
		}
		builder.WriteByte(value[i])
	}
	return builder.String()
}

func renderBodyTemplate(body *codegen.BodySchema) string {
	if body == nil {
		return "{}"
	}
	if strings.TrimSpace(body.Raw) != "" {
		return renderRawBody(body.Raw)
	}
	return renderBodyValue(body, nil, 0)
}

func renderRawBody(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}"
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(trimmed)); err != nil {
		return trimmed
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, compact.Bytes(), "", "  "); err != nil {
		return compact.String()
	}

	return pretty.String()
}

func renderBodyValue(schema *codegen.BodySchema, path []string, indent int) string {
	if schema == nil {
		return "null"
	}

	switch strings.ToLower(schema.Type) {
	case "object":
		return renderBodyObject(schema, path, indent)
	case "array":
		return renderBodyArray(schema, path, indent)
	case "integer", "number":
		return scalaPlaceholder(flattenPlaceholder(path))
	case "boolean":
		return scalaPlaceholder(flattenPlaceholder(path))
	default:
		return fmt.Sprintf("%q", scalaPlaceholder(flattenPlaceholder(path)))
	}
}

func renderBodyObject(schema *codegen.BodySchema, path []string, indent int) string {
	if len(schema.Fields) == 0 {
		return "{}"
	}

	lines := make([]string, 0, len(schema.Fields)+2)
	lines = append(lines, "{")
	for i, field := range schema.Fields {
		fieldPath := appendPath(path, field.Name)
		value := renderBodyValue(&field, fieldPath, indent+2)
		line := strings.Repeat(" ", indent+2) + fmt.Sprintf("%q: %s", field.Name, value)
		if i < len(schema.Fields)-1 {
			line += ","
		}
		lines = append(lines, line)
	}
	lines = append(lines, strings.Repeat(" ", indent)+"}")
	return strings.Join(lines, "\n")
}

func renderBodyArray(schema *codegen.BodySchema, path []string, indent int) string {
	if schema.Items == nil {
		return "[]"
	}

	itemValue := renderBodyValue(schema.Items, appendPath(path, "item"), indent+2)
	lines := []string{
		"[",
		strings.Repeat(" ", indent+2) + itemValue,
		strings.Repeat(" ", indent) + "]",
	}
	return strings.Join(lines, "\n")
}

func appendPath(path []string, part string) []string {
	part = strings.TrimSpace(part)
	if part == "" {
		return append([]string{}, path...)
	}
	next := append([]string{}, path...)
	next = append(next, part)
	return next
}

func flattenPlaceholder(parts []string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = codegen.LowerCamel(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return "value"
	}

	result := filtered[0]
	for _, part := range filtered[1:] {
		result += strings.ToUpper(part[:1]) + part[1:]
	}
	return result
}

func requestLabel(request codegen.Request) string {
	return strings.TrimSpace(request.Method) + " " + strings.TrimSpace(request.Path)
}

func scalaPlaceholder(name string) string {
	if strings.TrimSpace(name) == "" {
		name = "value"
	}
	return "${" + name + "}"
}

func pascalCase(value string) string {
	words := codegen.SplitWords(value)
	if len(words) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, word := range words {
		lower := strings.ToLower(word)
		builder.WriteString(strings.ToUpper(lower[:1]))
		builder.WriteString(lower[1:])
	}
	return builder.String()
}
