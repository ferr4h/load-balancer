package balancer

import (
	"sync/atomic"
)

// Backend represents a single backend server with its availability status.
// Uses atomic.Bool for thread-safe Online status checks without locks.
type Backend struct {
	URL    string
	Online atomic.Bool
}

// Balancer interface defines core load balancing operations.
// Сan implement different algorithms
type Balancer interface {
	Next() *Backend
	All() []*Backend
	MarkDown(url string)
	MarkUp(url string)
}

// RoundRobin implements Balancer
type RoundRobin struct {
	backends []*Backend
	last     uint64
}

// New creates a RoundRobin balancer with initial backends
// Pre-stores all backends as Online=true by default
func New(backends []string) *RoundRobin {
	bs := make([]*Backend, 0)
	for _, b := range backends {
		bs = append(bs, &Backend{URL: b})
		bs[len(bs)-1].Online.Store(true)
	}
	return &RoundRobin{backends: bs}
}

// Next returns an available backend using round-robin selection
// Implements starvation-free fallthrough: if current backend is down,
// continues searching until finding an available one.
// Returns nil if no backends available
func (r *RoundRobin) Next() *Backend {
	cnt := uint64(len(r.backends))
	start := atomic.AddUint64(&r.last, 1)

	// Linear probing fallback for failed nodes
	for i := uint64(0); i < cnt; i++ {
		b := r.backends[(start+i)%cnt]
		if b.Online.Load() {
			return b
		}
	}
	return nil // All backends down
}

// All returns all backends (including offline ones)
func (r *RoundRobin) All() []*Backend { return r.backends }

// MarkDown takes a backend offline by URL
func (r *RoundRobin) MarkDown(url string) {
	for _, b := range r.backends {
		if b.URL == url {
			b.Online.Store(false)
		}
	}
}

// MarkUp brings a backend online by URL
func (r *RoundRobin) MarkUp(url string) {
	for _, b := range r.backends {
		if b.URL == url {
			b.Online.Store(true)
		}
	}
}
