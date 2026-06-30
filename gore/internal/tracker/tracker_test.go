package tracker

import (
	"sync"
	"testing"

	goreerrors "gore/internal/errors"
)

type trackTestUser struct {
	ID   int
	Name string
}

type trackTestOrder struct {
	ID      int
	Amount  float64
	Status  string
	Deleted bool
}

type trackEmptyStruct struct{}

type trackUnexportedFields struct {
	id   int
	Name string
}

type trackMixedFields struct {
	Exported   int
	unexported int
}

// ===============================================================================
// Basic Tests
// ===============================================================================

func TestNew(t *testing.T) {
	tr := New()
	if tr == nil {
		t.Fatal("expected non-nil tracker")
	}
	if tr.entries == nil {
		t.Fatal("expected non-nil entries map")
	}
}

func TestNewCreatesEmptyEntries(t *testing.T) {
	tr := New()
	if len(tr.Entries()) != 0 {
		t.Fatal("expected 0 initial entries")
	}
}

// ===============================================================================
// Attach Boundary Tests
// ===============================================================================

func TestAttachNilEntity(t *testing.T) {
	tr := New()
	_, err := tr.Attach(nil)
	if err == nil {
		t.Fatal("expected error for nil entity")
	}
	if err != goreerrors.ErrNilEntity {
		t.Fatalf("expected ErrNilEntity, got %v", err)
	}
}

func TestAttachNonPointer(t *testing.T) {
	tr := New()
	user := trackTestUser{ID: 1, Name: "Alice"}
	_, err := tr.Attach(user)
	if err == nil {
		t.Fatal("expected error for non-pointer")
	}
	if err != goreerrors.ErrNilEntity {
		t.Fatalf("expected ErrNilEntity, got %v", err)
	}
}

func TestAttachNilPointer(t *testing.T) {
	tr := New()
	var user *trackTestUser
	_, err := tr.Attach(user)
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
	if err != goreerrors.ErrNilEntity {
		t.Fatalf("expected ErrNilEntity, got %v", err)
	}
}

func TestAttachPointerToPointer(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}
	_, err := tr.Attach(&user) // pointer to pointer
	if err == nil {
		t.Fatal("expected error for pointer to pointer")
	}
	if err != goreerrors.ErrInvalidEntity {
		t.Fatalf("expected ErrInvalidEntity, got %v", err)
	}
}

func TestAttachValueTypeNotStruct(t *testing.T) {
	tr := New()
	str := "not a struct"
	_, err := tr.Attach(&str)
	if err == nil {
		t.Fatal("expected error for non-struct pointer")
	}
	if err != goreerrors.ErrInvalidEntity {
		t.Fatalf("expected ErrInvalidEntity, got %v", err)
	}
}

func TestAttachEmptyStruct(t *testing.T) {
	tr := New()
	entity := &trackEmptyStruct{}
	entry, err := tr.Attach(entity)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if entry.State != StateUnchanged {
		t.Fatalf("expected StateUnchanged, got %v", entry.State)
	}
	if entry.Snapshot == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if len(entry.Snapshot) != 0 {
		t.Fatalf("expected empty snapshot for empty struct, got %d fields", len(entry.Snapshot))
	}
}

func TestAttachUnexportedFields(t *testing.T) {
	tr := New()
	entity := &trackUnexportedFields{id: 1, Name: "Alice"}
	entry, err := tr.Attach(entity)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	// Only Name should be in snapshot (exported field)
	if len(entry.Snapshot) != 1 {
		t.Fatalf("expected 1 field in snapshot (only exported), got %d", len(entry.Snapshot))
	}
	if _, ok := entry.Snapshot["Name"]; !ok {
		t.Fatal("expected Name in snapshot")
	}
	if _, ok := entry.Snapshot["id"]; ok {
		t.Fatal("did not expect unexported 'id' in snapshot")
	}
}

func TestAttachMixedFields(t *testing.T) {
	tr := New()
	entity := &trackMixedFields{Exported: 1, unexported: 2}
	entry, err := tr.Attach(entity)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	// Only Exported should be in snapshot
	if len(entry.Snapshot) != 1 {
		t.Fatalf("expected 1 field in snapshot, got %d", len(entry.Snapshot))
	}
	if entry.Snapshot["Exported"] != 1 {
		t.Fatal("expected Exported=1 in snapshot")
	}
}

// ===============================================================================
// MarkAdded Boundary Tests
// ===============================================================================

