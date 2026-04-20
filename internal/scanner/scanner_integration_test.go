package scanner

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fakeapate/pry/config"
)

func openDirHTML(entries ...string) string {
	var sb strings.Builder
	sb.WriteString(`<html><head><title>/test/</title></head><body><h1>/test/</h1><ul>`)
	for _, e := range entries {
		fmt.Fprintf(&sb, `<li><a href="%s">%s</a></li>`, e, e)
	}
	sb.WriteString(`</ul></body></html>`)
	return sb.String()
}

// TestScannerFindsFiles -- basic scan against a static open directory.
func TestScannerFindsFiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, openDirHTML("report.pdf", "data.sql", "readme.txt"))
		case "/test/report.pdf":
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Length", "2048")
			w.WriteHeader(http.StatusOK)
		case "/test/data.sql":
			w.Header().Set("Content-Type", "application/sql")
			w.Header().Set("Content-Length", "8192")
			w.WriteHeader(http.StatusOK)
		case "/test/readme.txt":
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "64")
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cfg := config.DefaultScannerConfig()
	cfg.Parallelism = 4
	cfg.RequestTimeout = 2 * time.Second
	cfg.RetryCount = 0

	s := newBareScanner(cfg, nil)
	result, err := s.RunScan(context.Background(), "test-1", srv.URL+"/test/")
	if err != nil {
		t.Fatalf("RunScan: %v", err)
	}
	if result.Stats.FindingCount != 3 {
		t.Errorf("expected 3 findings, got %d", result.Stats.FindingCount)
	}
	if result.Stats.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", result.Stats.ErrorCount)
	}
	if result.Stats.WarningCount != 0 {
		t.Errorf("expected 0 warnings, got %d", result.Stats.WarningCount)
	}
}

// TestScannerRetryOnTransient -- a 500 followed by 200 should produce 1 warning
// and ultimately succeed (via retry). Similarly for 429.
func TestScannerRetryOnTransient(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, openDirHTML("file.pdf"))
		case "/test/file.pdf":
			n := atomic.AddInt32(&hits, 1)
			if n == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	cfg := config.DefaultScannerConfig()
	cfg.Parallelism = 1
	cfg.RequestTimeout = 2 * time.Second
	cfg.RetryCount = 2
	cfg.RetryBackoff = 10 * time.Millisecond

	s := newBareScanner(cfg, nil)
	result, err := s.RunScan(context.Background(), "test-retry", srv.URL+"/test/")
	if err != nil {
		t.Fatalf("RunScan: %v", err)
	}
	if result.Stats.FindingCount != 1 {
		t.Errorf("expected 1 finding after retry, got %d", result.Stats.FindingCount)
	}
	if result.Stats.WarningCount < 1 {
		t.Errorf("expected >=1 warning (for the 500), got %d", result.Stats.WarningCount)
	}
	if result.Stats.ErrorCount != 0 {
		t.Errorf("expected 0 errors (retry succeeded), got %d", result.Stats.ErrorCount)
	}
}

// TestScannerExhaustsRetries -- always 500 exhausts retries and counts an error.
func TestScannerExhaustsRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, openDirHTML("broken.pdf"))
		case "/test/broken.pdf":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	cfg := config.DefaultScannerConfig()
	cfg.Parallelism = 1
	cfg.RequestTimeout = 2 * time.Second
	cfg.RetryCount = 2
	cfg.RetryBackoff = 10 * time.Millisecond

	s := newBareScanner(cfg, nil)
	result, err := s.RunScan(context.Background(), "test-exhaust", srv.URL+"/test/")
	if err != nil {
		t.Fatalf("RunScan: %v", err)
	}
	if result.Stats.FindingCount != 0 {
		t.Errorf("expected 0 findings, got %d", result.Stats.FindingCount)
	}
	if result.Stats.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", result.Stats.ErrorCount)
	}
	// Warnings should include each 500: initial + 2 retries = 3
	if result.Stats.WarningCount < 3 {
		t.Errorf("expected >=3 warnings (500s across all attempts), got %d", result.Stats.WarningCount)
	}
}

// TestScannerPartialFailure -- mix of 200 and 404 should complete without
// a scanner-level error; 404 is not transient so it doesn't count as a warning,
// just an error.
func TestScannerPartialFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, openDirHTML("ok.pdf", "missing.pdf"))
		case "/test/ok.pdf":
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cfg := config.DefaultScannerConfig()
	cfg.Parallelism = 2
	cfg.RequestTimeout = 2 * time.Second
	cfg.RetryCount = 0

	s := newBareScanner(cfg, nil)
	result, err := s.RunScan(context.Background(), "test-partial", srv.URL+"/test/")
	if err != nil {
		t.Fatalf("RunScan should not return error on partial failure, got: %v", err)
	}
	if result.Stats.FindingCount != 1 {
		t.Errorf("expected 1 finding, got %d", result.Stats.FindingCount)
	}
	if result.Stats.ErrorCount != 1 {
		t.Errorf("expected 1 error (for the 404), got %d", result.Stats.ErrorCount)
	}
	if result.Stats.WarningCount != 0 {
		t.Errorf("expected 0 warnings (404 is not transient), got %d", result.Stats.WarningCount)
	}
}

// TestScannerRateLimit429 -- 429 Too Many Requests is counted as a warning.
func TestScannerRateLimit429(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, openDirHTML("file.pdf"))
		case "/test/file.pdf":
			n := atomic.AddInt32(&hits, 1)
			if n == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Length", "50")
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	cfg := config.DefaultScannerConfig()
	cfg.Parallelism = 1
	cfg.RequestTimeout = 2 * time.Second
	cfg.RetryCount = 1
	cfg.RetryBackoff = 10 * time.Millisecond

	s := newBareScanner(cfg, nil)
	result, err := s.RunScan(context.Background(), "test-429", srv.URL+"/test/")
	if err != nil {
		t.Fatalf("RunScan: %v", err)
	}
	if result.Stats.FindingCount != 1 {
		t.Errorf("expected 1 finding after retry, got %d", result.Stats.FindingCount)
	}
	if result.Stats.WarningCount < 1 {
		t.Errorf("expected >=1 warning for 429, got %d", result.Stats.WarningCount)
	}
}
