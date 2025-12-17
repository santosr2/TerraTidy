package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/santosr2/terratidy/pkg/sdk"
)

// Formatter defines the interface for output formatters
type Formatter interface {
	Format(findings []sdk.Finding, w io.Writer) error
}

// TextFormatter outputs findings in human-readable text format
type TextFormatter struct {
	Verbose bool
}

// Format implements the Formatter interface for text output
func (f *TextFormatter) Format(findings []sdk.Finding, w io.Writer) error {
	if len(findings) == 0 {
		fmt.Fprintln(w, "✓ No issues found")
		return nil
	}

	for _, finding := range findings {
		icon := "ℹ"
		switch finding.Severity {
		case sdk.SeverityError:
			icon = "✗"
		case sdk.SeverityWarning:
			icon = "⚠"
		case sdk.SeverityInfo:
			icon = "ℹ"
		}

		if f.Verbose {
			fmt.Fprintf(w, "%s %s:%d:%d: %s (%s)\n",
				icon,
				finding.File,
				finding.Location.Start.Line,
				finding.Location.Start.Column,
				finding.Message,
				finding.Rule,
			)
		} else {
			fmt.Fprintf(w, "%s %s: %s (%s)\n",
				icon,
				finding.File,
				finding.Message,
				finding.Rule,
			)
		}
	}

	return nil
}

// JSONFormatter outputs findings in JSON format
type JSONFormatter struct {
	Pretty bool
}

// JSONOutput represents the JSON output structure
type JSONOutput struct {
	Findings []JSONFinding `json:"findings"`
	Summary  JSONSummary   `json:"summary"`
}

// JSONFinding represents a single finding in JSON format
type JSONFinding struct {
	Rule     string       `json:"rule"`
	Message  string       `json:"message"`
	File     string       `json:"file"`
	Location JSONLocation `json:"location"`
	Severity string       `json:"severity"`
	Fixable  bool         `json:"fixable"`
}

// JSONLocation represents a location in JSON format
type JSONLocation struct {
	Start JSONPosition `json:"start"`
	End   JSONPosition `json:"end"`
}

// JSONPosition represents a position in JSON format
type JSONPosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// JSONSummary represents the summary in JSON format
type JSONSummary struct {
	Total    int `json:"total"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

// Format implements the Formatter interface for JSON output
func (f *JSONFormatter) Format(findings []sdk.Finding, w io.Writer) error {
	output := JSONOutput{
		Findings: make([]JSONFinding, 0, len(findings)),
		Summary: JSONSummary{
			Total: len(findings),
		},
	}

	for _, finding := range findings {
		output.Findings = append(output.Findings, JSONFinding{
			Rule:    finding.Rule,
			Message: finding.Message,
			File:    finding.File,
			Location: JSONLocation{
				Start: JSONPosition{
					Line:   finding.Location.Start.Line,
					Column: finding.Location.Start.Column,
				},
				End: JSONPosition{
					Line:   finding.Location.End.Line,
					Column: finding.Location.End.Column,
				},
			},
			Severity: string(finding.Severity),
			Fixable:  finding.Fixable,
		})

		// Count by severity
		switch finding.Severity {
		case sdk.SeverityError:
			output.Summary.Errors++
		case sdk.SeverityWarning:
			output.Summary.Warnings++
		case sdk.SeverityInfo:
			output.Summary.Info++
		}
	}

	encoder := json.NewEncoder(w)
	if f.Pretty {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(output)
}

// GetFormatter returns the appropriate formatter based on the format string
func GetFormatter(format string, verbose bool, version string) (Formatter, error) {
	switch format {
	case "text", "":
		return &TextFormatter{Verbose: verbose}, nil
	case "json":
		return &JSONFormatter{Pretty: true}, nil
	case "json-compact":
		return &JSONFormatter{Pretty: false}, nil
	case "sarif":
		return &SARIFFormatter{Version: version}, nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", format)
	}
}
