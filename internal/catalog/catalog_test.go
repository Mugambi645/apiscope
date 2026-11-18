package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func valid() *API {
	return &API{
		Slug: "demo", Name: "Demo", Description: "A demo API used in unit tests only.",
		Categories: []string{"testing"}, Auth: "none", BaseURL: "https://api.demo.test", DocsURL: "https://demo.test/docs",
		Endpoints: []Endpoint{{Name: "get", Method: "GET", Path: "/items/{id}", Params: []Param{{Name: "id", In: "path", Required: true, Example: "1"}, {Name: "q", In: "query", Example: "x"}}}},
	}
}

func TestValidateAcceptsValid(t *testing.T) {
	if err := valid().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReportsAllProblems(t *testing.T) {
	a := valid()
	a.Slug, a.Auth, a.BaseURL = "Bad Slug", "magic", "http://insecure.test"
	a.Endpoints[0].Path = "/items/{missing}"
	err := a.Validate()
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, want := range []string{"slug", "auth", "base_url", "{missing}"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
}

func TestBuildURL(t *testing.T) {
	a := valid()
	got := a.BuildURL(a.Endpoints[0], map[string]string{"id": "a b"})
	if got != "https://api.demo.test/items/a%20b?q=x" {
		t.Fatalf("got %s", got)
	}
	if got := a.BuildURL(a.Endpoints[0], nil); got != "https://api.demo.test/items/1?q=x" {
		t.Fatalf("examples should be the default: %s", got)
	}
}

func TestLoadDirRejectsDuplicatesAndUnknownFields(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	good := `{"slug":"demo","name":"Demo","description":"A demo API used in unit tests only.","categories":["t"],"auth":"none","cors":true,"base_url":"https://a.test","docs_url":"https://a.test/d","endpoints":[{"name":"n","method":"GET","path":"/x","summary":"s"}]}`
	write("a.json", good)
	write("b.json", good)
	if _, err := LoadDir(dir); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	write("b.json", strings.Replace(good, `"slug":"demo"`, `"slug":"other","surprise":1`, 1))
	if _, err := LoadDir(dir); err == nil || !strings.Contains(err.Error(), "surprise") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestSeedCatalogIsValid(t *testing.T) {
	apis, err := LoadDir("../../catalog")
	if err != nil {
		t.Fatal(err)
	}
	if len(apis) < 10 {
		t.Fatalf("expected a seeded catalog, got %d", len(apis))
	}
}
