//go:build windows

package plugins

// loadAndRegisterBashRule is a no-op on Windows since bash scripts
// require a Unix shell to execute.
func (m *Manager) loadAndRegisterBashRule(_, _ string) error {
	return nil
}
