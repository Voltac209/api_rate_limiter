package ratelimiter

import (
	"testing"
)

func TestRedisLimiter_AllowsThenDenies(t *testing.T){
	addr := "127.0.0.1:6379"

	l := NewRedisLimiter(addr)
	key := "redis_test_key"

	r1 := l.Check(key,2,10)
	r2 := l.Check(key , 2,10)
	r3 := l.Check(key,2,10)

	if !r1.Allowed || !r2.Allowed {
		t.Fatalf("expected first 2 requests allowed got r1=%+v , r2=%+v",r1,r2)
	}
	if r3.Allowed {
		t.Fatalf("expected third request denied, got %+v", r3)
		
	}
	if r3.RetryAfterMs ==0 {
		t.Fatalf("expected retry_after_ms >0 on deny")
	}

}