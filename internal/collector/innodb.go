package collector

import (
	"context"
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// InnoDB collects InnoDB metrics from SHOW GLOBAL STATUS.
type InnoDB struct{}

func NewInnoDB() *InnoDB { return &InnoDB{} }

func (i *InnoDB) Name() string { return "innodb" }

func (i *InnoDB) Collect(ctx context.Context, db *sql.DB, instance string) error {
	status, err := GlobalStatus(ctx, db)
	if err != nil {
		return err
	}

	// Buffer pool pages.
	for _, p := range []struct {
		gauge *prometheus.GaugeVec
		state string
		key   string
	}{
		{metrics.InnoDBBufferPoolPages, "total", "Innodb_buffer_pool_pages_total"},
		{metrics.InnoDBBufferPoolPages, "free", "Innodb_buffer_pool_pages_free"},
		{metrics.InnoDBBufferPoolPages, "dirty", "Innodb_buffer_pool_pages_dirty"},
		{metrics.InnoDBBufferPoolPages, "data", "Innodb_buffer_pool_pages_data"},
	} {
		if v, ok := status[p.key]; ok {
			p.gauge.WithLabelValues(instance, p.state).Set(v)
		}
	}

	// Buffer pool bytes.
	for _, p := range []struct {
		state string
		key   string
	}{
		{"data", "Innodb_buffer_pool_bytes_data"},
		{"dirty", "Innodb_buffer_pool_bytes_dirty"},
	} {
		if v, ok := status[p.key]; ok {
			metrics.InnoDBBufferPoolBytes.WithLabelValues(instance, p.state).Set(v)
		}
	}

	// Hit ratio: 1 - (reads / read_requests). Avoid division by zero.
	reads := status["Innodb_buffer_pool_reads"]
	requests := status["Innodb_buffer_pool_read_requests"]
	if requests > 0 {
		metrics.InnoDBBufferPoolHitRatio.WithLabelValues(instance).Set(1 - reads/requests)
	}

	// Cumulative counters as gauges.
	setGauge := func(g *prometheus.GaugeVec, key string) {
		if v, ok := status[key]; ok {
			g.WithLabelValues(instance).Set(v)
		}
	}

	setGauge(metrics.InnoDBRowLockWaitsTotal, "Innodb_row_lock_waits")
	setGauge(metrics.InnoDBDeadlocksTotal, "Innodb_deadlocks")
	setGauge(metrics.InnoDBHistoryListLength, "Innodb_history_list_length")

	return nil
}
