package patterns

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func loadFixture(t *testing.T, name string) *goquery.Document {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return doc
}

func hdr(server string) http.Header {
	h := http.Header{}
	if server != "" {
		h.Set("Server", server)
	}
	return h
}

func hrefSet(entries []Entry) map[string]bool {
	out := make(map[string]bool, len(entries))
	for _, e := range entries {
		out[e.Href] = e.IsDir
	}
	return out
}

func TestDetect(t *testing.T) {
	cases := []struct {
		name        string
		fixture     string
		server      string
		wantPattern string
		wantEntries int
	}{
		{"apache classic", "apache.html", "Apache/2.4.58", "apache", 4},
		{"apache fancyindex", "apache_fancyindex.html", "Apache/2.4.58", "apache", 3},
		{"nginx", "nginx.html", "nginx/1.25.3", "nginx", 4},
		{"lighttpd (server header)", "lighttpd.html", "lighttpd/1.4.59", "lighttpd", 4},
		{"lighttpd (footer only)", "lighttpd.html", "", "lighttpd", 4},
		{"python", "python.html", "SimpleHTTP/0.6 Python/3.11.0", "python-http-server", 4},
		{"python (no header)", "python.html", "", "python-http-server", 4},
		{"iis", "iis.html", "Microsoft-IIS/10.0", "iis", 4},
		{"iis (no header)", "iis.html", "", "iis", 4},
		{"caddy", "caddy.html", "Caddy", "caddy", 4},
		{"caddy (footer only)", "caddy.html", "", "caddy", 4},
		{"tomcat", "tomcat.html", "Apache-Coyote/1.1", "tomcat", 3},
		{"tomcat (no header)", "tomcat.html", "", "tomcat", 3},
		{"jetty", "jetty.html", "Jetty(12.0.0)", "jetty", 3},
		{"jetty (no header)", "jetty.html", "", "jetty", 3},
		{"go fileserver", "go_fileserver.html", "", "go-fileserver", 4},
		{"generic fallback (title==h1)", "generic.html", "", "generic", 3},
		{"apache without server header falls back to generic", "apache.html", "", "generic", 4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := loadFixture(t, tc.fixture)
			p := Detect(doc, hdr(tc.server))
			if p == nil {
				t.Fatalf("no pattern matched")
			}
			if p.Name() != tc.wantPattern {
				t.Errorf("matched %q, want %q", p.Name(), tc.wantPattern)
			}
			entries := p.Entries(doc)
			if len(entries) != tc.wantEntries {
				t.Errorf("got %d entries (%+v), want %d", len(entries), entries, tc.wantEntries)
			}
		})
	}
}

// TestDetectEntryFlags spot-checks IsDir resolution on a couple of patterns
// where the distinction depends on row class rather than trailing slash.
func TestDetectEntryFlags(t *testing.T) {
	cases := []struct {
		fixture string
		server  string
		name    string
		href    string
		isDir   bool
	}{
		{"lighttpd.html", "lighttpd/1.4.59", "lighttpd dir", "docs/", true},
		{"lighttpd.html", "lighttpd/1.4.59", "lighttpd file", "README.txt", false},
		{"caddy.html", "Caddy", "caddy dir", "docs/", true},
		{"caddy.html", "Caddy", "caddy file", "README.txt", false},
		{"iis.html", "Microsoft-IIS/10.0", "iis dir", "/pub/docs/", true},
		{"iis.html", "Microsoft-IIS/10.0", "iis file", "/pub/README.txt", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := loadFixture(t, tc.fixture)
			p := Detect(doc, hdr(tc.server))
			if p == nil {
				t.Fatalf("no pattern matched for %s", tc.fixture)
			}
			got := hrefSet(p.Entries(doc))
			isDir, ok := got[tc.href]
			if !ok {
				t.Fatalf("href %q not in entries: %v", tc.href, got)
			}
			if isDir != tc.isDir {
				t.Errorf("href %q: got IsDir=%v, want %v", tc.href, isDir, tc.isDir)
			}
		})
	}
}

// TestDetectNoMatch confirms pages without listing markers return nil rather
// than being pulled into the generic fallback.
func TestDetectNoMatch(t *testing.T) {
	raw := `<html><head><title>Login</title></head><body><form><input name="user"></form></body></html>`
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader([]byte(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if p := Detect(doc, http.Header{}); p != nil {
		t.Errorf("expected no match, got %q", p.Name())
	}
}

// TestDetectConcurrent stresses concurrent Detect calls against the same
// shared document — verifies both goquery read-safety and Detect's
// fan-out/join internals.
func TestDetectConcurrent(t *testing.T) {
	doc := loadFixture(t, "lighttpd.html")
	headers := hdr("lighttpd/1.4.59")

	const N = 200
	names := make([]string, N)
	var wg sync.WaitGroup
	wg.Add(N)
	for i := range N {
		go func(i int) {
			defer wg.Done()
			if p := Detect(doc, headers); p != nil {
				names[i] = p.Name()
			}
		}(i)
	}
	wg.Wait()

	for i, n := range names {
		if n != "lighttpd" {
			t.Fatalf("run %d: got %q, want lighttpd", i, n)
		}
	}
}
