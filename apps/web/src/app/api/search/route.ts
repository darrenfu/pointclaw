import { NextRequest, NextResponse } from "next/server";
import { getRedis, cacheKey, lockKey, resultChannel, STREAM_KEY } from "@/lib/redis";
import type { CalendarData, DateResult } from "@/lib/types";

// GET /api/search?origin=SEA&dest=NRT&month=2026-06
// GET /api/search?origin=SEA&dest=NRT&date=2026-06-15
export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const origin = searchParams.get("origin");
  const dest = searchParams.get("dest");
  const month = searchParams.get("month");
  const date = searchParams.get("date");

  if (!origin || !dest) {
    return NextResponse.json({ error: "origin and dest required" }, { status: 400 });
  }

  const redis = getRedis();

  // Single date query — return flights from cache
  if (date) {
    return handleDateQuery(redis, origin, dest, date);
  }

  // Month query — return calendar data from cache
  if (month) {
    return handleMonthQuery(redis, origin, dest, month);
  }

  return NextResponse.json({ error: "month or date required" }, { status: 400 });
}

async function handleDateQuery(redis: ReturnType<typeof getRedis>, origin: string, dest: string, date: string) {
  // Check for full flight data (set by scraper or seed)
  const flightKey = `flights:${origin}:${dest}:${date}`;
  const flightData = await redis.get(flightKey);
  if (flightData) {
    try {
      return NextResponse.json(JSON.parse(flightData));
    } catch {
      // fall through
    }
  }

  // Check calendar cache entry
  const key = cacheKey(origin, dest, date);
  const cached = await redis.get(key);
  if (cached) {
    try {
      const data = JSON.parse(cached);
      return NextResponse.json({ cached: true, ...data });
    } catch {
      // fall through
    }
  }

  // No cache — return empty, FE can trigger a scrape job
  return NextResponse.json({
    cached: false,
    flights: [],
    message: "No cached data. Scrape job may be needed.",
  });
}

async function handleMonthQuery(redis: ReturnType<typeof getRedis>, origin: string, dest: string, month: string) {
  const [year, mon] = month.split("-").map(Number);
  const daysInMonth = new Date(year, mon, 0).getDate();

  const calendar: CalendarData = {};
  const pipeline = redis.pipeline();

  // Batch fetch all dates in the month from cache
  for (let d = 1; d <= daysInMonth; d++) {
    const dateStr = `${year}-${String(mon).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
    pipeline.get(cacheKey(origin, dest, dateStr));
  }

  const results = await pipeline.exec();

  let hasData = false;
  for (let d = 1; d <= daysInMonth; d++) {
    const dateStr = `${year}-${String(mon).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
    const result = results?.[d - 1];

    if (result && result[1]) {
      try {
        const raw = JSON.parse(result[1] as string);
        // Transform snake_case keys from Go scraper to camelCase for TypeScript
        const data: DateResult = {
          date: raw.date,
          status: raw.status,
          cheapest: raw.cheapest ?? null,
          flightCount: raw.flight_count ?? 0,
        };
        calendar[dateStr] = data;
        hasData = true;
      } catch {
        calendar[dateStr] = { date: dateStr, status: "loading", cheapest: null, flightCount: 0 };
      }
    } else {
      calendar[dateStr] = { date: dateStr, status: "loading", cheapest: null, flightCount: 0 };
    }
  }

  // If no data in cache, publish a scrape job
  if (!hasData) {
    const requestId = crypto.randomUUID();
    const lock = lockKey(origin, dest, month);

    // Try to acquire lock (dedup)
    const acquired = await redis.set(lock, requestId, "EX", 30, "NX");
    if (acquired) {
      // Publish job to Redis stream
      const job = JSON.stringify({
        request_id: requestId,
        origin,
        destination: dest,
        month,
        priority: "normal",
        requested_at: new Date().toISOString(),
      });
      await redis.xadd(STREAM_KEY, "*", "data", job);
    }
  }

  return NextResponse.json({ calendar, hasData });
}
