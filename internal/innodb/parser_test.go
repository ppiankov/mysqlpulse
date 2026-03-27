package innodb

import (
	"testing"
)

const sampleInnoDBStatus = `
=====================================
2026-03-27 12:00:00 0x7f123456 INNODB MONITOR OUTPUT
=====================================
Per second averages calculated from the last 30 seconds
-----------------
BACKGROUND THREAD
-----------------
srv_master_thread loops: 100 srv_active, 0 srv_shutdown, 500 srv_idle
srv_master_thread log flush and target: 600 log_writes
----------
SEMAPHORES
----------
OS WAIT ARRAY INFO: reservation count 42
Mutex spin waits 100, rounds 200, OS waits 50
RW-shared spins 30, rounds 60, OS waits 10
RW-excl spins 20, rounds 40, OS waits 5
Spin rounds per wait: 2.00 mutex, 2.00 RW-shared, 2.00 RW-excl
------------------------
LATEST DETECTED DEADLOCK
------------------------
*** (1) TRANSACTION:
TRANSACTION 12345, ACTIVE 5 sec starting index read
mysql tables in use 1, locked 1
LOCK WAIT 3 lock struct(s), heap size 1136, 2 row lock(s)
MySQL thread id 100, OS thread handle 123456, query id 999
UPDATE orders SET status = 'shipped' WHERE id = 42
*** (1) WAITING FOR THIS LOCK TO BE GRANTED:
RECORD LOCKS space id 100 page no 3 n bits 72 index PRIMARY of table orders
lock mode X waiting
*** (2) TRANSACTION:
TRANSACTION 12346, ACTIVE 3 sec starting index read
mysql tables in use 1, locked 1
3 lock struct(s), heap size 1136, 2 row lock(s)
MySQL thread id 101, OS thread handle 123457, query id 1000
UPDATE inventory SET qty = qty - 1 WHERE product_id = 7
*** (2) HOLDS THE LOCK:
RECORD LOCKS space id 100 page no 3 n bits 72 index PRIMARY of table orders
lock mode X
*** WE ROLL BACK TRANSACTION (1)
------------
TRANSACTIONS
------------
Trx id counter 12400
Purge done for trx's n:o < 12390 undo n:o < 0
History list length 150
LIST OF TRANSACTIONS FOR EACH SESSION:
---TRANSACTION 12345, ACTIVE 10 sec
2 lock struct(s)
---TRANSACTION 12346, ACTIVE 5 sec
1 lock struct(s)
--------
FILE I/O
--------
I/O thread 0 state: waiting for i/o request
-------------------------------------
INSERT BUFFER AND ADAPTIVE HASH INDEX
-------------------------------------
Ibuf: size 1, free list len 0
---
LOG
---
Log sequence number 10000000
Log flushed up to   9999000
Pages flushed up to 9998000
Last checkpoint at  9990000
0 pending log flushes, 0 pending chkp writes
----------------------
BUFFER POOL AND MEMORY
----------------------
Total large memory allocated 0
Dictionary memory allocated 300000
Buffer pool size   8192
Free buffers       4096
Database pages     4000
Old database pages 1476
Modified db pages  100
Pending reads      5
Buffer pool hit rate 999 / 1000, young-making rate 0 / 1000
--------------
ROW OPERATIONS
--------------
0 queries inside InnoDB, 0 queries in queue
2 read views open inside InnoDB
Process ID=12345, Main thread ID=0x7f123456, state=sleeping
Number of rows inserted 1000, updated 500, deleted 200, read 50000
100.50 inserts/s, 50.25 updates/s, 20.10 deletes/s, 5000.00 reads/s
----------------------------
END OF INNODB MONITOR OUTPUT
============================
`

func TestParse_BufferPool(t *testing.T) {
	s := Parse(sampleInnoDBStatus)

	if s.BufferPool.TotalPages != 8192 {
		t.Errorf("TotalPages = %d, want 8192", s.BufferPool.TotalPages)
	}
	if s.BufferPool.FreePages != 4096 {
		t.Errorf("FreePages = %d, want 4096", s.BufferPool.FreePages)
	}
	if s.BufferPool.DataPages != 4000 {
		t.Errorf("DataPages = %d, want 4000", s.BufferPool.DataPages)
	}
	if s.BufferPool.DirtyPages != 100 {
		t.Errorf("DirtyPages = %d, want 100", s.BufferPool.DirtyPages)
	}
	if s.BufferPool.PendingReads != 5 {
		t.Errorf("PendingReads = %d, want 5", s.BufferPool.PendingReads)
	}
	// 999/1000 = 0.999
	if s.BufferPool.HitRate < 0.998 || s.BufferPool.HitRate > 1.0 {
		t.Errorf("HitRate = %f, want ~0.999", s.BufferPool.HitRate)
	}
}

