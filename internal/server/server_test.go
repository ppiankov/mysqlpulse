package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

func TestHealthz(t *testing.T) {
	reg := prometheus.NewRegistry()
	srv := New(":0", reg)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", body["status"])
	}
}

func TestMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	// Set metrics so we have something to verify
	metrics.MySQLUp.WithLabelValues("test:3306").Set(1)
	metrics.ScrapeDuration.WithLabelValues("test:3306").Set(0.01)
	metrics.ScrapeErrors.WithLabelValues("test:3306").Inc()

	srv := New(":0", reg)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	srv.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "mysql_up") {
		t.Fatal("expected mysql_up in metrics output")
	}
	if !strings.Contains(body, "mysql_scrape_duration_seconds") {
		t.Fatal("expected mysql_scrape_duration_seconds in metrics output")
	}
}
