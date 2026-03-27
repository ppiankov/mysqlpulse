# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

Change classification: **Breaking** | **Additive** | **Behavioral** | **Internal**

## [Unreleased]

### Additive
- Core scaffold: Cobra CLI with `version`, `init`, `doctor`, `serve` commands
- `--format` flag (json/table) on all commands (ANCC requirement #3)
- `mysqlpulse init` generates config with sensible defaults (ANCC requirement #6)
- `mysqlpulse doctor` with ANCC-compliant output schema (readiness, provenance, dependencies)
- Provenance fields in all JSON output (ANCC Convention 5)
- Shared output renderer with JSON envelope and table formatting
- Config from environment variables (MYSQL_DSN, METRICS_PORT, POLL_INTERVAL)
- Multi-target DSN support via comma-separated MYSQL_DSN
