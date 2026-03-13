package postgres

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMetadataProviderIndexes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("mock init failed: %v", err)
	}
	defer db.Close()

	provider := NewMetadataProvider(db)

	query := regexp.QuoteMeta(`
SELECT
    i.relname AS index_name,
    t.relname AS table_name,
    ix.indisunique AS is_unique,
    am.amname AS method,
    pg_get_indexdef(ix.indexrelid) AS index_def,
    json_agg(a.attname ORDER BY x.ord) AS columns
FROM pg_index ix
JOIN pg_class t ON t.oid = ix.indrelid
JOIN pg_class i ON i.oid = ix.indexrelid
JOIN pg_am am ON am.oid = i.relam
JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS x(attnum, ord) ON true
JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = x.attnum
WHERE t.relname = $1
GROUP BY i.relname, t.relname, ix.indisunique, am.amname, ix.indexrelid
ORDER BY i.relname;
`)

	rows := sqlmock.NewRows([]string{"index_name", "table_name", "is_unique", "method", "index_def", "columns"}).
		AddRow("users_pkey", "users", true, "btree", "CREATE UNIQUE INDEX users_pkey ON users (id)", `["id"]`).
		AddRow("users_name_idx", "users", false, "btree", "CREATE INDEX users_name_idx ON users (name)", `["name"]`)

	mock.ExpectQuery(query).WithArgs("users").WillReturnRows(rows)

	indexes, err := provider.Indexes(context.Background(), "users")
	if err != nil {
		t.Fatalf("indexes failed: %v", err)
	}

	if len(indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(indexes))
	}

	if !indexes[0].IsBTree || indexes[0].Method != "btree" {
		t.Fatalf("expected btree index, got method=%s", indexes[0].Method)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
