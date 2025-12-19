package output

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/santosr2/terratidy/pkg/sdk"
)

// SARIFFormatter outputs findings in SARIF format for GitHub Code Scanning
type SARIFFormatter struct {
	Version string // TerraTidy version
}

// SARIF represents the root SARIF document
type SARIF struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []SARIFRun `json:"runs"`
}

// SARIFRun represents a single run of the tool
type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

// SARIFTool represents the tool information
type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

// SARIFDriver represents the tool driver
type SARIFDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []SARIFRule `json:"rules,omitempty"`
}

// SARIFRule represents a rule definition
type SARIFRule struct {
	ID               string              `json:"id"`
	ShortDescription SARIFMessage        `json:"shortDescription"`
	FullDescription  SARIFMessage        `json:"fullDescription,omitempty"`
	HelpURI          string              `json:"helpUri,omitempty"`
	Properties       SARIFRuleProperties `json:"properties,omitempty"`
}

// SARIFRuleProperties represents rule properties
type SARIFRuleProperties struct {
	Tags []string `json:"tags,omitempty"`
}

// SARIFMessage represents a message
type SARIFMessage struct {
	Text string `json:"text"`
}

// SARIFResult represents a single result/finding
type SARIFResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   SARIFMessage    `json:"message"`
	Locations []SARIFLocation `json:"locations"`
	Fixes     []SARIFFix      `json:"fixes,omitempty"`
}

// SARIFLocation represents a location in the source
type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

// SARIFPhysicalLocation represents a physical location
type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
	Region           SARIFRegion           `json:"region"`
}

// SARIFArtifactLocation represents an artifact location
type SARIFArtifactLocation struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId,omitempty"`
}

// SARIFRegion represents a region in the source
type SARIFRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
	EndLine     int `json:"endLine,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}

// SARIFFix represents a fix for a result
type SARIFFix struct {
	Description     SARIFMessage          `json:"description"`
	ArtifactChanges []SARIFArtifactChange `json:"artifactChanges"`
}

// SARIFArtifactChange represents a change to an artifact
type SARIFArtifactChange struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
	Replacements     []SARIFReplacement    `json:"replacements"`
}

// SARIFReplacement represents a replacement
type SARIFReplacement struct {
	DeletedRegion   SARIFRegion  `json:"deletedRegion"`
	InsertedContent SARIFMessage `json:"insertedContent,omitempty"`
}

// Format implements the Formatter interface for SARIF output
func (f *SARIFFormatter) Format(findings []sdk.Finding, w io.Writer) error {
	rules := buildSARIFRules(findings)
	results := buildSARIFResults(findings)
	sarif := f.buildSARIFDocument(rules, results)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sarif)
}

func buildSARIFRules(findings []sdk.Finding) []SARIFRule {
	rulesMap := make(map[string]bool)
	for _, finding := range findings {
		rulesMap[finding.Rule] = true
	}

	var rules []SARIFRule
	for ruleID := range rulesMap {
		rules = append(rules, SARIFRule{
			ID: ruleID,
			ShortDescription: SARIFMessage{
				Text: ruleID,
			},
			Properties: SARIFRuleProperties{
				Tags: []string{"terraform", "terragrunt", "quality"},
			},
		})
	}
	return rules
}

func buildSARIFResults(findings []sdk.Finding) []SARIFResult {
	var results []SARIFResult
	for _, finding := range findings {
		result := buildSARIFResult(finding)
		results = append(results, result)
	}
	return results
}

func buildSARIFResult(finding sdk.Finding) SARIFResult {
	result := SARIFResult{
		RuleID: finding.Rule,
		Level:  sarifLevel(finding.Severity),
		Message: SARIFMessage{
			Text: finding.Message,
		},
		Locations: []SARIFLocation{
			{
				PhysicalLocation: SARIFPhysicalLocation{
					ArtifactLocation: SARIFArtifactLocation{
						URI:       filepath.ToSlash(finding.File),
						URIBaseID: "%SRCROOT%",
					},
					Region: SARIFRegion{
						StartLine:   finding.Location.Start.Line,
						StartColumn: finding.Location.Start.Column,
						EndLine:     finding.Location.End.Line,
						EndColumn:   finding.Location.End.Column,
					},
				},
			},
		},
	}

	if finding.Fixable && finding.FixFunc != nil {
		result.Fixes = buildSARIFFixes(finding)
	}
	return result
}

func buildSARIFFixes(finding sdk.Finding) []SARIFFix {
	return []SARIFFix{
		{
			Description: SARIFMessage{
				Text: fmt.Sprintf("Auto-fix available for %s", finding.Rule),
			},
			ArtifactChanges: []SARIFArtifactChange{
				{
					ArtifactLocation: SARIFArtifactLocation{
						URI:       filepath.ToSlash(finding.File),
						URIBaseID: "%SRCROOT%",
					},
					Replacements: []SARIFReplacement{
						{
							DeletedRegion: SARIFRegion{
								StartLine:   finding.Location.Start.Line,
								StartColumn: finding.Location.Start.Column,
								EndLine:     finding.Location.End.Line,
								EndColumn:   finding.Location.End.Column,
							},
						},
					},
				},
			},
		},
	}
}

func (f *SARIFFormatter) buildSARIFDocument(rules []SARIFRule, results []SARIFResult) SARIF {
	return SARIF{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:           "TerraTidy",
						Version:        f.Version,
						InformationURI: "https://github.com/santosr2/terratidy",
						Rules:          rules,
					},
				},
				Results: results,
			},
		},
	}
}

// sarifLevel converts SDK severity to SARIF level
func sarifLevel(severity sdk.Severity) string {
	switch severity {
	case sdk.SeverityError:
		return "error"
	case sdk.SeverityWarning:
		return "warning"
	case sdk.SeverityInfo:
		return "note"
	default:
		return "warning"
	}
}
