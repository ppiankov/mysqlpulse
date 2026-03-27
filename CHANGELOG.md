# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

Change classification: **Breaking** | **Additive** | **Behavioral** | **Internal**

## [Unreleased]

## [0.1.0] - 2026-03-27

### Additive
- Core scaffold: Cobra CLI with 10 commands
- `mysqlpulse serve` — poll loop with Prometheus /metrics and /healthz endpoints
- `mysqlpulse check` — threshold-based health checks with Nagios-compatible exit codes (0/1/2/3)
- `mysqlpulse report` — one-shot diagnostic dump (structured pt-mysql-summary replacement)
- `mysqlpulse innodb` — structured InnoDB STATUS parser (deadlocks, buffer pool, redo log, row ops)
- `mysqlpulse topology` — replication topology discovery with --format dot for Graphviz
- `mysqlpulse diff` — config comparison across nodes (detect drift in replica fleets)
- `mysqlpulse watch` — live terminal dashboard with 4 modes (overview, processlist, replication, innodb)
- `mysqlpulse doctor` — ANCC-compliant readiness checks with provenance
- `mysqlpulse init` — config generation with sensible defaults
- 10 collectors: scrape health, connections, replication, InnoDB, queries, processlist, table stats, binlog, performance schema, global variables
- `--format` flag (json/table) on all commands
- Provenance classification in all JSON output (observed/declared/inferred/unknown)
- Multi-target DSN support via comma-separated MYSQL_DSN
- Exponential backoff retry for MySQL connectivity (3 attempts)
- Graceful shutdown with signal handling
