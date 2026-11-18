package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"apiscope/internal/catalog"
	"apiscope/internal/safeurl"
)

func apiFor(url, auth string) *catalog.API {
	return &catalog.API{Slug: "t", Name: "T", Auth: auth, BaseURL: url,
		Endpoints: []catalog.Endpoint{{Name: "ping", Method: "GET", Path: "/ping"}}}
}

func TestProbeOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write([]byte("{}"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	p := &Prober{Client: safeurl.NewClient(safeurl.Options{AllowPrivate: true}), Origin: "https://apiscope.dev"}
	res := p.ProbeOnce(context.Background(), apiFor(srv.URL, "none"))
	if !res.OK || !res.CORS || res.Status != 200 {
		t.Fatalf("unexpected result %+v", res)
	}
	// 401 is healthy for keyed APIs but a failure for open ones.
	locked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	defer locked.Close()
	if r := p.ProbeOnce(context.Background(), apiFor(locked.URL, "apikey")); !r.OK {
		t.Fatalf("keyed API returning 401 should be OK: %+v", r)
	}
	if r := p.ProbeOnce(context.Background(), apiFor(locked.URL, "none")); r.OK {
		t.Fatalf("open API returning 401 should fail: %+v", r)
	}
}

func TestRecorderSummaryAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "probes.json")
	rec := NewRecorder(path)
	now := time.Now().UTC()
	rec.Add("a", Result{Time: now, OK: true, LatencyMS: 100})
	rec.Add("a", Result{Time: now, OK: true, LatencyMS: 300})
	rec.Add("a", Result{Time: now, OK: false})
	rec.Add("a", Result{Time: now, OK: true, LatencyMS: 200, CORS: true})
	s := rec.Summary("a")
	if s.Checks != 4 || s.UptimePct != 75 || s.P50MS != 200 || !s.CORSVerified {
		t.Fatalf("bad summary %+v", s)
	}
	if err := rec.Save(); err != nil {
		t.Fatal(err)
	}
	if got := NewRecorder(path).Summary("a"); got.Checks != 4 {
		t.Fatalf("persistence lost data: %+v", got)
	}
	for i := 0; i < 150; i++ {
		rec.Add("b", Result{Time: now, OK: true})
	}
	if n := len(rec.History("b")); n != maxResults {
		t.Fatalf("ring buffer should cap at %d, got %d", maxResults, n)
	}
}
