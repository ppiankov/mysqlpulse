package engine

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/ppiankov/mysqlpulse/internal/alerter"
	"github.com/ppiankov/mysqlpulse/internal/collector"
	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

const maxRetries = 3

// Engine runs the poll loop, collecting metrics at a fixed interval.
type Engine struct {
	interval   time.Duration
	targets    []Target
	collectors []collector.Collector
	alerter    *alerter.Alerter
}

// Target pairs a DSN label with its database handle.
type Target struct {
	Instance string
	DSN      string
	DB       *sql.DB
}

// New creates an Engine.
func New(interval time.Duration, targets []Target, collectors []collector.Collector) *Engine {
	return &Engine{
		interval:   interval,
		targets:    targets,
		collectors: collectors,
	}
}

// SetAlerter configures the alerter for the engine.
func (e *Engine) SetAlerter(a *alerter.Alerter) {
	e.alerter = a
}

// Run starts the poll loop. Blocks until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) error {
	e.poll(ctx) // immediate first scrape

	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			e.poll(ctx)
		}
	}
}

func (e *Engine) poll(ctx context.Context) {
	for _, t := range e.targets {
		if err := PingWithRetry(ctx, t.DB, maxRetries); err != nil {
			metrics.MySQLUp.WithLabelValues(t.Instance).Set(0)
			metrics.ScrapeErrors.WithLabelValues(t.Instance).Inc()
			log.Printf("target %s unreachable after retries: %v", t.Instance, err)
			continue
		}

		for _, c := range e.collectors {
			if err := c.Collect(ctx, t.DB, t.Instance); err != nil {
				log.Printf("collector %s on %s: %v", c.Name(), t.Instance, err)
			}
		}

		// Evaluate alert conditions after collecting.
		alerter.Evaluate(ctx, t.DB, t.DSN, e.alerter)
	}
}
