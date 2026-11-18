// Package preview unfurls documentation links (title, description, favicon)
// and caches the result. Only URLs the caller has already authorised are fetched.
package preview

import (
	"context"
	"errors"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"apiscope/internal/safeurl"
)

type Preview struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Favicon     string    `json:"favicon"`
	FetchedAt   time.Time `json:"fetched_at"`
}

var (
	titleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	metaRe  = regexp.MustCompile(`(?is)<meta\s+[^>]*>`)
	linkRe  = regexp.MustCompile(`(?is)<link\s+[^>]*>`)
	attrRe  = regexp.MustCompile(`(?is)([a-zA-Z:-]+)\s*=\s*("([^"]*)"|'([^']*)')`)
	spaceRe = regexp.MustCompile(`\s+`)
)

func attrs(tag string) map[string]string {
	out := map[string]string{}
	for _, m := range attrRe.FindAllStringSubmatch(tag, -1) {
		v := m[3]
		if v == "" {
			v = m[4]
		}
		out[strings.ToLower(m[1])] = html.UnescapeString(v)
	}
	return out
}

func clean(s string) string {
	return strings.TrimSpace(spaceRe.ReplaceAllString(html.UnescapeString(s), " "))
}

// Parse extracts preview metadata from raw HTML. It is deliberately forgiving
// and regexp-based: previews are best-effort, so a missing field is not an error.
func Parse(base *url.URL, body []byte) Preview {
	src := string(body)
	p := Preview{URL: base.String()}
	if m := titleRe.FindStringSubmatch(src); m != nil {
		p.Title = clean(m[1])
	}
	var ogTitle, ogDesc, desc string
	for _, tag := range metaRe.FindAllString(src, -1) {
		a := attrs(tag)
		key := strings.ToLower(a["property"] + a["name"])
		switch key {
		case "og:title":
			ogTitle = clean(a["content"])
		case "og:description":
			ogDesc = clean(a["content"])
		case "description":
			desc = clean(a["content"])
		}
	}
	if ogTitle != "" {
		p.Title = ogTitle
	}
	p.Description = ogDesc
	if p.Description == "" {
		p.Description = desc
	}
	for _, tag := range linkRe.FindAllString(src, -1) {
		a := attrs(tag)
		rel := strings.ToLower(a["rel"])
		if (rel == "icon" || rel == "shortcut icon") && a["href"] != "" {
			if ref, err := url.Parse(a["href"]); err == nil {
				p.Favicon = base.ResolveReference(ref).String()
				break
			}
		}
	}
	if p.Favicon == "" {
		p.Favicon = base.ResolveReference(&url.URL{Path: "/favicon.ico"}).String()
	}
	return p
}

type Service struct {
	Client *http.Client
	TTL    time.Duration
	mu     sync.Mutex
	cache  map[string]Preview
}

func NewService(c *http.Client, ttl time.Duration) *Service {
	return &Service{Client: c, TTL: ttl, cache: map[string]Preview{}}
}

func (s *Service) Get(ctx context.Context, raw string) (Preview, error) {
	s.mu.Lock()
	if p, ok := s.cache[raw]; ok && time.Since(p.FetchedAt) < s.TTL {
		s.mu.Unlock()
		return p, nil
	}
	s.mu.Unlock()

	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return Preview{}, errors.New("invalid url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return Preview{}, err
	}
	req.Header.Set("User-Agent", "apiscope-preview/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := s.Client.Do(req)
	if err != nil {
		return Preview{}, err
	}
	defer resp.Body.Close()
	body, _, err := safeurl.ReadLimited(resp.Body, 256<<10)
	if err != nil {
		return Preview{}, err
	}
	p := Parse(resp.Request.URL, body)
	p.URL = raw
	p.FetchedAt = time.Now().UTC()
	s.mu.Lock()
	s.cache[raw] = p
	s.mu.Unlock()
	return p, nil
}
