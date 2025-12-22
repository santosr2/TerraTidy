// Package main provides rule management commands for TerraTidy.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/santosr2/terratidy/internal/engines/lint"
	"github.com/santosr2/terratidy/internal/engines/style"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	rulesEngine  string
	rulesVerbose bool
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Rule management commands",
	Long: `Manage and inspect TerraTidy rules.

Use subcommands to list available rules or generate documentation.`,
}

var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available rules",
	Long: `Display all built-in and custom rules.

Shows rule names, descriptions, default severities, and whether they're enabled.`,
	Example: `  # List all rules
  terratidy rules list

  # List rules for specific engine
  terratidy rules list --engine style

  # List with detailed descriptions
  terratidy rules list --verbose`,
	RunE: runRulesList,
}

var rulesDocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate rule documentation",
	Long: `Generate markdown documentation for all rules.

This creates documentation that can be used for reference or published.`,
	Example: `  # Generate docs to stdout
  terratidy rules docs

  # Generate docs for specific engine
  terratidy rules docs --engine lint`,
	RunE: runRulesDocs,
}

func init() {
	rulesListCmd.Flags().StringVar(&rulesEngine, "engine", "", "filter by engine (style|lint|policy)")
	rulesListCmd.Flags().BoolVarP(&rulesVerbose, "verbose", "v", false, "show detailed descriptions")

	rulesDocsCmd.Flags().StringVar(&rulesEngine, "engine", "", "filter by engine (style|lint|policy)")

	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesDocsCmd)
	rootCmd.AddCommand(rulesCmd)
}

// RuleInfo holds information about a rule for display.
type RuleInfo struct {
	Name        string
	Description string
	Engine      string
	Severity    string
	Enabled     bool
}

func runRulesList(_ *cobra.Command, _ []string) error {
	rules := getAllRules()

	// Filter by engine if specified
	if rulesEngine != "" {
		var filtered []RuleInfo
		for _, r := range rules {
			if strings.EqualFold(r.Engine, rulesEngine) {
				filtered = append(filtered, r)
			}
		}
		rules = filtered
	}

	if len(rules) == 0 {
		fmt.Println("No rules found")
		return nil
	}

	// Sort rules by engine then name
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Engine != rules[j].Engine {
			return rules[i].Engine < rules[j].Engine
		}
		return rules[i].Name < rules[j].Name
	})

	fmt.Printf("Available rules (%d total):\n\n", len(rules))

	// Group by engine
	currentEngine := ""
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	caser := cases.Title(language.English)
	for _, rule := range rules {
		if rule.Engine != currentEngine {
			if currentEngine != "" {
				_, _ = fmt.Fprintln(w)
			}
			currentEngine = rule.Engine
			_, _ = fmt.Fprintf(w, "%s Engine:\n", caser.String(currentEngine))
			_, _ = fmt.Fprintf(w, "  NAME\tSEVERITY\tDESCRIPTION\n")
			_, _ = fmt.Fprintf(w, "  ----\t--------\t-----------\n")
		}

		severity := rule.Severity
		if severity == "" {
			severity = "warning"
		}

		desc := rule.Description
		if !rulesVerbose && len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\n", rule.Name, severity, desc)
	}

	_ = w.Flush()

	fmt.Println()
	fmt.Println("Use 'terratidy rules list --verbose' for full descriptions")
	fmt.Println("Use 'terratidy rules docs' to generate markdown documentation")
	fmt.Println()
	fmt.Println("Note: TFLint rules are configured via .tflint.hcl")
	fmt.Println("      Run 'tflint --help' to see available TFLint rules and options")

	return nil
}

func runRulesDocs(_ *cobra.Command, _ []string) error {
	rules := getAllRules()

	// Filter by engine if specified
	if rulesEngine != "" {
		var filtered []RuleInfo
		for _, r := range rules {
			if strings.EqualFold(r.Engine, rulesEngine) {
				filtered = append(filtered, r)
			}
		}
		rules = filtered
	}

	// Sort rules by engine then name
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Engine != rules[j].Engine {
			return rules[i].Engine < rules[j].Engine
		}
		return rules[i].Name < rules[j].Name
	})

	// Generate markdown documentation
	fmt.Println("# TerraTidy Rules Reference")
	fmt.Println()
	fmt.Println("This document lists all available rules in TerraTidy.")
	fmt.Println()

	currentEngine := ""
	for _, rule := range rules {
		if rule.Engine != currentEngine {
			currentEngine = rule.Engine
			fmt.Printf("## %s Engine\n\n", cases.Title(language.English).String(currentEngine))
		}

		fmt.Printf("### %s\n\n", rule.Name)
		fmt.Printf("**Severity:** %s\n\n", rule.Severity)
		fmt.Printf("%s\n\n", rule.Description)
	}

	return nil
}

