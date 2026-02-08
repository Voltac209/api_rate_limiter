package ratelimiter

type Result struct {
	Allowed      bool
	Remaining    uint32
	RetryAfterMs uint64
}

type Limiter interface {
	Check(key string, limit uint32, windowSeconds uint32) Result
}
