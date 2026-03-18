package protocol

import "time"

// ScrapeJob is sent from FE to scraper via Redis Stream
type ScrapeJob struct {
	RequestID   string    `json:"request_id"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	Month       string    `json:"month"` // "2026-06"
	Priority    string    `json:"priority"`
	RequestedAt time.Time `json:"requested_at"`
}

// ScrapeResult is published per-date from scraper to FE via Redis Pub/Sub
type ScrapeResult struct {
	RequestID   string       `json:"request_id"`
	Date        string       `json:"date"` // "2026-06-01"
	Status      string       `json:"status"` // "success", "no_flights", "error"
	Origin      string       `json:"origin"`
	Destination string       `json:"destination"`
	Cheapest    *CheapestFare `json:"cheapest,omitempty"`
	FlightCount int          `json:"flight_count"`
	ScrapedAt   time.Time    `json:"scraped_at"`
}

// CheapestFare represents the lowest fare found for a date
type CheapestFare struct {
	Cabin string  `json:"cabin"`
	Miles int     `json:"miles"`
	Cash  float64 `json:"cash"`
}

// JobCompletion signals all dates in a job are done
type JobCompletion struct {
	RequestID      string `json:"request_id"`
	Status         string `json:"status"` // "done"
	TotalDates     int    `json:"total_dates"`
	AvailableDates int    `json:"available_dates"`
	FailedDates    int    `json:"failed_dates"`
}
