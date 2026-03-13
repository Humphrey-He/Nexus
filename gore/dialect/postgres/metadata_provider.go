package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"gore/internal/metadata"
)

// MetadataProvider loads PostgreSQL index metadata from pg_catalog.
type MetadataProvider struct {
	db *sql.DB
}

// NewMetadataProvider creates a PostgreSQL metadata provider.
func NewMetadataProvider(db *sql.DB) *MetadataProvider {
	return &MetadataProvider{db: db}
}

// Indexes returns index metadata for a given table.
// The query is read-only and avoids locks on user tables.
func (p *MetadataProvider) Indexes(ctx context.Context, table string) ([]metadata.IndexInfo, error) {
	const q = `
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
`

	rows, err := p.db.QueryContext(ctx, q, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []metadata.IndexInfo
	for rows.Next() {
		var info metadata.IndexInfo
		var indexDef string
		var columnsJSON []byte
		if err := rows.Scan(&info.Name, &info.Table, &info.Unique, &info.Method, &indexDef, &columnsJSON); err != nil {
			return nil, err
		}

		if len(columnsJSON) > 0 {
			if err := json.Unmarshal(columnsJSON, &info.Columns); err != nil {
				return nil, err
			}
		}
		info.IsBTree = info.Method == "btree"
		out = append(out, info)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
