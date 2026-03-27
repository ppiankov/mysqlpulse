package collector

import "testing"

func TestQueries_Name(t *testing.T) {
	q := NewQueries()
	if q.Name() != "queries" {
		t.Fatalf("expected queries, got %s", q.Name())
	}
}

func TestQueries_ImplementsCollector(t *testing.T) {
	var _ Collector = NewQueries()
}
