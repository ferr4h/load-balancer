package api

import (
	"encoding/json"
	"load-balancer/balancer"
	"load-balancer/logger"
	"net/http"
)

type API struct {
	RateLimiter *balancer.RateLimiter
}

type ClientConfig struct {
	ClientID   string `json:"client_id"`
	Capacity   int    `json:"capacity"`
	RatePerSec int    `json:"rate_per_sec"`
}

// POST /clients
func (api *API) AddClient(w http.ResponseWriter, r *http.Request) {
	var cfg ClientConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	api.RateLimiter.Cfgs[cfg.ClientID] = balancer.Config{
		Capacity:   cfg.Capacity,
		RatePerSec: cfg.RatePerSec,
	}
	logger.L.Printf("Added client config: %+v", cfg)
	w.WriteHeader(http.StatusOK)
}

// DELETE /clients/{id}
func (api *API) DeleteClient(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/clients/"):]
	delete(api.RateLimiter.Cfgs, id)
	delete(api.RateLimiter.Buckets, id)
	logger.L.Printf("Deleted client %s", id)
	w.WriteHeader(http.StatusOK)
}
