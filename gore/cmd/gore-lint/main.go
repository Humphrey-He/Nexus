package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"gore/dialect/mysql"
	"gore/dialect/postgres"
	"gore/internal/advisor"
	"gore/internal/advisor/rules"
	"gore/internal/migrate"
	"golang.org/x/tools/go/packages"
)

type config struct {
	dsn       string
	schema    string
	dbType    string
	target    string
	useStdout bool
	format    string
	output    string
	crossFile bool
}

type report struct {
	Version     string               `json:"version"`
	GeneratedAt string               `json:"generatedAt"`
	Target      string               `json:"target"`
	Suggestions []advisor.Suggestion `json:"suggestions"`
	Stats       reportStats          `json:"stats"`
}

type reportStats struct {
	Total    int `json:"total"`
	Info     int `json:"info"`
	Warn     int `json:"warn"`
	High     int `json:"high"`
	Critical int `json:"critical"`
}

type schemaCache struct {
	Tables []advisor.TableSchema `json:"tables"`
}

type literalValue struct {
	value any
	kind  string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "gore-lint: %s\n", err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing command: expected 'check'")
	}

	cmd := args[0]
	switch cmd {
	case "check":
		return runCheck(args[1:])
	case "migrate":
		return runMigrate(args[1:])
	case "-h", "--help", "help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cfg := &config{}
	fs.StringVar(&cfg.dsn, "dsn", "", "database DSN for live schema")
	fs.StringVar(&cfg.schema, "schema", "", "path to schema cache JSON")
	fs.StringVar(&cfg.dbType, "db-type", "postgres", "database type: postgres or mysql")
	fs.BoolVar(&cfg.useStdout, "stdout", true, "write diagnostics to stdout")
	fs.StringVar(&cfg.format, "format", "json", "output format: json, text, html, or sarif")
	fs.StringVar(&cfg.output, "output", "", "output file path (default: stdout)")
	fs.BoolVar(&cfg.crossFile, "cross-file", false, "enable cross-file constant resolution using go/packages")

	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("missing target path: e.g. gore-lint check ./...")
	}
	cfg.target = rest[0]

	if cfg.dsn == "" && cfg.schema == "" {
		return fmt.Errorf("either --dsn or --schema must be provided")
	}

	if cfg.dsn != "" && cfg.schema != "" {
		return fmt.Errorf("--dsn and --schema are mutually exclusive")
	}

	// Validate db-type
	if cfg.dbType != "postgres" && cfg.dbType != "mysql" {
		return fmt.Errorf("invalid --db-type: must be 'postgres' or 'mysql', got %q", cfg.dbType)
	}

	// Validate format
	validFormats := map[string]bool{"json": true, "text": true, "html": true, "sarif": true}
	if !validFormats[cfg.format] {
		return fmt.Errorf("invalid --format: must be 'json', 'text', 'html', or 'sarif', got %q", cfg.format)
	}

	// For non-stdout formats, disable stdout flag
	if cfg.output != "" {
		cfg.useStdout = false
	}

	schemasByTable := map[string]advisor.TableSchema{}
	if cfg.schema != "" {
		loaded, err := loadSchemaCache(cfg.schema)
		if err != nil {
			return err
		}
		for _, table := range loaded.Tables {
			schemasByTable[table.TableName] = table
		}
	}

	if cfg.dsn != "" {
		liveSchemas, err := loadLiveSchema(cfg.dsn, cfg.dbType)
		if err != nil {
			return err
		}
		schemasByTable = liveSchemas
	}

	var queries []*advisor.QueryMetadata
	var err error

	if cfg.crossFile {
		queries, err = extractQueriesWithPackages(cfg.target)
	} else {
		queries, err = extractQueries(cfg.target)
	}
	if err != nil {
		return err
	}

	engine := advisor.NewEngine(
		rules.NewLeftmostMatchRule(),
		rules.NewFunctionIndexRule(),
		rules.NewTypeMismatchRule(),
		rules.NewLikePrefixRule(),
		rules.NewOrderByIndexRule(),
		rules.NewJoinIndexRule(),
	)

	// Add MySQL-specific rules
	if cfg.dbType == "mysql" {
		engine = advisor.NewEngine(
			rules.NewLeftmostMatchRule(),
			rules.NewFunctionIndexRule(),
			rules.NewTypeMismatchRule(),
			rules.NewLikePrefixRule(),
			rules.NewOrderByIndexRule(),
			rules.NewJoinIndexRule(),
			rules.NewInvisibleIndexRule(),
			rules.NewDescendingIndexRule(),
			rules.NewHashIndexRangeRule(),
		)
	}

	var suggestions []advisor.Suggestion
	for _, query := range queries {
		schema, ok := schemasByTable[query.TableName]
		if !ok {
			continue
		}
		suggestions = append(suggestions, engine.Analyze(query, &schema)...)
	}

	rep := report{
		Version:     "0.1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Target:      cfg.target,
		Suggestions: suggestions,
		Stats:       buildStats(suggestions),
	}

	return writeReport(rep, cfg)
}

