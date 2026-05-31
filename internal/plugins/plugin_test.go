package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/TerraTidy/internal/engines/style/rules"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	dirs := []string{"/path/to/plugins", "~/.terratidy/plugins"}
	manager := NewManager(dirs, false)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.plugins)
	assert.NotNil(t, manager.rules)
	assert.NotNil(t, manager.engines)
	assert.NotNil(t, manager.formatters)
	assert.Equal(t, dirs, manager.directories)
	assert.False(t, manager.verifyIntegrity)
}

func TestManager_LoadAll_NonExistentDir(t *testing.T) {
	manager := NewManager([]string{"/nonexistent/path"}, false)

	// Should not error on non-existent directories
	err := manager.LoadAll()
	assert.NoError(t, err)
}

func TestManager_LoadAll_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager([]string{tmpDir}, false)

	err := manager.LoadAll()
	assert.NoError(t, err)
}

func TestManager_loadFromDirectory_NotADir(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	err := os.WriteFile(tmpFile, []byte("content"), 0o644)
	require.NoError(t, err)

	manager := NewManager(nil, false)
	err = manager.loadFromDirectory(tmpFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestManager_loadFromDirectory_ExpandsHome(t *testing.T) {
	// Create a manager with home directory path
	manager := NewManager(nil, false)

	// This should not panic even with ~ prefix
	// It will return nil for non-existent directory
	err := manager.loadFromDirectory("~/.terratidy-nonexistent")
	assert.NoError(t, err) // Returns nil for non-existent
}

// fakeRule implements sdk.Rule for testing.
// This is a test fake (not a mock framework) - a minimal implementation
// for verifying the plugin manager correctly handles Rule registrations.
type fakeRule struct {
	name        string
	description string
}

func (r *fakeRule) Name() string        { return r.name }
func (r *fakeRule) Description() string { return r.description }
func (r *fakeRule) Check(_ *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return nil, nil
}

func (r *fakeRule) Fix(_ *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	return nil, nil
}

func TestManager_RegisterRule(t *testing.T) {
	manager := NewManager(nil, false)

	rule := &fakeRule{name: "test-rule", description: "Test rule"}
	manager.RegisterRule(rule)

	rules := manager.GetRules()
	assert.Len(t, rules, 1)
	assert.Equal(t, rule, rules["test-rule"])
}

func TestManager_GetRule(t *testing.T) {
	manager := NewManager(nil, false)

	rule := &fakeRule{name: "test-rule", description: "Test rule"}
	manager.RegisterRule(rule)

	t.Run("existing rule", func(t *testing.T) {
		found, ok := manager.GetRule("test-rule")
		assert.True(t, ok)
		assert.Equal(t, rule, found)
	})

	t.Run("non-existing rule", func(t *testing.T) {
		_, ok := manager.GetRule("nonexistent")
		assert.False(t, ok)
	})
}

// fakeEngine implements sdk.Engine for testing.
// This is a test fake (not a mock framework) - a minimal implementation
// for verifying the plugin manager correctly handles Engine registrations.
type fakeEngine struct {
	name string
}

func (e *fakeEngine) Name() string { return e.name }
func (e *fakeEngine) Run(_ context.Context, _ []string) ([]sdk.Finding, error) {
	return nil, nil
}

func TestManager_RegisterEngine(t *testing.T) {
	manager := NewManager(nil, false)

	engine := &fakeEngine{name: "test-engine"}
	manager.RegisterEngine(engine)

	engines := manager.GetEngines()
	assert.Len(t, engines, 1)
	assert.Equal(t, engine, engines["test-engine"])
}

func TestManager_GetEngine(t *testing.T) {
	manager := NewManager(nil, false)

	engine := &fakeEngine{name: "test-engine"}
	manager.RegisterEngine(engine)

	t.Run("existing engine", func(t *testing.T) {
		found, ok := manager.GetEngine("test-engine")
		assert.True(t, ok)
		assert.Equal(t, engine, found)
	})

	t.Run("non-existing engine", func(t *testing.T) {
		_, ok := manager.GetEngine("nonexistent")
		assert.False(t, ok)
	})
}

// fakeFormatter implements FormatterPlugin for testing.
// This is a test fake (not a mock framework) - a minimal implementation
// for verifying the plugin manager correctly handles Formatter registrations.
type fakeFormatter struct {
	name string
}

func (f *fakeFormatter) Name() string { return f.name }
func (f *fakeFormatter) Format(_ []sdk.Finding, _ io.Writer) error {
	return nil
}

func TestManager_RegisterFormatter(t *testing.T) {
	manager := NewManager(nil, false)

	formatter := &fakeFormatter{name: "test-formatter"}
	manager.RegisterFormatter(formatter)

	formatters := manager.GetFormatters()
	assert.Len(t, formatters, 1)
	assert.Equal(t, formatter, formatters["test-formatter"])
}

func TestManager_GetFormatter(t *testing.T) {
	manager := NewManager(nil, false)

	formatter := &fakeFormatter{name: "test-formatter"}
	manager.RegisterFormatter(formatter)

	t.Run("existing formatter", func(t *testing.T) {
		found, ok := manager.GetFormatter("test-formatter")
		assert.True(t, ok)
		assert.Equal(t, formatter, found)
	})

	t.Run("non-existing formatter", func(t *testing.T) {
		_, ok := manager.GetFormatter("nonexistent")
		assert.False(t, ok)
	})
}

func TestManager_ListPlugins(t *testing.T) {
	manager := NewManager(nil, false)

	// Initially empty
	plugins := manager.ListPlugins()
	assert.Empty(t, plugins)

	// Add a plugin manually
	manager.mu.Lock()
	manager.plugins["test"] = &Plugin{
		Metadata: PluginMetadata{Name: "test"},
	}
	manager.mu.Unlock()

	plugins = manager.ListPlugins()
	assert.Len(t, plugins, 1)
	assert.Equal(t, "test", plugins[0].Metadata.Name)
}

func TestManager_GetRules_ReturnsCopy(t *testing.T) {
	manager := NewManager(nil, false)

	rule := &fakeRule{name: "test-rule"}
	manager.RegisterRule(rule)

	// Get rules and modify the returned map
	rules := manager.GetRules()
	delete(rules, "test-rule")

	// Original should still have the rule
	originalRules := manager.GetRules()
	assert.Len(t, originalRules, 1)
}

func TestManager_GetEngines_ReturnsCopy(t *testing.T) {
	manager := NewManager(nil, false)

	engine := &fakeEngine{name: "test-engine"}
	manager.RegisterEngine(engine)

	// Get engines and modify the returned map
	engines := manager.GetEngines()
	delete(engines, "test-engine")

	// Original should still have the engine
	originalEngines := manager.GetEngines()
	assert.Len(t, originalEngines, 1)
}

func TestManager_GetFormatters_ReturnsCopy(t *testing.T) {
	manager := NewManager(nil, false)

	formatter := &fakeFormatter{name: "test-formatter"}
	manager.RegisterFormatter(formatter)

	// Get formatters and modify the returned map
	formatters := manager.GetFormatters()
	delete(formatters, "test-formatter")

	// Original should still have the formatter
	originalFormatters := manager.GetFormatters()
	assert.Len(t, originalFormatters, 1)
}

func TestManager_ConcurrentAccess(t *testing.T) {
	// This test verifies the Manager is thread-safe under concurrent access.
	// It should complete without panics or data races (run with -race).
	require.NotPanics(t, func() {
		manager := NewManager(nil, false)
		done := make(chan bool)

		go func() {
			for i := range 100 {
				manager.RegisterRule(&fakeRule{name: "rule-" + string(rune('a'+i%26))})
			}
			done <- true
		}()

		go func() {
			for range 100 {
				_ = manager.GetRules()
			}
			done <- true
		}()

		go func() {
			for i := range 100 {
				manager.RegisterEngine(&fakeEngine{name: "engine-" + string(rune('a'+i%26))})
			}
			done <- true
		}()

		go func() {
			for range 100 {
				_ = manager.GetEngines()
			}
			done <- true
		}()

		// Wait for all goroutines
		for range 4 {
			<-done
		}
	})
}

// testRulePlugin implements RulePlugin for interface verification
type testRulePlugin struct{}

func (p *testRulePlugin) GetRules() []sdk.Rule {
	return []sdk.Rule{&fakeRule{name: "test"}}
}

func TestRulePluginInterface(t *testing.T) {
	// Verify the RulePlugin interface
	var _ RulePlugin = &testRulePlugin{}

	plugin := &testRulePlugin{}
	rules := plugin.GetRules()
	assert.Len(t, rules, 1)
}

func TestEnginePluginInterface(_ *testing.T) {
	// Verify the EnginePlugin interface
	var _ EnginePlugin = &fakeEngine{}
}

func TestFormatterPluginInterface(_ *testing.T) {
	// Verify the FormatterPlugin interface
	var _ FormatterPlugin = &fakeFormatter{}
}

func TestManager_LoadAll_WithYAMLFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid YAML rule file
	yamlFile := filepath.Join(tmpDir, "check.yaml")
	content := "name: check-rule\ndescription: A check rule\nseverity: warning\nenabled: true\n"
	err := os.WriteFile(yamlFile, []byte(content), 0o644)
	require.NoError(t, err)

	manager := NewManager([]string{tmpDir}, false)
	err = manager.LoadAll()
	assert.NoError(t, err)

	// YAML rule should be loaded
	plugins := manager.ListPlugins()
	assert.Len(t, plugins, 1)
}

func TestManager_LoadAll_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0o755)
	require.NoError(t, err)

	// Create a file in subdirectory
	testFile := filepath.Join(subDir, "test.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o644)
	require.NoError(t, err)

	manager := NewManager([]string{tmpDir}, false)
	err = manager.LoadAll()
	assert.NoError(t, err)
}

