package storage

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/darrenfu/pointclaw/scraper/pkg/types"
)

func getTestDB(t *testing.T) *DB {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping DB test")
	}

	ctx := context.Background()
	db, err := NewDB(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}
	return db
}

func TestDBConnection(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	t.Log("database connection OK")
}

func TestCreateTables(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	err := db.CreateTables(ctx)
	if err != nil {
		t.Fatalf("CreateTables failed: %v", err)
	}
	t.Log("tables created/verified OK")
}

func TestInsertAndQuerySearch(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Ensure tables exist
	if err := db.CreateTables(ctx); err != nil {
		t.Fatalf("CreateTables: %v", err)
	}

	// Insert a search
	rawJSON := json.RawMessage(`{"test": true, "slices": []}`)
	searchID, err := db.InsertSearch(ctx, "SEA", "NRT", "2026-06-15", "success", rawJSON)
	if err != nil {
		t.Fatalf("InsertSearch failed: %v", err)
	}
	if searchID <= 0 {
		t.Fatalf("expected positive search ID, got %d", searchID)
	}
	t.Logf("inserted search ID: %d", searchID)

	// Check recent search
	id, found, err := db.GetRecentSearch(ctx, "SEA", "NRT", "2026-06-15", 60*time.Second)
	if err != nil {
		t.Fatalf("GetRecentSearch error: %v", err)
	}
	if !found {
		t.Error("expected to find recent search")
	}
	if id != searchID {
		t.Errorf("GetRecentSearch returned id=%d, want %d", id, searchID)
	}
	t.Logf("GetRecentSearch found ID: %d", id)

	// Clean up
	db.pool.Exec(ctx, "DELETE FROM award_searches WHERE id = $1", searchID)
}

func TestInsertSearchWithFlights(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	if err := db.CreateTables(ctx); err != nil {
		t.Fatalf("CreateTables: %v", err)
	}

	flights := []types.NormalizedFlight{
		{
			FlightNumber: "JL 69",
			Carrier:      types.CarrierInfo{Code: "JL", Name: "Japan Airlines"},
			Departure:    types.AirportTime{Airport: "SEA", Time: "2026-06-15T12:00:00-07:00"},
			Arrival:      types.AirportTime{Airport: "NRT", Time: "2026-06-16T15:30:00+09:00"},
			Duration:     630,
			Aircraft:     "Boeing 787-9",
			IsDirect:     true,
			Amenities:    []string{"Wi-Fi", "Power"},
			Fares: []types.NormalizedFare{
				{Cabin: "economy", Miles: 25000, Cash: 5.60, SeatsRemaining: 3, BookingCode: "X", IsSaver: true},
				{Cabin: "business", Miles: 75000, Cash: 5.60, SeatsRemaining: 2, BookingCode: "I", IsSaver: false},
			},
		},
		{
			FlightNumber: "AS 135",
			Carrier:      types.CarrierInfo{Code: "AS", Name: "Alaska Airlines"},
			Departure:    types.AirportTime{Airport: "SEA", Time: "2026-06-15T16:00:00-07:00"},
			Arrival:      types.AirportTime{Airport: "NRT", Time: "2026-06-16T19:50:00+09:00"},
			Duration:     650,
			Aircraft:     "Boeing 737 MAX 9",
			IsDirect:     true,
			Fares: []types.NormalizedFare{
				{Cabin: "economy", Miles: 30000, Cash: 11.20, SeatsRemaining: 7, BookingCode: "Y", IsSaver: false},
			},
		},
	}

	rawJSON := json.RawMessage(`{"test": true}`)
	searchID, err := db.InsertSearchWithFlights(ctx, "SEA", "NRT", "2026-06-15", "success", rawJSON, flights)
	if err != nil {
		t.Fatalf("InsertSearchWithFlights failed: %v", err)
	}
	t.Logf("inserted search ID: %d with %d flights", searchID, len(flights))

	// Read back flights
	readFlights, err := db.GetFlightsBySearchID(ctx, searchID)
	if err != nil {
		t.Fatalf("GetFlightsBySearchID failed: %v", err)
	}

	if len(readFlights) != 2 {
		t.Fatalf("expected 2 flights, got %d", len(readFlights))
	}

	// Check first flight (JL 69)
	jl := readFlights[0]
	if jl.FlightNumber != "JL 69" {
		t.Errorf("flight number = %q, want %q", jl.FlightNumber, "JL 69")
	}
	if jl.Carrier.Code != "JL" {
		t.Errorf("carrier = %q, want %q", jl.Carrier.Code, "JL")
	}
	if len(jl.Fares) != 2 {
		t.Errorf("JL fares count = %d, want 2", len(jl.Fares))
	}
	// Fares sorted by miles_cost ASC, so economy (25K) first
	if jl.Fares[0].Miles != 25000 {
		t.Errorf("cheapest fare = %d, want 25000", jl.Fares[0].Miles)
	}
	if !jl.Fares[0].IsSaver {
		t.Error("cheapest fare should be saver")
	}
	if jl.Duration != 630 {
		t.Errorf("duration = %d, want 630", jl.Duration)
	}
	if !jl.IsDirect {
		t.Error("expected direct flight")
	}

	// Check amenities
	if len(jl.Amenities) != 2 || jl.Amenities[0] != "Wi-Fi" {
		t.Errorf("amenities = %v, want [Wi-Fi, Power]", jl.Amenities)
	}

	// Check second flight (AS 135)
	as := readFlights[1]
	if as.FlightNumber != "AS 135" {
		t.Errorf("flight number = %q, want %q", as.FlightNumber, "AS 135")
	}
	if len(as.Fares) != 1 {
		t.Errorf("AS fares count = %d, want 1", len(as.Fares))
	}

	t.Logf("read back %d flights with %d total fares — all correct",
		len(readFlights),
		len(readFlights[0].Fares)+len(readFlights[1].Fares))

	// Clean up (cascade deletes flights)
	db.pool.Exec(ctx, "DELETE FROM award_searches WHERE id = $1", searchID)
}

func TestGetRecentSearch_NotFound(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	if err := db.CreateTables(ctx); err != nil {
		t.Fatalf("CreateTables: %v", err)
	}

	// Query for a route that doesn't exist
	_, found, err := db.GetRecentSearch(ctx, "XXX", "YYY", "2099-12-31", 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected not found for nonexistent route")
	}
	t.Log("GetRecentSearch correctly returns not found")
}
