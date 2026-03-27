package cli

import (
	"github.com/spf13/cobra"
)

var (
	appVersion  = "dev"
	appRevision = "unknown"
	formatFlag  string
)

// SetVersion is called from main to inject build-time version info.
func SetVersion(version, revision string) {
	appVersion = version
	appRevision = revision
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "mysqlpulse",
		Short:         "MySQL observability for humans and agents",
		Long:          "mysqlpulse exposes MySQL health as Prometheus metrics, structured JSON, and human-readable tables. Zero infrastructure, single binary.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&formatFlag, "format", "table", "output format: json, table")

	cmd.AddCommand(
		newVersionCmd(),
		newInitCmd(),
		newDoctorCmd(),
		newServeCmd(),
		newCheckCmd(),
		newReportCmd(),
		newInnoDBCmd(),
		newTopologyCmd(),
		newDiffCmd(),
		newWatchCmd(),
	)

	return cmd
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}
