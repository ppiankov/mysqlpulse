package cli

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"

	"github.com/ppiankov/mysqlpulse/internal/config"
	"github.com/ppiankov/mysqlpulse/internal/output"
)

// DiffResult holds the comparison between nodes.
type DiffResult struct {
	Nodes       []string       `json:"nodes"`
	Differences []DiffVariable `json:"differences"`
	TotalVars   int            `json:"total_variables"`
}

// DiffVariable is a single variable that differs between nodes.
type DiffVariable struct {
	Name   string            `json:"name"`
	Values map[string]string `json:"values"`
}

func newDiffCmd() *cobra.Command {
	var showAll bool

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare MySQL configuration across nodes",
		Long:  "Compares SHOW GLOBAL VARIABLES between configured nodes. Shows only differences by default.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			if len(cfg.DSNs) < 2 {
				return fmt.Errorf("diff requires at least 2 nodes (MYSQL_DSN with comma-separated DSNs)")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			// Collect variables from each node.
			nodeVars := make(map[string]map[string]string)
			var nodeOrder []string

			for _, dsn := range cfg.DSNs {
				instance := instanceLabel(dsn)
				nodeOrder = append(nodeOrder, instance)

				db, err := sql.Open("mysql", dsn)
				if err != nil {
					nodeVars[instance] = map[string]string{"_error": err.Error()}
					continue
				}

				vars, err := fetchGlobalVars(ctx, db)
				_ = db.Close()
				if err != nil {
					nodeVars[instance] = map[string]string{"_error": err.Error()}
					continue
				}
				nodeVars[instance] = vars
			}

			// Build unified variable set.
			allVarNames := make(map[string]bool)
			for _, vars := range nodeVars {
				for k := range vars {
					allVarNames[k] = true
				}
			}

			sortedNames := make([]string, 0, len(allVarNames))
			for k := range allVarNames {
				sortedNames = append(sortedNames, k)
			}
			sort.Strings(sortedNames)

			result := DiffResult{
				Nodes:     nodeOrder,
				TotalVars: len(sortedNames),
			}

			for _, name := range sortedNames {
				values := make(map[string]string)
				for _, node := range nodeOrder {
					if vars, ok := nodeVars[node]; ok {
						values[node] = vars[name]
					}
				}

				differs := false
				var first string
				for i, node := range nodeOrder {
					if i == 0 {
						first = values[node]
					} else if values[node] != first {
						differs = true
						break
					}
				}

				if showAll || differs {
					result.Differences = append(result.Differences, DiffVariable{
						Name:   name,
						Values: values,
					})
				}
			}

			prov := map[string]output.Provenance{
				"differences": output.Observed,
				"nodes":       output.Declared,
			}

			table := diffTable(result, nodeOrder)
			return output.Render(formatFlag, output.Result{Data: result, Provenance: prov}, table)
		},
	}

	cmd.Flags().BoolVar(&showAll, "all", false, "show all variables, not just differences")

	return cmd
}

func fetchGlobalVars(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, "SHOW GLOBAL VARIABLES")
	if err != nil {
		return nil, fmt.Errorf("SHOW GLOBAL VARIABLES: %w", err)
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

func diffTable(r DiffResult, nodes []string) *output.Table {
	headers := []string{"VARIABLE"}
	headers = append(headers, nodes...)
	headers = append(headers, "DIFFERS")

	t := &output.Table{Headers: headers}

	for _, d := range r.Differences {
		row := []string{d.Name}
		differs := false
		var first string
		for i, node := range nodes {
			val := d.Values[node]
			row = append(row, val)
			if i == 0 {
				first = val
			} else if val != first {
				differs = true
			}
		}
		if differs {
			row = append(row, "***")
		} else {
			row = append(row, "")
		}
		t.Rows = append(t.Rows, row)
	}

	t.Rows = append(t.Rows, append([]string{
		fmt.Sprintf("%d differences / %d total", len(r.Differences), r.TotalVars),
	}, make([]string, len(nodes)+1)...))

	return t
}
