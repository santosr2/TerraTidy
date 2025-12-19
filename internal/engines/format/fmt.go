// Package format provides the formatting engine for TerraTidy.
// It uses HCL's hclwrite package to format Terraform configuration files
// according to the canonical HCL style.
package format

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/santosr2/terratidy/pkg/sdk"
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
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Format using hclwrite
	formatted := hclwrite.Format(content)

	// Check if formatting changed anything
	if string(formatted) == string(content) {
		return nil, nil // Already formatted
	}

	// In check mode, return a finding
	if e.config.Check {
		return &sdk.Finding{
			Rule:     "fmt.needs-formatting",
			Message:  "File needs formatting",
			File:     path,
			Severity: sdk.SeverityError,
			Fixable:  true,
			FixFunc: func() ([]byte, error) {
				return formatted, nil
			},
		}, nil
	}

	// In normal mode, write the formatted content
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return nil, fmt.Errorf("writing formatted file: %w", err)
	}

	return &sdk.Finding{
		Rule:     "fmt.formatted",
		Message:  "File formatted successfully",
		File:     path,
		Severity: sdk.SeverityInfo,
		Fixable:  false,
	}, nil
}

// isHCLFile checks if a file is an HCL file (.tf or .hcl)
func isHCLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".tf" || ext == ".hcl"
}

// Format formats the given content and returns the formatted result
func Format(content []byte) []byte {
	return hclwrite.Format(content)
}
