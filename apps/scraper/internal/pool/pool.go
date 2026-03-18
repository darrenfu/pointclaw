package pool

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/darrenfu/pointclaw/scraper/internal/protocol"
	"github.com/darrenfu/pointclaw/scraper/internal/worker"
	"github.com/playwright-community/playwright-go"
)

// Pool manages a set of browser workers sharing a single Playwright browser instance.
type Pool struct {
	size       int
	resultChan chan<- protocol.ScrapeResult
}

func New(size int, resultChan chan<- protocol.ScrapeResult) *Pool {
	return &Pool{
		size:       size,
		resultChan: resultChan,
	}
}

// Run starts the browser and N worker goroutines.
func (p *Pool) Run(ctx context.Context, jobChan <-chan protocol.ScrapeJob) {
	// Install Playwright browsers if needed
	err := playwright.Install()
	if err != nil {
		slog.Error("failed to install playwright", "error", err)
		return
	}

	// Start Playwright
	pw, err := playwright.Run()
	if err != nil {
		slog.Error("failed to start playwright", "error", err)
		return
	}
	defer pw.Stop()

	// Launch browser
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-dev-shm-usage"},
	})
	if err != nil {
		slog.Error("failed to launch browser", "error", err)
		return
	}
	defer browser.Close()

	slog.Info("browser pool started", "workers", p.size)

	// Fan out to N workers
	// Each worker reads from the shared jobChan
	done := make(chan struct{})
	for i := 0; i < p.size; i++ {
		w := worker.New(i, browser, p.resultChan)
		go func(id int) {
			w.Run(ctx, jobChan)
			slog.Info(fmt.Sprintf("worker %d stopped", id))
			done <- struct{}{}
		}(i)
	}

	// Wait for all workers to stop
	for i := 0; i < p.size; i++ {
		<-done
	}
	slog.Info("browser pool stopped")
}
