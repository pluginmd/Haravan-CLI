package client

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Haravan's leaky-bucket rate limit: 80-request bucket, 4 req/s leak.
// We start backing off at 70 to leave headroom for concurrent callers.
const (
	bucketSize     = 80
	leakRatePerSec = 4.0
	safeThreshold  = 70
)

// bucket tracks the most recent authoritative usage and decays it over time.
type bucket struct {
	mu          sync.Mutex
	knownUsage  float64
	knownMax    float64
	lastUpdated time.Time
}

func newBucket() *bucket {
	return &bucket{knownMax: bucketSize, lastUpdated: time.Now()}
}

// estimate projects current bucket fill based on the leak rate since the
// last authoritative update from a response header.
func (b *bucket) estimate(now time.Time) float64 {
	elapsed := now.Sub(b.lastUpdated).Seconds()
	est := b.knownUsage - elapsed*leakRatePerSec
	if est < 0 {
		return 0
	}
	return est
}

// update writes a fresh usage reading taken from a response header.
func (b *bucket) update(used, max float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.knownUsage = used
	if max > 0 {
		b.knownMax = max
	}
	b.lastUpdated = time.Now()
}

// waitIfFull blocks until the bucket is expected to have headroom.
// Honours ctx cancellation.
func (b *bucket) waitIfFull(ctx context.Context) error {
	b.mu.Lock()
	est := b.estimate(time.Now())
	b.mu.Unlock()

	if est < safeThreshold {
		return nil
	}
	over := est - safeThreshold
	waitDur := time.Duration(over/leakRatePerSec*float64(time.Second)) + 50*time.Millisecond
	t := time.NewTimer(waitDur)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// parseLimitHeader parses "used/max" from x-haravan-api-call-limit.
func parseLimitHeader(raw string) (used, max float64, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(raw), "/", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	u, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, false
	}
	m, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, false
	}
	return u, m, true
}

func parseRetryAfter(raw string) float64 {
	if raw == "" {
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0
	}
	return f
}
