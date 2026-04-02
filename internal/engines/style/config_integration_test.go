package style

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStyleEngineWithConfiguredRule(t *testing.T) {
	// Create temp file with "this" resource name
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "main.tf")
	err := os.WriteFile(tfFile, []byte(`resource "aws_cloudfront_public_key" "this" {
  name = "test"
}`), 0o644)
	require.NoError(t, err)

	// Create style config with the rule enabled
	cfg := &Config{
		Rules: map[string]RuleConfig{
			"style.resource-name-matches-type": {
				Enabled:  true,
				Severity: "info",
			},
		},
	}

	engine := New(cfg)

	findings, err := engine.Run(context.Background(), []string{tfFile})
	require.NoError(t, err)

	t.Logf("Found %d findings", len(findings))
	for _, f := range findings {
		t.Logf("- [%s] %s: %s", f.Severity, f.Rule, f.Message)
	}

	// Should find at least one finding about "this" being generic
	var foundGenericNameFinding bool
	for _, f := range findings {
		if f.Rule == "style.resource-name-matches-type" {
			foundGenericNameFinding = true
			t.Logf("Found expected finding: %s", f.Message)
		}
	}
	require.True(t, foundGenericNameFinding, "Expected to find generic name 'this' finding")
}
