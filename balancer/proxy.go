package balancer

import (
	"encoding/json"
	"load-balancer/logger"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	Balancer    Balancer     // Backend selection strategy (round-robin, etc.)
	RateLimiter *RateLimiter // Client rate limiting controller
}

// ServeHTTP implements http.Handler interface to proxy requests
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Client identification
	clientID := r.RemoteAddr
	if !p.RateLimiter.Allow(clientID) {
		writeJSONError(w, 429, "Rate limit exceeded")
		logger.L.Printf("Rate limit exceeded for %s", clientID)
		return
	}

	// Backend selection
	backend := p.Balancer.Next()
	if backend == nil {
		writeJSONError(w, 503, "No backends available")
		logger.L.Printf("All backends are down")
		return
	}
	remote, _ := url.Parse(backend.URL)

	// Proxy setup
	proxy := httputil.NewSingleHostReverseProxy(remote)

	// Enhanced request rewriting
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = remote.Host
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		p.Balancer.MarkDown(backend.URL)
		writeJSONError(w, 502, "Bad gateway: "+err.Error())
		logger.L.Printf("Backend %s down. Error: %v", backend.URL, err)
	}

	// Request processing
	logger.L.Printf("Proxying %s %s for %s via %s", r.Method, r.URL.Path, clientID, backend.URL)
	proxy.ServeHTTP(w, r)
}
func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": msg,
	})
}
