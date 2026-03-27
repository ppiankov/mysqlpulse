//go:build integration

package collector

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/testcontainers/testcontainers-go/modules/mysql"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

func setupMySQL(t *testing.T) (*sql.DB, string, func()) {
	t.Helper()

	ctx := context.Background()
	container, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("root"),
		mysql.WithPassword("test"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "parseTime=true")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Wait for MySQL to be ready.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		if ctx.Err() != nil {
			t.Fatalf("mysql not ready: %v", ctx.Err())
		}
		time.Sleep(500 * time.Millisecond)
	}

	cleanup := func() {
		_ = db.Close()
		_ = container.Terminate(context.Background())
	}

	return db, dsn, cleanup
}

func TestIntegration_ScrapeCollector(t *testing.T) {
	db, _, cleanup := setupMySQL(t)
	defer cleanup()

	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	c := NewScrape()
	err := c.Collect(context.Background(), db, "test:3306")
	if err != nil {
		t.Fatalf("scrape collect: %v", err)
	}
}

func TestIntegration_ConnectionsCollector(t *testing.T) {
	db, _, cleanup := setupMySQL(t)
	defer cleanup()

	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	c := NewConnections()
	err := c.Collect(context.Background(), db, "test:3306")
	if err != nil {
		t.Fatalf("connections collect: %v", err)
	}
}

func TestIntegration_InnoDBCollector(t *testing.T) {
	db, _, cleanup := setupMySQL(t)
	defer cleanup()

	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	c := NewInnoDB()
	err := c.Collect(context.Background(), db, "test:3306")
	if err != nil {
		t.Fatalf("innodb collect: %v", err)
	}
}

func TestIntegration_QueriesCollector(t *testing.T) {
	db, _, cleanup := setupMySQL(t)
	defer cleanup()

	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	c := NewQueries()
	err := c.Collect(context.Background(), db, "test:3306")
	if err != nil {
		t.Fatalf("queries collect: %v", err)
	}
}

func TestIntegration_GlobalVarsCollector(t *testing.T) {
	db, _, cleanup := setupMySQL(t)
	defer cleanup()

	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	c := NewGlobalVars()
	err := c.Collect(context.Background(), db, "test:3306")
	if err != nil {
		t.Fatalf("globalvars collect: %v", err)
	}
}

func TestIntegration_BinlogCollector(t *testing.T) {
	db, _, cleanup := setupMySQL(t)
	defer cleanup()

	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	c := NewBinlog()
	err := c.Collect(context.Background(), db, "test:3306")
	if err != nil {
		t.Fatalf("binlog collect: %v", err)
	}
}

func TestIntegration_GlobalStatus(t *testing.T) {
	db, _, cleanup := setupMySQL(t)
	defer cleanup()

	status, err := GlobalStatus(context.Background(), db)
	if err != nil {
		t.Fatalf("global status: %v", err)
	}

	if _, ok := status["Threads_connected"]; !ok {
		t.Error("expected Threads_connected in global status")
	}
	if _, ok := status["Uptime"]; !ok {
		t.Error("expected Uptime in global status")
	}
}
