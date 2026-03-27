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

// TopologyNode represents a node in the replication topology.
type TopologyNode struct {
	Instance   string   `json:"instance"`
	Role       string   `json:"role"`
	SourceHost string   `json:"source_host,omitempty"`
	ReplLag    *float64 `json:"repl_lag_seconds,omitempty"`
	IORunning  string   `json:"io_running,omitempty"`
	SQLRunning string   `json:"sql_running,omitempty"`
	GTIDExec   string   `json:"gtid_executed,omitempty"`
	Replicas   []string `json:"replicas,omitempty"`
	Error      string   `json:"error,omitempty"`
}

func newTopologyCmd() *cobra.Command {
	var dotFormat bool

	cmd := &cobra.Command{
		Use:   "topology",
		Short: "Discover and display MySQL replication topology",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			nodes := discoverTopology(ctx, cfg.DSNs)

			// If --format dot, override to DOT output.
			if dotFormat {
				fmt.Print(toDOT(nodes))
				return nil
			}

			prov := map[string]output.Provenance{
				"nodes": output.Observed,
			}

			table := topologyTable(nodes)
			return output.Render(formatFlag, output.Result{Data: nodes, Provenance: prov}, table)
		},
	}

	cmd.Flags().BoolVar(&dotFormat, "dot", false, "output Graphviz DOT format")

	return cmd
}

func discoverTopology(ctx context.Context, dsns []string) []TopologyNode {
	nodes := make([]TopologyNode, 0, len(dsns))
	replicaOf := make(map[string][]string) // source → []replica instances

	for _, dsn := range dsns {
		instance := instanceLabel(dsn)
		node := TopologyNode{Instance: instance, Role: "source"}

		db, err := sql.Open("mysql", dsn)
		if err != nil {
			node.Error = err.Error()
			node.Role = "unknown"
			nodes = append(nodes, node)
			continue
		}

		// Check GTID.
		var gName, gVal string
		row := db.QueryRowContext(ctx, "SHOW GLOBAL VARIABLES LIKE 'gtid_executed'")
		if err := row.Scan(&gName, &gVal); err == nil && gVal != "" {
			node.GTIDExec = gVal
		}

		// Check replication status.
		replInfo := queryReplInfo(ctx, db)
		_ = db.Close()

		if replInfo != nil {
			node.Role = "replica"
			node.SourceHost = replInfo.sourceHost
			node.IORunning = replInfo.ioRunning
			node.SQLRunning = replInfo.sqlRunning
			if replInfo.lag >= 0 {
				lag := replInfo.lag
				node.ReplLag = &lag
			}
			replicaOf[replInfo.sourceHost] = append(replicaOf[replInfo.sourceHost], instance)
		}

		nodes = append(nodes, node)
	}

	// Enrich source nodes with their replica list.
	for i := range nodes {
		if nodes[i].Role == "source" {
			// Match by host part of instance label.
			host := hostPart(nodes[i].Instance)
			if replicas, ok := replicaOf[host]; ok {
				nodes[i].Replicas = replicas
			}
			// Also check full instance label.
			if replicas, ok := replicaOf[nodes[i].Instance]; ok && len(nodes[i].Replicas) == 0 {
				nodes[i].Replicas = replicas
			}
		}
		// Detect intermediate (replica that is also a source).
		if nodes[i].Role == "replica" {
			host := hostPart(nodes[i].Instance)
			if _, ok := replicaOf[host]; ok {
				nodes[i].Role = "intermediate"
			}
		}
	}

	return nodes
}

type replInfo struct {
	sourceHost string
	ioRunning  string
	sqlRunning string
	lag        float64
}

func queryReplInfo(ctx context.Context, db *sql.DB) *replInfo {
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

		info := &replInfo{lag: -1}

		// Source host.
		for _, key := range []string{"Source_Host", "Master_Host"} {
			if v, ok := m[key]; ok && v != "" {
				info.sourceHost = v
				break
			}
		}
		if info.sourceHost == "" {
			return nil
		}

		// IO/SQL thread state.
		for _, key := range []string{"Replica_IO_Running", "Slave_IO_Running"} {
			if v, ok := m[key]; ok {
				info.ioRunning = v
				break
			}
		}
		for _, key := range []string{"Replica_SQL_Running", "Slave_SQL_Running"} {
			if v, ok := m[key]; ok {
				info.sqlRunning = v
				break
			}
		}

		// Lag.
		for _, key := range []string{"Seconds_Behind_Source", "Seconds_Behind_Master"} {
			if v, ok := m[key]; ok && v != "" {
				var f float64
				if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
					info.lag = f
				}
				break
			}
		}

		return info
	}
	return nil
}

func hostPart(instance string) string {
	if i := strings.Index(instance, ":"); i >= 0 {
		return instance[:i]
	}
	return instance
}

func toDOT(nodes []TopologyNode) string {
	var b strings.Builder
	b.WriteString("digraph topology {\n")
	b.WriteString("  rankdir=TB;\n")
	b.WriteString("  node [shape=box];\n")

	for _, n := range nodes {
		label := fmt.Sprintf("%s\\n[%s]", n.Instance, n.Role)
		b.WriteString(fmt.Sprintf("  %q [label=%q];\n", n.Instance, label))

		if n.SourceHost != "" {
			lagLabel := ""
			if n.ReplLag != nil {
				lagLabel = fmt.Sprintf("lag=%.0fs", *n.ReplLag)
			}
			b.WriteString(fmt.Sprintf("  %q -> %q [label=%q];\n", n.SourceHost, n.Instance, lagLabel))
		}
	}

	b.WriteString("}\n")
	return b.String()
}

func topologyTable(nodes []TopologyNode) *output.Table {
	t := &output.Table{
		Headers: []string{"INSTANCE", "ROLE", "SOURCE", "LAG", "IO", "SQL"},
	}

	for _, n := range nodes {
		lag := "-"
		if n.ReplLag != nil {
			lag = fmt.Sprintf("%.0fs", *n.ReplLag)
		}
		io := n.IORunning
		if io == "" {
			io = "-"
		}
		sqlR := n.SQLRunning
		if sqlR == "" {
			sqlR = "-"
		}
		source := n.SourceHost
		if source == "" {
			source = "-"
		}

		t.Rows = append(t.Rows, []string{n.Instance, n.Role, source, lag, io, sqlR})
	}

	return t
}