func TestManager_loadGoPlugin_NonExistentFile(t *testing.T) {
	manager := NewManager(nil, false)

	err := manager.loadGoPlugin("/nonexistent/plugin.so", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "opening plugin")
}

func TestManager_loadGoPlugin_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.so")

	// Create a file that's not a valid Go plugin
	err := os.WriteFile(invalidFile, []byte("not a plugin"), 0o644)
	require.NoError(t, err)

	manager := NewManager(nil, false)
	err = manager.loadGoPlugin(invalidFile, nil)
	assert.Error(t, err)
	// Will fail on plugin.Open
}

// Note: Testing actual plugin loading (loadRulePlugin, loadEnginePlugin, loadFormatterPlugin)
// requires building real .so files with proper symbols, which is better suited for
// integration tests. The functions are structured to return clear errors for missing
// symbols and incorrect types, which are tested via the error paths above.
//
// Paths that cannot be covered in unit tests:
//   - loadRulePlugin/loadEnginePlugin/loadFormatterPlugin wrong-signature branches:
//     require a real compiled .so with a New symbol of the wrong type.
//   - loadFromDirectory os.UserHomeDir() failure (line ~223): UserHomeDir reads $HOME
//     on Unix; unsetting it may still succeed via /etc/passwd on some systems.
//   - loadFromDirectory os.Stat() non-ENOENT error (line ~234): requires a path that
//     Stat fails on for reasons other than not-exist, which is not reliably triggerable
//     without mocks.
//   - os.ReadDir() error (line ~259) is covered by TestManager_LoadFromDirectory_PermissionIssues
//     on non-Windows via the chmod 0o000 technique.

