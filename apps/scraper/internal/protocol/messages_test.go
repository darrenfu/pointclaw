package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestScrapeJobJSON(t *testing.T) {
	job := ScrapeJob{
		RequestID:   "test-123",
		Origin:      "SEA",
		Destination: "NRT",
		Month:       "2026-06",
		Priority:    "normal",
		RequestedAt: time.Date(2026, 3, 17, 20, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded ScrapeJob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.RequestID != job.RequestID {
		t.Errorf("request_id = %q, want %q", decoded.RequestID, job.RequestID)
	}
	if decoded.Origin != "SEA" {
		t.Errorf("origin = %q, want %q", decoded.Origin, "SEA")
	}
	if decoded.Month != "2026-06" {
		t.Errorf("month = %q, want %q", decoded.Month, "2026-06")
	}
}

func TestScrapeResultJSON(t *testing.T) {
	result := ScrapeResult{
		RequestID:   "test-123",
		Date:        "2026-06-01",
		Status:      "success",
		Origin:      "SEA",
		Destination: "NRT",
		Cheapest: &CheapestFare{
			Cabin: "economy",
			Miles: 25000,
			Cash:  5.60,
		},
		FlightCount: 3,
		ScrapedAt:   time.Date(2026, 3, 17, 20, 30, 5, 0, time.UTC),
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded ScrapeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Cheapest == nil {
		t.Fatal("cheapest is nil")
	}
	if decoded.Cheapest.Miles != 25000 {
		t.Errorf("cheapest miles = %d, want 25000", decoded.Cheapest.Miles)
	}
	if decoded.FlightCount != 3 {
		t.Errorf("flight_count = %d, want 3", decoded.FlightCount)
	}
}

func TestScrapeResultJSON_NoFlights(t *testing.T) {
	result := ScrapeResult{
		RequestID: "test-456",
		Date:      "2026-06-02",
		Status:    "no_flights",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded ScrapeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Cheapest != nil {
		t.Error("expected cheapest to be nil for no_flights")
	}
}
