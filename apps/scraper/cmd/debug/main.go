// debug is a standalone tool to diagnose Alaska scraping issues.
// Run: PLAYWRIGHT_BROWSERS_PATH=/tmp/pw-browsers go run ./cmd/debug
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	origin := "SEA"
	dest := "NRT"
	date := "2026-10-01"
	if len(os.Args) > 1 {
		date = os.Args[1]
	}

	slog.Info("starting debug scrape", "origin", origin, "dest", dest, "date", date)

	pw, err := playwright.Run()
	if err != nil {
		slog.Error("failed to start playwright", "error", err)
		os.Exit(1)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), // Headless mode for sandbox/CI
		Args:     []string{"--no-sandbox", "--disable-dev-shm-usage"},
	})
	if err != nil {
		slog.Error("failed to launch browser", "error", err)
		os.Exit(1)
	}
	defer browser.Close()

	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
		Viewport:  &playwright.Size{Width: 1440, Height: 900},
		Locale:    playwright.String("en-US"),
	})
	if err != nil {
		slog.Error("failed to create context", "error", err)
		os.Exit(1)
	}
	defer ctx.Close()

	page, err := ctx.NewPage()
	if err != nil {
		slog.Error("failed to create page", "error", err)
		os.Exit(1)
	}

	// Log ALL network requests and responses
	page.OnRequest(func(req playwright.Request) {
		if strings.Contains(req.URL(), "search") || strings.Contains(req.URL(), "bff") {
			slog.Info(">>> REQUEST", "method", req.Method(), "url", req.URL())
		}
	})

	page.OnResponse(func(resp playwright.Response) {
		url := resp.URL()
		if strings.Contains(url, "search") || strings.Contains(url, "bff") || strings.Contains(url, "api") {
			contentType := ""
			headers := resp.Headers()
			if ct, ok := headers["content-type"]; ok {
				contentType = ct
			}
			slog.Info("<<< RESPONSE", "status", resp.Status(), "url", url, "content_type", contentType)

			// If it looks like JSON from searchbff, dump it
			if strings.Contains(url, "searchbff") {
				body, err := resp.Body()
				if err != nil {
					slog.Error("failed to read body", "error", err)
					return
				}
				slog.Info("searchbff body", "length", len(body))

				// Pretty print first 1000 chars
				preview := string(body)
				if len(preview) > 1000 {
					preview = preview[:1000] + "..."
				}
				fmt.Println("=== SEARCHBFF RESPONSE BODY ===")
				fmt.Println(preview)
				fmt.Println("=== END ===")

				// Try to parse as AlaskaResponse
				var raw map[string]interface{}
				if err := json.Unmarshal(body, &raw); err != nil {
					slog.Warn("not valid JSON", "error", err)
				} else {
					if slices, ok := raw["slices"]; ok {
						if arr, ok := slices.([]interface{}); ok {
							slog.Info("parsed response", "slice_count", len(arr))
						}
					} else {
						slog.Warn("response has no 'slices' key", "keys", getKeys(raw))
					}
				}
			}
		}
	})

	// Try all URL patterns
	urls := []struct {
		name string
		url  string
	}{
		{
			"calendar",
			fmt.Sprintf("https://www.alaskaair.com/search/calendar?O=%s&D=%s&OD=%s&A=1&RT=false&RequestType=Calendar&ShoppingMethod=onlineaward&locale=en-us&FareType=Lowest+price+available", origin, dest, date),
		},
		{
			"results",
			fmt.Sprintf("https://www.alaskaair.com/search/results?O=%s&D=%s&OD=%s&A=1&RT=false&ShoppingMethod=onlineaward", origin, dest, date),
		},
	}

	for _, u := range urls {
		fmt.Printf("\n========================================\n")
		fmt.Printf("TRYING: %s\n", u.name)
		fmt.Printf("URL: %s\n", u.url)
		fmt.Printf("========================================\n\n")

		_, err := page.Goto(u.url, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
			Timeout:   playwright.Float(30000),
		})
		if err != nil {
			slog.Error("navigation failed", "pattern", u.name, "error", err)
			continue
		}

		// Log where we ended up
		slog.Info("landed on", "url", page.URL())
		title, _ := page.Title()
		slog.Info("page title", "title", title)

		// Wait for potential XHR
		slog.Info("waiting 10s for XHR responses...")
		time.Sleep(10 * time.Second)

		// Screenshot
		screenshotPath := fmt.Sprintf("/tmp/pointclaw-debug-%s.png", u.name)
		_, err = page.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(screenshotPath),
			FullPage: playwright.Bool(true),
		})
		if err != nil {
			slog.Error("screenshot failed", "error", err)
		} else {
			slog.Info("screenshot saved", "path", screenshotPath)
		}

		// Dump HTML snippet
		content, _ := page.Content()
		if len(content) > 2000 {
			content = content[:2000]
		}
		fmt.Printf("=== PAGE HTML (first 2000 chars) ===\n%s\n=== END ===\n", content)
	}

	slog.Info("debug complete — check /tmp/pointclaw-debug-*.png")
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
