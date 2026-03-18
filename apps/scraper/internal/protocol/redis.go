package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	StreamKey     = "scrape:jobs"
	ConsumerGroup = "scraper-workers"
	CachePrefix   = "cache"
	LockPrefix    = "scrape:lock"
	ResultPrefix  = "scrape:results"
	StatusPrefix  = "scrape:status"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr string) *RedisClient {
	return &RedisClient{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

func (r *RedisClient) Client() *redis.Client {
	return r.client
}

// EnsureConsumerGroup creates the consumer group if it doesn't exist
func (r *RedisClient) EnsureConsumerGroup(ctx context.Context) error {
	err := r.client.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("create consumer group: %w", err)
	}
	return nil
}

// ReadJobs reads from the Redis stream (blocking)
func (r *RedisClient) ReadJobs(ctx context.Context, consumerName string) ([]ScrapeJob, error) {
	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: consumerName,
		Streams:  []string{StreamKey, ">"},
		Count:    1,
		Block:    5 * time.Second,
	}).Result()

	if err == redis.Nil {
		return nil, nil // timeout, no new messages
	}
	if err != nil {
		return nil, fmt.Errorf("xreadgroup: %w", err)
	}

	var jobs []ScrapeJob
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			data, ok := msg.Values["data"].(string)
			if !ok {
				slog.Warn("invalid message format", "id", msg.ID)
				continue
			}
			var job ScrapeJob
			if err := json.Unmarshal([]byte(data), &job); err != nil {
				slog.Warn("failed to parse job", "id", msg.ID, "error", err)
				continue
			}
			jobs = append(jobs, job)

			// Acknowledge the message
			r.client.XAck(ctx, StreamKey, ConsumerGroup, msg.ID)
		}
	}
	return jobs, nil
}

// PublishResult publishes a scrape result to the request's Pub/Sub channel
func (r *RedisClient) PublishResult(ctx context.Context, result ScrapeResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	channel := fmt.Sprintf("%s:%s", ResultPrefix, result.RequestID)
	return r.client.Publish(ctx, channel, data).Err()
}

// PublishCompletion publishes job completion to the request's Pub/Sub channel
func (r *RedisClient) PublishCompletion(ctx context.Context, completion JobCompletion) error {
	data, err := json.Marshal(completion)
	if err != nil {
		return fmt.Errorf("marshal completion: %w", err)
	}
	channel := fmt.Sprintf("%s:%s", ResultPrefix, completion.RequestID)
	return r.client.Publish(ctx, channel, data).Err()
}

// SetCache stores a result in Redis cache with TTL
func (r *RedisClient) SetCache(ctx context.Context, origin, dest, date string, data []byte, ttl time.Duration) error {
	key := fmt.Sprintf("%s:%s:%s:%s", CachePrefix, origin, dest, date)
	return r.client.Set(ctx, key, data, ttl).Err()
}

// GetCache retrieves a cached result
func (r *RedisClient) GetCache(ctx context.Context, origin, dest, date string) ([]byte, error) {
	key := fmt.Sprintf("%s:%s:%s:%s", CachePrefix, origin, dest, date)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return data, err
}
