package mysql

import (
	"context"
	"database/sql"

	"gore/internal/metadata"
)

// MetadataProvider loads MySQL index metadata from INFORMATION_SCHEMA.
type MetadataProvider struct {
	db *sql.DB
}

// NewMetadataProvider creates a MySQL metadata provider.
func NewMetadataProvider(db *sql.DB) *MetadataProvider {
	return &MetadataProvider{db: db}
}

// Indexes returns index metadata for a given table.
// Queries INFORMATION_SCHEMA.STATISTICS for MySQL 8.0+
func (p *MetadataProvider) Indexes(ctx context.Context, table string) ([]metadata.IndexInfo, error) {
	const q = `
SELECT
    INDEX_NAME,
    TABLE_NAME,
    NON_UNIQUE,
    INDEX_TYPE,
    COLUMN_NAME,
    SEQ_IN_INDEX,
    COLLATION,
    IS_VISIBLE
FROM INFORMATION_SCHEMA.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = ?
ORDER BY INDEX_NAME, SEQ_IN_INDEX;
`

	rows, err := p.db.QueryContext(ctx, q, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Aggregate columns by index
	indexMap := make(map[string]*metadata.IndexInfo)

	for rows.Next() {
		var indexName, tableName, columnName, indexType, collation, isVisible string
		var nonUnique bool
		var seqInIndex int

		if err := rows.Scan(&indexName, &tableName, &nonUnique, &indexType, &columnName, &seqInIndex, &collation, &isVisible); err != nil {
			return nil, err
		}

		info, exists := indexMap[indexName]
		if !exists {
			info = &metadata.IndexInfo{
				Name:      indexName,
				Table:     tableName,
				Unique:    !nonUnique,
				Method:    indexType,
				IsBTree:   indexType == "BTREE",
				IsVisible: isVisible == "YES",
			}
			indexMap[indexName] = info
		}

		// Append column maintaining order
		info.Columns = append(info.Columns, columnName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]metadata.IndexInfo, 0, len(indexMap))
	for _, info := range indexMap {
		out = append(out, *info)
	}

	return out, nil
}

// Columns returns column metadata for a given table.
func (p *MetadataProvider) Columns(ctx context.Context, table string) ([]metadata.ColumnInfo, error) {
	const q = `
SELECT
    COLUMN_NAME,
    DATA_TYPE,
    IS_NULLABLE,
    EXTRA
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = ?
ORDER BY ORDINAL_POSITION;
`

	rows, err := p.db.QueryContext(ctx, q, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []metadata.ColumnInfo
	for rows.Next() {
		var col metadata.ColumnInfo
		var dataType, isNullable, extra string

		if err := rows.Scan(&col.Name, &dataType, &isNullable, &extra); err != nil {
			return nil, err
		}

		// Build type string including extras like auto_increment
		if extra != "" {
			col.Type = dataType + " " + extra
		} else {
			col.Type = dataType
		}

		out = append(out, col)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

// Tables returns all tables in the current database.
func (p *MetadataProvider) Tables(ctx context.Context) ([]string, error) {
	const q = `
SELECT TABLE_NAME
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_TYPE = 'BASE TABLE'
ORDER BY TABLE_NAME;
`

	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}
