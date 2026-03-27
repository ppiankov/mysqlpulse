package collector

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// Processlist collects process metrics from SHOW PROCESSLIST.
type Processlist struct{}

func NewProcesslist() *Processlist { return &Processlist{} }

func (p *Processlist) Name() string { return "processlist" }

func (p *Processlist) Collect(ctx context.Context, db *sql.DB, instance string) error {
	rows, err := db.QueryContext(ctx, "SHOW PROCESSLIST")
	if err != nil {
		return fmt.Errorf("SHOW PROCESSLIST: %w", err)
	}
	defer func() { _ = rows.Close() }()

	stateCounts := make(map[string]float64)
	commandCounts := make(map[string]float64)
	userCounts := make(map[string]float64)
	var longest float64
	var locked float64

	for rows.Next() {
		var id int64
		var user, host, dbName, command, state, info sql.NullString
		var timeVal sql.NullInt64

		if err := rows.Scan(&id, &user, &host, &dbName, &command, &timeVal, &state, &info); err != nil {
			continue
		}

		cmdStr := nullStr(command)
		stateStr := nullStr(state)
		userStr := nullStr(user)
		seconds := float64(timeVal.Int64)

		if stateStr == "" {
			stateStr = "none"
		}
		stateCounts[stateStr]++
		commandCounts[cmdStr]++
		if userStr != "" {
			userCounts[userStr]++
		}

		if seconds > longest {
			longest = seconds
		}

		if strings.Contains(strings.ToLower(stateStr), "locked") {
			locked++
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Reset vector metrics to avoid stale labels from previous scrapes.
	metrics.ProcesslistByState.Reset()
	metrics.ProcesslistByCommand.Reset()
	metrics.ProcesslistByUser.Reset()

	for state, count := range stateCounts {
		metrics.ProcesslistByState.WithLabelValues(instance, state).Set(count)
	}
	for cmd, count := range commandCounts {
		metrics.ProcesslistByCommand.WithLabelValues(instance, cmd).Set(count)
	}
	for user, count := range userCounts {
		metrics.ProcesslistByUser.WithLabelValues(instance, user).Set(count)
	}

	metrics.ProcesslistLongest.WithLabelValues(instance).Set(longest)
	metrics.ProcesslistLocked.WithLabelValues(instance).Set(locked)

	return nil
}

func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
