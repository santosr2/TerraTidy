import * as assert from 'node:assert';
import { execFileSync } from 'node:child_process';
import * as vscode from 'vscode';

function findBinary(): string | undefined {
    // Check env var first, then PATH
    const envBin = process.env.TERRATIDY_BIN;
    if (envBin) {
        return envBin;
    }

    try {
        execFileSync('terratidy', ['version'], { stdio: 'ignore' });
        return 'terratidy';
    } catch {
        return undefined;
    }
}

suite('LSP Lifecycle', () => {
    const binary = findBinary();

    test('LSP client starts with real binary', async function () {
        if (!binary) {
            this.skip();
            return;
        }

        const ext = vscode.extensions.getExtension('santosr2.terratidy');
        assert.ok(ext, 'Extension should be installed');

        // Open a .tf file to trigger activation
        const doc = await vscode.workspace.openTextDocument({
            language: 'terraform',
            content: 'resource "null_resource" "test" {}',
        });
        await vscode.window.showTextDocument(doc);
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

        const ext = vscode.extensions.getExtension('santosr2.terratidy');
        assert.ok(ext?.isActive, 'Extension should still be active after restart');
    });
});
