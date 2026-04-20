package patterns

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Python detects Python's http.server / SimpleHTTPServer output:
// <title>Directory listing for /path</title>, <h1> of the same, then a
// <ul><li><a> list.
type Python struct{}

func (Python) Name() string { return "python-http-server" }

func (Python) Matches(doc *goquery.Document, h http.Header) bool {
	if strings.HasPrefix(h.Get("Server"), "SimpleHTTP/") {
		return true
	}
	title := strings.TrimSpace(doc.Find("title").First().Text())
	return strings.HasPrefix(title, "Directory listing for")
}

func (Python) Entries(doc *goquery.Document) []Entry {
	var out []Entry
	doc.Find("ul li a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if skipHref(href, s.Text()) {
			return
		}
		out = append(out, Entry{Href: href, IsDir: strings.HasSuffix(href, "/")})
	})
	return out
}
