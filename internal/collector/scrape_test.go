package collector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

func TestScrape_Name(t *testing.T) {
	s := NewScrape()
	if s.Name() != "scrape" {
		t.Fatalf("expected scrape, got %s", s.Name())
	}
}

func TestScrape_ImplementsCollector(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	var _ Collector = NewScrape()
}
