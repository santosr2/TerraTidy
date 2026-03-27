//go:build !windows

package plugins

import "fmt"

func (m *Manager) loadAndRegisterBashRule(path, name string) error {
	rule, err := loadBashRule(path)
	if err != nil {
		return fmt.Errorf("loading Bash rule %s: %w", name, err)
	}
	m.RegisterRule(rule)
	m.mu.Lock()
	m.plugins[rule.Name()] = &Plugin{
		Metadata: PluginMetadata{
			Name: rule.Name(),
			Type: PluginTypeRule,
			Path: path,
		},
		Instance: rule,
	}
	m.mu.Unlock()
	return nil
}
