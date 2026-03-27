package main

import (
	"os"

	"github.com/ppiankov/mysqlpulse/internal/cli"
)

// Set by -ldflags at build time.
var (
	version  = "dev"
	revision = "unknown"
)

func main() {
	cli.SetVersion(version, revision)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
