import * as assert from 'node:assert';
import * as vscode from 'vscode';
import { getInitializationOptions } from '../extension';

// Tests for settings behavior: configuration changes and LSP initialization options.
// Note: VSCode extension tests cannot interact with UI prompts (e.g., the "Restart?" dialog
// shown on config change). These tests verify that config changes don't crash and that
// settings are correctly reflected in LSP initialization options.
// Important: VSCode caches extension activation, so ext.activate() returns the cached
// promise on subsequent calls (see error-handling.test.ts for more details).
suite('Settings Behavior', () => {
  let ext: vscode.Extension<unknown> | undefined;

  // Activate extension once for all tests in this suite
  suiteSetup(async () => {
    ext = vscode.extensions.getExtension('santosr2.vscode-terratidy');
    assert.ok(ext, 'Extension should be installed');
    await ext.activate();
  });

  // Helper to update configuration
  async function updateConfig(key: string, value: unknown): Promise<void> {
    const config = vscode.workspace.getConfiguration('terratidy');
    await config.update(key, value, vscode.ConfigurationTarget.Global);
  }

  // Helper to reset configuration
  async function resetConfig(key: string): Promise<void> {
    const config = vscode.workspace.getConfiguration('terratidy');
    await config.update(key, undefined, vscode.ConfigurationTarget.Global);
  }

  // Test: changing executablePath triggers restart prompt (but doesn't crash)
  test('changing executablePath: config change handled gracefully', async function () {
    this.timeout(15000);
    assert.ok(ext, 'Extension should be activated');

    // Change executablePath - extension shows prompt but doesn't crash
    const originalPath = vscode.workspace.getConfiguration('terratidy').get<string>('executablePath');
    try {
      await updateConfig('executablePath', '/tmp/custom-terratidy');

      // Wait a moment for config change listener to fire (shows prompt)
      await new Promise((resolve) => setTimeout(resolve, 100));

      // Extension should remain active
      assert.strictEqual(ext.isActive, true, 'Extension should remain active after config change');

      // Manually trigger restart to apply new config
      // This will fail since the path doesn't exist, but shouldn't crash
      await vscode.commands.executeCommand('terratidy.restartServer');

      // Extension still active
      assert.strictEqual(ext.isActive, true, 'Extension should remain active after restart attempt');
    } finally {
      // Restore original path
      if (originalPath) {
        await updateConfig('executablePath', originalPath);
      } else {
        await resetConfig('executablePath');
      }
    }
  });

  // Test: changing configPath triggers restart prompt (but doesn't crash)
  test('changing configPath: config change handled gracefully', async function () {
    this.timeout(15000);
    assert.ok(ext, 'Extension should be activated');

    try {
      await updateConfig('configPath', '/tmp/custom-terratidy.yaml');

      // Wait for config change listener (shows prompt)
      await new Promise((resolve) => setTimeout(resolve, 100));

      // Extension should remain active
      assert.strictEqual(ext.isActive, true, 'Extension should remain active');

      // Config change should be reflected in initialization options
      const options = getInitializationOptions();
      assert.strictEqual(options.configPath, '/tmp/custom-terratidy.yaml', 'configPath should be updated');
    } finally {
      await resetConfig('configPath');
    }
  });

  // Test: changing engine toggles are reflected in initialization options
  test('changing engine toggles: reflected in initialization options', async function () {
    this.timeout(15000);
    assert.ok(ext, 'Extension should be activated');

    // Save original values
    const config = vscode.workspace.getConfiguration('terratidy');
    const originalFmt = config.get<boolean>('engines.fmt');
    const originalPolicy = config.get<boolean>('engines.policy');

    try {
      // Toggle fmt off (default is true)
      await updateConfig('engines.fmt', false);

      // Toggle policy on (default is false)
      await updateConfig('engines.policy', true);

      // Verify initialization options reflect changes (reads config synchronously)
      const options = getInitializationOptions();
      const engines = options.engines as Record<string, boolean>;

      assert.strictEqual(engines.fmt, false, 'engines.fmt should be false');
      assert.strictEqual(engines.policy, true, 'engines.policy should be true');
    } finally {
      // Restore original values
      if (originalFmt !== undefined) {
        await updateConfig('engines.fmt', originalFmt);
      } else {
        await resetConfig('engines.fmt');
      }
      if (originalPolicy !== undefined) {
        await updateConfig('engines.policy', originalPolicy);
      } else {
        await resetConfig('engines.policy');
      }
    }
  });

  // Test: profile setting is reflected in initialization options
  // Note: Default value test is in configuration.test.ts
  test('profile: update reflected in initialization options', async function () {
    this.timeout(10000);
    assert.ok(ext, 'Extension should be activated');

    try {
      // Set a profile
      await updateConfig('profile', 'production');

      const options = getInitializationOptions();
      assert.strictEqual(options.profile, 'production', 'profile should be set to production');
    } finally {
      await resetConfig('profile');
    }
  });

  // Test: severityThreshold setting is reflected when explicitly set
  // Note: Default undefined behavior tested in configuration.test.ts
  test('severityThreshold: explicit value reflected in initialization options', async function () {
    this.timeout(10000);
    assert.ok(ext, 'Extension should be activated');

    try {
      // Explicitly set threshold
      await updateConfig('severityThreshold', 'error');

      const options = getInitializationOptions();
      assert.strictEqual(options.severityThreshold, 'error', 'severityThreshold should be set to error');
    } finally {
      await resetConfig('severityThreshold');
    }
  });

  // Test: restartServer command executes without error after config change
  // Note: This verifies the command doesn't throw; the tautological assertion
  // about config values is intentionally removed as getInitializationOptions
  // reads config synchronously regardless of restart.
  test('restartServer: command executes after config change', async function () {
    this.timeout(20000);
    assert.ok(ext, 'Extension should be activated');

    try {
      // Change a setting
      await updateConfig('profile', 'after-restart');

      // Trigger restart - should not throw
      await vscode.commands.executeCommand('terratidy.restartServer');

      // Extension should remain functional
      assert.strictEqual(ext.isActive, true, 'Extension should remain active after restart');
    } finally {
      await resetConfig('profile');
    }
  });
});
