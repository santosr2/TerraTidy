import * as assert from 'node:assert';
import * as vscode from 'vscode';
import { findBinary, getExtension } from './helpers';

suite('LSP Lifecycle', () => {
  const binary = findBinary();

  test('LSP client starts with real binary', async function () {
    if (!binary) {
      this.skip();
      return;
    }

    const ext = getExtension();
    assert.ok(ext, 'Extension should be installed');

    // Activate the extension directly. Pattern-based activation (workspaceContains)
    // can't be easily triggered in tests, so we call activate() explicitly.
    await ext.activate();

    // Give the LSP client time to start
    await new Promise((resolve) => setTimeout(resolve, 5000));

    // The extension is active, meaning the LSP client was created.
    // We can't directly inspect the client state from tests, but
    // activation without error is the success criterion here.
    assert.strictEqual(ext.isActive, true, 'Extension should remain active after LSP start');
  });

  test('restart command does not throw', async function () {
    if (!binary) {
      this.skip();
      return;
    }

    // Execute restart command; it should not throw
    await vscode.commands.executeCommand('terratidy.restartServer');

    // Give time for restart
    await new Promise((resolve) => setTimeout(resolve, 3000));

    const ext = getExtension();
    assert.ok(ext?.isActive, 'Extension should still be active after restart');
  });
});
