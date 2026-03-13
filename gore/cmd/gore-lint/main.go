package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

	// TODO: Extract QueryMetadata from target source path.
	queries := []*advisor.QueryMetadata{}

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
