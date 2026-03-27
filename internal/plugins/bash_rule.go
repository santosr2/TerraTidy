package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/terratidy/pkg/sdk"
)

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
	name   string
	path   string
	desc   string
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

	cmd := exec.CommandContext(execCtx, "bash", r.path, ctx.File) //nolint:gosec // user-provided rule scripts are trusted
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Exit code 1 with output means findings were reported (not a script error)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && stdout.Len() > 0 {
			// fall through to parse output
		} else {
			return nil, fmt.Errorf("executing bash rule %s: %w (stderr: %s)", r.name, err, stderr.String())
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