func TestLoadManifest(t *testing.T) {
	t.Run("valid manifest", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, ManifestFileName)

		// Real SHA256 hashes (64 hex chars)
		hash1 := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // empty file
		hash2 := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9" // "hello world"
		content := fmt.Sprintf("# Comment line\n%s  plugin1.so\n%s  plugin2.so\n", hash1, hash2)
		err := os.WriteFile(manifestPath, []byte(content), 0o644)
		require.NoError(t, err)

		checksums, err := loadManifest(manifestPath)
		require.NoError(t, err)
		assert.Len(t, checksums, 2)
		assert.Equal(t, hash1, checksums["plugin1.so"])
		assert.Equal(t, hash2, checksums["plugin2.so"])
	})

	t.Run("empty manifest", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, ManifestFileName)

		err := os.WriteFile(manifestPath, []byte("# Only comments\n\n"), 0o644)
		require.NoError(t, err)

		checksums, err := loadManifest(manifestPath)
		require.NoError(t, err)
		assert.Empty(t, checksums)
	})

	t.Run("invalid hash length", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, ManifestFileName)

		content := `tooshort  plugin.so`
		err := os.WriteFile(manifestPath, []byte(content), 0o644)
		require.NoError(t, err)

		_, err = loadManifest(manifestPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected 64 hex chars")
	})

	t.Run("invalid hex", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, ManifestFileName)

		content := `gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg  plugin.so`
		err := os.WriteFile(manifestPath, []byte(content), 0o644)
		require.NoError(t, err)

		_, err = loadManifest(manifestPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid hex")
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := loadManifest("/nonexistent/manifest")
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestComputeFileHash(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		content := []byte("hello world")
		err := os.WriteFile(testFile, content, 0o644)
		require.NoError(t, err)

		hash, err := computeFileHash(testFile)
		require.NoError(t, err)

		// SHA256 of "hello world"
		expectedHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
		assert.Equal(t, expectedHash, hash)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := computeFileHash("/nonexistent/file.txt")
		require.Error(t, err)
	})
}

