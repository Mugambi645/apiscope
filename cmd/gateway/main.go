// Command gateway serves the APIScope public API, probes and playground proxy.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"apiscope/internal/catalog"
	"apiscope/internal/httpapi"
	"apiscope/internal/preview"
	"apiscope/internal/probe"
	"apiscope/internal/proxy"
	"apiscope/internal/ratelimit"
	"apiscope/internal/safeurl"
	"apiscope/internal/search"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envInt(k string, d int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return d
}

func main() {
	apis, err := catalog.LoadDir(env("CATALOG_DIR", "catalog"))
	if err != nil {
		log.Fatalf("catalog: %v", err)
	}
	store := catalog.NewMemoryStore(apis)
	log.Printf("loaded %d APIs", len(apis))

	rec := probe.NewRecorder(env("PROBE_FILE", "data/probes.json"))
	// Probes and previews follow redirects freely but never reach internal IPs.
	open := safeurl.NewClient(safeurl.Options{Timeout: 12 * time.Second})
	// The proxy additionally pins redirects to catalog hosts.
	pinned := safeurl.NewClient(safeurl.Options{Timeout: 15 * time.Second, AllowHost: store.HostAllowed})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if env("PROBES_ENABLED", "true") == "true" {
		p := &probe.Prober{
			Client: open, Store: store, Rec: rec, Origin: env("PROBE_ORIGIN", "https://apiscope.dev"),
			Interval: time.Duration(envInt("PROBE_INTERVAL_MIN", 30)) * time.Minute, Logf: log.Printf,
		}
		go p.Run(ctx)
	}

	read := ratelimit.New(envInt("READ_RPM", 240), 60)
	try := ratelimit.New(envInt("TRY_RPM", 30), 10)
	go func() {
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				read.Sweep(time.Hour)
				try.Sweep(time.Hour)
			}
		}
	}()

	srv := &httpapi.Server{
		Store: store, Ranker: search.Lexical{}, Probes: rec,
		Previews:  preview.NewService(open, 24*time.Hour),
		Proxy:     &proxy.Handler{Store: store, Client: pinned},
		RunnerURL: env("RUNNER_URL", ""), RunnerHTTP: &http.Client{Timeout: 20 * time.Second},
		AllowOrigin: env("ALLOW_ORIGIN", ""), TrustProxy: env("TRUST_PROXY", "false") == "true",
		ReadLimit: read, TryLimit: try,
	}
	hs := &http.Server{
		Addr: ":" + env("PORT", "8080"), Handler: srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		sc, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = hs.Shutdown(sc)
	}()
	log.Printf("gateway listening on %s", hs.Addr)
	if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