func buildStats(suggestions []advisor.Suggestion) reportStats {
	stats := reportStats{}
	stats.Total = len(suggestions)
	for _, s := range suggestions {
		switch s.Severity {
		case advisor.SeverityInfo:
			stats.Info++
		case advisor.SeverityWarn:
			stats.Warn++
		case advisor.SeverityHigh:
			stats.High++
		case advisor.SeverityCritical:
			stats.Critical++
		}
	}
	return stats
}

func severityLabel(s advisor.Severity) string {
	switch s {
	case advisor.SeverityInfo:
		return "Info"
	case advisor.SeverityWarn:
		return "Warning"
	case advisor.SeverityHigh:
		return "High"
	case advisor.SeverityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

func severityColor(s advisor.Severity) string {
	switch s {
	case advisor.SeverityInfo:
		return "#17a2b8"
	case advisor.SeverityWarn:
		return "#ffc107"
	case advisor.SeverityHigh:
		return "#fd7e14"
	case advisor.SeverityCritical:
		return "#dc3545"
	default:
		return "#6c757d"
	}
}

func writeReport(rep report, cfg *config) error {
	var out *os.File
	var err error

	if cfg.output != "" {
		out, err = os.Create(cfg.output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer out.Close()
	} else if cfg.useStdout {
		out = os.Stdout
	} else {
		out = os.Stderr
	}

	switch cfg.format {
	case "json":
		return writeJSONReport(rep, out)
	case "text":
		return writeTextReport(rep, out)
	case "html":
		return writeHTMLReport(rep, out)
	case "sarif":
		return writeSARIFReport(rep, out)
	}
	return nil
}

func writeJSONReport(rep report, out *os.File) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	_, err = out.Write(append(data, '\n'))
	return err
}

func writeTextReport(rep report, out *os.File) error {
	var buf strings.Builder
	var err error

	buf.WriteString("=== Gore-Lint Report ===\n")
	buf.WriteString(fmt.Sprintf("Generated: %s\n", rep.GeneratedAt))
	buf.WriteString(fmt.Sprintf("Target: %s\n", rep.Target))
	buf.WriteString(fmt.Sprintf("Total Issues: %d\n\n", rep.Stats.Total))

	if rep.Stats.Critical > 0 {
		buf.WriteString(fmt.Sprintf("  Critical: %d\n", rep.Stats.Critical))
	}
	if rep.Stats.High > 0 {
		buf.WriteString(fmt.Sprintf("  High: %d\n", rep.Stats.High))
	}
	if rep.Stats.Warn > 0 {
		buf.WriteString(fmt.Sprintf("  Warning: %d\n", rep.Stats.Warn))
	}
	if rep.Stats.Info > 0 {
		buf.WriteString(fmt.Sprintf("  Info: %d\n", rep.Stats.Info))
	}

	buf.WriteString("\n--- Suggestions ---\n\n")

	for _, s := range rep.Suggestions {
		buf.WriteString(fmt.Sprintf("[%s] %s\n", severityLabel(s.Severity), s.RuleID))
		buf.WriteString(fmt.Sprintf("  Message: %s\n", s.Message))
		if s.SourceFile != "" {
			buf.WriteString(fmt.Sprintf("  Location: %s:%d\n", s.SourceFile, s.LineNumber))
		}
		if s.Recommendation != "" {
			buf.WriteString(fmt.Sprintf("  Recommendation: %s\n", s.Recommendation))
		}
		buf.WriteString("\n")
	}

	_, err = out.WriteString(buf.String())
	return err
}

func writeHTMLReport(rep report, out *os.File) error {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gore-Lint Report</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif; background: #f5f5f5; color: #333; line-height: 1.6; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 8px; margin-bottom: 20px; }
        .header h1 { font-size: 28px; margin-bottom: 10px; }
        .header .meta { opacity: 0.9; font-size: 14px; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 15px; margin-bottom: 20px; }
        .stat-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); text-align: center; }
        .stat-card .number { font-size: 36px; font-weight: bold; }
        .stat-card .label { font-size: 14px; color: #666; text-transform: uppercase; letter-spacing: 1px; }
        .stat-card.total { border-top: 4px solid #667eea; }
        .stat-card.critical .number { color: #dc3545; }
        .stat-card.critical { border-top: 4px solid #dc3545; }
        .stat-card.high .number { color: #fd7e14; }
        .stat-card.high { border-top: 4px solid #fd7e14; }
        .stat-card.warn .number { color: #ffc107; }
        .stat-card.warn { border-top: 4px solid #ffc107; }
        .stat-card.info .number { color: #17a2b8; }
        .stat-card.info { border-top: 4px solid #17a2b8; }
        .suggestions { background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
        .suggestions h2 { padding: 20px; border-bottom: 1px solid #eee; font-size: 18px; }
        .suggestion { padding: 20px; border-bottom: 1px solid #eee; transition: background 0.2s; }
        .suggestion:hover { background: #f8f9fa; }
        .suggestion:last-child { border-bottom: none; }
        .suggestion-header { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
        .badge { padding: 4px 10px; border-radius: 20px; font-size: 12px; font-weight: bold; color: white; text-transform: uppercase; }
        .rule-id { font-family: 'Monaco', 'Menlo', monospace; font-size: 13px; color: #666; }
        .suggestion-message { font-size: 16px; margin-bottom: 10px; }
        .suggestion-meta { display: flex; gap: 20px; font-size: 13px; color: #666; flex-wrap: wrap; }
        .suggestion-meta span { display: flex; align-items: center; gap: 5px; }
        .recommendation { background: #e8f4fd; border-left: 4px solid #17a2b8; padding: 10px 15px; margin-top: 10px; border-radius: 0 4px 4px 0; font-size: 14px; }
        .reason { color: #666; font-size: 14px; margin-top: 8px; }
        .empty-state { text-align: center; padding: 60px 20px; color: #666; }
        .empty-state .icon { font-size: 48px; margin-bottom: 20px; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Gore-Lint Report</h1>
            <div class="meta">
                <div>Generated: ` + rep.GeneratedAt + `</div>
                <div>Target: ` + htmlEscape(rep.Target) + `</div>
            </div>
        </div>

        <div class="stats">
            <div class="stat-card total">
                <div class="number">` + fmt.Sprintf("%d", rep.Stats.Total) + `</div>
                <div class="label">Total Issues</div>
            </div>
            <div class="stat-card critical">
                <div class="number">` + fmt.Sprintf("%d", rep.Stats.Critical) + `</div>
                <div class="label">Critical</div>
            </div>
            <div class="stat-card high">
                <div class="number">` + fmt.Sprintf("%d", rep.Stats.High) + `</div>
                <div class="label">High</div>
            </div>
            <div class="stat-card warn">
                <div class="number">` + fmt.Sprintf("%d", rep.Stats.Warn) + `</div>
                <div class="label">Warning</div>
            </div>
            <div class="stat-card info">
                <div class="number">` + fmt.Sprintf("%d", rep.Stats.Info) + `</div>
                <div class="label">Info</div>
            </div>
        </div>

        <div class="suggestions">
            <h2>Suggestions</h2>`
	if len(rep.Suggestions) == 0 {
		html += `
            <div class="empty-state">
                <div class="icon">&#10003;</div>
                <div>No issues found</div>
            </div>`
	} else {
		for _, s := range rep.Suggestions {
			html += `
            <div class="suggestion">
                <div class="suggestion-header">
                    <span class="badge" style="background-color: ` + severityColor(s.Severity) + `">` + severityLabel(s.Severity) + `</span>
                    <span class="rule-id">` + htmlEscape(s.RuleID) + `</span>
                </div>
                <div class="suggestion-message">` + htmlEscape(s.Message) + `</div>
                <div class="suggestion-meta">`
			if s.SourceFile != "" {
				html += `<span>&#128193; ` + htmlEscape(s.SourceFile) + `:` + fmt.Sprintf("%d", s.LineNumber) + `</span>`
			}
			if len(s.Evidence) > 0 {
				html += `<span>&#128203; ` + htmlEscape(strings.Join(s.Evidence, ", ")) + `</span>`
			}
			html += `
                </div>`
			if s.Recommendation != "" {
				html += `
                <div class="recommendation">&#128161; ` + htmlEscape(s.Recommendation) + `</div>`
			}
			if s.Reason != "" {
				html += `
                <div class="reason">` + htmlEscape(s.Reason) + `</div>`
			}
			html += `
            </div>`
		}
	}
	html += `
        </div>

        <div class="footer">
            Generated by Gore-Lint v` + rep.Version + `
        </div>
    </div>
</body>
</html>`

	_, err := out.WriteString(html)
	return err
}

func htmlEscape(s string) string {
	var buf strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&#39;")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

func writeSARIFReport(rep report, out *os.File) error {
	sarif := map[string]any{
		"$schema": "https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.json",
		"version": "2.1.0",
		"runs": []map[string]any{
			{
				"tool": map[string]any{
					"driver": map[string]any{
						"name":           "gore-lint",
						"version":        rep.Version,
						"informationUri": "https://github.com/your-org/gore",
					},
				},
				"results": buildSARIFResults(rep.Suggestions),
			},
		},
	}

	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return err
	}
	_, err = out.Write(append(data, '\n'))
	return err
}

func buildSARIFResults(suggestions []advisor.Suggestion) []map[string]any {
	var results []map[string]any
	for _, s := range suggestions {
		level := "note"
		switch s.Severity {
		case advisor.SeverityCritical, advisor.SeverityHigh:
			level = "error"
		case advisor.SeverityWarn:
			level = "warning"
		}

		result := map[string]any{
			"ruleId":    s.RuleID,
			"level":     level,
			"message":   map[string]any{"text": s.Message},
			"locations": buildSARIFLocations(s),
		}
		results = append(results, result)
	}
	return results
}

func buildSARIFLocations(s advisor.Suggestion) []map[string]any {
	if s.SourceFile == "" {
		return []map[string]any{
			{
				"physicalLocation": map[string]any{
					"artifactLocation": map[string]any{"uri": "unknown"},
				},
			},
		}
	}
	return []map[string]any{
		{
			"physicalLocation": map[string]any{
				"artifactLocation": map[string]any{"uri": s.SourceFile},
				"region":          map[string]any{"startLine": s.LineNumber},
			},
		},
	}
}

func extractQueries(target string) ([]*advisor.QueryMetadata, error) {
	paths, err := collectGoFiles(target)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	seen := map[string]struct{}{}
	var out []*advisor.QueryMetadata

	for _, path := range paths {
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}

		consts := collectFileLiterals(file)
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			meta, key := buildQueryMetadata(call, fset, path, consts)
			if meta == nil || key == "" {
				return true
			}
			if _, exists := seen[key]; exists {
				return true
			}

			seen[key] = struct{}{}
			out = append(out, meta)
			return true
		})
	}

	return out, nil
}

// extractQueriesWithPackages uses go/packages to load the entire package
// and resolve constants across files.
func extractQueriesWithPackages(target string) ([]*advisor.QueryMetadata, error) {
	cfg := &packages.Config{
		Fset: token.NewFileSet(),
		Mode: packages.NeedSyntax | packages.NeedTypesInfo,
	}

	// Determine the package path from target
	// Handle both package paths (e.g., "./...") and directory paths
	pkgPath := target
	if strings.HasSuffix(target, "/...") || strings.HasSuffix(target, "\\...") {
		pkgPath = strings.TrimSuffix(target, "/...")
		pkgPath = strings.TrimSuffix(pkgPath, "\\...")
	}

	// If target looks like a path, convert to package pattern
	if !strings.Contains(pkgPath, ".") {
		pkgPath = "./" + pkgPath
	}
	if !strings.HasSuffix(pkgPath, "/...") && !strings.HasSuffix(pkgPath, "\\...") {
		pkgPath = pkgPath + "/..."
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	// Collect all constants from all packages
	allConsts := make(map[string]literalValue)

	for _, pkg := range pkgs {
		for _, syntax := range pkg.Syntax {
			consts := collectFileLiteralsWithTypes(syntax, pkg.TypesInfo, allConsts)
			for k, v := range consts {
				allConsts[k] = v
			}
		}
	}

	// Now extract queries using the collected constants
	fset := cfg.Fset
	seen := map[string]struct{}{}
	var out []*advisor.QueryMetadata

	for _, pkg := range pkgs {
		for i, syntax := range pkg.Syntax {
			filePath := pkg.Syntax[i].Pos()
			path := fset.Position(filePath).Filename

			ast.Inspect(syntax, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				meta, key := buildQueryMetadata(call, fset, path, allConsts)
				if meta == nil || key == "" {
					return true
				}
				if _, exists := seen[key]; exists {
					return true
				}

				seen[key] = struct{}{}
				out = append(out, meta)
				return true
			})
		}
	}

	return out, nil
}

// collectFileLiteralsWithTypes collects literal values from a file
// using type information to resolve constants.
func collectFileLiteralsWithTypes(file *ast.File, info *types.Info, globalConsts map[string]literalValue) map[string]literalValue {
	values := make(map[string]literalValue)

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR) {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				lit, ok := evalLiteral(valueSpec.Values[i], values)
				if !ok {
					// Try to get from type info
					if ident, ok := valueSpec.Values[i].(*ast.Ident); ok {
						if constObj := info.ObjectOf(ident); constObj != nil {
							if c := constObj.(*types.Const); c != nil {
								lit = typeConstToLiteral(c)
								ok = lit.value != nil
							}
						}
					}
				}
				if ok {
					values[name.Name] = lit
				}
			}
		}
	}
	return values
}

// typeConstToLiteral converts a types.Const to literalValue.
func typeConstToLiteral(c *types.Const) literalValue {
	switch constant.BoolVal(c.Val()) {
	case true, false:
		return literalValue{value: constant.BoolVal(c.Val()), kind: "bool"}
	}
	switch c.Val().Kind() {
	case constant.Int:
		if v, ok := constant.Int64Val(c.Val()); ok {
			return literalValue{value: v, kind: "int"}
		}
	case constant.String:
		return literalValue{value: constant.StringVal(c.Val()), kind: "string"}
	case constant.Float:
		if v, ok := constant.Float64Val(c.Val()); ok {
			return literalValue{value: v, kind: "float"}
		}
	}
	return literalValue{}
}

func collectFileLiterals(file *ast.File) map[string]literalValue {
	values := map[string]literalValue{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR) {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				lit, ok := evalLiteral(valueSpec.Values[i], values)
				if !ok {
					continue
				}
				values[name.Name] = lit
			}
		}
	}
	return values
}

func buildQueryMetadata(call *ast.CallExpr, fset *token.FileSet, path string, consts map[string]literalValue) (*advisor.QueryMetadata, string) {
	chain := collectCallChain(call)
	if len(chain) == 0 {
		return nil, ""
	}

	var queryPos token.Pos
	meta := &advisor.QueryMetadata{}
	for i := len(chain) - 1; i >= 0; i-- {
		item := chain[i]
		switch item.name {
		case "Query":
			queryPos = item.pos
		case "From":
			if v, ok := evalString(item.args, 0, consts); ok {
				meta.TableName = v
			}
		case "WhereField":
			cond, ok := parseWhereField(item.args, consts)
			if ok {
				meta.Conditions = append(meta.Conditions, cond)
			}
		case "WhereIn":
			cond, ok := parseWhereIn(item.args, consts)
			if ok {
				meta.Conditions = append(meta.Conditions, cond)
			}
		case "WhereLike":
			cond, ok := parseWhereLike(item.args, consts)
			if ok {
				meta.Conditions = append(meta.Conditions, cond)
			}
		case "OrderBy":
			if v, ok := evalString(item.args, 0, consts); ok {
				field, dir := parseOrderBy(v)
				if field != "" {
					meta.OrderBy = append(meta.OrderBy, advisor.OrderField{Field: field, Direction: dir})
				}
			}
		case "Limit":
			if v, ok := evalInt(item.args, 0, consts); ok {
				meta.Limit = &v
			}
		case "Offset":
			if v, ok := evalInt(item.args, 0, consts); ok {
				meta.Offset = &v
			}
		}
	}

	if meta.TableName == "" || queryPos == token.NoPos {
		return nil, ""
	}

	pos := fset.Position(queryPos)
	meta.SourceFile = path
	meta.LineNumber = pos.Line

	key := fmt.Sprintf("%s:%d:%d", path, pos.Line, pos.Column)
	return meta, key
}

type callItem struct {
	name string
	args []ast.Expr
	pos  token.Pos
}

func collectCallChain(call *ast.CallExpr) []callItem {
	var chain []callItem
	cur := call
	for {
		sel, ok := cur.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}
		chain = append(chain, callItem{name: sel.Sel.Name, args: cur.Args, pos: cur.Lparen})
		inner, ok := sel.X.(*ast.CallExpr)
		if !ok {
			break
		}
		cur = inner
	}
	return chain
}

func parseWhereField(args []ast.Expr, consts map[string]literalValue) (advisor.Condition, bool) {
	if len(args) < 3 {
		return advisor.Condition{}, false
	}
	field, ok := evalString(args, 0, consts)
	if !ok {
		return advisor.Condition{}, false
	}
	op, ok := evalString(args, 1, consts)
	if !ok {
		return advisor.Condition{}, false
	}
	value, valueType, ok := evalAny(args[2], consts)
	if !ok {
		return advisor.Condition{}, false
	}

	return advisor.Condition{
		Field:     field,
		Operator:  op,
		Value:     value,
		ValueType: valueType,
	}, true
}
func parseWhereIn(args []ast.Expr, consts map[string]literalValue) (advisor.Condition, bool) {
	if len(args) < 2 {
		return advisor.Condition{}, false
	}
	field, ok := evalString(args, 0, consts)
	if !ok {
		return advisor.Condition{}, false
	}
	value, valueType, ok := evalAny(args[1], consts)
	if !ok {
		return advisor.Condition{}, false
	}

	return advisor.Condition{
		Field:     field,
		Operator:  "IN",
		Value:     value,
		ValueType: valueType,
	}, true
}

func parseWhereLike(args []ast.Expr, consts map[string]literalValue) (advisor.Condition, bool) {
	if len(args) < 2 {
		return advisor.Condition{}, false
	}
	field, ok := evalString(args, 0, consts)
	if !ok {
		return advisor.Condition{}, false
	}
	value, valueType, ok := evalAny(args[1], consts)
	if !ok {
		return advisor.Condition{}, false
	}

	return advisor.Condition{
		Field:     field,
		Operator:  "LIKE",
		Value:     value,
		ValueType: valueType,
	}, true
}

func evalString(args []ast.Expr, idx int, consts map[string]literalValue) (string, bool) {
	if idx >= len(args) {
		return "", false
	}
	lit, ok := evalLiteral(args[idx], consts)
	if !ok || lit.kind != "string" {
		return "", false
	}
	v, ok := lit.value.(string)
	return v, ok
}

func evalInt(args []ast.Expr, idx int, consts map[string]literalValue) (int, bool) {
	if idx >= len(args) {
		return 0, false
	}
	lit, ok := evalLiteral(args[idx], consts)
	if !ok {
		return 0, false
	}

	switch v := lit.value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func evalAny(expr ast.Expr, consts map[string]literalValue) (any, string, bool) {
	lit, ok := evalLiteral(expr, consts)
	if ok {
		return lit.value, lit.kind, true
	}
	return nil, "", false
}

func evalLiteral(expr ast.Expr, consts map[string]literalValue) (literalValue, bool) {
	switch v := expr.(type) {
	case *ast.BasicLit:
		switch v.Kind {
		case token.STRING:
			parsed, err := strconv.Unquote(v.Value)
			if err != nil {
				return literalValue{}, false
			}
			return literalValue{value: parsed, kind: "string"}, true
		case token.INT:
			parsed, err := strconv.ParseInt(v.Value, 10, 64)
			if err != nil {
				return literalValue{}, false
			}
			return literalValue{value: parsed, kind: "int"}, true
		case token.FLOAT:
			parsed, err := strconv.ParseFloat(v.Value, 64)
			if err != nil {
				return literalValue{}, false
			}
			return literalValue{value: parsed, kind: "float"}, true
		}
	case *ast.Ident:
		switch v.Name {
		case "true":
			return literalValue{value: true, kind: "bool"}, true
		case "false":
			return literalValue{value: false, kind: "bool"}, true
		}
		if lit, ok := consts[v.Name]; ok {
			return lit, true
		}
	case *ast.UnaryExpr:
		if v.Op == token.SUB {
			if lit, ok := evalLiteral(v.X, consts); ok {
				switch val := lit.value.(type) {
				case int64:
					return literalValue{value: -val, kind: lit.kind}, true
				case float64:
					return literalValue{value: -val, kind: lit.kind}, true
				}
			}
		}
	case *ast.ParenExpr:
		return evalLiteral(v.X, consts)
	case *ast.BinaryExpr:
		return evalBinaryLiteral(v, consts)
	case *ast.CallExpr:
		if lit, ok := evalWrapperCall(v, consts); ok {
			return lit, true
		}
	}
	return literalValue{}, false
}

func evalBinaryLiteral(expr *ast.BinaryExpr, consts map[string]literalValue) (literalValue, bool) {
	if expr.Op != token.ADD {
		return literalValue{}, false
	}
	left, ok := evalLiteral(expr.X, consts)
	if !ok {
		return literalValue{}, false
	}
	right, ok := evalLiteral(expr.Y, consts)
	if !ok {
		return literalValue{}, false
	}

	if left.kind == "string" && right.kind == "string" {
		return literalValue{value: left.value.(string) + right.value.(string), kind: "string"}, true
	}
	if left.kind == "int" && right.kind == "int" {
		return literalValue{value: left.value.(int64) + right.value.(int64), kind: "int"}, true
	}
	if left.kind == "float" && right.kind == "float" {
		return literalValue{value: left.value.(float64) + right.value.(float64), kind: "float"}, true
	}
	return literalValue{}, false
}

func evalWrapperCall(call *ast.CallExpr, consts map[string]literalValue) (literalValue, bool) {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return literalValue{}, false
	}
	switch ident.Name {
	case "int", "int64", "float64", "string", "bool":
		if len(call.Args) != 1 {
			return literalValue{}, false
		}
		arg, ok := evalLiteral(call.Args[0], consts)
		if !ok {
			return literalValue{}, false
		}
		return literalValue{value: arg.value, kind: ident.Name}, true
	case "ptr":
		// unwrap simple ptr(x) wrappers
		if len(call.Args) != 1 {
			return literalValue{}, false
		}
		return evalLiteral(call.Args[0], consts)
	}
	return literalValue{}, false
}

func parseOrderBy(expr string) (string, string) {
	parts := strings.Fields(expr)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], "ASC"
	}
	return parts[0], strings.ToUpper(parts[1])
}

func collectGoFiles(target string) ([]string, error) {
	if strings.HasSuffix(target, "/...") || strings.HasSuffix(target, "\\...") {
		root := strings.TrimSuffix(target, "/...")
		root = strings.TrimSuffix(root, "\\...")
		return walkGoFiles(root)
	}

	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return walkGoFiles(target)
	}

	if strings.HasSuffix(target, ".go") {
		return []string{target}, nil
	}

	return nil, fmt.Errorf("unsupported target: %s", target)
}

func walkGoFiles(root string) ([]string, error) {
	var out []string
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(name, ".go") {
			out = append(out, path)
		}
		return nil
	}

	if err := filepath.WalkDir(root, walkFn); err != nil {
		return nil, err
	}
	return out, nil
}

func loadSchemaCache(path string) (schemaCache, error) {
	if path == "" {
		return schemaCache{}, fmt.Errorf("schema path is empty")
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return schemaCache{}, err
	}

	var cache schemaCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return schemaCache{}, err
	}

	return cache, nil
}

func loadLiveSchema(dsn, dbType string) (map[string]advisor.TableSchema, error) {
	db, err := sql.Open(dbType, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all tables
	var tables []string
	if dbType == "mysql" {
		rows, err := db.QueryContext(ctx, "SHOW TABLES")
		if err != nil {
			return nil, fmt.Errorf("failed to query tables: %w", err)
		}
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				rows.Close()
				return nil, err
			}
			tables = append(tables, table)
		}
		rows.Close()
	} else {
		rows, err := db.QueryContext(ctx, "SELECT tablename FROM pg_tables WHERE schemaname = 'public'")
		if err != nil {
			return nil, fmt.Errorf("failed to query tables: %w", err)
		}
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				rows.Close()
				return nil, err
			}
			tables = append(tables, table)
		}
		rows.Close()
	}

	schemasByTable := make(map[string]advisor.TableSchema)

	for _, table := range tables {
		var indexes []advisor.IndexInfo

		if dbType == "mysql" {
			provider := mysql.NewMetadataProvider(db)
			indexInfos, err := provider.Indexes(ctx, table)
			if err != nil {
				return nil, fmt.Errorf("failed to get indexes for %s: %w", table, err)
			}
			for _, idx := range indexInfos {
				indexes = append(indexes, advisor.IndexInfo{
					Name:     idx.Name,
					Columns:  idx.Columns,
					Unique:   idx.Unique,
					Method:   idx.Method,
					IsBTree:  idx.IsBTree,
					Metadata: map[string]any{"table": idx.Table},
				})
			}
		} else {
			provider := postgres.NewMetadataProvider(db)
			indexInfos, err := provider.Indexes(ctx, table)
			if err != nil {
				return nil, fmt.Errorf("failed to get indexes for %s: %w", table, err)
			}
			for _, idx := range indexInfos {
				indexes = append(indexes, advisor.IndexInfo{
					Name:     idx.Name,
					Columns:  idx.Columns,
					Unique:   idx.Unique,
					Method:   idx.Method,
					IsBTree:  idx.IsBTree,
					Metadata: map[string]any{"table": idx.Table},
				})
			}
		}

		schemasByTable[table] = advisor.TableSchema{
			TableName: table,
			Indexes:   indexes,
		}
	}

	return schemasByTable, nil
}

func runMigrate(args []string) error {
	if len(args) == 0 {
		printMigrateUsage()
		return nil
	}

	subCmd := args[0]
	switch subCmd {
	case "up":
		return runMigrateUp(args[1:])
	case "down":
		return runMigrateDown(args[1:])
	case "create":
		return runMigrateCreate(args[1:])
	case "status":
		return runMigrateStatus(args[1:])
	case "-h", "--help", "help":
		printMigrateUsage()
		return nil
	default:
		return fmt.Errorf("unknown migrate subcommand: %s", subCmd)
	}
}

func printMigrateUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  gore-lint migrate up [--dsn <dsn>] [--dir <path>]")
	fmt.Fprintln(os.Stderr, "  gore-lint migrate down [--dsn <dsn>] [--dir <path>] [--steps <n>]")
	fmt.Fprintln(os.Stderr, "  gore-lint migrate create [--dir <path>] <name>")
	fmt.Fprintln(os.Stderr, "  gore-lint migrate status [--dsn <dsn>] [--dir <path>]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Subcommands:")
	fmt.Fprintln(os.Stderr, "  up       Apply pending migrations")
	fmt.Fprintln(os.Stderr, "  down     Rollback migrations")
	fmt.Fprintln(os.Stderr, "  create   Create new migration files")
	fmt.Fprintln(os.Stderr, "  status   Show migration status")
}

