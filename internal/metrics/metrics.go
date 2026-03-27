package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	MySQLUp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_up",
		Help: "Whether MySQL is reachable (1=up, 0=down).",
	}, []string{"instance"})

	ScrapeDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_scrape_duration_seconds",
		Help: "Duration of the last scrape in seconds.",
	}, []string{"instance"})

	ScrapeErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mysql_scrape_errors_total",
		Help: "Total number of scrape errors.",
	}, []string{"instance"})
	// Connection metrics (WO-6).
	ThreadsConnected = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_threads_connected",
		Help: "Current number of open connections.",
	}, []string{"instance"})

	ThreadsRunning = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_threads_running",
		Help: "Current number of threads not sleeping.",
	}, []string{"instance"})

	ThreadsCached = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_threads_cached",
		Help: "Number of threads in the thread cache.",
	}, []string{"instance"})

	MaxUsedConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_max_used_connections",
		Help: "Maximum number of connections used simultaneously since server start.",
	}, []string{"instance"})

	// MySQL cumulative counters exposed as gauges — Prometheus rate() handles derivation.
	ConnectionsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_connections_total",
		Help: "Total number of connection attempts (cumulative from MySQL).",
	}, []string{"instance"})

	AbortedConnectsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_aborted_connects_total",
		Help: "Total number of failed connection attempts (cumulative from MySQL).",
	}, []string{"instance"})

	AbortedClientsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_aborted_clients_total",
		Help: "Total number of connections aborted due to client not closing properly (cumulative from MySQL).",
	}, []string{"instance"})

	// Replication metrics (WO-7).
	ReplLagSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_repl_lag_seconds",
		Help: "Seconds behind source (Seconds_Behind_Source).",
	}, []string{"instance"})

	ReplIORunning = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_repl_io_running",
		Help: "Whether replica IO thread is running (1=yes, 0=no).",
	}, []string{"instance"})

	ReplSQLRunning = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_repl_sql_running",
		Help: "Whether replica SQL thread is running (1=yes, 0=no).",
	}, []string{"instance"})

	ReplBehindBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_repl_behind_bytes",
		Help: "Bytes difference between read and exec log positions.",
	}, []string{"instance"})

	ReplRunning = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_repl_running",
		Help: "Whether replication is fully running (IO and SQL threads both up).",
	}, []string{"instance"})

	// InnoDB metrics (WO-8).
	InnoDBBufferPoolPages = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_innodb_buffer_pool_pages",
		Help: "InnoDB buffer pool pages by state.",
	}, []string{"instance", "state"})

	InnoDBBufferPoolBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_innodb_buffer_pool_bytes",
		Help: "InnoDB buffer pool bytes by state.",
	}, []string{"instance", "state"})

	InnoDBBufferPoolHitRatio = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_innodb_buffer_pool_hit_ratio",
		Help: "InnoDB buffer pool hit ratio (0.0-1.0).",
	}, []string{"instance"})

	InnoDBRowLockWaitsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_innodb_row_lock_waits_total",
		Help: "Total InnoDB row lock waits (cumulative from MySQL).",
	}, []string{"instance"})

	InnoDBDeadlocksTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_innodb_deadlocks_total",
		Help: "Total InnoDB deadlocks (cumulative from MySQL).",
	}, []string{"instance"})

	InnoDBHistoryListLength = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_innodb_history_list_length",
		Help: "InnoDB history list length (undo log entries not yet purged).",
	}, []string{"instance"})

	// Query metrics (WO-9).
	QueriesTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_queries_total",
		Help: "Total queries executed (cumulative from MySQL).",
	}, []string{"instance"})

	QuestionsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_questions_total",
		Help: "Total client-originated statements (cumulative from MySQL).",
	}, []string{"instance"})

	CommandsTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_commands_total",
		Help: "Total commands by type (cumulative from MySQL).",
	}, []string{"instance", "command"})

	SlowQueriesTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_slow_queries_total",
		Help: "Total slow queries (cumulative from MySQL).",
	}, []string{"instance"})

	SelectFullJoinTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_select_full_join_total",
		Help: "Total full joins without index (cumulative from MySQL).",
	}, []string{"instance"})

	SortMergePassesTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_sort_merge_passes_total",
		Help: "Total sort merge passes (cumulative from MySQL).",
	}, []string{"instance"})

	// Process list metrics (WO-10).
	ProcesslistByState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_processlist_count",
		Help: "Number of processes by state.",
	}, []string{"instance", "state"})

	ProcesslistByCommand = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_processlist_by_command",
		Help: "Number of processes by command type.",
	}, []string{"instance", "command"})

	ProcesslistLongest = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_processlist_longest_seconds",
		Help: "Duration of the longest running query in seconds.",
	}, []string{"instance"})

	ProcesslistByUser = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_processlist_by_user",
		Help: "Number of processes by user.",
	}, []string{"instance", "user"})

	ProcesslistLocked = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_processlist_locked",
		Help: "Number of processes in locked state.",
	}, []string{"instance"})

	// Table stats (WO-11).
	TableRows = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_table_rows",
		Help: "Estimated row count per table.",
	}, []string{"instance", "schema", "table"})

	TableDataBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_table_data_bytes",
		Help: "Data size in bytes per table.",
	}, []string{"instance", "schema", "table"})

	TableIndexBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_table_index_bytes",
		Help: "Index size in bytes per table.",
	}, []string{"instance", "schema", "table"})

	TableFreeBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_table_free_bytes",
		Help: "Free space (fragmentation) in bytes per table.",
	}, []string{"instance", "schema", "table"})

	TableAutoIncHeadroom = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_table_auto_increment_headroom",
		Help: "Remaining auto_increment headroom before overflow.",
	}, []string{"instance", "schema", "table"})

	// Binary log (WO-12).
	BinlogCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_binlog_count",
		Help: "Number of binary log files.",
	}, []string{"instance"})

	BinlogSizeBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_binlog_size_bytes",
		Help: "Total size of all binary log files.",
	}, []string{"instance"})

	BinlogCacheUseTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_binlog_cache_use_total",
		Help: "Number of transactions that used the binlog cache (cumulative).",
	}, []string{"instance"})

	BinlogCacheDiskUseTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_binlog_cache_disk_use_total",
		Help: "Number of transactions that used a temp file for binlog cache (cumulative).",
	}, []string{"instance"})

	// Performance schema (WO-13).
	PerfQueryAvgSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_perf_query_avg_seconds",
		Help: "Average query execution time in seconds.",
	}, []string{"instance", "digest"})

	PerfQueryCalls = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_perf_query_calls",
		Help: "Total number of query executions.",
	}, []string{"instance", "digest"})

	PerfQueryRowsExamined = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_perf_query_rows_examined",
		Help: "Total rows examined by query digest.",
	}, []string{"instance", "digest"})

	// Global variables (WO-14).
	MaxConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_max_connections",
		Help: "Configured max_connections value.",
	}, []string{"instance"})

	InnoDBBufferPoolSizeBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_innodb_buffer_pool_size_bytes",
		Help: "Configured innodb_buffer_pool_size in bytes.",
	}, []string{"instance"})

	ReadOnly = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_read_only",
		Help: "Whether read_only is enabled (1=on, 0=off).",
	}, []string{"instance"})

	SuperReadOnly = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_super_read_only",
		Help: "Whether super_read_only is enabled (1=on, 0=off).",
	}, []string{"instance"})

	GTIDMode = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mysql_gtid_mode",
		Help: "GTID mode (1=ON, 0=OFF).",
	}, []string{"instance"})
)

