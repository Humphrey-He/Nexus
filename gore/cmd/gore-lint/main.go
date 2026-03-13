package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"gore/internal/advisor"
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

	// TODO: Load schema from live database or schema cache based on flags.
	// TODO: Extract QueryMetadata from target source path.

	engine := advisor.NewEngine()
	_ = engine

	rep := report{
		Version:     "0.1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Target:      cfg.target,
		Suggestions: []advisor.Suggestion{},
		Stats:       reportStats{},
	}

	return writeReport(rep, cfg.useStdout)
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
