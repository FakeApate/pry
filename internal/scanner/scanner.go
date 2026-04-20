package scanner

import (
	"bytes"
	"context"
	"crypto/tls"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"
	"github.com/fakeapate/mullvadproxy"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/fakeapate/pry/config"
	"github.com/fakeapate/pry/internal/scanner/patterns"
	"github.com/fakeapate/pry/model"
)

type Scanner struct {
	collector  *colly.Collector
	cfg        config.ScannerConfig
	onProgress ProgressFunc
}

func NewScanner(ctx context.Context, cfg config.ScannerConfig, mullvadCfg mullvadproxy.MullvadConfig, progress ProgressFunc) (*Scanner, error) {
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: cfg.Parallelism})
	c.SetRequestTimeout(cfg.RequestTimeout)
	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})
	extensions.RandomUserAgent(c)

	connected, err := mullvadproxy.IsConnected(ctx)
	if err != nil {
		log.Warn("Mullvad connectivity check failed, running without proxies!", "err", err)
	} else if connected {
		updaterErrs := mullvadproxy.StartUpdater(ctx, mullvadCfg)
		go func() {
			for err := range updaterErrs {
				log.Warn("Mullvad relay update failed", "err", err)
			}
		}()
		proxies, err := mullvadproxy.SelectProxies(mullvadCfg, cfg.ProxyCount, mullvadproxy.RelayFilter{
			Weight: func(num int) bool {
				return num <= cfg.MaxRelayWeight
			},
		})
		if err != nil {
			log.Warn("Failed to select proxies, running without proxies", "err", err)
		} else if p, err := proxy.RoundRobinProxySwitcher(proxies...); err == nil {
			c.SetProxyFunc(p)
		}
	} else {
		log.Warn("Not connected to Mullvad, running without proxies")
	}

	return &Scanner{collector: c, cfg: cfg, onProgress: progress}, nil
}

func tagging(f *model.ScanFinding, headers *http.Header) {
	for k, v := range *headers {
		switch k {
		case "Content-Length":
			if len(v) > 1 {
				log.Debugf("Multiple values in header\n%s", v)
			}
			f.ContentLength, _ = strconv.ParseInt(v[0], 10, 64)
		case "Content-Type":
			if len(v) > 1 {
				log.Debugf("Multiple values in header\n%s", v)
			}
			f.ContentType = v[0]
		case "Last-Modified":
			if len(v) > 1 {
				log.Debugf("Multiple values in header\n%s", v)
			}
			parsed, err := time.Parse(time.RFC1123, v[0])
			if err == nil {
				f.LastModified = parsed
			}
		case "Date":
			if len(v) > 1 {
				log.Debugf("Multiple values in header\n%s", v)
			}
			parsed, err := time.Parse(time.RFC1123, v[0])
			if err == nil {
				f.ScanTime = parsed
			}
		}
	}
}

// newBareScanner builds a Scanner without Mullvad integration. Used by tests
// so they don't depend on the host's Mullvad state and don't make live
// requests to am.i.mullvad.net.
func newBareScanner(cfg config.ScannerConfig, progress ProgressFunc) *Scanner {
	c := colly.NewCollector(colly.Async(true))
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: cfg.Parallelism})
	c.SetRequestTimeout(cfg.RequestTimeout)
	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})
	extensions.RandomUserAgent(c)
	return &Scanner{collector: c, cfg: cfg, onProgress: progress}
}

// isTransient reports whether an HTTP response status warrants a retry and
// a warning: 429 (rate-limited) or any 5xx (server error).
func isTransient(status int) bool {
	return status == http.StatusTooManyRequests || (status >= 500 && status < 600)
}

