package templates

import "embed"

// Files contains embedded code generation templates.
//
//go:embed scala-sbt/*.tmpl
var Files embed.FS
