package balancer

import (
	"encoding/json"
	"load-balancer/logger"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	Balancer    Balancer
	RateLimiter *RateLimiter
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientID := r.RemoteAddr
	if !p.RateLimiter.Allow(clientID) {
		writeJSONError(w, 429, "Rate limit exceeded")
		logger.L.Printf("Rate limit exceeded for %s", clientID)
		return
	}
	backend := p.Balancer.Next()
	if backend == nil {
		writeJSONError(w, 503, "No backends available")
		logger.L.Printf("All backends are down")
		return
	}
	remote, _ := url.Parse(backend.URL)
	proxy := httputil.NewSingleHostReverseProxy(remote)
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
