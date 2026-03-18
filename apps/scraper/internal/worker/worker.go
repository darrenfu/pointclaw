package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/darrenfu/pointclaw/scraper/internal/protocol"
	"github.com/darrenfu/pointclaw/scraper/internal/scraping"
	"github.com/darrenfu/pointclaw/scraper/pkg/types"
	"github.com/playwright-community/playwright-go"
)

// Worker owns a single Playwright browser context and processes one scrape at a time.
type Worker struct {
	id         int
	browser    playwright.Browser
	resultChan chan<- protocol.ScrapeResult
}

func New(id int, browser playwright.Browser, resultChan chan<- protocol.ScrapeResult) *Worker {
	return &Worker{
		id:         id,
		browser:    browser,
		resultChan: resultChan,
	}
}

// Run processes jobs from the job channel until context is cancelled.
func (w *Worker) Run(ctx context.Context, jobChan <-chan protocol.ScrapeJob) {
	slog.Info("worker started", "worker_id", w.id)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopping", "worker_id", w.id)
			return
		case job := <-jobChan:
			w.processJob(ctx, job)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job protocol.ScrapeJob) {
	logger := slog.With("worker_id", w.id, "request_id", job.RequestID, "route", job.Origin+"→"+job.Destination, "month", job.Month)
	logger.Info("processing job")

	// Generate dates for the month
	dates, err := datesInMonth(job.Month)
	if err != nil {
		logger.Error("invalid month", "error", err)
		return
	}

	// Create a browser context with anti-bot settings
	width, height := scraping.RandomViewport()
	contextOpts := playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(scraping.RandomUA()),
		Viewport: &playwright.Size{
			Width:  width,
			Height: height,
		},
		Locale:       playwright.String("en-US"),
		TimezoneId:   playwright.String(scraping.TimezoneForAirport(job.Origin)),
	}

	browserCtx, err := w.browser.NewContext(contextOpts)
	if err != nil {
		logger.Error("failed to create browser context", "error", err)
		return
	}
	defer browserCtx.Close()

	availableDates := 0
	failedDates := 0

	for _, date := range dates {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Anti-bot: random jitter between requests
		jitter := scraping.GaussianJitter(4.0, 1.5, 2.0, 8.0)
		time.Sleep(jitter)

		page, err := browserCtx.NewPage()
		if err != nil {
			logger.Error("failed to create page", "error", err)
			failedDates++
			continue
		}

		flights, _, err := scraping.ScrapeAlaskaAwards(page, job.Origin, job.Destination, date)
		page.Close()

		result := protocol.ScrapeResult{
			RequestID:   job.RequestID,
			Date:        date,
			Origin:      job.Origin,
			Destination: job.Destination,
			ScrapedAt:   time.Now(),
		}

		if err != nil {
			logger.Warn("scrape failed", "date", date, "error", err)
			result.Status = "error"
			failedDates++
		} else if len(flights) == 0 {
			result.Status = "no_flights"
			result.FlightCount = 0
		} else {
			result.Status = "success"
			result.FlightCount = len(flights)
			availableDates++

			// Find cheapest fare
			cheapest := findCheapest(flights)
			if cheapest != nil {
				result.Cheapest = cheapest
			}
		}

		w.resultChan <- result
	}

	logger.Info("job complete",
		"total", len(dates),
		"available", availableDates,
		"failed", failedDates,
	)
}

func findCheapest(flights []types.NormalizedFlight) *protocol.CheapestFare {
	var best *protocol.CheapestFare
	for _, f := range flights {
		for _, fare := range f.Fares {
			if best == nil || fare.Miles < best.Miles {
				best = &protocol.CheapestFare{
					Cabin: fare.Cabin,
					Miles: fare.Miles,
					Cash:  fare.Cash,
				}
			}
		}
	}
	return best
}

// datesInMonth returns all dates in a given month string "YYYY-MM".
func datesInMonth(month string) ([]string, error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, fmt.Errorf("parse month %q: %w", month, err)
	}

	var dates []string
	for d := t; d.Month() == t.Month(); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}
	return dates, nil
}
