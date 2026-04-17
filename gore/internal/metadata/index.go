package metadata

// IndexInfo holds index metadata from database catalogs.
type IndexInfo struct {
	Name     string
	Table    string
	Columns  []string
	Unique   bool
	Method   string
	IsBTree  bool
	IsVisible bool // MySQL 8.0+ invisible index flag
}