func TestMarkAddedNilEntity(t *testing.T) {
	tr := New()
	_, err := tr.MarkAdded(nil)
	if err == nil {
		t.Fatal("expected error for nil entity")
	}
	if err != goreerrors.ErrNilEntity {
		t.Fatalf("expected ErrNilEntity, got %v", err)
	}
}

func TestMarkAddedNonPointer(t *testing.T) {
	tr := New()
	order := trackTestOrder{ID: 1, Amount: 100.0}
	_, err := tr.MarkAdded(order)
	if err == nil {
		t.Fatal("expected error for non-pointer")
	}
}

func TestMarkAddedNilPointer(t *testing.T) {
	tr := New()
	var order *trackTestOrder
	_, err := tr.MarkAdded(order)
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

// ===============================================================================
// MarkDeleted Boundary Tests
// ===============================================================================

func TestMarkDeletedNilEntity(t *testing.T) {
	tr := New()
	_, err := tr.MarkDeleted(nil)
	if err == nil {
		t.Fatal("expected error for nil entity")
	}
	if err != goreerrors.ErrNilEntity {
		t.Fatalf("expected ErrNilEntity, got %v", err)
	}
}

func TestMarkDeletedNonPointer(t *testing.T) {
	tr := New()
	user := trackTestUser{ID: 1}
	_, err := tr.MarkDeleted(user)
	if err == nil {
		t.Fatal("expected error for non-pointer")
	}
}

func TestMarkDeletedNilPointer(t *testing.T) {
	tr := New()
	var user *trackTestUser
	_, err := tr.MarkDeleted(user)
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestMarkDeletedWithoutAttach(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}
	entry, err := tr.MarkDeleted(user)
	if err != nil {
		t.Fatalf("mark deleted failed: %v", err)
	}
	if entry.State != StateDeleted {
		t.Fatalf("expected StateDeleted, got %v", entry.State)
	}
}

// ===============================================================================
// DetectChanges Boundary Tests
// ===============================================================================

func TestDetectChangesEmptyTracker(t *testing.T) {
	tr := New()
	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("expected 0 changes, got %d", len(changes))
	}
}

func TestDetectChangesNoModification(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}
	tr.Attach(user)

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("expected 0 changes, got %d", len(changes))
	}
}

func TestDetectChangesAllFieldTypes(t *testing.T) {
	tr := New()
	order := &trackTestOrder{
		ID:      1,
		Amount:  100.50,
		Status:  "pending",
		Deleted: false,
	}
	tr.Attach(order)

	order.Amount = 200.75
	order.Status = "completed"
	order.Deleted = true

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change entry, got %d", len(changes))
	}
	if changes[0].State != StateModified {
		t.Fatalf("expected StateModified, got %v", changes[0].State)
	}
	// All three modified fields should be in Changes
	if len(changes[0].Changes) != 3 {
		t.Fatalf("expected 3 changed fields, got %d", len(changes[0].Changes))
	}
}

func TestDetectChangesIntegerOverflow(t *testing.T) {
	tr := New()
	type IntTest struct {
		Val int
	}
	entity := &IntTest{Val: 2147483647} // max int32
	tr.Attach(entity)

	entity.Val = -2147483648 // min int32

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change entry, got %d", len(changes))
	}
}

func TestDetectChangesFloatPrecision(t *testing.T) {
	tr := New()
	type FloatTest struct {
		Val float64
	}
	entity := &FloatTest{Val: 0.1}
	tr.Attach(entity)

	entity.Val = 0.2

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	// 0.1 != 0.2 should be detected as change
	if len(changes) != 1 {
		t.Fatalf("expected 1 change entry, got %d", len(changes))
	}
}

func TestDetectChangesBooleanToggle(t *testing.T) {
	tr := New()
	type BoolTest struct {
		Active bool
	}
	entity := &BoolTest{Active: false}
	tr.Attach(entity)

	entity.Active = true

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change entry, got %d", len(changes))
	}
	if changes[0].Changes["Active"] != true {
		t.Fatal("expected Active=true in changes")
	}
}

func TestDetectChangesStringEmptyToValue(t *testing.T) {
	tr := New()
	type StringTest struct {
		Name string
	}
	entity := &StringTest{Name: ""}
	tr.Attach(entity)

	entity.Name = "Alice"

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change entry, got %d", len(changes))
	}
	if changes[0].Changes["Name"] != "Alice" {
		t.Fatal("expected Name='Alice' in changes")
	}
}

