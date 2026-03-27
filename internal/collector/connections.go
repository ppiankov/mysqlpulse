package collector

import (
	"context"
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// Connections collects connection-related metrics from SHOW GLOBAL STATUS.
type Connections struct{}

func NewConnections() *Connections { return &Connections{} }

func (c *Connections) Name() string { return "connections" }

func (c *Connections) Collect(ctx context.Context, db *sql.DB, instance string) error {
	status, err := GlobalStatus(ctx, db)
	if err != nil {
		return err
	}

	for _, m := range []struct {
		gauge *prometheus.GaugeVec
		key   string
	}{
		{metrics.ThreadsConnected, "Threads_connected"},
		{metrics.ThreadsRunning, "Threads_running"},
		{metrics.ThreadsCached, "Threads_cached"},
		{metrics.MaxUsedConnections, "Max_used_connections"},
		{metrics.ConnectionsTotal, "Connections"},
		{metrics.AbortedConnectsTotal, "Aborted_connects"},
		{metrics.AbortedClientsTotal, "Aborted_clients"},
	} {
		if v, ok := status[m.key]; ok {
			m.gauge.WithLabelValues(instance).Set(v)
		}
	}

	return nil
}
