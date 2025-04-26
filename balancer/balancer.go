package balancer

import (
	"sync/atomic"
)

type Backend struct {
	URL    string
	Online atomic.Bool
}

type Balancer interface {
	Next() *Backend
	All() []*Backend
	MarkDown(url string)
	MarkUp(url string)
}

type RoundRobin struct {
	backends []*Backend
	last     uint64
}

func New(backends []string) *RoundRobin {
	bs := make([]*Backend, 0)
	for _, b := range backends {
		bs = append(bs, &Backend{URL: b})
		bs[len(bs)-1].Online.Store(true)
	}
	return &RoundRobin{backends: bs}
}

func (r *RoundRobin) Next() *Backend {
	cnt := uint64(len(r.backends))
	start := atomic.AddUint64(&r.last, 1)
	for i := uint64(0); i < cnt; i++ {
		b := r.backends[(start+i)%cnt]
		if b.Online.Load() {
			return b
		}
	}
	return nil
}
func (r *RoundRobin) All() []*Backend { return r.backends }
func (r *RoundRobin) MarkDown(url string) {
	for _, b := range r.backends {
		if b.URL == url {
			b.Online.Store(false)
		}
	}
}
func (r *RoundRobin) MarkUp(url string) {
	for _, b := range r.backends {
		if b.URL == url {
			b.Online.Store(true)
		}
	}
}
