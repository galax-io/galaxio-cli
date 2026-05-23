package parser

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/galax-io/galaxio-cli/internal/codegen"
	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v2 "github.com/pb33f/libopenapi/datamodel/high/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

var versionSegmentPattern = regexp.MustCompile(`^v\d+$`)

// SwaggerParser parses Swagger/OpenAPI documents into the shared codegen IR.
type SwaggerParser struct{}

var _ Parser = (*SwaggerParser)(nil)

// NewSwaggerParser creates a Swagger/OpenAPI parser instance.
func NewSwaggerParser() *SwaggerParser {
	return &SwaggerParser{}
}

// Parse converts Swagger/OpenAPI bytes into the codegen intermediate representation.
func (p *SwaggerParser) Parse(_ context.Context, data []byte) (*codegen.Spec, error) {
	document, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("create swagger document: %w", err)
	}

	specInfo := document.GetSpecInfo()
	if specInfo == nil {
		return nil, fmt.Errorf("inspect swagger document: missing spec info")
	}

	if specInfo.VersionNumeric >= 3 {
		return parseV3Document(document)
	}

	return parseV2Document(document)
}

func parseV2Document(document libopenapi.Document) (*codegen.Spec, error) {
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

	spec.AuthSchemes = extractAuthSchemes(model.Model.SecurityDefinitions)

	groups, err := extractGroups(model.Model.Paths)
	if err != nil {
		return nil, err
	}
	spec.Groups = groups

	return spec, nil
}

func parseV3Document(document libopenapi.Document) (*codegen.Spec, error) {
	model, err := document.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("build openapi v3 model: %w", err)
	}

	spec := &codegen.Spec{
		BasePath: basePathFromServers(model.Model.Servers),
	}
	if model.Model.Info != nil {
		spec.Title = model.Model.Info.Title
		spec.Version = model.Model.Info.Version
	}

	if model.Model.Components != nil {
		spec.AuthSchemes = extractAuthSchemesV3(model.Model.Components.SecuritySchemes)
	}

	groups, err := extractGroupsV3(model.Model.Paths)
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

