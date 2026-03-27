package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	"github.com/ppiankov/mysqlpulse/internal/collector"
	"github.com/ppiankov/mysqlpulse/internal/config"
	"github.com/ppiankov/mysqlpulse/internal/engine"
	"github.com/ppiankov/mysqlpulse/internal/metrics"
	"github.com/ppiankov/mysqlpulse/internal/server"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the metrics exporter (Prometheus /metrics + /healthz)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			targets, closers, err := openTargets(cfg.DSNs)
			if err != nil {
				return err
			}
			defer func() {
				for _, cl := range closers {
					_ = cl.Close()
				}
			}()

			registry := prometheus.NewRegistry()
			metrics.Register(registry)

			collectors := []collector.Collector{
				collector.NewScrape(),
				collector.NewConnections(),
				collector.NewReplication(),
				collector.NewInnoDB(),
				collector.NewQueries(),
				collector.NewProcesslist(),
				collector.NewTableStats(),
				collector.NewBinlog(),
				collector.NewPerfSchema(),
				collector.NewGlobalVars(),
				collector.NewGTID(),
				collector.NewGroupReplication(),
			}

			eng := engine.New(cfg.PollInterval, targets, collectors)

			addr := fmt.Sprintf(":%d", cfg.MetricsPort)
			srv := server.New(addr, registry)

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			go func() {
				log.Printf("listening on %s", addr)
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("http server error: %v", err)
				}
			}()

			go func() {
				log.Printf("polling %d target(s) every %s", len(targets), cfg.PollInterval)
				if err := eng.Run(ctx); err != nil {
					log.Printf("engine error: %v", err)
				}
			}()

			<-ctx.Done()
			log.Println("shutting down")

			drainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return srv.Shutdown(drainCtx)
		},
	}
}

func openTargets(dsns []string) ([]engine.Target, []*sql.DB, error) {
	var targets []engine.Target
	var closers []*sql.DB

	for _, dsn := range dsns {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			for _, cl := range closers {
				_ = cl.Close()
			}
			return nil, nil, fmt.Errorf("open %s: %w", instanceLabel(dsn), err)
		}
		db.SetMaxOpenConns(5)
		db.SetMaxIdleConns(2)
		db.SetConnMaxLifetime(5 * time.Minute)

		targets = append(targets, engine.Target{
			Instance: instanceLabel(dsn),
			DB:       db,
		})
		closers = append(closers, db)
	}
	return targets, closers, nil
}

// instanceLabel extracts a readable label from a DSN.
// "user:pass@tcp(host:3306)/db" → "host:3306"
func instanceLabel(dsn string) string {
	if i := strings.Index(dsn, "tcp("); i >= 0 {
		rest := dsn[i+4:]
		if j := strings.Index(rest, ")"); j >= 0 {
			return rest[:j]
		}
	}
	return net.JoinHostPort("localhost", "3306")
}
