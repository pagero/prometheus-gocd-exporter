package main

import (
	"expvar"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"
)

func healthcheck(dur time.Duration, ch <-chan struct{}) {
	var (
		mu sync.RWMutex
		t  time.Time
	)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if time.Since(t) > dur {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	})

	expvar.Publish("heartbeat", &t)
	for range ch {
		mu.Lock()
		t = time.Now()
		mu.Unlock()
	}
}
