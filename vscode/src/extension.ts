import * as vscode from 'vscode';
import {
    LanguageClient,
    type LanguageClientOptions,
    type ServerOptions,
    TransportKind,
} from 'vscode-languageclient/node';

// Global LSP client instance
let client: LanguageClient | undefined;

// Output channel for logging
let outputChannel: vscode.OutputChannel;

// Extension activation
export async function activate(context: vscode.ExtensionContext) {
    outputChannel = vscode.window.createOutputChannel('TerraTidy');
    context.subscriptions.push(outputChannel);

    // Start the LSP client
    await startLanguageClient(context);

    // Register commands that don't use LSP
    context.subscriptions.push(
        vscode.commands.registerCommand('terratidy.init', initTerraTidy),
        vscode.commands.registerCommand('terratidy.showOutput', () => outputChannel.show()),
        vscode.commands.registerCommand('terratidy.restartServer', async () => {
            await stopLanguageClient();
            await startLanguageClient(context);
        })
    );

    outputChannel.appendLine('TerraTidy extension activated');
}

// Extension deactivation
export async function deactivate() {
    await stopLanguageClient();
}

// Start the Language Server Protocol client
async function startLanguageClient(context: vscode.ExtensionContext): Promise<void> {
    const config = vscode.workspace.getConfiguration('terratidy');
    const executablePath = config.get<string>('executablePath') || 'terratidy';

    // Check if terratidy is available
    try {
        const cp = require('node:child_process');
        cp.execSync(`${executablePath} --version`, { stdio: 'ignore' });
    } catch (error) {
        const message =
            'TerraTidy executable not found. Please install TerraTidy or configure terratidy.executablePath.';
        outputChannel.appendLine(message);
        vscode.window.showErrorMessage(message);
        return;
    }

    // Server options: launch the LSP server
    const serverOptions: ServerOptions = {
        command: executablePath,
        args: ['lsp'],
        transport: TransportKind.stdio,
        options: {
            env: process.env,
        },
    };

    // Client options: configure the LSP client
    const clientOptions: LanguageClientOptions = {
        documentSelector: [
            { scheme: 'file', language: 'terraform' },
            { scheme: 'file', language: 'hcl' },
            { scheme: 'file', pattern: '**/*.tf' },
            { scheme: 'file', pattern: '**/*.tfvars' },
            { scheme: 'file', pattern: '**/*.hcl' },
        ],
        synchronize: {
            // Notify the server about file configuration changes
            fileEvents: vscode.workspace.createFileSystemWatcher('**/.terratidy.{yaml,yml}'),
        },
        outputChannel: outputChannel,
        traceOutputChannel: outputChannel,
        initializationOptions: getInitializationOptions(),
        middleware: {
            workspace: {
                configuration: (params, token, next) => {
                    // Provide configuration to the server
                    return next(params, token);
                },
            },
        },
    };

    // Create and start the language client
    client = new LanguageClient('terratidy', 'TerraTidy Language Server', serverOptions, clientOptions);

    try {
        await client.start();
        outputChannel.appendLine('TerraTidy LSP server started');
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        outputChannel.appendLine(`Failed to start LSP server: ${errorMessage}`);
        vscode.window.showErrorMessage(`TerraTidy LSP server failed to start: ${errorMessage}`);
    }

    // Listen for configuration changes and restart server if needed
    context.subscriptions.push(
        vscode.workspace.onDidChangeConfiguration(async (event) => {
            if (event.affectsConfiguration('terratidy')) {
                const action = await vscode.window.showInformationMessage(
                    'TerraTidy configuration changed. Restart the language server?',
                    'Restart',
                    'Later'
                );
                if (action === 'Restart') {
                    await stopLanguageClient();
                    await startLanguageClient(context);
                }
            }
        })
    );
}

// Stop the language client
async function stopLanguageClient(): Promise<void> {
    if (client) {
        outputChannel.appendLine('Stopping TerraTidy LSP server');
        await client.stop();
        client = undefined;
    }
}

// Get initialization options from configuration
function getInitializationOptions(): Record<string, unknown> {
    const config = vscode.workspace.getConfiguration('terratidy');

    const engines: { [key: string]: boolean } = {
        fmt: config.get<boolean>('engines.fmt', true),
        style: config.get<boolean>('engines.style', true),
        lint: config.get<boolean>('engines.lint', true),
        policy: config.get<boolean>('engines.policy', false),
    };

    return {
        profile: config.get<string>('profile') || undefined,
        configPath: config.get<string>('configPath') || undefined,
        engines: engines,
        severityThreshold: config.get<string>('severityThreshold', 'warning'),
        formatOnSave: config.get<boolean>('formatOnSave', false),
        runOnSave: config.get<boolean>('runOnSave', false),
        fixOnSave: config.get<boolean>('fixOnSave', false),
    };
}

// Initialize TerraTidy configuration (runs 'terratidy init' command)
async function initTerraTidy(): Promise<void> {
    const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
    if (!workspaceFolder) {
        vscode.window.showWarningMessage('No workspace folder open');
        return;
    }

    const config = vscode.workspace.getConfiguration('terratidy');
    const executablePath = config.get<string>('executablePath') || 'terratidy';
    const cwd = workspaceFolder.uri.fsPath;

    try {
        const cp = require('node:child_process');
        await new Promise<void>((resolve, reject) => {
            const process = cp.spawn(executablePath, ['init'], {
                cwd,
                shell: true,
            });

            let stdout = '';
            let stderr = '';

            process.stdout.on('data', (data: Buffer) => {
                stdout += data.toString();
            });

            process.stderr.on('data', (data: Buffer) => {
                stderr += data.toString();
            });

            process.on('close', (code: number) => {
                if (code === 0) {
                    outputChannel.appendLine(`Init output: ${stdout}`);
                    resolve();
                } else {
                    outputChannel.appendLine(`Init failed: ${stderr}`);
                    reject(new Error(stderr || 'Init failed'));
                }
            });

            process.on('error', (error: Error) => {
                reject(error);
            });
        });

        vscode.window.showInformationMessage('TerraTidy: Configuration initialized successfully');
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        vscode.window.showErrorMessage(`TerraTidy init error: ${errorMessage}`);
        outputChannel.appendLine(`Init error: ${errorMessage}`);
    }
}
