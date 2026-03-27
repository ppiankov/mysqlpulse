package alerter

import (
	"testing"
	"time"
)

func TestMaskDSN(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"root@tcp(localhost:3306)/", "r***t@tcp(localhost:3306)/"},
		{"admin:secret123@tcp(db:3306)/mydb", "a***n:s***3@tcp(db:3306)/mydb"},
		{"u:p@tcp(host:3306)/", "***:***@tcp(host:3306)/"},
		{"tcp(host:3306)/", "tcp(host:3306)/"},
	}
	for _, tt := range tests {
		got := MaskDSN(tt.input)
		if got != tt.want {
			t.Errorf("MaskDSN(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHostFromDSN(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"root@tcp(localhost:3306)/", "localhost:3306"},
		{"admin:pass@tcp(db.example.com:3306)/mydb", "db.example.com:3306"},
		{"root@localhost/", "unknown"},
	}
	for _, tt := range tests {
		got := HostFromDSN(tt.input)
		if got != tt.want {
			t.Errorf("HostFromDSN(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskMiddle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"admin", "a***n"},
		{"ab", "***"},
		{"a", "***"},
		{"secret123", "s***3"},
	}
	for _, tt := range tests {
		got := maskMiddle(tt.input)
		if got != tt.want {
			t.Errorf("maskMiddle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAlerterCooldown(t *testing.T) {
	a := New(Config{
		WebhookURL: "http://localhost:9999/noop",
		Cooldown:   1 * time.Hour,
	})

	alert := Alert{
		Type:     AlertDeadlocks,
		Message:  "test",
		Instance: "test:3306",
		Host:     "test",
	}

	// First send should record the time.
	a.mu.Lock()
	a.lastSent[string(alert.Type)+":"+alert.Instance] = time.Now()
	a.mu.Unlock()

	// Second send within cooldown should be suppressed (no HTTP call).
	// We just verify no panic occurs.
	a.Send(alert)
}

func TestNewReturnsNilWhenNoChannels(t *testing.T) {
	a := New(Config{})
	if a != nil {
		t.Error("expected nil alerter when no channels configured")
	}
}
