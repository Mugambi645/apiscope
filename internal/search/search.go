// Package search ranks catalog entries for a text query plus structured filters.
// The Ranker interface is the seam where pgvector semantic search plugs in later.
package search

import (
	"sort"
	"strings"
	"unicode"

	"apiscope/internal/catalog"
)

type Filter struct {
	Query    string
	Category string
	Auth     string
	CORS     *bool
	Region   string
}

type Ranker interface {
	Search(apis []*catalog.API, f Filter) []*catalog.API
}

type Lexical struct{}

func tokens(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}

func (Lexical) Search(apis []*catalog.API, f Filter) []*catalog.API {
	type scored struct {
		api   *catalog.API
		score int
	}
	qt := tokens(f.Query)
	var hits []scored
	for _, a := range apis {
		if f.Category != "" && !contains(a.Categories, f.Category) {
			continue
		}
		if f.Auth != "" && a.Auth != f.Auth {
			continue
		}
		if f.CORS != nil && a.CORS != *f.CORS {
			continue
		}
		if f.Region != "" && !contains(a.Regions, f.Region) {
			continue
		}
		score := 1
		if len(qt) > 0 {
			score = 0
			name := strings.ToLower(a.Name + " " + a.Slug)
			cats := strings.ToLower(strings.Join(a.Categories, " "))
			tags := strings.ToLower(strings.Join(a.Tags, " ") + " " + strings.Join(a.Regions, " "))
			desc := strings.ToLower(a.Description)
			matched := 0
			for _, t := range qt {
				s := 0
				switch {
				case strings.Contains(name, t):
					s = 5
				case strings.Contains(tags, t):
					s = 3
				case strings.Contains(cats, t):
					s = 3
				case strings.Contains(desc, t):
					s = 1
				}
				if s > 0 {
					matched++
				}
				score += s
			}
			if matched < len(qt) { // every query term must match somewhere
				continue
			}
		}
		hits = append(hits, scored{a, score})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].api.Name < hits[j].api.Name
	})
	out := make([]*catalog.API, len(hits))
	for i, h := range hits {
		out[i] = h.api
	}
	return out
}
