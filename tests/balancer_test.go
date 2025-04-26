package tests

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"load-balancer/balancer"
)

// startTestBackend - mock HTTP server for testing
func startTestBackend() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

// TestRateLimiterIntegration tests the complete rate limiter workflow
func TestRateLimiterIntegration(t *testing.T) {
	// Test environment setup
	configs := map[string]balancer.Config{
		"premium": {Capacity: 10, RatePerSec: 5},
	}
	rl := balancer.NewRateLimiter(configs, balancer.Config{Capacity: 2, RatePerSec: 1})

	// Test 1: Verify allowed request quota
	allowed := 0
	for i := 0; i < 10; i++ {
		if rl.Allow("premium") {
			allowed++
		}
	}
	if allowed != 10 {
		t.Errorf("Expected 10 allowed requests, got %d", allowed)
	}

	// Test 2: Verify rate limiting
	time.Sleep(1 * time.Second)
	if !rl.Allow("premium") {
		t.Error("Expected request to be allowed after refill")
	}
}

// TestBalancerWithRealHTTP tests integration with real HTTP servers
func TestBalancerWithRealHTTP(t *testing.T) {
	// Start 3 test servers
	servers := make([]*httptest.Server, 3)
	for i := range servers {
		servers[i] = startTestBackend()
		defer servers[i].Close()
	}

	// Collect server URLs
	urls := make([]string, len(servers))
	for i, s := range servers {
		urls[i] = s.URL
	}

	b := balancer.New(urls)
	proxy := &balancer.Proxy{
		Balancer:    b,
		RateLimiter: balancer.NewRateLimiter(nil, balancer.Config{Capacity: 100, RatePerSec: 100}),
	}

	// Verify request distribution
	counts := make(map[string]int)
	for i := 0; i < 100; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		proxy.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Unexpected status code: %d", rr.Code)
		}
	}
	for backend, count := range counts {
		if count < 20 || count > 40 { //~33 for each backend
			t.Errorf("Unbalanced distribution: %s got %d requests", backend, count)
		}
	}

	// Test health check functionality
	b.MarkDown(urls[0])
	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		proxy.ServeHTTP(rr, req)
		if rr.Body.String() == urls[0] {
			t.Error("Request was routed to downed backend")
		}
	}
}

// BenchmarkRateLimiter measures rate limiter performance
func BenchmarkRateLimiter(b *testing.B) {
	rl := balancer.NewRateLimiter(nil, balancer.Config{Capacity: 1000000, RatePerSec: 1000000})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.Allow("test-client")
		}
	})
}

// BenchmarkBalancerUnderLoad tests performance under heavy load
func BenchmarkBalancerUnderLoad(b *testing.B) {
	servers := make([]*httptest.Server, 10)
	for i := range servers {
		servers[i] = startTestBackend()
		defer servers[i].Close()
	}

	urls := make([]string, len(servers))
	for i, s := range servers {
		urls[i] = s.URL
	}

	proxy := &balancer.Proxy{
		Balancer:    balancer.New(urls),
		RateLimiter: balancer.NewRateLimiter(nil, balancer.Config{Capacity: 1e6, RatePerSec: 1e6}),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		for pb.Next() {
			proxy.ServeHTTP(rr, req)
		}
	})
}

// TestConcurrentSafety verifies thread-safe operation
func TestConcurrentSafety(t *testing.T) {
	rl := balancer.NewRateLimiter(nil, balancer.Config{Capacity: 100, RatePerSec: 100})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rl.Allow("client-" + strconv.Itoa(id))
			}
		}(i)
	}
	wg.Wait()
}
