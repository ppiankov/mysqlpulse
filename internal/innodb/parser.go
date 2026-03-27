package innodb

import (
	"regexp"
	"strconv"
	"strings"
)

// Status is the structured representation of SHOW ENGINE INNODB STATUS.
type Status struct {
	Semaphores   Semaphores   `json:"semaphores"`
	Deadlocks    []Deadlock   `json:"deadlocks,omitempty"`
	Transactions Transactions `json:"transactions"`
	BufferPool   BufferPool   `json:"buffer_pool"`
	RowOps       RowOps       `json:"row_operations"`
	RedoLog      RedoLog      `json:"redo_log"`
}

// Semaphores holds mutex/rw-lock wait information.
type Semaphores struct {
	Waits         int64 `json:"waits"`
	Spins         int64 `json:"spins"`
	Rounds        int64 `json:"rounds"`
	OSWaits       int64 `json:"os_waits"`
	SpinRounds    int64 `json:"spin_rounds"`
	SpinOSWaits   int64 `json:"spin_os_waits"`
	RWSharedSpins int64 `json:"rw_shared_spins"`
	RWExclSpins   int64 `json:"rw_excl_spins"`
}

// Deadlock is a parsed deadlock event.
type Deadlock struct {
	Transaction1 DeadlockTrx `json:"transaction_1"`
	Transaction2 DeadlockTrx `json:"transaction_2"`
	Victim       string      `json:"victim"`
}

// DeadlockTrx is a transaction involved in a deadlock.
type DeadlockTrx struct {
	ID      string `json:"id,omitempty"`
	Query   string `json:"query,omitempty"`
	Table   string `json:"table,omitempty"`
	Lock    string `json:"lock_type,omitempty"`
	Waiting bool   `json:"waiting"`
}

// Transactions holds active transaction summary.
type Transactions struct {
	ActiveCount       int   `json:"active_count"`
	HistoryListLength int64 `json:"history_list_length"`
	PurgeLag          int64 `json:"purge_lag"`
}

// BufferPool holds buffer pool statistics.
type BufferPool struct {
	TotalPages   int64   `json:"total_pages"`
	FreePages    int64   `json:"free_pages"`
	DirtyPages   int64   `json:"dirty_pages"`
	DataPages    int64   `json:"data_pages"`
	HitRate      float64 `json:"hit_rate"`
	PendingReads int64   `json:"pending_reads"`
}

// RowOps holds row operation rates.
type RowOps struct {
	ReadsPerSec   float64 `json:"reads_per_sec"`
	InsertsPerSec float64 `json:"inserts_per_sec"`
	UpdatesPerSec float64 `json:"updates_per_sec"`
	DeletesPerSec float64 `json:"deletes_per_sec"`
}

// RedoLog holds redo log/LSN information.
type RedoLog struct {
	LSN           int64 `json:"lsn"`
	CheckpointLSN int64 `json:"checkpoint_lsn"`
	CheckpointAge int64 `json:"checkpoint_age"`
	Flushed       int64 `json:"flushed_to"`
}

// Parse parses the raw output of SHOW ENGINE INNODB STATUS.
func Parse(raw string) Status {
	sections := splitSections(raw)
	s := Status{}

	s.Semaphores = parseSemaphores(sections["SEMAPHORES"])
	s.Deadlocks = parseDeadlocks(sections["LATEST DETECTED DEADLOCK"])
	s.Transactions = parseTransactions(sections["TRANSACTIONS"])
	s.BufferPool = parseBufferPool(sections["BUFFER POOL AND MEMORY"])
	s.RowOps = parseRowOps(sections["ROW OPERATIONS"])
	s.RedoLog = parseRedoLog(sections["LOG"])

	return s
}

