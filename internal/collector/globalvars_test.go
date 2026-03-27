package collector

import "testing"

func TestGlobalVars_Name(t *testing.T) {
	c := NewGlobalVars()
	if c.Name() != "globalvars" {
		t.Fatalf("expected globalvars, got %s", c.Name())
	}
}

func TestGlobalVars_ImplementsCollector(t *testing.T) {
	var _ Collector = NewGlobalVars()
}

func TestOnOffToFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"ON", 1},
		{"on", 1},
		{"OFF", 0},
		{"off", 0},
		{"", 0},
	}
	for _, tt := range tests {
		got := onOffToFloat(tt.input)
		if got != tt.want {
			t.Errorf("onOffToFloat(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