func extractAuthSchemesV3(definitions *orderedmap.Map[string, *v3.SecurityScheme]) []codegen.AuthScheme {
	if definitions == nil {
		return nil
	}

	authSchemes := make([]codegen.AuthScheme, 0)
	for key, scheme := range definitions.FromOldest() {
		authSchemes = append(authSchemes, codegen.AuthScheme{
			Key:      key,
			Type:     normalizeAuthTypeV3(scheme),
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

func extractGroupsV3(paths *v3.Paths) ([]codegen.Group, error) {
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

			groupName := groupNameForV3(path, operation)
			idx, ok := groupIndex[groupName]
			if !ok {
				groupIndex[groupName] = len(groups)
				groups = append(groups, codegen.Group{Name: groupName})
				idx = len(groups) - 1
			}

			request, err := buildRequestV3(method, path, pathItem, operation)
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
		Method:  strings.ToUpper(method),
		Path:    path,
		Name:    requestNameFor(method, path, operation),
		Summary: operation.Summary,
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
				In:       parameter.In,
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

func buildRequestV3(method string, path string, pathItem *v3.PathItem, operation *v3.Operation) (codegen.Request, error) {
	request := codegen.Request{
		Method:  strings.ToUpper(method),
		Path:    path,
		Name:    requestNameForV3(method, path, operation),
		Summary: operation.Summary,
	}

	parameters := append([]*v3.Parameter{}, pathItem.Parameters...)
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
		default:
			paramType, paramFormat := extractV3ParameterSchema(parameter)
			request.Params = append(request.Params, codegen.Param{
				Name:     parameter.Name,
				In:       parameter.In,
				Type:     paramType,
				Format:   paramFormat,
				Required: required,
			})
		}
	}

	if operation.RequestBody != nil {
		body, err := buildRequestBodyV3(operation.RequestBody)
		if err != nil {
			return codegen.Request{}, fmt.Errorf("build request body: %w", err)
		}
		request.Body = body
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

func buildRequestBodyV3(requestBody *v3.RequestBody) (*codegen.BodySchema, error) {
	if requestBody == nil {
		return nil, nil
	}

	mediaType := selectRequestBodyMediaType(requestBody.Content)
	if mediaType == nil || mediaType.Schema == nil {
		return nil, nil
	}

	required := requestBody.Required != nil && *requestBody.Required
	return buildBodySchema(mediaType.Schema, "", required)
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

func extractV3ParameterSchema(parameter *v3.Parameter) (string, string) {
	if parameter == nil {
		return "", ""
	}

	if parameter.Schema != nil {
		schema := parameter.Schema.Schema()
		if schema != nil {
			return schemaType(schema), schema.Format
		}
	}

	if mediaType := selectRequestBodyMediaType(parameter.Content); mediaType != nil && mediaType.Schema != nil {
		schema := mediaType.Schema.Schema()
		if schema != nil {
			return schemaType(schema), schema.Format
		}
	}

	return "", ""
}

func selectRequestBodyMediaType(content *orderedmap.Map[string, *v3.MediaType]) *v3.MediaType {
	if content == nil {
		return nil
	}

	for _, mediaTypeName := range []string{"application/json", "application/*+json"} {
		if mediaType, ok := content.Get(mediaTypeName); ok {
			return mediaType
		}
	}

	for _, mediaType := range content.FromOldest() {
		return mediaType
	}

	return nil
}

func schemaType(schema *highbase.Schema) string {
	if len(schema.Type) > 0 {
		for _, candidate := range schema.Type {
			if !strings.EqualFold(candidate, "null") {
				return candidate
			}
		}
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

func normalizeAuthTypeV3(scheme *v3.SecurityScheme) string {
	if scheme == nil {
		return ""
	}

	switch strings.ToLower(scheme.Type) {
	case "apikey":
		return "apikey"
	case "http":
		switch strings.ToLower(scheme.Scheme) {
		case "bearer":
			return "bearer"
		case "basic":
			return "basic"
		}
		return strings.ToLower(scheme.Scheme)
	case "oauth2":
		return "oauth2"
	default:
		return strings.ToLower(scheme.Type)
	}
}

func groupNameFor(path string, operation *v2.Operation) string {
	if operation != nil {
		for _, tag := range operation.Tags {
			if strings.TrimSpace(tag) != "" {
				return codegen.LowerCamel(tag)
			}
		}
	}

	segments := meaningfulSegments(path)
	if len(segments) == 0 {
		return "default"
	}

	return codegen.LowerCamel(segments[0])
}

func groupNameForV3(path string, operation *v3.Operation) string {
	if operation != nil {
		for _, tag := range operation.Tags {
			if strings.TrimSpace(tag) != "" {
				return codegen.LowerCamel(tag)
			}
		}
	}

	segments := meaningfulSegments(path)
	if len(segments) == 0 {
		return "default"
	}

	return codegen.LowerCamel(segments[0])
}

func requestNameFor(method string, path string, operation *v2.Operation) string {
	if operation != nil && strings.TrimSpace(operation.OperationId) != "" {
		return codegen.LowerCamel(operation.OperationId)
	}

	segments := meaningfulSegments(path)
	if len(segments) == 0 {
		return codegen.LowerCamel(method + " request")
	}

	words := make([]string, 0, len(segments)+1)
	words = append(words, strings.ToLower(method))
	for _, segment := range segments {
		words = append(words, segment)
	}

	return codegen.LowerCamel(strings.Join(words, " "))
}

func requestNameForV3(method string, path string, operation *v3.Operation) string {
	if operation != nil && strings.TrimSpace(operation.OperationId) != "" {
		return codegen.LowerCamel(operation.OperationId)
	}

	segments := meaningfulSegments(path)
	if len(segments) == 0 {
		return codegen.LowerCamel(method + " request")
	}

	words := make([]string, 0, len(segments)+1)
	words = append(words, strings.ToLower(method))
	for _, segment := range segments {
		words = append(words, segment)
	}

	return codegen.LowerCamel(strings.Join(words, " "))
}

func basePathFromServers(servers []*v3.Server) string {
	for _, server := range servers {
		if server == nil || strings.TrimSpace(server.URL) == "" {
			continue
		}

		serverURL := strings.TrimSpace(server.URL)
		parsed, err := url.Parse(serverURL)
		if err != nil {
			if strings.HasPrefix(serverURL, "/") {
				return strings.TrimRight(serverURL, "/")
			}
			continue
		}

		if parsed.Path == "" {
			return ""
		}

		path := strings.TrimRight(parsed.Path, "/")
		if path == "" {
			return "/"
		}
		return path
	}

	return ""
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
