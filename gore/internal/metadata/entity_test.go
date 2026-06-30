package metadata

import (
	"reflect"
	"sync"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestRegistryGetEmpty(t *testing.T) {
	r := NewRegistry()
	meta, ok := r.Get(reflect.TypeOf(struct{}{}))
	if ok {
		t.Fatal("expected not found for empty registry")
	}
	if meta != nil {
		t.Fatal("expected nil meta for empty registry")
	}
}

func TestRegistryPutAndGet(t *testing.T) {
	r := NewRegistry()

	meta := &EntityMeta{
		Type:  reflect.TypeOf(struct{ Name string }{}),
		Table: "users",
		Fields: []FieldMeta{
			{Name: "Name", Column: "name", Type: reflect.TypeOf("")},
		},
	}

	r.Put(meta)

	got, ok := r.Get(meta.Type)
	if !ok {
		t.Fatal("expected to find meta")
	}
	if got.Table != "users" {
		t.Fatalf("expected table 'users', got '%s'", got.Table)
	}
	if len(got.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(got.Fields))
	}
}

func TestRegistryPutNilMeta(t *testing.T) {
	r := NewRegistry()

	r.Put(nil)

	// Should not panic and nothing should be added
	_, ok := r.Get(reflect.TypeOf(struct{}{}))
	if ok {
		t.Fatal("should not have any entries")
	}
}

func TestRegistryPutNilType(t *testing.T) {
	r := NewRegistry()

	meta := &EntityMeta{
		Type:  nil,
		Table: "users",
	}

	r.Put(meta)

	// Should not panic and nothing should be added
	_, ok := r.Get(reflect.TypeOf(struct{}{}))
	if ok {
		t.Fatal("should not have any entries")
	}
}

func TestRegistryPutMultiple(t *testing.T) {
	r := NewRegistry()

	type User struct{}
	type Order struct{}
	type Product struct{}

	r.Put(&EntityMeta{Type: reflect.TypeOf(User{}), Table: "users"})
	r.Put(&EntityMeta{Type: reflect.TypeOf(Order{}), Table: "orders"})
	r.Put(&EntityMeta{Type: reflect.TypeOf(Product{}), Table: "products"})

	if _, ok := r.Get(reflect.TypeOf(User{})); !ok {
		t.Fatal("expected to find User meta")
	}
	if _, ok := r.Get(reflect.TypeOf(Order{})); !ok {
		t.Fatal("expected to find Order meta")
	}
	if _, ok := r.Get(reflect.TypeOf(Product{})); !ok {
		t.Fatal("expected to find Product meta")
	}
}

func TestRegistryOverwrite(t *testing.T) {
	r := NewRegistry()

	type User struct{}

	meta1 := &EntityMeta{Type: reflect.TypeOf(User{}), Table: "users"}
	meta2 := &EntityMeta{Type: reflect.TypeOf(User{}), Table: "users_v2"}

	r.Put(meta1)
	r.Put(meta2) // Overwrite

	got, _ := r.Get(reflect.TypeOf(User{}))
	if got.Table != "users_v2" {
		t.Fatalf("expected 'users_v2', got '%s'", got.Table)
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewRegistry()

	type User struct{}

	var wg sync.WaitGroup
	const goroutines = 100

	// Concurrent writes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			meta := &EntityMeta{
				Type:  reflect.TypeOf(User{}),
				Table: "users",
				Fields: []FieldMeta{
					{Name: "ID", Column: "id", Type: reflect.TypeOf(0)},
				},
			}
			r.Put(meta)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Get(reflect.TypeOf(User{}))
		}()
	}

	wg.Wait()

	// Should not panic and should have exactly one entry
	_, ok := r.Get(reflect.TypeOf(User{}))
	if !ok {
		t.Fatal("expected to find User meta after concurrent access")
	}
}

func TestRegistryConcurrentMixed(t *testing.T) {
	r := NewRegistry()

	type Entity1 struct{ ID int }
	type Entity2 struct{ ID int }
	type Entity3 struct{ ID int }

	var wg sync.WaitGroup
	const iterations = 50

	// Mixed operations on different types
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			switch i % 3 {
			case 0:
				r.Put(&EntityMeta{Type: reflect.TypeOf(Entity1{}), Table: "entity1s"})
				r.Get(reflect.TypeOf(Entity1{}))
			case 1:
				r.Put(&EntityMeta{Type: reflect.TypeOf(Entity2{}), Table: "entity2s"})
				r.Get(reflect.TypeOf(Entity2{}))
			case 2:
				r.Put(&EntityMeta{Type: reflect.TypeOf(Entity3{}), Table: "entity3s"})
				r.Get(reflect.TypeOf(Entity3{}))
			}
		}(i)
	}

	wg.Wait()
}

func TestEntityMetaFields(t *testing.T) {
	meta := &EntityMeta{
		Type:  reflect.TypeOf(struct{ Name string }{}),
		Table: "test_table",
		Fields: []FieldMeta{
			{Name: "Name", Column: "name", Type: reflect.TypeOf(""), Index: true},
		},
	}

	if meta.Type == nil {
		t.Fatal("expected non-nil Type")
	}
	if meta.Table != "test_table" {
		t.Fatalf("expected 'test_table', got '%s'", meta.Table)
	}
	if len(meta.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(meta.Fields))
	}
	if !meta.Fields[0].Index {
		t.Fatal("expected Index to be true")
	}
}

