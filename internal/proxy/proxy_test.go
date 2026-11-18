package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"apiscope/internal/catalog"
	"apiscope/internal/safeurl"
)

func setup(t *testing.T, upstream http.HandlerFunc) (*Handler, string) {
	t.Helper()
	up := httptest.NewServer(upstream)
	t.Cleanup(up.Close)
	u, _ := url.Parse(up.URL)
	store := catalog.NewMemoryStore([]*catalog.API{{Slug: "x", BaseURL: up.URL}})
	_ = u
	return &Handler{Store: store, Client: safeurl.NewClient(safeurl.Options{AllowPrivate: true}), AllowHTTP: true}, up.URL
}

func do(h *Handler, req Request) (*httptest.ResponseRecorder, Response) {
	b, _ := json.Marshal(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/proxy", bytes.NewReader(b)))
	var res Response
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	return rec, res
}

func TestRelaysAllowedHost(t *testing.T) {
	h, base := setup(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "leak=1")
		w.Write([]byte(`{"key":"` + r.Header.Get("X-API-Key") + `"}`))
	})
	rec, res := do(h, Request{Method: "get", URL: base + "/v1/x", Headers: map[string]string{"X-API-Key": "abc"}})
	if rec.Code != 200 || res.Status != 200 || !strings.Contains(res.Body, `"abc"`) {
		t.Fatalf("unexpected %d %+v", rec.Code, res)
	}
	if _, leaked := res.Headers["Set-Cookie"]; leaked {
		t.Fatal("Set-Cookie must not be forwarded")
	}
}

func TestRejectsPolicyViolations(t *testing.T) {
	h, base := setup(t, func(w http.ResponseWriter, r *http.Request) {})
	bad := []Request{
		{Method: "DELETE", URL: base},
		{Method: "GET", URL: "http://evil.example/"},
		{Method: "GET", URL: "https://user:pw@" + strings.TrimPrefix(base, "http://")},
		{Method: "GET", URL: base, Headers: map[string]string{"Cookie": "a=b"}},
		{Method: "GET", URL: base, Headers: map[string]string{"X-Forwarded-For": "1.1.1.1"}},
		{Method: "GET", URL: base, Headers: map[string]string{"Bad Header": "v"}},
		{Method: "POST", URL: base, Body: strings.Repeat("a", maxRequestBody+1)},
	}
	for i, r := range bad {
		if rec, _ := do(h, r); rec.Code != http.StatusBadRequest {
			t.Errorf("case %d: expected 400, got %d (%s)", i, rec.Code, rec.Body.String())
		}
	}
}

func TestTruncatesLargeResponses(t *testing.T) {
	h, base := setup(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("a", maxResponseBody+10)))
	})
	_, res := do(h, Request{Method: "GET", URL: base})
	if !res.Truncated || len(res.Body) != maxResponseBody {
		t.Fatalf("expected truncation, got truncated=%v len=%d", res.Truncated, len(res.Body))
	}
}
