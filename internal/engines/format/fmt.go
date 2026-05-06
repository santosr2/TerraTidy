// Package format provides the formatting engine for TerraTidy.
// It uses HCL's hclwrite package to format Terraform configuration files
// according to the canonical HCL style.
package format

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/santosr2/TerraTidy/internal/cache"
	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// Engine represents the formatter engine
type Engine struct {
	config *Config
}

// Config holds the formatter configuration
type Config struct {
	Check bool // Check mode (don't modify files)
	Diff  bool // Show diff of changes
}

// ConfigFromEngine creates a format.Config from the config package's FmtEngineConfig.
// This converts the typed config struct used for YAML parsing into the engine's
// internal Config type.
func ConfigFromEngine(engineCfg config.FmtEngineConfig) *Config {
	return &Config{
		Check: engineCfg.Check,
		Diff:  engineCfg.Diff,
	}
}

// New creates a new formatter engine
func New(config *Config) *Engine {
	if config == nil {
		config = &Config{}
	}
	return &Engine{config: config}
}

// Name returns the engine name
func (e *Engine) Name() string {
	return "fmt"
}

// Run executes the formatter on the given files
func (e *Engine) Run(ctx context.Context, files []string) ([]sdk.Finding, error) {
	var findings []sdk.Finding

	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Skip non-HCL files
		if !isHCLFile(file) {
			continue
		}

		result, err := e.formatFile(file)
		if err != nil {
			return nil, fmt.Errorf("formatting %s: %w", file, err)
		}

		if result != nil {
			findings = append(findings, *result)
		}
	}

	return findings, nil
}

// formatFile formats a single file and returns a finding if changes are needed
func (e *Engine) formatFile(path string) (*sdk.Finding, error) {
	// Try to get content from cache first
	var content []byte
	var err error

	if entry, cacheErr := cache.Default().GetOrParse(path); cacheErr == nil && entry != nil {
		content = entry.Content
	} else {
		// Fallback to direct read
		content, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}
	}

	// Format using hclwrite
	formatted := hclwrite.Format(content)

	// Check if formatting changed anything
	if bytes.Equal(formatted, content) {
		return nil, nil // Already formatted
	}

	// Generate unified diff if requested
	var diffText string
	if e.config.Diff {
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(string(content)),
			B:        difflib.SplitLines(string(formatted)),
			FromFile: path,
			ToFile:   path,
			Context:  3,
		}
		diffText, err = difflib.GetUnifiedDiffString(diff)
		if err != nil {
			return nil, fmt.Errorf("generating diff: %w", err)
		}
	}

	// In check mode, return a finding without writing
	if e.config.Check {
		message := "File needs formatting"
		if diffText != "" {
			message = diffText
		}
		return &sdk.Finding{
			Rule:     "fmt.needs-formatting",
			Message:  message,
			File:     path,
			Severity: sdk.SeverityError,
			Fixable:  true,
			IsDiff:   diffText != "",
		}, nil
	}

	// In normal mode, write the formatted content while preserving the file's
	// existing permission bits. Fall back to 0o600 if Stat fails.
	if err := writeFilePreservingMode(path, formatted); err != nil {
		return nil, fmt.Errorf("writing formatted file: %w", err)
	}

	message := "File formatted successfully"
	if diffText != "" {
		message = diffText
	}
	return &sdk.Finding{
		Rule:     "fmt.formatted",
		Message:  message,
		File:     path,
		Severity: sdk.SeverityInfo,
		IsDiff:   diffText != "",
	}, nil
}

// isHCLFile checks if a file has a Terraform/HCL extension.
func isHCLFile(path string) bool {
	return sdk.IsHCLFile(path)
}

// Format formats the given content and returns the formatted result
func Format(content []byte) []byte {
	return hclwrite.Format(content)
}

// writeFilePreservingMode writes content to path, preserving the file's
// existing permission bits. If Stat fails (e.g., a concurrent modification or
// permissions issue), it falls back to mode 0o600. A trailing Chmod ensures
// the captured mode wins even on the rare path where WriteFile recreates the
// file (and any umask would otherwise apply).
func writeFilePreservingMode(path string, content []byte) error {
	mode := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}
