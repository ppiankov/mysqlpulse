package engine

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ppiankov/mysqlpulse/internal/metrics"
)

func TestEngine_RunRespectsContext(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	eng := New(20*time.Millisecond, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	err := eng.Run(ctx)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestEngine_ImmediateFirstPoll(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics.Register(reg)

	// Engine with no targets — verifies loop mechanics without DB
	eng := New(10*time.Second, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := eng.Run(ctx)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