func TestDetectChangesStringValueToEmpty(t *testing.T) {
	tr := New()
	type StringTest struct {
		Name string
	}
	entity := &StringTest{Name: "Alice"}
	tr.Attach(entity)

	entity.Name = ""

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change entry, got %d", len(changes))
	}
	if changes[0].Changes["Name"] != "" {
		t.Fatal("expected Name='' in changes")
	}
}

func TestDetectChangesBackAndForth(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}
	tr.Attach(user)

	user.Name = "Bob"
	changes1, _ := tr.DetectChanges()
	if len(changes1) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes1))
	}

	user.Name = "Alice" // Change back to original
	// Snapshot is still original "Alice", so no change from snapshot
	changes2, _ := tr.DetectChanges()
	if len(changes2) != 0 {
		t.Fatalf("expected 0 changes after reverting to original, got %d", len(changes2))
	}
}

func TestDetectChangesMultipleEntities(t *testing.T) {
	tr := New()
	u1 := &trackTestUser{ID: 1, Name: "Alice"}
	u2 := &trackTestUser{ID: 2, Name: "Bob"}
	u3 := &trackTestUser{ID: 3, Name: "Charlie"}

	tr.Attach(u1)
	tr.Attach(u2)
	tr.Attach(u3)

	u1.Name = "Alice-Modified"
	u3.Name = "Charlie-Modified"

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
}

// ===============================================================================
// Clear Tests
// ===============================================================================

func TestClearEmptyTracker(t *testing.T) {
	tr := New()
	tr.Clear() // Should not panic

	if len(tr.Entries()) != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", len(tr.Entries()))
	}
}

func TestClearAfterMultipleOperations(t *testing.T) {
	tr := New()
	tr.Attach(&trackTestUser{ID: 1, Name: "A"})
	tr.Attach(&trackTestUser{ID: 2, Name: "B"})
	tr.MarkAdded(&trackTestOrder{ID: 1, Amount: 100.0})

	tr.Clear()

	if len(tr.Entries()) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(tr.Entries()))
	}
}

func TestClearThenAdd(t *testing.T) {
	tr := New()
	tr.Attach(&trackTestUser{ID: 1, Name: "Alice"})
	tr.Clear()
	tr.Attach(&trackTestUser{ID: 2, Name: "Bob"})

	entries := tr.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Entity.(*trackTestUser).Name != "Bob" {
		t.Fatal("expected entity to be Bob")
	}
}

// ===============================================================================
// Entries Tests
// ===============================================================================

func TestEntriesEmpty(t *testing.T) {
	tr := New()
	entries := tr.Entries()
	if entries == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestEntriesOrder(t *testing.T) {
	tr := New()
	u1 := &trackTestUser{ID: 1, Name: "First"}
	u2 := &trackTestUser{ID: 2, Name: "Second"}
	u3 := &trackTestUser{ID: 3, Name: "Third"}

	tr.Attach(u1)
	tr.Attach(u2)
	tr.Attach(u3)

	entries := tr.Entries()
	// Map iteration order is random, so we just verify all 3 are present
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestEntriesIncludesDeleted(t *testing.T) {
	tr := New()
	u1 := &trackTestUser{ID: 1, Name: "Alice"}
	u2 := &trackTestUser{ID: 2, Name: "Bob"}

	tr.Attach(u1)
	tr.Attach(u2)
	tr.MarkDeleted(u1)

	entries := tr.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (including deleted), got %d", len(entries))
	}
}

func TestEntriesCapacity(t *testing.T) {
	tr := New()
	for i := 0; i < 10; i++ {
		tr.Attach(&trackTestUser{ID: i, Name: "User"})
	}

	entries := tr.Entries()
	if cap(entries) < len(entries) {
		t.Fatal("capacity should not be less than length")
	}
}

// ===============================================================================
// ReAttach Tests
// ===============================================================================

func TestReAttachSameEntity(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}

	tr.Attach(user)
	tr.Attach(user) // Re-attach same entity

	entries := tr.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestReAttachAfterModify(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}

	tr.Attach(user)
	user.Name = "Bob"
	tr.DetectChanges()

	user.Name = "Charlie"
	tr.Attach(user) // Re-attach after modification

	entries := tr.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// Snapshot should be reset to new values
	if entries[0].Snapshot["Name"] != "Charlie" {
		t.Fatalf("expected snapshot to have 'Charlie', got '%v'", entries[0].Snapshot["Name"])
	}
}

