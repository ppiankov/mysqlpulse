package collector

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

const topNQueries = 10

// PerfSchema collects top-N query metrics from performance_schema.
type PerfSchema struct{}

func NewPerfSchema() *PerfSchema { return &PerfSchema{} }

func (p *PerfSchema) Name() string { return "perfschema" }

func (p *PerfSchema) Collect(ctx context.Context, db *sql.DB, instance string) error {
	const query = `SELECT IFNULL(DIGEST_TEXT, ''), COUNT_STAR,
		AVG_TIMER_WAIT / 1000000000000.0,
		SUM_ROWS_EXAMINED
		FROM performance_schema.events_statements_summary_by_digest
		ORDER BY SUM_TIMER_WAIT DESC
		LIMIT ?`

	rows, err := db.QueryContext(ctx, query, topNQueries)
	if err != nil {
		// performance_schema may be disabled.
		return nil
	}
	defer func() { _ = rows.Close() }()

	// Reset to clear stale digests.
	metrics.PerfQueryAvgSeconds.Reset()
	metrics.PerfQueryCalls.Reset()
	metrics.PerfQueryRowsExamined.Reset()

	for rows.Next() {
		var digest string
		var calls, rowsExamined float64
		var avgSec float64

		if err := rows.Scan(&digest, &calls, &avgSec, &rowsExamined); err != nil {
			continue
		}

		// Truncate digest for label cardinality.
		if len(digest) > 64 {
			digest = digest[:64] + "..."
		}
		if digest == "" {
			digest = "<unknown>"
		}

		metrics.PerfQueryAvgSeconds.WithLabelValues(instance, digest).Set(avgSec)
		metrics.PerfQueryCalls.WithLabelValues(instance, digest).Set(calls)
		metrics.PerfQueryRowsExamined.WithLabelValues(instance, digest).Set(rowsExamined)
	}

	return rows.Err()
}

// DigestInfo is a structured query digest for the report command.
type DigestInfo struct {
	Digest       string  `json:"digest"`
	Calls        float64 `json:"calls"`
	AvgSeconds   float64 `json:"avg_seconds"`
	RowsExamined float64 `json:"rows_examined"`
}

// TopQueries returns the top-N queries for use in report/innodb commands.
func TopQueries(ctx context.Context, db *sql.DB, n int) ([]DigestInfo, error) {
	const query = `SELECT IFNULL(DIGEST_TEXT, ''), COUNT_STAR,
		AVG_TIMER_WAIT / 1000000000000.0,
		SUM_ROWS_EXAMINED
		FROM performance_schema.events_statements_summary_by_digest
		ORDER BY SUM_TIMER_WAIT DESC
		LIMIT ?`

	rows, err := db.QueryContext(ctx, query, n)
	if err != nil {
		return nil, fmt.Errorf("perf schema: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []DigestInfo
	for rows.Next() {
		var d DigestInfo
		if err := rows.Scan(&d.Digest, &d.Calls, &d.AvgSeconds, &d.RowsExamined); err != nil {
			continue
		}
		result = append(result, d)
	}
	return result, rows.Err()
}