func (w *Scanner) RunScan(ctx context.Context, scanID string, scanURL string) (model.ScanResult, error) {
	c := w.collector.Clone()

	var mu sync.Mutex
	var discoveredURLs []string
	startTime := time.Now()

	// Atomic counters for progress reporting
	var folderCount atomic.Int64
	var skippedCount atomic.Int64
	var errorCount atomic.Int64
	var warningCount atomic.Int64
	var findingCount atomic.Int64
	var totalBytes atomic.Int64

	emitProgress := func() {
		if w.onProgress != nil {
			w.onProgress(ProgressEvent{
				ScanID:       scanID,
				FolderCount:  int(folderCount.Load()),
				FindingCount: int(findingCount.Load()),
				SkippedCount: int(skippedCount.Load()),
				ErrorCount:   int(errorCount.Load()),
				WarningCount: int(warningCount.Load()),
				TotalBytes:   totalBytes.Load(),
			})
		}
	}

	// Start progress reporter
	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				emitProgress()
			}
		}
	}()

	cancelled := false
	c.OnRequest(func(r *colly.Request) {
		if ctx.Err() != nil {
			r.Abort()
			cancelled = true
			return
		}
		log.Debugf("Visiting %s", r.URL)
	})

	c.OnResponse(func(r *colly.Response) {
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(r.Body))
		if err != nil {
			log.Error(err)
			return
		}

		headers := http.Header{}
		if r.Headers != nil {
			headers = *r.Headers
		}
		p := patterns.Detect(doc, headers)
		if p == nil {
			log.Error("Open directory validation failed", "url", r.Request.URL)
			return
		}
		log.Debug("Matched index pattern", "pattern", p.Name(), "url", r.Request.URL)

		for _, e := range p.Entries(doc) {
			if e.IsDir {
				if w.visitSubdir(r.Request, e.Href) {
					folderCount.Add(1)
				} else {
					skippedCount.Add(1)
				}
			} else {
				u := w.filterFinding(*r.Request.URL.JoinPath(e.Href))
				if u == "" {
					skippedCount.Add(1)
				} else {
					mu.Lock()
					discoveredURLs = append(discoveredURLs, u)
					mu.Unlock()
				}
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		status := r.StatusCode
		if status == http.StatusForbidden {
			log.Warn("Skipping forbidden URL", "url", r.Request.URL)
			return
		}
		if isTransient(status) {
			warningCount.Add(1)
			attempts, _ := strconv.Atoi(r.Ctx.Get("retry_attempts"))
			if attempts < w.cfg.RetryCount {
				backoff := time.Duration(attempts+1) * w.cfg.RetryBackoff
				log.Warn("Transient failure, retrying",
					"url", r.Request.URL, "status", status, "attempt", attempts+1, "backoff", backoff)
				r.Ctx.Put("retry_attempts", strconv.Itoa(attempts+1))
				time.Sleep(backoff)
				if err := r.Request.Retry(); err == nil {
					return
				}
			}
		}
		log.Error("Request failed", "url", r.Request.URL, "status", status, "err", err)
		errorCount.Add(1)
	})

	c.Visit(scanURL)
	c.Wait()

	if cancelled {
		return model.ScanResult{}, ctx.Err()
	}

	// Phase 2: tag discovered URLs with HEAD requests
	tagC := w.collector.Clone()
	var findings []model.ScanFinding

	tagC.OnRequest(func(r *colly.Request) {
		if ctx.Err() != nil {
			r.Abort()
		}
	})

	tagC.OnResponseHeaders(func(r *colly.Response) {
		if r.StatusCode < 200 || r.StatusCode >= 300 {
			return
		}
		f := model.ScanFinding{
			Url: r.Request.URL.String(),
		}
		tagging(&f, r.Headers)
		mu.Lock()
		findings = append(findings, f)
		mu.Unlock()

		findingCount.Add(1)
		totalBytes.Add(f.ContentLength)
	})

	tagC.OnError(func(r *colly.Response, err error) {
		status := r.StatusCode
		if status == http.StatusForbidden {
			return
		}
		if isTransient(status) {
			warningCount.Add(1)
			attempts, _ := strconv.Atoi(r.Ctx.Get("retry_attempts"))
			if attempts < w.cfg.RetryCount {
				backoff := time.Duration(attempts+1) * w.cfg.RetryBackoff
				r.Ctx.Put("retry_attempts", strconv.Itoa(attempts+1))
				time.Sleep(backoff)
				if err := r.Request.Retry(); err == nil {
					return
				}
			}
		}
		errorCount.Add(1)
	})

	for _, u := range discoveredURLs {
		tagC.Head(u)
	}
	tagC.Wait()

	emitProgress()

	stats := model.ScanStats{
		FindingCount: int(findingCount.Load()),
		FolderCount:  int(folderCount.Load()),
		SkippedCount: int(skippedCount.Load()),
		ErrorCount:   int(errorCount.Load()),
		WarningCount: int(warningCount.Load()),
		TotalBytes:   totalBytes.Load(),
		DurationMs:   time.Since(startTime).Milliseconds(),
	}

	return model.ScanResult{Stats: stats, Findings: findings}, nil
}

func (w *Scanner) filterFinding(u url.URL) string {
	ext := filepath.Ext(u.String())
	filetype := mime.TypeByExtension(ext)
	for _, prefix := range w.cfg.SkipMimePrefixes {
		if strings.HasPrefix(filetype, prefix) {
			return ""
		}
	}
	log.Debug("Adding finding", "file", filepath.Base(u.String()), "mime", filetype)
	return u.String()
}

// visitSubdir visits href as a subdirectory and returns true if it was visited,
// false if it was skipped due to a keyword match.
func (w *Scanner) visitSubdir(r *colly.Request, href string) bool {
	abs := r.URL.JoinPath(href).String()
	for _, keyword := range w.cfg.SkipSubdirKeywords {
		if strings.Contains(abs, keyword) {
			return false
		}
	}
	r.Visit(href)
	return true
}