func TestReAttachAfterDelete(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}

	tr.Attach(user)
	tr.MarkDeleted(user)
	tr.Attach(user) // Re-attach after delete

	entries := tr.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].State != StateUnchanged {
		t.Fatalf("expected StateUnchanged after re-attach, got %v", entries[0].State)
	}
}

// ===============================================================================
// State Transition Tests
// ===============================================================================

func TestStateTransitionAddedToModified(t *testing.T) {
	tr := New()
	order := &trackTestOrder{ID: 1, Amount: 100.0, Status: "new"}

	tr.MarkAdded(order)
	order.Amount = 200.0

	changes, _ := tr.DetectChanges()
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].State != StateAdded {
		t.Fatalf("expected StateAdded to remain unchanged for Added entity, got %v", changes[0].State)
	}
}

func TestStateTransitionModifiedToDeleted(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}

	tr.Attach(user)
	user.Name = "Bob"

	// Mark as deleted while still having changes
	tr.MarkDeleted(user)

	changes, _ := tr.DetectChanges()
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].State != StateDeleted {
		t.Fatalf("expected StateDeleted, got %v", changes[0].State)
	}
}

func TestStateTransitionUnchangedToDeleted(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}

	tr.Attach(user)
	tr.MarkDeleted(user)

	changes, _ := tr.DetectChanges()
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].State != StateDeleted {
		t.Fatalf("expected StateDeleted, got %v", changes[0].State)
	}
}

// ===============================================================================
// Snapshot Tests
// ===============================================================================

func TestSnapshotUnexportedFieldsExcluded(t *testing.T) {
	tr := New()
	entity := &trackUnexportedFields{id: 42, Name: "Alice"}

	entry, _ := tr.Attach(entity)

	// id is unexported, should not be in snapshot
	if _, ok := entry.Snapshot["id"]; ok {
		t.Fatal("unexported field 'id' should not be in snapshot")
	}
	if entry.Snapshot["Name"] != "Alice" {
		t.Fatal("expected Name='Alice' in snapshot")
	}
}

func TestSnapshotOfIntegerTypes(t *testing.T) {
	tr := New()
	type IntTypes struct {
		Int8   int8
		Int16  int16
		Int32  int32
		Int64  int64
		Uint8  uint8
		Uint16 uint16
		Uint32 uint32
		Uint64 uint64
		Int    int
		Uint   uint
	}
	entity := &IntTypes{Int8: -8, Int16: -16, Int32: -32, Int64: -64, Uint8: 8, Uint16: 16, Uint32: 32, Uint64: 64, Int: -1, Uint: 1}
	entry, err := tr.Attach(entity)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if len(entry.Snapshot) != 10 {
		t.Fatalf("expected 10 fields, got %d", len(entry.Snapshot))
	}
}

func TestSnapshotOfFloatTypes(t *testing.T) {
	tr := New()
	type FloatTypes struct {
		Float32 float32
		Float64 float64
	}
	entity := &FloatTypes{Float32: 1.5, Float64: 2.5}
	entry, err := tr.Attach(entity)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if entry.Snapshot["Float32"] != float32(1.5) {
		t.Fatalf("expected Float32=1.5, got %v", entry.Snapshot["Float32"])
	}
	if entry.Snapshot["Float64"] != float64(2.5) {
		t.Fatalf("expected Float64=2.5, got %v", entry.Snapshot["Float64"])
	}
}

func TestSnapshotOfComplexTypes(t *testing.T) {
	tr := New()
	type ComplexTypes struct {
		Bool   bool
		String string
		Byte   byte
		Rune   rune
	}
	entity := &ComplexTypes{Bool: true, String: "test", Byte: 255, Rune: 'A'}
	entry, err := tr.Attach(entity)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if entry.Snapshot["Bool"] != true {
		t.Fatal("expected Bool=true")
	}
	if entry.Snapshot["String"] != "test" {
		t.Fatal("expected String='test'")
	}
	if entry.Snapshot["Byte"] != byte(255) {
		t.Fatalf("expected Byte=255, got %v", entry.Snapshot["Byte"])
	}
	if entry.Snapshot["Rune"] != rune('A') {
		t.Fatalf("expected Rune='A', got %v", entry.Snapshot["Rune"])
	}
}

func TestSnapshotOfSlice(t *testing.T) {
	tr := New()
	type SliceTest struct {
		Slice []int
	}
	entity := &SliceTest{Slice: []int{1, 2, 3}}
	entry, err := tr.Attach(entity)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if entry.Snapshot["Slice"] == nil {
		t.Fatal("expected Slice in snapshot")
	}
}

