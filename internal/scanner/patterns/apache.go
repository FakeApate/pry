package patterns

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Apache detects mod_autoindex output — both the classic <pre> listing and
// the FancyIndex table#indexlist variant. Requires an Apache Server header
// and an "Index of" h1; Tomcat's "Apache-Coyote" header is caught by the
// higher-priority Tomcat pattern.
type Apache struct{}

func (Apache) Name() string { return "apache" }

func (Apache) Matches(doc *goquery.Document, h http.Header) bool {
	if !strings.HasPrefix(h.Get("Server"), "Apache") {
		return false
	}
	h1 := strings.TrimSpace(doc.Find("h1").First().Text())
	if !strings.HasPrefix(h1, "Index of") {
		return false
	}
	return doc.Find("pre").Length() > 0 || doc.Find("table#indexlist").Length() > 0
}

func (Apache) Entries(doc *goquery.Document) []Entry {
	if doc.Find("table#indexlist").Length() > 0 {
		var out []Entry
		doc.Find("table#indexlist tr").Each(func(_ int, tr *goquery.Selection) {
			a := tr.Find("td.indexcolname a[href]").First()
			href, ok := a.Attr("href")
			if !ok {
				return
			}
			if skipHref(href, a.Text()) {
				return
			}
			out = append(out, Entry{Href: href, IsDir: strings.HasSuffix(href, "/")})
		})
		return out
	}
	return extractPreEntries(doc)
}
