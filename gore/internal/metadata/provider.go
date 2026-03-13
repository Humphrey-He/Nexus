package metadata

import "context"

// Provider loads schema metadata for a given table.
type Provider interface {
	Indexes(ctx context.Context, table string) ([]IndexInfo, error)
}