// ===============================================================================
// Concurrent Tests
// ===============================================================================

func TestConcurrentAttachDifferentEntities(t *testing.T) {
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	tr := New()

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			user := &trackTestUser{ID: id, Name: "User"}
			if _, err := tr.Attach(user); err != nil {
				t.Errorf("attach failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	if len(tr.Entries()) != goroutines {
		t.Fatalf("expected %d entries, got %d", goroutines, len(tr.Entries()))
	}
}

func TestConcurrentModifyDifferentEntities(t *testing.T) {
	const goroutines = 30
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	tr := New()

	// Pre-attach entities
	users := make([]*trackTestUser, goroutines)
	for i := 0; i < goroutines; i++ {
		users[i] = &trackTestUser{ID: i, Name: "Original"}
		tr.Attach(users[i])
	}

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				users[id].Name = "Modified"
				if _, err := tr.DetectChanges(); err != nil {
					t.Errorf("detect failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestConcurrentNewTracker(t *testing.T) {
	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			tr := New()
			tr.Attach(&trackTestUser{ID: 1, Name: "Test"})
			tr.DetectChanges()
			tr.Clear()
		}()
	}

	wg.Wait()
}

// ===============================================================================
// Edge Cases
// ===============================================================================

func TestEntryChangesNilInitially(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}
	entry, _ := tr.Attach(user)

	// Changes should be empty map, not nil
	if entry.Changes == nil {
		t.Fatal("expected non-nil Changes map")
	}
	if len(entry.Changes) != 0 {
		t.Fatalf("expected empty Changes, got %d", len(entry.Changes))
	}
}

func TestDetectChangesWithNilSnapshotEntity(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}
	entry, _ := tr.Attach(user)

	// Manually clear snapshot to simulate edge case
	entry.Snapshot = nil

	_, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect should handle nil snapshot gracefully, got: %v", err)
	}
}

func TestLargeNumberOfChanges(t *testing.T) {
	tr := New()
	type LargeEntity struct {
		Field00, Field01, Field02, Field03, Field04 bool
		Field05, Field06, Field07, Field08, Field09 bool
		Field10, Field11, Field12, Field13, Field14 bool
		Field15, Field16, Field17, Field18, Field19 bool
	}

	entity := &LargeEntity{}
	tr.Attach(entity)

	// Change all boolean fields
	e := entity
	e.Field00, e.Field01, e.Field02, e.Field03, e.Field04 = true, true, true, true, true
	e.Field05, e.Field06, e.Field07, e.Field08, e.Field09 = true, true, true, true, true
	e.Field10, e.Field11, e.Field12, e.Field13, e.Field14 = true, true, true, true, true
	e.Field15, e.Field16, e.Field17, e.Field18, e.Field19 = true, true, true, true, true

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change entry, got %d", len(changes))
	}
	if len(changes[0].Changes) != 20 {
		t.Fatalf("expected 20 changed fields, got %d", len(changes[0].Changes))
	}
}

func TestAttachAfterClear(t *testing.T) {
	tr := New()
	u1 := &trackTestUser{ID: 1, Name: "Alice"}
	u2 := &trackTestUser{ID: 2, Name: "Bob"}

	tr.Attach(u1)
	tr.Clear()
	tr.Attach(u2)

	entries := tr.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Entity.(*trackTestUser).ID != 2 {
		t.Fatal("expected entity ID 2")
	}
}

// ===============================================================================
// EntityState Tests
// ===============================================================================

func TestEntityStateValues(t *testing.T) {
	if StateUnchanged != 0 {
		t.Fatalf("expected StateUnchanged=0, got %d", StateUnchanged)
	}
	if StateAdded != 1 {
		t.Fatalf("expected StateAdded=1, got %d", StateAdded)
	}
	if StateModified != 2 {
		t.Fatalf("expected StateModified=2, got %d", StateModified)
	}
	if StateDeleted != 3 {
		t.Fatalf("expected StateDeleted=3, got %d", StateDeleted)
	}
}

func TestEntryStatePersists(t *testing.T) {
	tr := New()
	user := &trackTestUser{ID: 1, Name: "Alice"}
	entry, _ := tr.Attach(user)

	if entry.State != StateUnchanged {
		t.Fatalf("expected StateUnchanged, got %v", entry.State)
	}

	user.Name = "Bob"
	tr.DetectChanges()

	// State should be updated to Modified
	entries := tr.Entries()
	if entries[0].State != StateModified {
		t.Fatalf("expected StateModified, got %v", entries[0].State)
	}
}