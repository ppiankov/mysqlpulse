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
```

## Commands

| Command | Description |
|---------|-------------|
| `mysqlpulse init` | Generate config with sensible defaults |
| `mysqlpulse doctor` | Check MySQL connectivity and permissions |
| `mysqlpulse serve` | Start Prometheus metrics exporter |
| `mysqlpulse version` | Print version and build info |

Every command supports `--format json` for machine consumption and `--format table` (default) for humans.

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MYSQL_DSN` | `root@tcp(localhost:3306)/` | MySQL DSN(s), comma-separated for multi-target |
| `METRICS_PORT` | `9104` | Prometheus metrics port |
| `POLL_INTERVAL` | `15s` | Collection interval |

## Architecture

```
cmd/mysqlpulse/     main.go — entry point, injects version
internal/
  cli/              Cobra commands (root, version, init, doctor, serve)
  config/           Config loading from env vars
  output/           Shared JSON/table renderer with provenance
  engine/           Poll loop (WO-2)
  collector/        MySQL metric collectors (WO-6+)
```

## Known Limitations

- `serve` command is scaffolded but not yet implemented (WO-2)
- No config file loading yet — env vars only
- Single-binary distribution not yet set up (WO-24, WO-25)

## Roadmap

See work orders in the project tracker. Phases:
1. Foundation — scaffold, poll loop, doctor, multi-target
2. Collectors — connections, replication, InnoDB, queries, processlist
3. Advanced — GTID, group replication, topology discovery, config diff
4. Operations — alerting, annotations, watch mode, Grafana dashboard, Helm chart
5. Release — ANCC compliance, integration tests, Homebrew formula

## License

MIT
