package ratelimit

import (
	"testing"
	"time"
)

func TestBucketRefill(t *testing.T) {
	l := New(60, 2) // 1 token/sec, burst 2
	clock := time.Unix(1000, 0)
	l.now = func() time.Time { return clock }
	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("burst should allow two")
	}
	if l.Allow("a") {
		t.Fatal("third request should be limited")
	}
	if !l.Allow("b") {
		t.Fatal("keys are independent")
	}
	clock = clock.Add(1500 * time.Millisecond)
	if !l.Allow("a") {
		t.Fatal("token should have refilled")
	}
	l.Sweep(time.Second)
	clock = clock.Add(time.Hour)
	l.Sweep(time.Second)
	if len(l.buckets) != 0 {
		t.Fatalf("expected eviction, have %d", len(l.buckets))
	}
}
