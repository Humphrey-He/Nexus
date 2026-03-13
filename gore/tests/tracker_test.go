package tests

import (
	"testing"

	"gore/internal/tracker"
)

type TrackUser struct {
	ID   int
	Name string
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
