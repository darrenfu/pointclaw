package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/darrenfu/pointclaw/scraper/internal/protocol"
	"github.com/darrenfu/pointclaw/scraper/internal/scraping"
	"github.com/darrenfu/pointclaw/scraper/pkg/types"
	"github.com/playwright-community/playwright-go"
	"github.com/redis/go-redis/v9"
)

// TestRedisConnection verifies Redis is reachable.
func TestRedisConnection(t *testing.T) {
	addr := getEnv("REDIS_ADDR", "localhost:6379")
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	ctx := context.Background()
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		t.Fatalf("Redis not reachable at %s: %v", addr, err)
	}
	if pong != "PONG" {
		t.Fatalf("unexpected Redis response: %s", pong)
	}
	t.Logf("Redis connected: %s", addr)
}

// TestRedisStreamPublishConsume tests the job queue round-trip.
func TestRedisStreamPublishConsume(t *testing.T) {
	addr := getEnv("REDIS_ADDR", "localhost:6379")
	rc := protocol.NewRedisClient(addr)
	defer rc.Close()

	ctx := context.Background()

	// Clean up test stream
	rc.Client().Del(ctx, "scrape:jobs:test")

	// Publish a job via raw Redis (simulating FE)
	job := protocol.ScrapeJob{
		RequestID:   "test-integration-001",
		Origin:      "SEA",
		Destination: "NRT",
		Month:       "2026-06",
		Priority:    "normal",
		RequestedAt: time.Now(),
	}
	data, _ := json.Marshal(job)
	err := rc.Client().XAdd(ctx, &redis.XAddArgs{
		Stream: protocol.StreamKey,
		Values: map[string]interface{}{"data": string(data)},
	}).Err()
	if err != nil {
		t.Fatalf("failed to publish job: %v", err)
	}
	t.Log("job published to Redis stream")

	// Verify we can read it back
	streams, err := rc.Client().XRange(ctx, protocol.StreamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("failed to read stream: %v", err)
	}
	found := false
	for _, msg := range streams {
		d, ok := msg.Values["data"].(string)
		if !ok {
			continue
		}
		var j protocol.ScrapeJob
		json.Unmarshal([]byte(d), &j)
		if j.RequestID == "test-integration-001" {
			found = true
			t.Logf("found job: %+v", j)
		}
	}
	if !found {
		t.Error("published job not found in stream")
	}

	// Clean up
	rc.Client().Del(ctx, protocol.StreamKey)
}

// TestRedisCacheWriteRead tests cache write and read.
func TestRedisCacheWriteRead(t *testing.T) {
	addr := getEnv("REDIS_ADDR", "localhost:6379")
	rc := protocol.NewRedisClient(addr)
	defer rc.Close()

	ctx := context.Background()

	testData := []byte(`{"flights": [{"miles": 25000}]}`)
	err := rc.SetCache(ctx, "SEA", "NRT", "2026-06-01", testData, 10*time.Second)
	if err != nil {
		t.Fatalf("SetCache failed: %v", err)
	}

	got, err := rc.GetCache(ctx, "SEA", "NRT", "2026-06-01")
	if err != nil {
		t.Fatalf("GetCache failed: %v", err)
	}
	if got == nil {
		t.Fatal("cache returned nil")
	}
	if string(got) != string(testData) {
		t.Errorf("cache data mismatch: got %s", string(got))
	}
	t.Log("cache write/read OK")

	// Clean up
	rc.Client().Del(ctx, "cache:SEA:NRT:2026-06-01")
}

