package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"unicode"

	"github.com/galax-io/galaxio-cli/internal/codegen"
)

type HARParser struct {
	includeStatic bool
}

var _ Parser = (*HARParser)(nil)

func NewHARParser(includeStatic bool) *HARParser {
	return &HARParser{includeStatic: includeStatic}
}

func (p *HARParser) Parse(_ context.Context, data []byte) (*codegen.Spec, error) {
	var doc harDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode har document: %w", err)
	}

	spec := &codegen.Spec{
		Title:    strings.TrimSpace(firstHARPageTitle(doc.Log.Pages)),
		Version:  strings.TrimSpace(doc.Log.Version),
		BasePath: "",
	}
	if spec.Version == "" {
		spec.Version = "1.2"
	}
	if spec.Title == "" {
		spec.Title = "HAR Recording"
	}
	if len(doc.Log.Entries) == 0 {
		return nil, fmt.Errorf("decode har document: missing log.entries")
	}

	requests := make([]codegen.Request, 0, len(doc.Log.Entries))
	groupName := ""
	nameCounts := make(map[string]int)
	for _, entry := range doc.Log.Entries {
		if !p.includeStatic && isStaticHAREntry(entry) {
			continue
		}

		request, err := buildHARRequest(entry, nameCounts)
		if err != nil {
			return nil, err
		}
		if request.Name == "" {
			continue
		}
		if groupName == "" {
			groupName = harGroupName(entry.Request.URL)
		}
		requests = append(requests, request)
	}

	if groupName == "" {
		groupName = "recording"
	}
	if len(requests) > 0 {
		spec.Groups = []codegen.Group{{
			Name:     groupName,
			Requests: requests,
		}}
	}

	return spec, nil
}

type harDocument struct {
	Log harLog `json:"log"`
}

type harLog struct {
	Version string     `json:"version"`
	Pages   []harPage  `json:"pages"`
	Entries []harEntry `json:"entries"`
}

type harPage struct {
	Title string `json:"title"`
}

type harEntry struct {
	Request  harRequest  `json:"request"`
	Response harResponse `json:"response"`
}

type harRequest struct {
	Method      string         `json:"method"`
	URL         string         `json:"url"`
	Headers     []harNameValue `json:"headers"`
	QueryString []harNameValue `json:"queryString"`
	Cookies     []harCookie    `json:"cookies"`
	PostData    *harPostData   `json:"postData"`
}

type harResponse struct {
	Status  int        `json:"status"`
	Content harContent `json:"content"`
}

type harPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type harContent struct {
	MimeType string `json:"mimeType"`
}

type harNameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func firstHARPageTitle(pages []harPage) string {
	for _, page := range pages {
		if strings.TrimSpace(page.Title) != "" {
			return page.Title
		}
	}
	return ""
}

func buildHARRequest(entry harEntry, nameCounts map[string]int) (codegen.Request, error) {
	parsedURL, err := url.Parse(entry.Request.URL)
	if err != nil {
		return codegen.Request{}, fmt.Errorf("parse har request url %q: %w", entry.Request.URL, err)
	}

	baseName := harRequestName(entry.Request.Method, parsedURL.Path)
	nameCounts[baseName]++
	requestName := baseName
	if nameCounts[baseName] > 1 {
		requestName = fmt.Sprintf("%s%d", baseName, nameCounts[baseName])
	}

	request := codegen.Request{
		Method:  strings.ToUpper(entry.Request.Method),
		Path:    parsedURL.EscapedPath(),
		Name:    requestName,
		Summary: strings.TrimSpace(entry.Request.Method) + " " + parsedURL.EscapedPath(),
	}
	if request.Path == "" {
		request.Path = "/"
	}

	request.Headers = harHeaders(entry.Request.Headers, entry.Request.Cookies)
	request.Params = harParams(entry.Request.QueryString)
	if body := harBodySchema(entry.Request.PostData); body != nil {
		request.Body = body
	}
	if entry.Response.Status > 0 {
		request.Responses = []codegen.Response{{
			StatusCode: fmt.Sprintf("%d", entry.Response.Status),
		}}
	}

	return request, nil
}

