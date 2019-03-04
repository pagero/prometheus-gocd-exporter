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
	duration := flag.String(
		"duration", env("DURATION", "1m"), "The duration between scrapes")
	ns := flag.String("namespace", "gocd", "Prometheus namespace (prefix)")
	addr := flag.String(
		"listen-address", ":8080", "The address to listen on for Prometheus requests.")
	url := flag.String(
		"gocdURL", env("GOCD_URL", "http://localhost:8153"), "URL of the GoCD server")
	user := flag.String(
		"gocdUser", env("GOCD_USER", "admin"), "GoCD dashboard user login")
	passwd := flag.String(
		"gocdPass", env("GOCD_PASS", "badger"), "GoCD dashboard user password")
	agentMaxPages := flag.Int(
		"maxPages", 3, "Agent job history maximum number of pages to parse")

	flag.Parse()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	dur, err := time.ParseDuration(*duration)
	if err != nil {
		log.Fatal(err)
	}
	healthCh := make(chan struct{}, 1)
	go healthcheck(dur, healthCh)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		scrape, err := gocdexporter.NewScraper(&gocdexporter.Config{
			*ns, prometheus.DefaultRegisterer, *url, *user, *passwd, *agentMaxPages,
		})
		if err != nil {
			log.Fatal(err)
		}
		ticker := time.NewTicker(dur)
		defer ticker.Stop()
		errorGauge := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "exporter",
				Name:      "scrape_error",
				Help:      "prometheus-gocd-exporter scrape error",
			},
			[]string{})
		if err := prometheus.DefaultRegisterer.Register(errorGauge); err != nil {
			log.Fatal(err)
		}

		trigger := func(ctx context.Context, now time.Time) {
			select {
			case healthCh <- struct{}{}:
			default:
			}
			ctx, cancel := context.WithDeadline(ctx, now.Add(dur))
			defer cancel()
			errorGauge.Reset()
			errorGauge.WithLabelValues().Set(0)
			if err := scrape(ctx); err != nil {
				switch err {
				case context.DeadlineExceeded:
					log.Println(err)
				default:
					log.Println(err)
				}
				errorGauge.WithLabelValues().Set(1)
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

// env key to lookup with fallback if not set.
func env(key, fallback string) string {
	if s, ok := os.LookupEnv(key); ok {
		return s
	}
	return fallback
}
