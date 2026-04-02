package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJUnitFormatter_Format(t *testing.T) {
	t.Run("empty findings produces passing test", func(t *testing.T) {
		formatter := &JUnitFormatter{Version: "1.0.0"}
		var buf bytes.Buffer

		err := formatter.Format([]sdk.Finding{}, &buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "<?xml")
		assert.Contains(t, output, "<testsuites")
		assert.Contains(t, output, "all_checks_passed")
		assert.Contains(t, output, `tests="1"`)
		assert.Contains(t, output, `errors="0"`)
		assert.Contains(t, output, `failures="0"`)
	})

	t.Run("error finding produces error element", func(t *testing.T) {
		formatter := &JUnitFormatter{Version: "1.0.0"}
		var buf bytes.Buffer

		findings := []sdk.Finding{
			{
				Rule:     "test.rule",
				Message:  "Test error message",
				File:     "test.tf",
				Location: hcl.Range{Start: hcl.Pos{Line: 10, Column: 5}},
				Severity: sdk.SeverityError,
			},
		}

		err := formatter.Format(findings, &buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "<error")
		assert.Contains(t, output, "Test error message")
		assert.Contains(t, output, "test.rule")
		assert.Contains(t, output, `errors="1"`)
	})

	t.Run("warning finding produces failure element", func(t *testing.T) {
		formatter := &JUnitFormatter{Version: "1.0.0"}
		var buf bytes.Buffer

		findings := []sdk.Finding{
			{
				Rule:     "test.rule",
				Message:  "Test warning message",
				File:     "test.tf",
				Location: hcl.Range{Start: hcl.Pos{Line: 10, Column: 5}},
				Severity: sdk.SeverityWarning,
			},
		}

		err := formatter.Format(findings, &buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "<failure")
		assert.Contains(t, output, "Test warning message")
		assert.Contains(t, output, `failures="1"`)
	})

	t.Run("multiple findings grouped by file", func(t *testing.T) {
		formatter := &JUnitFormatter{Version: "1.0.0"}
		var buf bytes.Buffer

		findings := []sdk.Finding{
			{
				Rule:     "rule1",
				Message:  "Message 1",
				File:     "file1.tf",
				Severity: sdk.SeverityError,
			},
			{
				Rule:     "rule2",
				Message:  "Message 2",
				File:     "file1.tf",
				Severity: sdk.SeverityWarning,
			},
			{
				Rule:     "rule3",
				Message:  "Message 3",
				File:     "file2.tf",
				Severity: sdk.SeverityInfo,
			},
		}

		err := formatter.Format(findings, &buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "file1.tf")
		assert.Contains(t, output, "file2.tf")
		assert.Contains(t, output, `tests="3"`)
	})

	t.Run("produces valid XML structure", func(t *testing.T) {
		formatter := &JUnitFormatter{Version: "1.0.0"}
		var buf bytes.Buffer

		findings := []sdk.Finding{
			{
				Rule:     "test.rule",
				Message:  "Test message",
				File:     "test.tf",
				Severity: sdk.SeverityWarning,
			},
		}

		err := formatter.Format(findings, &buf)
		require.NoError(t, err)

		output := buf.String()
		// Check XML structure
		assert.True(t, strings.HasPrefix(output, "<?xml"))
		assert.Contains(t, output, "<testsuites")
		assert.Contains(t, output, "</testsuites>")
		assert.Contains(t, output, "<testsuite")
		assert.Contains(t, output, "</testsuite>")
		assert.Contains(t, output, "<testcase")
		assert.Contains(t, output, "</testcase>")
	})
}
