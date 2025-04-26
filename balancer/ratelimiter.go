package balancer

import (
	"sync"
	"time"
)

type Bucket struct {
	Capacity   int
	Tokens     int
	Rate       int
	LastRefill time.Time
	mu         sync.Mutex
}

type RateLimiter struct {
	Buckets map[string]*Bucket
	Cfgs    map[string]Config
	Default Config
	mu      sync.RWMutex
}

type Config struct {
	Capacity   int
	RatePerSec int
}

func NewRateLimiter(cfgs map[string]Config, defCfg Config) *RateLimiter {
	return &RateLimiter{
		Buckets: make(map[string]*Bucket),
		Cfgs:    cfgs,
		Default: defCfg,
	}
}

func (rl *RateLimiter) Allow(client string) bool {
	rl.mu.RLock()
	bkt, exist := rl.Buckets[client]
	rl.mu.RUnlock()

	if !exist {
		rl.mu.Lock()
		cfg, ok := rl.Cfgs[client]
		if !ok {
			cfg = rl.Default
		}
		bkt = &Bucket{
			Capacity:   cfg.Capacity,
			Tokens:     cfg.Capacity,
			Rate:       cfg.RatePerSec,
			LastRefill: time.Now(),
		}
		rl.Buckets[client] = bkt
		rl.mu.Unlock()
	}

	bkt.mu.Lock()
	defer bkt.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(bkt.LastRefill).Seconds()
	newTokens := int(elapsed * float64(bkt.Rate))
	if newTokens > 0 {
		bkt.Tokens = min(bkt.Capacity, bkt.Tokens+newTokens)
		bkt.LastRefill = now
	}
	if bkt.Tokens > 0 {
		bkt.Tokens--
		return true
	}
	return false
}
