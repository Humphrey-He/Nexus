package postgres

import (
	"testing"

	"gore/dialect"
)

func TestBuildSelect(t *testing.T) {
	d := &Dialector{}

	tests := []struct {
		name     string
		ast      *dialect.QueryAST
		wantSQL  string
		wantErr  bool
	}{
		{
			name:    "nil ast",
			ast:     nil,
			wantErr: true,
		},
		{
			name:    "empty table",
			ast:     &dialect.QueryAST{Table: ""},
			wantErr: true,
		},
		{
			name:    "basic select",
			ast:     &dialect.QueryAST{Table: "users"},
			wantSQL: "SELECT * FROM users",
			wantErr: false,
		},
		{
			name: "select with columns",
			ast: &dialect.QueryAST{
				Table:   "users",
				Columns: []string{"id", "name"},
			},
			wantSQL: "SELECT id, name FROM users",
			wantErr: false,
		},
		{
			name: "select with where",
			ast: &dialect.QueryAST{
				Table: "users",
				Where: []string{"id = $1"},
			},
			wantSQL: "SELECT * FROM users WHERE id = $1",
			wantErr: false,
		},
		{
			name: "select with order by",
			ast: &dialect.QueryAST{
				Table:   "users",
				OrderBy: []string{"created_at DESC"},
			},
			wantSQL: "SELECT * FROM users ORDER BY created_at DESC",
			wantErr: false,
		},
		{
			name: "select with limit offset",
			ast: &dialect.QueryAST{
				Table:  "users",
				Limit:  10,
				Offset: 20,
			},
			wantSQL: "SELECT * FROM users LIMIT 10 OFFSET 20",
			wantErr: false,
		},
		{
			name: "full query",
			ast: &dialect.QueryAST{
				Table:   "users",
				Columns: []string{"id", "name"},
				Where:   []string{"age > $1", "status = $2"},
				OrderBy: []string{"name ASC"},
				Limit:   10,
			},
			wantSQL: "SELECT id, name FROM users WHERE age > $1 AND status = $2 ORDER BY name ASC LIMIT 10",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := d.BuildSelect(tt.ast)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildSelect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantSQL {
				t.Errorf("BuildSelect() = %q, want %q", got, tt.wantSQL)
			}
		})
	}
}

func TestBuildInsert(t *testing.T) {
	d := &Dialector{}

	tests := []struct {
		name    string
		ast     *dialect.InsertAST
		wantSQL string
		wantErr bool
	}{
		{
			name:    "nil ast",
			ast:     nil,
			wantErr: true,
		},
		{
			name:    "empty table",
			ast:     &dialect.InsertAST{Table: ""},
			wantErr: true,
		},
		{
			name:    "no columns",
			ast:     &dialect.InsertAST{Table: "users"},
			wantErr: true,
		},
		{
			name: "basic insert",
			ast: &dialect.InsertAST{
				Table:   "users",
				Columns: []string{"name", "email"},
			},
			wantSQL: "INSERT INTO users (name, email) VALUES ($1, $2)",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := d.BuildInsert(tt.ast)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildInsert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantSQL {
				t.Errorf("BuildInsert() = %q, want %q", got, tt.wantSQL)
			}
		})
	}
}

func TestBuildUpdate(t *testing.T) {
	d := &Dialector{}

	tests := []struct {
		name    string
		ast     *dialect.UpdateAST
		wantSQL string
		wantErr bool
	}{
		{
			name:    "nil ast",
			ast:     nil,
			wantErr: true,
		},
		{
			name:    "empty table",
			ast:     &dialect.UpdateAST{Table: ""},
			wantErr: true,
		},
		{
			name:    "no columns",
			ast:     &dialect.UpdateAST{Table: "users"},
			wantErr: true,
		},
		{
			name: "basic update",
			ast: &dialect.UpdateAST{
				Table:   "users",
				Columns: []string{"name", "status"},
			},
			wantSQL: "UPDATE users SET name = $1, status = $2",
			wantErr: false,
		},
		{
			name: "update with where",
			ast: &dialect.UpdateAST{
				Table:   "users",
				Columns: []string{"status"},
				Where:   []string{"id = $1"},
			},
			wantSQL: "UPDATE users SET status = $1 WHERE id = $1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := d.BuildUpdate(tt.ast)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantSQL {
				t.Errorf("BuildUpdate() = %q, want %q", got, tt.wantSQL)
			}
		})
	}
}

func TestBuildDelete(t *testing.T) {
	d := &Dialector{}

	tests := []struct {
		name    string
		ast     *dialect.DeleteAST
		wantSQL string
		wantErr bool
	}{
		{
			name:    "nil ast",
			ast:     nil,
			wantErr: true,
		},
		{
			name:    "empty table",
			ast:     &dialect.DeleteAST{Table: ""},
			wantErr: true,
		},
		{
			name:    "basic delete",
			ast:     &dialect.DeleteAST{Table: "users"},
			wantSQL: "DELETE FROM users",
			wantErr: false,
		},
		{
			name: "delete with where",
			ast: &dialect.DeleteAST{
				Table: "users",
				Where: []string{"id = $1", "status = $2"},
			},
			wantSQL: "DELETE FROM users WHERE id = $1 AND status = $2",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := d.BuildDelete(tt.ast)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantSQL {
				t.Errorf("BuildDelete() = %q, want %q", got, tt.wantSQL)
			}
		})
	}
}
