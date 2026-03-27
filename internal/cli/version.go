package cli

import (
	"github.com/ppiankov/mysqlpulse/internal/output"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and build info",
		RunE: func(cmd *cobra.Command, args []string) error {
			data := map[string]string{
				"version":  appVersion,
				"revision": appRevision,
			}

			result := output.Result{
				Data: data,
				Provenance: map[string]output.Provenance{
					"version":  output.Declared,
					"revision": output.Declared,
				},
			}

			table := &output.Table{
				Headers: []string{"FIELD", "VALUE"},
				Rows: [][]string{
					{"version", appVersion},
					{"revision", appRevision},
				},
			}

			return output.Render(formatFlag, result, table)
		},
	}
}
