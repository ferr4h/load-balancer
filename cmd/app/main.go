package main

import (
	"flag"
	"fmt"
	"load-balancer/api"
	"load-balancer/balancer"
	"load-balancer/config"
	"load-balancer/logger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configFile := flag.String("config", "config.yaml", "config file")
	flag.Parse()
	fmt.Println(*configFile)

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Cannot load config: %v", err)
	}
	bal := balancer.New(func() []string {
		var urls []string
		for _, b := range cfg.Backends {
			urls = append(urls, b.URL)
		}
		return urls
	}())
	defLimit := balancer.Config{Capacity: 100, RatePerSec: 10}
	clientLimits := make(map[string]balancer.Config)
	for id, l := range cfg.Clients {
		clientLimits[id] = balancer.Config{
			Capacity:   l.Capacity,
			RatePerSec: l.RatePerSec,
		}
	}
	rl := balancer.NewRateLimiter(clientLimits, defLimit)
	balancer.StartHealthChecks(bal, cfg.CheckFreq)

	prx := &balancer.Proxy{Balancer: bal, RateLimiter: rl}
	apiH := &api.API{RateLimiter: rl}

	mux := http.NewServeMux()
	mux.Handle("/", prx)
	mux.HandleFunc("/clients", apiH.AddClient)
	mux.HandleFunc("/clients/", apiH.DeleteClient)

	srv := &http.Server{Addr: cfg.Listen, Handler: mux}
	go func() {
		logger.L.Printf("Listening on %s", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil {
			logger.L.Println(err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	logger.L.Println("Shutting down...")
	srv.Close()
}
