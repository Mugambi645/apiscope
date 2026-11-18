// Command runner exposes the sandbox over HTTP. It is only reachable from the
// gateway and is the sole component that touches the Docker socket.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"apiscope/internal/sandbox"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func reply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	cfg := sandbox.DefaultConfig()
	cfg.Network = env("SANDBOX_NETWORK", cfg.Network)
	cfg.ProxyURL = env("EGRESS_PROXY_URL", cfg.ProxyURL)
	runner := sandbox.NewRunner(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { reply(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /run", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, int64(cfg.MaxCode)+1024)
		var req sandbox.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			reply(w, 400, map[string]string{"error": "invalid json"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), cfg.Timeout+10*time.Second)
		defer cancel()
		res, err := runner.Run(ctx, req)
		switch {
		case errors.Is(err, sandbox.ErrUnsupported), errors.Is(err, sandbox.ErrTooLarge):
			reply(w, 400, map[string]string{"error": err.Error()})
		case errors.Is(err, sandbox.ErrBusy):
			reply(w, 429, map[string]string{"error": err.Error()})
		case err != nil:
			log.Printf("run failed: %v", err)
			reply(w, 500, map[string]string{"error": "execution failed"})
		default:
			reply(w, 200, res)
		}
	})
	srv := &http.Server{Addr: ":" + env("PORT", "8081"), Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("runner listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
