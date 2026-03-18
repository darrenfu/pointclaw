package scraping

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/darrenfu/pointclaw/scraper/pkg/types"
	"github.com/playwright-community/playwright-go"
)

// ScrapeAlaskaAwards scrapes Alaska Airlines award availability for a given route and date.
// It uses Playwright to load the search page and intercept the searchbff/V3 XHR response.
func ScrapeAlaskaAwards(page playwright.Page, origin, dest, date string) ([]types.NormalizedFlight, *types.AlaskaResponse, error) {
	// Block tracking domains to speed up page load
	err := page.Route("**/*", func(route playwright.Route) {
		url := route.Request().URL()
		for _, domain := range BlockedDomains {
			if strings.Contains(url, domain) {
				route.Abort()
				return
			}
		}
		route.Continue()
	})
	if err != nil {
		return nil, nil, fmt.Errorf("setup route blocking: %w", err)
	}

	// Set up response interception BEFORE navigating
	var rawResponse *types.AlaskaResponse
	var interceptErr error
	var mu sync.Mutex
	responseReceived := make(chan struct{}, 1)

	// Log ALL responses for debugging
	page.OnResponse(func(response playwright.Response) {
		url := response.URL()
		status := response.Status()

		// Log interesting responses
		if strings.Contains(url, "searchbff") || strings.Contains(url, "search") {
			slog.Debug("response intercepted",
				"url", url,
				"status", status,
				"content_type", response.Headers()["content-type"],
			)
		}

		// Catch the award search response
		if strings.Contains(url, "searchbff") && status == 200 {
			mu.Lock()
			defer mu.Unlock()

			body, err := response.Body()
			if err != nil {
				interceptErr = fmt.Errorf("read response body: %w", err)
				return
			}

			slog.Info("searchbff response captured",
				"url", url,
				"status", status,
				"body_len", len(body),
			)

			var resp types.AlaskaResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				// Maybe it's not the right JSON format, log and continue
				slog.Warn("searchbff response not AlaskaResponse format",
					"url", url,
					"body_preview", string(body[:min(200, len(body))]),
					"error", err,
				)
				return
			}
			rawResponse = &resp
			select {
			case responseReceived <- struct{}{}:
			default:
			}
		}
	})

	// Try multiple URL patterns — Alaska may use different URL formats
	searchURLs := []string{
		// Pattern 1: Calendar URL (from user's original URL)
		fmt.Sprintf(
			"https://www.alaskaair.com/search/calendar?O=%s&D=%s&OD=%s&A=1&RT=false&RequestType=Calendar&ShoppingMethod=onlineaward&locale=en-us&FareType=Lowest+price+available",
			origin, dest, date,
		),
		// Pattern 2: Results page
		fmt.Sprintf(
			"https://www.alaskaair.com/search/results?O=%s&D=%s&OD=%s&A=1&RT=false&ShoppingMethod=onlineaward",
			origin, dest, date,
		),
		// Pattern 3: Direct search page (AwardWiz pattern)
		fmt.Sprintf(
			"https://www.alaskaair.com/search?O=%s&D=%s&OD=%s&A=1&RT=false&ShoppingMethod=onlineaward",
			origin, dest, date,
		),
	}

	for i, searchURL := range searchURLs {
		slog.Info("navigating", "attempt", i+1, "url", searchURL)

		_, err = page.Goto(searchURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
			Timeout:   playwright.Float(30000),
		})
		if err != nil {
			slog.Warn("navigation failed, trying next URL", "attempt", i+1, "error", err)
			continue
		}

		// Log current URL (may have redirected)
		currentURL := page.URL()
		slog.Info("page loaded", "current_url", currentURL)

		// Log page title for debugging
		title, _ := page.Title()
		slog.Info("page title", "title", title)

		// Wait for XHR to fire — use a timeout channel
		select {
		case <-responseReceived:
			slog.Info("searchbff response received")
		case <-time.After(15 * time.Second):
			slog.Warn("timeout waiting for searchbff response", "attempt", i+1)

			// Debug: take a screenshot and dump network log
			screenshotPath := fmt.Sprintf("/tmp/pointclaw-debug-%s-%s-%s-attempt%d.png", origin, dest, date, i+1)
			page.Screenshot(playwright.PageScreenshotOptions{
				Path:     playwright.String(screenshotPath),
				FullPage: playwright.Bool(true),
			})
			slog.Info("debug screenshot saved", "path", screenshotPath)

			// Dump page content snippet for debugging
			content, _ := page.Content()
			if len(content) > 500 {
				content = content[:500]
			}
			slog.Debug("page content preview", "html", content)

			continue // try next URL pattern
		}

		if rawResponse != nil {
			break
		}
	}

	if interceptErr != nil {
		return nil, nil, interceptErr
	}
	if rawResponse == nil {
		return nil, nil, fmt.Errorf("no searchbff response intercepted (tried %d URL patterns)", len(searchURLs))
	}

	// Normalize the response
	flights := NormalizeAlaskaResponse(rawResponse, origin, dest)
	return flights, rawResponse, nil
}

// NormalizeAlaskaResponse converts raw Alaska API response to our normalized format.
func NormalizeAlaskaResponse(raw *types.AlaskaResponse, queryOrigin, queryDest string) []types.NormalizedFlight {
	if raw.Slices == nil {
		return nil
	}

	var flights []types.NormalizedFlight
	for _, slice := range raw.Slices {
		// Only direct flights (single segment) for now
		if len(slice.Segments) != 1 {
			continue
		}
		seg := slice.Segments[0]

		// Validate origin/destination match
		if seg.DepartureStation != queryOrigin || seg.ArrivalStation != queryDest {
			continue
		}

		flight := types.NormalizedFlight{
			FlightNumber: fmt.Sprintf("%s %d", seg.PublishingCarrier.CarrierCode, seg.PublishingCarrier.FlightNumber),
			Carrier: types.CarrierInfo{
				Code: seg.PublishingCarrier.CarrierCode,
				Name: seg.PublishingCarrier.CarrierFullName,
			},
			Departure: types.AirportTime{
				Airport: seg.DepartureStation,
				Time:    seg.DepartureTime,
			},
			Arrival: types.AirportTime{
				Airport: seg.ArrivalStation,
				Time:    seg.ArrivalTime,
			},
			Duration:  seg.Duration,
			Aircraft:  seg.Aircraft,
			IsDirect:  true,
			Amenities: seg.Amenities,
		}

		// Process fares — keep lowest per cabin
		cabinBest := make(map[string]types.NormalizedFare)
		for _, fare := range slice.Fares {
			if len(fare.BookingCodes) == 0 || len(fare.Cabins) == 0 {
				continue
			}
			cabin := types.CabinName(fare.Cabins[0])
			normalized := types.NormalizedFare{
				Cabin:          cabin,
				Miles:          fare.MilesPoints,
				Cash:           fare.GrandTotal,
				SeatsRemaining: fare.SeatsRemaining,
				BookingCode:    fare.BookingCodes[0],
				IsSaver:        types.IsSaverCabin(fare.Cabins[0]),
			}

			if existing, ok := cabinBest[cabin]; !ok || normalized.Miles < existing.Miles {
				cabinBest[cabin] = normalized
			}
		}

		for _, fare := range cabinBest {
			flight.Fares = append(flight.Fares, fare)
		}

		flights = append(flights, flight)
	}

	return flights
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
