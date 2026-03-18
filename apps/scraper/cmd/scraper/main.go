package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/darrenfu/pointclaw/scraper/internal/config"
	"github.com/darrenfu/pointclaw/scraper/internal/dispatcher"
	"github.com/darrenfu/pointclaw/scraper/internal/pool"
	"github.com/darrenfu/pointclaw/scraper/internal/processor"
	"github.com/darrenfu/pointclaw/scraper/internal/protocol"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Initialize Redis client
	redisClient := protocol.NewRedisClient(cfg.RedisAddr)
	defer redisClient.Close()

	// Initialize channels (actor mailboxes)
	jobChan := make(chan protocol.ScrapeJob, 100)
	resultChan := make(chan protocol.ScrapeResult, 100)

	// Start Result Processor (writes to DB + Redis cache + publishes results)
	proc := processor.New(redisClient, cfg.DatabaseURL)
	go proc.Run(ctx, resultChan)

	// Start Browser Worker Pool
	browserPool := pool.New(cfg.PoolSize, resultChan)
	go browserPool.Run(ctx, jobChan)

	// Start Dispatcher (consumes Redis stream, feeds job channel)
	disp := dispatcher.New(redisClient, jobChan)

	slog.Info("scraper started",
		"pool_size", cfg.PoolSize,
		"redis", cfg.RedisAddr,
	)

	// Dispatcher blocks until context is cancelled
	if err := disp.Run(ctx); err != nil {
		slog.Error("dispatcher error", "error", err)
	}

	slog.Info("scraper shutting down")
}
