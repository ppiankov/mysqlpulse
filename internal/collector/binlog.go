package collector

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// Binlog collects binary log metrics.
type Binlog struct{}

func NewBinlog() *Binlog { return &Binlog{} }

func (b *Binlog) Name() string { return "binlog" }

func (b *Binlog) Collect(ctx context.Context, db *sql.DB, instance string) error {
	// Binary log file count and size.
	rows, err := db.QueryContext(ctx, "SHOW BINARY LOGS")
	if err != nil {
		// Binary logging may be disabled — not an error.
		return nil
	}
	defer func() { _ = rows.Close() }()

	var count float64
	var totalSize float64

	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("binlog columns: %w", err)
	}

	for rows.Next() {
		vals := make([]interface{}, len(cols))
		for i := range vals {
			vals[i] = new(sql.RawBytes)
		}
		if err := rows.Scan(vals...); err != nil {
			continue
		}
		count++
		// File_size is typically the second column.
		if len(vals) >= 2 {
			b := vals[1].(*sql.RawBytes)
			var size float64
			if _, err := fmt.Sscanf(string(*b), "%f", &size); err == nil {
				totalSize += size
			}
		}
	}

	metrics.BinlogCount.WithLabelValues(instance).Set(count)
	metrics.BinlogSizeBytes.WithLabelValues(instance).Set(totalSize)

	// Cache stats from SHOW GLOBAL STATUS.
	status, err := GlobalStatus(ctx, db)
	if err != nil {
		return err
	}

	for _, m := range []struct {
		gauge *prometheus.GaugeVec
		key   string
	}{
		{metrics.BinlogCacheUseTotal, "Binlog_cache_use"},
		{metrics.BinlogCacheDiskUseTotal, "Binlog_cache_disk_use"},
	} {
		if v, ok := status[m.key]; ok {
			m.gauge.WithLabelValues(instance).Set(v)
		}
	}

	return rows.Err()
}
