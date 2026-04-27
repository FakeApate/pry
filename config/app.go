package config

import (
	"fmt"

	"github.com/fakeapate/mullvadproxy"
)

type AppConfig struct {
	Database        DatabaseConfig             `toml:"Database"`
	Scanner         ScannerConfig              `toml:"Scanner"`
	Mullvad         mullvadproxy.MullvadConfig `toml:"Mullvad"`
	Workers         int                        `toml:"workers"`
	DisableDatabase bool                       `toml:"disable_database"`
}

func DefaultAppConfig() AppConfig {
	return AppConfig{
		Database:        DefaultDatabaseConfig(),
		Scanner:         DefaultScannerConfig(),
		Mullvad:         mullvadproxy.DefaultMullvadConfig(),
		Workers:         1,
		DisableDatabase: false,
	}
}

// Validate checks that the config has usable values. It returns a descriptive
// error on the first problem it finds.
func (c AppConfig) Validate() error {
	if c.Workers < 1 {
		return fmt.Errorf("workers must be >= 1, got %d", c.Workers)
	}
	if c.Scanner.Parallelism < 1 {
		return fmt.Errorf("scanner.parallelism must be >= 1, got %d", c.Scanner.Parallelism)
	}
	if c.Scanner.RequestTimeout <= 0 {
		return fmt.Errorf("scanner.request_timeout must be > 0, got %s", c.Scanner.RequestTimeout)
	}
	if c.Scanner.RetryCount < 0 {
		return fmt.Errorf("scanner.retry_count must be >= 0, got %d", c.Scanner.RetryCount)
	}
	if c.Scanner.RetryBackoff < 0 {
		return fmt.Errorf("scanner.retry_backoff must be >= 0, got %s", c.Scanner.RetryBackoff)
	}
	if !c.DisableDatabase && c.Database.DBPath == "" {
		return fmt.Errorf("database.db_path must not be empty when the database is enabled")
	}
	return nil
}

var config = DefaultAppConfig()

func GetConfig() *AppConfig {
	return &config
}

func SetConfig(cfg AppConfig) {
	config = cfg
}