// splitSections splits the InnoDB STATUS output into named sections.
// The format is: line of dashes, section name, line of dashes, then content.
func splitSections(raw string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(raw, "\n")

	isDashes := func(s string) bool {
		s = strings.TrimRight(s, "\r")
		return len(s) >= 3 && strings.Trim(s, "-=") == ""
	}

	var currentSection string
	var sectionLines []string

	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")

		// Pattern: dashes, then name, then dashes.
		if isDashes(line) && i+2 < len(lines) {
			nameLine := strings.TrimRight(lines[i+1], "\r")
			nextLine := strings.TrimRight(lines[i+2], "\r")

			if isDashes(nextLine) && nameLine != "" && !isDashes(nameLine) {
				// Save previous section.
				if currentSection != "" {
					result[currentSection] = strings.Join(sectionLines, "\n")
				}
				currentSection = strings.TrimSpace(nameLine)
				sectionLines = nil
				i += 2 // skip name + closing dashes
				continue
			}
		}

		sectionLines = append(sectionLines, line)
	}

	if currentSection != "" {
		result[currentSection] = strings.Join(sectionLines, "\n")
	}

	return result
}

var reInt = regexp.MustCompile(`(\d+)`)

func parseSemaphores(text string) Semaphores {
	s := Semaphores{}
	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "mutex spin waits"):
			nums := reInt.FindAllString(line, -1)
			if len(nums) >= 3 {
				s.Spins = atoi64(nums[0])
				s.Rounds = atoi64(nums[1])
				s.OSWaits = atoi64(nums[2])
			}
		case strings.Contains(lower, "rw-shared spins"):
			nums := reInt.FindAllString(line, -1)
			if len(nums) >= 1 {
				s.RWSharedSpins = atoi64(nums[0])
			}
		case strings.Contains(lower, "rw-excl spins"):
			nums := reInt.FindAllString(line, -1)
			if len(nums) >= 1 {
				s.RWExclSpins = atoi64(nums[0])
			}
		case strings.Contains(lower, "spin rounds"):
			nums := reInt.FindAllString(line, -1)
			if len(nums) >= 1 {
				s.SpinRounds = atoi64(nums[0])
			}
		case strings.Contains(lower, "os waits") && !strings.Contains(lower, "spin"):
			nums := reInt.FindAllString(line, -1)
			if len(nums) >= 1 {
				s.Waits = atoi64(nums[0])
			}
		}
	}
	return s
}

func parseDeadlocks(text string) []Deadlock {
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	var dl Deadlock
	var current *DeadlockTrx
	var found bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		switch {
		case strings.Contains(lower, "(1) transaction"):
			found = true
			dl = Deadlock{}
			current = &dl.Transaction1
		case strings.Contains(lower, "(2) transaction"):
			current = &dl.Transaction2
		case strings.Contains(lower, "we roll back transaction"):
			if strings.Contains(lower, "(1)") {
				dl.Victim = "transaction_1"
			} else {
				dl.Victim = "transaction_2"
			}
		case strings.Contains(lower, "waiting for this lock") && current != nil:
			current.Waiting = true
		case strings.Contains(lower, "lock mode") && current != nil:
			current.Lock = trimmed
		case strings.Contains(lower, "mysql tables in use") && current != nil:
			// table info
		case strings.HasPrefix(lower, "table") && current != nil:
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				current.Table = parts[1]
			}
		case current != nil && (strings.HasPrefix(lower, "insert") || strings.HasPrefix(lower, "update") ||
			strings.HasPrefix(lower, "delete") || strings.HasPrefix(lower, "select")):
			current.Query = trimmed
		}
	}

	if found {
		return []Deadlock{dl}
	}
	return nil
}

