package patterns

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Generic is the loose fallback. It matches when no server-specific pattern
// recognized the page but the document still looks like a directory index:
// either the first <h1> equals the <title> (the original detector) or any
// h1/h2/h3 heading contains "Index of" or starts with "Directory". Entries
// are every remaining <a href>, after chrome filtering.
type Generic struct{}

func (Generic) Name() string { return "generic" }

func (Generic) Matches(doc *goquery.Document, _ http.Header) bool {
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if title != "" {
		h1 := strings.TrimSpace(doc.Find("h1").First().Text())
		if h1 != "" && h1 == title {
			return true
		}
	}

	matched := false
	doc.Find("h1, h2, h3").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		t := strings.TrimSpace(s.Text())
		if strings.Contains(t, "Index of") ||
			strings.HasPrefix(t, "Directory") {
			matched = true
			return false
		}
		return true
	})
	return matched
}

func (Generic) Entries(doc *goquery.Document) []Entry {
	var out []Entry
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if skipHref(href, s.Text()) {
			return
		}
		out = append(out, Entry{Href: href, IsDir: strings.HasSuffix(href, "/")})
	})
	return out
}
