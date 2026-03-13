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
