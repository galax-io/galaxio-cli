package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/galax-io/galaxio-cli/internal/codegen"
)

type PostmanParser struct{}

var _ Parser = (*PostmanParser)(nil)

func NewPostmanParser() *PostmanParser {
	return &PostmanParser{}
}

func (p *PostmanParser) Parse(_ context.Context, data []byte) (*codegen.Spec, error) {
	var collection postmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return nil, fmt.Errorf("decode postman collection: %w", err)
	}
	if strings.TrimSpace(collection.Info.Name) == "" || !strings.Contains(collection.Info.Schema, "v2.1.0") {
		return nil, fmt.Errorf("decode postman collection: expected collection v2.1")
	}
	if len(collection.Items) == 0 {
		return nil, fmt.Errorf("decode postman collection: missing items")
	}

	vars := make(map[string]string, len(collection.Variables))
	for _, variable := range collection.Variables {
		key := strings.TrimSpace(variable.Key)
		if key != "" {
			vars[key] = variable.Value
		}
	}

	spec := &codegen.Spec{
		Title:   collection.Info.Name,
		Version: "2.1.0",
	}
	authSchemes := make([]codegen.AuthScheme, 0)
	authSeen := make(map[string]struct{})
	collectPostmanAuth(authSeen, &authSchemes, "collectionAuth", collection.Auth)

	groupOrder := make([]string, 0)
	groups := make(map[string]*codegen.Group)
	ensureGroup := func(name string) *codegen.Group {
		if group, ok := groups[name]; ok {
			return group
		}
		group := &codegen.Group{Name: name}
		groups[name] = group
		groupOrder = append(groupOrder, name)
		return group
	}

	defaultScripts := extractPreRequestScripts(collection.Events)
	nameCounts := make(map[string]int)
	for _, item := range collection.Items {
		if err := walkPostmanItem(item, postmanWalkContext{
			groupName:       "default",
			authScopeName:   "collection",
			inheritedAuth:   collection.Auth,
			inheritedEvents: defaultScripts,
			variables:       vars,
			nameCounts:      nameCounts,
			ensureGroup:     ensureGroup,
			authSeen:        authSeen,
			authSchemes:     &authSchemes,
		}); err != nil {
			return nil, err
		}
	}

	spec.AuthSchemes = authSchemes
	spec.Groups = make([]codegen.Group, 0, len(groupOrder))
	for _, name := range groupOrder {
		spec.Groups = append(spec.Groups, *groups[name])
	}
	return spec, nil
}

type postmanCollection struct {
	Info      postmanInfo       `json:"info"`
	Items     []postmanItem     `json:"item"`
	Variables []postmanVariable `json:"variable"`
	Auth      *postmanAuth      `json:"auth"`
	Events    []postmanEvent    `json:"event"`
}

type postmanInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

type postmanVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postmanItem struct {
	Name    string          `json:"name"`
	Items   []postmanItem   `json:"item"`
	Request *postmanRequest `json:"request"`
	Auth    *postmanAuth    `json:"auth"`
	Events  []postmanEvent  `json:"event"`
}

type postmanRequest struct {
	Method string          `json:"method"`
	Header []postmanHeader `json:"header"`
	Body   *postmanBody    `json:"body"`
	URL    postmanURL      `json:"url"`
	Auth   *postmanAuth    `json:"auth"`
}

type postmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postmanBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

type postmanURL struct {
	Raw      string         `json:"raw"`
	Host     []string       `json:"host"`
	Path     []string       `json:"path"`
	Query    []postmanQuery `json:"query"`
	Protocol string         `json:"protocol"`
	Port     string         `json:"port"`
}

type postmanQuery struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postmanEvent struct {
	Listen string        `json:"listen"`
	Script postmanScript `json:"script"`
}

type postmanScript struct {
	Exec []string `json:"exec"`
}

type postmanAuth struct {
	Type   string            `json:"type"`
	Bearer []postmanAuthAttr `json:"bearer"`
	APIKey []postmanAuthAttr `json:"apikey"`
	OAuth2 []postmanAuthAttr `json:"oauth2"`
	Basic  []postmanAuthAttr `json:"basic"`
}

type postmanAuthAttr struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type postmanWalkContext struct {
	groupName       string
	authScopeName   string
	inheritedAuth   *postmanAuth
	inheritedEvents []string
	variables       map[string]string
	nameCounts      map[string]int
	ensureGroup     func(string) *codegen.Group
	authSeen        map[string]struct{}
	authSchemes     *[]codegen.AuthScheme
}

