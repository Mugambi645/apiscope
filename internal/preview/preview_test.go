package preview

import (
	"net/url"
	"testing"
)

func TestParse(t *testing.T) {
	base, _ := url.Parse("https://docs.example.com/guide/")
	html := `<html><head><title> Fallback   Title </title>
	<meta property="og:title" content="Example &amp; Co API">
	<meta name="description" content="Plain description">
	<meta property="og:description" content='Rich description'>
	<link rel="icon" href="/static/i.png"></head></html>`
	p := Parse(base, []byte(html))
	if p.Title != "Example & Co API" || p.Description != "Rich description" || p.Favicon != "https://docs.example.com/static/i.png" {
		t.Fatalf("unexpected preview %+v", p)
	}
	q := Parse(base, []byte("<title>Only title</title>"))
	if q.Title != "Only title" || q.Favicon != "https://docs.example.com/favicon.ico" {
		t.Fatalf("fallbacks failed %+v", q)
	}
}
