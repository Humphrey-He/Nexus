import * as vscode from 'vscode';
import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

interface GoreLintSuggestion {
  ruleId: string;
  severity: number;
  message: string;
  reason?: string;
  recommendation?: string;
  sourceFile: string;
  lineNumber: number;
}

interface GoreLintReport {
  version: string;
  generatedAt: string;
  target: string;
  suggestions: GoreLintSuggestion[];
  stats: {
    total: number;
    info: number;
    warn: number;
    high: number;
    critical: number;
  };
}

export function activate(context: vscode.ExtensionContext) {
  const config = vscode.workspace.getConfiguration('goreLint');
  const isEnabled = config.get<boolean>('enable', true);

  if (!isEnabled) {
    vscode.window.showInformationMessage('gore-lint is disabled');
    return;
  }

  // Register the run command
  const runCommand = vscode.commands.registerCommand('gore-lint.run', async () => {
    await runGoreLint();
  });

  // Register enable/disable commands
  const enableCommand = vscode.commands.registerCommand('gore-lint.enable', async () => {
    await config.update('enable', true, vscode.ConfigurationTarget.Workspace);
    vscode.window.showInformationMessage('gore-lint has been enabled');
  });

  const disableCommand = vscode.commands.registerCommand('gore-lint.disable', async () => {
    await config.update('enable', false, vscode.ConfigurationTarget.Workspace);
    vscode.window.showInformationMessage('gore-lint has been disabled');
  });

  context.subscriptions.push(runCommand, enableCommand, disableCommand);

  // Run analysis when Go files change
  const fileWatcher = vscode.workspace.createFileSystemWatcher('**/*.go');

  const runDebounced = debounce(async () => {
    await runGoreLint();
  }, 2000);

  fileWatcher.onDidChange(runDebounced);
  fileWatcher.onDidCreate(runDebounced);

  context.subscriptions.push(fileWatcher);

  // Run initial analysis
  runGoreLint().catch(err => {
    console.error('gore-lint initial run failed:', err);
  });
}

async function runGoreLint(): Promise<void> {
  const config = vscode.workspace.getConfiguration('goreLint');
  const goreLintPath = config.get<string>('path', 'gore-lint');
  const dsn = config.get<string>('dsn', '');
  const schemaFile = config.get<string>('schemaFile', '');
  const failOn = config.get<string>('failOn', 'critical');
  const extraArgs = config.get<string>('arguments', '--format json');

  // Build the command
  let args = `check ${extraArgs}`;

  if (dsn) {
    args += ` --dsn "${dsn}"`;
  } else if (schemaFile) {
    args += ` --schema "${schemaFile}"`;
  } else {
    vscode.window.showWarningMessage('gore-lint: No --dsn or --schema configured. Skipping analysis.');
    return;
  }

  args += ` --fail-on ${failOn}`;
  args += ' ./...';

  try {
    const { stdout, stderr } = await execAsync(`${goreLintPath} ${args}`, {
      cwd: vscode.workspace.rootPath,
      maxBuffer: 10 * 1024 * 1024
    });

    if (stderr) {
      console.warn('gore-lint stderr:', stderr);
    }

    const report: GoreLintReport = JSON.parse(stdout);
    displayDiagnostics(report);

  } catch (error: any) {
    if (error.code === 'ENOENT') {
      vscode.window.showErrorMessage(`gore-lint not found at: ${goreLintPath}`);
    } else if (error.code === 1 || error.code === 2 || error.code === 3 || error.code === 4) {
      // Exit codes 1-4 indicate issues found (not a command error)
      // Try to parse partial output if available
      try {
        const report: GoreLintReport = JSON.parse(error.stdout || '{}');
        displayDiagnostics(report);
      } catch {
        vscode.window.showWarningMessage('gore-lint found issues but could not parse output');
      }
    } else {
      vscode.window.showErrorMessage(`gore-lint failed: ${error.message}`);
    }
  }
}

function displayDiagnostics(report: GoreLintReport): void {
  const diagnosticCollection = vscode.languages.createDiagnosticCollection('gore-lint');

  // Clear previous diagnostics
  diagnosticCollection.clear();

  const workspaceRoot = vscode.workspace.rootPath || '';

  for (const suggestion of report.suggestions) {
    const filePath = suggestion.sourceFile.replace(/\\/g, '/');

    // Skip if file path doesn't look valid
    if (!filePath || !filePath.endsWith('.go')) {
      continue;
    }

    const uri = vscode.Uri.file(filePath);
    const severity = severityFromNumber(suggestion.severity);

    const diagnostic = new vscode.Diagnostic(
      new vscode.Range(
        new vscode.Position(suggestion.lineNumber - 1, 0),
        new vscode.Position(suggestion.lineNumber - 1, 1000)
      ),
      `[${suggestion.ruleId}] ${suggestion.message}${suggestion.reason ? `\n\nReason: ${suggestion.reason}` : ''}${suggestion.recommendation ? `\n\nRecommendation: ${suggestion.recommendation}` : ''}`,
      severity
    );

    diagnostic.source = 'gore-lint';
    diagnostic.code = suggestion.ruleId;

    // Add hover information
    const hoverMessage = new vscode.MarkdownString();
    hoverMessage.appendMarkdown(`## ${suggestion.ruleId}\n\n`);
    hoverMessage.appendMarkdown(`**Severity**: ${severityLabel(suggestion.severity)}\n\n`);
    hoverMessage.appendMarkdown(`**Message**: ${suggestion.message}\n\n`);

    if (suggestion.reason) {
      hoverMessage.appendMarkdown(`**Reason**: ${suggestion.reason}\n\n`);
    }

    if (suggestion.recommendation) {
      hoverMessage.appendMarkdown(`**Recommendation**: ${suggestion.recommendation}\n\n`);
    }

    diagnostic.hoverMessage = hoverMessage;

    diagnosticCollection.set(uri, [...(diagnosticCollection.get(uri) || []), diagnostic]);
  }

  // Update status bar
  const stats = report.stats;
  if (stats.total > 0) {
    vscode.window.setStatusBarMessage(
      `gore-lint: ${stats.total} issue${stats.total === 1 ? '' : 's'} (${stats.info} info, ${stats.warn} warn, ${stats.high} high, ${stats.critical} critical)`,
      5000
    );
  } else {
    vscode.window.setStatusBarMessage('gore-lint: No issues found', 3000);
  }
}

function severityFromNumber(severity: number): vscode.DiagnosticSeverity {
  switch (severity) {
    case 0:
      return vscode.DiagnosticSeverity.Information;
    case 1:
      return vscode.DiagnosticSeverity.Warning;
    case 2:
      return vscode.DiagnosticSeverity.Error;
    case 3:
      return vscode.DiagnosticSeverity.Error;
    default:
      return vscode.DiagnosticSeverity.Information;
  }
}

function severityLabel(severity: number): string {
  switch (severity) {
    case 0:
      return 'Info';
    case 1:
      return 'Warning';
    case 2:
      return 'High';
    case 3:
      return 'Critical';
    default:
      return 'Unknown';
  }
}

function debounce<T extends (...args: any[]) => any>(fn: T, ms: number): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null;

  return (...args: Parameters<T>) => {
    if (timeout) {
      clearTimeout(timeout);
    }
    timeout = setTimeout(() => {
      fn(...args);
    }, ms);
  };
}
