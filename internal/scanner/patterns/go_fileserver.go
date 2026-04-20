package patterns

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// GoFileServer detects the minimalist output of Go's net/http FileServer:
// a bare <pre> block containing <a href> entries, with no <title> and no
// <h1>. Go's stdlib does not set a Server header, so detection is purely
// structural.
type GoFileServer struct{}

func (GoFileServer) Name() string { return "go-fileserver" }

func (GoFileServer) Matches(doc *goquery.Document, _ http.Header) bool {
	if doc.Find("title").Length() > 0 {
		return false
	}
	if doc.Find("h1, h2, h3").Length() > 0 {
		return false
	}
	pre := doc.Find("pre").First()
	if pre.Length() == 0 {
		return false
	}
	links := pre.Find("a[href]")
	if links.Length() == 0 {
		return false
	}
	allLocal := true
	links.EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		if strings.HasPrefix(href, "?") || strings.Contains(href, "://") {
			allLocal = false
			return false
		}
		return true
	})
	return allLocal
}

func (GoFileServer) Entries(doc *goquery.Document) []Entry {
	return extractPreEntries(doc)
}
