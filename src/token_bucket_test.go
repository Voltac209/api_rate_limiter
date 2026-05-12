package ratelimiter

import (
	"testing"
	"time"
)

func TestTokenBucket_AllowsUpToLimitThenDenies(t *testing.T){
	l := NewTokenBucketLimiter()
	key := "tb-limit"

	for i:=0 ; i<3 ; i++ {
		res := l.Check(key,3,10)
		if !res.Allowed {
			t.Fatalf("expected allowed at request %d, got denied",i+1)
		}
	}
	res := l.Check(key,3,10)
	if res.Allowed {
		t.Fatalf("expected deny after limit exhausted")
	}
	if res.RetryAfterMs==0 {
		t.Fatalf("expected RetryAfterMs to be > 0 after deny")
	}
}

func TestTokenBucket_RefillsAfterWindow(t *testing.T){
	l :=NewTokenBucketLimiter()
	key := "tb-refill"

	first :=l.Check(key,1,1)
	if !first.Allowed {
		t.Fatalf("expected first request to be allowed")
	}

	second :=l.Check(key,1,1)
	if second.Allowed {
		t.Fatalf("expected second request to be denied immediately")
	}

	time.Sleep(1000*time.Millisecond)

	third:=l.Check(key,1,1)
	if !third.Allowed {
		t.Fatalf("expected request to be allowed after refill period")
	}
}