func TestManager_VerifyPluginChecksum(t *testing.T) {
	t.Run("valid checksum", func(t *testing.T) {
		tmpDir := t.TempDir()
		pluginPath := filepath.Join(tmpDir, "plugin.so")

		content := []byte("fake plugin content")
		err := os.WriteFile(pluginPath, content, 0o644)
		require.NoError(t, err)

		// Compute actual hash
		actualHash, err := computeFileHash(pluginPath)
		require.NoError(t, err)

		checksums := map[string]string{
			"plugin.so": actualHash,
		}

		manager := NewManager(nil, true)
		err = manager.verifyPluginChecksum(pluginPath, checksums)
		assert.NoError(t, err)
	})

	t.Run("checksum mismatch", func(t *testing.T) {
		tmpDir := t.TempDir()
		pluginPath := filepath.Join(tmpDir, "plugin.so")

		content := []byte("fake plugin content")
		err := os.WriteFile(pluginPath, content, 0o644)
		require.NoError(t, err)

		checksums := map[string]string{
			"plugin.so": "0000000000000000000000000000000000000000000000000000000000000000",
		}

		manager := NewManager(nil, true)
		err = manager.verifyPluginChecksum(pluginPath, checksums)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")
	})

	t.Run("plugin not in manifest warns but continues", func(t *testing.T) {
		tmpDir := t.TempDir()
		pluginPath := filepath.Join(tmpDir, "plugin.so")

		content := []byte("fake plugin content")
		err := os.WriteFile(pluginPath, content, 0o644)
		require.NoError(t, err)

		checksums := map[string]string{
			"other.so": "0000000000000000000000000000000000000000000000000000000000000000",
		}

		manager := NewManager(nil, true)
		err = manager.verifyPluginChecksum(pluginPath, checksums)
		// Should not error - just warn
		assert.NoError(t, err)
	})

	t.Run("file read error", func(t *testing.T) {
		// Test with a nonexistent file that's in the checksums
		checksums := map[string]string{
			"missing.so": "0000000000000000000000000000000000000000000000000000000000000000",
		}

		manager := NewManager(nil, true)
		err := manager.verifyPluginChecksum("/nonexistent/missing.so", checksums)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "computing hash")
	})
}

