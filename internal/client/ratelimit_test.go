package client

import (
	"context"
	"testing"
	"time"
)

func TestParseLimitHeader(t *testing.T) {
	tests := []struct {
		in          string
		wantUsed    float64
		wantMax     float64
		wantOK      bool
	}{
		{"2/80", 2, 80, true},
		{" 60 / 80 ", 60, 80, true},
		{"", 0, 0, false},
		{"garbage", 0, 0, false},
		{"2/", 0, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			u, m, ok := parseLimitHeader(tc.in)
			if ok != tc.wantOK || u != tc.wantUsed || m != tc.wantMax {
				t.Errorf("parseLimitHeader(%q) = (%v,%v,%v), want (%v,%v,%v)",
					tc.in, u, m, ok, tc.wantUsed, tc.wantMax, tc.wantOK)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	if got := parseRetryAfter(""); got != 0 {
		t.Errorf("empty: got %v, want 0", got)
	}
	if got := parseRetryAfter("2.5"); got != 2.5 {
		t.Errorf("2.5: got %v, want 2.5", got)
	}
	if got := parseRetryAfter("not-a-number"); got != 0 {
		t.Errorf("garbage: got %v, want 0", got)
	}
}

func TestBucketDecay(t *testing.T) {
	b := newBucket()
	b.update(60, 80)
	// Shift lastUpdated back 10s → 10s * 4/s = 40 units of decay → 20.
	b.lastUpdated = b.lastUpdated.Add(-10 * time.Second)
	est := b.estimate(time.Now())
	if est > 25 || est < 15 {
		t.Errorf("decay: got %v, want ~20", est)
	}
}

func TestBucketWaitIfFullHonorsContext(t *testing.T) {
	b := newBucket()
	b.update(78, 80) // above safeThreshold=70
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := b.waitIfFull(ctx)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("expected ctx cancellation error, got nil")
	}
	if elapsed >= 500*time.Millisecond {
		t.Errorf("wait should have stopped on ctx cancel quickly, took %v", elapsed)
	}
}
