package executor

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewPooledExecutorWithNilConfig(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)
	if executor == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestNewPooledExecutorWithConfig(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	cfg := &PoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 10 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	executor := NewPooledExecutor(db, cfg)
	if executor == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestPooledExecutorQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Alice")
	mock.ExpectQuery("SELECT \\? FROM users").WithArgs(1).WillReturnRows(rows)

	result, err := executor.Query(context.Background(), "SELECT ? FROM users", 1)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil rows")
	}

	if !result.Next() {
		t.Fatal("expected at least one row")
	}

	var id int
	var name string
	if err := result.Scan(&id, &name); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if id != 1 || name != "Alice" {
		t.Fatalf("unexpected values: id=%d, name=%s", id, name)
	}

	result.Close()
}

func TestPooledExecutorQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("query error"))

	_, err = executor.Query(context.Background(), "SELECT * FROM users")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPooledExecutorExec(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	mock.ExpectExec("INSERT INTO users").WithArgs("Alice").WillReturnResult(sqlmock.NewResult(1, 1))

	result, err := executor.Exec(context.Background(), "INSERT INTO users (name) VALUES (?)", "Alice")
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected failed: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected 1 affected row, got %d", affected)
	}
}

func TestPooledExecutorExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	mock.ExpectExec("INSERT INTO users").WillReturnError(errors.New("exec error"))

	_, err = executor.Exec(context.Background(), "INSERT INTO users (name) VALUES (?)", "Alice")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPooledExecutorClose(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}

	mock.ExpectClose()

	executor := NewPooledExecutor(db, nil)

	if err := executor.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestPooledExecutorStats(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	stats := executor.Stats()
	if stats == (sql.DBStats{}) {
		t.Fatal("expected non-empty stats")
	}
}

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.MaxOpenConns != 25 {
		t.Fatalf("expected MaxOpenConns=25, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Fatalf("expected MaxIdleConns=5, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Fatalf("expected ConnMaxLifetime=5m, got %v", cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime != 1*time.Minute {
		t.Fatalf("expected ConnMaxIdleTime=1m, got %v", cfg.ConnMaxIdleTime)
	}
}

func TestPoolConfigBoundaryValues(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *PoolConfig
		wantMax int
		wantIdle int
	}{
		{
			name:    "zero values",
			cfg:     &PoolConfig{MaxOpenConns: 0, MaxIdleConns: 0},
			wantMax: 0,
			wantIdle: 0,
		},
		{
			name:    "negative values",
			cfg:     &PoolConfig{MaxOpenConns: -1, MaxIdleConns: -1},
			wantMax: -1,
			wantIdle: -1,
		},
		{
			name:    "large values",
			cfg:     &PoolConfig{MaxOpenConns: 1000, MaxIdleConns: 500},
			wantMax: 1000,
			wantIdle: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock: %v", err)
			}
			defer db.Close()

			executor := NewPooledExecutor(db, tt.cfg)
			if executor == nil {
				t.Fatal("expected non-nil executor")
			}

			stats := executor.Stats()
			if tt.wantMax >= 0 && stats.MaxOpenConnections != tt.wantMax {
				t.Fatalf("expected MaxOpenConnections=%d, got %d", tt.wantMax, stats.MaxOpenConnections)
			}
		})
	}
}

func TestPooledExecutorQueryContextCancellation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mock.ExpectQuery("SELECT").WillReturnError(context.Canceled)

	_, err = executor.Query(ctx, "SELECT * FROM users")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestPooledExecutorQueryDeadlineExceeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	mock.ExpectQuery("SELECT").WillReturnError(context.DeadlineExceeded)

	_, err = executor.Query(ctx, "SELECT * FROM users")
	if err == nil {
		t.Fatal("expected error for deadline exceeded")
	}
}

func TestSqlRowsNextMultiple(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	rows := sqlmock.NewRows([]string{"id"}).
		AddRow(1).
		AddRow(2).
		AddRow(3)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := executor.Query(context.Background(), "SELECT id FROM users")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	count := 0
	for result.Next() {
		count++
	}
	if count != 3 {
		t.Fatalf("expected 3 rows, got %d", count)
	}

	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
}

func TestSqlRowsCloseTwice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := executor.Query(context.Background(), "SELECT id FROM users")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	result.Close()
	result.Close() // Second close should not panic
}

func TestSqlResultRowsAffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 5))

	result, err := executor.Exec(context.Background(), "UPDATE users SET status = 1")
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected failed: %v", err)
	}
	if affected != 5 {
		t.Fatalf("expected 5 affected rows, got %d", affected)
	}
}

func TestPooledExecutorEmptyQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	rows := sqlmock.NewRows([]string{})
	mock.ExpectQuery("").WillReturnRows(rows)

	result, err := executor.Query(context.Background(), "")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if result.Next() {
		t.Fatal("expected no rows for empty query")
	}
}

func TestPooledExecutorNilResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	executor := NewPooledExecutor(db, nil)

	mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 0))

	result, err := executor.Exec(context.Background(), "DELETE FROM users")
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected failed: %v", err)
	}
	if affected != 0 {
		t.Fatalf("expected 0 affected rows, got %d", affected)
	}
}