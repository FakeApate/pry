package patterns

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Caddy detects Caddy file_server browse output. Fingerprint is the
// caddyserver.com footer link the default template includes, or a Caddy
// Server header. Entries live in <tbody><tr> rows with a class "dir" on
// directory rows.
type Caddy struct{}

func (Caddy) Name() string { return "caddy" }

func (Caddy) Matches(doc *goquery.Document, h http.Header) bool {
	if strings.HasPrefix(h.Get("Server"), "Caddy") {
		return true
	}
	match := false
	doc.Find("footer a[href]").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		if strings.Contains(href, "caddyserver.com") {
			match = true
			return false
		}
		return true
	})
	return match
}

func (Caddy) Entries(doc *goquery.Document) []Entry {
	var out []Entry
	doc.Find("tbody tr").Each(func(_ int, tr *goquery.Selection) {
		a := tr.Find("td a[href]").First()
		href, ok := a.Attr("href")
		if !ok {
			return
		}
		if skipHref(href, a.Text()) {
			return
		}
		isDir := tr.HasClass("dir") || strings.HasSuffix(href, "/")
		out = append(out, Entry{Href: href, IsDir: isDir})
	})
	return out
}