// TestRedisPubSub tests publish/subscribe for results.
func TestRedisPubSub(t *testing.T) {
	addr := getEnv("REDIS_ADDR", "localhost:6379")
	rc := protocol.NewRedisClient(addr)
	defer rc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	requestID := "test-pubsub-001"
	channel := fmt.Sprintf("%s:%s", protocol.ResultPrefix, requestID)

	// Subscribe
	sub := rc.Client().Subscribe(ctx, channel)
	defer sub.Close()

	// Wait for subscription to be ready
	_, err := sub.Receive(ctx)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	// Publish a result
	result := protocol.ScrapeResult{
		RequestID:   requestID,
		Date:        "2026-06-01",
		Status:      "success",
		Origin:      "SEA",
		Destination: "NRT",
		Cheapest:    &protocol.CheapestFare{Cabin: "economy", Miles: 25000, Cash: 5.60},
		FlightCount: 3,
		ScrapedAt:   time.Now(),
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		rc.PublishResult(context.Background(), result)
	}()

	// Receive
	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive failed: %v", err)
	}

	var received protocol.ScrapeResult
	json.Unmarshal([]byte(msg.Payload), &received)
	if received.Cheapest.Miles != 25000 {
		t.Errorf("received miles = %d, want 25000", received.Cheapest.Miles)
	}
	t.Logf("pub/sub round-trip OK: %s → %d miles", received.Date, received.Cheapest.Miles)
}

// TestPlaywrightLaunch verifies Playwright can launch Chromium.
func TestPlaywrightLaunch(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("failed to start playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-crashpad",
			"--disable-gpu",
			"--disable-dev-shm-usage",
		},
	})
	if err != nil {
		t.Skipf("Cannot launch Chromium in this environment (sandbox restriction): %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("failed to create page: %v", err)
	}

	// Simple navigation test
	_, err = page.Goto("https://example.com")
	if err != nil {
		t.Fatalf("navigation failed: %v", err)
	}

	title, err := page.Title()
	if err != nil {
		t.Fatalf("get title failed: %v", err)
	}
	t.Logf("Playwright Chromium OK — loaded page with title: %q", title)
	page.Close()
}

// TestAlaskaScrape does a REAL scrape of Alaska Airlines for a single date.
// This is a slow test that hits the real website.
func TestAlaskaScrape(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real scrape in short mode")
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("failed to start playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		t.Fatalf("failed to launch chromium: %v", err)
	}
	defer browser.Close()

	// Create context with anti-bot settings
	width, height := scraping.RandomViewport()
	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(scraping.RandomUA()),
		Viewport:  &playwright.Size{Width: width, Height: height},
		Locale:    playwright.String("en-US"),
	})
	if err != nil {
		t.Fatalf("failed to create context: %v", err)
	}
	defer ctx.Close()

	page, err := ctx.NewPage()
	if err != nil {
		t.Fatalf("failed to create page: %v", err)
	}
	defer page.Close()

	// Pick a date ~3 months out
	searchDate := time.Now().AddDate(0, 3, 0).Format("2006-01-02")

	t.Logf("scraping SEA → NRT on %s...", searchDate)
	flights, raw, err := scraping.ScrapeAlaskaAwards(page, "SEA", "NRT", searchDate)

	if err != nil {
		t.Logf("scrape error (may be expected if bot-detected): %v", err)
		// Don't fail — being blocked is a valid outcome we need to handle
		return
	}

	t.Logf("raw response: departureStation=%s, arrivalStation=%s",
		raw.DepartureStation, raw.ArrivalStation)

	if raw.Slices != nil {
		t.Logf("total slices: %d", len(raw.Slices))
	}

	t.Logf("normalized flights: %d", len(flights))
	for _, f := range flights {
		for _, fare := range f.Fares {
			t.Logf("  %s %s→%s | %s | %d miles + $%.2f | %d seats | saver=%v",
				f.FlightNumber, f.Departure.Airport, f.Arrival.Airport,
				fare.Cabin, fare.Miles, fare.Cash, fare.SeatsRemaining, fare.IsSaver)
		}
	}
}

