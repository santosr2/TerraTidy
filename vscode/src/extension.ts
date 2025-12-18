import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';

// Output channel for logging
let outputChannel: vscode.OutputChannel;

// Diagnostic collection for showing problems
let diagnosticCollection: vscode.DiagnosticCollection;

// Extension activation
export function activate(context: vscode.ExtensionContext) {
    outputChannel = vscode.window.createOutputChannel('TerraTidy');
    diagnosticCollection = vscode.languages.createDiagnosticCollection('terratidy');

    context.subscriptions.push(outputChannel);
    context.subscriptions.push(diagnosticCollection);

    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('terratidy.run', () => runTerraTidy('check')),
        vscode.commands.registerCommand('terratidy.format', () => runTerraTidy('fmt')),
        vscode.commands.registerCommand('terratidy.lint', () => runTerraTidy('lint')),
        vscode.commands.registerCommand('terratidy.style', () => runTerraTidy('style')),
        vscode.commands.registerCommand('terratidy.fix', () => runTerraTidy('fix')),
        vscode.commands.registerCommand('terratidy.init', initTerraTidy),
        vscode.commands.registerCommand('terratidy.showOutput', () => outputChannel.show())
    );

    // Register document formatter
    const terraformSelector: vscode.DocumentSelector = [
        { language: 'terraform', scheme: 'file' },
        { language: 'hcl', scheme: 'file' }
    ];

    context.subscriptions.push(
        vscode.languages.registerDocumentFormattingEditProvider(
            terraformSelector,
            new TerraTidyFormattingProvider()
        )
    );

    // Set up on-save handlers
    context.subscriptions.push(
        vscode.workspace.onDidSaveTextDocument(handleDocumentSave)
    );

    // Run initial check on open documents
    const config = vscode.workspace.getConfiguration('terratidy');
    if (config.get<boolean>('runOnSave')) {
        vscode.workspace.textDocuments.forEach(doc => {
            if (isTerraformFile(doc)) {
                runTerraTidyOnDocument(doc);
            }
        });
    }

    outputChannel.appendLine('TerraTidy extension activated');
}

// Extension deactivation
export function deactivate() {
    if (diagnosticCollection) {
        diagnosticCollection.dispose();
    }
}

// Check if a document is a Terraform/HCL file
function isTerraformFile(document: vscode.TextDocument): boolean {
    const languageId = document.languageId;
    const fileName = document.fileName;
    return (
        languageId === 'terraform' ||
        languageId === 'hcl' ||
        fileName.endsWith('.tf') ||
        fileName.endsWith('.tfvars') ||
        fileName.endsWith('.hcl')
    );
}

// Handle document save
async function handleDocumentSave(document: vscode.TextDocument) {
    if (!isTerraformFile(document)) {
        return;
    }

    const config = vscode.workspace.getConfiguration('terratidy');

    if (config.get<boolean>('fixOnSave')) {
        await runTerraTidyOnDocument(document, 'fix');
    } else if (config.get<boolean>('runOnSave')) {
        await runTerraTidyOnDocument(document, 'check');
    }
}

// Get the terratidy executable path
function getTerraTidyPath(): string {
    const config = vscode.workspace.getConfiguration('terratidy');
    const configPath = config.get<string>('executablePath');
    if (configPath && configPath.length > 0) {
        return configPath;
    }
    return 'terratidy';
}

// Get configuration file path
function getConfigPath(): string[] {
    const config = vscode.workspace.getConfiguration('terratidy');
    const configPath = config.get<string>('configPath');
    if (configPath && configPath.length > 0) {
        return ['--config', configPath];
    }
    return [];
}

// Get profile argument
function getProfileArgs(): string[] {
    const config = vscode.workspace.getConfiguration('terratidy');
    const profile = config.get<string>('profile');
    if (profile && profile.length > 0) {
        return ['--profile', profile];
    }
    return [];
}

// Get enabled engines
function getEnginesArgs(): string[] {
    const config = vscode.workspace.getConfiguration('terratidy');
    const engines: string[] = [];

    if (config.get<boolean>('engines.fmt')) engines.push('fmt');
    if (config.get<boolean>('engines.style')) engines.push('style');
    if (config.get<boolean>('engines.lint')) engines.push('lint');
    if (config.get<boolean>('engines.policy')) engines.push('policy');

    if (engines.length > 0 && engines.length < 4) {
        return ['--engines', engines.join(',')];
    }
    return [];
}

// Run TerraTidy command
async function runTerraTidy(command: string): Promise<void> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        vscode.window.showWarningMessage('No active editor');
        return;
    }

    const document = editor.document;
    if (!isTerraformFile(document)) {
        vscode.window.showWarningMessage('Not a Terraform/HCL file');
        return;
    }

    await runTerraTidyOnDocument(document, command);
}

// Run TerraTidy on a specific document
async function runTerraTidyOnDocument(
    document: vscode.TextDocument,
    command: string = 'check'
): Promise<void> {
    const filePath = document.fileName;
    const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
    const cwd = workspaceFolder?.uri.fsPath || path.dirname(filePath);

    const terraTidyPath = getTerraTidyPath();
    const args = [
        command,
        '--format', 'json',
        '--paths', filePath,
        ...getConfigPath(),
        ...getProfileArgs(),
        ...getEnginesArgs()
    ];

    outputChannel.appendLine(`Running: ${terraTidyPath} ${args.join(' ')}`);

    try {
        const result = await execTerraTidy(terraTidyPath, args, cwd);
        processTerraTidyOutput(document, result);
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        outputChannel.appendLine(`Error: ${errorMessage}`);
        vscode.window.showErrorMessage(`TerraTidy error: ${errorMessage}`);
    }
}

