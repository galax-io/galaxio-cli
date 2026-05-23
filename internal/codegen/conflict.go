package codegen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// IfExistsSuffix writes a sibling *.generated file when the target already exists.
	IfExistsSuffix = "suffix"
	// IfExistsMerge writes conflict markers into the existing target file.
	IfExistsMerge = "merge"
	// IfExistsSkip leaves the existing target untouched.
	IfExistsSkip = "skip"
	// IfExistsOverwrite replaces the existing target file completely.
	IfExistsOverwrite = "overwrite"
)

// ConflictResult reports how one file write was resolved.
type ConflictResult struct {
	Path        string
	WrittenPath string
	Status      string
}

// ValidateIfExistsStrategy validates supported conflict strategies.
func ValidateIfExistsStrategy(strategy string) error {
	switch strings.TrimSpace(strategy) {
	case IfExistsSuffix, IfExistsMerge, IfExistsSkip, IfExistsOverwrite:
		return nil
	default:
		return fmt.Errorf("unknown if-exists strategy %q", strategy)
	}
}

// WriteFileWithStrategy writes content to targetPath honoring the selected conflict strategy.
func WriteFileWithStrategy(targetPath string, content []byte, strategy string) (ConflictResult, error) {
	if err := ValidateIfExistsStrategy(strategy); err != nil {
		return ConflictResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return ConflictResult{}, err
	}

	existing, err := os.ReadFile(targetPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return ConflictResult{}, err
		}
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			return ConflictResult{}, err
		}
		return ConflictResult{Path: targetPath, WrittenPath: targetPath, Status: "written"}, nil
	}

	switch strategy {
	case IfExistsSuffix:
		generatedPath := generatedSiblingPath(targetPath)
		if err := os.WriteFile(generatedPath, content, 0o644); err != nil {
			return ConflictResult{}, err
		}
		return ConflictResult{Path: targetPath, WrittenPath: generatedPath, Status: "written"}, nil
	case IfExistsMerge:
		merged := mergeConflictContent(content, existing)
		if err := os.WriteFile(targetPath, merged, 0o644); err != nil {
			return ConflictResult{}, err
		}
		return ConflictResult{Path: targetPath, WrittenPath: targetPath, Status: "conflict"}, nil
	case IfExistsSkip:
		return ConflictResult{Path: targetPath, WrittenPath: targetPath, Status: "skipped"}, nil
	case IfExistsOverwrite:
		if bytes.Equal(existing, content) {
			return ConflictResult{Path: targetPath, WrittenPath: targetPath, Status: "overwritten"}, nil
		}
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			return ConflictResult{}, err
		}
		return ConflictResult{Path: targetPath, WrittenPath: targetPath, Status: "overwritten"}, nil
	default:
		return ConflictResult{}, fmt.Errorf("unknown if-exists strategy %q", strategy)
	}
}

func generatedSiblingPath(targetPath string) string {
	ext := filepath.Ext(targetPath)
	base := strings.TrimSuffix(targetPath, ext)
	if ext == "" {
		return base + ".generated"
	}
	return base + ".generated" + ext
}

func mergeConflictContent(generated []byte, existing []byte) []byte {
	var builder strings.Builder
	builder.WriteString("<<<< generated\n")
	builder.Write(generated)
	if len(generated) == 0 || generated[len(generated)-1] != '\n' {
		builder.WriteByte('\n')
	}
	builder.WriteString("====\n")
	builder.Write(existing)
	if len(existing) == 0 || existing[len(existing)-1] != '\n' {
		builder.WriteByte('\n')
	}
	builder.WriteString(">>>> existing\n")
	return []byte(builder.String())
}
