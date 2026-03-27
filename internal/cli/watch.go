package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"

	"github.com/ppiankov/mysqlpulse/internal/collector"
	"github.com/ppiankov/mysqlpulse/internal/config"
)

type watchMode int

const (
	modeOverview watchMode = iota
	modeProcesslist
	modeReplication
	modeInnoDB
	modeCount
)

func (m watchMode) String() string {
	switch m {
	case modeOverview:
		return "overview"
	case modeProcesslist:
		return "processlist"
	case modeReplication:
		return "replication"
	case modeInnoDB:
		return "innodb"
	}
	return "unknown"
}

func newWatchCmd() *cobra.Command {
	var interval time.Duration

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Live dashboard with terminal refresh (innotop replacement)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			var targets []watchTarget
			for _, dsn := range cfg.DSNs {
				db, err := sql.Open("mysql", dsn)
				if err != nil {
					fmt.Fprintf(os.Stderr, "open %s: %v\n", instanceLabel(dsn), err)
					continue
				}
				defer func() { _ = db.Close() }()
				targets = append(targets, watchTarget{instance: instanceLabel(dsn), db: db})
			}
			if len(targets) == 0 {
				return fmt.Errorf("no reachable targets")
			}

			mode := modeOverview

			restoreTerminal := enableRawTerminal()
			defer restoreTerminal()

			keys := make(chan byte, 10)
			go readKeys(keys)

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			renderWatchScreen(ctx, targets, mode)

			for {
				select {
				case <-ctx.Done():
					clearScreen()
					return nil
				case <-ticker.C:
					renderWatchScreen(ctx, targets, mode)
				case key := <-keys:
					switch key {
					case 'q', 'Q', 3:
						clearScreen()
						return nil
					case '\t', 'n', 'N':
						mode = (mode + 1) % modeCount
						renderWatchScreen(ctx, targets, mode)
					case '1':
						mode = modeOverview
						renderWatchScreen(ctx, targets, mode)
					case '2':
						mode = modeProcesslist
						renderWatchScreen(ctx, targets, mode)
					case '3':
						mode = modeReplication
						renderWatchScreen(ctx, targets, mode)
					case '4':
						mode = modeInnoDB
						renderWatchScreen(ctx, targets, mode)
					}
				}
			}
		},
	}

	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "refresh interval")
	return cmd
}

type watchTarget struct {
	instance string
	db       *sql.DB
}

func renderWatchScreen(ctx context.Context, targets []watchTarget, mode watchMode) {
	clearScreen()
	now := time.Now().Format("15:04:05")
	fmt.Printf("\033[1mmysqlpulse watch\033[0m | %s | mode: \033[1m%s\033[0m | [tab] cycle [1-4] select [q] quit\n\n", now, mode)

	for _, t := range targets {
		fmt.Printf("\033[1;36m%s\033[0m\n", t.instance)
		switch mode {
		case modeOverview:
			renderOverview(ctx, t.db)
		case modeProcesslist:
			renderProcesslistWatch(ctx, t.db)
		case modeReplication:
			renderReplicationWatch(ctx, t.db)
		case modeInnoDB:
			renderInnoDBWatch(ctx, t.db)
		}
		fmt.Println()
	}
}

func renderOverview(ctx context.Context, db *sql.DB) {
	status, err := collector.GlobalStatus(ctx, db)
	if err != nil {
		fmt.Printf("  error: %v\n", err)
		return
	}

	for _, p := range []struct {
		label, key string
	}{
		{"Connections", "Threads_connected"},
		{"Running", "Threads_running"},
		{"QPS", "Queries"},
		{"Slow queries", "Slow_queries"},
		{"InnoDB rows read", "Innodb_rows_read"},
		{"InnoDB deadlocks", "Innodb_deadlocks"},
		{"Aborted connects", "Aborted_connects"},
		{"Uptime", "Uptime"},
	} {
		if v, ok := status[p.key]; ok {
			fmt.Printf("  %-22s %12.0f\n", p.label, v)
		}
	}
}

func renderProcesslistWatch(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, "SHOW PROCESSLIST")
	if err != nil {
		fmt.Printf("  error: %v\n", err)
		return
	}
	defer func() { _ = rows.Close() }()

	fmt.Printf("  %-8s %-16s %-10s %6s  %-20s\n", "ID", "USER", "COMMAND", "TIME", "STATE")
	fmt.Printf("  %s\n", strings.Repeat("-", 70))

	for rows.Next() {
		var id int64
		var user, host, dbName, command, state, info sql.NullString
		var timeVal sql.NullInt64

		if err := rows.Scan(&id, &user, &host, &dbName, &command, &timeVal, &state, &info); err != nil {
			continue
		}

		stateStr := ""
		if state.Valid && len(state.String) > 20 {
			stateStr = state.String[:20]
		} else if state.Valid {
			stateStr = state.String
		}

		fmt.Printf("  %-8d %-16s %-10s %6d  %-20s\n",
			id,
			truncate(nullStrCLI(user), 16),
			truncate(nullStrCLI(command), 10),
			timeVal.Int64,
			stateStr,
		)
	}
}

func renderReplicationWatch(ctx context.Context, db *sql.DB) {
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

		for _, key := range []string{
			"Replica_IO_Running", "Slave_IO_Running",
			"Replica_SQL_Running", "Slave_SQL_Running",
			"Seconds_Behind_Source", "Seconds_Behind_Master",
			"Source_Host", "Master_Host",
			"Last_SQL_Error", "Last_Error",
			"Executed_Gtid_Set",
		} {
			if v, ok := m[key]; ok && v != "" {
				fmt.Printf("  %-30s %s\n", key, v)
			}
		}
		return
	}
	fmt.Println("  not a replica")
}

func renderInnoDBWatch(ctx context.Context, db *sql.DB) {
	status, err := collector.GlobalStatus(ctx, db)
	if err != nil {
		fmt.Printf("  error: %v\n", err)
		return
	}

	for _, p := range []struct {
		label, key string
	}{
		{"Buffer pool pages total", "Innodb_buffer_pool_pages_total"},
		{"Buffer pool pages free", "Innodb_buffer_pool_pages_free"},
		{"Buffer pool pages dirty", "Innodb_buffer_pool_pages_dirty"},
		{"Buffer pool reads", "Innodb_buffer_pool_reads"},
		{"Buffer pool read requests", "Innodb_buffer_pool_read_requests"},
		{"Row lock waits", "Innodb_row_lock_waits"},
		{"Deadlocks", "Innodb_deadlocks"},
		{"History list length", "Innodb_history_list_length"},
		{"Rows inserted", "Innodb_rows_inserted"},
		{"Rows updated", "Innodb_rows_updated"},
		{"Rows deleted", "Innodb_rows_deleted"},
		{"Rows read", "Innodb_rows_read"},
	} {
		if v, ok := status[p.key]; ok {
			fmt.Printf("  %-30s %12.0f\n", p.label, v)
		}
	}
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func enableRawTerminal() func() {
	cmd := exec.Command("stty", "-echo", "cbreak")
	cmd.Stdin = os.Stdin
	_ = cmd.Run()
	return func() {
		cmd := exec.Command("stty", "echo", "-cbreak")
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
	}
}

func readKeys(ch chan<- byte) {
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return
		}
		ch <- buf[0]
	}
}
