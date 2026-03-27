package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the metrics exporter (Prometheus /metrics + /healthz)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Placeholder — WO-2 implements the full poll loop and HTTP server.
			fmt.Println("serve: not yet implemented (WO-2)")
			return nil
		},
	}
}
