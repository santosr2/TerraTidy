package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBashRule_Name(t *testing.T) {
	rule := NewBashRule("/path/to/my-rule.sh")
	assert.Equal(t, "my-rule", rule.Name())
}

func TestBashRule_Description(t *testing.T) {
	rule := NewBashRule("/path/to/check-tags.sh")
	assert.Equal(t, "Bash rule: check-tags", rule.Description())
}

func TestBashRule_Check_EmptyFile(t *testing.T) {
	rule := NewBashRule("/path/to/rule.sh")
	ctx := &sdk.Context{File: ""}
	findings, err := rule.Check(ctx, &hcl.File{})
	require.NoError(t, err)
	assert.Nil(t, findings)
}

func TestBashRule_Check_ScriptWithFindings(t *testing.T) {
	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "check.sh")
	content := `#!/usr/bin/env bash
set -euo pipefail
FILE="$1"
echo '{"findings": [{"file": "'"$FILE"'", "line": 1, "column": 1, "message": "test finding", "severity": "warning"}]}'
`
	require.NoError(t, os.WriteFile(script, []byte(content), 0o755))

	rule := NewBashRule(script)
	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, &hcl.File{})
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	assert.Equal(t, "test finding", findings[0].Message)
	assert.Equal(t, sdk.SeverityWarning, findings[0].Severity)
}

func TestBashRule_Check_ScriptNoFindings(t *testing.T) {
	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "clean.sh")
	content := `#!/usr/bin/env bash
# No output means no findings
`
	require.NoError(t, os.WriteFile(script, []byte(content), 0o755))

	rule := NewBashRule(script)
	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, &hcl.File{})
	require.NoError(t, err)
	assert.Nil(t, findings)
}

func TestBashRule_Check_ScriptError(t *testing.T) {
	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "fail.sh")
	content := `#!/usr/bin/env bash
echo "something went wrong" >&2
exit 2
`
	require.NoError(t, os.WriteFile(script, []byte(content), 0o755))

	rule := NewBashRule(script)
	ctx := &sdk.Context{File: "test.tf"}
	_, err := rule.Check(ctx, &hcl.File{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executing bash rule")
}

func TestBashRule_Check_UsesRuleNameWhenNotInOutput(t *testing.T) {
	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "my-check.sh")
	content := `#!/usr/bin/env bash
echo '{"findings": [{"file": "test.tf", "line": 5, "message": "issue found", "severity": "error"}]}'
`
	require.NoError(t, os.WriteFile(script, []byte(content), 0o755))

	rule := NewBashRule(script)
	ctx := &sdk.Context{File: "test.tf"}
	findings, err := rule.Check(ctx, &hcl.File{})
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	assert.Equal(t, "my-check", findings[0].Rule)
}

func TestBashRule_Fix_ReturnsNil(t *testing.T) {
	rule := NewBashRule("/path/to/rule.sh")
	result, err := rule.Fix(&sdk.Context{}, &hcl.File{})
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestLoadBashRule(t *testing.T) {
	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "test-rule.sh")
	content := `#!/usr/bin/env bash
echo '{"findings": []}'
`
	require.NoError(t, os.WriteFile(script, []byte(content), 0o755))

	rule, err := loadBashRule(script)
	require.NoError(t, err)
	assert.Equal(t, "test-rule", rule.Name())
}

func TestLoadBashRule_NotExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "noexec.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/bash"), 0o644))

	_, err := loadBashRule(script)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not executable")
}

func TestLoadBashRule_FileNotFound(t *testing.T) {
	_, err := loadBashRule("/nonexistent/rule.sh")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading bash rule")
}

func TestManager_LoadAll_WithBashRule(t *testing.T) {
	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "bash-test.sh")
	content := `#!/usr/bin/env bash
echo '{"findings": []}'
`
	require.NoError(t, os.WriteFile(script, []byte(content), 0o755))

	manager := NewManager([]string{tmpDir})
	err := manager.LoadAll()
	require.NoError(t, err)

	plugins := manager.ListPlugins()
	assert.Len(t, plugins, 1)
	assert.Equal(t, "bash-test", plugins[0].Metadata.Name)

	rule, ok := manager.GetRule("bash-test")
	assert.True(t, ok)
	assert.Equal(t, "bash-test", rule.Name())
}

func TestManager_LoadAll_MixedPluginTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// YAML rule
	yamlContent := `name: yaml-rule
description: YAML rule
severity: warning
enabled: true
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "check.yaml"), []byte(yamlContent), 0o644))

	// Bash rule
	bashContent := `#!/usr/bin/env bash
echo '{"findings": []}'
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "check.sh"), []byte(bashContent), 0o755))

	manager := NewManager([]string{tmpDir})
	err := manager.LoadAll()
	require.NoError(t, err)

	plugins := manager.ListPlugins()
	assert.Len(t, plugins, 2)

	_, yamlOk := manager.GetRule("yaml-rule")
	assert.True(t, yamlOk)

	_, bashOk := manager.GetRule("check")
	assert.True(t, bashOk)
}
