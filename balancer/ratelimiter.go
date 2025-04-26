package balancer

import (
	"sync"
	"time"
)

// Bucket represents a token bucket for rate limiting.
// Uses mutex-protected fields for thread-safe operations.
type Bucket struct {
	Capacity   int        // Maximum token capacity
	Tokens     int        // Current available tokens
	Rate       int        // Tokens replenished per second
	LastRefill time.Time  // Last refill timestamp
	mu         sync.Mutex // Protects all mutable state
}

// RateLimiter manages client-specific token buckets
type RateLimiter struct {
	Buckets map[string]*Bucket // Client → bucket mapping
	Cfgs    map[string]Config  // Client-specific configurations
	Default Config             // Fallback configuration
	mu      sync.RWMutex       // Protects Buckets and Cfgs maps
}

// Config defines rate limiting parameters for a client
type Config struct {
	Capacity   int // Maximum burst capacity
	RatePerSec int // Sustained requests per second
}

// NewRateLimiter creates a RateLimiter with predefined configurations
func NewRateLimiter(cfgs map[string]Config, defCfg Config) *RateLimiter {
	return &RateLimiter{
		Buckets: make(map[string]*Bucket),
		Cfgs:    cfgs,
		Default: defCfg,
	}
}

// Allow checks if a request is permitted by the rate limiter
func (rl *RateLimiter) Allow(client string) bool {
	// Fast path: check existing bucket
	rl.mu.RLock()
	bkt, exist := rl.Buckets[client]
	rl.mu.RUnlock()

	// Slow path: initialize new bucket
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

	// Token refill calculation
	now := time.Now()
	elapsed := now.Sub(bkt.LastRefill).Seconds()
	newTokens := int(elapsed * float64(bkt.Rate))

	if newTokens > 0 {
		bkt.Tokens = min(bkt.Capacity, bkt.Tokens+newTokens)
		bkt.LastRefill = now // Reset refill timer
	}

	// Token consumption
	if bkt.Tokens > 0 {
		bkt.Tokens--
		return true
	}
	return false
}
