package search

import (
	"testing"

	"apiscope/internal/catalog"
)

var sample = []*catalog.API{
	{Slug: "open-meteo", Name: "Open-Meteo", Description: "Weather forecast API", Categories: []string{"weather", "agriculture"}, Tags: []string{"forecast"}, Auth: "none", CORS: true, Regions: []string{"kenya"}},
	{Slug: "github", Name: "GitHub", Description: "Repository data and weather of code", Categories: []string{"developer-tools"}, Auth: "none", CORS: true},
	{Slug: "darksky", Name: "Dark Sky", Description: "Weather with a key", Categories: []string{"weather"}, Auth: "apikey", CORS: false},
}

func slugs(in []*catalog.API) []string {
	var o []string
	for _, a := range in {
		o = append(o, a.Slug)
	}
	return o
}

func TestRanking(t *testing.T) {
	got := slugs(Lexical{}.Search(sample, Filter{Query: "weather"}))
	// category/tag match outranks a description-only mention
	if len(got) != 3 || got[2] != "github" {
		t.Fatalf("unexpected ranking %v", got)
	}
}

func TestFilters(t *testing.T) {
	yes := true
	if got := slugs(Lexical{}.Search(sample, Filter{Auth: "none", CORS: &yes, Category: "weather"})); len(got) != 1 || got[0] != "open-meteo" {
		t.Fatalf("got %v", got)
	}
	if got := slugs(Lexical{}.Search(sample, Filter{Region: "kenya"})); len(got) != 1 {
		t.Fatalf("region filter failed: %v", got)
	}
}

func TestAllTermsMustMatch(t *testing.T) {
	got := Lexical{}.Search(sample, Filter{Query: "weather agriculture nonsense"})
	if len(got) != 0 {
		t.Fatalf("expected no hits, got %v", slugs(got))
	}
}

func TestRegionIsSearchable(t *testing.T) {
	got := Lexical{}.Search(sample, Filter{Query: "kenya"})
	if len(got) != 1 || got[0].Slug != "open-meteo" {
		t.Fatalf("got %v", slugs(got))
	}
}
