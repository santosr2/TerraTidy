//go:build !windows

package plugins

import (
	"fmt"
	"path/filepath"
)

// loadAndRegisterBashRule loads a bash rule and verifies its checksum if enabled.
// If checksums is non-nil and verification is enabled, the script is verified first.
func (m *Manager) loadAndRegisterBashRule(path, name string, checksums map[string]string) error {
	// Verify bash script integrity before loading (if enabled and manifest exists)
	if m.verifyIntegrity && checksums != nil {
		if err := m.verifyPluginChecksum(path, checksums); err != nil {
			// In warn-only mode (first release), log warning but continue
			// TODO(#228): enforce verification once warn-only deprecation period closes
			m.logger.Printf("[WARN] bash rule verification failed for %s: %v (loading anyway - warn-only mode)", filepath.Base(path), err)
		}
	}

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
