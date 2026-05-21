// Package codegen defines the intermediate representation used by generate
// parsers before template rendering.
package codegen

// Spec is the top-level intermediate representation of an API specification.
type Spec struct {
	Title       string
	Version     string
	BasePath    string
	Groups      []Group
	AuthSchemes []AuthScheme
}

// Group collects related requests, usually by tag or a stable path segment.
type Group struct {
	Name     string
	Requests []Request
}

// Request describes one generated API request shape.
type Request struct {
	Method    string
	Path      string
	Name      string
	Summary   string
	Headers   []Header
	Params    []Param
	Body      *BodySchema
	Responses []Response
}

// BodySchema captures a recursive request body schema tree.
type BodySchema struct {
	Name     string
	Type     string
	Format   string
	Required bool
	Fields   []BodySchema
	Items    *BodySchema
}

// AuthScheme describes one supported authentication scheme.
type AuthScheme struct {
	Key      string
	Type     string
	Name     string
	Location string
}

// Param describes one non-header request parameter.
type Param struct {
	Name     string
	In       string
	Type     string
	Format   string
	Required bool
}

// Header describes one request header requirement.
type Header struct {
	Name     string
	Value    string
	Required bool
}

// Response captures one response status and description from the source spec.
type Response struct {
	StatusCode  string
	Description string
}
