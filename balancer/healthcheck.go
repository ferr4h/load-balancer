package balancer

import (
	"load-balancer/logger"
	"net/http"
	"time"
)

// StartHealthChecks initiates background health checks for all backends with configurable frequency
func StartHealthChecks(bal Balancer, freqSec int) {
	go func() {
		for {
			for _, backend := range bal.All() {
				go func(backend *Backend) {
					client := http.Client{Timeout: 2 * time.Second}

					// Simple GET request to standardized health endpoint
					resp, err := client.Get(backend.URL + "/healthz")

					if err != nil || resp.StatusCode != 200 {
						backend.Online.Store(false)
						logger.L.Printf("Backend %s down: %v", backend.URL, err)
						return
					}

					backend.Online.Store(true)
				}(backend)
			}
			time.Sleep(time.Duration(freqSec) * time.Second)
		}
	}()

}
