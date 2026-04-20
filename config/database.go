package config

type DatabaseConfig struct {
	DBPath string `toml:"db_path"`
}

func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		DBPath: "pry.db",
	}
}
