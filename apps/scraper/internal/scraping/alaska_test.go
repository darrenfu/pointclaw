package scraping

import (
	"testing"

	"github.com/darrenfu/pointclaw/scraper/pkg/types"
)

func TestNormalizeAlaskaResponse_NoSlices(t *testing.T) {
	raw := &types.AlaskaResponse{
		DepartureStation: "SEA",
		ArrivalStation:   "NRT",
	}
	flights := NormalizeAlaskaResponse(raw, "SEA", "NRT")
	if len(flights) != 0 {
		t.Errorf("expected 0 flights for nil slices, got %d", len(flights))
	}
}

func TestNormalizeAlaskaResponse_DirectFlight(t *testing.T) {
	raw := &types.AlaskaResponse{
		DepartureStation: "SEA",
		ArrivalStation:   "NRT",
		Slices: []types.AlaskaSlice{
			{
				Segments: []types.AlaskaSegment{
					{
						PublishingCarrier: types.AlaskaCarrier{
							CarrierCode:     "AS",
							CarrierFullName: "Alaska Airlines",
							FlightNumber:    123,
						},
						DepartureStation: "SEA",
						ArrivalStation:   "NRT",
						Aircraft:         "Boeing 737-900",
						Duration:         660,
						DepartureTime:    "2026-06-01T10:30:00-07:00",
						ArrivalTime:      "2026-06-02T14:30:00+09:00",
						Amenities:        []string{"Wi-Fi"},
					},
				},
				Fares: map[string]types.AlaskaFare{
					"saver": {
						GrandTotal:     5.60,
						MilesPoints:    25000,
						SeatsRemaining: 5,
						Cabins:         []string{"SAVER"},
						BookingCodes:   []string{"X"},
					},
					"main": {
						GrandTotal:     11.20,
						MilesPoints:    35000,
						SeatsRemaining: 9,
						Cabins:         []string{"MAIN"},
						BookingCodes:   []string{"Y"},
					},
					"first": {
						GrandTotal:     11.20,
						MilesPoints:    70000,
						SeatsRemaining: 2,
						Cabins:         []string{"FIRST"},
						BookingCodes:   []string{"I"},
					},
				},
			},
		},
	}

	flights := NormalizeAlaskaResponse(raw, "SEA", "NRT")

	if len(flights) != 1 {
		t.Fatalf("expected 1 flight, got %d", len(flights))
	}

	f := flights[0]
	if f.FlightNumber != "AS 123" {
		t.Errorf("flight number = %q, want %q", f.FlightNumber, "AS 123")
	}
	if f.Carrier.Code != "AS" {
		t.Errorf("carrier code = %q, want %q", f.Carrier.Code, "AS")
	}
	if f.Duration != 660 {
		t.Errorf("duration = %d, want %d", f.Duration, 660)
	}
	if !f.IsDirect {
		t.Error("expected direct flight")
	}
	if f.Departure.Airport != "SEA" {
		t.Errorf("departure airport = %q, want %q", f.Departure.Airport, "SEA")
	}

	// Should have 2 fares: economy (cheapest of SAVER/MAIN) and business (FIRST)
	if len(f.Fares) != 2 {
		t.Fatalf("expected 2 fares (economy + business), got %d", len(f.Fares))
	}

	// Find economy fare — should be the SAVER (25K), not MAIN (35K)
	var econFare, bizFare *types.NormalizedFare
	for i := range f.Fares {
		switch f.Fares[i].Cabin {
		case "economy":
			econFare = &f.Fares[i]
		case "business":
			bizFare = &f.Fares[i]
		}
	}

	if econFare == nil {
		t.Fatal("no economy fare found")
	}
	if econFare.Miles != 25000 {
		t.Errorf("economy miles = %d, want 25000 (should pick cheapest)", econFare.Miles)
	}
	if !econFare.IsSaver {
		t.Error("economy fare should be flagged as saver")
	}

	if bizFare == nil {
		t.Fatal("no business fare found")
	}
	if bizFare.Miles != 70000 {
		t.Errorf("business miles = %d, want 70000", bizFare.Miles)
	}
}

func TestNormalizeAlaskaResponse_SkipsConnecting(t *testing.T) {
	raw := &types.AlaskaResponse{
		Slices: []types.AlaskaSlice{
			{
				// 2 segments = connecting flight, should be skipped
				Segments: []types.AlaskaSegment{
					{DepartureStation: "SEA", ArrivalStation: "LAX"},
					{DepartureStation: "LAX", ArrivalStation: "NRT"},
				},
				Fares: map[string]types.AlaskaFare{},
			},
		},
	}
	flights := NormalizeAlaskaResponse(raw, "SEA", "NRT")
	if len(flights) != 0 {
		t.Errorf("expected 0 flights for connecting, got %d", len(flights))
	}
}

func TestNormalizeAlaskaResponse_SkipsMismatchedRoute(t *testing.T) {
	raw := &types.AlaskaResponse{
		Slices: []types.AlaskaSlice{
			{
				Segments: []types.AlaskaSegment{
					{
						DepartureStation: "LAX", // not SEA
						ArrivalStation:   "NRT",
						PublishingCarrier: types.AlaskaCarrier{CarrierCode: "AS", FlightNumber: 1},
					},
				},
				Fares: map[string]types.AlaskaFare{},
			},
		},
	}
	flights := NormalizeAlaskaResponse(raw, "SEA", "NRT")
	if len(flights) != 0 {
		t.Errorf("expected 0 flights for mismatched origin, got %d", len(flights))
	}
}
