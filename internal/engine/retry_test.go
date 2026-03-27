package engine

import (
	"context"
	"errors"
	"testing"
)

type fakePinger struct {
	calls    int
	failFor  int // fail this many times before succeeding
	failWith error
}

func (f *fakePinger) PingContext(ctx context.Context) error {
	f.calls++
	if f.calls <= f.failFor {
		return f.failWith
	}
	return nil
}

func TestPingWithRetry_SuccessFirst(t *testing.T) {
	p := &fakePinger{}
	err := PingWithRetry(context.Background(), p, 3)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if p.calls != 1 {
		t.Fatalf("expected 1 call, got %d", p.calls)
	}
}

func TestPingWithRetry_SuccessAfterRetries(t *testing.T) {
	p := &fakePinger{failFor: 2, failWith: errors.New("connection refused")}
	err := PingWithRetry(context.Background(), p, 3)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if p.calls != 3 {
		t.Fatalf("expected 3 calls, got %d", p.calls)
	}
}

func TestPingWithRetry_Exhausted(t *testing.T) {
	want := errors.New("connection refused")
	p := &fakePinger{failFor: 10, failWith: want}
	err := PingWithRetry(context.Background(), p, 3)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// 1 initial + 3 retries = 4 attempts
	if p.calls != 4 {
		t.Fatalf("expected 4 calls, got %d", p.calls)
	}
}

func TestPingWithRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := &fakePinger{failFor: 10, failWith: errors.New("fail")}
	err := PingWithRetry(ctx, p, 3)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if p.calls != 0 {
		t.Fatalf("expected 0 calls on cancelled ctx, got %d", p.calls)
	}
}
