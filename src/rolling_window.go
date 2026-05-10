package ratelimiter

import (
	"sync"
	"time"
)

type RollingWindow struct {
	limit    int           // Stores the max allowed requests
	window   time.Duration // Stores the Time Window
	requests []time.Time   // Stores the timestamps of past requests
	mutex    sync.Mutex
}

func NewRollingWindow(limit int, window time.Duration) *RollingWindow {
	return &RollingWindow{
		limit:  limit,
		window: window,
	}
}

func (rw *RollingWindow) Allow() bool {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()
	now := time.Now()             // current request time
	cutoff := now.Add(-rw.window) //oldest allowed timestamp
	valid := rw.requests[:0]      //empties the slice without freeing the memory

	// this part removes the timestamps older than cutoff

	for _, req_time := range rw.requests {
		if req_time.After(cutoff) {
			valid = append(valid, req_time)
		}
	}
	rw.requests = valid
	if len(rw.requests) < rw.limit {
		rw.requests = append(rw.requests, now)
		return true
	}
	return false
}
