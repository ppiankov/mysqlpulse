package cli

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"

	"github.com/ppiankov/mysqlpulse/internal/config"
	"github.com/ppiankov/mysqlpulse/internal/output"
)

// StatusData is the per-node health summary.
type StatusData struct {
	Instance    string `json:"instance"`
	Version     string `json:"version"`
	Uptime      string `json:"uptime"`
	ReadOnly    string `json:"read_only"`
	Connections string `json:"connections"`
	Running     string `json:"threads_running"`
	SlowQueries string `json:"slow_queries"`
	ReplState   string `json:"repl_state"`
	ReplLag     string `json:"repl_lag"`
	BufferPool  string `json:"buffer_pool_hit_ratio"`
	Error       string `json:"error,omitempty"`
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "One-shot health summary per node",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var statuses []StatusData

			for _, dsn := range cfg.DSNs {
				instance := instanceLabel(dsn)
				db, err := sql.Open("mysql", dsn)
				if err != nil {
					statuses = append(statuses, StatusData{Instance: instance, Error: err.Error()})
					continue
				}

				s := collectStatus(ctx, db, instance)
				_ = db.Close()
				statuses = append(statuses, s)
			}

			prov := map[string]output.Provenance{
				"statuses": output.Observed,
			}

			table := statusTable(statuses)
			return output.Render(formatFlag, output.Result{Data: statuses, Provenance: prov}, table)
		},
	}
}

func collectStatus(ctx context.Context, db *sql.DB, instance string) StatusData {
	s := StatusData{Instance: instance}

	// Version.
	var k, v string
	row := db.QueryRowContext(ctx, "SHOW GLOBAL VARIABLES LIKE 'version'")
	if err := row.Scan(&k, &v); err == nil {
		s.Version = v
	}

	// Uptime.
	s.Uptime = queryStatusVal(ctx, db, "Uptime")

	// Read-only.
	row = db.QueryRowContext(ctx, "SHOW GLOBAL VARIABLES LIKE 'read_only'")
	if err := row.Scan(&k, &v); err == nil {
		s.ReadOnly = v
	}

	// Connections/running.
	s.Connections = queryStatusVal(ctx, db, "Threads_connected")
	s.Running = queryStatusVal(ctx, db, "Threads_running")
	s.SlowQueries = queryStatusVal(ctx, db, "Slow_queries")

	// Buffer pool hit ratio.
	reads := queryStatusVal(ctx, db, "Innodb_buffer_pool_reads")
	requests := queryStatusVal(ctx, db, "Innodb_buffer_pool_read_requests")
	var readsF, requestsF float64
	if _, err := fmt.Sscanf(reads, "%f", &readsF); err == nil {
		if _, err := fmt.Sscanf(requests, "%f", &requestsF); err == nil && requestsF > 0 {
			s.BufferPool = fmt.Sprintf("%.4f", 1-readsF/requestsF)
		}
	}

	// Replication.
	s.ReplState = "not replicating"
	s.ReplLag = "-"
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

		io := firstNonEmpty(m, "Replica_IO_Running", "Slave_IO_Running")
		sql := firstNonEmpty(m, "Replica_SQL_Running", "Slave_SQL_Running")
		if io == "Yes" && sql == "Yes" {
			s.ReplState = "running"
		} else {
			s.ReplState = fmt.Sprintf("IO=%s SQL=%s", io, sql)
		}
		s.ReplLag = firstNonEmpty(m, "Seconds_Behind_Source", "Seconds_Behind_Master")
		if s.ReplLag == "" {
			s.ReplLag = "NULL"
		}
		break
	}

	return s
}

func firstNonEmpty(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != "" {
			return v
		}
	}
	return ""
}

func statusTable(statuses []StatusData) *output.Table {
	t := &output.Table{
		Headers: []string{"INSTANCE", "VERSION", "UPTIME", "RO", "CONN", "RUN", "SLOW", "REPL", "LAG", "BP HIT"},
	}
	for _, s := range statuses {
		if s.Error != "" {
			t.Rows = append(t.Rows, []string{s.Instance, "ERROR", "", "", "", "", "", "", "", s.Error})
			continue
		}
		t.Rows = append(t.Rows, []string{
			s.Instance, s.Version, s.Uptime, s.ReadOnly,
			s.Connections, s.Running, s.SlowQueries,
			s.ReplState, s.ReplLag, s.BufferPool,
		})
	}
	return t
}
