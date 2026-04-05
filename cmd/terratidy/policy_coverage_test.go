package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/internal/config"
)

func TestRunPolicyCheckWithConfig(t *testing.T) {
	dir := t.TempDir()
	content := `resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
}
`
	tmpFile := filepath.Join(dir, "test.tf")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	ctx := context.Background()
	cfg := config.DefaultConfig()
	findings, err := runPolicyCheckWithConfig(ctx, cfg, []string{tmpFile}, 4, true)
	require.NoError(t, err)
	_ = findings
}

func TestBuildPolicyConfig_WithEngineConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Engines.Policy.PolicyDirs = []string{"./policies"}
	cfg.Engines.Policy.PolicyFiles = []string{"custom.rego"}

	policyCfg := buildPolicyConfig(cfg)
	assert.Equal(t, []string{"./policies"}, policyCfg.PolicyDirs)
	assert.Equal(t, []string{"custom.rego"}, policyCfg.PolicyFiles)
}
