package collector

import "testing"

func TestTableStats_Name(t *testing.T) {
	c := NewTableStats()
	if c.Name() != "tablestats" {
		t.Fatalf("expected tablestats, got %s", c.Name())
	}
}

func TestTableStats_ImplementsCollector(t *testing.T) {
	var _ Collector = NewTableStats()
}
