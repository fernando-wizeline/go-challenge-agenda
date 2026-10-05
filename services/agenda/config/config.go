package config

import (
	"os"
)

type Config struct {
	GRPCAddr string
	DBDriver string // "sqlite3" or "postgres"
	DBSource string

	MetricsAddr string // address of the Prometheus /metrics endpoint

	LogLevel  string // "debug", "info", "warn" or "error"
	LogFormat string // "json" or "text"
}

func Load() Config {
	c := Config{
		GRPCAddr: getenv("AGENDA_GRPC_ADDR", ":50051"),
		DBDriver: getenv("DB_DRIVER", "sqlite3"),
		DBSource: getenv("DB_SOURCE", "agenda.db"),

		MetricsAddr: getenv("AGENDA_METRICS_ADDR", ":9090"),

		LogLevel:  getenv("LOG_LEVEL", "info"),
		LogFormat: getenv("LOG_FORMAT", "json"),
	}
	return c
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