func TestParse_RedoLog(t *testing.T) {
	s := Parse(sampleInnoDBStatus)

	if s.RedoLog.LSN != 10000000 {
		t.Errorf("LSN = %d, want 10000000", s.RedoLog.LSN)
	}
	if s.RedoLog.CheckpointLSN != 9990000 {
		t.Errorf("CheckpointLSN = %d, want 9990000", s.RedoLog.CheckpointLSN)
	}
	if s.RedoLog.CheckpointAge != 10000 {
		t.Errorf("CheckpointAge = %d, want 10000", s.RedoLog.CheckpointAge)
	}
	if s.RedoLog.Flushed != 9999000 {
		t.Errorf("Flushed = %d, want 9999000", s.RedoLog.Flushed)
	}
}

func TestParse_Transactions(t *testing.T) {
	s := Parse(sampleInnoDBStatus)

	if s.Transactions.HistoryListLength != 150 {
		t.Errorf("HistoryListLength = %d, want 150", s.Transactions.HistoryListLength)
	}
	if s.Transactions.ActiveCount != 2 {
		t.Errorf("ActiveCount = %d, want 2", s.Transactions.ActiveCount)
	}
}

func TestParse_Semaphores(t *testing.T) {
	s := Parse(sampleInnoDBStatus)

	if s.Semaphores.Spins != 100 {
		t.Errorf("Spins = %d, want 100", s.Semaphores.Spins)
	}
	if s.Semaphores.Rounds != 200 {
		t.Errorf("Rounds = %d, want 200", s.Semaphores.Rounds)
	}
	if s.Semaphores.OSWaits != 50 {
		t.Errorf("OSWaits = %d, want 50", s.Semaphores.OSWaits)
	}
	if s.Semaphores.RWSharedSpins != 30 {
		t.Errorf("RWSharedSpins = %d, want 30", s.Semaphores.RWSharedSpins)
	}
	if s.Semaphores.RWExclSpins != 20 {
		t.Errorf("RWExclSpins = %d, want 20", s.Semaphores.RWExclSpins)
	}
}

func TestParse_RowOps(t *testing.T) {
	s := Parse(sampleInnoDBStatus)

	if s.RowOps.ReadsPerSec != 5000.00 {
		t.Errorf("ReadsPerSec = %f, want 5000.00", s.RowOps.ReadsPerSec)
	}
	if s.RowOps.InsertsPerSec != 100.50 {
		t.Errorf("InsertsPerSec = %f, want 100.50", s.RowOps.InsertsPerSec)
	}
	if s.RowOps.UpdatesPerSec != 50.25 {
		t.Errorf("UpdatesPerSec = %f, want 50.25", s.RowOps.UpdatesPerSec)
	}
	if s.RowOps.DeletesPerSec != 20.10 {
		t.Errorf("DeletesPerSec = %f, want 20.10", s.RowOps.DeletesPerSec)
	}
}

func TestParse_Deadlocks(t *testing.T) {
	s := Parse(sampleInnoDBStatus)

	if len(s.Deadlocks) != 1 {
		t.Fatalf("Deadlocks count = %d, want 1", len(s.Deadlocks))
	}
	dl := s.Deadlocks[0]
	if dl.Victim != "transaction_1" {
		t.Errorf("Victim = %s, want transaction_1", dl.Victim)
	}
	if !dl.Transaction1.Waiting {
		t.Error("Transaction1 should be waiting")
	}
}

func TestParse_EmptyInput(t *testing.T) {
	s := Parse("")
	// Should not panic, all fields zero.
	if s.BufferPool.TotalPages != 0 {
		t.Error("expected zero values for empty input")
	}
}

func TestSplitSections(t *testing.T) {
	sections := splitSections(sampleInnoDBStatus)
	expected := []string{"SEMAPHORES", "LATEST DETECTED DEADLOCK", "TRANSACTIONS", "LOG", "BUFFER POOL AND MEMORY", "ROW OPERATIONS"}
	for _, name := range expected {
		if _, ok := sections[name]; !ok {
			t.Errorf("missing section: %s", name)
		}
	}
}
