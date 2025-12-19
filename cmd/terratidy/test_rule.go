// Package main provides the test-rule command for TerraTidy.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santosr2/terratidy/internal/engines/policy"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	testRuleFixtures string
	testRuleExpect   string
	testRuleVerbose  bool
)

var testRuleCmd = &cobra.Command{
	Use:   "test-rule [rule-path]",
	Short: "Test a specific rule",
	Long: `Test a rule against fixture files and expected findings.

This command allows you to test custom rules during development.
It runs the rule against test fixtures and compares the findings
with expected results.`,
	Example: `  # Test a Rego policy
  terratidy test-rule ./policies/my-rule.rego

  # Test with specific fixtures directory
  terratidy test-rule ./policies/my-rule.rego --fixtures ./test_fixtures

  # Test with expected findings file
  terratidy test-rule ./policies/my-rule.rego --expect ./expected.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runTestRule,
}

func init() {
	testRuleCmd.Flags().StringVar(&testRuleFixtures, "fixtures", "test_fixtures/", "fixtures directory")
	testRuleCmd.Flags().StringVar(&testRuleExpect, "expect", "", "expected findings file (YAML or JSON)")
	testRuleCmd.Flags().BoolVarP(&testRuleVerbose, "verbose", "v", false, "verbose output")
	rootCmd.AddCommand(testRuleCmd)
}

// ExpectedFinding represents an expected finding in test fixtures.
type ExpectedFinding struct {
	Rule     string `yaml:"rule" json:"rule"`
	Severity string `yaml:"severity" json:"severity"`
	Message  string `yaml:"message" json:"message"`
	File     string `yaml:"file" json:"file"`
}

// ExpectedResults represents expected test results.
type ExpectedResults struct {
	Findings []ExpectedFinding `yaml:"findings" json:"findings"`
}

func runTestRule(_ *cobra.Command, args []string) error {
	rulePath := args[0]

	// Check if rule file exists
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		return fmt.Errorf("rule file not found: %s", rulePath)
	}

	fmt.Printf("Testing rule: %s\n\n", rulePath)

	// Determine rule type based on extension
	ext := strings.ToLower(filepath.Ext(rulePath))
	switch ext {
	case ".rego":
		return testRegoRule(rulePath)
	default:
		return fmt.Errorf("unsupported rule type: %s", ext)
	}
}

func testRegoRule(rulePath string) error {
	// Find fixture files
	fixtures, err := findFixtures(testRuleFixtures)
	if err != nil {
		return fmt.Errorf("finding fixtures: %w", err)
	}

	if len(fixtures) == 0 {
		fmt.Printf("No fixtures found in %s\n", testRuleFixtures)
		fmt.Println()
		fmt.Println("Create test fixtures:")
		fmt.Printf("  mkdir -p %s\n", testRuleFixtures)
		fmt.Printf("  # Add .tf files to test against\n")
		return nil
	}

	fmt.Printf("Found %d fixture file(s)\n\n", len(fixtures))

	// Create policy engine with the rule
	engine := policy.New(&policy.Config{
		PolicyFiles: []string{rulePath},
	})

	// Run the rule against fixtures
	ctx := context.Background()
	findings, err := engine.Run(ctx, fixtures)
	if err != nil {
		return fmt.Errorf("running rule: %w", err)
	}

	// Display findings
	fmt.Printf("Results: %d finding(s)\n\n", len(findings))

	for _, finding := range findings {
		icon := "i"
		switch finding.Severity {
		case sdk.SeverityError:
			icon = "!"
		case sdk.SeverityWarning:
			icon = "!"
		}

		fmt.Printf("  [%s] %s\n", icon, finding.Rule)
		fmt.Printf("      %s\n", finding.Message)
		if finding.File != "" {
			fmt.Printf("      File: %s\n", finding.File)
		}
		fmt.Println()
	}

	// Compare with expected findings if provided
	if testRuleExpect != "" {
		return compareExpected(findings, testRuleExpect)
	}

	return nil
}

func findFixtures(dir string) ([]string, error) {
	var fixtures []string

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fixtures, nil
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isHCLFile(path) {
			fixtures = append(fixtures, path)
		}
		return nil
	})

	return fixtures, err
}

func compareExpected(findings []sdk.Finding, expectFile string) error {
	// Load expected findings
	data, err := os.ReadFile(expectFile)
	if err != nil {
		return fmt.Errorf("reading expected file: %w", err)
	}

	var expected ExpectedResults
	ext := strings.ToLower(filepath.Ext(expectFile))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &expected); err != nil {
			return fmt.Errorf("parsing JSON: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &expected); err != nil {
			return fmt.Errorf("parsing YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported expected file format: %s", ext)
	}

	// Compare findings
	fmt.Println("---")
	fmt.Println("Comparing with expected findings...")
	fmt.Println()

	passed := true
	matched := make(map[int]bool)

	// Check each expected finding
	for _, exp := range expected.Findings {
		found := false
		for i, actual := range findings {
			if matched[i] {
				continue
			}
			if matchesFinding(exp, actual) {
				matched[i] = true
				found = true
				break
			}
		}

		if found {
			fmt.Printf("  [+] Expected finding matched: %s\n", exp.Rule)
		} else {
			fmt.Printf("  [-] Expected finding NOT found: %s\n", exp.Rule)
			if exp.Message != "" {
				fmt.Printf("      Message: %s\n", exp.Message)
			}
			passed = false
		}
	}

	// Check for unexpected findings
	for i, actual := range findings {
		if !matched[i] {
			fmt.Printf("  [?] Unexpected finding: %s\n", actual.Rule)
			fmt.Printf("      Message: %s\n", actual.Message)
			passed = false
		}
	}

	fmt.Println()

	if passed {
		fmt.Println("All tests passed!")
		return nil
	}

	return fmt.Errorf("test failed: expected findings do not match actual findings")
}

func matchesFinding(expected ExpectedFinding, actual sdk.Finding) bool {
	// Rule must match
	if expected.Rule != "" && !strings.HasSuffix(actual.Rule, expected.Rule) {
		return false
	}

	// Severity must match if specified
	if expected.Severity != "" && string(actual.Severity) != expected.Severity {
		return false
	}

	// Message must contain if specified
	if expected.Message != "" && !strings.Contains(actual.Message, expected.Message) {
		return false
	}

	return true
}
