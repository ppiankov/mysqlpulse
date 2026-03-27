package collector

import (
	"context"
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// Queries collects query throughput metrics from SHOW GLOBAL STATUS.
type Queries struct{}

func NewQueries() *Queries { return &Queries{} }

func (q *Queries) Name() string { return "queries" }

func (q *Queries) Collect(ctx context.Context, db *sql.DB, instance string) error {
	status, err := GlobalStatus(ctx, db)
	if err != nil {
		return err
	}

	setGauge := func(g *prometheus.GaugeVec, key string) {
		if v, ok := status[key]; ok {
			g.WithLabelValues(instance).Set(v)
		}
	}

	setGauge(metrics.QueriesTotal, "Queries")
	setGauge(metrics.QuestionsTotal, "Questions")
	setGauge(metrics.SlowQueriesTotal, "Slow_queries")
	setGauge(metrics.SelectFullJoinTotal, "Select_full_join")
	setGauge(metrics.SortMergePassesTotal, "Sort_merge_passes")

	// Per-command counters.
	for _, cmd := range []string{"select", "insert", "update", "delete"} {
		key := "Com_" + cmd
		if v, ok := status[key]; ok {
			metrics.CommandsTotal.WithLabelValues(instance, cmd).Set(v)
		}
	}

	return nil
}