// getAllRules returns all available rules from all engines.
func getAllRules() []RuleInfo {
	var rules []RuleInfo

	// Get style rules
	styleEngine := style.New(nil)
	for _, rule := range styleEngine.GetAllRules() {
		rules = append(rules, RuleInfo{
			Name:        rule.Name(),
			Description: getStyleRuleDescription(rule.Name()),
			Engine:      "style",
			Severity:    "warning",
			Enabled:     true,
		})
	}

	// Get lint rules (built-in)
	lintEngine := lint.New(nil)
	for _, rule := range lintEngine.GetAllRules() {
		rules = append(rules, RuleInfo{
			Name:        rule.Name(),
			Description: rule.Description(),
			Engine:      "lint",
			Severity:    "warning",
			Enabled:     true,
		})
	}

	// Add TFLint integration note
	rules = append(rules, RuleInfo{
		Name:        "lint.tflint",
		Description: "TFLint integration - runs all enabled TFLint rules (configure via .tflint.hcl)",
		Engine:      "lint",
		Severity:    "variable",
		Enabled:     true,
	})

	// Add policy rules (built-in)
	policyRules := []RuleInfo{
		{
			Name:        "policy.required-terraform-block",
			Description: "Require terraform block with required_version",
			Engine:      "policy", Severity: "warning", Enabled: true,
		},
		{
			Name:        "policy.required-version",
			Description: "Require required_version in terraform block",
			Engine:      "policy", Severity: "warning", Enabled: true,
		},
		{
			Name:        "policy.required-providers",
			Description: "Require required_providers block when using providers",
			Engine:      "policy", Severity: "warning", Enabled: true,
		},
		{
			Name:        "policy.no-public-ssh",
			Description: "Disallow security groups with public SSH access",
			Engine:      "policy", Severity: "error", Enabled: true,
		},
		{
			Name:        "policy.no-public-s3",
			Description: "Disallow S3 buckets with public-read ACL",
			Engine:      "policy", Severity: "error", Enabled: true,
		},
		{
			Name:        "policy.no-public-rds",
			Description: "Disallow publicly accessible RDS instances",
			Engine:      "policy", Severity: "error", Enabled: true,
		},
		{
			Name:        "policy.required-tags",
			Description: "Require tags on taggable resources",
			Engine:      "policy", Severity: "warning", Enabled: true,
		},
		{
			Name:        "policy.module-version",
			Description: "Require version constraint on external modules",
			Engine:      "policy", Severity: "warning", Enabled: true,
		},
	}
	rules = append(rules, policyRules...)

	return rules
}

// getStyleRuleDescription returns a description for a style rule.
func getStyleRuleDescription(name string) string {
	descriptions := map[string]string{
		"style.blank-lines-between-blocks": "Ensure blank lines between resource blocks",
		"style.block-label-case":           "Enforce snake_case naming for block labels",
		"style.for-each-count-first":       "Place for_each/count as first attribute in blocks",
		"style.lifecycle-at-end":           "Place lifecycle block at end of resource",
		"style.tags-at-end":                "Place tags attribute at end of resource",
		"style.depends-on-order":           "Order depends_on references alphabetically",
		"style.source-version-grouped":     "Group source and version in module blocks",
		"style.variable-order":             "Order variable blocks consistently",
		"style.output-order":               "Order output blocks consistently",
		"style.terraform-block-first":      "Place terraform block first in file",
		"style.provider-block-order":       "Order provider blocks after terraform block",
		"style.no-empty-blocks":            "Disallow empty blocks",
	}

	if desc, ok := descriptions[name]; ok {
		return desc
	}
	return "No description available"
}
