package patterns

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Tomcat detects the default Tomcat DefaultServlet listing:
// <h1>Directory Listing For /path</h1> and an <h3>Apache Tomcat/X.Y</h3>
// footer. The Server header is typically Apache-Coyote/1.1.
type Tomcat struct{}

func (Tomcat) Name() string { return "tomcat" }

func (Tomcat) Matches(doc *goquery.Document, h http.Header) bool {
	if strings.HasPrefix(h.Get("Server"), "Apache-Coyote") {
		return true
	}
	h1 := strings.TrimSpace(doc.Find("h1").First().Text())
	if strings.HasPrefix(h1, "Directory Listing For") {
		return true
	}
	h3 := strings.TrimSpace(doc.Find("h3").First().Text())
	return strings.HasPrefix(h3, "Apache Tomcat")
}

func (Tomcat) Entries(doc *goquery.Document) []Entry {
	var out []Entry
	doc.Find("table tr").Each(func(_ int, tr *goquery.Selection) {
		a := tr.Find("a[href]").First()
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
