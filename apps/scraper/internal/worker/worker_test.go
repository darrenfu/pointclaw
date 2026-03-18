package worker

import (
	"testing"
	"time"

	"github.com/darrenfu/pointclaw/scraper/pkg/types"
)

func TestDatesInMonth(t *testing.T) {
	tests := []struct {
		month    string
		wantLen  int
		wantErr  bool
	}{
		{"2026-01", 31, false},
		{"2026-02", 28, false},
		{"2024-02", 29, false}, // leap year
		{"2026-06", 30, false},
		{"2026-12", 31, false},
		{"invalid", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.month, func(t *testing.T) {
			dates, err := datesInMonth(tt.month)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(dates) != tt.wantLen {
				t.Errorf("got %d dates, want %d", len(dates), tt.wantLen)
			}
			// First date should be YYYY-MM-01
			if dates[0] != tt.month+"-01" {
				t.Errorf("first date = %q, want %q", dates[0], tt.month+"-01")
			}
			// All dates should parse
			for _, d := range dates {
				if _, err := time.Parse("2006-01-02", d); err != nil {
					t.Errorf("invalid date %q: %v", d, err)
				}
			}
		})
	}
}

func TestDatesInMonth_June(t *testing.T) {
	dates, err := datesInMonth("2026-06")
	if err != nil {
		t.Fatal(err)
	}
	if dates[0] != "2026-06-01" {
		t.Errorf("first = %q", dates[0])
	}
	if dates[len(dates)-1] != "2026-06-30" {
		t.Errorf("last = %q", dates[len(dates)-1])
	}
}

func TestFindCheapest(t *testing.T) {
	// Import types indirectly via the function
	// findCheapest is in this package so we can test it directly
	flights := []types.NormalizedFlight{} // empty
	result := findCheapest(flights)
	if result != nil {
		t.Error("expected nil for empty flights")
	}
}
