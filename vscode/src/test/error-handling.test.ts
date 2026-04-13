import * as assert from 'node:assert';
import * as fs from 'node:fs';
import * as os from 'node:os';
import * as path from 'node:path';
import * as vscode from 'vscode';

// Note: VSCode extension tests have a limitation - calling ext.activate() on an already-active
// extension returns the cached activation promise and does not re-run initialization with new config.
// These tests still verify the extension doesn't crash and remains functional, but may not fully
// exercise the error paths when the extension was already activated by earlier tests.
suite('Error Handling', () => {
  // Helper to get extension
  function getExtension() {
    return vscode.extensions.getExtension('santosr2.vscode-terratidy');
  }

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

  // Helper for tests that need a temp config file
  // Creates temp dir, writes content, sets configPath, runs test, cleans up
  async function withTempConfig(content: string, fn: () => Promise<void>): Promise<void> {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'terratidy-test-'));
    const configPath = path.join(tmpDir, '.terratidy.yaml');

    try {
      fs.writeFileSync(configPath, content);
      await updateConfig('configPath', configPath);
      await fn();
    } finally {
      await resetConfig('configPath');
      try {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      } catch (e) {
        console.warn('Temp cleanup failed:', e);
      }
    }
  }

  // Helper for tests that need a fake executable that crashes
  // Creates a shell script that passes 'version' but exits on 'lsp'
  async function withCrashScript(fn: () => Promise<void>): Promise<void> {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'terratidy-crash-test-'));
    const crashScript = path.join(tmpDir, 'terratidy-crash');

    try {
      // Script passes version check but exits on any other command (simulates crash)
      fs.writeFileSync(
        crashScript,
        `#!/bin/sh
if [ "$1" = "version" ]; then
  echo "terratidy version test"
  exit 0
fi
exit 1
`
      );
      fs.chmodSync(crashScript, 0o755);
      await updateConfig('executablePath', crashScript);
      await fn();
    } finally {
      await resetConfig('executablePath');
      try {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      } catch (e) {
        console.warn('Temp cleanup failed:', e);
      }
    }
  }

  test('binary not found: extension activates gracefully', async function () {
    this.timeout(10000);

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    // Set executablePath to a non-existent binary
    await updateConfig('executablePath', '/nonexistent/path/to/terratidy-does-not-exist');

    try {
      // Activate the extension - should not throw even with invalid binary
      await ext.activate();

      // Extension should still be marked as active (activation succeeded).
      // With a missing binary, the spawn fails immediately - no async wait needed.
      assert.strictEqual(ext.isActive, true, 'Extension should activate despite missing binary');
    } finally {
      // Reset configuration to avoid affecting other tests
      await resetConfig('executablePath');
    }
  });

  test('binary not found: commands still registered', async function () {
    this.timeout(10000);

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    // Set executablePath to non-existent binary
    await updateConfig('executablePath', '/nonexistent/terratidy-binary');

    try {
      await ext.activate();

      const commands = await vscode.commands.getCommands(true);

      // All commands should still be registered even when binary is missing
      const expected = ['terratidy.init', 'terratidy.showOutput', 'terratidy.restartServer'];

      for (const cmd of expected) {
        assert.ok(commands.includes(cmd), `Command "${cmd}" should still be registered`);
      }
    } finally {
      await resetConfig('executablePath');
    }
  });

  test('binary not found: showOutput command works', async function () {
    this.timeout(10000);

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    await updateConfig('executablePath', '/nonexistent/terratidy-binary');

    try {
      await ext.activate();

      // showOutput should not throw even when binary is missing
      // (output channel is created during activation regardless).
      // Success criterion: reaching this line without throwing.
      await vscode.commands.executeCommand('terratidy.showOutput');
    } finally {
      await resetConfig('executablePath');
    }
  });

  test('binary not found: restartServer command handles error gracefully', async function () {
    this.timeout(15000);

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    await updateConfig('executablePath', '/nonexistent/terratidy-binary');

    try {
      await ext.activate();

      // restartServer should not throw even when binary is missing.
      // With a non-existent binary, the spawn fails immediately.
      await vscode.commands.executeCommand('terratidy.restartServer');

      // Extension should still be active after failed restart attempt
      assert.strictEqual(ext.isActive, true, 'Extension should remain active after restart attempt');
    } finally {
      await resetConfig('executablePath');
    }
  });

  test('workspace without .tf files: extension activates', async () => {
    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    // Extension uses workspaceContains activation which may or may not trigger
    // in test environment. Direct activation should still work.
    await ext.activate();

    assert.strictEqual(ext.isActive, true, 'Extension should activate in workspace without .tf files');
  });

  test('workspace without .tf files: commands available', async () => {
    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    await ext.activate();

    const commands = await vscode.commands.getCommands(true);

    // Commands should be available regardless of workspace content
    assert.ok(commands.includes('terratidy.init'), 'init command should be available');
    assert.ok(commands.includes('terratidy.showOutput'), 'showOutput command should be available');
    assert.ok(commands.includes('terratidy.restartServer'), 'restartServer command should be available');
  });

  // Malformed config tests verify the extension doesn't crash when encountering
  // invalid configuration. Due to VSCode's activation caching (see file header),
  // these primarily verify non-crash behavior rather than detailed error paths.
  // The restartServer command is used to trigger config re-reading.

  test('malformed config: extension handles gracefully', async function () {
    this.timeout(15000);

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    // Malformed YAML: unclosed bracket, duplicate key
    const malformedYaml = `version: 1
engines:
  fmt:
    enabled: [true  # unclosed bracket
  style:
    enabled: true
engines:  # duplicate key
  lint: true
`;

    await withTempConfig(malformedYaml, async () => {
      await ext.activate();

      // Restart server to apply bad config - should not throw
      await vscode.commands.executeCommand('terratidy.restartServer');

      // Extension should remain active after encountering malformed config
      assert.strictEqual(ext.isActive, true, 'Extension should remain active with malformed config');

      // Commands should still be available
      const commands = await vscode.commands.getCommands(true);
      assert.ok(commands.includes('terratidy.showOutput'), 'showOutput should still work');
    });
  });

  test('malformed config: invalid version field handled', async function () {
    this.timeout(15000);

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    // Invalid version type (string instead of number)
    const invalidVersionYaml = `version: "not-a-number"
engines:
  fmt:
    enabled: true
`;

    await withTempConfig(invalidVersionYaml, async () => {
      await ext.activate();
      assert.strictEqual(ext.isActive, true, 'Extension should remain active');
    });
  });

  test('malformed config: empty file handled', async function () {
    this.timeout(15000);

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    await withTempConfig('', async () => {
      await ext.activate();
      assert.strictEqual(ext.isActive, true, 'Extension should remain active with empty config');
    });
  });

  test('config not found: nonexistent path handled', async function () {
    this.timeout(15000);

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    try {
      // Set configPath to nonexistent file
      await updateConfig('configPath', '/nonexistent/path/to/.terratidy.yaml');

      await ext.activate();

      // Restart to apply config change
      await vscode.commands.executeCommand('terratidy.restartServer');

      // Extension should handle missing config gracefully
      assert.strictEqual(ext.isActive, true, 'Extension should remain active with missing config');
    } finally {
      await resetConfig('configPath');
    }
  });

  // LSP server exit tests verify graceful degradation when the server exits unexpectedly.
  // Note: vscode-languageclient handles server exits internally. The extension doesn't
  // currently implement a custom restart prompt - this tests the default behavior.
  // These are smoke tests: the primary assertion is that commands don't throw.
  // ext.isActive cannot become false after VSCode extension activation, so we don't
  // assert on it meaningfully here.

  test('server exit: extension handles gracefully', async function () {
    // Skip on Windows where shell scripts don't work the same way
    if (process.platform === 'win32') {
      this.skip();
    }

    this.timeout(15000);

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    await withCrashScript(async () => {
      await ext.activate();

      // Smoke test: restartServer should not throw even when server exits immediately.
      // If this throws, the test fails - that's the actual assertion.
      await vscode.commands.executeCommand('terratidy.restartServer');

      // Verify commands are still registered (expected to pass, but confirms no state corruption)
      const commands = await vscode.commands.getCommands(true);
      assert.ok(commands.includes('terratidy.restartServer'), 'restartServer should still be available');
    });
  });

  test('server exit: multiple restart attempts handled', async function () {
    // Skip on Windows where shell scripts don't work the same way
    if (process.platform === 'win32') {
      this.skip();
    }

    this.timeout(30000);

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    await withCrashScript(async () => {
      await ext.activate();

      // Smoke test: multiple restart attempts should all complete without throwing.
      // The actual assertion is that these commands don't throw.
      await vscode.commands.executeCommand('terratidy.restartServer');
      await vscode.commands.executeCommand('terratidy.restartServer');
      await vscode.commands.executeCommand('terratidy.restartServer');
    });
  });
});
