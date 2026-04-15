# gore-lint VS Code Extension

Index Advisor for Go - Lint your Go code for database index issues directly in VS Code.

## Features

- **Real-time diagnostics** - Automatically scans Go files for index-related issues
- **Problems Panel integration** - Shows issues in VS Code's Problems panel
- **Hover information** - Detailed explanations when hovering over issues
- **Severity filtering** - Configurable severity threshold (info, warn, high, critical)
- **Schema configuration** - Use either live DSN or schema file

## Requirements

- VS Code 1.75.0 or higher
- gore-lint installed and accessible in PATH, or configured path

## Installation

1. Build the extension:
   ```bash
   cd extensions/vscode
   npm install
   npm run compile
   ```

2. Package the extension:
   ```bash
   npx vsce package
   ```

3. Install the .vsix file in VS Code:
   ```bash
   code --install-extension gore-lint-*.vsix
   ```

## Configuration

In VS Code settings (`.vscode/settings.json` or UI):

```json
{
  "goreLint.enable": true,
  "goreLint.path": "gore-lint",
  "goreLint.dsn": "postgres://user:pass@localhost:5432/db?sslmode=disable",
  "goreLint.schemaFile": "./schema.json",
  "goreLint.failOn": "critical"
}
```

### Configuration Options

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `goreLint.enable` | boolean | `true` | Enable/disable the extension |
| `goreLint.path` | string | `gore-lint` | Path to gore-lint executable |
| `goreLint.dsn` | string | `""` | Database DSN for live schema |
| `goreLint.schemaFile` | string | `""` | Path to schema JSON file |
| `goreLint.failOn` | string | `critical` | Minimum severity to show as error |

## Usage

### Commands

- `gore-lint: Run Analysis` - Manually trigger analysis
- `gore-lint: Enable` - Enable gore-lint
- `gore-lint: Disable` - Disable gore-lint

### Diagnostic Display

Issues appear in the Problems panel with severity-based icons:

- 🔵 Info - Informational suggestions
- 🟡 Warning - Warnings (e.g., skipped index columns)
- 🔴 Error - High severity issues
- 🚨 Critical - Critical issues

### Hover Information

Hover over a diagnostic to see:
- Rule ID
- Severity level
- Detailed message
- Reason explanation
- Recommendation

## Example Output

```
IDX-001: 联合索引 users_name_age_idx 的首列 name 未在 WHERE 中出现

Reason: B-Tree 索引遵循最左匹配原则，必须从第一列开始
Recommendation: 考虑在 WHERE 中加入 name 条件或调整索引顺序
```

## License

MIT
