package output

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// JUnitFormatter outputs findings in JUnit XML format for CI/CD integration.
// This format is widely supported by CI systems like Jenkins, GitLab CI, GitHub Actions, etc.
type JUnitFormatter struct {
	Version       string
	AbsolutePaths bool
}

// JUnitTestSuites represents the root element of JUnit XML output
type JUnitTestSuites struct {
	XMLName   xml.Name         `xml:"testsuites"`
	Name      string           `xml:"name,attr"`
	Tests     int              `xml:"tests,attr"`
	Errors    int              `xml:"errors,attr"`
	Failures  int              `xml:"failures,attr"`
	Time      float64          `xml:"time,attr"`
	Timestamp string           `xml:"timestamp,attr"`
	Suites    []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a test suite (grouped by file or rule type)
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Errors    int             `xml:"errors,attr"`
	Failures  int             `xml:"failures,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      float64         `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents an individual test case (one per finding)
type JUnitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Error     *JUnitError   `xml:"error,omitempty"`
}

// JUnitFailure represents a test failure (warnings)
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// JUnitError represents a test error (errors)
type JUnitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// Format implements the Formatter interface for JUnit XML output
func (f *JUnitFormatter) Format(findings []sdk.Finding, w io.Writer) error {
	timestamp := time.Now().Format(time.RFC3339)

	// Group findings by file
	byFile := make(map[string][]sdk.Finding)
	for i := range findings {
		byFile[findings[i].File] = append(byFile[findings[i].File], findings[i])
	}

	// Count totals
	var totalErrors, totalFailures int
	for i := range findings {
		switch findings[i].Severity {
		case sdk.SeverityError:
			totalErrors++
		case sdk.SeverityWarning:
			totalFailures++
		}
	}

	// Build test suites (one per file), sorted for deterministic output
	files := make([]string, 0, len(byFile))
	for file := range byFile {
		files = append(files, file)
	}
	sort.Strings(files)

	var suites []JUnitTestSuite
	for _, file := range files {
		suite := f.buildTestSuite(file, byFile[file], timestamp)
		suites = append(suites, suite)
	}

	// If no findings, create a single passing test suite
	if len(findings) == 0 {
		suites = append(suites, JUnitTestSuite{
			Name:      "terratidy",
			Tests:     1,
			Timestamp: timestamp,
			TestCases: []JUnitTestCase{
				{
					Name:      "all_checks_passed",
					ClassName: "terratidy",
				},
			},
		})
	}

	// Build root element
	testSuites := JUnitTestSuites{
		Name:      "TerraTidy",
		Tests:     len(findings),
		Errors:    totalErrors,
		Failures:  totalFailures,
		Timestamp: timestamp,
		Suites:    suites,
	}

	// Marshal to XML with indentation
	output, err := xml.MarshalIndent(testSuites, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JUnit XML: %w", err)
	}

	// Write XML declaration and content
	_, err = fmt.Fprintf(w, "%s\n%s\n", xml.Header, output)
	return err
}

func (f *JUnitFormatter) buildTestSuite(file string, findings []sdk.Finding, timestamp string) JUnitTestSuite {
	var errors, failures int
	var testCases []JUnitTestCase
	displayFile := displayPath(file, f.AbsolutePaths)

	for i := range findings {
		testCase := JUnitTestCase{
			Name:      findings[i].Rule,
			ClassName: displayFile,
		}

		// Build detailed message with location
		detail := fmt.Sprintf("File: %s\nLine: %d, Column: %d\n\n%s",
			displayFile,
			findings[i].Location.StartLine,
			findings[i].Location.StartColumn,
			findings[i].Message,
		)

		switch findings[i].Severity {
		case sdk.SeverityError:
			errors++
			testCase.Error = &JUnitError{
				Message: findings[i].Message,
				Type:    string(findings[i].Severity),
				Content: detail,
			}
		case sdk.SeverityWarning:
			failures++
			testCase.Failure = &JUnitFailure{
				Message: findings[i].Message,
				Type:    string(findings[i].Severity),
				Content: detail,
			}
		default:
			// Info severity - treat as a passing test with a note
			// No failure or error element means the test passed
		}

		testCases = append(testCases, testCase)
	}

	return JUnitTestSuite{
		Name:      file,
		Tests:     len(findings),
		Errors:    errors,
		Failures:  failures,
		Timestamp: timestamp,
		TestCases: testCases,
	}
}
