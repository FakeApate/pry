package patterns

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// IIS detects Microsoft-IIS DirectoryListingModule output: a <pre> block
// whose first line is "[To Parent Directory]" followed by rows of
// "date time <size|<dir>> <a href=...>name</a>".
type IIS struct{}

func (IIS) Name() string { return "iis" }

func (IIS) Matches(doc *goquery.Document, h http.Header) bool {
	if strings.HasPrefix(h.Get("Server"), "Microsoft-IIS") {
		return true
	}
	pre := doc.Find("pre").First()
	if pre.Length() == 0 {
		return false
	}
	return strings.Contains(pre.Text(), "[To Parent Directory]")
}

func (IIS) Entries(doc *goquery.Document) []Entry {
	var out []Entry
	doc.Find("pre a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		text := strings.TrimSpace(s.Text())
		if text == "[To Parent Directory]" {
			return
		}
		if skipHref(href, text) {
			return
		}
		out = append(out, Entry{Href: href, IsDir: strings.HasSuffix(href, "/")})
	})
	return out
}
