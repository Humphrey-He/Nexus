package metadata

// IndexInfo holds index metadata from pg_catalog.
type IndexInfo struct {
	Name    string
	Table   string
	Columns []string
	Unique  bool
	Method  string
	IsBTree bool
}
