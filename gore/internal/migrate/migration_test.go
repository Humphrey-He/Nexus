package migrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateMigration(t *testing.T) {
	dir := t.TempDir()

	upFile, downFile, err := CreateMigration(dir, "create users table")
	if err != nil {
		t.Fatalf("CreateMigration failed: %v", err)
	}

	if _, err := os.Stat(upFile); os.IsNotExist(err) {
		t.Errorf("up file not created: %s", upFile)
	}
	if _, err := os.Stat(downFile); os.IsNotExist(err) {
		t.Errorf("down file not created: %s", downFile)
	}

	upContent, _ := os.ReadFile(upFile)
	if len(upContent) == 0 {
		t.Error("up file is empty")
	}

	downContent, _ := os.ReadFile(downFile)
	if len(downContent) == 0 {
		t.Error("down file is empty")
	}
}

func TestLoadMigrations(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "20240101120000_create_users.up.sql"), []byte("CREATE TABLE users (id INT);"), 0644)
	os.WriteFile(filepath.Join(dir, "20240101120000_create_users.down.sql"), []byte("DROP TABLE users;"), 0644)
	os.WriteFile(filepath.Join(dir, "20240102130000_add_email.up.sql"), []byte("ALTER TABLE users ADD email VARCHAR(255);"), 0644)
	os.WriteFile(filepath.Join(dir, "20240102130000_add_email.down.sql"), []byte("ALTER TABLE users DROP COLUMN email;"), 0644)

	migrations, err := LoadMigrations(dir)
	if err != nil {
		t.Fatalf("LoadMigrations failed: %v", err)
	}

	if len(migrations) != 2 {
		t.Errorf("expected 2 migrations, got %d", len(migrations))
	}

	if migrations[0].Version != "20240101120000" {
		t.Errorf("expected version 20240101120000, got %s", migrations[0].Version)
	}
	if migrations[0].Name != "create_users" {
		t.Errorf("expected name create_users, got %s", migrations[0].Name)
	}
	if migrations[0].UpSQL != "CREATE TABLE users (id INT);" {
		t.Errorf("unexpected UpSQL: %s", migrations[0].UpSQL)
	}
	if migrations[0].DownSQL != "DROP TABLE users;" {
		t.Errorf("unexpected DownSQL: %s", migrations[0].DownSQL)
	}

	if migrations[1].Version != "20240102130000" {
		t.Errorf("expected version 20240102130000, got %s", migrations[1].Version)
	}
}

func TestRunnerInit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	runner := NewRunner(db)
	ctx := context.Background()

	if err := runner.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestRunnerUpDown(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	runner := NewRunner(db)
	ctx := context.Background()

	migration := &Migration{
		Version: "20240101120000",
		Name:    "create_users",
		UpSQL:   "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		DownSQL: "DROP TABLE users",
	}

	mock.ExpectBegin()
	mock.ExpectExec("CREATE TABLE users").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").WithArgs("20240101120000").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := runner.Up(ctx, migration); err != nil {
		t.Fatalf("Up failed: %v", err)
	}

	mock.ExpectQuery("SELECT version FROM schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("20240101120000"))

	versions, err := runner.AppliedVersions(ctx)
	if err != nil {
		t.Fatalf("AppliedVersions failed: %v", err)
	}
	if len(versions) != 1 || versions[0] != "20240101120000" {
		t.Errorf("expected version 20240101120000 to be applied, got %v", versions)
	}

	mock.ExpectBegin()
	mock.ExpectExec("DROP TABLE users").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM schema_migrations").WithArgs("20240101120000").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := runner.Down(ctx, migration); err != nil {
		t.Fatalf("Down failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestLoadMigrationsEmptyDir(t *testing.T) {
	dir := t.TempDir()

	migrations, err := LoadMigrations(dir)
	if err != nil {
		t.Fatalf("LoadMigrations failed: %v", err)
	}

	if len(migrations) != 0 {
		t.Errorf("expected 0 migrations in empty dir, got %d", len(migrations))
	}
}

func TestLoadMigrationsNonExistentDir(t *testing.T) {
	migrations, err := LoadMigrations("/nonexistent/path")
	if err != nil {
		t.Fatalf("LoadMigrations should not fail for non-existent dir: %v", err)
	}

	if migrations != nil {
		t.Errorf("expected nil migrations for non-existent dir, got %v", migrations)
	}
}
