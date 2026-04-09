import * as cp from 'node:child_process';
import * as os from 'node:os';
import * as path from 'node:path';
import { promisify } from 'node:util';
import * as vscode from 'vscode';
import { LanguageClient, type LanguageClientOptions, type ServerOptions } from 'vscode-languageclient/node';

// Global LSP client instance
let client: LanguageClient | undefined;

// Output channel for logging
let outputChannel: vscode.OutputChannel;

// Config change listener (tracked to avoid subscription leaks on restart)
let configListener: vscode.Disposable | undefined;

// Config file watcher (tracked to avoid subscription leaks on restart)
let configWatcher: vscode.FileSystemWatcher | undefined;

// Resolve ~ prefix to the user's home directory
function resolveExecutablePath(execPath: string): string {
  if (execPath.startsWith('~/') || execPath === '~') {
    return path.join(os.homedir(), execPath.slice(1));
  }
  return execPath;
}

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
export async function deactivate(): Promise<void> {
  configListener?.dispose();
  configWatcher?.dispose();
  await stopLanguageClient();
}

// Start the Language Server Protocol client
async function startLanguageClient(context: vscode.ExtensionContext): Promise<void> {
  const config = vscode.workspace.getConfiguration('terratidy');
  const rawPath = config.get<string>('executablePath') || 'terratidy';
  const executablePath = resolveExecutablePath(rawPath);

  outputChannel.appendLine(`Config executablePath: "${rawPath}"`);
  outputChannel.appendLine(`Resolved executablePath: "${executablePath}"`);

  // Check if terratidy is available (async to avoid blocking event loop)
  const execFile = promisify(cp.execFile);
  try {
    await execFile(executablePath, ['version']);
  } catch {
    const message =
      'TerraTidy executable not found. Please install TerraTidy or configure terratidy.executablePath. ' +
      'See https://github.com/santosr2/TerraTidy#installation for installation instructions.';
    outputChannel.appendLine(message);
    vscode.window.showErrorMessage(message);
    return;
  }

  // Server options: launch the LSP server
  // Note: Don't use TransportKind.stdio as it adds --stdio flag.
  // Our LSP server uses stdio by default without any flags.
  const serverOptions: ServerOptions = {
    command: executablePath,
    args: ['lsp'],
    options: {
      env: process.env,
    },
  };

  // Dispose any previous config watcher before creating a new one (prevents
  // duplicate watchers after multiple server restarts)
  if (configWatcher) {
    configWatcher.dispose();
  }
  configWatcher = vscode.workspace.createFileSystemWatcher('**/.terratidy.{yaml,yml}');

  // Client options: configure the LSP client
  const clientOptions: LanguageClientOptions = {
    // Use pattern-based selectors only. Language selectors are removed to avoid
    // conflicts with HashiCorp Terraform and other HCL syntax highlighting extensions.
    documentSelector: [
      { scheme: 'file', pattern: '**/*.tf' },
      { scheme: 'file', pattern: '**/*.tfvars' },
      { scheme: 'file', pattern: '**/*.hcl' },
    ],
    synchronize: {
      // Notify the server about file configuration changes
      fileEvents: configWatcher,
    },
    outputChannel: outputChannel,
    traceOutputChannel: outputChannel,
    initializationOptions: getInitializationOptions(),
  };

  // Create and start the language client
  client = new LanguageClient('terratidy', 'TerraTidy Language Server', serverOptions, clientOptions);

  try {
    await client.start();
    outputChannel.appendLine('TerraTidy LSP server started');
    // Note: configWatcher is managed manually via module-level variable,
    // not pushed to subscriptions to avoid duplicate disposal on restart
  } catch (error) {
    // Dispose the watcher since we won't be using it
    configWatcher?.dispose();
    configWatcher = undefined;
    const errorMessage = error instanceof Error ? error.message : String(error);
    outputChannel.appendLine(`Failed to start LSP server: ${errorMessage}`);
    vscode.window.showErrorMessage(`TerraTidy LSP server failed to start: ${errorMessage}`);
  }

  // Dispose old config listener before creating a new one (prevents
  // duplicate "Restart?" prompts after multiple server restarts)
  if (configListener) {
    configListener.dispose();
  }
  configListener = vscode.workspace.onDidChangeConfiguration(async (event) => {
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
  });
  // Note: configListener is managed manually via module-level variable,
  // not pushed to subscriptions to avoid duplicate disposal on restart
}

// Stop the language client
async function stopLanguageClient(): Promise<void> {
  // Dispose config watcher alongside client
  if (configWatcher) {
    configWatcher.dispose();
    configWatcher = undefined;
  }
  if (client) {
    outputChannel.appendLine('Stopping TerraTidy LSP server');
    try {
      await client.stop();
    } catch {
      // Server may exit before client finishes cleanup - this is normal
    }
    client = undefined;
  }
}

// Get initialization options from configuration
export function getInitializationOptions(): Record<string, unknown> {
  const config = vscode.workspace.getConfiguration('terratidy');

  const engines: { [key: string]: boolean } = {
    fmt: config.get<boolean>('engines.fmt', true),
    style: config.get<boolean>('engines.style', true),
    lint: config.get<boolean>('engines.lint', true),
    policy: config.get<boolean>('engines.policy', false),
  };

  // Only send severityThreshold if explicitly set by user (not default)
  // This allows .terratidy.yaml to control the threshold
  const thresholdInspect = config.inspect<string>('severityThreshold');
  const severityThreshold =
    thresholdInspect?.workspaceValue ??
    thresholdInspect?.globalValue ??
    thresholdInspect?.workspaceFolderValue ??
    undefined;

  return {
    profile: config.get<string>('profile') || undefined,
    configPath: config.get<string>('configPath') || undefined,
    engines: engines,
    severityThreshold: severityThreshold,
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
  const executablePath = resolveExecutablePath(config.get<string>('executablePath') || 'terratidy');
  const cwd = workspaceFolder.uri.fsPath;

  try {
    await new Promise<void>((resolve, reject) => {
      const child = cp.spawn(executablePath, ['init'], {
        cwd,
      });

      let stdout = '';
      let stderr = '';

      child.stdout.on('data', (data: Buffer) => {
        stdout += data.toString();
      });

      child.stderr.on('data', (data: Buffer) => {
        stderr += data.toString();
      });

      child.on('close', (code: number | null) => {
        if (code === 0) {
          outputChannel.appendLine(`Init output: ${stdout}`);
          resolve();
        } else {
          outputChannel.appendLine(`Init failed: ${stderr}`);
          reject(new Error(stderr || `Init failed with code ${code ?? 'signal'}`));
        }
      });

      child.on('error', (error: Error) => {
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
