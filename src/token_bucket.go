package src

import (
	"sync"
	"time"
)

// sync imported to get concurrency tools

//In this scenarion Mutex used to handle multiple server requests at once

var mu sync.Mutex

type TBucket struct {
	max_tokens     float64    //stores maximum number of tokens
	current_tokens float64    //stores current number of tokens
	refill_rate    float64    //rate at which tokens added
	lastRefill     time.Time  //stores the time at which last refill
	bucket_mutex   sync.Mutex //Ensures only request modifies our bucket.
}

// Returns a pointer to Token Bucket Struct
func newTokenBucket(capacity, refillRate float64) *TBucket {
	return &TBucket{
		max_tokens:     capacity,
		current_tokens: capacity,
		refill_rate:    refillRate,
		lastRefill:     time.Now(),
	}
}

//Here refill is a method belonging to the struct TBucket

func (tb *TBucket) refill() {
	now := time.Now()
	time_elapsed := now.Sub(tb.lastRefill).Seconds()

	tb.current_tokens += tb.refill_rate * time_elapsed

	if tb.current_tokens > tb.max_tokens {
		tb.current_tokens = tb.max_tokens
	}
	tb.lastRefill = now
}

//Here allow method checks wether request can be allowed based on tokens remaining in bucket
//Inside the allow method bucket_mutex added to preserve thread safety
//defer here makes sure the mutex lock is always released.

func (tb *TBucket) Allow() bool {
	tb.bucket_mutex.Lock()
	defer tb.bucket_mutex.Unlock()

	//Tokens refilled before checking for addition just in case refill time occurs between requests

	tb.refill()
	if tb.current_tokens >= 1 {
		tb.current_tokens--
		return true
	}
	return false
}

func main() {

}
