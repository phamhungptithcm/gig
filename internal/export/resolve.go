package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ResolveOptions struct {
	RawFormat      string
	FormatExplicit bool
	JSONOutput     bool
	OutputPath     string
	DefaultFormat  Format
}

type FormatConflictError struct {
	Requested  Format
	Inferred   Format
	OutputPath string
}

func (e *FormatConflictError) Error() string {
	return fmt.Sprintf("export format %q does not match output file %q", e.Requested, filepath.Base(e.OutputPath))
}

func ResolveOutputFormat(options ResolveOptions) (ResolvedOutput, error) {
	format := normalizeFormat(options.RawFormat, options.DefaultFormat)
	formatExplicit := options.FormatExplicit
	if options.JSONOutput {
		format = FormatJSON
		formatExplicit = true
	}

	outputPath := strings.TrimSpace(options.OutputPath)
	if outputPath == "" {
		return ResolvedOutput{Format: format, Target: TargetStdout}, nil
	}

	inferred, target, ok := InferOutputFormat(outputPath)
	if ok {
		if formatExplicit && format != inferred {
			return ResolvedOutput{}, &FormatConflictError{
				Requested:  format,
				Inferred:   inferred,
				OutputPath: outputPath,
			}
		}
		if !formatExplicit {
			format = inferred
		}
		return ResolvedOutput{Format: format, Target: target, Path: outputPath}, nil
	}

	if !formatExplicit {
		return ResolvedOutput{Format: format, Target: TargetFile, Path: outputPath}, nil
	}
	if format == FormatCSV && outputLooksLikeDirectory(outputPath) {
		return ResolvedOutput{Format: format, Target: TargetDirectory, Path: outputPath}, nil
	}
	return ResolvedOutput{Format: format, Target: TargetFile, Path: outputPath}, nil
}

func InferOutputFormat(outputPath string) (Format, TargetKind, bool) {
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return "", "", false
	}
	if outputLooksLikeDirectory(outputPath) {
		return FormatCSV, TargetDirectory, true
	}
	ext := strings.ToLower(filepath.Ext(strings.TrimRight(outputPath, `/\`)))
	switch ext {
	case ".xlsx":
		return FormatXLSX, TargetFile, true
	case ".csv":
		return FormatCSV, TargetFile, true
	case ".json":
		return FormatJSON, TargetFile, true
	default:
		return "", TargetFile, false
	}
}

func normalizeFormat(raw string, fallback Format) Format {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return fallback
	}
	return Format(raw)
}

func outputLooksLikeDirectory(outputPath string) bool {
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return false
	}
	if strings.HasSuffix(outputPath, "/") || strings.HasSuffix(outputPath, `\`) {
		return true
	}
	info, err := os.Stat(outputPath)
	return err == nil && info.IsDir()
}
