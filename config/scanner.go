package config

import "time"

type ScannerConfig struct {
	Parallelism        int           `toml:"parallelism"`
	RequestTimeout     time.Duration `toml:"request_timeout"`
	ProxyCount         int           `toml:"proxy_count"`
	MaxRelayWeight     int           `toml:"max_relay_weight"`
	RetryCount         int           `toml:"retry_count"`
	RetryBackoff       time.Duration `toml:"retry_backoff"`
	SkipMimePrefixes   []string      `toml:"skip_mime_prefixes"`
	SkipSubdirKeywords []string      `toml:"skip_subdir_keywords"`
}

func DefaultScannerConfig() ScannerConfig {
	return ScannerConfig{
		Parallelism:    32,
		RequestTimeout: 15 * time.Second,
		ProxyCount:     50,
		MaxRelayWeight: 99,
		RetryCount:     2,
		RetryBackoff:   time.Second,
		SkipMimePrefixes: []string{
			"image", "font", "text/css", "audio", "video",
		},
		SkipSubdirKeywords: []string{
			".git", ".svn", ".hg",
			"node_modules", "bower_components",
			"venv", "vendor",
			"__pycache__", "site-packages",
			".cache", ".idea",
		},
	}
}
