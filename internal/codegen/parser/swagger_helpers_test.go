package parser

import (
	"context"
	"testing"

	"github.com/galax-io/galaxio-cli/internal/codegen"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v2 "github.com/pb33f/libopenapi/datamodel/high/v2"
	liborderedmap "github.com/pb33f/libopenapi/orderedmap"
)

func TestSwaggerParserParseRejectsInvalidDocument(t *testing.T) {
	t.Parallel()

	_, err := NewSwaggerParser().Parse(context.Background(), []byte("{"))
	if err == nil {
		t.Fatal("expected invalid swagger document to return an error")
	}
}

func TestExtractGroupsHandlesNilPaths(t *testing.T) {
	t.Parallel()

	groups, err := extractGroups(nil)
	if err != nil {
		t.Fatalf("extractGroups(nil) error = %v", err)
	}
	if groups != nil {
		t.Fatalf("expected nil groups for nil paths, got %#v", groups)
	}
}

func TestBuildBodySchemaHandlesNilProxy(t *testing.T) {
	t.Parallel()

	schema, err := buildBodySchema(nil, "ignored", true)
	if err != nil {
		t.Fatalf("buildBodySchema(nil) error = %v", err)
	}
	if schema != nil {
		t.Fatalf("expected nil schema for nil proxy, got %#v", schema)
	}
}

func TestSchemaTypeCoversBranches(t *testing.T) {
	t.Parallel()

	arrayItems := highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}})

	tests := []struct {
		name string
		in   *highbase.Schema
		want string
	}{
		{
			name: "explicit type wins",
			in:   &highbase.Schema{Type: []string{"integer"}},
			want: "integer",
		},
		{
			name: "object inferred from properties",
			in: &highbase.Schema{
				Properties: orderedProperties("name", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}})),
			},
			want: "object",
		},
		{
			name: "array inferred from items",
			in: &highbase.Schema{
				Items: &highbase.DynamicValue[*highbase.SchemaProxy, bool]{A: arrayItems},
			},
			want: "array",
		},
		{
			name: "empty schema",
			in:   &highbase.Schema{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := schemaType(tt.in); got != tt.want {
				t.Fatalf("schemaType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeAuthTypeCoversBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   *v2.SecurityScheme
		want string
	}{
		{
			name: "nil scheme",
			in:   nil,
			want: "",
		},
		{
			name: "basic auth",
			in:   &v2.SecurityScheme{Type: "basic"},
			want: "basic",
		},
		{
			name: "bearer disguised as api key",
			in: &v2.SecurityScheme{
				Type:        "apiKey",
				In:          "header",
				Name:        "Authorization",
				Description: "Use Bearer token",
			},
			want: "bearer",
		},
		{
			name: "plain api key",
			in: &v2.SecurityScheme{
				Type: "apiKey",
				In:   "query",
				Name: "api_key",
			},
			want: "apikey",
		},
		{
			name: "fallback lower case",
			in:   &v2.SecurityScheme{Type: "OAuth2"},
			want: "oauth2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeAuthType(tt.in); got != tt.want {
				t.Fatalf("normalizeAuthType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGroupNameForFallbackAndTags(t *testing.T) {
	t.Parallel()

	if got := groupNameFor("/api/v2/pets/{id}", &v2.Operation{Tags: []string{"pet-admin"}}); got != "petAdmin" {
		t.Fatalf("groupNameFor(tags) = %q, want %q", got, "petAdmin")
	}

	if got := groupNameFor("/api/v2/pets/{id}", &v2.Operation{}); got != "pets" {
		t.Fatalf("groupNameFor(fallback) = %q, want %q", got, "pets")
	}

	if got := groupNameFor("/", nil); got != "default" {
		t.Fatalf("groupNameFor(default) = %q, want %q", got, "default")
	}
}

func TestRequestNameForFallbacks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		op     *v2.Operation
		want   string
	}{
		{
			name:   "uses operation id",
			method: "get",
			path:   "/pets/{id}",
			op:     &v2.Operation{OperationId: "find pet by id"},
			want:   "findPetById",
		},
		{
			name:   "fallback from method and path",
			method: "get",
			path:   "/api/v2/pets/{id}",
			op:     &v2.Operation{},
			want:   "getPets",
		},
		{
			name:   "fallback when no meaningful segments",
			method: "get",
			path:   "/",
			op:     nil,
			want:   "getRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := requestNameFor(tt.method, tt.path, tt.op); got != tt.want {
				t.Fatalf("requestNameFor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitWordsSplitsPathLikeTokens(t *testing.T) {
	t.Parallel()

	words := codegen.SplitWords("  /{id}/pets")
	if len(words) != 2 || words[0] != "id" || words[1] != "pets" {
		t.Fatalf("expected path-like tokens to be split consistently, got %#v", words)
	}
}

func orderedProperties(name string, proxy *highbase.SchemaProxy) *liborderedmap.Map[string, *highbase.SchemaProxy] {
	properties := liborderedmap.New[string, *highbase.SchemaProxy]()
	properties.Set(name, proxy)
	return properties
}