func walkPostmanItem(item postmanItem, ctx postmanWalkContext) error {
	currentAuth := ctx.inheritedAuth
	currentScope := ctx.authScopeName
	if item.Auth != nil {
		currentAuth = item.Auth
		currentScope = codegen.LowerCamel(item.Name)
		if currentScope == "" {
			currentScope = "item"
		}
		collectPostmanAuth(ctx.authSeen, ctx.authSchemes, currentScope+"Auth", item.Auth)
	}

	currentEvents := append([]string{}, ctx.inheritedEvents...)
	currentEvents = append(currentEvents, extractPreRequestScripts(item.Events)...)

	if len(item.Items) > 0 {
		groupName := codegen.LowerCamel(item.Name)
		if groupName == "" {
			groupName = ctx.groupName
		}
		for _, child := range item.Items {
			if err := walkPostmanItem(child, postmanWalkContext{
				groupName:       groupName,
				authScopeName:   currentScope,
				inheritedAuth:   currentAuth,
				inheritedEvents: currentEvents,
				variables:       ctx.variables,
				nameCounts:      ctx.nameCounts,
				ensureGroup:     ctx.ensureGroup,
				authSeen:        ctx.authSeen,
				authSchemes:     ctx.authSchemes,
			}); err != nil {
				return err
			}
		}
		return nil
	}

	if item.Request == nil {
		return nil
	}

	request, err := buildPostmanRequest(item, currentAuth, currentEvents, ctx.variables, ctx.nameCounts)
	if err != nil {
		return err
	}
	ctx.ensureGroup(ctx.groupName).Requests = append(ctx.ensureGroup(ctx.groupName).Requests, request)
	if item.Request.Auth != nil {
		scope := codegen.LowerCamel(item.Name)
		if scope == "" {
			scope = "request"
		}
		collectPostmanAuth(ctx.authSeen, ctx.authSchemes, scope+"Auth", item.Request.Auth)
	}
	return nil
}

func buildPostmanRequest(item postmanItem, inheritedAuth *postmanAuth, inheritedEvents []string, variables map[string]string, nameCounts map[string]int) (codegen.Request, error) {
	requestName := codegen.LowerCamel(item.Name)
	if requestName == "" {
		requestName = "request"
	}
	nameCounts[requestName]++
	if nameCounts[requestName] > 1 {
		requestName = fmt.Sprintf("%s%d", requestName, nameCounts[requestName])
	}

	requestPath := buildPostmanPath(item.Request.URL, variables)
	req := codegen.Request{
		Method:  strings.ToUpper(strings.TrimSpace(item.Request.Method)),
		Path:    requestPath,
		Name:    requestName,
		Summary: item.Name,
	}
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Path == "" {
		req.Path = "/"
	}

	for _, script := range inheritedEvents {
		req.Comments = append(req.Comments, "TODO prerequest script: "+script)
	}

	for _, header := range item.Request.Header {
		key := strings.TrimSpace(header.Key)
		if key == "" {
			continue
		}
		req.Headers = append(req.Headers, codegen.Header{
			Name:  key,
			Value: mapPostmanVariables(header.Value),
		})
	}

	for _, query := range item.Request.URL.Query {
		key := strings.TrimSpace(query.Key)
		if key == "" {
			continue
		}
		req.Params = append(req.Params, codegen.Param{
			Name:     key,
			In:       "query",
			Type:     "string",
			Required: true,
		})
	}

	if body := buildPostmanBody(item.Request.Body); body != nil {
		req.Body = body
	}

	if item.Request.Auth != nil {
		req.Headers = append(req.Headers, postmanAuthHeaders(item.Request.Auth)...)
	} else if inheritedAuth != nil {
		req.Headers = append(req.Headers, postmanAuthHeaders(inheritedAuth)...)
	}

	return req, nil
}

func buildPostmanBody(body *postmanBody) *codegen.BodySchema {
	if body == nil || strings.TrimSpace(body.Raw) == "" {
		return nil
	}
	trimmed := mapPostmanVariables(body.Raw)
	bodyType := "string"
	if strings.HasPrefix(strings.TrimSpace(trimmed), "{") {
		bodyType = "object"
	} else if strings.HasPrefix(strings.TrimSpace(trimmed), "[") {
		bodyType = "array"
	}
	return &codegen.BodySchema{
		Type:     bodyType,
		Required: true,
		Raw:      trimmed,
	}
}

