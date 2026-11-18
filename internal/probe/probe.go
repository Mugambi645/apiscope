// Package probe measures real health for every catalogued API: uptime, latency,
// HTTPS and CORS. Results are kept in a bounded ring per API and persisted to a
// JSON file so badges survive restarts.
package probe

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"apiscope/internal/catalog"
)

const maxResults = 100

type Result struct {
	Time      time.Time `json:"time"`
	OK        bool      `json:"ok"`
	Status    int       `json:"status"`
	LatencyMS int64     `json:"latency_ms"`
	HTTPS     bool      `json:"https"`
	CORS      bool      `json:"cors"`
	Error     string    `json:"error,omitempty"`
}

type Summary struct {
	Slug         string     `json:"slug"`
	Checks       int        `json:"checks"`
	UptimePct    float64    `json:"uptime_pct"`
	P50MS        int64      `json:"p50_ms"`
	LastOK       bool       `json:"last_ok"`
	LastVerified *time.Time `json:"last_verified,omitempty"`
	HTTPS        bool       `json:"https"`
	CORSVerified bool       `json:"cors_verified"`
}

type Recorder struct {
	mu   sync.RWMutex
	data map[string][]Result
	path string
}

func NewRecorder(path string) *Recorder {
	r := &Recorder{data: map[string][]Result{}, path: path}
	if path != "" {
		if raw, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(raw, &r.data)
		}
	}
	return r
}

func (r *Recorder) Add(slug string, res Result) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := append(r.data[slug], res)
	if len(list) > maxResults {
		list = list[len(list)-maxResults:]
	}
	r.data[slug] = list
}

func (r *Recorder) History(slug string) []Result {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Result(nil), r.data[slug]...)
}

func (r *Recorder) Summary(slug string) Summary {
	h := r.History(slug)
	s := Summary{Slug: slug, Checks: len(h)}
	if len(h) == 0 {
		return s
	}
	var okCount int
	var lat []int64
	for _, x := range h {
		if x.OK {
			okCount++
			lat = append(lat, x.LatencyMS)
		}
	}
	last := h[len(h)-1]
	s.UptimePct = float64(okCount) * 100 / float64(len(h))
	s.LastOK, s.HTTPS, s.CORSVerified = last.OK, last.HTTPS, last.CORS
	t := last.Time
	s.LastVerified = &t
	if len(lat) > 0 {
		sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
		s.P50MS = lat[len(lat)/2]
	}
	return s
}

func (r *Recorder) Save() error {
	if r.path == "" {
		return nil
	}
	r.mu.RLock()
	raw, err := json.Marshal(r.data)
	r.mu.RUnlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, r.path) // atomic replace
}

type Prober struct {
	Client   *http.Client
	Store    catalog.Store
	Rec      *Recorder
	Interval time.Duration
	Origin   string // sent as Origin to test CORS
	Workers  int
	Logf     func(format string, args ...any)
}

// ProbeOnce performs one health check against the API's first GET endpoint.
func (p *Prober) ProbeOnce(ctx context.Context, a *catalog.API) Result {
	res := Result{Time: time.Now().UTC(), HTTPS: len(a.BaseURL) > 8 && a.BaseURL[:8] == "https://"}
	ep, ok := a.ProbeEndpoint()
	if !ok {
		res.Error = "no GET endpoint to probe"
		return res
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.BuildURL(ep, nil), nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	req.Header.Set("User-Agent", "apiscope-probe/1.0 (+https://github.com/mugambi645/apiscope)")
	req.Header.Set("Accept", "application/json, */*;q=0.5")
	if p.Origin != "" {
		req.Header.Set("Origin", p.Origin)
	}
	start := time.Now()
	resp, err := p.Client.Do(req)
	res.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	res.Status = resp.StatusCode
	acao := resp.Header.Get("Access-Control-Allow-Origin")
	res.CORS = acao == "*" || (p.Origin != "" && acao == p.Origin)
	// Keyed APIs answer 401/403 without a key: reachable counts as healthy.
	if a.Auth == "none" {
		res.OK = resp.StatusCode < 400
	} else {
		res.OK = resp.StatusCode < 500
	}
	return res
}

// Round probes every API once with bounded concurrency, then persists.
func (p *Prober) Round(ctx context.Context) {
	workers := p.Workers
	if workers <= 0 {
		workers = 4
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for _, a := range p.Store.All() {
		wg.Add(1)
		sem <- struct{}{}
		go func(a *catalog.API) {
			defer wg.Done()
			defer func() { <-sem }()
			cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			p.Rec.Add(a.Slug, p.ProbeOnce(cctx, a))
		}(a)
	}
	wg.Wait()
	if err := p.Rec.Save(); err != nil && p.Logf != nil {
		p.Logf("probe: save failed: %v", err)
	}
}

// Run probes immediately and then on every interval until ctx is cancelled.
func (p *Prober) Run(ctx context.Context) {
	p.Round(ctx)
	t := time.NewTicker(p.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.Round(ctx)
		}
	}
}
