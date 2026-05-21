// Package parser turns API specifications into the internal code generation
// intermediate representation.
package parser

import (
	"context"

	"github.com/galax-io/galaxio-cli/internal/codegen"
)

// Parser converts raw API specification bytes into the shared IR model.
type Parser interface {
	Parse(ctx context.Context, data []byte) (*codegen.Spec, error)
}
