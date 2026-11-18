// Package proxy implements the playground's "Try it" tab: a request relay that
// only talks to hosts present in the catalog, caps sizes and never stores or
// logs credentials. API keys are supplied per request by the browser.
package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"apiscope/internal/catalog"
	"apiscope/internal/safeurl"
)

const (
	maxRequestBody  = 64 << 10
	maxResponseBody = 1 << 20
)

type Request struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type Response struct {
	Status     int               `json:"status"`
	StatusText string            `json:"status_text"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Truncated  bool              `json:"truncated"`
	LatencyMS  int64             `json:"latency_ms"`
}

type Handler struct {
	Store     catalog.Store
	Client    *http.Client
	AllowHTTP bool // tests only
}

var headerNameRe = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

var forbiddenHeaders = map[string]bool{
	"host": true, "cookie": true, "content-length": true, "connection": true,
	"transfer-encoding": true, "upgrade": true, "forwarded": true, "origin": true,
	"referer": true, "te": true, "trailer": true, "keep-alive": true,
}

var exposedResponseHeaders = []string{
	"Content-Type", "Cache-Control", "ETag", "Retry-After", "Link",
	"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset",
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// Validate checks a request against policy and returns the parsed URL.
func (h *Handler) Validate(req *Request) (*url.URL, error) {
	req.Method = strings.ToUpper(req.Method)
	if req.Method != http.MethodGet && req.Method != http.MethodPost {
		return nil, fmt.Errorf("method %q is not allowed", req.Method)
	}
	if len(req.Body) > maxRequestBody {
		return nil, fmt.Errorf("request body exceeds %d bytes", maxRequestBody)
	}
	u, err := url.Parse(req.URL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid url")
	}
	if u.Scheme != "https" && !(h.AllowHTTP && u.Scheme == "http") {
		return nil, fmt.Errorf("only https urls are allowed")
	}
	if u.User != nil {
		return nil, fmt.Errorf("credentials in the url are not allowed")
	}
	if !h.Store.HostAllowed(u.Hostname()) {
		return nil, fmt.Errorf("host %q is not in the catalog", u.Hostname())
	}
	for k := range req.Headers {
		if !headerNameRe.MatchString(k) {
			return nil, fmt.Errorf("invalid header name %q", k)
		}
		if forbiddenHeader(k) {
			return nil, fmt.Errorf("header %q is not allowed", k)
		}
	}
	return u, nil
}

// forbiddenHeader reports headers the caller may not set on the upstream request.
func forbiddenHeader(name string) bool {
	n := strings.ToLower(name)
	if forbiddenHeaders[n] {
		return true
	}
	for _, prefix := range []string{"proxy-", "x-forwarded-", "sec-"} {
		if strings.HasPrefix(n, prefix) {
			return true
		}
	}
	return false
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody+4096)
	var in Request
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		fail(w, http.StatusBadRequest, "invalid json body")
		return
	}
	u, err := h.Validate(&in)
	if err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	var body io.Reader
	if in.Body != "" {
		body = strings.NewReader(in.Body)
	}
	out, err := http.NewRequestWithContext(r.Context(), in.Method, u.String(), body)
	if err != nil {
		fail(w, http.StatusBadRequest, "could not build request")
		return
	}
	for k, v := range in.Headers {
		out.Header.Set(k, v)
	}
	out.Header.Set("User-Agent", "apiscope-playground/1.0")

	start := time.Now()
	resp, err := h.Client.Do(out)
	if err != nil {
		fail(w, http.StatusBadGateway, "upstream request failed: "+sanitize(err))
		return
	}
	defer resp.Body.Close()
	raw, truncated, err := safeurl.ReadLimited(resp.Body, maxResponseBody)
	if err != nil {
		fail(w, http.StatusBadGateway, "failed reading upstream response")
		return
	}
	res := Response{
		Status: resp.StatusCode, StatusText: http.StatusText(resp.StatusCode),
		Headers: map[string]string{}, Truncated: truncated,
		LatencyMS: time.Since(start).Milliseconds(),
	}
	for _, k := range exposedResponseHeaders {
		if v := resp.Header.Get(k); v != "" {
			res.Headers[k] = v
		}
	}
	if utf8.Valid(raw) {
		res.Body = string(raw)
	} else {
		res.Body = fmt.Sprintf("[binary response, %d bytes, not displayed]", len(raw))
	}
	writeJSON(w, http.StatusOK, res)
}

// sanitize keeps upstream error text short and free of internal addresses.
func sanitize(err error) string {
	s := err.Error()
	if len(s) > 160 {
		s = s[:160]
	}
	return s
}
