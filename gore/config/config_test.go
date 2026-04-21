package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.MaxOpenConns != 25 {
		t.Errorf("expected MaxOpenConns=25, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("expected MaxIdleConns=5, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("expected ConnMaxLifetime=5m, got %v", cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime != 1*time.Minute {
		t.Errorf("expected ConnMaxIdleTime=1m, got %v", cfg.ConnMaxIdleTime)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel=info, got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("expected LogFormat=text, got %s", cfg.LogFormat)
	}
	if !cfg.EnableTracking {
		t.Error("expected EnableTracking=true")
	}
	if cfg.EnableMetrics {
		t.Error("expected EnableMetrics=false")
	}
	if cfg.QueryTimeout != 30*time.Second {
		t.Errorf("expected QueryTimeout=30s, got %v", cfg.QueryTimeout)
	}
	if cfg.CommandTimeout != 60*time.Second {
		t.Errorf("expected CommandTimeout=60s, got %v", cfg.CommandTimeout)
	}
	if cfg.LintDBType != "postgres" {
		t.Errorf("expected LintDBType=postgres, got %s", cfg.LintDBType)
	}
	if cfg.LintFormat != "text" {
		t.Errorf("expected LintFormat=text, got %s", cfg.LintFormat)
	}
}

func TestLoadConfig(t *testing.T) {
	// Clear environment variables first
	envVars := []string{
		"GORE_DSN",
		"GORE_MAX_OPEN_CONNS",
		"GORE_MAX_IDLE_CONNS",
		"GORE_CONN_MAX_LIFETIME",
		"GORE_LOG_LEVEL",
		"GORE_LOG_FORMAT",
		"GORE_LINT_SCHEMA",
		"GORE_LINT_DB_TYPE",
		"GORE_LINT_FORMAT",
	}
	originalValues := make(map[string]string)
	for _, v := range envVars {
		originalValues[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for _, v := range envVars {
			if originalValues[v] != "" {
				os.Setenv(v, originalValues[v])
			}
		}
	}()

	// Test with custom values
	os.Setenv("GORE_DSN", "postgres://custom:pass@localhost:5432/custom")
	os.Setenv("GORE_MAX_OPEN_CONNS", "50")
	os.Setenv("GORE_MAX_IDLE_CONNS", "10")
	os.Setenv("GORE_CONN_MAX_LIFETIME", "10m")
	os.Setenv("GORE_LOG_LEVEL", "debug")
	os.Setenv("GORE_LOG_FORMAT", "json")
	os.Setenv("GORE_LINT_SCHEMA", "/path/to/schema.json")
	os.Setenv("GORE_LINT_DB_TYPE", "mysql")
	os.Setenv("GORE_LINT_FORMAT", "json")

	cfg := LoadConfig()

	if cfg.DSN != "postgres://custom:pass@localhost:5432/custom" {
		t.Errorf("expected custom DSN, got %s", cfg.DSN)
	}
	if cfg.MaxOpenConns != 50 {
		t.Errorf("expected MaxOpenConns=50, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 10 {
		t.Errorf("expected MaxIdleConns=10, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 10*time.Minute {
		t.Errorf("expected ConnMaxLifetime=10m, got %v", cfg.ConnMaxLifetime)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel=debug, got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("expected LogFormat=json, got %s", cfg.LogFormat)
	}
	if cfg.LintSchemaPath != "/path/to/schema.json" {
		t.Errorf("expected LintSchemaPath=/path/to/schema.json, got %s", cfg.LintSchemaPath)
	}
	if cfg.LintDBType != "mysql" {
		t.Errorf("expected LintDBType=mysql, got %s", cfg.LintDBType)
	}
	if cfg.LintFormat != "json" {
		t.Errorf("expected LintFormat=json, got %s", cfg.LintFormat)
	}
}

func TestLoadConfigInvalidMaxOpenConns(t *testing.T) {
	os.Setenv("GORE_MAX_OPEN_CONNS", "invalid")
	defer os.Unsetenv("GORE_MAX_OPEN_CONNS")

	cfg := LoadConfig()
	if cfg.MaxOpenConns != 25 {
		t.Errorf("expected default MaxOpenConns=25 for invalid input, got %d", cfg.MaxOpenConns)
	}
}

func TestLoadConfigInvalidMaxIdleConns(t *testing.T) {
	os.Setenv("GORE_MAX_IDLE_CONNS", "invalid")
	defer os.Unsetenv("GORE_MAX_IDLE_CONNS")

	cfg := LoadConfig()
	if cfg.MaxIdleConns != 5 {
		t.Errorf("expected default MaxIdleConns=5 for invalid input, got %d", cfg.MaxIdleConns)
	}
}

func TestLoadConfigInvalidConnMaxLifetime(t *testing.T) {
	os.Setenv("GORE_CONN_MAX_LIFETIME", "invalid")
	defer os.Unsetenv("GORE_CONN_MAX_LIFETIME")

	cfg := LoadConfig()
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("expected default ConnMaxLifetime=5m for invalid input, got %v", cfg.ConnMaxLifetime)
	}
}

func TestIsDevelopment(t *testing.T) {
	originalEnv := os.Getenv("GORE_ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GORE_ENV", originalEnv)
		} else {
			os.Unsetenv("GORE_ENV")
		}
	}()

	os.Unsetenv("GORE_ENV")
	if !IsDevelopment() {
		t.Error("expected IsDevelopment()=true when GORE_ENV is not set")
	}

	os.Setenv("GORE_ENV", "development")
	if !IsDevelopment() {
		t.Error("expected IsDevelopment()=true for GORE_ENV=development")
	}

	os.Setenv("GORE_ENV", "staging")
	if !IsDevelopment() {
		t.Error("expected IsDevelopment()=true for GORE_ENV=staging")
	}

	os.Setenv("GORE_ENV", "production")
	if IsDevelopment() {
		t.Error("expected IsDevelopment()=false for GORE_ENV=production")
	}
}

func TestIsProduction(t *testing.T) {
	originalEnv := os.Getenv("GORE_ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GORE_ENV", originalEnv)
		} else {
			os.Unsetenv("GORE_ENV")
		}
	}()

	os.Unsetenv("GORE_ENV")
	if IsProduction() {
		t.Error("expected IsProduction()=false when GORE_ENV is not set")
	}

	os.Setenv("GORE_ENV", "production")
	if !IsProduction() {
		t.Error("expected IsProduction()=true for GORE_ENV=production")
	}

	os.Setenv("GORE_ENV", "development")
	if IsProduction() {
		t.Error("expected IsProduction()=false for GORE_ENV=development")
	}
}

