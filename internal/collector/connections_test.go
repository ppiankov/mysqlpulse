package collector

import (
	"testing"
)

func TestConnections_Name(t *testing.T) {
	c := NewConnections()
	if c.Name() != "connections" {
		t.Fatalf("expected connections, got %s", c.Name())
	}
}

func TestConnections_ImplementsCollector(t *testing.T) {
	var _ Collector = NewConnections()
}
