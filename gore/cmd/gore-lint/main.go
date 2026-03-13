package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gore/internal/advisor"
	"gore/internal/advisor/rules"
)

type config struct {
	dsn       string
	schema    string
	target    string
	useStdout bool
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
	fs.BoolVar(&cfg.useStdout, "stdout", true, "write diagnostics to stdout")

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
		// TODO: implement live schema fetch via metadata provider.
		return errors.New("live schema loading via --dsn is not implemented yet")
	}

	queries, err := extractQueries(cfg.target)
	if err != nil {
		return err
	}

	engine := advisor.NewEngine(
		rules.NewLeftmostMatchRule(),
	)

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

	return writeReport(rep, cfg.useStdout)
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

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			meta, key := buildQueryMetadata(call, fset, path)
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

func buildQueryMetadata(call *ast.CallExpr, fset *token.FileSet, path string) (*advisor.QueryMetadata, string) {
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
			if v, ok := parseStringLiteral(item.args, 0); ok {
				meta.TableName = v
			}
		case "WhereField":
			cond, ok := parseWhereField(item.args)
			if ok {
				meta.Conditions = append(meta.Conditions, cond)
			}
		case "OrderBy":
			if v, ok := parseStringLiteral(item.args, 0); ok {
				field, dir := parseOrderBy(v)
				if field != "" {
					meta.OrderBy = append(meta.OrderBy, advisor.OrderField{Field: field, Direction: dir})
				}
			}
		case "Limit":
			if v, ok := parseIntLiteral(item.args, 0); ok {
				meta.Limit = &v
			}
		case "Offset":
			if v, ok := parseIntLiteral(item.args, 0); ok {
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

func parseWhereField(args []ast.Expr) (advisor.Condition, bool) {
	if len(args) < 3 {
		return advisor.Condition{}, false
	}
	field, ok := parseStringLiteral(args, 0)
	if !ok {
		return advisor.Condition{}, false
	}
	op, ok := parseStringLiteral(args, 1)
	if !ok {
		return advisor.Condition{}, false
	}
	value, valueType, ok := parseLiteral(args[2])
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

func parseStringLiteral(args []ast.Expr, idx int) (string, bool) {
	if idx >= len(args) {
		return "", false
	}
	lit, ok := args[idx].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return v, true
}

func parseIntLiteral(args []ast.Expr, idx int) (int, bool) {
	if idx >= len(args) {
		return 0, false
	}
	lit, ok := args[idx].(*ast.BasicLit)
	if !ok || (lit.Kind != token.INT && lit.Kind != token.FLOAT) {
		return 0, false
	}
	value, err := strconv.ParseFloat(lit.Value, 64)
	if err != nil {
		return 0, false
	}
	return int(value), true
}

func parseLiteral(expr ast.Expr) (any, string, bool) {
	switch v := expr.(type) {
	case *ast.BasicLit:
		switch v.Kind {
		case token.STRING:
			parsed, err := strconv.Unquote(v.Value)
			if err != nil {
				return nil, "", false
			}
			return parsed, "string", true
		case token.INT:
			parsed, err := strconv.ParseInt(v.Value, 10, 64)
			if err != nil {
				return nil, "", false
			}
			return parsed, "int", true
		case token.FLOAT:
			parsed, err := strconv.ParseFloat(v.Value, 64)
			if err != nil {
				return nil, "", false
			}
			return parsed, "float", true
		}
	case *ast.Ident:
		switch v.Name {
		case "true":
			return true, "bool", true
		case "false":
			return false, "bool", true
		}
	}
	return nil, "", false
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

func writeReport(rep report, useStdout bool) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}

	out := os.Stdout
	if !useStdout {
		out = os.Stderr
	}

	_, err = out.Write(append(data, '\n'))
	return err
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  gore-lint check [--dsn <dsn> | --schema <file>] [--stdout] <target>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  check        Analyze source paths and report index risks")
}