func TestManager_LoadAll_WithVerification(t *testing.T) {
	t.Run("verification disabled skips manifest", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a YAML rule (not .so, so we can actually test loading)
		yamlContent := `name: test-rule
description: Test rule
severity: warning
enabled: true
pattern:
  type: resource
  missing_attribute: tags
message: "Missing tags"
`
		err := os.WriteFile(filepath.Join(tmpDir, "test.yaml"), []byte(yamlContent), 0o644)
		require.NoError(t, err)

		// No manifest file - should work fine with verification disabled
		manager := NewManager([]string{tmpDir}, false)
		err = manager.LoadAll()
		assert.NoError(t, err)
		assert.Len(t, manager.rules, 1)
	})
}

func TestManager_MultipleDirectories(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	manager := NewManager([]string{tmpDir1, tmpDir2}, false)
	err := manager.LoadAll()
	assert.NoError(t, err)

	// Should handle multiple directories without error
	assert.Equal(t, []string{tmpDir1, tmpDir2}, manager.directories)
}

func TestManager_LoadFromDirectory_WithDotFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hidden file (dotfile)
	dotFile := filepath.Join(tmpDir, ".hidden")
	err := os.WriteFile(dotFile, []byte("hidden"), 0o644)
	require.NoError(t, err)

	manager := NewManager([]string{tmpDir}, false)
	err = manager.LoadAll()
	assert.NoError(t, err)

	// Hidden files should be processed (they're just regular files)
	// but won't be valid plugins, so no plugins loaded
	plugins := manager.ListPlugins()
	assert.Empty(t, plugins)
}

func TestPluginMetadata_AllFields(t *testing.T) {
	meta := PluginMetadata{
		Name:        "comprehensive-plugin",
		Version:     "2.1.0",
		Description: "A comprehensive test plugin with all fields",
		Author:      "Test Suite",
		Type:        PluginTypeEngine,
		Path:        "/full/path/to/plugin.so",
	}

	// Verify all fields are set correctly
	assert.Equal(t, "comprehensive-plugin", meta.Name)
	assert.Equal(t, "2.1.0", meta.Version)
	assert.Equal(t, "A comprehensive test plugin with all fields", meta.Description)
	assert.Equal(t, "Test Suite", meta.Author)
	assert.Equal(t, PluginTypeEngine, meta.Type)
	assert.Equal(t, "/full/path/to/plugin.so", meta.Path)
}

func TestManager_RegisterMultipleRulesWithSameName(t *testing.T) {
	manager := NewManager(nil, false)

	rule1 := &fakeRule{name: "duplicate", description: "First rule"}
	rule2 := &fakeRule{name: "duplicate", description: "Second rule"}

	manager.RegisterRule(rule1)
	manager.RegisterRule(rule2)

	// Last registered should win
	rules := manager.GetRules()
	assert.Len(t, rules, 1)
	assert.Equal(t, "Second rule", rules["duplicate"].Description())
}

func TestManager_RegisterMultipleEnginesWithSameName(t *testing.T) {
	manager := NewManager(nil, false)

	engine1 := &fakeEngine{name: "duplicate"}
	engine2 := &fakeEngine{name: "duplicate"}

	manager.RegisterEngine(engine1)
	manager.RegisterEngine(engine2)

	// Last registered should win
	engines := manager.GetEngines()
	assert.Len(t, engines, 1)
	assert.Equal(t, engine2, engines["duplicate"])
}

func TestManager_RegisterMultipleFormattersWithSameName(t *testing.T) {
	manager := NewManager(nil, false)

	formatter1 := &fakeFormatter{name: "duplicate"}
	formatter2 := &fakeFormatter{name: "duplicate"}

	manager.RegisterFormatter(formatter1)
	manager.RegisterFormatter(formatter2)

	// Last registered should win
	formatters := manager.GetFormatters()
	assert.Len(t, formatters, 1)
	assert.Equal(t, formatter2, formatters["duplicate"])
}

