package scraping

import (
	"math"
	"math/rand"
	"time"
)

// BlockedDomains are tracking/analytics domains to block (from AwardWiz).
// Blocking these speeds up page load and reduces fingerprinting.
var BlockedDomains = []string{
	"cdn.appdynamics.com",
	"siteintercept.qualtrics.com",
	"dc.services.visualstudio.com",
	"js.adsrvr.org",
	"bing.com",
	"tiktok.com",
	"www.googletagmanager.com",
	"facebook.net",
	"demdex.net",
	"cdn.uplift-platform.com",
	"doubleclick.net",
	"www.google-analytics.com",
	"collect.tealiumiq.com",
	"alaskaair-app.quantummetric.com",
	"facebook.com",
	"rl.quantummetric.com",
	"app.securiti.ai",
	"cdn.optimizely.com",
}

// UserAgents is a pool of real Chrome UA strings to rotate.
var UserAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:133.0) Gecko/20100101 Firefox/133.0",
}

// RandomUA returns a random user agent string.
func RandomUA() string {
	return UserAgents[rand.Intn(len(UserAgents))]
}

// RandomViewport returns a random viewport size.
func RandomViewport() (width, height int) {
	width = 1280 + rand.Intn(640)   // 1280-1920
	height = 720 + rand.Intn(360)   // 720-1080
	return
}

// GaussianJitter returns a random delay with Gaussian distribution.
// Mean and stddev are in seconds. Result is clamped to [min, max] seconds.
func GaussianJitter(mean, stddev, min, max float64) time.Duration {
	delay := rand.NormFloat64()*stddev + mean
	delay = math.Max(min, math.Min(max, delay))
	return time.Duration(delay * float64(time.Second))
}

// ExponentialBackoff returns the backoff duration for attempt n.
// Formula: 2^n * baseMs + random jitter, capped at maxMs.
func ExponentialBackoff(attempt int, baseMs, maxMs int) time.Duration {
	backoff := float64(baseMs) * math.Pow(2, float64(attempt))
	jitter := rand.Float64() * float64(baseMs) * math.Pow(2, float64(attempt-1))
	total := math.Min(backoff+jitter, float64(maxMs))
	return time.Duration(total) * time.Millisecond
}

// TimezoneForAirport returns an IANA timezone for common origin airports.
func TimezoneForAirport(code string) string {
	tzMap := map[string]string{
		"SEA": "America/Los_Angeles",
		"LAX": "America/Los_Angeles",
		"SFO": "America/Los_Angeles",
		"PDX": "America/Los_Angeles",
		"YVR": "America/Vancouver",
		"JFK": "America/New_York",
		"ORD": "America/Chicago",
		"DFW": "America/Chicago",
		"MIA": "America/New_York",
		"HNL": "Pacific/Honolulu",
	}
	if tz, ok := tzMap[code]; ok {
		return tz
	}
	return "America/Los_Angeles"
}
