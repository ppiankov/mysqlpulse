package collector

import "testing"

func TestPerfSchema_Name(t *testing.T) {
	c := NewPerfSchema()
	if c.Name() != "perfschema" {
		t.Fatalf("expected perfschema, got %s", c.Name())
	}
}

func TestPerfSchema_ImplementsCollector(t *testing.T) {
	var _ Collector = NewPerfSchema()
}
