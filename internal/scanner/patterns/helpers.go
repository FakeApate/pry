package patterns

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// skipHref filters out chrome links that are not file/dir entries: sort-order
// query params, fragments, external URLs, and "Parent Directory" navigation.
func skipHref(href, text string) bool {
	if href == "" {
		return true
	}
	if strings.HasPrefix(href, "?") || strings.HasPrefix(href, "#") {
		return true
	}
	if strings.Contains(href, "://") {
		return true
	}
	if href == "../" || href == ".." || href == "./" {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(text), "parent directory") {
		return true
	}
	return false
}

// extractPreEntries walks <pre> <a href> entries — the shape used by Apache
// mod_autoindex classic, nginx autoindex, and Go's net/http FileServer.
func extractPreEntries(doc *goquery.Document) []Entry {
	var out []Entry
	doc.Find("pre a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if skipHref(href, s.Text()) {
			return
		}
		out = append(out, Entry{Href: href, IsDir: strings.HasSuffix(href, "/")})
	})
	return out
}
