package tests

import (
	"sync"
	"testing"

	"gore/internal/tracker"
)

type TrackUser struct {
	ID   int
	Name string
}

type TrackOrder struct {
	ID     int
	Amount int
	Status string
}

func TestTrackerNew(t *testing.T) {
	tr := tracker.New()
	if tr == nil {
		t.Fatal("expected non-nil tracker")
	}
	entries := tr.Entries()
	if len(entries) != 0 {
		t.Fatalf("expected 0 initial entries, got %d", len(entries))
	}
}

func TestTrackerAttach(t *testing.T) {
	tr := tracker.New()
	user := &TrackUser{ID: 1, Name: "Alice"}

	entry, err := tr.Attach(user)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	if entry.State != tracker.StateUnchanged {
		t.Fatalf("expected unchanged state after attach, got %v", entry.State)
	}
	if entry.Snapshot == nil {
		t.Fatal("expected non-nil snapshot")
	}
}

func TestTrackerAttachNilPointer(t *testing.T) {
	tr := tracker.New()
	_, err := tr.Attach(nil)
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestTrackerAttachNonPointer(t *testing.T) {
	tr := tracker.New()
	user := TrackUser{ID: 1, Name: "Alice"}
	_, err := tr.Attach(user)
	if err == nil {
		t.Fatal("expected error for non-pointer")
	}
}

func TestTrackerMarkAdded(t *testing.T) {
	tr := tracker.New()
	order := &TrackOrder{ID: 10, Amount: 100, Status: "new"}

	entry, err := tr.MarkAdded(order)
	if err != nil {
		t.Fatalf("mark added failed: %v", err)
	}
	if entry.State != tracker.StateAdded {
		t.Fatalf("expected added state, got %v", entry.State)
	}
}

func TestTrackerMarkAddedNilPointer(t *testing.T) {
	tr := tracker.New()
	_, err := tr.MarkAdded(nil)
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestTrackerMarkDeleted(t *testing.T) {
	tr := tracker.New()
	user := &TrackUser{ID: 1, Name: "Alice"}

	entry, err := tr.MarkDeleted(user)
	if err != nil {
		t.Fatalf("mark deleted failed: %v", err)
	}
	if entry.State != tracker.StateDeleted {
		t.Fatalf("expected deleted state, got %v", entry.State)
	}
}

func TestTrackerMarkDeletedNilPointer(t *testing.T) {
	tr := tracker.New()
	_, err := tr.MarkDeleted(nil)
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestTrackerDetectChanges(t *testing.T) {
	tr := tracker.New()
	user := &TrackUser{ID: 1, Name: "Alice"}

	if _, err := tr.Attach(user); err != nil {
		t.Fatalf("attach failed: %v", err)
	}

	user.Name = "Bob"

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change entry, got %d", len(changes))
	}
	if changes[0].State != tracker.StateModified {
		t.Fatalf("expected modified state, got %v", changes[0].State)
	}
}

func TestTrackerDetectChangesNoChanges(t *testing.T) {
	tr := tracker.New()
	user := &TrackUser{ID: 1, Name: "Alice"}

	tr.Attach(user)

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("expected 0 changes, got %d", len(changes))
	}
}

func TestTrackerDetectChangesUnchanged(t *testing.T) {
	tr := tracker.New()
	user := &TrackUser{ID: 1, Name: "Alice"}

	tr.Attach(user)
	tr.DetectChanges() // First detection

	// Re-detect without changes
	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("expected 0 changes on re-detect, got %d", len(changes))
	}
}

func TestTrackerDetectChangesAdded(t *testing.T) {
	tr := tracker.New()
	order := &TrackOrder{ID: 10, Amount: 100, Status: "new"}

	tr.MarkAdded(order)

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].State != tracker.StateAdded {
		t.Fatalf("expected added state, got %v", changes[0].State)
	}
}

func TestTrackerDetectChangesDeleted(t *testing.T) {
	tr := tracker.New()
	user := &TrackUser{ID: 1, Name: "Alice"}

	tr.MarkDeleted(user)

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].State != tracker.StateDeleted {
		t.Fatalf("expected deleted state, got %v", changes[0].State)
	}
}

func TestTrackerClear(t *testing.T) {
	tr := tracker.New()
	user := &TrackUser{ID: 1, Name: "Alice"}
	tr.Attach(user)

	tr.Clear()

	entries := tr.Entries()
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestTrackerEntries(t *testing.T) {
	tr := tracker.New()
	u1 := &TrackUser{ID: 1, Name: "Alice"}
	u2 := &TrackUser{ID: 2, Name: "Bob"}

	tr.Attach(u1)
	tr.Attach(u2)

	entries := tr.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestTrackerEntriesAfterDelete(t *testing.T) {
	tr := tracker.New()
	u1 := &TrackUser{ID: 1, Name: "Alice"}
	u2 := &TrackUser{ID: 2, Name: "Bob"}

	tr.Attach(u1)
	tr.Attach(u2)
	tr.MarkDeleted(u1)

	entries := tr.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (including deleted), got %d", len(entries))
	}
}

func TestTrackerComplexStates(t *testing.T) {
	tr := tracker.New()

	u1 := &TrackUser{ID: 1, Name: "Alice"}
	u2 := &TrackUser{ID: 2, Name: "Bob"}
	o1 := &TrackOrder{ID: 10, Amount: 100, Status: "new"}

	if _, err := tr.Attach(u1); err != nil {
		t.Fatalf("attach u1 failed: %v", err)
	}
	if _, err := tr.Attach(u2); err != nil {
		t.Fatalf("attach u2 failed: %v", err)
	}
	if _, err := tr.MarkAdded(o1); err != nil {
		t.Fatalf("mark added failed: %v", err)
	}
	if _, err := tr.MarkDeleted(u2); err != nil {
		t.Fatalf("mark deleted failed: %v", err)
	}

	u1.Name = "Alice-Updated"

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(changes))
	}

	var modified, added, deleted int
	for _, entry := range changes {
		switch entry.State {
		case tracker.StateModified:
			modified++
		case tracker.StateAdded:
			added++
		case tracker.StateDeleted:
			deleted++
		}
	}

	if modified != 1 || added != 1 || deleted != 1 {
		t.Fatalf("unexpected states: modified=%d added=%d deleted=%d", modified, added, deleted)
	}
}

func TestTrackerConcurrentIndependent(t *testing.T) {
	const workers = 6
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				tr := tracker.New()
				user := &TrackUser{ID: id, Name: "User"}
				if _, err := tr.Attach(user); err != nil {
					t.Errorf("attach failed: %v", err)
					return
				}
				user.Name = "User-Updated"
				if _, err := tr.DetectChanges(); err != nil {
					t.Errorf("detect failed: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestTrackerMultipleFieldsModified(t *testing.T) {
	tr := tracker.New()
	user := &TrackUser{ID: 1, Name: "Alice"}

	tr.Attach(user)
	user.ID = 2
	user.Name = "Bob"

	changes, err := tr.DetectChanges()
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change entry, got %d", len(changes))
	}
	if changes[0].State != tracker.StateModified {
		t.Fatalf("expected modified state, got %v", changes[0].State)
	}
	if changes[0].Changes == nil {
		t.Fatal("expected non-nil changes map")
	}
}

func TestTrackerAttachSameEntityTwice(t *testing.T) {
	tr := tracker.New()
	user := &TrackUser{ID: 1, Name: "Alice"}

	tr.Attach(user)
	tr.Attach(user) // Second attach should be idempotent

	entries := tr.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}
