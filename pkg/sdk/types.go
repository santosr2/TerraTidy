package sdk

import (
	"github.com/hashicorp/hcl/v2"
	"log"
)

// Severity represents the severity level of a finding
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Finding represents a rule violation or issue found in a file
type Finding struct {
	Rule     string    `json:"rule"`
	Message  string    `json:"message"`
	File     string    `json:"file"`
	Location hcl.Range `json:"location"`
	Severity Severity  `json:"severity"`
	Fixable  bool      `json:"fixable"`
	FixFunc  func() ([]byte, error) `json:"-"`
}

// Context provides context for rule execution
type Context struct {
	Config  map[string]interface{}
	Logger  *log.Logger
	WorkDir string
	File    string
}

// Rule defines the interface that all rules must implement
type Rule interface {
	Name() string
	Description() string
	Check(ctx *Context, file *hcl.File) ([]Finding, error)
	Fix(ctx *Context, file *hcl.File) ([]byte, error)
}

