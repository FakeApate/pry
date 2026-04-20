package patterns

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Lighttpd detects lighttpd's mod_dirlisting output: <h2>Index of ...</h2>
// followed by a <table summary="Directory Listing"> and a
// <div class="foot">lighttpd/X.Y.Z</div>.
type Lighttpd struct{}

func (Lighttpd) Name() string { return "lighttpd" }

func (Lighttpd) Matches(doc *goquery.Document, h http.Header) bool {
	if strings.HasPrefix(strings.ToLower(h.Get("Server")), "lighttpd") {
		return true
	}
	if strings.HasPrefix(strings.TrimSpace(doc.Find("div.foot").First().Text()), "lighttpd/") {
		return true
	}
	h2 := strings.TrimSpace(doc.Find("h2").First().Text())
	return strings.HasPrefix(h2, "Index of") &&
		doc.Find(`table[summary="Directory Listing"]`).Length() > 0
}

func (Lighttpd) Entries(doc *goquery.Document) []Entry {
	var out []Entry
	doc.Find(`table[summary="Directory Listing"] tbody tr`).Each(func(_ int, tr *goquery.Selection) {
		a := tr.Find("td.n a[href]").First()
		href, ok := a.Attr("href")
		if !ok {
			return
		}
		if skipHref(href, a.Text()) {
			return
		}
		isDir := tr.HasClass("d") || strings.HasSuffix(href, "/")
		out = append(out, Entry{Href: href, IsDir: isDir})
	})
	return out
}
