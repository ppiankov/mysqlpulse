package collector

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

// GlobalStatus runs SHOW GLOBAL STATUS and returns a map of variable_name → value.
func GlobalStatus(ctx context.Context, db *sql.DB) (map[string]float64, error) {
	rows, err := db.QueryContext(ctx, "SHOW GLOBAL STATUS")
	if err != nil {
		return nil, fmt.Errorf("SHOW GLOBAL STATUS: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]float64)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			result[name] = v
		}
	}
	return result, rows.Err()
}
