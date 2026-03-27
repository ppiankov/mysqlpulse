package collector

import (
	"context"
	"database/sql"
	"strings"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// GTID collects GTID-based replication metrics.
type GTID struct{}

func NewGTID() *GTID { return &GTID{} }

func (g *GTID) Name() string { return "gtid" }

func (g *GTID) Collect(ctx context.Context, db *sql.DB, instance string) error {
	vars, err := queryGlobalVars(ctx, db)
	if err != nil {
		return err
	}

	if v, ok := vars["gtid_executed"]; ok && v != "" {
		metrics.GTIDExecutedCount.WithLabelValues(instance).Set(float64(countGTIDSets(v)))
	}

	if v, ok := vars["gtid_purged"]; ok && v != "" {
		metrics.GTIDPurgedCount.WithLabelValues(instance).Set(float64(countGTIDSets(v)))
	}

	return nil
}

// countGTIDSets counts the number of GTID sets in a GTID string.
// Format: "uuid:interval,uuid:interval,..." — count unique UUIDs.
func countGTIDSets(gtid string) int {
	if gtid == "" {
		return 0
	}
	uuids := make(map[string]bool)
	for _, part := range strings.Split(gtid, ",") {
		part = strings.TrimSpace(part)
		if i := strings.Index(part, ":"); i > 0 {
			uuids[part[:i]] = true
		}
	}
	return len(uuids)
}
