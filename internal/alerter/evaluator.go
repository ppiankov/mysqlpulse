package alerter

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// Thresholds for built-in alerts.
const (
	ReplLagThreshold  = 30.0 // seconds
	BufferPoolFreeMin = 0.10 // 10% free pages
	ConnExhaustionMax = 0.80 // 80% of max_connections
	HistoryListMax    = 1000.0
)

// Evaluate checks alert conditions against a MySQL instance and fires alerts/annotations.
func Evaluate(ctx context.Context, db *sql.DB, instance string, a *Alerter, ann *Annotator) {
	if a == nil && ann == nil {
		return
	}

	host := HostFromDSN(instance)
	masked := MaskDSN(instance)

	fire := func(alert Alert) {
		a.Send(alert)
		ann.Annotate(alert.Type, alert.Host, alert.Message, nil)
	}

	// Replication checks.
	evaluateReplication(ctx, db, masked, host, fire)

	// Status-based checks.
	status, err := globalStatus(ctx, db)
	if err != nil {
		return
	}

	vars, _ := globalVars(ctx, db)

	// Connection exhaustion.
	if maxStr, ok := vars["max_connections"]; ok {
		if maxConns, err := strconv.ParseFloat(maxStr, 64); err == nil && maxConns > 0 {
			if current, ok := status["Threads_connected"]; ok {
				ratio := current / maxConns
				if ratio > ConnExhaustionMax {
					fire(Alert{
						Type:     AlertConnExhaustion,
						Message:  fmt.Sprintf("Connection usage at %.0f%% (%.0f/%.0f)", ratio*100, current, maxConns),
						Instance: masked,
						Host:     host,
					})
				}
			}
		}
	}

	// Buffer pool pressure.
	if total, ok := status["Innodb_buffer_pool_pages_total"]; ok && total > 0 {
		if free, ok := status["Innodb_buffer_pool_pages_free"]; ok {
			freeRatio := free / total
			if freeRatio < BufferPoolFreeMin {
				fire(Alert{
					Type:     AlertBufferPool,
					Message:  fmt.Sprintf("Buffer pool %.1f%% free (%.0f/%.0f pages)", freeRatio*100, free, total),
					Instance: masked,
					Host:     host,
				})
			}
		}
	}

	// Deadlocks (rate > 0 since last check — simplified: any non-zero value triggers).
	if deadlocks, ok := status["Innodb_deadlocks"]; ok && deadlocks > 0 {
		// Only alert once via cooldown — the cumulative counter will always be > 0.
		// This is handled by the alerter's dedup/cooldown.
		fire(Alert{
			Type:     AlertDeadlocks,
			Message:  fmt.Sprintf("Deadlocks detected (cumulative: %.0f)", deadlocks),
			Instance: masked,
			Host:     host,
		})
	}

	// History list length.
	if hll, ok := status["Innodb_history_list_length"]; ok && hll > HistoryListMax {
		fire(Alert{
			Type:     AlertHistoryList,
			Message:  fmt.Sprintf("History list length: %.0f (threshold: %.0f)", hll, HistoryListMax),
			Instance: masked,
			Host:     host,
		})
	}
}

func evaluateReplication(ctx context.Context, db *sql.DB, masked, host string, fire func(Alert)) {
	for _, query := range []string{"SHOW REPLICA STATUS", "SHOW SLAVE STATUS"} {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			continue
		}
		cols, err := rows.Columns()
		if err != nil {
			_ = rows.Close()
			continue
		}
		if !rows.Next() {
			_ = rows.Close()
			continue
		}
		vals := make([]interface{}, len(cols))
		for i := range vals {
			vals[i] = new(sql.RawBytes)
		}
		if err := rows.Scan(vals...); err != nil {
			_ = rows.Close()
			continue
		}
		_ = rows.Close()

		m := make(map[string]string)
		for i, col := range cols {
			b := vals[i].(*sql.RawBytes)
			m[col] = string(*b)
		}

		// IO/SQL thread stopped.
		ioRunning := firstVal(m, "Replica_IO_Running", "Slave_IO_Running")
		sqlRunning := firstVal(m, "Replica_SQL_Running", "Slave_SQL_Running")

		if !strings.EqualFold(ioRunning, "Yes") || !strings.EqualFold(sqlRunning, "Yes") {
			fire(Alert{
				Type:     AlertReplStopped,
				Message:  fmt.Sprintf("Replication stopped (IO=%s, SQL=%s)", ioRunning, sqlRunning),
				Instance: masked,
				Host:     host,
			})
		}

		// Replication lag.
		for _, key := range []string{"Seconds_Behind_Source", "Seconds_Behind_Master"} {
			if v, ok := m[key]; ok && v != "" {
				var lag float64
				if _, err := fmt.Sscanf(v, "%f", &lag); err == nil && lag > ReplLagThreshold {
					fire(Alert{
						Type:     AlertReplLag,
						Message:  fmt.Sprintf("Replication lag: %.0fs (threshold: %.0fs)", lag, ReplLagThreshold),
						Instance: masked,
						Host:     host,
					})
				}
				break
			}
		}

		return
	}
}

func firstVal(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != "" {
			return v
		}
	}
	return ""
}

func globalStatus(ctx context.Context, db *sql.DB) (map[string]float64, error) {
	rows, err := db.QueryContext(ctx, "SHOW GLOBAL STATUS")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]float64)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			result[name] = v
		}
	}
	return result, rows.Err()
}

func globalVars(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, "SHOW GLOBAL VARIABLES")
	if err != nil {
		return nil, err
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
