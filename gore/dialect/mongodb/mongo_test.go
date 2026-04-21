package mongodb

import (
	"testing"

	"gore/dialect"
)

func TestDialector_Name(t *testing.T) {
	d := &Dialector{}
	if got := d.Name(); got != "mongodb" {
		t.Errorf("Name() = %v, want mongodb", got)
	}
}

func TestDialector_BuildSelect(t *testing.T) {
	d := &Dialector{}

	tests := []struct {
		name    string
		ast     *dialect.QueryAST
		want    string
		wantErr bool
	}{
		{
			name:    "nil ast",
			ast:     nil,
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty table",
			ast:     &dialect.QueryAST{},
			want:    "",
			wantErr: true,
		},
		{
			name: "simple select all",
			ast: &dialect.QueryAST{
				Table: "users",
			},
			want:    "db.users.find({})",
			wantErr: false,
		},
		{
			name: "select with where",
			ast: &dialect.QueryAST{
				Table: "users",
				Where: []string{"age: 25"},
			},
			want:    "db.users.find({ age: 25 })",
			wantErr: false,
		},
		{
			name: "select with columns",
			ast: &dialect.QueryAST{
				Table:   "users",
				Columns: []string{"name", "email"},
			},
			want:    "db.users.find({}, { name: 1, email: 1 })",
			wantErr: false,
		},
		{
			name: "select with order by",
			ast: &dialect.QueryAST{
				Table:   "users",
				OrderBy: []string{"name"},
			},
			want:    "db.users.find({}).sort({ name: 1 })",
			wantErr: false,
		},
		{
			name: "select with limit",
			ast: &dialect.QueryAST{
				Table:  "users",
				Limit:  10,
			},
			want:    "db.users.find({}).limit(10)",
			wantErr: false,
		},
		{
			name: "select with offset",
			ast: &dialect.QueryAST{
				Table:   "users",
				Offset:  20,
			},
			want:    "db.users.find({}).skip(20)",
			wantErr: false,
		},
		{
			name: "select with all options",
			ast: &dialect.QueryAST{
				Table:   "users",
				Columns: []string{"name", "email"},
				Where:   []string{"age: {$gte: 18}"},
				OrderBy: []string{"name"},
				Limit:   10,
				Offset:  5,
			},
			want:    "db.users.find({ age: {$gte: 18} }, { name: 1, email: 1 }).sort({ name: 1 }).limit(10).skip(5)",
			wantErr: false,
		},
		{
			name: "select with descending order",
			ast: &dialect.QueryAST{
				Table:   "users",
				OrderBy: []string{"created_at DESC"},
			},
			want:    "db.users.find({}).sort({ created_at: -1 })",
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
			if got != tt.want {
				t.Errorf("BuildSelect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDialector_BuildInsert(t *testing.T) {
	d := &Dialector{}

	tests := []struct {
		name    string
		ast     *dialect.InsertAST
		want    string
		wantErr bool
	}{
		{
			name:    "nil ast",
			ast:     nil,
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty table",
			ast:     &dialect.InsertAST{},
			want:    "",
			wantErr: true,
		},
		{
			name:    "no columns",
			ast:     &dialect.InsertAST{Table: "users"},
			want:    "",
			wantErr: true,
		},
		{
			name: "single insert",
			ast: &dialect.InsertAST{
				Table:   "users",
				Columns: []string{"name", "email"},
				Values:  [][]any{{"John", "john@example.com"}},
			},
			want:    `db.users.insertOne({ name: John, email: john@example.com })`,
			wantErr: false,
		},
		{
			name: "batch insert",
			ast: &dialect.InsertAST{
				Table:   "users",
				Columns: []string{"name", "email"},
				Values: [][]any{
					{"John", "john@example.com"},
					{"Jane", "jane@example.com"},
				},
			},
			want:    `db.users.insertMany([{ name: John, email: john@example.com }, { name: Jane, email: jane@example.com }])`,
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
			if got != tt.want {
				t.Errorf("BuildInsert() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDialector_BuildUpdate(t *testing.T) {
	d := &Dialector{}

	tests := []struct {
		name    string
		ast     *dialect.UpdateAST
		want    string
		wantErr bool
	}{
		{
			name:    "nil ast",
			ast:     nil,
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty table",
			ast:     &dialect.UpdateAST{},
			want:    "",
			wantErr: true,
		},
		{
			name:    "no columns",
			ast:     &dialect.UpdateAST{Table: "users"},
			want:    "",
			wantErr: true,
		},
		{
			name: "simple update",
			ast: &dialect.UpdateAST{
				Table:   "users",
				Columns: []string{"name"},
			},
			want:    "db.users.updateOne({}, { $set: { name: ? } })",
			wantErr: false,
		},
		{
			name: "update with where",
			ast: &dialect.UpdateAST{
				Table:   "users",
				Columns: []string{"name", "email"},
				Where:   []string{"_id: ObjectId('123')"},
			},
			want:    "db.users.updateOne({ _id: ObjectId('123') }, { $set: { name: ?, email: ? } })",
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
			if got != tt.want {
				t.Errorf("BuildUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDialector_BuildDelete(t *testing.T) {
	d := &Dialector{}

	tests := []struct {
		name    string
		ast     *dialect.DeleteAST
		want    string
		wantErr bool
	}{
		{
			name:    "nil ast",
			ast:     nil,
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty table",
			ast:     &dialect.DeleteAST{},
			want:    "",
			wantErr: true,
		},
		{
			name: "delete all",
			ast: &dialect.DeleteAST{
				Table: "users",
			},
			want:    "db.users.deleteOne({})",
			wantErr: false,
		},
		{
			name: "delete with where",
			ast: &dialect.DeleteAST{
				Table: "users",
				Where: []string{"status: 'inactive'"},
			},
			want:    "db.users.deleteOne({ status: 'inactive' })",
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
			if got != tt.want {
				t.Errorf("BuildDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}
