//go:build !windows

package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// maxStderrLen is the maximum length of stderr to include in error messages.
const maxStderrLen = 500

// absolutePathPattern matches Unix absolute paths for sanitization.
// Matches paths like /home/user/file.tf but not ./relative/path.
// Only matches paths preceded by whitespace or at start of string.
var absolutePathPattern = regexp.MustCompile(`(^|\s)(/(?:[a-zA-Z0-9._-]+/)+[a-zA-Z0-9._-]+)`)

// sanitizeStderr truncates and sanitizes stderr output for error messages.
// It truncates to maxStderrLen and replaces absolute paths with relative names.
func sanitizeStderr(stderr string) string {
	// Replace absolute paths with just the filename, preserving preceding whitespace
	sanitized := absolutePathPattern.ReplaceAllStringFunc(stderr, func(match string) string {
		// Find where the path starts (after any whitespace)
		pathStart := 0
		for i, c := range match {
			if c == '/' {
				pathStart = i
				break
			}
		}
		prefix := match[:pathStart]
		path := match[pathStart:]
		return prefix + filepath.Base(path)
	})

	// Truncate if too long
	if len(sanitized) > maxStderrLen {
		sanitized = sanitized[:maxStderrLen] + "... (truncated)"
	}

	return sanitized
}

// bashRuleTimeout is the maximum execution time for a Bash rule script.
const bashRuleTimeout = 30 * time.Second

// bashRuleOutput represents the JSON output from a Bash rule script.
type bashRuleOutput struct {
	Findings []bashFinding `json:"findings"`
}

// bashFinding represents a single finding from a Bash rule script.
type bashFinding struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Rule     string `json:"rule"`
}

// BashRule implements sdk.Rule by executing an external Bash script.
type BashRule struct {
	name string
	path string
	desc string
}

// NewBashRule creates a BashRule from a script path.
// The rule name is derived from the script filename (without extension).
func NewBashRule(path string) *BashRule {
	name := filepath.Base(path)
	name = name[:len(name)-len(filepath.Ext(name))]
	return &BashRule{
		name: name,
		path: path,
		desc: fmt.Sprintf("Bash rule: %s", name),
	}
}

// Name returns the rule name.
func (r *BashRule) Name() string { return r.name }

// Description returns the rule description.
func (r *BashRule) Description() string { return r.desc }

// Check executes the Bash script with the file path as an argument
// and parses the JSON output into findings.
func (r *BashRule) Check(ctx *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	if ctx.File == "" {
		return nil, nil
	}

	execCtx, cancel := context.WithTimeout(context.Background(), bashRuleTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "bash", r.path, ctx.File)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Exit code 1 means findings were reported (not a script error).
		// Any other error (exit code 2+, signal, timeout) is a real failure.
		exitErr := &exec.ExitError{}
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
			return nil, fmt.Errorf("executing bash rule %s: %w (stderr: %s)", r.name, err, sanitizeStderr(stderr.String()))
		}
	}

	if stdout.Len() == 0 {
		return nil, nil
	}

	var output bashRuleOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("parsing output from bash rule %s: %w", r.name, err)
	}

	findings := make([]sdk.Finding, 0, len(output.Findings))
	for _, f := range output.Findings {
		ruleName := f.Rule
		if ruleName == "" {
			ruleName = r.name
		}
		findings = append(findings, sdk.Finding{
			Rule:    ruleName,
			Message: f.Message,
			File:    f.File,
			Location: hcl.Range{
				Filename: f.File,
				Start:    hcl.Pos{Line: f.Line, Column: f.Column},
				End:      hcl.Pos{Line: f.Line, Column: f.Column},
			},
			Severity: parseSeverity(f.Severity),
		})
	}

	return findings, nil
}

// Fix is not supported for Bash rules.
func (r *BashRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

// loadBashRule loads a Bash rule from a script file.
// The script must be executable.
func loadBashRule(path string) (*BashRule, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("reading bash rule %s: %w", path, err)
	}

	// Check the file is executable (owner execute bit)
	if info.Mode()&0o100 == 0 {
		return nil, fmt.Errorf("bash rule %s is not executable (chmod +x to fix)", path)
	}

	return NewBashRule(path), nil
}