func TestFieldMeta(t *testing.T) {
	fm := FieldMeta{
		Name:   "UserName",
		Column: "user_name",
		Type:   reflect.TypeOf(""),
		Index:  true,
	}

	if fm.Name != "UserName" {
		t.Fatalf("expected 'UserName', got '%s'", fm.Name)
	}
	if fm.Column != "user_name" {
		t.Fatalf("expected 'user_name', got '%s'", fm.Column)
	}
	if fm.Type != reflect.TypeOf("") {
		t.Fatal("expected string type")
	}
	if !fm.Index {
		t.Fatal("expected Index to be true")
	}
}

func TestColumnInfo(t *testing.T) {
	ci := ColumnInfo{
		Name: "id",
		Type: "INT",
	}

	if ci.Name != "id" {
		t.Fatalf("expected 'id', got '%s'", ci.Name)
	}
	if ci.Type != "INT" {
		t.Fatalf("expected 'INT', got '%s'", ci.Type)
	}
}

func TestIndexInfo(t *testing.T) {
	idx := IndexInfo{
		Name:     "idx_users_name",
		Table:    "users",
		Columns:  []string{"name"},
		Unique:   false,
		Method:   "BTREE",
		IsBTree:  true,
		IsVisible: true,
	}

	if idx.Name != "idx_users_name" {
		t.Fatalf("expected 'idx_users_name', got '%s'", idx.Name)
	}
	if idx.Table != "users" {
		t.Fatalf("expected 'users', got '%s'", idx.Table)
	}
	if len(idx.Columns) != 1 || idx.Columns[0] != "name" {
		t.Fatalf("expected ['name'], got %v", idx.Columns)
	}
	if idx.Unique {
		t.Fatal("expected Unique to be false")
	}
	if idx.Method != "BTREE" {
		t.Fatalf("expected Method 'BTREE', got '%s'", idx.Method)
	}
	if !idx.IsBTree {
		t.Fatal("expected IsBTree to be true")
	}
	if !idx.IsVisible {
		t.Fatal("expected IsVisible to be true")
	}
}

func TestIndexInfoDefaults(t *testing.T) {
	idx := IndexInfo{}

	if idx.Name != "" {
		t.Fatalf("expected empty name, got '%s'", idx.Name)
	}
	if idx.Unique {
		t.Fatal("expected Unique to be false by default")
	}
	if idx.IsBTree {
		t.Fatal("expected IsBTree to be false by default")
	}
	if idx.IsVisible {
		t.Fatal("expected IsVisible to be false by default")
	}
}

func TestRegistryEmptyType(t *testing.T) {
	r := NewRegistry()

	// Using a named type vs anonymous
	type User struct{ Name string }
	userType := reflect.TypeOf(User{})

	r.Put(&EntityMeta{Type: userType, Table: "users"})

	got, ok := r.Get(userType)
	if !ok {
		t.Fatal("expected to find meta by named type")
	}
	if got.Table != "users" {
		t.Fatalf("expected 'users', got '%s'", got.Table)
	}
}

func TestRegistryPointerVsValue(t *testing.T) {
	r := NewRegistry()

	type User struct{ Name string }

	// Put with pointer type
	r.Put(&EntityMeta{Type: reflect.TypeOf(&User{}), Table: "users_ptr"})

	// Get with pointer type - should work
	_, ok := r.Get(reflect.TypeOf(&User{}))
	if !ok {
		t.Fatal("expected to find meta with pointer type")
	}

	// Get with value type - should not work
	_, ok = r.Get(reflect.TypeOf(User{}))
	if ok {
		t.Fatal("should not find meta with value type when pointer was stored")
	}
}

func TestRegistryWithEmptyFields(t *testing.T) {
	r := NewRegistry()

	meta := &EntityMeta{
		Type:   reflect.TypeOf(struct{}{}),
		Table:  "empty",
		Fields: []FieldMeta{},
	}

	r.Put(meta)

	got, ok := r.Get(meta.Type)
	if !ok {
		t.Fatal("expected to find meta")
	}
	if len(got.Fields) != 0 {
		t.Fatalf("expected 0 fields, got %d", len(got.Fields))
	}
}

func TestRegistryWithManyFields(t *testing.T) {
	r := NewRegistry()

	meta := &EntityMeta{
		Type:  reflect.TypeOf(struct{}{}),
		Table: "many_fields",
		Fields: make([]FieldMeta, 100),
	}

	for i := 0; i < 100; i++ {
		meta.Fields[i] = FieldMeta{
			Name:   "Field",
			Column: "field",
			Type:   reflect.TypeOf(""),
		}
	}

	r.Put(meta)

	got, ok := r.Get(meta.Type)
	if !ok {
		t.Fatal("expected to find meta")
	}
	if len(got.Fields) != 100 {
		t.Fatalf("expected 100 fields, got %d", len(got.Fields))
	}
}