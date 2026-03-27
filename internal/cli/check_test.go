package cli

import (
	"testing"
)

func TestEscalate(t *testing.T) {
	tests := []struct {
		name        string
		initial     int
		code        int
		wantExit    int
		wantVerdict string
	}{
		{"upgrade from OK to WARNING", exitOK, exitWarning, exitWarning, "WARNING"},
		{"upgrade from WARNING to CRITICAL", exitWarning, exitCritical, exitCritical, "CRITICAL"},
		{"no downgrade from CRITICAL to WARNING", exitCritical, exitWarning, exitCritical, "CRITICAL"},
		{"no downgrade from CRITICAL to OK", exitCritical, exitOK, exitCritical, "CRITICAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CheckResult{Exit: tt.initial, Verdict: "PREV"}
			escalate(r, tt.code, tt.wantVerdict)
			if r.Exit != tt.wantExit {
				t.Errorf("exit = %d, want %d", r.Exit, tt.wantExit)
			}
		})
	}
}

func TestAvailableMetrics(t *testing.T) {
	metrics := availableMetrics()
	if len(metrics) == 0 {
		t.Fatal("expected at least one metric")
	}

	want := map[string]bool{
		"repl-lag":        false,
		"connections":     false,
		"threads-running": false,
		"buffer-pool":     false,
		"slow-queries":    false,
		"deadlocks":       false,
	}

	for _, m := range metrics {
		if _, ok := want[m]; ok {
			want[m] = true
		}
	}

	for k, found := range want {
		if !found {
			t.Errorf("missing metric: %s", k)
		}
	}
}

func TestCheckTable(t *testing.T) {
	r := CheckResult{
		Metric:  "connections",
		Verdict: "WARNING",
		Exit:    exitWarning,
		Nodes: []NodeResult{
			{Instance: "db1:3306", Value: 50, Warn: 40, Crit: 80, Status: "WARNING"},
			{Instance: "db2:3306", Value: 10, Warn: 40, Crit: 80, Status: "OK"},
		},
	}

	table := checkTable(r, "connections")
	if len(table.Headers) != 5 {
		t.Fatalf("expected 5 headers, got %d", len(table.Headers))
	}
	// nodes + summary row
	if len(table.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(table.Rows))
	}
}
