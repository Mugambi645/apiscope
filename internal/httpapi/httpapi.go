// Package httpapi wires the gateway's public HTTP surface.
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"apiscope/internal/catalog"
	"apiscope/internal/preview"
	"apiscope/internal/probe"
	"apiscope/internal/ratelimit"
	"apiscope/internal/search"
)

type Server struct {
	Store       catalog.Store
	Ranker      search.Ranker
	Probes      *probe.Recorder
	Previews    *preview.Service
	Proxy       http.Handler
	RunnerURL   string // empty disables /api/run
	RunnerHTTP  *http.Client
	AllowOrigin string // CORS origin for the web app, "" disables CORS headers
	TrustProxy  bool
	ReadLimit   *ratelimit.Limiter
	TryLimit    *ratelimit.Limiter // proxy and run are costlier, limited harder
}

type listItem struct {
	*catalog.API
	Health probe.Summary `json:"health"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"status": "ok", "apis": len(s.Store.All())})
	})
	mux.Handle("GET /api/apis", s.ReadLimit.Middleware(s.TrustProxy, http.HandlerFunc(s.list)))
	mux.Handle("GET /api/apis/{slug}", s.ReadLimit.Middleware(s.TrustProxy, http.HandlerFunc(s.detail)))
	mux.Handle("GET /api/apis/{slug}/health", s.ReadLimit.Middleware(s.TrustProxy, http.HandlerFunc(s.health)))
	mux.Handle("GET /api/apis/{slug}/preview", s.ReadLimit.Middleware(s.TrustProxy, http.HandlerFunc(s.preview)))
	mux.Handle("GET /api/categories", s.ReadLimit.Middleware(s.TrustProxy, http.HandlerFunc(s.categories)))
	mux.Handle("POST /api/proxy", s.TryLimit.Middleware(s.TrustProxy, s.Proxy))
	mux.Handle("POST /api/run", s.TryLimit.Middleware(s.TrustProxy, http.HandlerFunc(s.run)))
	return s.withMiddleware(mux)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if s.AllowOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.AllowOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := search.Filter{
		Query: q.Get("q"), Category: q.Get("category"), Auth: q.Get("auth"), Region: q.Get("region"),
	}
	if v := q.Get("cors"); v == "true" || v == "false" {
		b := v == "true"
		f.CORS = &b
	}
	hits := s.Ranker.Search(s.Store.All(), f)
	items := make([]listItem, len(hits))
	for i, a := range hits {
		items[i] = listItem{a, s.Probes.Summary(a.Slug)}
	}
	writeJSON(w, 200, map[string]any{"count": len(items), "items": items})
}

func (s *Server) detail(w http.ResponseWriter, r *http.Request) {
	a, ok := s.Store.Get(r.PathValue("slug"))
	if !ok {
		fail(w, 404, "api not found")
		return
	}
	writeJSON(w, 200, listItem{a, s.Probes.Summary(a.Slug)})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if _, ok := s.Store.Get(slug); !ok {
		fail(w, 404, "api not found")
		return
	}
	writeJSON(w, 200, map[string]any{"summary": s.Probes.Summary(slug), "history": s.Probes.History(slug)})
}

func (s *Server) preview(w http.ResponseWriter, r *http.Request) {
	a, ok := s.Store.Get(r.PathValue("slug"))
	if !ok {
		fail(w, 404, "api not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	p, err := s.Previews.Get(ctx, a.DocsURL) // only the catalogued docs URL, never user input
	if err != nil {
		fail(w, 502, "could not fetch preview")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, 200, p)
}

func (s *Server) categories(w http.ResponseWriter, r *http.Request) {
	counts := map[string]int{}
	for _, a := range s.Store.All() {
		for _, c := range a.Categories {
			counts[strings.ToLower(c)]++
		}
	}
	type row struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	rows := make([]row, 0, len(counts))
	for n, c := range counts {
		rows = append(rows, row{n, c})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		return rows[i].Name < rows[j].Name
	})
	writeJSON(w, 200, rows)
}

func (s *Server) run(w http.ResponseWriter, r *http.Request) {
	if s.RunnerURL == "" {
		fail(w, 503, "code runner is not configured")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<10))
	if err != nil {
		fail(w, 413, "request too large")
		return
	}
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, s.RunnerURL+"/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.RunnerHTTP.Do(req)
	if err != nil {
		fail(w, 502, "code runner unavailable")
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 256<<10))
}
