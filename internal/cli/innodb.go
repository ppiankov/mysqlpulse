package cli

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"

	"github.com/ppiankov/mysqlpulse/internal/config"
	"github.com/ppiankov/mysqlpulse/internal/innodb"
	"github.com/ppiankov/mysqlpulse/internal/output"
)

// InnoDBReport is the per-node InnoDB status output.
type InnoDBReport struct {
	Instance string        `json:"instance"`
	Status   innodb.Status `json:"status"`
	Error    string        `json:"error,omitempty"`
}

func newInnoDBCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "innodb",
		Short: "Parse and display structured InnoDB engine status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			var reports []InnoDBReport

			for _, dsn := range cfg.DSNs {
				instance := instanceLabel(dsn)
				db, err := sql.Open("mysql", dsn)
				if err != nil {
					reports = append(reports, InnoDBReport{
						Instance: instance,
						Error:    err.Error(),
					})
					continue
				}

				raw, err := queryInnoDBStatus(ctx, db)
				_ = db.Close()
				if err != nil {
					reports = append(reports, InnoDBReport{
						Instance: instance,
						Error:    err.Error(),
					})
					continue
				}

				reports = append(reports, InnoDBReport{
					Instance: instance,
					Status:   innodb.Parse(raw),
				})
			}

			prov := map[string]output.Provenance{
				"status": output.Observed,
			}

			table := innodbTable(reports)
			return output.Render(formatFlag, output.Result{Data: reports, Provenance: prov}, table)
		},
	}
}

func queryInnoDBStatus(ctx context.Context, db *sql.DB) (string, error) {
	var typ, name, status string
	row := db.QueryRowContext(ctx, "SHOW ENGINE INNODB STATUS")
	if err := row.Scan(&typ, &name, &status); err != nil {
		return "", fmt.Errorf("SHOW ENGINE INNODB STATUS: %w", err)
	}
	return status, nil
}

func innodbTable(reports []InnoDBReport) *output.Table {
	t := &output.Table{
		Headers: []string{"INSTANCE", "SECTION", "METRIC", "VALUE"},
	}

	for _, r := range reports {
		if r.Error != "" {
			t.Rows = append(t.Rows, []string{r.Instance, "error", "", r.Error})
			continue
		}

		s := r.Status
		inst := r.Instance

		// Buffer pool.
		t.Rows = append(t.Rows,
			[]string{inst, "buffer_pool", "total_pages", fmt.Sprintf("%d", s.BufferPool.TotalPages)},
			[]string{inst, "buffer_pool", "free_pages", fmt.Sprintf("%d", s.BufferPool.FreePages)},
			[]string{inst, "buffer_pool", "dirty_pages", fmt.Sprintf("%d", s.BufferPool.DirtyPages)},
			[]string{inst, "buffer_pool", "hit_rate", fmt.Sprintf("%.4f", s.BufferPool.HitRate)},
			[]string{inst, "buffer_pool", "pending_reads", fmt.Sprintf("%d", s.BufferPool.PendingReads)},
		)

		// Redo log.
		t.Rows = append(t.Rows,
			[]string{inst, "redo_log", "lsn", fmt.Sprintf("%d", s.RedoLog.LSN)},
			[]string{inst, "redo_log", "checkpoint_age", fmt.Sprintf("%d", s.RedoLog.CheckpointAge)},
		)

		// Transactions.
		t.Rows = append(t.Rows,
			[]string{inst, "transactions", "active_count", fmt.Sprintf("%d", s.Transactions.ActiveCount)},
			[]string{inst, "transactions", "history_list_length", fmt.Sprintf("%d", s.Transactions.HistoryListLength)},
		)

		// Row operations.
		t.Rows = append(t.Rows,
			[]string{inst, "row_ops", "reads/s", fmt.Sprintf("%.2f", s.RowOps.ReadsPerSec)},
			[]string{inst, "row_ops", "inserts/s", fmt.Sprintf("%.2f", s.RowOps.InsertsPerSec)},
			[]string{inst, "row_ops", "updates/s", fmt.Sprintf("%.2f", s.RowOps.UpdatesPerSec)},
			[]string{inst, "row_ops", "deletes/s", fmt.Sprintf("%.2f", s.RowOps.DeletesPerSec)},
		)

		// Deadlocks.
		if len(s.Deadlocks) > 0 {
			t.Rows = append(t.Rows,
				[]string{inst, "deadlocks", "count", fmt.Sprintf("%d", len(s.Deadlocks))},
				[]string{inst, "deadlocks", "victim", s.Deadlocks[0].Victim},
			)
		}
	}

	return t
}
