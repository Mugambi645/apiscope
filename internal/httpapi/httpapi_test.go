package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"apiscope/internal/catalog"
	"apiscope/internal/preview"
	"apiscope/internal/probe"
	"apiscope/internal/proxy"
	"apiscope/internal/ratelimit"
	"apiscope/internal/safeurl"
	"apiscope/internal/search"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	apis, err := catalog.LoadDir("../../catalog")
	if err != nil {
		t.Fatal(err)
	}
	store := catalog.NewMemoryStore(apis)
	c := safeurl.NewClient(safeurl.Options{})
	s := &Server{
		Store: store, Ranker: search.Lexical{}, Probes: probe.NewRecorder(""),
		Previews: preview.NewService(c, time.Hour), Proxy: &proxy.Handler{Store: store, Client: c},
		RunnerHTTP: http.DefaultClient, AllowOrigin: "http://localhost:3000",
		ReadLimit: ratelimit.New(6000, 1000), TryLimit: ratelimit.New(60, 2),
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, url string, v any) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if v != nil {
		_ = json.NewDecoder(resp.Body).Decode(v)
	}
	return resp
}

func TestListSearchAndFilters(t *testing.T) {
	ts := newServer(t)
	var out struct {
		Count int `json:"count"`
		Items []struct {
			Slug string `json:"slug"`
		} `json:"items"`
	}
	get(t, ts.URL+"/api/apis?q=weather&category=agriculture", &out)
	if out.Count < 2 {
		t.Fatalf("expected weather+agriculture hits, got %d", out.Count)
	}
	get(t, ts.URL+"/api/apis?cors=false", &out)
	if out.Count != 1 || out.Items[0].Slug != "world-bank" {
		t.Fatalf("cors filter wrong: %+v", out)
	}
}

func TestDetailNotFoundAndCORS(t *testing.T) {
	ts := newServer(t)
	if resp := get(t, ts.URL+"/api/apis/nope", nil); resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	resp := get(t, ts.URL+"/api/apis/open-meteo", nil)
	if resp.StatusCode != 200 || resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("detail/CORS wrong: %d %v", resp.StatusCode, resp.Header)
	}
}

func TestRunDisabledWithoutRunner(t *testing.T) {
	ts := newServer(t)
	resp, err := http.Post(ts.URL+"/api/run", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 503 {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
}

func TestTryEndpointsAreRateLimited(t *testing.T) {
	ts := newServer(t)
	var limited bool
	for i := 0; i < 6; i++ {
		resp, _ := http.Post(ts.URL+"/api/run", "application/json", nil)
		if resp.StatusCode == http.StatusTooManyRequests {
			limited = true
		}
	}
	if !limited {
		t.Fatal("expected rate limiting to kick in")
	}
}
