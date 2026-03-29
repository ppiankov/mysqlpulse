# mysqlpulse

MySQL observability for humans and agents. Single binary, zero infrastructure.

## What mysqlpulse is

A CLI tool that exposes MySQL health as Prometheus metrics, structured JSON, and human-readable tables. Point it at one or many MySQL instances and get diagnostics immediately — no agents, no Docker stack, no SaaS subscription.

## What mysqlpulse is NOT

- **Does NOT manage** MySQL configuration, users, or schemas — it observes, never mutates
- **Does NOT replace** Prometheus, Grafana, or your alerting stack — it feeds them
- **Does NOT store** metrics or historical data — it collects and exposes, storage is your choice
- **Does NOT execute** queries on your behalf — it reads system tables and status variables only
- **Does NOT own** your monitoring pipeline — it is one link in the chain, not the chain

## Philosophy

Principiis obsta. Observe everything, touch nothing. Every output is structured, every command works for both a human at 3 AM and an agent in a CI pipeline. If you need a monitoring stack to monitor your database, the tool is wrong.

## Quick Start

```bash
brew install ppiankov/tap/mysqlpulse

# Generate config
mysqlpulse init

# Check connectivity and permissions
export MYSQL_DSN="user:pass@tcp(localhost:3306)/"
mysqlpulse doctor

# Start Prometheus exporter
mysqlpulse serve

# One-shot diagnostic (JSON for agents)
mysqlpulse report --format json

# Health check for CI pipelines (Nagios-compatible exit codes)
mysqlpulse check repl-lag --warn 5 --crit 30

# Live terminal dashboard
mysqlpulse watch
```

## Commands

| Command | Description | Exit codes |
|---------|-------------|------------|
| `mysqlpulse serve` | Start Prometheus metrics exporter (/metrics, /healthz) | 0/1 |
| `mysqlpulse check <metric>` | Threshold health check (Nagios-compatible) | 0=OK, 1=WARN, 2=CRIT, 3=UNKNOWN |
| `mysqlpulse report` | One-shot diagnostic dump (all metrics) | 0/1 |
| `mysqlpulse innodb` | Structured InnoDB STATUS parser | 0/1 |
| `mysqlpulse topology` | Replication topology discovery (supports `--dot`) | 0/1 |
| `mysqlpulse diff` | Compare configuration across nodes | 0/1 |
| `mysqlpulse watch` | Live terminal dashboard (4 modes) | 0 |
| `mysqlpulse status` | One-shot health summary per node | 0/1 |
| `mysqlpulse doctor` | Check MySQL connectivity and permissions | 0/1 |
| `mysqlpulse init` | Generate config with sensible defaults | 0/1 |
| `mysqlpulse version` | Print version and build info | 0 |

Every command supports `--format json` for machine consumption and `--format table` (default) for humans.

### Check Metrics

Available metrics for `mysqlpulse check`:

- `repl-lag` — Replication lag in seconds
- `connections` — Active connections count
- `threads-running` — Running threads count
- `buffer-pool` — InnoDB buffer pool hit ratio
- `slow-queries` — Cumulative slow query count
- `deadlocks` — Cumulative deadlock count

### Watch Modes

`mysqlpulse watch` cycles through 4 modes with `Tab` or `1-4`:

