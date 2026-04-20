// Package patterns recognizes open-directory index pages produced by various
// HTTP servers and extracts their file/directory entries. Patterns are
// registered in priority order; the most specific fingerprints come first and
// a loose generic fallback runs last.
package patterns

import (
	"net/http"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// Entry is one link on an index page.
type Entry struct {
	Href  string
	IsDir bool
}

// IndexPattern recognizes a server-specific listing page and extracts entries.
// Matches must be cheap, side-effect free, and safe for concurrent calls on
// the same document — Detect invokes every registered pattern in parallel.
type IndexPattern interface {
	Name() string
	Matches(doc *goquery.Document, headers http.Header) bool
	Entries(doc *goquery.Document) []Entry
}

var (
	registryMu sync.RWMutex
	registry   []IndexPattern
)

// Register adds p to the end of the detection order. Typically called from
// init() in each pattern file. Later-registered patterns are lower priority.
func Register(p IndexPattern) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = append(registry, p)
}

// Registered returns a snapshot of the registered patterns in priority order.
// Primarily for tests.
func Registered() []IndexPattern {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]IndexPattern, len(registry))
	copy(out, registry)
	return out
}

// Detect runs every registered pattern's Matches concurrently and returns the
// first match in registration order. Returns nil if none match.
//
// Concurrency: goroutines only read doc/headers and write to distinct slots in
// a pre-sized results slice, so no locking is required. goquery.Document walks
// are safe for concurrent readers because Find does not mutate the tree.
func Detect(doc *goquery.Document, headers http.Header) IndexPattern {
	patterns := Registered()
	if len(patterns) == 0 {
		return nil
	}

	results := make([]bool, len(patterns))
	var wg sync.WaitGroup
	wg.Add(len(patterns))
	for i, p := range patterns {
		go func(i int, p IndexPattern) {
			defer wg.Done()
			results[i] = p.Matches(doc, headers)
		}(i, p)
	}
	wg.Wait()

	for i, matched := range results {
		if matched {
			return patterns[i]
		}
	}
	return nil
}
