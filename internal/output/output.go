package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

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
	case "html":
		return &HTMLFormatter{Title: "TerraTidy Report", Version: version}, nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", format)
	}
}

// HTMLFormatter outputs findings as an HTML report
type HTMLFormatter struct {
	Title   string
	Version string
}

// Format implements the Formatter interface for HTML output
func (f *HTMLFormatter) Format(findings []sdk.Finding, w io.Writer) error {
	// Count by severity
	var errors, warnings, info int
	for _, finding := range findings {
		switch finding.Severity {
		case sdk.SeverityError:
			errors++
		case sdk.SeverityWarning:
			warnings++
		case sdk.SeverityInfo:
			info++
		}
	}

	// Group findings by file
	byFile := make(map[string][]sdk.Finding)
	for _, finding := range findings {
		byFile[finding.File] = append(byFile[finding.File], finding)
	}

	// Generate HTML
	html := f.generateHTML(findings, byFile, errors, warnings, info)
	_, err := w.Write([]byte(html))
	return err
}

func (f *HTMLFormatter) generateHTML(findings []sdk.Finding, byFile map[string][]sdk.Finding, errors, warnings, info int) string {
	total := len(findings)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        :root {
            --error-color: #dc3545;
            --warning-color: #ffc107;
            --info-color: #17a2b8;
            --success-color: #28a745;
            --bg-color: #f8f9fa;
            --card-bg: #ffffff;
            --text-color: #212529;
            --border-color: #dee2e6;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: var(--bg-color);
            color: var(--text-color);
            line-height: 1.6;
            padding: 2rem;
        }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { margin-bottom: 1rem; color: #2c3e50; }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }
        .summary-card {
            background: var(--card-bg);
            border-radius: 8px;
            padding: 1rem;
            text-align: center;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .summary-card.error { border-left: 4px solid var(--error-color); }
        .summary-card.warning { border-left: 4px solid var(--warning-color); }
        .summary-card.info { border-left: 4px solid var(--info-color); }
        .summary-card.total { border-left: 4px solid #6c757d; }
        .summary-card .number { font-size: 2rem; font-weight: bold; }
        .summary-card .label { font-size: 0.875rem; color: #6c757d; }
        .file-section {
            background: var(--card-bg);
            border-radius: 8px;
            margin-bottom: 1rem;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .file-header {
            background: #2c3e50;
            color: white;
            padding: 0.75rem 1rem;
            font-family: monospace;
            font-size: 0.9rem;
        }
        .finding {
            padding: 1rem;
            border-bottom: 1px solid var(--border-color);
            display: flex;
            align-items: flex-start;
            gap: 1rem;
        }
        .finding:last-child { border-bottom: none; }
        .finding-icon {
            width: 24px;
            height: 24px;
            border-radius: 50%%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
            color: white;
            flex-shrink: 0;
        }
        .finding-icon.error { background: var(--error-color); }
        .finding-icon.warning { background: var(--warning-color); color: #212529; }
        .finding-icon.info { background: var(--info-color); }
        .finding-content { flex: 1; }
        .finding-message { margin-bottom: 0.25rem; }
        .finding-meta {
            font-size: 0.8rem;
            color: #6c757d;
            font-family: monospace;
        }
        .finding-rule { color: #6c757d; }
        .finding-location { margin-left: 1rem; }
        .badge {
            display: inline-block;
            padding: 0.2rem 0.5rem;
            border-radius: 4px;
            font-size: 0.75rem;
            font-weight: 500;
        }
        .badge-fixable { background: #d4edda; color: #155724; }
        .no-issues {
            text-align: center;
            padding: 3rem;
            color: var(--success-color);
        }
        .no-issues svg { width: 64px; height: 64px; margin-bottom: 1rem; }
        footer {
            text-align: center;
            margin-top: 2rem;
            color: #6c757d;
            font-size: 0.875rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>%s</h1>
        %s
        %s
        <footer>Generated by TerraTidy %s</footer>
    </div>
</body>
</html>`,
		f.Title,
		f.Title,
		f.generateSummary(total, errors, warnings, info),
		f.generateFindings(findings, byFile),
		f.Version,
	)
}

func (f *HTMLFormatter) generateSummary(total, errors, warnings, info int) string {
	return fmt.Sprintf(`
        <div class="summary">
            <div class="summary-card total">
                <div class="number">%d</div>
                <div class="label">Total Issues</div>
            </div>
            <div class="summary-card error">
                <div class="number">%d</div>
                <div class="label">Errors</div>
            </div>
            <div class="summary-card warning">
                <div class="number">%d</div>
                <div class="label">Warnings</div>
            </div>
            <div class="summary-card info">
                <div class="number">%d</div>
                <div class="label">Info</div>
            </div>
        </div>`, total, errors, warnings, info)
}

func (f *HTMLFormatter) generateFindings(findings []sdk.Finding, byFile map[string][]sdk.Finding) string {
	if len(findings) == 0 {
		return `
        <div class="no-issues">
            <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
            </svg>
            <h2>All checks passed!</h2>
            <p>No issues found in your Terraform code.</p>
        </div>`
	}

	var sections string
	for file, fileFindings := range byFile {
		sections += f.generateFileSection(file, fileFindings)
	}
	return sections
}

func (f *HTMLFormatter) generateFileSection(file string, findings []sdk.Finding) string {
	var findingsHTML string
	for _, finding := range findings {
		findingsHTML += f.generateFindingHTML(finding)
	}

	return fmt.Sprintf(`
        <div class="file-section">
            <div class="file-header">%s</div>
            %s
        </div>`, escapeHTML(file), findingsHTML)
}

func (f *HTMLFormatter) generateFindingHTML(finding sdk.Finding) string {
	iconClass := "info"
	iconSymbol := "i"
	switch finding.Severity {
	case sdk.SeverityError:
		iconClass = "error"
		iconSymbol = "!"
	case sdk.SeverityWarning:
		iconClass = "warning"
		iconSymbol = "!"
	}

	fixableBadge := ""
	if finding.Fixable {
		fixableBadge = `<span class="badge badge-fixable">Fixable</span>`
	}

	return fmt.Sprintf(`
            <div class="finding">
                <div class="finding-icon %s">%s</div>
                <div class="finding-content">
                    <div class="finding-message">%s %s</div>
                    <div class="finding-meta">
                        <span class="finding-rule">%s</span>
                        <span class="finding-location">Line %d, Column %d</span>
                    </div>
                </div>
            </div>`,
		iconClass,
		iconSymbol,
		escapeHTML(finding.Message),
		fixableBadge,
		escapeHTML(finding.Rule),
		finding.Location.Start.Line,
		finding.Location.Start.Column,
	)
}

// escapeHTML escapes special HTML characters
func escapeHTML(s string) string {
	// Use strings.Replacer to avoid infinite loop when & is replaced with &amp;
	// The replacer processes left-to-right, replacing each match exactly once
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}
