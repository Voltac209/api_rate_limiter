package ratelimiter

import (
	"testing"
	"time"
)

func TestRollingWindow_AllowsUpToLimitThenDenies(t *testing.T) {
	l := NewRollingWindowLimiter()
	key := "rw-limit"

	for i := 0; i < 3; i++ {
		res := l.Check(key, 3, 1)
		if !res.Allowed {
			t.Fatalf("expected allowed at request %d, got denied", i+1)
		}
	}

	res := l.Check(key, 3, 1)
	if res.Allowed {
		t.Fatalf("expected deny after limit exhausted")
	}
	if res.RetryAfterMs == 0 {
		t.Fatalf("expected retry_after_ms > 0 on deny")
	}
}

func TestRollingWindow_AllowsAfterWindowExpires(t *testing.T) {
	l := NewRollingWindowLimiter()
	key := "rw-refill"

	first := l.Check(key, 1, 1)
	if !first.Allowed {
		t.Fatalf("expected first request allowed")
	}

	second := l.Check(key, 1, 1)
	if second.Allowed {
		t.Fatalf("expected second request denied")
	}

	time.Sleep(1100 * time.Millisecond)

	third := l.Check(key, 1, 1)
	if !third.Allowed {
		t.Fatalf("expected request allowed after window expiration")
	}
}
