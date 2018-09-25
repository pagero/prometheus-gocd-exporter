package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/pagero/prometheus-gocd-exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	once := flag.Bool("once", false, "Set to run one scrape and then exit")
	duration := flag.Duration("duration", time.Minute, "The duration between scrapes")
	ns := flag.String("namespace", "gocd", "Prometheus namespace (prefix)")
	addr := flag.String(
		"listen-address", ":8080", "The address to listen on for Prometheus requests.")
	url := flag.String(
		"gocdURL", environment("GOCD_URL", "http://localhost:8153"), "URL of the GoCD server")
	user := flag.String(
		"gocdUser", environment("GOCD_USER", "admin"), "GoCD dashboard user login")
	passwd := flag.String(
		"gocdPass", environment("GOCD_PASS", "badger"), "GoCD dashboard user password")
	flag.Parse()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		scrape, err := gocdexporter.NewScraper(&gocdexporter.Config{
			*ns, prometheus.DefaultRegisterer, *url, *user, *passwd,
		})
		if err != nil {
			log.Fatal(err)
		}
		ticker := time.NewTicker(*duration)
		defer ticker.Stop()

		trigger := func(ctx context.Context, now time.Time) {
			ctx, cancel := context.WithDeadline(ctx, now.Add(*duration))
			defer cancel()
			if err := scrape(ctx); err != nil {
				switch err {
				case context.DeadlineExceeded:
					log.Println(err)
				default:
					log.Fatal(err)
				}
			}
		}

		// First scrape before waiting for tick duration.
		trigger(ctx, time.Now())
		if *once {
			return
		}

		for now := range ticker.C {
			trigger(ctx, now)
		}
	}()

	// Expose the registered metrics via HTTP to be scraped by Prometheus.
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.Serve(ln, nil)
		if err != nil {
			select {
			case <-ctx.Done():
				// Don't log "listener closed" error if we're shutting down.
			default:
				log.Fatalln(err)
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)
	select {
	case <-ctx.Done():
		log.Println("gocd scrape loop closed")
	case s := <-sigCh:
		log.Println(s)
	}
}

// environment key to lookup with fallback if not set.
func environment(key, fallback string) string {
	if s, ok := os.LookupEnv(key); ok {
		return s
	}
	return fallback
}
