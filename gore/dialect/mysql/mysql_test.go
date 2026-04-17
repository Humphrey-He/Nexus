package mysql

import (
	"testing"

	"gore/dialect"
)

func TestMySQLName(t *testing.T) {
	d := &Dialector{}
	if got := d.Name(); got != "mysql" {
		t.Fatalf("expected 'mysql', got %q", got)
	}
}

func TestBuildSelectBasic(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.QueryAST{
		Table: "users",
	}

	sql, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "SELECT * FROM `users`"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildSelectWithColumns(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.QueryAST{
		Table:   "users",
		Columns: []string{"id", "name"},
	}

	sql, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "SELECT `id`, `name` FROM `users`"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildSelectWithWhere(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.QueryAST{
		Table: "users",
		Where: []string{"name = ?"},
	}

	sql, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "SELECT * FROM `users` WHERE name = ?"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildSelectWithLimitOffset(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.QueryAST{
		Table:  "users",
		Limit:  10,
		Offset: 20,
	}

	sql, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "SELECT * FROM `users` LIMIT 20, 10"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildSelectWithLimitOnly(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.QueryAST{
		Table: "users",
		Limit: 10,
	}

	sql, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "SELECT * FROM `users` LIMIT 10"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildSelectWithOffsetOnly(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.QueryAST{
		Table:  "users",
		Offset: 20,
	}

	sql, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// MySQL requires LIMIT when using OFFSET, defaults to limit = offset
	expected := "SELECT * FROM `users` LIMIT 20"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildSelectWithOrderBy(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.QueryAST{
		Table:   "users",
		OrderBy: []string{"name ASC", "created_at DESC"},
	}

	sql, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "SELECT * FROM `users` ORDER BY name ASC, created_at DESC"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildSelectWithGroupBy(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.QueryAST{
		Table:   "users",
		GroupBy: []string{"status", "category"},
	}

	sql, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "SELECT * FROM `users` GROUP BY `status`, `category`"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildSelectFull(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.QueryAST{
		Table:   "users",
		Columns: []string{"id", "name"},
		Where:   []string{"status > ?"},
		OrderBy: []string{"created_at DESC"},
		Limit:   10,
		Offset:  5,
	}

	sql, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "SELECT `id`, `name` FROM `users` WHERE status > ? ORDER BY created_at DESC LIMIT 5, 10"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildSelectNilAST(t *testing.T) {
	d := &Dialector{}
	_, _, err := d.BuildSelect(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestBuildSelectEmptyTable(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.QueryAST{
		Table: "",
	}
	_, _, err := d.BuildSelect(ast)
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

func TestBuildInsertBasic(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.InsertAST{
		Table:   "users",
		Columns: []string{"name", "email"},
	}

	sql, _, err := d.BuildInsert(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "INSERT INTO `users` (`name`, `email`) VALUES (?, ?)"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildInsertNilAST(t *testing.T) {
	d := &Dialector{}
	_, _, err := d.BuildInsert(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestBuildInsertEmptyTable(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.InsertAST{
		Table:   "",
		Columns: []string{"name"},
	}
	_, _, err := d.BuildInsert(ast)
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

func TestBuildInsertNoColumns(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.InsertAST{
		Table:   "users",
		Columns: []string{},
	}
	_, _, err := d.BuildInsert(ast)
	if err == nil {
		t.Fatal("expected error for no columns")
	}
}

func TestBuildUpdateBasic(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.UpdateAST{
		Table:   "users",
		Columns: []string{"name", "email"},
		Where:   []string{"id = ?"},
	}

	sql, _, err := d.BuildUpdate(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "UPDATE `users` SET `name` = ?, `email` = ? WHERE id = ?"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildUpdateNilAST(t *testing.T) {
	d := &Dialector{}
	_, _, err := d.BuildUpdate(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestBuildUpdateEmptyTable(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.UpdateAST{
		Table:   "",
		Columns: []string{"name"},
	}
	_, _, err := d.BuildUpdate(ast)
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

func TestBuildUpdateNoColumns(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.UpdateAST{
		Table:   "users",
		Columns: []string{},
	}
	_, _, err := d.BuildUpdate(ast)
	if err == nil {
		t.Fatal("expected error for no columns")
	}
}

func TestBuildDeleteBasic(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.DeleteAST{
		Table: "users",
		Where: []string{"id = ?"},
	}

	sql, _, err := d.BuildDelete(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "DELETE FROM `users` WHERE id = ?"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildDeleteNoWhere(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.DeleteAST{
		Table: "users",
	}

	sql, _, err := d.BuildDelete(ast)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "DELETE FROM `users`"
	if sql != expected {
		t.Fatalf("expected %q, got %q", expected, sql)
	}
}

func TestBuildDeleteNilAST(t *testing.T) {
	d := &Dialector{}
	_, _, err := d.BuildDelete(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestBuildDeleteEmptyTable(t *testing.T) {
	d := &Dialector{}
	ast := &dialect.DeleteAST{
		Table: "",
	}
	_, _, err := d.BuildDelete(ast)
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

func TestQuoteIdentifier(t *testing.T) {
	d := &Dialector{}
	tests := []struct {
		input    string
		expected string
	}{
		{"name", "`name`"},
		{"user_id", "`user_id`"},
		{"", "``"},
	}

	for _, tt := range tests {
		got := d.quoteIdentifier(tt.input)
		if got != tt.expected {
			t.Errorf("quoteIdentifier(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestQuoteColumns(t *testing.T) {
	d := &Dialector{}
	cols := []string{"id", "name", "email"}
	got := d.quoteColumns(cols)
	expected := "`id`, `name`, `email`"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