func TestDSN(t *testing.T) {
	originalEnv := os.Getenv("GORE_DSN")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GORE_DSN", originalEnv)
		} else {
			os.Unsetenv("GORE_DSN")
		}
	}()

	os.Unsetenv("GORE_DSN")
	dsn := DSN()
	if dsn != "postgres://gore:gore123@localhost:5432/gore?sslmode=disable" {
		t.Errorf("expected default DSN, got %s", dsn)
	}

	os.Setenv("GORE_DSN", "mysql://user:pass@localhost:3306/db")
	dsn = DSN()
	if dsn != "mysql://user:pass@localhost:3306/db" {
		t.Errorf("expected custom DSN, got %s", dsn)
	}
}

func TestDSNPriority(t *testing.T) {
	originalEnv := os.Getenv("GORE_DSN")
	defer func() {
		if originalEnv != "" {
			os.Setenv("GORE_DSN", originalEnv)
		} else {
			os.Unsetenv("GORE_DSN")
		}
	}()

	// LoadConfig should use GORE_DSN env var over default
	os.Setenv("GORE_DSN", "postgres://priority:pass@localhost:5432/priority")
	cfg := LoadConfig()
	if cfg.DSN != "postgres://priority:pass@localhost:5432/priority" {
		t.Errorf("expected priority DSN, got %s", cfg.DSN)
	}
}
