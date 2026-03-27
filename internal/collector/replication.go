package collector

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// Replication collects replication metrics from SHOW REPLICA STATUS.
// Falls back to SHOW SLAVE STATUS for MySQL < 8.0.22.
type Replication struct{}

func NewReplication() *Replication { return &Replication{} }

func (r *Replication) Name() string { return "replication" }

func (r *Replication) Collect(ctx context.Context, db *sql.DB, instance string) error {
	cols, vals, err := replicaStatus(ctx, db)
	if err != nil {
		// Not a replica — replication metrics not applicable.
		return nil
	}

	m := make(map[string]string, len(cols))
	for i, col := range cols {
		if b, ok := vals[i].(*sql.RawBytes); ok {
			m[col] = string(*b)
		}
	}

	// Seconds_Behind_Source (or Seconds_Behind_Master).
	if v, ok := lagSeconds(m); ok {
		metrics.ReplLagSeconds.WithLabelValues(instance).Set(v)
	}

	ioRunning := boolToFloat(m["Replica_IO_Running"], m["Slave_IO_Running"])
	sqlRunning := boolToFloat(m["Replica_SQL_Running"], m["Slave_SQL_Running"])

	metrics.ReplIORunning.WithLabelValues(instance).Set(ioRunning)
	metrics.ReplSQLRunning.WithLabelValues(instance).Set(sqlRunning)

	if ioRunning == 1 && sqlRunning == 1 {
		metrics.ReplRunning.WithLabelValues(instance).Set(1)
	} else {
		metrics.ReplRunning.WithLabelValues(instance).Set(0)
	}

	// Bytes behind: Read_Source_Log_Pos - Exec_Source_Log_Pos.
	readPos := floatFromMap(m, "Read_Source_Log_Pos", "Read_Master_Log_Pos")
	execPos := floatFromMap(m, "Exec_Source_Log_Pos", "Exec_Master_Log_Pos")
	if readPos >= 0 && execPos >= 0 {
		metrics.ReplBehindBytes.WithLabelValues(instance).Set(readPos - execPos)
	}

	return nil
}

func replicaStatus(ctx context.Context, db *sql.DB) ([]string, []interface{}, error) {
	// Try modern syntax first, fall back to legacy.
	for _, query := range []string{"SHOW REPLICA STATUS", "SHOW SLAVE STATUS"} {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			continue
		}
		defer func() { _ = rows.Close() }()

		cols, err := rows.Columns()
		if err != nil {
			return nil, nil, err
		}
		if !rows.Next() {
			return nil, nil, fmt.Errorf("no replication status")
		}

		vals := make([]interface{}, len(cols))
		for i := range vals {
			vals[i] = new(sql.RawBytes)
		}
		if err := rows.Scan(vals...); err != nil {
			return nil, nil, err
		}
		return cols, vals, nil
	}
	return nil, nil, fmt.Errorf("replication status unavailable")
}

func lagSeconds(m map[string]string) (float64, bool) {
	for _, key := range []string{"Seconds_Behind_Source", "Seconds_Behind_Master"} {
		if v, ok := m[key]; ok && v != "" {
			f, err := strconv.ParseFloat(v, 64)
			if err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

func boolToFloat(values ...string) float64 {
	for _, v := range values {
		if strings.EqualFold(v, "Yes") {
			return 1
		}
		if strings.EqualFold(v, "No") {
			return 0
		}
	}
	return 0
}

func floatFromMap(m map[string]string, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			f, err := strconv.ParseFloat(v, 64)
			if err == nil {
				return f
			}
		}
	}
	return -1
}
