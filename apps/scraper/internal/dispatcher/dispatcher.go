package dispatcher

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/darrenfu/pointclaw/scraper/internal/protocol"
)

// Dispatcher consumes scrape jobs from the Redis stream and routes them to workers.
type Dispatcher struct {
	redis    *protocol.RedisClient
	jobChan  chan<- protocol.ScrapeJob
	seen     map[string]time.Time // dedup: origin:dest:month → last seen
	seenMu   sync.Mutex
	dedupTTL time.Duration
}

func New(redis *protocol.RedisClient, jobChan chan<- protocol.ScrapeJob) *Dispatcher {
	return &Dispatcher{
		redis:    redis,
		jobChan:  jobChan,
		seen:     make(map[string]time.Time),
		dedupTTL: 60 * time.Second,
	}
}

func (d *Dispatcher) Run(ctx context.Context) error {
	// Ensure consumer group exists
	if err := d.redis.EnsureConsumerGroup(ctx); err != nil {
		return err
	}

	slog.Info("dispatcher started, consuming from Redis stream")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		jobs, err := d.redis.ReadJobs(ctx, "dispatcher-1")
		if err != nil {
			slog.Error("read jobs failed", "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, job := range jobs {
			if d.isDuplicate(job) {
				slog.Info("skipping duplicate job",
					"origin", job.Origin,
					"dest", job.Destination,
					"month", job.Month,
				)
				continue
			}
			d.markSeen(job)

			slog.Info("dispatching job",
				"request_id", job.RequestID,
				"origin", job.Origin,
				"dest", job.Destination,
				"month", job.Month,
			)
			d.jobChan <- job
		}
	}
}

func (d *Dispatcher) isDuplicate(job protocol.ScrapeJob) bool {
	key := job.Origin + ":" + job.Destination + ":" + job.Month
	d.seenMu.Lock()
	defer d.seenMu.Unlock()

	if lastSeen, ok := d.seen[key]; ok && time.Since(lastSeen) < d.dedupTTL {
		return true
	}
	return false
}

func (d *Dispatcher) markSeen(job protocol.ScrapeJob) {
	key := job.Origin + ":" + job.Destination + ":" + job.Month
	d.seenMu.Lock()
	defer d.seenMu.Unlock()
	d.seen[key] = time.Now()
}
