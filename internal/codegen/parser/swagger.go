package parser

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/galax-io/galaxio-cli/internal/codegen"
	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v2 "github.com/pb33f/libopenapi/datamodel/high/v2"
)

var versionSegmentPattern = regexp.MustCompile(`^v\d+$`)

// SwaggerParser parses Swagger 2.0 documents into the shared codegen IR.
type SwaggerParser struct{}

var _ Parser = (*SwaggerParser)(nil)

// NewSwaggerParser creates a Swagger 2.0 parser instance.
func NewSwaggerParser() *SwaggerParser {
	return &SwaggerParser{}
}

// Parse converts Swagger 2.0 bytes into the codegen intermediate
// representation.
func (p *SwaggerParser) Parse(_ context.Context, data []byte) (*codegen.Spec, error) {
	document, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("create swagger document: %w", err)
	}

	model, err := document.BuildV2Model()
	if err != nil {
		return nil, fmt.Errorf("build swagger v2 model: %w", err)
	}

	spec := &codegen.Spec{
		BasePath: model.Model.BasePath,
	}
	if model.Model.Info != nil {
		spec.Title = model.Model.Info.Title
		spec.Version = model.Model.Info.Version
	}

	authSchemes := extractAuthSchemes(model.Model.SecurityDefinitions)
	spec.AuthSchemes = authSchemes

	groups, err := extractGroups(model.Model.Paths)
	if err != nil {
		return nil, err
	}
	spec.Groups = groups

	return spec, nil
}

func extractAuthSchemes(definitions *v2.SecurityDefinitions) []codegen.AuthScheme {
	if definitions == nil || definitions.Definitions == nil {
		return nil
	}

	authSchemes := make([]codegen.AuthScheme, 0)
	for key, scheme := range definitions.Definitions.FromOldest() {
		authSchemes = append(authSchemes, codegen.AuthScheme{
			Key:      key,
			Type:     normalizeAuthType(scheme),
			Name:     scheme.Name,
			Location: scheme.In,
		})
	}

	return authSchemes
}

func extractGroups(paths *v2.Paths) ([]codegen.Group, error) {
	if paths == nil || paths.PathItems == nil {
		return nil, nil
	}

	groupIndex := make(map[string]int)
	groups := make([]codegen.Group, 0)

	for path, pathItem := range paths.PathItems.FromOldest() {
		if pathItem == nil {
			continue
		}

		for method, operation := range pathItem.GetOperations().FromOldest() {
			if operation == nil {
				continue
			}

			groupName := groupNameFor(path, operation)
			idx, ok := groupIndex[groupName]
			if !ok {
				groupIndex[groupName] = len(groups)
				groups = append(groups, codegen.Group{Name: groupName})
				idx = len(groups) - 1
			}

			request, err := buildRequest(method, path, pathItem, operation)
			if err != nil {
				return nil, fmt.Errorf("build request for %s %s: %w", strings.ToUpper(method), path, err)
			}
			groups[idx].Requests = append(groups[idx].Requests, request)
		}
	}

	if len(groups) == 0 {
		return nil, nil
	}

	return groups, nil
}

func buildRequest(method string, path string, pathItem *v2.PathItem, operation *v2.Operation) (codegen.Request, error) {
	request := codegen.Request{
		Method: strings.ToUpper(method),
		Path:   path,
		Name:   requestNameFor(method, path, operation),
	}

	parameters := append([]*v2.Parameter{}, pathItem.Parameters...)
	parameters = append(parameters, operation.Parameters...)

	for _, parameter := range parameters {
		if parameter == nil {
			continue
		}

		required := parameter.Required != nil && *parameter.Required
		switch parameter.In {
		case "header":
			request.Headers = append(request.Headers, codegen.Header{
				Name:     parameter.Name,
				Required: required,
			})
		case "body":
			if parameter.Schema == nil {
				continue
			}

			body, err := buildBodySchema(parameter.Schema, "", required)
			if err != nil {
				return codegen.Request{}, fmt.Errorf("build body schema: %w", err)
			}
			request.Body = body
		default:
			request.Params = append(request.Params, codegen.Param{
				Name:     parameter.Name,
				Type:     parameter.Type,
				Format:   parameter.Format,
				Required: required,
			})
		}
	}

	if operation.Responses != nil {
		if operation.Responses.Codes != nil {
			for code, response := range operation.Responses.Codes.FromOldest() {
				request.Responses = append(request.Responses, codegen.Response{
					StatusCode:  code,
					Description: response.Description,
				})
			}
		}
		if operation.Responses.Default != nil {
			request.Responses = append(request.Responses, codegen.Response{
				StatusCode:  "default",
				Description: operation.Responses.Default.Description,
			})
		}
	}

	return request, nil
}

