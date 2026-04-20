package patterns

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Nginx detects nginx's autoindex module output: <h1>Index of /path</h1>
// followed by a <pre> block. Requires an nginx Server header — without it
// the output is indistinguishable from Apache classic and the Generic
// fallback handles extraction.
type Nginx struct{}

func (Nginx) Name() string { return "nginx" }

func (Nginx) Matches(doc *goquery.Document, h http.Header) bool {
	if !strings.HasPrefix(strings.ToLower(h.Get("Server")), "nginx") {
		return false
	}
	h1 := strings.TrimSpace(doc.Find("h1").First().Text())
	if !strings.HasPrefix(h1, "Index of") {
		return false
	}
	return doc.Find("pre").Length() > 0
}

func (Nginx) Entries(doc *goquery.Document) []Entry {
	return extractPreEntries(doc)
}
