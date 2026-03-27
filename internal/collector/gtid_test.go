package collector

import "testing"

func TestGTID_Name(t *testing.T) {
	c := NewGTID()
	if c.Name() != "gtid" {
		t.Fatalf("expected gtid, got %s", c.Name())
	}
}

func TestGTID_ImplementsCollector(t *testing.T) {
	var _ Collector = NewGTID()
}

func TestCountGTIDSets(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"3E11FA47-71CA-11E1-9E33-C80AA9429562:1-5", 1},
		{"3E11FA47-71CA-11E1-9E33-C80AA9429562:1-5,4B11FA47-71CA-11E1-9E33-C80AA9429562:1-3", 2},
		{"3E11FA47-71CA-11E1-9E33-C80AA9429562:1-5, 3E11FA47-71CA-11E1-9E33-C80AA9429562:7-10", 1},
	}
	for _, tt := range tests {
		got := countGTIDSets(tt.input)
		if got != tt.want {
			t.Errorf("countGTIDSets(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
