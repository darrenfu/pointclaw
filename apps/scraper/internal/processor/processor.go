package processor

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/darrenfu/pointclaw/scraper/internal/protocol"
)

// Processor receives scrape results and handles persistence + notification.
type Processor struct {
	redis       *protocol.RedisClient
	databaseURL string
	cacheTTL    time.Duration
}

func New(redis *protocol.RedisClient, databaseURL string) *Processor {
	return &Processor{
		redis:       redis,
		databaseURL: databaseURL,
		cacheTTL:    60 * time.Second,
	}
}

// Run processes results from the result channel.
func (p *Processor) Run(ctx context.Context, resultChan <-chan protocol.ScrapeResult) {
	slog.Info("result processor started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("result processor stopping")
			return
		case result := <-resultChan:
			p.processResult(ctx, result)
		}
	}
}

func (p *Processor) processResult(ctx context.Context, result protocol.ScrapeResult) {
	logger := slog.With(
		"request_id", result.RequestID,
		"date", result.Date,
		"status", result.Status,
	)

	// 1. Write to Redis cache
	data, err := json.Marshal(result)
	if err != nil {
		logger.Error("failed to marshal result", "error", err)
		return
	}

	if err := p.redis.SetCache(ctx, result.Origin, result.Destination, result.Date, data, p.cacheTTL); err != nil {
		logger.Warn("failed to write cache", "error", err)
	}

	// 2. Write to Postgres
	// TODO: implement DB persistence via pgx
	// - INSERT INTO award_searches (origin_code, dest_code, search_date, searched_at, status)
	// - INSERT INTO award_flights (search_id, flight_number, ...) for each flight

	// 3. Publish result to Redis Pub/Sub (FE listens here)
	if err := p.redis.PublishResult(ctx, result); err != nil {
		logger.Warn("failed to publish result", "error", err)
	}

	logger.Info("result processed",
		"flights", result.FlightCount,
		"origin", result.Origin,
		"dest", result.Destination,
	)
}
