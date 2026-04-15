const vscode = require('vscode');
const { exec } = require('child_process');
const path = require('path');

// Severity mapping
const SEVERITY_MAP = {
  0: vscode.DiagnosticSeverity.Information,
  1: vscode.DiagnosticSeverity.Warning,
  2: vscode.DiagnosticSeverity.Error,
  3: vscode.DiagnosticSeverity.Error
};

const SEVERITY_LABELS = ['Info', 'Warning', 'High', 'Critical'];

/**
 * @param {vscode.ExtensionContext} context
 */
function activate(context) {
  const config = vscode.workspace.getConfiguration('goreLint');
  const isEnabled = config.get('enable', true);

  if (!isEnabled) {
    return;
  }

  // Register commands
  const runCommand = vscode.commands.registerCommand('gore-lint.run', async () => {
    await runGoreLint();
  });

  const enableCommand = vscode.commands.registerCommand('gore-lint.enable', async () => {
    await config.update('enable', true, vscode.ConfigurationTarget.Workspace);
    vscode.window.showInformationMessage('gore-lint has been enabled');
  });

  const disableCommand = vscode.commands.registerCommand('gore-lint.disable', async () => {
    await config.update('enable', false, vscode.ConfigurationTarget.Workspace);
    vscode.window.showInformationMessage('gore-lint has been disabled');
  });

  context.subscriptions.push(runCommand, enableCommand, disableCommand);

  // Watch for file changes with debounce
  let debounceTimer = null;
  const fileWatcher = vscode.workspace.createFileSystemWatcher('**/*.go');

  fileWatcher.onDidChange(() => {
    scheduleRun();
  });

  fileWatcher.onDidCreate(() => {
    scheduleRun();
  });

  context.subscriptions.push(fileWatcher);

  // Run initial analysis
  scheduleRun();
}

function scheduleRun() {
  if (debounceTimer) {
    clearTimeout(debounceTimer);
  }
  debounceTimer = setTimeout(() => {
    runGoreLint().catch(err => console.error('gore-lint error:', err));
  }, 2000);
}

async function runGoreLint() {
  const config = vscode.workspace.getConfiguration('goreLint');
  const goreLintPath = config.get('path', 'gore-lint');
  const dsn = config.get('dsn', '');
  const schemaFile = config.get('schemaFile', '');
  const failOn = config.get('failOn', 'critical');

  // Build command
  const args = ['check', '--format', 'json'];

  if (dsn) {
    args.push('--dsn', dsn);
  } else if (schemaFile) {
    args.push('--schema', schemaFile);
  } else {
    vscode.window.showWarningMessage('gore-lint: No --dsn or --schema configured. Skipping analysis.');
    return;
  }

  args.push('--fail-on', failOn);
  args.push('./...');

  const command = `${goreLintPath} ${args.join(' ')}`;

  return new Promise((resolve) => {
    exec(command, {
      cwd: vscode.workspace.rootPath,
      maxBuffer: 10 * 1024 * 1024
    }, (error, stdout, stderr) => {
      if (stderr) {
        console.warn('gore-lint stderr:', stderr);
      }

      // Parse output if available
      if (stdout) {
        try {
          const report = JSON.parse(stdout);
          displayDiagnostics(report);
        } catch (e) {
          console.error('Failed to parse gore-lint output:', e);
        }
      }

      // Handle exit codes
      if (error && error.code >= 2 && error.code <= 4) {
        // Issues found but command ran successfully
        vscode.window.setStatusBarMessage(
          `gore-lint: Analysis complete with issues (exit code: ${error.code})`,
          5000
        );
      }

      resolve();
    });
  });
}

function displayDiagnostics(report) {
  const collection = vscode.languages.createDiagnosticCollection('gore-lint');
  collection.clear();

  if (!report.suggestions || !Array.isArray(report.suggestions)) {
    return;
  }

  for (const suggestion of report.suggestions) {
    if (!suggestion.sourceFile || !suggestion.sourceFile.endsWith('.go')) {
      continue;
    }

    const uri = vscode.Uri.file(suggestion.sourceFile);
    const severity = SEVERITY_MAP[suggestion.severity] || vscode.DiagnosticSeverity.Information;

    const range = new vscode.Range(
      new vscode.Position((suggestion.lineNumber || 1) - 1, 0),
      new vscode.Position((suggestion.lineNumber || 1) - 1, 1000)
    );

    const message = `[${suggestion.ruleId}] ${suggestion.message}`;
    const detailedMessage = `${message}\n\nReason: ${suggestion.reason || 'N/A'}\n\nRecommendation: ${suggestion.recommendation || 'N/A'}`;

    const diagnostic = new vscode.Diagnostic(range, detailedMessage, severity);
    diagnostic.source = 'gore-lint';
    diagnostic.code = suggestion.ruleId;

    // Create hover with markdown
    const hoverContent = new vscode.MarkdownString();
    hoverContent.appendMarkdown(`## ${suggestion.ruleId} - ${SEVERITY_LABELS[suggestion.severity] || 'Unknown'}\n\n`);
    hoverContent.appendMarkdown(`**Message**: ${suggestion.message}\n\n`);

    if (suggestion.reason) {
      hoverContent.appendMarkdown(`**Reason**: ${suggestion.reason}\n\n`);
    }

    if (suggestion.recommendation) {
      hoverContent.appendMarkdown(`**Recommendation**: ${suggestion.recommendation}\n\n`);
    }

    diagnostic.hoverMessage = hoverContent;

    const existing = collection.get(uri) || [];
    existing.push(diagnostic);
    collection.set(uri, existing);
  }

  // Update status bar
  const stats = report.stats || {};
  if (stats.total > 0) {
    const msg = `gore-lint: ${stats.total} issue${stats.total === 1 ? '' : 's'} (${stats.info || 0} info, ${stats.warn || 0} warn, ${stats.high || 0} high, ${stats.critical || 0} critical)`;
    vscode.window.setStatusBarMessage(msg, 5000);
  } else {
    vscode.window.setStatusBarMessage('gore-lint: No issues found', 3000);
  }
}

module.exports = { activate };
