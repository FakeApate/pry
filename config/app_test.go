package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidate_DefaultIsValid(t *testing.T) {
	if err := DefaultAppConfig().Validate(); err != nil {
		t.Fatalf("default config should be valid, got: %v", err)
	}
}

func TestValidate_RejectsInvalid(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*AppConfig)
		wantSub string
	}{
		{
			name:    "workers zero",
			mutate:  func(c *AppConfig) { c.Workers = 0 },
			wantSub: "workers must be >= 1",
		},
		{
			name:    "workers negative",
			mutate:  func(c *AppConfig) { c.Workers = -1 },
			wantSub: "workers must be >= 1",
		},
		{
			name:    "parallelism zero",
			mutate:  func(c *AppConfig) { c.Scanner.Parallelism = 0 },
			wantSub: "scanner.parallelism must be >= 1",
		},
		{
			name:    "request timeout zero",
			mutate:  func(c *AppConfig) { c.Scanner.RequestTimeout = 0 },
			wantSub: "scanner.request_timeout must be > 0",
		},
		{
			name:    "retry count negative",
			mutate:  func(c *AppConfig) { c.Scanner.RetryCount = -1 },
			wantSub: "scanner.retry_count must be >= 0",
		},
		{
			name:    "retry backoff negative",
			mutate:  func(c *AppConfig) { c.Scanner.RetryBackoff = -time.Second },
			wantSub: "scanner.retry_backoff must be >= 0",
		},
		{
			name:    "empty db path with db enabled",
			mutate:  func(c *AppConfig) { c.Database.DBPath = "" },
			wantSub: "database.db_path must not be empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultAppConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantSub)
			}
		})
	}
}

func TestValidate_EmptyDBPathAllowedWhenDisabled(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Database.DBPath = ""
	cfg.DisableDatabase = true
	if err := cfg.Validate(); err != nil {
		t.Fatalf("empty db_path should be allowed when db is disabled, got: %v", err)
	}
}

func TestValidate_RetryCountZeroAllowed(t *testing.T) {
	cfg := DefaultAppConfig()
	cfg.Scanner.RetryCount = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("retry count 0 should be allowed (disable retries), got: %v", err)
	}
}
