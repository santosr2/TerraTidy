// Package sdk provides the core types and interfaces for TerraTidy rules and engines.
// It defines the Finding, Context, and Rule types that all engines and rules use
// to report issues and apply fixes to Terraform configurations.
package sdk

import (
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

// Severity represents the severity level of a finding.
type Severity string

// Severity constants define the available severity levels for findings.
// Error-severity findings cause a non-zero exit code. Warning and info
// findings are reported but do not fail by default.
const (
	// SeverityError indicates a critical issue that must be fixed.
	SeverityError Severity = "error"
	// SeverityWarning indicates a best-practice violation that should be fixed.
	SeverityWarning Severity = "warning"
	// SeverityInfo indicates a suggestion or informational finding.
	SeverityInfo Severity = "info"
)

// ParseSeverity converts a string to a Severity value.
// Returns defaultSev if the string does not match a known severity.
func ParseSeverity(s string, defaultSev Severity) Severity {
	switch strings.ToLower(s) {
	case "error":
		return SeverityError
	case "warning":
		return SeverityWarning
	case "info":
		return SeverityInfo
	default:
		return defaultSev
	}
}

// Finding represents a single issue detected by a rule. Findings are collected
// by engines and formatted for output. Each finding identifies the rule that
// produced it, the file and location where the issue was found, and optionally
// a function to auto-fix the issue.
type Finding struct {
	// Rule is the identifier of the rule that produced this finding (e.g., "style.block-label-case").
	Rule string `json:"rule"`
	// Message is a human-readable description of the issue, shown to the user.
	Message string `json:"message"`
	// File is the path to the file where the issue was found.
	File string `json:"file"`
	// Location is the line/column range from the HCL parser (Start and End positions).
	Location hcl.Range `json:"location"`
	// Severity indicates the importance of this finding.
	Severity Severity `json:"severity"`
	// Fixable indicates whether this finding can be auto-fixed via FixFunc.
	Fixable bool `json:"fixable"`
	// FixFunc applies the fix and returns the corrected file content.
	// Not serialized to JSON. May be nil even when Fixable is true (e.g., YAML rules).
	FixFunc func() ([]byte, error) `json:"-"`
}

// Context provides runtime information to rules during execution. Each rule
// invocation receives a Context with the current file path, working directory,
// and any rule-specific configuration from .terratidy.yaml.
type Context struct {
	// Config holds rule-specific options from the "options" map in .terratidy.yaml.
	Config map[string]any
	// Logger for diagnostic output (not user-facing findings).
	Logger *log.Logger
	// WorkDir is the directory TerraTidy was invoked from.
	WorkDir string
	// File is the absolute path to the file being checked.
	File string
}

// Rule defines the interface that all rules must implement. Built-in rules,
// Go plugin rules, YAML rules, and Bash rules all satisfy this interface.
// The Check method detects issues; the Fix method applies corrections.
type Rule interface {
	// Name returns a unique identifier for the rule (e.g., "style.block-label-case").
	Name() string
	// Description returns a human-readable description of what the rule checks.
	Description() string
	// Check evaluates the rule against a parsed HCL file and returns any findings.
	// Return nil findings and nil error if the file passes the check.
	Check(ctx *Context, file *hcl.File) ([]Finding, error)
	// Fix applies an automatic fix and returns the corrected file content as bytes.
	// Return nil, nil if the rule does not support auto-fixing.
	Fix(ctx *Context, file *hcl.File) ([]byte, error)
}