func buildBodySchema(proxy *highbase.SchemaProxy, name string, required bool) (*codegen.BodySchema, error) {
	if proxy == nil {
		return nil, nil
	}

	schema := proxy.Schema()
	if schema == nil {
		if err := proxy.GetBuildError(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("schema resolved to nil")
	}

	body := &codegen.BodySchema{
		Name:     name,
		Type:     schemaType(schema),
		Format:   schema.Format,
		Required: required,
	}

	if schema.Properties != nil {
		requiredFields := make(map[string]struct{}, len(schema.Required))
		for _, fieldName := range schema.Required {
			requiredFields[fieldName] = struct{}{}
		}

		for fieldName, fieldProxy := range schema.Properties.FromOldest() {
			_, fieldRequired := requiredFields[fieldName]
			field, err := buildBodySchema(fieldProxy, fieldName, fieldRequired)
			if err != nil {
				return nil, fmt.Errorf("build field %q: %w", fieldName, err)
			}
			if field != nil {
				body.Fields = append(body.Fields, *field)
			}
		}
	}

	if schema.Items != nil && schema.Items.IsA() && schema.Items.A != nil {
		items, err := buildBodySchema(schema.Items.A, "", false)
		if err != nil {
			return nil, fmt.Errorf("build array items: %w", err)
		}
		body.Items = items
	}

	return body, nil
}

func schemaType(schema *highbase.Schema) string {
	if len(schema.Type) > 0 {
		return schema.Type[0]
	}
	if schema.Properties != nil {
		return "object"
	}
	if schema.Items != nil && schema.Items.IsA() {
		return "array"
	}
	return ""
}

func normalizeAuthType(scheme *v2.SecurityScheme) string {
	if scheme == nil {
		return ""
	}

	switch strings.ToLower(scheme.Type) {
	case "basic":
		return "basic"
	case "apikey":
		if strings.EqualFold(scheme.In, "header") &&
			strings.EqualFold(scheme.Name, "Authorization") &&
			strings.Contains(strings.ToLower(scheme.Description), "bearer") {
			return "bearer"
		}
		return "apikey"
	default:
		return strings.ToLower(scheme.Type)
	}
}

func groupNameFor(path string, operation *v2.Operation) string {
	if operation != nil {
		for _, tag := range operation.Tags {
			if strings.TrimSpace(tag) != "" {
				return lowerCamel(tag)
			}
		}
	}

	segments := meaningfulSegments(path)
	if len(segments) == 0 {
		return "default"
	}

	return lowerCamel(segments[0])
}

func requestNameFor(method string, path string, operation *v2.Operation) string {
	if operation != nil && strings.TrimSpace(operation.OperationId) != "" {
		return lowerCamel(operation.OperationId)
	}

	segments := meaningfulSegments(path)
	if len(segments) == 0 {
		return lowerCamel(method + " request")
	}

	words := make([]string, 0, len(segments)+1)
	words = append(words, strings.ToLower(method))
	for _, segment := range segments {
		words = append(words, segment)
	}

	return lowerCamel(strings.Join(words, " "))
}

func meaningfulSegments(path string) []string {
	parts := strings.Split(path, "/")
	segments := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(part, "{") || strings.HasPrefix(part, ":") {
			continue
		}

		normalized := strings.ToLower(strings.Trim(part, "/"))
		if normalized == "api" || versionSegmentPattern.MatchString(normalized) {
			continue
		}

		segments = append(segments, part)
	}

	return segments
}

func lowerCamel(value string) string {
	words := splitWords(value)
	if len(words) == 0 {
		return ""
	}

	for i := range words {
		words[i] = strings.ToLower(words[i])
	}

	result := words[0]
	for _, word := range words[1:] {
		result += strings.ToUpper(word[:1]) + word[1:]
	}

	return result
}

func splitWords(value string) []string {
	replacer := strings.NewReplacer("/", " ", "-", " ", "_", " ", ".", " ")
	value = replacer.Replace(value)

	parts := strings.FieldsFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})

	words := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}

		var builder strings.Builder
		runes := []rune(part)
		for i, r := range runes {
			if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(runes[i-1]) || i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
				words = appendWord(words, builder.String())
				builder.Reset()
			}
			builder.WriteRune(r)
		}
		words = appendWord(words, builder.String())
	}

	return words
}

func appendWord(words []string, word string) []string {
	word = strings.TrimSpace(word)
	if word == "" {
		return words
	}
	if strings.HasPrefix(word, "{") && strings.HasSuffix(word, "}") {
		return words
	}
	return append(words, word)
}
