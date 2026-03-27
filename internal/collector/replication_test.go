package collector

import "testing"

func TestReplication_Name(t *testing.T) {
	r := NewReplication()
	if r.Name() != "replication" {
		t.Fatalf("expected replication, got %s", r.Name())
	}
}

func TestReplication_ImplementsCollector(t *testing.T) {
	var _ Collector = NewReplication()
}

func TestBoolToFloat(t *testing.T) {
	tests := []struct {
		input []string
		want  float64
	}{
		{[]string{"Yes"}, 1},
		{[]string{"No"}, 0},
		{[]string{"", "Yes"}, 1},
		{[]string{"", ""}, 0},
	}
	for _, tt := range tests {
		got := boolToFloat(tt.input...)
		if got != tt.want {
			t.Errorf("boolToFloat(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFloatFromMap(t *testing.T) {
	m := map[string]string{"A": "42", "B": "bad"}
	if v := floatFromMap(m, "A"); v != 42 {
		t.Fatalf("expected 42, got %v", v)
	}
	if v := floatFromMap(m, "B"); v != -1 {
		t.Fatalf("expected -1 for bad value, got %v", v)
	}
	if v := floatFromMap(m, "missing"); v != -1 {
		t.Fatalf("expected -1 for missing key, got %v", v)
	}
}
