package ratelimiter

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedisTokenBucket_AllowsThenDenies(t *testing.T) {
	runRedisLimiterTest(t, "token_bucket", "redis_token_bucket_test")
}

func TestRedisRollingWindow_AllowsThenDenies(t *testing.T) {
	runRedisLimiterTest(t, "rolling_window", "redis_rolling_window_test")
}

func runRedisLimiterTest(t *testing.T, algorithm string, key string) {
	t.Helper()

	addr := "127.0.0.1:6379"
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable at %s: %v", addr, err)
	}

	limiter := NewRedisLimiter(addr, algorithm)

	if err := client.Del(ctx, "rl:"+key, "rl:"+key+":seq").Err(); err != nil {
		t.Fatalf("failed to clear redis key: %v", err)
	}

	r1 := limiter.Check(key, 2, 10)
	r2 := limiter.Check(key, 2, 10)
	r3 := limiter.Check(key, 2, 10)

	if !r1.Allowed || !r2.Allowed {
		t.Fatalf("expected first 2 requests allowed got r1=%+v , r2=%+v", r1, r2)
	}
	if r3.Allowed {
		t.Fatalf("expected third request denied, got %+v", r3)
	}
	if r3.RetryAfterMs == 0 {
		t.Fatalf("expected retry_after_ms >0 on deny")
	}
}