func buildPostmanPath(u postmanURL, variables map[string]string) string {
	pathOnly := "/" + strings.Join(transformPostmanPathSegments(u.Path), "/")
	pathOnly = strings.ReplaceAll(pathOnly, "//", "/")
	pathOnly = mapPostmanVariables(pathOnly)
	if pathOnly == "/" && strings.TrimSpace(u.Raw) == "" {
		return "/"
	}

	if shouldUsePostmanRawURL(u, variables) {
		return mapPostmanVariables(u.Raw)
	}
	return pathOnly
}

func shouldUsePostmanRawURL(u postmanURL, variables map[string]string) bool {
	raw := strings.TrimSpace(u.Raw)
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return true
	}
	if len(u.Host) == 1 {
		host := strings.TrimSpace(u.Host[0])
		if strings.HasPrefix(host, "{{") && strings.HasSuffix(host, "}}") {
			key := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(host, "{{"), "}}"))
			_, known := variables[key]
			return !known
		}
	}
	return false
}

func transformPostmanPathSegments(segments []string) []string {
	out := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		out = append(out, segment)
	}
	return out
}

func extractPreRequestScripts(events []postmanEvent) []string {
	var scripts []string
	for _, event := range events {
		if !strings.EqualFold(event.Listen, "prerequest") {
			continue
		}
		for _, line := range event.Script.Exec {
			line = strings.TrimSpace(line)
			if line != "" {
				scripts = append(scripts, line)
			}
		}
	}
	return scripts
}

func collectPostmanAuth(seen map[string]struct{}, schemes *[]codegen.AuthScheme, key string, auth *postmanAuth) {
	if auth == nil {
		return
	}
	scheme := normalizePostmanAuth(key, auth)
	if scheme.Key == "" {
		return
	}
	dedupe := scheme.Key + "|" + scheme.Type + "|" + scheme.Name + "|" + scheme.Location
	if _, ok := seen[dedupe]; ok {
		return
	}
	seen[dedupe] = struct{}{}
	*schemes = append(*schemes, scheme)
}

func normalizePostmanAuth(key string, auth *postmanAuth) codegen.AuthScheme {
	if auth == nil {
		return codegen.AuthScheme{}
	}
	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case "bearer":
		return codegen.AuthScheme{Key: key, Type: "bearer", Name: "Authorization", Location: "header"}
	case "apikey":
		name := findPostmanAuthValue(auth.APIKey, "key")
		location := findPostmanAuthValue(auth.APIKey, "in")
		if location == "" {
			location = "header"
		}
		return codegen.AuthScheme{Key: key, Type: "apikey", Name: name, Location: location}
	case "oauth2":
		return codegen.AuthScheme{Key: key, Type: "oauth2", Name: "Authorization", Location: "header"}
	case "basic":
		return codegen.AuthScheme{Key: key, Type: "basic", Name: "Authorization", Location: "header"}
	default:
		return codegen.AuthScheme{Key: key, Type: strings.ToLower(strings.TrimSpace(auth.Type))}
	}
}

func postmanAuthHeaders(auth *postmanAuth) []codegen.Header {
	if auth == nil {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case "bearer":
		token := findPostmanAuthValue(auth.Bearer, "token")
		if token == "" {
			return nil
		}
		return []codegen.Header{{Name: "Authorization", Value: "Bearer " + mapPostmanVariables(token)}}
	case "apikey":
		name := findPostmanAuthValue(auth.APIKey, "key")
		value := findPostmanAuthValue(auth.APIKey, "value")
		if name == "" || value == "" {
			return nil
		}
		return []codegen.Header{{Name: name, Value: mapPostmanVariables(value)}}
	default:
		return nil
	}
}

func findPostmanAuthValue(attrs []postmanAuthAttr, key string) string {
	for _, attr := range attrs {
		if strings.EqualFold(strings.TrimSpace(attr.Key), key) {
			return strings.TrimSpace(postmanAuthRawValue(attr.Value))
		}
	}
	return ""
}

func postmanAuthRawValue(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return asString
	}

	var asArray []string
	if err := json.Unmarshal(raw, &asArray); err == nil {
		return strings.Join(asArray, ",")
	}

	return string(raw)
}

func mapPostmanVariables(value string) string {
	var builder strings.Builder
	for {
		start := strings.Index(value, "{{")
		if start < 0 {
			builder.WriteString(value)
			break
		}
		end := strings.Index(value[start+2:], "}}")
		if end < 0 {
			builder.WriteString(value)
			break
		}
		end += start + 2
		builder.WriteString(value[:start])
		name := strings.TrimSpace(value[start+2 : end])
		builder.WriteString("${" + codegen.LowerCamel(name) + "}")
		value = value[end+2:]
	}
	return builder.String()
}
