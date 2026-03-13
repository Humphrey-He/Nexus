package tracker

import (
	"errors"
	"reflect"
)

// EntityState represents a tracked entity state.
type EntityState int

const (
	StateUnchanged EntityState = iota
	StateAdded
	StateModified
	StateDeleted
)

// Entry holds tracking info for an entity.
type Entry struct {
	Entity   any
	State    EntityState
	Snapshot map[string]any
	Changes  map[string]any
}

// Tracker manages entity snapshots and states.
type Tracker struct {
	entries map[uintptr]*Entry
}

// New creates a Tracker.
func New() *Tracker {
	return &Tracker{
		entries: make(map[uintptr]*Entry),
	}
}

// Attach registers an entity with an initial snapshot.
func (t *Tracker) Attach(entity any) (*Entry, error) {
	ptr, rv, err := entityPtr(entity)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		Entity:   entity,
		State:    StateUnchanged,
		Snapshot: snapshot(rv),
		Changes:  make(map[string]any),
	}

	t.entries[ptr] = entry
	return entry, nil
}

// MarkAdded registers an entity as Added.
func (t *Tracker) MarkAdded(entity any) (*Entry, error) {
	ptr, rv, err := entityPtr(entity)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		Entity:   entity,
		State:    StateAdded,
		Snapshot: snapshot(rv),
		Changes:  make(map[string]any),
	}

	t.entries[ptr] = entry
	return entry, nil
}

// MarkDeleted registers an entity as Deleted.
func (t *Tracker) MarkDeleted(entity any) (*Entry, error) {
	ptr, _, err := entityPtr(entity)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		Entity: entity,
		State:  StateDeleted,
	}

	t.entries[ptr] = entry
	return entry, nil
}

// DetectChanges diffs current entity values against snapshots.
func (t *Tracker) DetectChanges() ([]*Entry, error) {
	var changed []*Entry

	for _, entry := range t.entries {
		if entry.State == StateAdded || entry.State == StateDeleted {
			changed = append(changed, entry)
			continue
		}

		rv := reflect.ValueOf(entry.Entity)
		if rv.Kind() != reflect.Pointer || rv.IsNil() {
			return nil, errors.New("tracker: entity must be non-nil pointer")
		}
		rv = rv.Elem()
		if rv.Kind() != reflect.Struct {
			return nil, errors.New("tracker: entity must point to struct")
		}

		entry.Changes = diffSnapshot(entry.Snapshot, rv)
		if len(entry.Changes) > 0 {
			entry.State = StateModified
			changed = append(changed, entry)
		} else {
			entry.State = StateUnchanged
		}
	}

	return changed, nil
}

// Clear removes all tracked entries.
func (t *Tracker) Clear() {
	t.entries = make(map[uintptr]*Entry)
}

// Entries returns the tracked entries.
func (t *Tracker) Entries() []*Entry {
	out := make([]*Entry, 0, len(t.entries))
	for _, entry := range t.entries {
		out = append(out, entry)
	}
	return out
}

func entityPtr(entity any) (uintptr, reflect.Value, error) {
	rv := reflect.ValueOf(entity)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return 0, reflect.Value{}, errors.New("tracker: entity must be non-nil pointer")
	}
	if rv.Elem().Kind() != reflect.Struct {
		return 0, reflect.Value{}, errors.New("tracker: entity must point to struct")
	}
	return rv.Pointer(), rv.Elem(), nil
}

func snapshot(rv reflect.Value) map[string]any {
	snap := make(map[string]any)
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		snap[field.Name] = rv.Field(i).Interface()
	}
	return snap
}

func diffSnapshot(snap map[string]any, rv reflect.Value) map[string]any {
	changes := make(map[string]any)
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		current := rv.Field(i).Interface()
		if prev, ok := snap[field.Name]; !ok || !reflect.DeepEqual(prev, current) {
			changes[field.Name] = current
		}
	}
	return changes
}
