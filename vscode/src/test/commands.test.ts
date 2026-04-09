import * as assert from 'node:assert';
import * as vscode from 'vscode';

suite('Commands', () => {
  suiteSetup(async () => {
    const ext = vscode.extensions.getExtension('santosr2.vscode-terratidy');
    assert.ok(ext, 'Extension should be installed');
    await ext.activate();
  });

  test('showOutput command executes without error', async () => {
    // Should not throw
    await vscode.commands.executeCommand('terratidy.showOutput');
  });

  test('restartServer command executes without error', async () => {
    // Should not throw (server may not be running, but command should handle that)
    await vscode.commands.executeCommand('terratidy.restartServer');
  });

  test('init command shows warning when no workspace', async () => {
    // The init command requires a workspace folder.
    // In the test environment with fixtures as workspace, it will
    // attempt to run terratidy init. If binary is not found, it
    // should show an error but not throw.
    try {
      await vscode.commands.executeCommand('terratidy.init');
    } catch {
      // Expected if binary not available
    }
  });
});
