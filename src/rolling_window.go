package ratelimiter

import (
	"sync"
	"time"
)

type rollingWindow struct {
	limit    uint32
	window   time.Duration
	requests []time.Time
	mutex    sync.Mutex
}

type RollingWindowLimiter struct {
	windows map[string]*rollingWindow
	mu      sync.Mutex
}

func NewRollingWindowLimiter() *RollingWindowLimiter {
	return &RollingWindowLimiter{
		windows: make(map[string]*rollingWindow),
	}
}

func (rw *rollingWindow) allow() Result {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rw.window)
	valid := rw.requests[:0]

	for _, requestTime := range rw.requests {
		if requestTime.After(cutoff) {
			valid = append(valid, requestTime)
		}
	}
	rw.requests = valid

	if uint32(len(rw.requests)) < rw.limit {
		rw.requests = append(rw.requests, now)
		return Result{
			Allowed:      true,
			Remaining:    rw.limit - uint32(len(rw.requests)),
			RetryAfterMs: 0,
		}
	}

	retryAfter := rw.requests[0].Add(rw.window).Sub(now)
	if retryAfter < 0 {
		retryAfter = 0
	}

	return Result{
		Allowed:      false,
		Remaining:    0,
		RetryAfterMs: uint64(retryAfter.Milliseconds()),
	}
}

func (l *RollingWindowLimiter) Check(
	key string,
	limit uint32,
	windowSeconds uint32,
) Result {
	window := l.getWindow(key, limit, windowSeconds)
	return window.allow()
}

func (l *RollingWindowLimiter) getWindow(
	key string,
	limit uint32,
	windowSeconds uint32,
) *rollingWindow {
	l.mu.Lock()
	defer l.mu.Unlock()

	windowDuration := time.Duration(windowSeconds) * time.Second
	if window, ok := l.windows[key]; ok {
		if window.limit != limit || window.window != windowDuration {
			window.limit = limit
			window.window = windowDuration
			window.requests = nil
		}
		return window
	}

	window := &rollingWindow{
		limit:  limit,
		window: windowDuration,
	}
	l.windows[key] = window
	return window
}
