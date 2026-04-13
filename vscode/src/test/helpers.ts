import { execFileSync } from 'node:child_process';
import * as fs from 'node:fs';
import * as os from 'node:os';
import * as path from 'node:path';
import * as vscode from 'vscode';

/**
 * Find the TerraTidy binary.
 * Checks TERRATIDY_BIN env var first, then PATH.
 * Returns undefined if binary is not available.
 */
export function findBinary(): string | undefined {
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

/**
 * Create a temporary Terraform file with the given content.
 * Returns the absolute path to the file.
 * The file is placed in a temp directory that should be cleaned up with cleanupTempFile().
 */
export function createTempTfFile(content: string, prefix: string = 'terratidy-test-'): string {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  const tfFile = path.join(tmpDir, 'main.tf');
  fs.writeFileSync(tfFile, content);
  return tfFile;
}

/**
 * Clean up a temporary file and its parent directory.
 * Silently ignores errors (e.g., if already deleted).
 */
export function cleanupTempFile(filePath: string): void {
  try {
    const dir = path.dirname(filePath);
    fs.rmSync(dir, { recursive: true, force: true });
  } catch (e) {
    console.warn('Temp cleanup failed:', e);
  }
}

/**
 * Wait for diagnostics to appear on a URI with polling.
 * Returns diagnostics when at least expectedCount are present, or after timeout.
 *
 * @param uri - The document URI to check
 * @param expectedCount - Minimum number of diagnostics to wait for (must be > 0 for polling to be useful)
 * @param timeoutMs - Maximum time to wait (default 10000ms)
 */
export async function waitForDiagnostics(
  uri: vscode.Uri,
  expectedCount: number,
  timeoutMs: number = 10000
): Promise<vscode.Diagnostic[]> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const diagnostics = vscode.languages.getDiagnostics(uri);
    if (diagnostics.length >= expectedCount) {
      return diagnostics;
    }
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  // Return whatever we have after timeout
  return vscode.languages.getDiagnostics(uri);
}

/**
 * Get code actions for a range in a document.
 * Returns empty array if no actions available.
 */
export async function getCodeActions(uri: vscode.Uri, range: vscode.Range): Promise<vscode.CodeAction[]> {
  const actions = await vscode.commands.executeCommand<vscode.CodeAction[]>(
    'vscode.executeCodeActionProvider',
    uri,
    range
  );
  return actions || [];
}

/**
 * Get the TerraTidy extension instance.
 */
export function getExtension(): vscode.Extension<unknown> | undefined {
  return vscode.extensions.getExtension('santosr2.vscode-terratidy');
}
