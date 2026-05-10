package ratelimiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLimiter struct {client *redis.Client}

func NewRedisLimiter(addr string) *RedisLimiter{
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisLimiter{client: client}
}

func toInt64(v interface{}) int64 {
	switch t :=v.(type) {
	case int64:
		return t
	case string:
		n, _ :=strconv.ParseInt(t,10,64)
		return n
	default: 
		n, _ := strconv.ParseInt(fmt.Sprint(v),10,64)
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

	local refill_rate = limit/window
	local ttl_ms=math.floor(window*2000)

	local tokens = tonumber(redis.call("HGET" , key , "tokens"))
	local last_refill = tonumber(redis.call("HGET", key ,"last_refill"))
	local old_limit = tonumber(redis.call("HGET" , key , "limit"))
	local old_window = tonumber(redis.call("HGET" , key , "window"))

	if (not tokens) or (not last_refill) or old_limit ~= limit or old_window ~= window then
		tokens=limit
		last_refill=now
	end

	local elapsed = (now-last_refill) / 1000.0
	tokens=math.min(limit,tokens+elapsed*refill_rate)

	local allowed=0
	local remaining=0
	local retry_after_ms=0

	if tokens>=1 then
		tokens=tokens-1
		allowed=1
		remaining=math.floor(tokens)
	else 
		allowed=0
		remaining=0
		retry_after_ms=math.ceil(((1-tokens)/refill_rate)*1000)
	end

	redis.call("HSET",key,
	"tokens" , tokens,
	"last_refill",now,
	"limit", limit,
	"window",window)
	redis.call("PEXPIRE",key,ttl_ms)
	
	return {allowed,remaining,retry_after_ms} 
`)

func (l* RedisLimiter) Check (key string, limit uint32, windowSeconds uint32) Result {
	ctx:=context.Background()
	nowMs := time.Now().UnixMilli()

	res,err :=redisTokenBucketScript.Run(
		ctx,
		l.client,
		[]string{"rl:"+key},
		limit,
		windowSeconds,
		nowMs,
	).Result()

	if err!=nil {
		return Result{Allowed: false, Remaining: 0, RetryAfterMs: 1000}
	}
	values , ok :=res.([]interface{})
	if !ok || len(values) !=3 {
		return Result{Allowed: false, Remaining: 0, RetryAfterMs: 1000}
	}

	allowed := toInt64(values[0]) == 1
	remaining := uint32(toInt64(values[1]))
	retry := uint64(toInt64(values[2]))

	return Result{
		Allowed: allowed,
		Remaining: remaining,
		RetryAfterMs: retry,
	}
}