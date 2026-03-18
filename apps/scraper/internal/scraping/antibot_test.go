package scraping

import (
	"testing"
	"time"
)

func TestRandomUA(t *testing.T) {
	ua := RandomUA()
	if ua == "" {
		t.Error("expected non-empty user agent")
	}
	// Should be a valid-looking UA string
	if len(ua) < 50 {
		t.Errorf("UA too short: %q", ua)
	}
}

func TestRandomViewport(t *testing.T) {
	for i := 0; i < 100; i++ {
		w, h := RandomViewport()
		if w < 1280 || w > 1920 {
			t.Errorf("width %d out of range [1280, 1920]", w)
		}
		if h < 720 || h > 1080 {
			t.Errorf("height %d out of range [720, 1080]", h)
		}
	}
}

func TestGaussianJitter(t *testing.T) {
	for i := 0; i < 100; i++ {
		d := GaussianJitter(4.0, 1.5, 2.0, 8.0)
		if d < 2*time.Second || d > 8*time.Second {
			t.Errorf("jitter %v out of range [2s, 8s]", d)
		}
	}
}

func TestExponentialBackoff(t *testing.T) {
	// Attempt 0: ~1000ms base
	d0 := ExponentialBackoff(0, 1000, 300000)
	if d0 < 500*time.Millisecond || d0 > 3*time.Second {
		t.Errorf("attempt 0 backoff %v unexpected", d0)
	}

	// Attempt 4: should be significantly larger
	d4 := ExponentialBackoff(4, 1000, 300000)
	if d4 < 10*time.Second {
		t.Errorf("attempt 4 backoff %v too small", d4)
	}

	// Should cap at maxMs
	d10 := ExponentialBackoff(10, 1000, 300000)
	if d10 > 300*time.Second {
		t.Errorf("attempt 10 backoff %v exceeds max", d10)
	}
}

func TestTimezoneForAirport(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"SEA", "America/Los_Angeles"},
		{"LAX", "America/Los_Angeles"},
		{"JFK", "America/New_York"},
		{"HNL", "Pacific/Honolulu"},
		{"YVR", "America/Vancouver"},
		{"UNKNOWN", "America/Los_Angeles"}, // fallback
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := TimezoneForAirport(tt.code)
			if got != tt.want {
				t.Errorf("TimezoneForAirport(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}
