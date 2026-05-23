package parser

import (
	"context"
	"strings"
	"testing"
)

func TestHARParserParseSupportsShortFixture(t *testing.T) {
	t.Parallel()

	fixture := readTestFixture(t, "har-short.json")

	spec, err := NewHARParser(false).Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if spec.Title != "HAR Recording" {
		t.Fatalf("expected title %q, got %q", "HAR Recording", spec.Title)
	}
	if len(spec.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(spec.Groups))
	}
	group := spec.Groups[0]
	if group.Name != "httpbinOrg" {
		t.Fatalf("expected group name %q, got %q", "httpbinOrg", group.Name)
	}
	if len(group.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(group.Requests))
	}
	request := findRequest(t, group, "getGet")
	if request.Path != "/get" {
		t.Fatalf("expected path %q, got %q", "/get", request.Path)
	}
	if len(request.Headers) != 0 || len(request.Params) != 0 || request.Body != nil {
		t.Fatalf("expected minimal request shape, got %#v", request)
	}
}

func TestHARParserParseSupportsHeadersFixture(t *testing.T) {
	t.Parallel()

	fixture := readTestFixture(t, "har-headers.json")

	spec, err := NewHARParser(false).Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	request := findRequest(t, spec.Groups[0], "getGet")
	if len(request.Headers) != 2 {
		t.Fatalf("expected 2 request headers, got %#v", request.Headers)
	}
	if request.Headers[0].Name != "accept" || request.Headers[1].Name != "x-foo" {
		t.Fatalf("unexpected headers: %#v", request.Headers)
	}
}

func TestHARParserParseSupportsCookiesFixture(t *testing.T) {
	t.Parallel()

	fixture := readTestFixture(t, "har-cookies.json")

	spec, err := NewHARParser(false).Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	request := findRequest(t, spec.Groups[0], "getCookies")
	if len(request.Headers) != 1 {
		t.Fatalf("expected cookie header only, got %#v", request.Headers)
	}
	if request.Headers[0].Name != "Cookie" || request.Headers[0].Value != "foo=bar; bar=baz" {
		t.Fatalf("unexpected cookie header: %#v", request.Headers[0])
	}
}

func TestHARParserParseSupportsEncodedQueryFixture(t *testing.T) {
	t.Parallel()

	fixture := readTestFixture(t, "har-query-encoded.json")

	spec, err := NewHARParser(false).Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	request := findRequest(t, spec.Groups[0], "getAnything")
	if len(request.Params) != 5 {
		t.Fatalf("expected 5 query params, got %#v", request.Params)
	}
	if request.Params[0].Name != "stringPound" || request.Params[4].Name != "array" {
		t.Fatalf("unexpected query params: %#v", request.Params)
	}
}

func TestHARParserParseSupportsApplicationJSONFixture(t *testing.T) {
	t.Parallel()

	fixture := readTestFixture(t, "har-application-json.json")

	spec, err := NewHARParser(false).Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	request := findRequest(t, spec.Groups[0], "postPost")
	if request.Body == nil {
		t.Fatal("expected raw request body")
	}
	if !strings.Contains(request.Body.Raw, `"number":1`) {
		t.Fatalf("expected recorded JSON body, got %q", request.Body.Raw)
	}
	if len(request.Headers) != 1 || request.Headers[0].Name != "content-type" {
		t.Fatalf("unexpected request headers: %#v", request.Headers)
	}
}

func TestHARParserParseFiltersStaticResourcesByDefault(t *testing.T) {
	t.Parallel()

	fixture := readTestFixture(t, "har-static-asset.json")

	spec, err := NewHARParser(false).Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if len(spec.Groups) != 0 {
		t.Fatalf("expected static-only fixture to be filtered, got %#v", spec.Groups)
	}
}

func TestHARParserParseIncludesStaticWhenRequested(t *testing.T) {
	t.Parallel()

	fixture := readTestFixture(t, "har-static-asset.json")

	spec, err := NewHARParser(true).Parse(context.Background(), fixture)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if len(spec.Groups) != 1 || len(spec.Groups[0].Requests) != 1 {
		t.Fatalf("expected static entry to be preserved, got %#v", spec.Groups)
	}
	if spec.Groups[0].Requests[0].Path != "/assets/app.js" {
		t.Fatalf("expected static asset path, got %#v", spec.Groups[0].Requests[0])
	}
}

func TestHARParserRejectsInvalidDocument(t *testing.T) {
	t.Parallel()

	_, err := NewHARParser(false).Parse(context.Background(), []byte(`{"log":{}}`))
	if err == nil {
		t.Fatal("expected invalid HAR document to return an error")
	}
}
