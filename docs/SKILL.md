# mysqlpulse

MySQL observability CLI for humans and agents. Single binary, zero infrastructure.

## What it does

Exposes MySQL health as Prometheus metrics, structured JSON, and human-readable tables. Monitors connections, replication, InnoDB, queries, processlist, table stats, binary logs, performance schema, and global variables.

## What it does NOT do

- Does not replace a full monitoring stack (Grafana, PMM, Datadog)
- Does not modify MySQL configuration or data
- Does not require any infrastructure beyond a MySQL connection
- Does not store historical data — it is a point-in-time observer

## Commands

| Command | Description | Exit codes |
|---------|-------------|------------|
| `version` | Print version and build info | 0 |
| `init` | Generate config file template | 0/1 |
| `doctor` | Check MySQL connectivity and permissions | 0/1 |
| `serve` | Start Prometheus metrics exporter (/metrics, /healthz) | 0/1 |
| `check <metric>` | Threshold health check (Nagios-compatible) | 0=OK, 1=WARN, 2=CRIT, 3=UNKNOWN |
| `report` | One-shot diagnostic dump (all metrics) | 0/1 |
| `innodb` | Structured InnoDB STATUS parser | 0/1 |
| `topology` | Replication topology discovery | 0/1 |
| `diff` | Compare configuration across nodes | 0/1 |
| `watch` | Live terminal dashboard (4 modes) | 0 |
| `status` | One-shot health summary per node | 0/1 |

## Configuration

Environment variables:
- `MYSQL_DSN` — MySQL connection string(s), comma-separated for multi-target
- `METRICS_PORT` — HTTP port for Prometheus metrics (default: 9104)
- `POLL_INTERVAL` — Query interval (default: 15s, minimum: 1s)

## Output formats

All commands support `--format json` and `--format table` (default).

JSON output follows the envelope pattern:
```json
{
  "data": { ... },
  "provenance": {
    "field": "observed|declared|inferred|unknown"
  }
}
```

## Check metrics

Available metrics for `mysqlpulse check`:
- `repl-lag` — Replication lag in seconds
- `connections` — Active connections count
- `threads-running` — Running threads count
- `buffer-pool` — InnoDB buffer pool hit ratio
- `slow-queries` — Cumulative slow query count
- `deadlocks` — Cumulative deadlock count

## Provenance classification

Every JSON field is classified by data source:
- **observed** — Live from MySQL query
- **declared** — From configuration or annotation
- **inferred** — Computed or derived from observed values
- **unknown** — Source unclear or stale

## Dependencies

- MySQL 5.7+ or 8.0+ (supports both modern and legacy syntax)
- No external dependencies beyond a MySQL connection string
