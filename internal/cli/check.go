package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"

	"github.com/ppiankov/mysqlpulse/internal/config"
	"github.com/ppiankov/mysqlpulse/internal/output"
)

// Nagios-compatible exit codes.
const (
	exitOK       = 0
	exitWarning  = 1
	exitCritical = 2
	exitUnknown  = 3
)

// CheckResult is the structured output of a check command.
type CheckResult struct {
	Metric  string       `json:"metric"`
	Nodes   []NodeResult `json:"nodes"`
	Verdict string       `json:"verdict"`
	Exit    int          `json:"exit_code"`
}

// NodeResult is the per-node check result.
type NodeResult struct {
	Instance string  `json:"instance"`
	Value    float64 `json:"value"`
	Warn     float64 `json:"warn_threshold"`
	Crit     float64 `json:"crit_threshold"`
	Status   string  `json:"status"`
}

// checkMetric defines how to query a specific metric from MySQL.
type checkMetric struct {
	query func(ctx context.Context, db *sql.DB) (float64, error)
	unit  string
}

var checkMetrics = map[string]checkMetric{
	"repl-lag": {
		query: queryReplLag,
		unit:  "seconds",
	},
	"connections": {
		query: queryGlobalStatusFloat("Threads_connected"),
		unit:  "connections",
	},
	"threads-running": {
		query: queryGlobalStatusFloat("Threads_running"),
		unit:  "threads",
	},
	"buffer-pool": {
		query: queryBufferPoolHitRatio,
		unit:  "ratio",
	},
	"slow-queries": {
		query: queryGlobalStatusFloat("Slow_queries"),
		unit:  "queries",
	},
	"deadlocks": {
		query: queryGlobalStatusFloat("Innodb_deadlocks"),
		unit:  "deadlocks",
	},
}

func newCheckCmd() *cobra.Command {
	var warn, crit float64

	cmd := &cobra.Command{
		Use:   "check <metric>",
		Short: "Check a metric against thresholds (Nagios-compatible exit codes)",
		Long: fmt.Sprintf("Metrics: %s\n\nExit codes: 0=OK, 1=WARNING, 2=CRITICAL, 3=UNKNOWN",
			strings.Join(availableMetrics(), ", ")),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			metricName := args[0]
			cm, ok := checkMetrics[metricName]
			if !ok {
				fmt.Fprintf(os.Stderr, "unknown metric: %s (available: %s)\n",
					metricName, strings.Join(availableMetrics(), ", "))
				os.Exit(exitUnknown)
				return nil
			}

			cfg, err := config.FromEnv()
			if err != nil {
				fmt.Fprintf(os.Stderr, "config: %v\n", err)
				os.Exit(exitUnknown)
				return nil
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result := CheckResult{
				Metric:  metricName,
				Verdict: "OK",
				Exit:    exitOK,
			}

			for _, dsn := range cfg.DSNs {
				instance := instanceLabel(dsn)
				nr := NodeResult{
					Instance: instance,
					Warn:     warn,
					Crit:     crit,
					Status:   "OK",
				}

				db, err := sql.Open("mysql", dsn)
				if err != nil {
					nr.Status = "UNKNOWN"
					result.Nodes = append(result.Nodes, nr)
					escalate(&result, exitUnknown, "UNKNOWN")
					continue
				}

				val, err := cm.query(ctx, db)
				_ = db.Close()
				if err != nil {
					nr.Status = "UNKNOWN"
					result.Nodes = append(result.Nodes, nr)
					escalate(&result, exitUnknown, "UNKNOWN")
					continue
				}

				nr.Value = val

				switch {
				case crit > 0 && val >= crit:
					nr.Status = "CRITICAL"
					escalate(&result, exitCritical, "CRITICAL")
				case warn > 0 && val >= warn:
					nr.Status = "WARNING"
					escalate(&result, exitWarning, "WARNING")
				}

				result.Nodes = append(result.Nodes, nr)
			}

			prov := map[string]output.Provenance{
				"nodes":   output.Observed,
				"verdict": output.Inferred,
			}

			table := checkTable(result, cm.unit)
			if err := output.Render(formatFlag, output.Result{Data: result, Provenance: prov}, table); err != nil {
				fmt.Fprintf(os.Stderr, "render: %v\n", err)
			}

			os.Exit(result.Exit)
			return nil
		},
	}

	cmd.Flags().Float64Var(&warn, "warn", 0, "warning threshold")
	cmd.Flags().Float64Var(&crit, "crit", 0, "critical threshold")

	return cmd
}

func escalate(r *CheckResult, code int, verdict string) {
	if code > r.Exit {
		r.Exit = code
		r.Verdict = verdict
	}
}

func availableMetrics() []string {
	keys := make([]string, 0, len(checkMetrics))
	for k := range checkMetrics {
		keys = append(keys, k)
	}
	return keys
}

func checkTable(r CheckResult, unit string) *output.Table {
	t := &output.Table{
		Headers: []string{"INSTANCE", "VALUE", "WARN", "CRIT", "STATUS"},
	}
	for _, n := range r.Nodes {
		t.Rows = append(t.Rows, []string{
			n.Instance,
			fmt.Sprintf("%.2f %s", n.Value, unit),
			fmt.Sprintf("%.2f", n.Warn),
			fmt.Sprintf("%.2f", n.Crit),
			n.Status,
		})
	}
	t.Rows = append(t.Rows, []string{"", "", "", "", r.Verdict})
	return t
}

// Query helpers.

func queryReplLag(ctx context.Context, db *sql.DB) (float64, error) {
	for _, query := range []string{
		"SHOW REPLICA STATUS",
		"SHOW SLAVE STATUS",
	} {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			continue
		}
		defer func() { _ = rows.Close() }()

		cols, err := rows.Columns()
		if err != nil {
			continue
		}
		if !rows.Next() {
			continue
		}

		vals := make([]interface{}, len(cols))
		for i := range vals {
			vals[i] = new(sql.RawBytes)
		}
		if err := rows.Scan(vals...); err != nil {
			continue
		}

		for i, col := range cols {
			if col == "Seconds_Behind_Source" || col == "Seconds_Behind_Master" {
				b := vals[i].(*sql.RawBytes)
				var v float64
				if _, err := fmt.Sscanf(string(*b), "%f", &v); err == nil {
					return v, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("replication status unavailable")
}

func queryGlobalStatusFloat(variable string) func(context.Context, *sql.DB) (float64, error) {
	return func(ctx context.Context, db *sql.DB) (float64, error) {
		var name, value string
		row := db.QueryRowContext(ctx,
			fmt.Sprintf("SHOW GLOBAL STATUS LIKE '%s'", variable))
		if err := row.Scan(&name, &value); err != nil {
			return 0, err
		}
		var v float64
		_, err := fmt.Sscanf(value, "%f", &v)
		return v, err
	}
}

func queryBufferPoolHitRatio(ctx context.Context, db *sql.DB) (float64, error) {
	var reads, requests float64
	for _, pair := range []struct {
		variable string
		target   *float64
	}{
		{"Innodb_buffer_pool_reads", &reads},
		{"Innodb_buffer_pool_read_requests", &requests},
	} {
		var name, value string
		row := db.QueryRowContext(ctx,
			fmt.Sprintf("SHOW GLOBAL STATUS LIKE '%s'", pair.variable))
		if err := row.Scan(&name, &value); err != nil {
			return 0, err
		}
		if _, err := fmt.Sscanf(value, "%f", pair.target); err != nil {
			return 0, err
		}
	}
	if requests == 0 {
		return 1, nil
	}
	return 1 - reads/requests, nil
}
