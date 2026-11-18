// Package catalog defines the API manifest model, validates manifests and
// loads them from a directory. Manifests are what contributors submit in a
// pull request; CI runs the same validation as the gateway does at boot.
package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Param struct {
	Name        string `json:"name"`
	In          string `json:"in"` // "query" or "path"
	Required    bool   `json:"required"`
	Example     string `json:"example"`
	Description string `json:"description,omitempty"`
}

type Endpoint struct {
	Name    string  `json:"name"`
	Method  string  `json:"method"`
	Path    string  `json:"path"`
	Summary string  `json:"summary"`
	Params  []Param `json:"params,omitempty"`
}

type API struct {
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Categories  []string   `json:"categories"`
	Tags        []string   `json:"tags,omitempty"`
	Regions     []string   `json:"regions,omitempty"`
	Auth        string     `json:"auth"` // none | apikey | oauth
	CORS        bool       `json:"cors"` // declared by the contributor, verified by probes
	License     string     `json:"license,omitempty"`
	BaseURL     string     `json:"base_url"`
	DocsURL     string     `json:"docs_url"`
	SpecURL     string     `json:"spec_url,omitempty"`
	Endpoints   []Endpoint `json:"endpoints"`
}

var slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var pathParamRe = regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)

// Validate returns every problem found, so contributors fix them in one pass.
func (a *API) Validate() error {
	var errs []error
	add := func(f string, args ...any) { errs = append(errs, fmt.Errorf(f, args...)) }

	if !slugRe.MatchString(a.Slug) {
		add("slug %q must be lowercase kebab-case", a.Slug)
	}
	if strings.TrimSpace(a.Name) == "" {
		add("name is required")
	}
	if len(a.Description) < 20 {
		add("description must be at least 20 characters")
	}
	if len(a.Categories) == 0 {
		add("at least one category is required")
	}
	switch a.Auth {
	case "none", "apikey", "oauth":
	default:
		add("auth must be none, apikey or oauth (got %q)", a.Auth)
	}
	if err := requireHTTPS(a.BaseURL); err != nil {
		add("base_url: %v", err)
	}
	if err := requireHTTPS(a.DocsURL); err != nil {
		add("docs_url: %v", err)
	}
	if a.SpecURL != "" {
		if err := requireHTTPS(a.SpecURL); err != nil {
			add("spec_url: %v", err)
		}
	}
	if len(a.Endpoints) == 0 {
		add("at least one endpoint is required")
	}
	for i, e := range a.Endpoints {
		p := fmt.Sprintf("endpoints[%d]", i)
		if e.Name == "" {
			add("%s.name is required", p)
		}
		if e.Method != "GET" && e.Method != "POST" {
			add("%s.method must be GET or POST", p)
		}
		if !strings.HasPrefix(e.Path, "/") {
			add("%s.path must start with /", p)
		}
		declared := map[string]bool{}
		for _, pr := range e.Params {
			if pr.In != "query" && pr.In != "path" {
				add("%s param %q: in must be query or path", p, pr.Name)
			}
			if pr.Required && pr.Example == "" {
				add("%s param %q: required params need an example value", p, pr.Name)
			}
			declared[pr.Name] = pr.In == "path"
		}
		for _, m := range pathParamRe.FindAllStringSubmatch(e.Path, -1) {
			if isPath, ok := declared[m[1]]; !ok || !isPath {
				add("%s.path uses {%s} but no matching path param is declared", p, m[1])
			}
		}
	}
	return errors.Join(errs...)
}

func requireHTTPS(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%q is not a valid absolute URL", raw)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("%q must use https", raw)
	}
	return nil
}

// Host returns the hostname the API is served from.
func (a *API) Host() string {
	u, err := url.Parse(a.BaseURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// BuildURL substitutes path params and appends query params for an endpoint
// using the supplied values, falling back to each param's example.
func (a *API) BuildURL(e Endpoint, values map[string]string) string {
	path := e.Path
	q := url.Values{}
	for _, p := range e.Params {
		v, ok := values[p.Name]
		if !ok || v == "" {
			v = p.Example
		}
		if p.In == "path" {
			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(v))
		} else if v != "" {
			q.Set(p.Name, v)
		}
	}
	out := strings.TrimRight(a.BaseURL, "/") + path
	if len(q) > 0 {
		out += "?" + q.Encode()
	}
	return out
}

// ProbeEndpoint returns the first GET endpoint, which probes use as health check.
func (a *API) ProbeEndpoint() (Endpoint, bool) {
	for _, e := range a.Endpoints {
		if e.Method == "GET" {
			return e, true
		}
	}
	return Endpoint{}, false
}

// Store is the read interface the HTTP layer depends on. The in-memory
// implementation below can be swapped for Postgres without touching handlers.
type Store interface {
	All() []*API
	Get(slug string) (*API, bool)
	HostAllowed(host string) bool
}

type MemoryStore struct {
	apis  []*API
	bySlg map[string]*API
	hosts map[string]bool
}

func NewMemoryStore(apis []*API) *MemoryStore {
	s := &MemoryStore{bySlg: map[string]*API{}, hosts: map[string]bool{}}
	sorted := append([]*API(nil), apis...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for _, a := range sorted {
		s.apis = append(s.apis, a)
		s.bySlg[a.Slug] = a
		s.hosts[a.Host()] = true
	}
	return s
}

func (s *MemoryStore) All() []*API                  { return s.apis }
func (s *MemoryStore) Get(slug string) (*API, bool) { a, ok := s.bySlg[slug]; return a, ok }
func (s *MemoryStore) HostAllowed(h string) bool    { return s.hosts[strings.ToLower(h)] }

// LoadDir reads every *.json manifest in dir and validates it. It fails on the
// first invalid or duplicate manifest so bad data never reaches production.
func LoadDir(dir string) ([]*API, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no manifests found in %s", dir)
	}
	seen := map[string]string{}
	var out []*API
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var a API
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&a); err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		if err := a.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		if prev, dup := seen[a.Slug]; dup {
			return nil, fmt.Errorf("%s: duplicate slug %q (also in %s)", filepath.Base(f), a.Slug, prev)
		}
		seen[a.Slug] = filepath.Base(f)
		out = append(out, &a)
	}
	return out, nil
}
