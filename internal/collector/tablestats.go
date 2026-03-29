package collector

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// TableStats collects per-table metrics from information_schema.TABLES.
type TableStats struct{}

func NewTableStats() *TableStats { return &TableStats{} }

func (t *TableStats) Name() string { return "tablestats" }

func (t *TableStats) Collect(ctx context.Context, db *sql.DB, instance string) error {
	const query = `SELECT t.TABLE_SCHEMA, t.TABLE_NAME, t.TABLE_ROWS,
		t.DATA_LENGTH, t.INDEX_LENGTH, t.DATA_FREE,
		t.AUTO_INCREMENT, COALESCE(
			CASE c.DATA_TYPE
				WHEN 'tinyint' THEN 255
				WHEN 'smallint' THEN 65535
				WHEN 'mediumint' THEN 16777215
				WHEN 'int' THEN 4294967295
				WHEN 'bigint' THEN 18446744073709551615
			END, 0) AS MAX_AI
		FROM information_schema.TABLES t
		LEFT JOIN information_schema.COLUMNS c
			ON t.TABLE_SCHEMA = c.TABLE_SCHEMA
			AND t.TABLE_NAME = c.TABLE_NAME
			AND c.EXTRA LIKE '%%auto_increment%%'
		WHERE t.TABLE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
			AND t.TABLE_TYPE = 'BASE TABLE'`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("table stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Reset to clear stale tables.
	metrics.TableRows.Reset()
	metrics.TableDataBytes.Reset()
	metrics.TableIndexBytes.Reset()
	metrics.TableFreeBytes.Reset()
	metrics.TableAutoIncHeadroom.Reset()

	for rows.Next() {
		var schema, table string
		var tableRows, dataLen, indexLen, dataFree sql.NullInt64
		var autoInc, maxAI sql.NullFloat64

		if err := rows.Scan(&schema, &table, &tableRows, &dataLen, &indexLen, &dataFree, &autoInc, &maxAI); err != nil {
			continue
		}

		labels := []string{instance, schema, table}

		if tableRows.Valid {
			metrics.TableRows.WithLabelValues(labels...).Set(float64(tableRows.Int64))
		}
		if dataLen.Valid {
			metrics.TableDataBytes.WithLabelValues(labels...).Set(float64(dataLen.Int64))
		}
		if indexLen.Valid {
			metrics.TableIndexBytes.WithLabelValues(labels...).Set(float64(indexLen.Int64))
		}
		if dataFree.Valid {
			metrics.TableFreeBytes.WithLabelValues(labels...).Set(float64(dataFree.Int64))
		}
		if autoInc.Valid && maxAI.Valid && maxAI.Float64 > 0 {
			headroom := maxAI.Float64 - autoInc.Float64
			metrics.TableAutoIncHeadroom.WithLabelValues(labels...).Set(math.Max(headroom, 0))
		}
	}
	return rows.Err()
}
