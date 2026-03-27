package collector

import "testing"

func TestProcesslist_Name(t *testing.T) {
	p := NewProcesslist()
	if p.Name() != "processlist" {
		t.Fatalf("expected processlist, got %s", p.Name())
	}
}

func TestProcesslist_ImplementsCollector(t *testing.T) {
	var _ Collector = NewProcesslist()
}
