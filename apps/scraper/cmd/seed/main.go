// seed populates Redis cache with mock award data for e2e testing.
// Run: go run ./cmd/seed
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/darrenfu/pointclaw/scraper/internal/protocol"
)

type MockFlight struct {
	FlightNumber string    `json:"flightNumber"`
	Carrier      carrier   `json:"carrier"`
	Departure    airTime   `json:"departure"`
	Arrival      airTime   `json:"arrival"`
	Duration     int       `json:"duration"`
	Aircraft     string    `json:"aircraft"`
	IsDirect     bool      `json:"isDirect"`
	Fares        []fare    `json:"fares"`
	Amenities    []string  `json:"amenities"`
}

type carrier struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type airTime struct {
	Airport string `json:"airport"`
	Time    string `json:"time"`
}

type fare struct {
	Cabin          string  `json:"cabin"`
	Miles          int     `json:"miles"`
	Cash           float64 `json:"cash"`
	SeatsRemaining int     `json:"seatsRemaining"`
	BookingCode    string  `json:"bookingCode"`
	IsSaver        bool    `json:"isSaver"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	origin := "SEA"
	dest := "NRT"
	month := time.Now().Format("2006-01") // current month

	if len(os.Args) > 1 {
		month = os.Args[1]
	}

	rc := protocol.NewRedisClient(redisAddr)
	defer rc.Close()

	ctx := context.Background()
	year, mon := parseMonth(month)
	daysInMonth := time.Date(year, time.Month(mon+1), 0, 0, 0, 0, 0, time.UTC).Day()

	slog.Info("seeding mock data", "origin", origin, "dest", dest, "month", month, "days", daysInMonth)

	seeded := 0
	for d := 1; d <= daysInMonth; d++ {
		dateStr := fmt.Sprintf("%d-%02d-%02d", year, mon, d)

		// 70% chance of availability
		if rand.Float64() > 0.7 {
			// No flights
			result := protocol.ScrapeResult{
				RequestID:   "seed",
				Date:        dateStr,
				Status:      "no_flights",
				Origin:      origin,
				Destination: dest,
				FlightCount: 0,
				ScrapedAt:   time.Now(),
			}
			data, _ := json.Marshal(result)
			rc.SetCache(ctx, origin, dest, dateStr, data, 24*time.Hour)
			continue
		}

		// Generate 1-3 mock flights
		numFlights := 1 + rand.Intn(3)
		flights := generateFlights(origin, dest, dateStr, numFlights)

		// Find cheapest
		cheapestMiles := 999999
		cheapestCabin := "economy"
		cheapestCash := 5.60
		for _, f := range flights {
			for _, fare := range f.Fares {
				if fare.Miles < cheapestMiles {
					cheapestMiles = fare.Miles
					cheapestCabin = fare.Cabin
					cheapestCash = fare.Cash
				}
			}
		}

		result := protocol.ScrapeResult{
			RequestID:   "seed",
			Date:        dateStr,
			Status:      "success",
			Origin:      origin,
			Destination: dest,
			Cheapest: &protocol.CheapestFare{
				Cabin: cheapestCabin,
				Miles: cheapestMiles,
				Cash:  cheapestCash,
			},
			FlightCount: len(flights),
			ScrapedAt:   time.Now(),
		}
		data, _ := json.Marshal(result)
		rc.SetCache(ctx, origin, dest, dateStr, data, 24*time.Hour)

		// Also cache the full flight list for date detail view
		flightsData, _ := json.Marshal(map[string]interface{}{
			"flights": flights,
			"cached":  true,
			"date":    dateStr,
			"origin":  origin,
			"destination": dest,
		})
		flightKey := fmt.Sprintf("flights:%s:%s:%s", origin, dest, dateStr)
		rc.Client().Set(ctx, flightKey, flightsData, 24*time.Hour)

		seeded++
	}

	slog.Info("seed complete", "available_days", seeded, "total_days", daysInMonth)
}

func generateFlights(origin, dest, date string, count int) []MockFlight {
	carriers := []struct {
		code, name, aircraft string
	}{
		{"JL", "Japan Airlines", "Boeing 787-9"},
		{"AS", "Alaska Airlines", "Boeing 737 MAX 9"},
		{"NH", "ANA", "Boeing 777-300ER"},
	}

	departureTimes := []string{"10:30", "12:00", "14:30", "16:00", "22:00"}

	var flights []MockFlight
	for i := 0; i < count && i < len(carriers); i++ {
		c := carriers[i]
		depHour := departureTimes[rand.Intn(len(departureTimes))]
		flightNum := 100 + rand.Intn(900)

		// Economy saver: 20-35K
		saverMiles := 20000 + rand.Intn(15000)
		saverMiles = (saverMiles / 500) * 500 // round to 500

		flight := MockFlight{
			FlightNumber: fmt.Sprintf("%s %d", c.code, flightNum),
			Carrier:      carrier{Code: c.code, Name: c.name},
			Departure:    airTime{Airport: origin, Time: fmt.Sprintf("%sT%s:00-07:00", date, depHour)},
			Arrival:      airTime{Airport: dest, Time: fmt.Sprintf("%sT%s:00+09:00", nextDay(date), addHours(depHour, 10+rand.Intn(3)))},
			Duration:     600 + rand.Intn(120),
			Aircraft:     c.aircraft,
			IsDirect:     true,
			Amenities:    []string{"Wi-Fi"},
			Fares: []fare{
				{Cabin: "economy", Miles: saverMiles, Cash: 5.60, SeatsRemaining: 1 + rand.Intn(9), BookingCode: "X", IsSaver: true},
				{Cabin: "economy", Miles: saverMiles + 15000, Cash: 5.60, SeatsRemaining: 9, BookingCode: "Y", IsSaver: false},
				{Cabin: "business", Miles: saverMiles*2 + 20000, Cash: 5.60, SeatsRemaining: 1 + rand.Intn(4), BookingCode: "I", IsSaver: false},
			},
		}
		flights = append(flights, flight)
	}
	return flights
}

func parseMonth(m string) (int, int) {
	var year, mon int
	fmt.Sscanf(m, "%d-%d", &year, &mon)
	return year, mon
}

func nextDay(date string) string {
	t, _ := time.Parse("2006-01-02", date)
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

func addHours(timeStr string, hours int) string {
	h := 0
	m := 0
	fmt.Sscanf(timeStr, "%d:%d", &h, &m)
	h = (h + hours) % 24
	return fmt.Sprintf("%02d:%02d", h, m)
}
