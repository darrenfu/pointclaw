import Redis from "ioredis";

const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";

// Singleton Redis client for server-side use
let redis: Redis | null = null;

export function getRedis(): Redis {
  if (!redis) {
    redis = new Redis(redisUrl);
  }
  return redis;
}

// Cache key helpers
export function cacheKey(origin: string, dest: string, date: string): string {
  return `cache:${origin}:${dest}:${date}`;
}

export function lockKey(origin: string, dest: string, month: string): string {
  return `scrape:lock:${origin}:${dest}:${month}`;
}

export function resultChannel(requestId: string): string {
  return `scrape:results:${requestId}`;
}

export const STREAM_KEY = "scrape:jobs";