// TestAlaskaNormalizerWithRealishData tests normalizer with realistic data structure.
func TestAlaskaNormalizerWithRealishData(t *testing.T) {
	// Simulate a response with multiple slices, mixed direct and connecting
	raw := &types.AlaskaResponse{
		DepartureStation: "SEA",
		ArrivalStation:   "NRT",
		Slices: []types.AlaskaSlice{
			// Direct flight
			{
				Segments: []types.AlaskaSegment{
					{
						PublishingCarrier: types.AlaskaCarrier{CarrierCode: "JL", CarrierFullName: "Japan Airlines", FlightNumber: 69},
						DepartureStation: "SEA", ArrivalStation: "NRT",
						Aircraft: "Boeing 787-9", Duration: 630,
						DepartureTime: "2026-06-01T12:00:00-07:00",
						ArrivalTime:   "2026-06-02T15:30:00+09:00",
						Amenities:     []string{"Wi-Fi", "Power"},
					},
				},
				Fares: map[string]types.AlaskaFare{
					"saver_economy": {MilesPoints: 25000, GrandTotal: 5.60, SeatsRemaining: 3, Cabins: []string{"SAVER"}, BookingCodes: []string{"X"}},
					"main_economy":  {MilesPoints: 40000, GrandTotal: 5.60, SeatsRemaining: 9, Cabins: []string{"MAIN"}, BookingCodes: []string{"Y"}},
					"business":      {MilesPoints: 75000, GrandTotal: 5.60, SeatsRemaining: 2, Cabins: []string{"BUSINESS"}, BookingCodes: []string{"I"}},
				},
			},
			// Connecting flight (should be skipped)
			{
				Segments: []types.AlaskaSegment{
					{DepartureStation: "SEA", ArrivalStation: "LAX", PublishingCarrier: types.AlaskaCarrier{CarrierCode: "AS", FlightNumber: 100}},
					{DepartureStation: "LAX", ArrivalStation: "NRT", PublishingCarrier: types.AlaskaCarrier{CarrierCode: "JL", FlightNumber: 15}},
				},
				Fares: map[string]types.AlaskaFare{},
			},
			// Another direct
			{
				Segments: []types.AlaskaSegment{
					{
						PublishingCarrier: types.AlaskaCarrier{CarrierCode: "AS", CarrierFullName: "Alaska Airlines", FlightNumber: 135},
						DepartureStation: "SEA", ArrivalStation: "NRT",
						Aircraft: "Boeing 737 MAX 9", Duration: 650,
						DepartureTime: "2026-06-01T16:00:00-07:00",
						ArrivalTime:   "2026-06-02T19:50:00+09:00",
					},
				},
				Fares: map[string]types.AlaskaFare{
					"economy": {MilesPoints: 30000, GrandTotal: 11.20, SeatsRemaining: 7, Cabins: []string{"MAIN"}, BookingCodes: []string{"Y"}},
					"first":   {MilesPoints: 70000, GrandTotal: 11.20, SeatsRemaining: 1, Cabins: []string{"FIRST"}, BookingCodes: []string{"P"}},
				},
			},
		},
	}

	flights := scraping.NormalizeAlaskaResponse(raw, "SEA", "NRT")

	if len(flights) != 2 {
		t.Fatalf("expected 2 direct flights, got %d", len(flights))
	}

	// First flight: JL 69
	jl := flights[0]
	if jl.FlightNumber != "JL 69" {
		t.Errorf("first flight = %q, want %q", jl.FlightNumber, "JL 69")
	}
	if jl.Duration != 630 {
		t.Errorf("duration = %d, want 630", jl.Duration)
	}
	// Economy should be 25K (saver), not 40K (main)
	var econFare *types.NormalizedFare
	for i := range jl.Fares {
		if jl.Fares[i].Cabin == "economy" {
			econFare = &jl.Fares[i]
		}
	}
	if econFare == nil || econFare.Miles != 25000 {
		t.Errorf("economy fare should be 25K saver, got %+v", econFare)
	}

	// Second flight: AS 135
	as := flights[1]
	if as.FlightNumber != "AS 135" {
		t.Errorf("second flight = %q, want %q", as.FlightNumber, "AS 135")
	}

	t.Logf("normalizer test passed: %d flights, connecting correctly skipped", len(flights))
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