// fakeFixableRule is a rule that returns a precomputed fix result.
type fakeFixableRule struct {
	name      string
	findings  []sdk.Finding
	fixResult *sdk.FixResult
}

func (r *fakeFixableRule) Name() string        { return r.name }
func (r *fakeFixableRule) Description() string { return "Fixable rule for testing" }
func (r *fakeFixableRule) Check(_ *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return r.findings, nil
}

func (r *fakeFixableRule) Fix(_ *sdk.Context, _ *hcl.File) (*sdk.FixResult, error) {
	return r.fixResult, nil
}

// TestGoPluginRuleFixApplied verifies that when a rule implements Fix()
// and returns a non-nil FixResult, those edits are available for applying.
func TestGoPluginRuleFixApplied(t *testing.T) {
	t.Run("rule with fix implementation returns fix result", func(t *testing.T) {
		originalContent := []byte(`resource "aws_instance" "test" {
  ami = "ami-123"
}
`)
		fixedContent := []byte(`resource "aws_instance" "test" {
  ami           = "ami-123"
  instance_type = "t2.micro"
  tags = {
    Name = "test"
  }
}
`)
		rule := &fakeFixableRule{
			name: "fixable-rule",
			findings: []sdk.Finding{
				{Rule: "fixable-rule", Message: "Missing tags"},
			},
			fixResult: rules.WholeFileEdit(originalContent, fixedContent),
		}

		// Verify Fix() returns a single whole-file edit covering the original range.
		ctx := &sdk.Context{File: "test.tf"}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)
		assert.Equal(t, 0, result.Edits[0].Start)
		assert.Equal(t, len(originalContent), result.Edits[0].End)
		assert.Equal(t, fixedContent, result.Edits[0].Replacement)
	})

	t.Run("rule without fix returns nil", func(t *testing.T) {
		rule := &fakeRule{name: "non-fixable-rule"}

		ctx := &sdk.Context{File: "test.tf"}
		result, err := rule.Fix(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, result, "rule without fix implementation returns nil")
	})

	t.Run("fixable rule registered with manager", func(t *testing.T) {
		manager := NewManager(nil, false)
		originalContent := []byte("original content")
		fixedContent := []byte("fixed content")
		rule := &fakeFixableRule{
			name:      "managed-fixable-rule",
			fixResult: rules.WholeFileEdit(originalContent, fixedContent),
		}

		manager.RegisterRule(rule)

		// Retrieve and verify - need to type assert to Fixer interface
		retrieved, ok := manager.GetRule("managed-fixable-rule")
		require.True(t, ok)

		fixer, isFixer := retrieved.(sdk.Fixer)
		require.True(t, isFixer, "rule should implement sdk.Fixer")

		result, err := fixer.Fix(&sdk.Context{}, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Edits, 1)
		assert.Equal(t, 0, result.Edits[0].Start)
		assert.Equal(t, len(originalContent), result.Edits[0].End)
		assert.Equal(t, fixedContent, result.Edits[0].Replacement)
	})
}

func TestRelativePath(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	t.Run("absolute path in cwd becomes relative", func(t *testing.T) {
		absPath := filepath.Join(cwd, "subdir", "file.txt")
		rel := relativePath(absPath)
		assert.Equal(t, filepath.Join("subdir", "file.txt"), rel)
	})

	t.Run("relative path stays relative", func(t *testing.T) {
		relPath := "subdir/file.txt"
		result := relativePath(relPath)
		assert.Equal(t, relPath, result)
	})

	t.Run("absolute path outside cwd", func(t *testing.T) {
		// Path outside cwd should still return a relative path (with ..)
		// Use filepath.Abs to get a platform-appropriate absolute path
		absPath, err := filepath.Abs(filepath.Join(string(filepath.Separator), "tmp", "outside", "file.txt"))
		require.NoError(t, err)
		result := relativePath(absPath)
		// Result should be relative (may have .. components)
		assert.NotEqual(t, absPath, result)
	})
}
