package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ppiankov/mysqlpulse/internal/output"
	"github.com/spf13/cobra"
)

const defaultConfigYAML = `# mysqlpulse configuration
# Docs: https://github.com/ppiankov/mysqlpulse

# MySQL DSNs to monitor (comma-separated in env, list in YAML).
dsns:
  - "root@tcp(localhost:3306)/"

# Prometheus metrics port.
metrics_port: 9104

# Poll interval (Go duration: 15s, 1m, etc).
poll_interval: "15s"

# Enabled collectors.
collectors:
  - connections
  - replication
  - innodb
  - queries
  - processlist
  - variables
  - binlog
`

func newInitCmd() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a config file with sensible defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputPath == "" {
				outputPath = "mysqlpulse.yaml"
			}

			abs, err := filepath.Abs(outputPath)
			if err != nil {
				return err
			}

			if _, err := os.Stat(abs); err == nil {
				return fmt.Errorf("file already exists: %s", abs)
			}

			if err := os.WriteFile(abs, []byte(defaultConfigYAML), 0644); err != nil {
				return err
			}

			if formatFlag == "json" {
				result := output.Result{
					Data: map[string]string{
						"path":   abs,
						"status": "created",
					},
					Provenance: map[string]output.Provenance{
						"path":   output.Declared,
						"status": output.Declared,
					},
				}
				return output.Render("json", result, nil)
			}

			fmt.Printf("Config written to %s\n", abs)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "output path (default: ./mysqlpulse.yaml)")
	return cmd
}