func parseTransactions(text string) Transactions {
	t := Transactions{}
	reTrxActive := regexp.MustCompile(`(?i)TRANSACTION\s+\d+,\s+ACTIVE`)
	reHistList := regexp.MustCompile(`(?i)History list length\s+(\d+)`)
	rePurge := regexp.MustCompile(`(?i)Purge done for trx.*undo n:o\s*<\s*(\d+)`)

	for _, line := range strings.Split(text, "\n") {
		if m := reHistList.FindStringSubmatch(line); len(m) > 1 {
			t.HistoryListLength = atoi64(m[1])
		}
		if reTrxActive.MatchString(line) {
			t.ActiveCount++
		}
		if m := rePurge.FindStringSubmatch(line); len(m) > 1 {
			t.PurgeLag = atoi64(m[1])
		}
	}
	return t
}

func parseBufferPool(text string) BufferPool {
	bp := BufferPool{}
	rePages := regexp.MustCompile(`(?i)^Database pages\s+(\d+)`)
	reFree := regexp.MustCompile(`(?i)Free buffers\s+(\d+)`)
	reDirty := regexp.MustCompile(`(?i)Modified db pages\s+(\d+)`)
	reTotal := regexp.MustCompile(`(?i)Buffer pool size\s+(\d+)`)
	reHitRate := regexp.MustCompile(`(?i)Buffer pool hit rate\s+(\d+)\s*/\s*(\d+)`)
	rePending := regexp.MustCompile(`(?i)Pending reads\s+(\d+)`)

	for _, line := range strings.Split(text, "\n") {
		if m := reTotal.FindStringSubmatch(line); len(m) > 1 {
			bp.TotalPages = atoi64(m[1])
		}
		if m := rePages.FindStringSubmatch(line); len(m) > 1 {
			bp.DataPages = atoi64(m[1])
		}
		if m := reFree.FindStringSubmatch(line); len(m) > 1 {
			bp.FreePages = atoi64(m[1])
		}
		if m := reDirty.FindStringSubmatch(line); len(m) > 1 {
			bp.DirtyPages = atoi64(m[1])
		}
		if m := reHitRate.FindStringSubmatch(line); len(m) > 2 {
			num := atoi64(m[1])
			denom := atoi64(m[2])
			if denom > 0 {
				bp.HitRate = float64(num) / float64(denom)
			}
		}
		if m := rePending.FindStringSubmatch(line); len(m) > 1 {
			bp.PendingReads = atoi64(m[1])
		}
	}
	return bp
}

func parseRowOps(text string) RowOps {
	r := RowOps{}
	re := regexp.MustCompile(`(?i)(\d+\.?\d*)\s+(reads|inserts|updates|deletes)/s`)
	for _, line := range strings.Split(text, "\n") {
		matches := re.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) < 3 {
				continue
			}
			val := atof64(m[1])
			switch strings.ToLower(m[2]) {
			case "reads":
				r.ReadsPerSec = val
			case "inserts":
				r.InsertsPerSec = val
			case "updates":
				r.UpdatesPerSec = val
			case "deletes":
				r.DeletesPerSec = val
			}
		}
	}
	return r
}

func parseRedoLog(text string) RedoLog {
	rl := RedoLog{}
	reLSN := regexp.MustCompile(`(?i)Log sequence number\s+(\d+)`)
	reCheckpoint := regexp.MustCompile(`(?i)Last checkpoint at\s+(\d+)`)
	reFlushed := regexp.MustCompile(`(?i)Log flushed up to\s+(\d+)`)

	for _, line := range strings.Split(text, "\n") {
		if m := reLSN.FindStringSubmatch(line); len(m) > 1 {
			rl.LSN = atoi64(m[1])
		}
		if m := reCheckpoint.FindStringSubmatch(line); len(m) > 1 {
			rl.CheckpointLSN = atoi64(m[1])
		}
		if m := reFlushed.FindStringSubmatch(line); len(m) > 1 {
			rl.Flushed = atoi64(m[1])
		}
	}

	if rl.LSN > 0 && rl.CheckpointLSN > 0 {
		rl.CheckpointAge = rl.LSN - rl.CheckpointLSN
	}

	return rl
}

func atoi64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func atof64(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
