package collector

import (
	"context"
	"database/sql"
)

// Collector gathers metrics from a MySQL instance.
type Collector interface {
	Name() string
	Collect(ctx context.Context, db *sql.DB, instance string) error
}
