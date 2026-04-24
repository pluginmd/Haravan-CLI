package tools

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is a concurrent-safe in-memory index of registered Tools.
// The default Registry is exposed via the package-level helpers
// (Register, Get, All, ByCategory) so tool packages can register in
// init() without passing a registry around.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
}

// NewRegistry constructs an empty registry. Most callers should use the
// package-level Default() registry instead.
func NewRegistry() *Registry { return &Registry{tools: map[string]*Tool{}} }

// Register inserts a Tool. Duplicate names panic at startup — they would
// otherwise silently shadow each other and are always bugs.
func (r *Registry) Register(t *Tool) {
	if t == nil {
		panic("tools: cannot register nil tool")
	}
	if t.Name == "" {
		panic("tools: tool with empty Name")
	}
	if t.Handler == nil {
		panic(fmt.Sprintf("tools: tool %q has nil Handler", t.Name))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[t.Name]; exists {
		panic(fmt.Sprintf("tools: duplicate tool %q", t.Name))
	}
	r.tools[t.Name] = t
}

// Get fetches a Tool by name or returns nil.
func (r *Registry) Get(name string) *Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// All returns every registered Tool sorted by name.
func (r *Registry) All() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ByCategory returns tools in the given category, sorted by name.
func (r *Registry) ByCategory(cat Category) []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Tool, 0)
	for _, t := range r.tools {
		if t.Category == cat {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Categories returns every distinct category present in the registry,
// sorted alphabetically.
func (r *Registry) Categories() []Category {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := map[Category]struct{}{}
	for _, t := range r.tools {
		seen[t.Category] = struct{}{}
	}
	out := make([]Category, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Count returns the number of tools in the registry.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// --- default registry -------------------------------------------------------

var defaultRegistry = NewRegistry()

// Default returns the process-wide registry.
func Default() *Registry { return defaultRegistry }

// Register adds a tool to the default registry.
func Register(t *Tool) { defaultRegistry.Register(t) }
