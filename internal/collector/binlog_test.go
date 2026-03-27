package collector

import "testing"

func TestBinlog_Name(t *testing.T) {
	c := NewBinlog()
	if c.Name() != "binlog" {
		t.Fatalf("expected binlog, got %s", c.Name())
	}
}

func TestBinlog_ImplementsCollector(t *testing.T) {
	var _ Collector = NewBinlog()
}
