package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

type Migration struct {
	Version   string
	Name      string
	UpSQL     string
	DownSQL   string
	AppliedAt *time.Time
}

type Runner struct {
	db        *sql.DB
	tableName string
}

func NewRunner(db *sql.DB) *Runner {
	return &Runner{
		db:        db,
		tableName: "schema_migrations",
	}
}

func (r *Runner) Init(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`, r.tableName)
	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *Runner) AppliedVersions(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf("SELECT version FROM %s ORDER BY version", r.tableName)
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []string
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (r *Runner) Up(ctx context.Context, m *Migration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.UpSQL); err != nil {
		return fmt.Errorf("migration %s failed: %w", m.Version, err)
	}

	query := fmt.Sprintf("INSERT INTO %s (version) VALUES ($1)", r.tableName)
	if _, err := tx.ExecContext(ctx, query, m.Version); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Runner) Down(ctx context.Context, m *Migration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.DownSQL); err != nil {
		return fmt.Errorf("rollback %s failed: %w", m.Version, err)
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE version = $1", r.tableName)
	if _, err := tx.ExecContext(ctx, query, m.Version); err != nil {
		return err
	}

	return tx.Commit()
}

func LoadMigrations(dir string) ([]*Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	migrations := make(map[string]*Migration)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 {
			continue
		}

		version := parts[0]
		rest := strings.TrimSuffix(parts[1], ".sql")

		var direction Direction
		if strings.HasSuffix(rest, ".up") {
			direction = DirectionUp
			rest = strings.TrimSuffix(rest, ".up")
		} else if strings.HasSuffix(rest, ".down") {
			direction = DirectionDown
			rest = strings.TrimSuffix(rest, ".down")
		} else {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}

		m, exists := migrations[version]
		if !exists {
			m = &Migration{
				Version: version,
				Name:    rest,
			}
			migrations[version] = m
		}

		if direction == DirectionUp {
			m.UpSQL = string(content)
		} else {
			m.DownSQL = string(content)
		}
	}

	result := make([]*Migration, 0, len(migrations))
	for _, m := range migrations {
		result = append(result, m)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

func CreateMigration(dir, name string) (string, string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err
	}

	version := time.Now().UTC().Format("20060102150405")
	safeName := strings.ReplaceAll(strings.ToLower(name), " ", "_")

	upFile := filepath.Join(dir, fmt.Sprintf("%s_%s.up.sql", version, safeName))
	downFile := filepath.Join(dir, fmt.Sprintf("%s_%s.down.sql", version, safeName))

	upTemplate := fmt.Sprintf("-- Migration: %s\n-- Created: %s\n\n", name, time.Now().Format(time.RFC3339))
	downTemplate := fmt.Sprintf("-- Rollback: %s\n-- Created: %s\n\n", name, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(upFile, []byte(upTemplate), 0644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(downFile, []byte(downTemplate), 0644); err != nil {
		return "", "", err
	}

	return upFile, downFile, nil
}
