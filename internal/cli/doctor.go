package cli

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ppiankov/mysqlpulse/internal/config"
	"github.com/ppiankov/mysqlpulse/internal/output"
	"github.com/spf13/cobra"
)

// DoctorCheck is a single health check result.
type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// DoctorResult is the ANCC-compliant doctor output.
type DoctorResult struct {
	Status       string        `json:"status"`
	Readiness    float64       `json:"readiness"`
	Checks       []DoctorCheck `json:"checks"`
	Dependencies []string      `json:"dependencies"`
	Capabilities []string      `json:"capabilities"`
	Version      string        `json:"version"`
	Revision     string        `json:"revision,omitempty"`
	SourceRepo   string        `json:"source_repo"`
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check MySQL connectivity and permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil {
				return err
			}

			var allChecks []DoctorCheck
			passed := 0
			total := 0

			for _, dsn := range cfg.DSNs {
				node := nodeName(dsn)
				checks := runDoctorChecks(dsn, node)
				for _, c := range checks {
					total++
					if c.Status == "pass" {
						passed++
					}
				}
				allChecks = append(allChecks, checks...)
			}

			readiness := 0.0
			if total > 0 {
				readiness = float64(passed) / float64(total)
			}

			overallStatus := "healthy"
			if readiness < 1.0 {
				overallStatus = "degraded"
			}
			if readiness == 0.0 {
				overallStatus = "unavailable"
			}

			result := DoctorResult{
				Status:    overallStatus,
				Readiness: readiness,
				Checks:    allChecks,
				Dependencies: []string{
					"mysql (5.7+, 8.0+, 8.4+)",
				},
				Capabilities: []string{
					"connections", "replication", "innodb", "queries",
					"processlist", "variables", "binlog",
				},
				Version:    appVersion,
				Revision:   appRevision,
				SourceRepo: "https://github.com/ppiankov/mysqlpulse",
			}

			prov := map[string]output.Provenance{
				"status":       output.Inferred,
				"readiness":    output.Inferred,
				"checks":       output.Observed,
				"dependencies": output.Declared,
				"capabilities": output.Declared,
				"version":      output.Declared,
				"revision":     output.Declared,
				"source_repo":  output.Declared,
			}

			table := doctorTable(result)
			return output.Render(formatFlag, output.Result{Data: result, Provenance: prov}, table)
		},
	}
}

func runDoctorChecks(dsn, node string) []DoctorCheck {
	var checks []DoctorCheck
	prefix := node + "/"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return []DoctorCheck{{
			Name: prefix + "connect", Status: "fail", Message: err.Error(),
		}}
	}
	defer db.Close()

	// Connectivity.
	if err := db.Ping(); err != nil {
		return []DoctorCheck{{
			Name: prefix + "connect", Status: "fail", Message: err.Error(),
		}}
	}
	checks = append(checks, DoctorCheck{Name: prefix + "connect", Status: "pass"})

	// Version.
	var version string
	if err := db.QueryRow("SELECT VERSION()").Scan(&version); err != nil {
		checks = append(checks, DoctorCheck{Name: prefix + "version", Status: "fail", Message: err.Error()})
	} else {
		checks = append(checks, DoctorCheck{Name: prefix + "version", Status: "pass", Message: version})
	}

	// SHOW GLOBAL STATUS permission.
	checks = append(checks, checkQuery(db, prefix+"global_status", "SHOW GLOBAL STATUS LIKE 'Uptime'"))

	// SHOW GLOBAL VARIABLES permission.
	checks = append(checks, checkQuery(db, prefix+"global_variables", "SHOW GLOBAL VARIABLES LIKE 'max_connections'"))

	// Performance schema access.
	checks = append(checks, checkQuery(db, prefix+"performance_schema",
		"SELECT 1 FROM performance_schema.events_statements_summary_by_digest LIMIT 1"))

	// Replication status (try modern first, fallback to legacy).
	replCheck := checkQuery(db, prefix+"replica_status", "SHOW REPLICA STATUS")
	if replCheck.Status == "fail" {
		replCheck = checkQuery(db, prefix+"replica_status", "SHOW SLAVE STATUS")
	}
	checks = append(checks, replCheck)

	return checks
}

func checkQuery(db *sql.DB, name, query string) DoctorCheck {
	rows, err := db.Query(query)
	if err != nil {
		return DoctorCheck{Name: name, Status: "fail", Message: err.Error()}
	}
	defer rows.Close()
	return DoctorCheck{Name: name, Status: "pass"}
}

func doctorTable(r DoctorResult) *output.Table {
	t := &output.Table{
		Headers: []string{"CHECK", "STATUS", "MESSAGE"},
	}
	for _, c := range r.Checks {
		icon := "PASS"
		if c.Status != "pass" {
			icon = "FAIL"
		}
		t.Rows = append(t.Rows, []string{c.Name, icon, c.Message})
	}
	t.Rows = append(t.Rows, []string{"", "", ""})
	t.Rows = append(t.Rows, []string{"overall", strings.ToUpper(r.Status), fmt.Sprintf("readiness=%.2f", r.Readiness)})
	return t
}

func nodeName(dsn string) string {
	// Extract host:port from DSN for display.
	if idx := strings.Index(dsn, "tcp("); idx != -1 {
		end := strings.Index(dsn[idx:], ")")
		if end != -1 {
			return dsn[idx+4 : idx+end]
		}
	}
	return dsn
}
