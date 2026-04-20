package patterns

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Jetty detects Eclipse Jetty's default resource-handler listing:
// <h1>Directory: /path</h1> followed by a plain <table> of links. Server
// header looks like "Jetty(12.0.0)".
type Jetty struct{}

func (Jetty) Name() string { return "jetty" }

func (Jetty) Matches(doc *goquery.Document, h http.Header) bool {
	if strings.HasPrefix(h.Get("Server"), "Jetty") {
		return true
	}
	h1 := strings.TrimSpace(doc.Find("h1").First().Text())
	return strings.HasPrefix(h1, "Directory: ")
}

func (Jetty) Entries(doc *goquery.Document) []Entry {
	var out []Entry
	doc.Find("table a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if skipHref(href, s.Text()) {
			return
		}
		out = append(out, Entry{Href: href, IsDir: strings.HasSuffix(href, "/")})
	})
	return out
}
