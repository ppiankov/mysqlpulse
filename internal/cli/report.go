package cli

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"

	"github.com/ppiankov/mysqlpulse/internal/config"
	"github.com/ppiankov/mysqlpulse/internal/output"
)

// ReportData is the top-level report structure.
type ReportData struct {
	Timestamp string       `json:"timestamp"`
	Nodes     []NodeReport `json:"nodes"`
}

// NodeReport is the per-node diagnostic snapshot.
type NodeReport struct {
	Instance    string            `json:"instance"`
	ServerInfo  map[string]string `json:"server_info"`
	Connections map[string]string `json:"connections"`
	Replication map[string]string `json:"replication,omitempty"`
	InnoDB      map[string]string `json:"innodb"`
	Queries     map[string]string `json:"queries"`
	Processlist []ProcessInfo     `json:"processlist"`
}

// ProcessInfo is a simplified process entry for the report.
type ProcessInfo struct {
	ID      string `json:"id"`
	User    string `json:"user"`
	Host    string `json:"host"`
	Command string `json:"command"`
	Time    string `json:"time"`
	State   string `json:"state"`
}

func newReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "One-shot diagnostic dump of all MySQL metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			report := ReportData{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}

			for _, dsn := range cfg.DSNs {
				instance := instanceLabel(dsn)
				db, err := sql.Open("mysql", dsn)
				if err != nil {
					report.Nodes = append(report.Nodes, NodeReport{
						Instance:   instance,
						ServerInfo: map[string]string{"error": err.Error()},
					})
					continue
				}

				nr := collectNodeReport(ctx, db, instance)
				_ = db.Close()
				report.Nodes = append(report.Nodes, nr)
			}

			prov := map[string]output.Provenance{
				"nodes":       output.Observed,
				"timestamp":   output.Declared,
				"server_info": output.Observed,
				"connections": output.Observed,
				"replication": output.Observed,
				"innodb":      output.Observed,
				"queries":     output.Observed,
				"processlist": output.Observed,
			}

			table := reportTable(report)
			return output.Render(formatFlag, output.Result{Data: report, Provenance: prov}, table)
		},
	}
}

func collectNodeReport(ctx context.Context, db *sql.DB, instance string) NodeReport {
	nr := NodeReport{Instance: instance}

	nr.ServerInfo = queryVars(ctx, db, "version", "version_comment", "hostname", "port")
	nr.ServerInfo["uptime"] = queryStatusVal(ctx, db, "Uptime")

	nr.Connections = queryStatusVals(ctx, db,
		"Threads_connected", "Threads_running", "Threads_cached",
		"Max_used_connections", "Connections", "Aborted_connects", "Aborted_clients")

	nr.Replication = queryReplicationSummary(ctx, db)
	if len(nr.Replication) == 0 {
		nr.Replication = nil
	}

	nr.InnoDB = queryStatusVals(ctx, db,
		"Innodb_buffer_pool_pages_total", "Innodb_buffer_pool_pages_free",
		"Innodb_buffer_pool_pages_dirty", "Innodb_buffer_pool_reads",
		"Innodb_buffer_pool_read_requests", "Innodb_row_lock_waits",
		"Innodb_deadlocks", "Innodb_history_list_length")

	nr.Queries = queryStatusVals(ctx, db,
		"Queries", "Questions", "Slow_queries",
		"Com_select", "Com_insert", "Com_update", "Com_delete",
		"Select_full_join", "Sort_merge_passes")

	nr.Processlist = queryProcesslist(ctx, db)

	return nr
}

func queryVars(ctx context.Context, db *sql.DB, names ...string) map[string]string {
	result := make(map[string]string)
	for _, name := range names {
		var k, v string
		row := db.QueryRowContext(ctx,
			fmt.Sprintf("SHOW GLOBAL VARIABLES LIKE '%s'", name))
		if err := row.Scan(&k, &v); err == nil {
			result[name] = v
		}
	}
	return result
}

func queryStatusVal(ctx context.Context, db *sql.DB, name string) string {
	var k, v string
	row := db.QueryRowContext(ctx,
		fmt.Sprintf("SHOW GLOBAL STATUS LIKE '%s'", name))
	if err := row.Scan(&k, &v); err == nil {
		return v
	}
	return ""
}

func queryStatusVals(ctx context.Context, db *sql.DB, names ...string) map[string]string {
	result := make(map[string]string)
	for _, name := range names {
		if v := queryStatusVal(ctx, db, name); v != "" {
			result[name] = v
		}
	}
	return result
}

func queryReplicationSummary(ctx context.Context, db *sql.DB) map[string]string {
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
		want := map[string]bool{
			"Replica_IO_Running": true, "Slave_IO_Running": true,
			"Replica_SQL_Running": true, "Slave_SQL_Running": true,
			"Seconds_Behind_Source": true, "Seconds_Behind_Master": true,
			"Last_Error": true, "Last_SQL_Error": true,
			"Source_Host": true, "Master_Host": true,
		}
		for i, col := range cols {
			if want[col] {
				b := vals[i].(*sql.RawBytes)
				if len(*b) > 0 {
					m[col] = string(*b)
				}
			}
		}
		return m
	}
	return nil
}

func queryProcesslist(ctx context.Context, db *sql.DB) []ProcessInfo {
	rows, err := db.QueryContext(ctx, "SHOW PROCESSLIST")
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	var procs []ProcessInfo
	for rows.Next() {
		var id int64
		var user, host, dbName, command, state, info sql.NullString
		var timeVal sql.NullInt64

		if err := rows.Scan(&id, &user, &host, &dbName, &command, &timeVal, &state, &info); err != nil {
			continue
		}

		procs = append(procs, ProcessInfo{
			ID:      fmt.Sprintf("%d", id),
			User:    nullStrCLI(user),
			Host:    nullStrCLI(host),
			Command: nullStrCLI(command),
			Time:    fmt.Sprintf("%d", timeVal.Int64),
			State:   nullStrCLI(state),
		})
	}
	return procs
}

func reportTable(r ReportData) *output.Table {
	t := &output.Table{
		Headers: []string{"SECTION", "KEY", "VALUE"},
	}

	for _, node := range r.Nodes {
		t.Rows = append(t.Rows, []string{"---", node.Instance, "---"})

		addSection(t, "server", node.ServerInfo)
		addSection(t, "connections", node.Connections)
		if node.Replication != nil {
			addSection(t, "replication", node.Replication)
		}
		addSection(t, "innodb", node.InnoDB)
		addSection(t, "queries", node.Queries)

		for _, p := range node.Processlist {
			t.Rows = append(t.Rows, []string{
				"process",
				fmt.Sprintf("%s@%s", p.User, p.Host),
				fmt.Sprintf("%s %ss %s", p.Command, p.Time, p.State),
			})
		}
	}

	return t
}

func addSection(t *output.Table, section string, m map[string]string) {
	// Sort keys for stable output.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		t.Rows = append(t.Rows, []string{section, k, m[k]})
	}
}

func nullStrCLI(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && strings.Compare(s[j-1], s[j]) > 0; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
