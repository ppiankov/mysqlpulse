package collector

import (
	"context"
	"database/sql"
	"time"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// Scrape checks basic MySQL connectivity and records scrape health metrics.
type Scrape struct{}

func NewScrape() *Scrape { return &Scrape{} }

func (s *Scrape) Name() string { return "scrape" }

func (s *Scrape) Collect(ctx context.Context, db *sql.DB, instance string) error {
	start := time.Now()
	err := db.PingContext(ctx)
	elapsed := time.Since(start).Seconds()

	metrics.ScrapeDuration.WithLabelValues(instance).Set(elapsed)

	if err != nil {
		metrics.MySQLUp.WithLabelValues(instance).Set(0)
		metrics.ScrapeErrors.WithLabelValues(instance).Inc()
		return err
	}

	metrics.MySQLUp.WithLabelValues(instance).Set(1)
	return nil
}
