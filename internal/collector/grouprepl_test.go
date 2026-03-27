package collector

import "testing"

func TestGroupReplication_Name(t *testing.T) {
	c := NewGroupReplication()
	if c.Name() != "grouprepl" {
		t.Fatalf("expected grouprepl, got %s", c.Name())
	}
}

func TestGroupReplication_ImplementsCollector(t *testing.T) {
	var _ Collector = NewGroupReplication()
}
