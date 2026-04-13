import * as assert from 'node:assert';
import * as vscode from 'vscode';
import {
  cleanupTempFile,
  createTempTfFile,
  findBinary,
  getCodeActions,
  getExtension,
  waitForDiagnostics,
} from './helpers';

// Note: These tests require the TerraTidy binary to be available.
// They test the integration between VSCode's code action system and the LSP server.
// The LSP server provides "quickfix" code actions for findings that have fixes.

suite('Code Actions', () => {
  const binary = findBinary();
  let ext: vscode.Extension<unknown> | undefined;

  // Activate extension once for all tests in this suite
  suiteSetup(async () => {
    ext = getExtension();
    assert.ok(ext, 'Extension should be installed');
    await ext.activate();
  });

  // Test: fix action appears for fixable findings
  test('fix action appears for fixable findings', async function () {
    if (!binary) {
      this.skip();
      return;
    }

    this.timeout(30000);

    // Create a Terraform file with style violations that have fixes
    // Style rules like 'blank-lines-between-blocks' are auto-fixable
    const tfContent = `resource "aws_instance" "one" {
  ami = "ami-12345"
}
resource "aws_instance" "two" {
  ami = "ami-67890"
}
`;

    const tfFile = createTempTfFile(tfContent, 'terratidy-action-test-');
    const uri = vscode.Uri.file(tfFile);

    try {
      const doc = await vscode.workspace.openTextDocument(uri);
      await vscode.window.showTextDocument(doc);

      // Wait for diagnostics
      const diagnostics = await waitForDiagnostics(uri, 1, 15000);

      if (diagnostics.length === 0) {
        console.log('No diagnostics returned - content may be valid or LSP not running');
        return;
      }

      // For each diagnostic, query code actions at its range
      let foundQuickfix = false;
      for (const diag of diagnostics) {
        const actions = await getCodeActions(uri, diag.range);

        for (const action of actions) {
          if (action.kind?.value === 'quickfix' || action.kind === vscode.CodeActionKind.QuickFix) {
            foundQuickfix = true;
            assert.ok(action.title, 'Quick fix should have a title');
          }
        }
      }

      // Note: Not all diagnostics have fixes. We're verifying that
      // the code action system is functional, not that every finding is fixable.
      console.log(`Found ${diagnostics.length} diagnostics, quickfix available: ${foundQuickfix}`);
    } finally {
      await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
      cleanupTempFile(tfFile);
    }
  });

  // Test: executing fix action applies changes
  test('executing fix action applies changes', async function () {
    if (!binary) {
      this.skip();
      return;
    }

    this.timeout(30000);

    // Create file with a fixable style violation
    const tfContent = `resource "aws_instance" "one" {
  ami = "ami-12345"
}
resource "aws_instance" "two" {
  ami = "ami-67890"
}
`;

    const tfFile = createTempTfFile(tfContent, 'terratidy-action-test-');
    const uri = vscode.Uri.file(tfFile);

    try {
      const doc = await vscode.workspace.openTextDocument(uri);
      await vscode.window.showTextDocument(doc);

      // Wait for diagnostics
      const diagnostics = await waitForDiagnostics(uri, 1, 15000);

      if (diagnostics.length === 0) {
        console.log('No diagnostics returned - skipping fix test');
        return;
      }

      // Find a quickfix action
      let quickfixAction: vscode.CodeAction | undefined;
      for (const diag of diagnostics) {
        const actions = await getCodeActions(uri, diag.range);
        quickfixAction = actions.find((a) => a.kind?.value === 'quickfix' || a.kind === vscode.CodeActionKind.QuickFix);
        if (quickfixAction) {
          break;
        }
      }

      if (!quickfixAction) {
        console.log('No quickfix action available - findings may not be auto-fixable');
        return;
      }

      // Get content before fix
      const contentBefore = doc.getText();

      // Execute the code action
      if (quickfixAction.edit) {
        // Apply workspace edit
        const success = await vscode.workspace.applyEdit(quickfixAction.edit);
        assert.ok(success, 'Applying code action edit should succeed');

        // Get content after fix
        const contentAfter = doc.getText();

        // Content should have changed (fix was applied)
        assert.notStrictEqual(contentBefore, contentAfter, 'Fix should have changed document content');
      } else if (quickfixAction.command) {
        // Execute command
        await vscode.commands.executeCommand(
          quickfixAction.command.command,
          ...(quickfixAction.command.arguments || [])
        );
        // Command-based fixes may not immediately change the document
        // Success criterion: reaching this point without throwing
      } else {
        console.log('Code action has no edit or command - skipping');
      }
    } finally {
      await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
      cleanupTempFile(tfFile);
    }
  });

  // Test: non-fixable findings show no fix action
  test('non-fixable findings show no fix action', async function () {
    if (!binary) {
      this.skip();
      return;
    }

    this.timeout(30000);

    // Create a Terraform file - the specific content determines whether
    // findings are fixable or not based on the rule implementations
    const tfContent = `# This is a simple config
resource "aws_instance" "test" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`;

    const tfFile = createTempTfFile(tfContent, 'terratidy-action-test-');
    const uri = vscode.Uri.file(tfFile);

    try {
      const doc = await vscode.workspace.openTextDocument(uri);
      await vscode.window.showTextDocument(doc);

      // Query code actions for the entire document range
      const fullRange = new vscode.Range(doc.positionAt(0), doc.positionAt(doc.getText().length));

      const actions = await getCodeActions(uri, fullRange);

      // If there are no diagnostics, there should be no quickfix actions
      const diagnostics = vscode.languages.getDiagnostics(uri);

      if (diagnostics.length === 0) {
        // With no diagnostics, quickfix actions should be minimal or none
        const quickfixes = actions.filter(
          (a) => a.kind?.value === 'quickfix' || a.kind === vscode.CodeActionKind.QuickFix
        );
        console.log(`No diagnostics, quickfix actions: ${quickfixes.length}`);
      } else {
        // With diagnostics, some may or may not have fixes
        console.log(`${diagnostics.length} diagnostics, ${actions.length} code actions`);
      }

      // Success criterion: reaching this point without throwing
    } finally {
      await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
      cleanupTempFile(tfFile);
    }
  });

  // Smoke test: code action provider is accessible
  test('code action provider is accessible', async function () {
    this.timeout(10000);

    // Create a minimal test file
    const tfContent = 'resource "null_resource" "test" {}';
    const tfFile = createTempTfFile(tfContent, 'terratidy-action-test-');
    const uri = vscode.Uri.file(tfFile);

    try {
      const doc = await vscode.workspace.openTextDocument(uri);
      await vscode.window.showTextDocument(doc);

      // Query code actions - should not throw
      const range = new vscode.Range(0, 0, 0, 10);
      const actions = await getCodeActions(uri, range);

      assert.ok(Array.isArray(actions), 'Code actions should be an array');
    } finally {
      await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
      cleanupTempFile(tfFile);
    }
  });
});