func runMigrateUp(args []string) error {
	fs := flag.NewFlagSet("migrate up", flag.ContinueOnError)
	dsn := fs.String("dsn", "", "database DSN")
	dir := fs.String("dir", "migrations", "migrations directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *dsn == "" {
		return fmt.Errorf("--dsn is required")
	}

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	runner := migrate.NewRunner(db)
	ctx := context.Background()

	if err := runner.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize migrations table: %w", err)
	}

	migrations, err := migrate.LoadMigrations(*dir)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := runner.AppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	appliedSet := make(map[string]bool)
	for _, v := range applied {
		appliedSet[v] = true
	}

	count := 0
	for _, m := range migrations {
		if appliedSet[m.Version] {
			continue
		}

		fmt.Printf("Applying migration %s: %s\n", m.Version, m.Name)
		if err := runner.Up(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		count++
	}

	if count == 0 {
		fmt.Println("No pending migrations")
	} else {
		fmt.Printf("Applied %d migration(s)\n", count)
	}

	return nil
}

func runMigrateDown(args []string) error {
	fs := flag.NewFlagSet("migrate down", flag.ContinueOnError)
	dsn := fs.String("dsn", "", "database DSN")
	dir := fs.String("dir", "migrations", "migrations directory")
	steps := fs.Int("steps", 1, "number of migrations to rollback")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *dsn == "" {
		return fmt.Errorf("--dsn is required")
	}

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	runner := migrate.NewRunner(db)
	ctx := context.Background()

	migrations, err := migrate.LoadMigrations(*dir)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := runner.AppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	if len(applied) == 0 {
		fmt.Println("No migrations to rollback")
		return nil
	}

	migrationsMap := make(map[string]*migrate.Migration)
	for _, m := range migrations {
		migrationsMap[m.Version] = m
	}

	count := 0
	for i := len(applied) - 1; i >= 0 && count < *steps; i-- {
		version := applied[i]
		m, exists := migrationsMap[version]
		if !exists {
			return fmt.Errorf("migration file not found for version %s", version)
		}

		fmt.Printf("Rolling back migration %s: %s\n", m.Version, m.Name)
		if err := runner.Down(ctx, m); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}
		count++
	}

	fmt.Printf("Rolled back %d migration(s)\n", count)
	return nil
}

