import * as assert from 'node:assert';
import * as vscode from 'vscode';

suite('Extension Activation', () => {
  test('extension is present', () => {
    const ext = vscode.extensions.getExtension('santosr2.vscode-terratidy');
    assert.ok(ext, 'Extension should be installed');
  });

  test('extension activates on .tf file', async () => {
    const ext = vscode.extensions.getExtension('santosr2.vscode-terratidy');
    assert.ok(ext, 'Extension should be installed');

    // Activate the extension directly. Pattern-based activation (workspaceContains)
    // can't be easily triggered in tests, so we call activate() explicitly.
    // The extension uses file patterns, not language IDs, to avoid conflicts
    // with other extensions like HashiCorp Terraform.
    await ext.activate();
    assert.strictEqual(ext.isActive, true, 'Extension should be active');
  });

  test('registers all commands', async () => {
    const commands = await vscode.commands.getCommands(true);

    const expected = ['terratidy.init', 'terratidy.showOutput', 'terratidy.restartServer'];

    for (const cmd of expected) {
      assert.ok(commands.includes(cmd), `Command "${cmd}" should be registered`);
    }
  });
});