func harHeaders(headers []harNameValue, cookies []harCookie) []codegen.Header {
	result := make([]codegen.Header, 0, len(headers)+1)
	seen := make(map[string]struct{})
	for _, header := range headers {
		name := strings.TrimSpace(header.Name)
		value := strings.TrimSpace(header.Value)
		if name == "" || value == "" {
			continue
		}

		lowerName := strings.ToLower(name)
		if _, ok := seen[lowerName]; ok {
			continue
		}
		if shouldSkipHARHeader(lowerName) {
			continue
		}

		result = append(result, codegen.Header{
			Name:  name,
			Value: value,
		})
		seen[lowerName] = struct{}{}
	}

	if cookieValue := harCookieHeader(cookies); cookieValue != "" {
		result = append(result, codegen.Header{
			Name:  "Cookie",
			Value: cookieValue,
		})
	}

	return result
}

func shouldSkipHARHeader(name string) bool {
	switch name {
	case "", "cookie", "content-length", "host", ":authority", ":method", ":path", ":scheme":
		return true
	default:
		return false
	}
}

func harCookieHeader(cookies []harCookie) string {
	parts := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		name := strings.TrimSpace(cookie.Name)
		if name == "" {
			continue
		}
		parts = append(parts, name+"="+cookie.Value)
	}
	return strings.Join(parts, "; ")
}

func harParams(query []harNameValue) []codegen.Param {
	params := make([]codegen.Param, 0, len(query))
	for _, param := range query {
		name := strings.TrimSpace(param.Name)
		if name == "" {
			continue
		}
		params = append(params, codegen.Param{
			Name:     name,
			In:       "query",
			Type:     "string",
			Required: true,
		})
	}
	return params
}

func harBodySchema(postData *harPostData) *codegen.BodySchema {
	if postData == nil || strings.TrimSpace(postData.Text) == "" {
		return nil
	}

	bodyType := "string"
	trimmed := strings.TrimSpace(postData.Text)
	if strings.HasPrefix(trimmed, "{") {
		bodyType = "object"
	}
	if strings.HasPrefix(trimmed, "[") {
		bodyType = "array"
	}

	return &codegen.BodySchema{
		Type:     bodyType,
		Required: true,
		Raw:      trimmed,
	}
}

func isStaticHAREntry(entry harEntry) bool {
	mimeType := strings.ToLower(strings.TrimSpace(entry.Response.Content.MimeType))
	for _, prefix := range []string{"image/", "font/", "audio/", "video/"} {
		if strings.HasPrefix(mimeType, prefix) {
			return true
		}
	}
	for _, exact := range []string{
		"text/css",
		"text/javascript",
		"application/javascript",
		"application/x-javascript",
	} {
		if mimeType == exact {
			return true
		}
	}

	parsedURL, err := url.Parse(entry.Request.URL)
	if err != nil {
		return false
	}
	switch strings.ToLower(path.Ext(parsedURL.Path)) {
	case ".css", ".js", ".mjs", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico", ".woff", ".woff2", ".ttf", ".eot":
		return true
	default:
		return false
	}
}

func harGroupName(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err == nil {
		host := strings.TrimSpace(parsedURL.Hostname())
		if host != "" {
			return lowerCamelHAR(host)
		}
		if segment := firstMeaningfulHARSegment(parsedURL.Path); segment != "" {
			return lowerCamelHAR(segment)
		}
	}
	return "recording"
}

func harRequestName(method string, rawPath string) string {
	parts := []string{strings.ToLower(strings.TrimSpace(method))}
	parts = append(parts, meaningfulHARSegments(rawPath)...)
	if len(parts) == 1 {
		parts = append(parts, "request")
	}
	return lowerCamelHAR(strings.Join(parts, " "))
}

func firstMeaningfulHARSegment(rawPath string) string {
	segments := meaningfulHARSegments(rawPath)
	if len(segments) == 0 {
		return ""
	}
	return segments[0]
}

func meaningfulHARSegments(rawPath string) []string {
	pieces := strings.Split(rawPath, "/")
	segments := make([]string, 0, len(pieces))
	for _, piece := range pieces {
		piece = strings.TrimSpace(piece)
		if piece == "" {
			continue
		}
		lowerPiece := strings.ToLower(piece)
		if lowerPiece == "api" || versionSegmentPattern.MatchString(lowerPiece) || isNumericHARSegment(lowerPiece) {
			continue
		}
		segments = append(segments, piece)
	}
	return segments
}

func isNumericHARSegment(value string) bool {
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return value != ""
}

func lowerCamelHAR(value string) string {
	words := splitHARWords(value)
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

func splitHARWords(value string) []string {
	replacer := strings.NewReplacer("/", " ", "-", " ", "_", " ", ".", " ", ":", " ")
	value = replacer.Replace(value)

	parts := strings.FieldsFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})

	words := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			words = append(words, part)
		}
	}
	return words
}
