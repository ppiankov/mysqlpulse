package collector

import "testing"

func TestInnoDB_Name(t *testing.T) {
	i := NewInnoDB()
	if i.Name() != "innodb" {
		t.Fatalf("expected innodb, got %s", i.Name())
	}
}

func TestInnoDB_ImplementsCollector(t *testing.T) {
	var _ Collector = NewInnoDB()
}
