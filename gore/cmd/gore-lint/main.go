package main

import (
	"context"
	"database/sql"
	"encoding/json"
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

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"gore/dialect/mysql"
	"gore/dialect/postgres"
	"gore/internal/advisor"
	"gore/internal/advisor/rules"
	"gore/internal/migrate"
)

type config struct {
	dsn       string
	schema    string
	dbType    string
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

	queries, err := extractQueries(cfg.target)
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
	fmt.Fprintln(os.Stderr, "  gore-lint check [--dsn <dsn> | --schema <file>] [--db-type <type>] [--stdout] <target>")
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
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Migrate Flags:")
	fmt.Fprintln(os.Stderr, "  --dsn <dsn>          database DSN (required for up/down/status)")
	fmt.Fprintln(os.Stderr, "  --dir <path>         migrations directory (default: migrations)")
	fmt.Fprintln(os.Stderr, "  --steps <n>          number of migrations to rollback (down only, default: 1)")
}
