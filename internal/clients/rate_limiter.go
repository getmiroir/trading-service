package clients

import (
	"sync"
	"time"

	"github.com/getmiroir/trading-service/internal"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	limits  []RateLimit
	mu      sync.Mutex
	buckets map[string]*tokenBucket
}

// RateLimit represents a rate limit configuration
type RateLimit struct {
	Requests int // Number of requests allowed
	Seconds  int // Time window in seconds
}

// tokenBucket represents a token bucket for rate limiting
type tokenBucket struct {
	capacity   int
	tokens     int
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the given limits
func NewRateLimiter(limits []RateLimit) *RateLimiter {
	return &RateLimiter{
		limits:  limits,
		buckets: make(map[string]*tokenBucket),
	}
}

// TryAcquire attempts to acquire a token for the given key
// Returns an error if the rate limit is exceeded
func (rl *RateLimiter) TryAcquire(key string) error {
	rl.mu.Lock()
	bucket, exists := rl.buckets[key]
	if !exists {
		// Create buckets for this key based on limits
		bucket = rl.createBucket()
		rl.buckets[key] = bucket
	}
	rl.mu.Unlock()

	return bucket.tryAcquire()
}

// createBucket creates a token bucket based on the rate limits
// Uses the most restrictive limit (lowest requests per second)
func (rl *RateLimiter) createBucket() *tokenBucket {
	if len(rl.limits) == 0 {
		// No limits, allow unlimited requests
		return &tokenBucket{
			capacity:   1000000,
			tokens:     1000000,
			refillRate: 1000000,
			lastRefill: time.Now(),
		}
	}

	// Find the most restrictive limit (lowest requests per second)
	minRequestsPerSecond := float64(rl.limits[0].Requests) / float64(rl.limits[0].Seconds)
	capacity := rl.limits[0].Requests

	for _, limit := range rl.limits {
		requestsPerSecond := float64(limit.Requests) / float64(limit.Seconds)
		if requestsPerSecond < minRequestsPerSecond {
			minRequestsPerSecond = requestsPerSecond
			capacity = limit.Requests
		}
	}

	return &tokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: minRequestsPerSecond,
		lastRefill: time.Now(),
	}
}

// tryAcquire attempts to acquire a token from the bucket
func (tb *tokenBucket) tryAcquire() error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed * tb.refillRate)
	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}

	// Check if we have tokens available
	if tb.tokens <= 0 {
		return &internal.LimiterException{
			Message: "rate limit exceeded",
		}
	}

	// Consume a token
	tb.tokens--
	return nil
}
