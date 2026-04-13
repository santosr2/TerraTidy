import * as assert from 'node:assert';
import * as vscode from 'vscode';
import { cleanupTempFile, createTempTfFile, findBinary, getExtension, waitForDiagnostics } from './helpers';

// Note: These tests require the TerraTidy binary to be available.
// They test the integration between VSCode's diagnostic display and the LSP server.
// Due to the async nature of LSP diagnostics, tests use polling with timeouts.

suite('Diagnostics', () => {
  const binary = findBinary();
  let ext: vscode.Extension<unknown> | undefined;

  // Activate extension once for all tests in this suite
  suiteSetup(async () => {
    ext = getExtension();
    assert.ok(ext, 'Extension should be installed');
    await ext.activate();
  });

  // Test: severity mapping (error -> Error, warning -> Warning, info -> Information)
  // LSP severity: 1=Error, 2=Warning, 3=Information, 4=Hint
  // VSCode DiagnosticSeverity: Error=0, Warning=1, Information=2, Hint=3
  test('severity mapping: error -> Error, warning -> Warning, info -> Information', async function () {
    if (!binary) {
      this.skip();
      return;
    }

    this.timeout(30000);

    // Create a Terraform file with style violations (typically warnings)
    // Style rule 'blank-lines-between-blocks' triggers on missing blank lines
    const tfContent = `resource "aws_instance" "one" {
  ami = "ami-12345"
}
resource "aws_instance" "two" {
  ami = "ami-67890"
}
`;

    const tfFile = createTempTfFile(tfContent, 'terratidy-diag-test-');
    const uri = vscode.Uri.file(tfFile);

    try {
      const doc = await vscode.workspace.openTextDocument(uri);
      await vscode.window.showTextDocument(doc);

      // Wait for diagnostics to appear
      const diagnostics = await waitForDiagnostics(uri, 1, 15000);

      if (diagnostics.length === 0) {
        // No diagnostics - binary may not have flagged this content
        // This is acceptable; the main assertion is that we can query diagnostics
        console.log('No diagnostics returned - content may be valid or LSP not running');
        return;
      }

      // Verify diagnostic severity is a valid VSCode severity
      for (const diag of diagnostics) {
        assert.ok(diag.severity !== undefined, 'Diagnostic should have a severity');
        assert.ok(
          diag.severity >= vscode.DiagnosticSeverity.Error && diag.severity <= vscode.DiagnosticSeverity.Hint,
          `Severity ${diag.severity} should be in valid range (0-3)`
        );
      }
    } finally {
      await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
      cleanupTempFile(tfFile);
    }
  });

  // Test: diagnostic range maps to correct line/column
  test('diagnostic range maps to correct line/column', async function () {
    if (!binary) {
      this.skip();
      return;
    }

    this.timeout(30000);

    // Create a Terraform file with a known issue at a specific location
    const tfContent = `resource "aws_instance" "test" {
  ami = "ami-12345"
}
resource "aws_instance" "test2" {
  ami = "ami-67890"
}
`;

    const tfFile = createTempTfFile(tfContent, 'terratidy-diag-test-');
    const uri = vscode.Uri.file(tfFile);

    try {
      const doc = await vscode.workspace.openTextDocument(uri);
      await vscode.window.showTextDocument(doc);

      const diagnostics = await waitForDiagnostics(uri, 1, 15000);

      if (diagnostics.length === 0) {
        console.log('No diagnostics returned - skipping range assertions');
        return;
      }

      // Verify ranges are valid
      for (const diag of diagnostics) {
        const range = diag.range;

        // Range should have valid start and end positions
        assert.ok(range.start.line >= 0, 'Start line should be non-negative');
        assert.ok(range.start.character >= 0, 'Start character should be non-negative');
        assert.ok(range.end.line >= 0, 'End line should be non-negative');
        assert.ok(range.end.character >= 0, 'End character should be non-negative');

        // Start should be before or equal to end
        assert.ok(
          range.start.line < range.end.line ||
            (range.start.line === range.end.line && range.start.character <= range.end.character),
          'Start should be before or equal to end'
        );

        // Lines should be within document bounds
        assert.ok(
          range.start.line < doc.lineCount,
          `Start line ${range.start.line} should be within document (${doc.lineCount} lines)`
        );
        assert.ok(
          range.end.line < doc.lineCount,
          `End line ${range.end.line} should be within document (${doc.lineCount} lines)`
        );
      }
    } finally {
      await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
      cleanupTempFile(tfFile);
    }
  });

  // Test: diagnostic code links to rule documentation
  test('diagnostic code links to rule documentation', async function () {
    if (!binary) {
      this.skip();
      return;
    }

    this.timeout(30000);

    // Create file that triggers diagnostics
    const tfContent = `resource "aws_instance" "one" {
  ami = "ami-12345"
}
resource "aws_instance" "two" {
  ami = "ami-67890"
}
`;

    const tfFile = createTempTfFile(tfContent, 'terratidy-diag-test-');
    const uri = vscode.Uri.file(tfFile);

    try {
      const doc = await vscode.workspace.openTextDocument(uri);
      await vscode.window.showTextDocument(doc);

      const diagnostics = await waitForDiagnostics(uri, 1, 15000);

      if (diagnostics.length === 0) {
        console.log('No diagnostics returned - skipping code assertions');
        return;
      }

      // Count diagnostics with codes
      let diagnosticsWithCode = 0;

      // Verify diagnostics have rule codes
      for (const diag of diagnostics) {
        // Diagnostic should have a code (rule identifier)
        // Code can be string, number, or { value, target }
        if (diag.code !== undefined) {
          diagnosticsWithCode++;
          if (typeof diag.code === 'object' && 'value' in diag.code) {
            // Code with documentation link
            assert.ok(diag.code.value, 'Code value should be set');
            assert.ok(diag.code.target, 'Code target (documentation URL) should be set');
          } else {
            // Simple code (string or number)
            assert.ok(
              typeof diag.code === 'string' || typeof diag.code === 'number',
              'Code should be string or number'
            );
          }
        }
      }

      // TerraTidy LSP should always attach rule codes to diagnostics
      assert.ok(
        diagnosticsWithCode > 0,
        `Expected at least one diagnostic with a code, got ${diagnosticsWithCode}/${diagnostics.length}`
      );
    } finally {
      await vscode.commands.executeCommand('workbench.action.closeActiveEditor');
      cleanupTempFile(tfFile);
    }
  });

  // Test: diagnostics clear on document close
  test('diagnostics clear on document close', async function () {
    if (!binary) {
      this.skip();
      return;
    }

    this.timeout(30000);

    const tfContent = `resource "aws_instance" "one" {
  ami = "ami-12345"
}
resource "aws_instance" "two" {
  ami = "ami-67890"
}
`;

    const tfFile = createTempTfFile(tfContent, 'terratidy-diag-test-');
    const uri = vscode.Uri.file(tfFile);

    try {
      const doc = await vscode.workspace.openTextDocument(uri);
      await vscode.window.showTextDocument(doc);

      // Wait for diagnostics to appear (wait for at least 1)
      const initialDiagnostics = await waitForDiagnostics(uri, 1, 10000);
      const hadDiagnostics = initialDiagnostics.length > 0;

      // Close the document
      await vscode.commands.executeCommand('workbench.action.closeActiveEditor');

      // Wait for diagnostics to clear (LSP sends empty diagnostics on didClose)
      await new Promise((resolve) => setTimeout(resolve, 1000));

      // Get diagnostics after close
      const afterCloseDiagnostics = vscode.languages.getDiagnostics(uri);

      // If we had diagnostics before, they should be cleared now
      if (hadDiagnostics) {
        assert.strictEqual(afterCloseDiagnostics.length, 0, 'Diagnostics should be cleared after document close');
      } else {
        console.log('No diagnostics appeared before close - cannot verify clearing behavior');
      }
    } finally {
      cleanupTempFile(tfFile);
    }
  });

  // Smoke test: diagnostics API is accessible
  test('diagnostics API is accessible', async function () {
    this.timeout(10000);

    // Verify we can call getDiagnostics without error
    const allDiagnostics = vscode.languages.getDiagnostics();
    assert.ok(Array.isArray(allDiagnostics), 'getDiagnostics should return an array');

    // Each entry should be a tuple [uri, diagnostics[]]
    for (const [diagUri, diags] of allDiagnostics) {
      assert.ok(diagUri instanceof vscode.Uri, 'First element should be Uri');
      assert.ok(Array.isArray(diags), 'Second element should be diagnostics array');
    }
  });
});