func runMigrateCreate(args []string) error {
	fs := flag.NewFlagSet("migrate create", flag.ContinueOnError)
	dir := fs.String("dir", "migrations", "migrations directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("migration name is required")
	}

	name := strings.Join(rest, " ")
	upFile, downFile, err := migrate.CreateMigration(*dir, name)
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	fmt.Printf("Created migration files:\n")
	fmt.Printf("  %s\n", upFile)
	fmt.Printf("  %s\n", downFile)

	return nil
}

func runMigrateStatus(args []string) error {
	fs := flag.NewFlagSet("migrate status", flag.ContinueOnError)
	dsn := fs.String("dsn", "", "database DSN")
	dir := fs.String("dir", "migrations", "migrations directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *dsn == "" {
		return fmt.Errorf("--dsn is required")
	}

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	runner := migrate.NewRunner(db)
	ctx := context.Background()

	if err := runner.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize migrations table: %w", err)
	}

	migrations, err := migrate.LoadMigrations(*dir)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := runner.AppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	appliedSet := make(map[string]bool)
	for _, v := range applied {
		appliedSet[v] = true
	}

	fmt.Println("Migration Status:")
	fmt.Println("Version          | Name                           | Status")
	fmt.Println("-----------------|--------------------------------|--------")

	for _, m := range migrations {
		status := "pending"
		if appliedSet[m.Version] {
			status = "applied"
		}
		fmt.Printf("%-16s | %-30s | %s\n", m.Version, m.Name, status)
	}

	return nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  gore-lint check [--dsn <dsn> | --schema <file>] [--db-type <type>] [--format <format>] [--output <file>] [--cross-file] <target>")
	fmt.Fprintln(os.Stderr, "  gore-lint migrate <subcommand> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  check                Analyze source paths and report index risks")
	fmt.Fprintln(os.Stderr, "  migrate up           Apply pending migrations")
	fmt.Fprintln(os.Stderr, "  migrate down         Rollback migrations")
	fmt.Fprintln(os.Stderr, "  migrate create       Create new migration files")
	fmt.Fprintln(os.Stderr, "  migrate status       Show migration status")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Check Flags:")
	fmt.Fprintln(os.Stderr, "  --dsn <dsn>          database DSN for live schema")
	fmt.Fprintln(os.Stderr, "  --schema <file>      path to schema cache JSON")
	fmt.Fprintln(os.Stderr, "  --db-type <type>     database type: postgres or mysql (default: postgres)")
	fmt.Fprintln(os.Stderr, "  --stdout             write diagnostics to stdout (default: true)")
	fmt.Fprintln(os.Stderr, "  --format <format>    output format: json, text, html, or sarif (default: json)")
	fmt.Fprintln(os.Stderr, "  --output <file>      output file path (default: stdout)")
	fmt.Fprintln(os.Stderr, "  --cross-file         enable cross-file constant resolution using go/packages")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Migrate Flags:")
	fmt.Fprintln(os.Stderr, "  --dsn <dsn>          database DSN (required for up/down/status)")
	fmt.Fprintln(os.Stderr, "  --dir <path>         migrations directory (default: migrations)")
	fmt.Fprintln(os.Stderr, "  --steps <n>          number of migrations to rollback (down only, default: 1)")
}
