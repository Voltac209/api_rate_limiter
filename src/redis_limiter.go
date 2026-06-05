package ratelimiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLimiter struct {
	client    *redis.Client
	algorithm string
}

func NewRedisLimiter(addr string, algorithm string) *RedisLimiter {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisLimiter{
		client:    client,
		algorithm: algorithm,
	}
}

func (l *RedisLimiter) Check(key string, limit uint32, windowSeconds uint32) Result {
	ctx := context.Background()
	nowMs := time.Now().UnixMilli()

	script := redisTokenBucketScript
	if l.algorithm == "rolling_window" {
		script = redisRollingWindowScript
	}

	res, err := script.Run(
		ctx,
		l.client,
		[]string{"rl:" + key},
		limit,
		windowSeconds,
		nowMs,
	).Result()

	if err != nil {
		return Result{Allowed: false, Remaining: 0, RetryAfterMs: 1000}
	}

	values, ok := res.([]interface{})
	if !ok || len(values) != 3 {
		return Result{Allowed: false, Remaining: 0, RetryAfterMs: 1000}
	}

	allowed := toInt64(values[0]) == 1
	remaining := uint32(toInt64(values[1]))
	retry := uint64(toInt64(values[2]))

	return Result{
		Allowed:      allowed,
		Remaining:    remaining,
		RetryAfterMs: retry,
	}
}

func toInt64(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		return n
	}
}

var redisTokenBucketScript = redis.NewScript(`
	-- KEYS[1] = bucket key
	-- ARGV[1] = limit
	-- ARGV[2] = window_seconds
	-- ARGV[3] = now_ms

	local key = KEYS[1]
	local limit = tonumber(ARGV[1])
	local window = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])

	local refill_rate = limit / window
	local ttl_ms = math.floor(window * 2000)

	local tokens = tonumber(redis.call("HGET", key, "tokens"))
	local last_refill = tonumber(redis.call("HGET", key, "last_refill"))
	local old_limit = tonumber(redis.call("HGET", key, "limit"))
	local old_window = tonumber(redis.call("HGET", key, "window"))

	if (not tokens) or (not last_refill) or old_limit ~= limit or old_window ~= window then
		tokens = limit
		last_refill = now
	end

	local elapsed = (now - last_refill) / 1000.0
	tokens = math.min(limit, tokens + elapsed * refill_rate)

	local allowed = 0
	local remaining = 0
	local retry_after_ms = 0

	if tokens >= 1 then
		tokens = tokens - 1
		allowed = 1
		remaining = math.floor(tokens)
	else
		retry_after_ms = math.ceil(((1 - tokens) / refill_rate) * 1000)
	end

	redis.call("HSET", key,
		"tokens", tokens,
		"last_refill", now,
		"limit", limit,
		"window", window
	)
	redis.call("PEXPIRE", key, ttl_ms)

	return {allowed, remaining, retry_after_ms}
`)

var redisRollingWindowScript = redis.NewScript(`
	-- KEYS[1] = sorted-set key
	-- ARGV[1] = limit
	-- ARGV[2] = window_seconds
	-- ARGV[3] = now_ms

	local key = KEYS[1]
	local limit = tonumber(ARGV[1])
	local window = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])
	local window_ms = window * 1000
	local cutoff = now - window_ms

	redis.call("ZREMRANGEBYSCORE", key, "-inf", cutoff)
	local current = redis.call("ZCARD", key)

	if current < limit then
		local member = tostring(now) .. "-" .. tostring(redis.call("INCR", key .. ":seq"))
		redis.call("ZADD", key, now, member)
		redis.call("PEXPIRE", key, window_ms)
		redis.call("PEXPIRE", key .. ":seq", window_ms)
		return {1, limit - current - 1, 0}
	end

	local oldest = redis.call("ZRANGE", key, 0, 0, "WITHSCORES")
	local retry_after_ms = window_ms
	if oldest[2] ~= nil then
		retry_after_ms = math.max(0, window_ms - (now - tonumber(oldest[2])))
	end

	redis.call("PEXPIRE", key, window_ms)
	redis.call("PEXPIRE", key .. ":seq", window_ms)
	return {0, 0, retry_after_ms}
`)
