package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/terratidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginTypeConstants(t *testing.T) {
	assert.Equal(t, PluginType("rule"), PluginTypeRule)
	assert.Equal(t, PluginType("engine"), PluginTypeEngine)
	assert.Equal(t, PluginType("formatter"), PluginTypeFormatter)
}

func TestPluginMetadata(t *testing.T) {
	meta := PluginMetadata{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Author:      "Test Author",
		Type:        PluginTypeRule,
		Path:        "/path/to/plugin.so",
	}

	assert.Equal(t, "test-plugin", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Equal(t, "A test plugin", meta.Description)
	assert.Equal(t, "Test Author", meta.Author)
	assert.Equal(t, PluginTypeRule, meta.Type)
	assert.Equal(t, "/path/to/plugin.so", meta.Path)
}

func TestPlugin(t *testing.T) {
	plugin := Plugin{
		Metadata: PluginMetadata{
			Name:    "test",
			Version: "1.0.0",
			Type:    PluginTypeRule,
		},
		Instance: "mock-instance",
	}

	assert.Equal(t, "test", plugin.Metadata.Name)
	assert.Equal(t, "mock-instance", plugin.Instance)
}

func TestNewManager(t *testing.T) {
	dirs := []string{"/path/to/plugins", "~/.terratidy/plugins"}
	manager := NewManager(dirs)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.plugins)
	assert.NotNil(t, manager.rules)
	assert.NotNil(t, manager.engines)
	assert.NotNil(t, manager.formatters)
	assert.Equal(t, dirs, manager.directories)
}

func TestManager_LoadAll_NonExistentDir(t *testing.T) {
	manager := NewManager([]string{"/nonexistent/path"})

	// Should not error on non-existent directories
	err := manager.LoadAll()
	assert.NoError(t, err)
}

func TestManager_LoadAll_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager([]string{tmpDir})

	err := manager.LoadAll()
	assert.NoError(t, err)
}

func TestManager_loadFromDirectory_NotADir(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	err := os.WriteFile(tmpFile, []byte("content"), 0o644)
	require.NoError(t, err)

	manager := NewManager(nil)
	err = manager.loadFromDirectory(tmpFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestManager_loadFromDirectory_ExpandsHome(t *testing.T) {
	// Create a manager with home directory path
	manager := NewManager(nil)

	// This should not panic even with ~ prefix
	// It will return nil for non-existent directory
	err := manager.loadFromDirectory("~/.terratidy-nonexistent")
	assert.NoError(t, err) // Returns nil for non-existent
}

// MockRule implements sdk.Rule for testing
type MockRule struct {
	name        string
	description string
}

func (r *MockRule) Name() string        { return r.name }
func (r *MockRule) Description() string { return r.description }
func (r *MockRule) Check(_ *sdk.Context, _ *hcl.File) ([]sdk.Finding, error) {
	return nil, nil
}
func (r *MockRule) Fix(_ *sdk.Context, _ *hcl.File) ([]byte, error) {
	return nil, nil
}

func TestManager_RegisterRule(t *testing.T) {
	manager := NewManager(nil)

	rule := &MockRule{name: "test-rule", description: "Test rule"}
	manager.RegisterRule(rule)

	rules := manager.GetRules()
	assert.Len(t, rules, 1)
	assert.Equal(t, rule, rules["test-rule"])
}

func TestManager_GetRule(t *testing.T) {
	manager := NewManager(nil)

	rule := &MockRule{name: "test-rule", description: "Test rule"}
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

// MockEngine implements EnginePlugin for testing
type MockEngine struct {
	name string
}

func (e *MockEngine) Name() string { return e.name }
func (e *MockEngine) Run(_ context.Context, _ []string) ([]sdk.Finding, error) {
	return nil, nil
}

func TestManager_RegisterEngine(t *testing.T) {
	manager := NewManager(nil)

	engine := &MockEngine{name: "test-engine"}
	manager.RegisterEngine(engine)

	engines := manager.GetEngines()
	assert.Len(t, engines, 1)
	assert.Equal(t, engine, engines["test-engine"])
}

func TestManager_GetEngine(t *testing.T) {
	manager := NewManager(nil)

	engine := &MockEngine{name: "test-engine"}
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

// MockFormatter implements FormatterPlugin for testing
type MockFormatter struct {
	name string
}

func (f *MockFormatter) Name() string { return f.name }
func (f *MockFormatter) Format(_ []sdk.Finding, _ interface{}) error {
	return nil
}

func TestManager_RegisterFormatter(t *testing.T) {
	manager := NewManager(nil)

	formatter := &MockFormatter{name: "test-formatter"}
	manager.RegisterFormatter(formatter)

	formatters := manager.GetFormatters()
	assert.Len(t, formatters, 1)
	assert.Equal(t, formatter, formatters["test-formatter"])
}

func TestManager_GetFormatter(t *testing.T) {
	manager := NewManager(nil)

	formatter := &MockFormatter{name: "test-formatter"}
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
	manager := NewManager(nil)

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
	manager := NewManager(nil)

	rule := &MockRule{name: "test-rule"}
	manager.RegisterRule(rule)

	// Get rules and modify the returned map
	rules := manager.GetRules()
	delete(rules, "test-rule")

	// Original should still have the rule
	originalRules := manager.GetRules()
	assert.Len(t, originalRules, 1)
}

func TestManager_GetEngines_ReturnsCopy(t *testing.T) {
	manager := NewManager(nil)

	engine := &MockEngine{name: "test-engine"}
	manager.RegisterEngine(engine)

	// Get engines and modify the returned map
	engines := manager.GetEngines()
	delete(engines, "test-engine")

	// Original should still have the engine
	originalEngines := manager.GetEngines()
	assert.Len(t, originalEngines, 1)
}

func TestManager_GetFormatters_ReturnsCopy(t *testing.T) {
	manager := NewManager(nil)

	formatter := &MockFormatter{name: "test-formatter"}
	manager.RegisterFormatter(formatter)

	// Get formatters and modify the returned map
	formatters := manager.GetFormatters()
	delete(formatters, "test-formatter")

	// Original should still have the formatter
	originalFormatters := manager.GetFormatters()
	assert.Len(t, originalFormatters, 1)
}

func TestManager_ConcurrentAccess(_ *testing.T) {
	manager := NewManager(nil)

	// Run concurrent operations
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			manager.RegisterRule(&MockRule{name: "rule-" + string(rune('a'+i%26))})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.GetRules()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			manager.RegisterEngine(&MockEngine{name: "engine-" + string(rune('a'+i%26))})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.GetEngines()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Should not panic
}

// testRulePlugin implements RulePlugin for interface verification
type testRulePlugin struct{}

func (p *testRulePlugin) GetRules() []sdk.Rule {
	return []sdk.Rule{&MockRule{name: "test"}}
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
	var _ EnginePlugin = &MockEngine{}
}

func TestFormatterPluginInterface(_ *testing.T) {
	// Verify the FormatterPlugin interface
	var _ FormatterPlugin = &MockFormatter{}
}
