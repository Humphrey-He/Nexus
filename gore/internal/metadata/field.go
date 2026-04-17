package metadata

import "reflect"

// EntityMeta describes entity mapping information.
type EntityMeta struct {
	Type   reflect.Type
	Table  string
	Fields []FieldMeta
}

// FieldMeta describes field mapping information.
type FieldMeta struct {
	Name   string
	Column string
	Type   reflect.Type
	Index  bool
}

// ColumnInfo holds column metadata from database.
type ColumnInfo struct {
	Name string
	Type string
}