1. **overview** — connections, QPS, slow queries, deadlocks, uptime
2. **processlist** — live process list (innotop Q mode replacement)
3. **replication** — IO/SQL threads, lag, GTID, source host
4. **innodb** — buffer pool, row ops, lock waits, history list

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MYSQL_DSN` | `root@tcp(localhost:3306)/` | MySQL DSN(s), comma-separated for multi-target |
| `METRICS_PORT` | `9104` | Prometheus metrics port |
| `POLL_INTERVAL` | `15s` | Collection interval |
| `ALERT_TELEGRAM_TOKEN` | | Telegram bot token for alerts |
| `ALERT_TELEGRAM_CHAT` | | Telegram chat ID for alerts |
| `ALERT_WEBHOOK_URL` | | Webhook URL for alerts |
| `GRAFANA_URL` | | Grafana URL for anomaly annotations |
| `GRAFANA_TOKEN` | | Grafana API token for annotations |

## Collectors

12 collectors run on each poll cycle:

| Collector | Source | Metrics |
|-----------|--------|---------|
| scrape | `db.Ping` | mysql_up, scrape_duration, scrape_errors |
| connections | `SHOW GLOBAL STATUS` | threads_connected/running/cached, max_used, aborted |
| replication | `SHOW REPLICA STATUS` | lag, IO/SQL running, behind_bytes |
| innodb | `SHOW GLOBAL STATUS` | buffer pool pages/bytes/hit_ratio, deadlocks, row_lock_waits, history_list |
| queries | `SHOW GLOBAL STATUS` | queries, questions, commands by type, slow_queries |
| processlist | `SHOW PROCESSLIST` | by state/command/user, longest, locked |
| tablestats | `information_schema.TABLES` | rows, data/index/free bytes, auto_increment headroom |
| binlog | `SHOW BINARY LOGS` | count, total size, cache usage |
| perfschema | `events_statements_summary_by_digest` | top-N queries by time |
| globalvars | `SHOW GLOBAL VARIABLES` | max_connections, buffer_pool_size, read_only, gtid_mode |
| gtid | `SHOW GLOBAL VARIABLES` | gtid_executed/purged set counts |
| grouprepl | `performance_schema.replication_group_members` | member state, queue, conflicts |

## Alerting

Built-in alerts fire via Telegram and/or webhook when thresholds are breached:

- Replication stopped (IO or SQL thread down)
- Replication lag > 30s
- Buffer pool < 10% free pages
- Connections > 80% of max_connections
- Deadlocks detected
- History list length > 1000

Alerts include the **host in the header** for immediate identification and use cooldown/dedup to prevent alert storms.

## Deployment

### Docker

```bash
docker run -e MYSQL_DSN="user:pass@tcp(db:3306)/" ghcr.io/ppiankov/mysqlpulse:0.1.1
```

### Helm

```bash
helm install mysqlpulse ./charts/mysqlpulse \
  --set 'targets[0].name=primary' \
  --set 'targets[0].dsn=user:pass@tcp(db:3306)/' \
  --set serviceMonitor.enabled=true \
  --set prometheusRule.enabled=true
```

The Helm chart includes:
- One Deployment, Secret, and Service per target
- Support for multiple targets (one exporter per database)
- `existingSecret` support for pre-provisioned DSN secrets
- ServiceMonitor with per-target job labels (optional, for Prometheus Operator)
- PrometheusRule with 7 alert rules (optional)

### Grafana

Import `grafana/mysqlpulse-dashboard.json` for a pre-built dashboard with connections, replication, InnoDB, QPS, commands, deadlocks, processlist, and binlog panels.

## Architecture

```
cmd/mysqlpulse/     Entry point, ldflags version injection
internal/
  cli/              11 Cobra commands
  config/           Config loading from env vars
  output/           Shared JSON/table renderer with provenance
  engine/           Poll loop with retry and alerting
  collector/        12 MySQL metric collectors
  metrics/          Prometheus metric descriptors
  server/           HTTP server (/metrics, /healthz)
  alerter/          Telegram, webhook, Grafana annotations
  innodb/           Structured InnoDB STATUS parser
charts/             Helm chart
grafana/            Grafana dashboard JSON
```

## Known Limitations

- No config file loading — env vars only (init generates YAML but serve doesn't read it yet)
- Integration tests require Docker (`go test -tags integration`)
- Watch mode uses `stty` for raw terminal — may not work on all terminals
- Table stats collector can be slow on schemas with many tables

## License

[MIT](LICENSE)