// Execute terratidy and return output
function execTerraTidy(
    executable: string,
    args: string[],
    cwd: string
): Promise<string> {
    return new Promise((resolve, reject) => {
        const process = cp.spawn(executable, args, {
            cwd,
            shell: true
        });

        let stdout = '';
        let stderr = '';

        process.stdout.on('data', (data) => {
            stdout += data.toString();
        });

        process.stderr.on('data', (data) => {
            stderr += data.toString();
        });

        process.on('close', (code) => {
            if (stderr) {
                outputChannel.appendLine(`stderr: ${stderr}`);
            }
            // TerraTidy returns non-zero when issues found, but still outputs valid JSON
            resolve(stdout);
        });

        process.on('error', (error) => {
            reject(new Error(`Failed to run terratidy: ${error.message}`));
        });
    });
}

// Process TerraTidy JSON output
function processTerraTidyOutput(document: vscode.TextDocument, output: string): void {
    const diagnostics: vscode.Diagnostic[] = [];

    try {
        const result = JSON.parse(output);
        const findings = result.findings || [];

        for (const finding of findings) {
            const diagnostic = createDiagnostic(finding);
            if (diagnostic && finding.file === document.fileName) {
                diagnostics.push(diagnostic);
            }
        }

        const summary = result.summary || {};
        const total = summary.total || 0;
        const errors = summary.errors || 0;
        const warnings = summary.warnings || 0;
        const info = summary.info || 0;

        outputChannel.appendLine(
            `Found ${total} issue(s): ${errors} error(s), ${warnings} warning(s), ${info} info`
        );

        if (total === 0) {
            vscode.window.showInformationMessage('TerraTidy: No issues found');
        } else if (errors > 0) {
            vscode.window.showErrorMessage(`TerraTidy: Found ${errors} error(s)`);
        } else if (warnings > 0) {
            vscode.window.showWarningMessage(`TerraTidy: Found ${warnings} warning(s)`);
        }
    } catch (e) {
        // Not JSON output, might be text output
        outputChannel.appendLine(`Output: ${output}`);
    }

    diagnosticCollection.set(document.uri, diagnostics);
}

// Create a diagnostic from a finding
function createDiagnostic(finding: TerraTidyFinding): vscode.Diagnostic | null {
    if (!finding.location) {
        return null;
    }

    const startLine = Math.max(0, (finding.location.start?.line || 1) - 1);
    const startCol = Math.max(0, (finding.location.start?.column || 1) - 1);
    const endLine = Math.max(0, (finding.location.end?.line || startLine + 1) - 1);
    const endCol = Math.max(0, (finding.location.end?.column || 1) - 1);

    const range = new vscode.Range(startLine, startCol, endLine, endCol);
    const severity = getSeverity(finding.severity);

    const diagnostic = new vscode.Diagnostic(range, finding.message, severity);
    diagnostic.source = 'terratidy';
    diagnostic.code = finding.rule;

    return diagnostic;
}

// Map severity string to VS Code DiagnosticSeverity
function getSeverity(severity: string): vscode.DiagnosticSeverity {
    switch (severity) {
        case 'error':
            return vscode.DiagnosticSeverity.Error;
        case 'warning':
            return vscode.DiagnosticSeverity.Warning;
        case 'info':
            return vscode.DiagnosticSeverity.Information;
        default:
            return vscode.DiagnosticSeverity.Hint;
    }
}

// Initialize TerraTidy configuration
async function initTerraTidy(): Promise<void> {
    const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
    if (!workspaceFolder) {
        vscode.window.showWarningMessage('No workspace folder open');
        return;
    }

    const terraTidyPath = getTerraTidyPath();
    const cwd = workspaceFolder.uri.fsPath;

    try {
        await execTerraTidy(terraTidyPath, ['init'], cwd);
        vscode.window.showInformationMessage('TerraTidy: Configuration initialized');
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        vscode.window.showErrorMessage(`TerraTidy init error: ${errorMessage}`);
    }
}

// Document formatting provider
class TerraTidyFormattingProvider implements vscode.DocumentFormattingEditProvider {
    async provideDocumentFormattingEdits(
        document: vscode.TextDocument,
        _options: vscode.FormattingOptions,
        _token: vscode.CancellationToken
    ): Promise<vscode.TextEdit[]> {
        const terraTidyPath = getTerraTidyPath();
        const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
        const cwd = workspaceFolder?.uri.fsPath || path.dirname(document.fileName);

        try {
            // Run format and get the formatted output
            const args = ['fmt', '--paths', document.fileName];
            await execTerraTidy(terraTidyPath, args, cwd);

            // Read the formatted file content
            // Note: fmt modifies the file in place, so we need to reload
            // For a better experience, we should use stdin/stdout
            return [];
        } catch (error) {
            outputChannel.appendLine(`Format error: ${error}`);
            return [];
        }
    }
}

// Types for TerraTidy findings
interface TerraTidyFinding {
    rule: string;
    message: string;
    file: string;
    severity: string;
    fixable: boolean;
    location?: {
        start?: { line: number; column: number };
        end?: { line: number; column: number };
    };
}