// Register adds all metric descriptors to the given registerer.
func Register(reg prometheus.Registerer) {
	reg.MustRegister(
		// Scrape health.
		MySQLUp, ScrapeDuration, ScrapeErrors,
		// Connections (WO-6).
		ThreadsConnected, ThreadsRunning, ThreadsCached,
		MaxUsedConnections, ConnectionsTotal,
		AbortedConnectsTotal, AbortedClientsTotal,
		// Replication (WO-7).
		ReplLagSeconds, ReplIORunning, ReplSQLRunning,
		ReplBehindBytes, ReplRunning,
		// InnoDB (WO-8).
		InnoDBBufferPoolPages, InnoDBBufferPoolBytes,
		InnoDBBufferPoolHitRatio, InnoDBRowLockWaitsTotal,
		InnoDBDeadlocksTotal, InnoDBHistoryListLength,
		// Queries (WO-9).
		QueriesTotal, QuestionsTotal, CommandsTotal,
		SlowQueriesTotal, SelectFullJoinTotal, SortMergePassesTotal,
		// Process list (WO-10).
		ProcesslistByState, ProcesslistByCommand,
		ProcesslistLongest, ProcesslistByUser, ProcesslistLocked,
		// Table stats (WO-11).
		TableRows, TableDataBytes, TableIndexBytes,
		TableFreeBytes, TableAutoIncHeadroom,
		// Binary log (WO-12).
		BinlogCount, BinlogSizeBytes,
		BinlogCacheUseTotal, BinlogCacheDiskUseTotal,
		// Performance schema (WO-13).
		PerfQueryAvgSeconds, PerfQueryCalls, PerfQueryRowsExamined,
		// Global variables (WO-14).
		MaxConnections, InnoDBBufferPoolSizeBytes,
		ReadOnly, SuperReadOnly, GTIDMode,
	)
}
