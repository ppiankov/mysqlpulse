package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the runtime configuration for mysqlpulse.
type Config struct {
	DSNs         []string
	MetricsPort  int
	PollInterval time.Duration
	Format       string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		DSNs:         []string{"root@tcp(localhost:3306)/"},
		MetricsPort:  9104,
		PollInterval: 15 * time.Second,
		Format:       "table",
	}
}

// FromEnv loads configuration from environment variables, falling back to defaults.
func FromEnv() (Config, error) {
	cfg := DefaultConfig()

	if v := os.Getenv("MYSQL_DSN"); v != "" {
		dsns := strings.Split(v, ",")
		cleaned := make([]string, 0, len(dsns))
		for _, d := range dsns {
			d = strings.TrimSpace(d)
			if d != "" {
				cleaned = append(cleaned, d)
			}
		}
		if len(cleaned) > 0 {
			cfg.DSNs = cleaned
		}
	}

	if v := os.Getenv("METRICS_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid METRICS_PORT %q: %w", v, err)
		}
		if port < 1 || port > 65535 {
			return cfg, fmt.Errorf("METRICS_PORT %d out of range 1-65535", port)
		}
		cfg.MetricsPort = port
	}

	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid POLL_INTERVAL %q: %w", v, err)
		}
		if d < 1*time.Second {
			return cfg, fmt.Errorf("POLL_INTERVAL %s too short, minimum 1s", d)
		}
		cfg.PollInterval = d
	}

	return cfg, nil
}
