package scanner

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/fakeapate/pry/config"
	"github.com/fakeapate/pry/model"
)

func TestTaggingContentLength(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected int64
	}{
		{"valid", "12345", 12345},
		{"zero", "0", 0},
		{"large", "1073741824", 1073741824},
		{"invalid", "notanumber", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f model.ScanFinding
			h := http.Header{"Content-Length": {tt.header}}
			tagging(&f, &h)
			if f.ContentLength != tt.expected {
				t.Errorf("expected ContentLength=%d, got %d", tt.expected, f.ContentLength)
			}
		})
	}
}

func TestTaggingContentType(t *testing.T) {
	var f model.ScanFinding
	h := http.Header{"Content-Type": {"application/pdf"}}
	tagging(&f, &h)
	if f.ContentType != "application/pdf" {
		t.Errorf("expected ContentType=application/pdf, got %q", f.ContentType)
	}
}

func TestTaggingLastModified(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		expectSet bool
	}{
		{"valid RFC1123", "Mon, 15 Mar 2026 10:00:00 GMT", true},
		{"invalid", "not-a-date", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f model.ScanFinding
			h := http.Header{"Last-Modified": {tt.header}}
			tagging(&f, &h)
			if tt.expectSet && f.LastModified.IsZero() {
				t.Error("expected LastModified to be set")
			}
			if !tt.expectSet && !f.LastModified.IsZero() {
				t.Error("expected LastModified to be zero")
			}
		})
	}
}

func TestTaggingDate(t *testing.T) {
	var f model.ScanFinding
	h := http.Header{"Date": {"Thu, 10 Apr 2026 10:00:00 GMT"}}
	tagging(&f, &h)
	expected, _ := time.Parse(time.RFC1123, "Thu, 10 Apr 2026 10:00:00 GMT")
	if !f.ScanTime.Equal(expected) {
		t.Errorf("expected ScanTime=%v, got %v", expected, f.ScanTime)
	}
}

func TestTaggingAllHeaders(t *testing.T) {
	var f model.ScanFinding
	h := http.Header{
		"Content-Length": {"2048"},
		"Content-Type":  {"text/html; charset=utf-8"},
		"Last-Modified": {"Mon, 15 Mar 2026 10:00:00 GMT"},
		"Date":          {"Thu, 10 Apr 2026 10:00:00 GMT"},
	}
	tagging(&f, &h)

	if f.ContentLength != 2048 {
		t.Errorf("ContentLength: got %d, want 2048", f.ContentLength)
	}
	if f.ContentType != "text/html; charset=utf-8" {
		t.Errorf("ContentType: got %q", f.ContentType)
	}
	if f.LastModified.IsZero() {
		t.Error("LastModified should be set")
	}
	if f.ScanTime.IsZero() {
		t.Error("ScanTime should be set")
	}
}

func TestTaggingEmptyHeaders(t *testing.T) {
	var f model.ScanFinding
	h := http.Header{}
	tagging(&f, &h)

	if f.ContentLength != 0 {
		t.Errorf("expected ContentLength=0, got %d", f.ContentLength)
	}
	if f.ContentType != "" {
		t.Errorf("expected empty ContentType, got %q", f.ContentType)
	}
}

func TestFilterFindingAllowed(t *testing.T) {
	s := &Scanner{cfg: config.DefaultScannerConfig()}

	tests := []struct {
		name    string
		rawURL  string
		allowed bool
	}{
		{"pdf allowed", "https://example.com/report.pdf", true},
		{"sql allowed", "https://example.com/data.sql", true},
		{"exe allowed", "https://example.com/app.exe", true},
		{"no extension", "https://example.com/README", true},
		{"png skipped", "https://example.com/photo.png", false},
		{"jpg skipped", "https://example.com/photo.jpg", false},
		{"css skipped", "https://example.com/style.css", false},
		{"mp3 skipped", "https://example.com/song.mp3", false},
		{"mp4 skipped", "https://example.com/video.mp4", false},
		{"woff skipped", "https://example.com/font.woff", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.rawURL)
			if err != nil {
				t.Fatal(err)
			}
			result := s.filterFinding(*u)
			if tt.allowed && result == "" {
				t.Errorf("expected %q to be allowed", tt.rawURL)
			}
			if !tt.allowed && result != "" {
				t.Errorf("expected %q to be filtered out", tt.rawURL)
			}
		})
	}
}

func TestFilterFindingCustomSkip(t *testing.T) {
	s := &Scanner{
		cfg: config.ScannerConfig{
			SkipMimePrefixes: []string{"application/pdf"},
		},
	}
	u, _ := url.Parse("https://example.com/report.pdf")
	if result := s.filterFinding(*u); result != "" {
		t.Errorf("expected pdf to be filtered with custom skip rule")
	}
}

func TestPathSegmentMatches(t *testing.T) {
	kws := []string{".git", "node_modules", "venv"}
	tests := []struct {
		path string
		want bool
	}{
		{"/repo/.git/", true},
		{"/deep/nested/.git/objects/", true},
		{"/node_modules/pkg/file.js", true},
		{"/venv/", true},

		// Regression: substring matches on path fragments or hostnames must
		// not trip the filter.
		{"/my.github.io/site/", false},
		{"/logistics/", false},
		{"/nodes/", false},
		{"/venvironment/", false},
		{"/just/a/path/", false},
		{"/", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := pathSegmentMatches(tt.path, kws); got != tt.want {
				t.Errorf("pathSegmentMatches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
