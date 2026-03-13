package metadata

import (
	"reflect"
	"sync"
)

// Registry caches entity metadata by Go type.
type Registry struct {
	mu    sync.RWMutex
	cache map[reflect.Type]*EntityMeta
}

// NewRegistry creates a metadata registry.
func NewRegistry() *Registry {
	return &Registry{
		cache: make(map[reflect.Type]*EntityMeta),
	}
}

// Get returns cached metadata.
func (r *Registry) Get(t reflect.Type) (*EntityMeta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	meta, ok := r.cache[t]
	return meta, ok
}

// Put stores metadata.
func (r *Registry) Put(meta *EntityMeta) {
	if meta == nil || meta.Type == nil {
		return
	}
	r.mu.Lock()
	r.cache[meta.Type] = meta
	r.mu.Unlock()
}
