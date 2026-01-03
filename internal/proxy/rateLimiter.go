package proxy

import (
	"net"
	"sync"
	"time"

	"database_firewall/internal/config"
)

type RateLimiter interface {
	Allow(net.IP) bool
}

var _ RateLimiter = (*TokenBucketLimiter)(nil)

type TokenBucketLimiter struct {
	mu       sync.Mutex
	cfg      config.RateLimiterConfig
	rate     int64
	capacity int64
	buckets  map[string]bucket
}

type bucket struct {
	tokens     int64
	lastRefill time.Time
}

func NewTokenBucketLimiter(cfg *config.RateLimiterConfig) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		cfg:      *cfg,
		rate:     cfg.RateLimiter.TokenBucketLimiter.Rate,
		capacity: cfg.RateLimiter.TokenBucketLimiter.Capacity,
		buckets:  make(map[string]bucket),
	}
}

func (t *TokenBucketLimiter) Allow(ip net.IP) bool {
	if t.rate == 0 {
		return true
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	b, ok := t.buckets[ip.String()]
	if !ok {
		b = bucket{
			tokens:     t.capacity,
			lastRefill: now,
		}
	}

	//-------------refill--------------
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		refill := int64(elapsed * float64(t.rate))
		b.tokens += refill
		if b.tokens > t.capacity {
			b.tokens = t.capacity
		}
		b.lastRefill = now
	}

	//-------------allow/deny------------
	if b.tokens <= 0 {
		t.buckets[ip.String()] = b
		return false
	}

	b.tokens -= 1
	t.buckets[ip.String()] = b
	return true
}
