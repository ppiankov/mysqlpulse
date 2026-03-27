package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.DSNs) != 1 {
		t.Fatalf("expected 1 default DSN, got %d", len(cfg.DSNs))
	}
	if cfg.MetricsPort != 9104 {
		t.Fatalf("expected port 9104, got %d", cfg.MetricsPort)
	}
	if cfg.PollInterval != 15*time.Second {
		t.Fatalf("expected 15s interval, got %s", cfg.PollInterval)
	}
}

func TestFromEnv(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		os.Unsetenv("MYSQL_DSN")
		os.Unsetenv("METRICS_PORT")
		os.Unsetenv("POLL_INTERVAL")

		cfg, err := FromEnv()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MetricsPort != 9104 {
			t.Errorf("expected port 9104, got %d", cfg.MetricsPort)
		}
	})

	t.Run("custom DSN", func(t *testing.T) {
		t.Setenv("MYSQL_DSN", "user:pass@tcp(db1:3306)/, user:pass@tcp(db2:3306)/")
		cfg, err := FromEnv()
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg.DSNs) != 2 {
			t.Fatalf("expected 2 DSNs, got %d", len(cfg.DSNs))
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		t.Setenv("METRICS_PORT", "abc")
		_, err := FromEnv()
		if err == nil {
			t.Fatal("expected error for invalid port")
		}
	})

	t.Run("port out of range", func(t *testing.T) {
		t.Setenv("METRICS_PORT", "99999")
		_, err := FromEnv()
		if err == nil {
			t.Fatal("expected error for out-of-range port")
		}
	})

	t.Run("invalid interval", func(t *testing.T) {
		t.Setenv("POLL_INTERVAL", "nope")
		_, err := FromEnv()
		if err == nil {
			t.Fatal("expected error for invalid interval")
		}
	})

	t.Run("interval too short", func(t *testing.T) {
		t.Setenv("POLL_INTERVAL", "500ms")
		_, err := FromEnv()
		if err == nil {
			t.Fatal("expected error for short interval")
		}
	})
}
