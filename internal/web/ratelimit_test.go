package web

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsUpToLimit(t *testing.T) {
	l := newRateLimiter(3, time.Minute)
	for i := range 3 {
		if !l.allow("1.2.3.4") {
			t.Errorf("allow() call %d = false, want true (within limit)", i+1)
		}
	}
}

func TestRateLimiterBlocksOverLimit(t *testing.T) {
	l := newRateLimiter(3, time.Minute)
	for range 3 {
		l.allow("1.2.3.4")
	}
	if l.allow("1.2.3.4") {
		t.Error("allow() call 4 = true, want false (over limit)")
	}
}

func TestRateLimiterIsPerKey(t *testing.T) {
	l := newRateLimiter(1, time.Minute)
	if !l.allow("1.2.3.4") {
		t.Fatal("first allow() for key A = false, want true")
	}
	if !l.allow("5.6.7.8") {
		t.Error("first allow() for key B = false, want true (separate key, separate budget)")
	}
	if l.allow("1.2.3.4") {
		t.Error("second allow() for key A = true, want false (over limit)")
	}
}

func TestRateLimiterWindowExpiry(t *testing.T) {
	l := newRateLimiter(1, 10*time.Millisecond)
	if !l.allow("1.2.3.4") {
		t.Fatal("first allow() = false, want true")
	}
	if l.allow("1.2.3.4") {
		t.Fatal("second allow() within window = true, want false")
	}
	time.Sleep(20 * time.Millisecond)
	if !l.allow("1.2.3.4") {
		t.Error("allow() after window expiry = false, want true")
	}
}
