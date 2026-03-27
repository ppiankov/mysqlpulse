package collector

import (
	"context"
	"database/sql"
	"strings"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

// GroupReplication collects MySQL Group Replication metrics.
type GroupReplication struct{}

func NewGroupReplication() *GroupReplication { return &GroupReplication{} }

func (g *GroupReplication) Name() string { return "grouprepl" }

func (g *GroupReplication) Collect(ctx context.Context, db *sql.DB, instance string) error {
	// Check if GR tables exist.
	members, err := queryGRMembers(ctx, db)
	if err != nil {
		// Group Replication not active — not an error.
		return nil
	}

	metrics.GRMembersTotal.WithLabelValues(instance).Set(float64(len(members)))

	for _, m := range members {
		val := float64(0)
		if strings.EqualFold(m.state, "ONLINE") {
			val = 1
		}
		metrics.GRMemberState.WithLabelValues(instance, m.host).Set(val)
	}

	// Applier stats from replication_group_member_stats.
	stats, err := queryGRStats(ctx, db)
	if err != nil {
		return nil
	}

	metrics.GRTransactionsInQueue.WithLabelValues(instance).Set(stats.transactionsInQueue)
	metrics.GRConflictDetectedTotal.WithLabelValues(instance).Set(stats.conflictsDetected)
	metrics.GRFlowControlCount.WithLabelValues(instance).Set(stats.flowControlCount)

	return nil
}

type grMember struct {
	host  string
	state string
}

func queryGRMembers(ctx context.Context, db *sql.DB) ([]grMember, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT MEMBER_HOST, MEMBER_STATE FROM performance_schema.replication_group_members")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var members []grMember
	for rows.Next() {
		var m grMember
		if err := rows.Scan(&m.host, &m.state); err != nil {
			continue
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

type grStats struct {
	transactionsInQueue float64
	conflictsDetected   float64
	flowControlCount    float64
}

func queryGRStats(ctx context.Context, db *sql.DB) (grStats, error) {
	var s grStats
	row := db.QueryRowContext(ctx,
		`SELECT COALESCE(COUNT_TRANSACTIONS_IN_QUEUE, 0),
			COALESCE(COUNT_CONFLICTS_DETECTED, 0),
			COALESCE(COUNT_TRANSACTIONS_ROWS_VALIDATING, 0)
		FROM performance_schema.replication_group_member_stats
		WHERE MEMBER_ID = @@server_uuid
		LIMIT 1`)
	err := row.Scan(&s.transactionsInQueue, &s.conflictsDetected, &s.flowControlCount)
	return s, err
}
