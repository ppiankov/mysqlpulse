package collector

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// GlobalVars collects key configuration variables as metrics.
type GlobalVars struct{}

func NewGlobalVars() *GlobalVars { return &GlobalVars{} }

func (g *GlobalVars) Name() string { return "globalvars" }

func (g *GlobalVars) Collect(ctx context.Context, db *sql.DB, instance string) error {
	vars, err := queryGlobalVars(ctx, db)
	if err != nil {
		return err
	}

	// Numeric variables.
	for _, m := range []struct {
		gauge *prometheus.GaugeVec
		key   string
	}{
		{metrics.MaxConnections, "max_connections"},
		{metrics.InnoDBBufferPoolSizeBytes, "innodb_buffer_pool_size"},
	} {
		if v, ok := vars[m.key]; ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				m.gauge.WithLabelValues(instance).Set(f)
			}
		}
	}

	// Boolean ON/OFF variables.
	for _, m := range []struct {
		gauge *prometheus.GaugeVec
		key   string
	}{
		{metrics.ReadOnly, "read_only"},
		{metrics.SuperReadOnly, "super_read_only"},
	} {
		if v, ok := vars[m.key]; ok {
			m.gauge.WithLabelValues(instance).Set(onOffToFloat(v))
		}
	}

	// GTID mode.
	if v, ok := vars["gtid_mode"]; ok {
		metrics.GTIDMode.WithLabelValues(instance).Set(onOffToFloat(v))
	}

	return nil
}

func queryGlobalVars(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, "SHOW GLOBAL VARIABLES")
	if err != nil {
		return nil, fmt.Errorf("SHOW GLOBAL VARIABLES: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		result[name] = value
	}
	return result, rows.Err()
}

func onOffToFloat(v string) float64 {
	if strings.EqualFold(v, "ON") {
		return 1
	}
	return 0
}
