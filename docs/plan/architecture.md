# Smart Booking — System Architecture

Last updated: 2026-03-17

## 1. System Overview

```
┌────────────────────────────────────────────────────────────────────────────┐
│                              BROWSER (User)                                │
│   Route Picker → Calendar Heatmap → Flight List                            │
└───────────────────────────────┬────────────────────────────────────────────┘
                                │
                                ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                      FRONTEND — Next.js (self-hosted)                      │
│                                                                            │
│  On search request:                                                        │
│  1. Check Redis cache ──── hit? ──── return immediately                    │
│  2. Check Postgres    ──── fresh (<60s)? ──── return from DB               │
│  3. Publish scrape job to Redis Stream                                     │
│  4. Subscribe to Redis Pub/Sub for results                                 │
│  5. Stream results to browser via SSE                                      │
│                                                                            │
│  Direct access to:  Redis (read cache)  +  Postgres (read DB)              │
└──────────┬────────────────────┬────────────────────┬───────────────────────┘
           │                    │                    │
           │ publish job        │ read cache         │ read/write DB
           ▼                    ▼                    ▼
┌─────────────────────────────────────────┐  ┌──────────────────────────────┐
│          REDIS (local, self-hosted)      │  │     POSTGRES (Supabase)      │
│                                         │  │                              │
│  Streams:                               │  │  airports, routes            │
│  • scrape:jobs (job queue)              │  │  award_searches              │
│                                         │  │  award_flights               │
│  Pub/Sub:                               │  │  scrape_jobs                 │
│  • scrape:results:{id}                  │  │                              │
│                                         │  │                              │
│  Keys:                                  │  │                              │
│  • cache:{origin}:{dest}:{date} (60s)   │  │                              │
│  • scrape:lock:{origin}:{dest}:{date}   │  │                              │
└──────────┬──────────────────────────────┘  └──────────────┬───────────────┘
           │                                                │
           │ consume jobs              write results        │
           │ publish results           write cache          │
           ▼                                                │
┌──────────────────────────────────────────────────────────────────────────┐
│                     SCRAPER — Go Actor System (self-hosted)               │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐    │
│  │                        Supervisor                                │    │
│  │  • Monitors all goroutines, restarts on panic                    │    │
│  └──────────┬──────────────────┬──────────────────┬─────────────────┘    │
│             │                  │                  │                      │
│             ▼                  ▼                  ▼                      │
│  ┌────────────────┐  ┌──────────────┐  ┌───────────────────┐           │
│  │   Dispatcher   │  │ Pool Manager │  │ Result Processor  │           │
│  │                │  │              │  │                   │           │
│  │ • XREAD from   │  │ • Manages N  │  │ • Normalize data  │           │
│  │   scrape:jobs  │  │   Browser    │  │ • Write Postgres  │           │
│  │ • Deduplicate  │  │   Workers    │  │ • Write Redis     │           │
│  │ • Batch dates  │  │ • Goroutine  │  │   cache           │           │
│  │   into optimal │  │   pool       │  │ • PUBLISH result  │           │
│  │   groups       │  │ • Health     │  │   to channel      │           │
│  │                │  │   checks     │  │                   │           │
│  └───────┬────────┘  └──────┬───────┘  └───────────────────┘           │
│          │                  │                                           │
│          │            ┌─────┴──────┐                                    │
│          │            ▼            ▼                                    │
│          │   ┌──────────────┐ ┌──────────────┐                         │
│          │   │  Browser     │ │  Browser     │  (N workers, tunable)   │
│          │   │  Worker #1   │ │  Worker #N   │                         │
│          └──▶│              │ │              │                         │
│              │ • Owns 1     │ │ • Owns 1     │                         │
│              │   Playwright │ │   Playwright │                         │
│              │   context    │ │   context    │                         │
│              │ • Anti-bot   │ │ • Anti-bot   │                         │
│              │   (jitter,   │ │   (jitter,   │                         │
│              │    UA rotate,│ │    UA rotate,│                         │
│              │    backoff)  │ │    backoff)  │                         │
│              └──────┬───────┘ └──────┬───────┘                         │
│                     │                │                                  │
│                     └───────┬────────┘                                  │
│                             ▼                                           │
│                    ┌─────────────────┐                                  │
│                    │  Chromium       │────────▶ Alaska Airlines         │
│                    │  (playwright-go)│◀──────── searchbff/V3 JSON      │
│                    └─────────────────┘                                  │
└──────────────────────────────────────────────────────────────────────────┘
```

## 2. Tech Stack (Final)

| Layer | Technology | Hosting (MVP) |
|-------|-----------|---------------|
| Frontend | Next.js 14, React, Tailwind, TanStack Query | Self-hosted |
| Scraper | **Go**, playwright-go (fallback: rod) | Self-hosted |
| Message Bus | **Redis Streams + Pub/Sub** | Local Redis |
| Cache | **Redis** (keys with 60s TTL) | Local Redis |
| Database | Postgres + pgvector, Drizzle ORM | Supabase (free tier) |
| Monorepo | Turborepo + pnpm (FE), Go module (scraper) | — |

## 3. Storage Strategy (Write-Through)

```
                    ┌──────────────┐
                    │ User Request │
                    └──────┬───────┘
                           │
                    FE reads directly:
                           │
                    ┌──────▼──────┐
                    │ Redis Cache  │──── HIT & fresh? ──▶ Return immediately
                    └──────┬──────┘
                           │ MISS
                    ┌──────▼──────┐
                    │  Postgres   │──── Recent (<60s)? ──▶ Return from DB
                    └──────┬──────┘
                           │ MISS
                    ┌──────▼──────┐
                    │ Publish to  │──── Redis Stream ──▶ Scraper picks up
                    │ scrape:jobs │
                    └──────┬──────┘
                           │
                    Subscribe to scrape:results:{id}
                           │
                    ┌──────▼──────┐
                    │ Scraper     │
                    │ completes   │
                    └──────┬──────┘
                           │
                  ┌────────┴────────┐
                  ▼                 ▼
         ┌──────────────┐  ┌──────────────┐
         │ Redis Cache  │  │  Postgres    │
         │ (60s TTL)    │  │  (permanent) │
         └──────────────┘  └──────────────┘
                  │
                  ▼
         ┌──────────────┐
         │ Redis Pub/Sub│──▶ FE receives, streams to browser via SSE
         └──────────────┘
```

## 4. Go Scraper Architecture

### 4.1 Go Tech Stack

| Concern | Library | Notes |
|---------|---------|-------|
| Browser automation | `playwright-go` | Playwright bindings for Go. Fallback: `rod` (CDP native) |
| Redis | `go-redis/redis/v9` | Streams, Pub/Sub, cache keys |
| Postgres | `jackc/pgx/v5` | Fastest Go Postgres driver |
| HTTP server | `net/http` or `chi` | Health endpoint, metrics |
| JSON | `encoding/json` | stdlib, fast enough |
| Retry/backoff | `cenkalti/backoff/v4` | Exponential backoff with jitter |
| Logging | `slog` | Go stdlib structured logging (Go 1.21+) |
| Config | `caarlos0/env` | Env var parsing |
| Concurrency | goroutines + channels | Native Go, no framework needed |

### 4.2 Actor System via Goroutines + Channels

Go doesn't need an actor framework — goroutines + channels **are** the actor model:

```
                    ┌─────────────────────────────┐
                    │         main()               │
                    │  Start all goroutines         │
                    │  Monitor with errgroup        │
                    └──────────┬──────────────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
     ┌──────────────┐  ┌────────────┐  ┌──────────────┐
     │  Dispatcher  │  │   Pool     │  │  Processor   │
     │  goroutine   │  │  Manager   │  │  goroutine   │
     │              │  │  goroutine │  │              │
     │  XREAD loop  │  │            │  │  Receives    │
     │  ──────────▶ │  │  jobChan   │  │  from        │
     │  Dedup       │  │ ─────────▶ │  │  resultChan  │
     │  Batch       │  │  Dispatch  │  │  ──────────▶ │
     │  ──────────▶ │  │  to idle   │  │  Normalize   │
     │  jobChan     │  │  worker    │  │  Write DB    │
     │              │  │            │  │  Write Redis │
     └──────┬───────┘  └─────┬──────┘  │  Publish     │
            │                │         └──────────────┘
            │          ┌─────┴──────┐
            │          ▼            ▼
            │  ┌────────────┐ ┌────────────┐
            │  │  Worker    │ │  Worker    │
            │  │  goroutine │ │  goroutine │
            └─▶│  #1        │ │  #N        │
               │            │ │            │
               │ playwright │ │ playwright │
               │ context    │ │ context    │
               │ ─────────▶ │ │ ─────────▶ │
               │ resultChan │ │ resultChan │
               └────────────┘ └────────────┘

     Channels:
     jobChan    chan ScrapeJob    (Dispatcher → Pool → Workers)
     resultChan chan ScrapeResult (Workers → Processor)
```

### 4.3 Job Batching Strategy

**Goal:** Minimize job count and scraped URL count.

Alaska's calendar page likely fetches data differently than individual date searches. We should test and tune:

| Strategy | Jobs | URLs Hit | Latency |
|----------|------|----------|---------|
| **1 job per date** | 30/month | 30 URLs | High (30 × 5-10s, even with parallelism) |
| **1 job per month** (calendar URL) | 1/month | 1 URL | Best if calendar returns all dates |
| **Batched by week** | 4-5/month | 4-5 URLs | Good balance |
| **Adaptive** | Varies | Varies | Best — test to find optimal batch size |

**Approach:** Start with the calendar URL you provided:
```
https://www.alaskaair.com/search/calendar?O={origin}&D={dest}&OD={month-start}
  &A=1&RT=false&RequestType=Calendar&ShoppingMethod=onlineaward
```

If this returns all dates in a month in a single page load, we only need **1 scrape per route per month** instead of 30. That's a 30x reduction.

**Testing plan:**
1. Load calendar URL via playwright-go
2. Intercept all XHR responses
3. See if we get a month's worth of data in 1 response or multiple
4. Tune batch size based on what Alaska's calendar endpoint returns
5. If calendar gives full month → 1 job = 1 month = 1 URL
6. If calendar paginates → batch by what the calendar returns per page

### 4.4 Anti-Bot Techniques

| Technique | Implementation |
|-----------|---------------|
| **UA rotation** | Pool of 10+ Chrome UA strings, rotate per context |
| **Viewport randomization** | Random 1280-1920 × 720-1080 per context |
| **Jitter** | Gaussian random delay 2-8s between requests |
| **Exponential backoff** | `cenkalti/backoff`: 2^n × 1s + random jitter, cap 5 min |
| **Circuit breaker** | 5 consecutive failures → pause 15 min |
| **Request budget** | Cap ~500 scrapes/day |
| **Block tracking domains** | 18+ domains (AwardWiz list) via `page.Route()` |
| **Session reuse** | Keep browser contexts alive, reuse cookies |
| **Timezone matching** | Set context TZ to match origin airport |
| **Referrer chain** | Navigate home → search → calendar (not deep-link) |

### 4.5 Deduplication

```
FE publishes job:
  SETNX scrape:lock:{origin}:{dest}:{month} {request_id} EX 30
  → Acquired? Publish to scrape:jobs
  → Exists?   Subscribe to existing result channel instead

Scraper Dispatcher:
  Maintain in-memory dedup set (origin:dest:month → request_id)
  TTL: 60s
  Duplicate? Skip, subscriber will get results from original job
```

## 5. Communication Protocol

### 5.1 Message Schemas

**Scrape Request** (FE → Redis Stream `scrape:jobs`)
```json
{
  "request_id": "uuid-v4",
  "origin": "SEA",
  "destination": "NRT",
  "month": "2026-06",
  "priority": "normal",
  "requested_at": "2026-03-17T20:30:00Z"
}
```

**Scrape Result** (Scraper → Redis Pub/Sub `scrape:results:{request_id}`)
```json
{
  "request_id": "uuid-v4",
  "date": "2026-06-01",
  "status": "success",
  "origin": "SEA",
  "destination": "NRT",
  "cheapest": { "cabin": "economy", "miles": 25000, "cash": 5.60 },
  "flight_count": 3,
  "scraped_at": "2026-03-17T20:30:05Z"
}
```

**Job Completion** (Scraper → Redis Pub/Sub `scrape:results:{request_id}`)
```json
{
  "request_id": "uuid-v4",
  "status": "done",
  "total_dates": 30,
  "available_dates": 18,
  "failed_dates": 0
}
```

### 5.2 Protocol Flow

```
User searches SEA → NRT, June 2026

1. Browser → Next.js: GET /api/search/calendar?origin=SEA&dest=NRT&month=2026-06
2. Next.js → Redis: GET cache:SEA:NRT:2026-06-* (check all dates)
3. Cache miss → Next.js → Redis: SETNX scrape:lock:SEA:NRT:2026-06 (dedup)
4. Lock acquired → Next.js → Redis: XADD scrape:jobs {request}
5. Next.js → Redis: SUBSCRIBE scrape:results:{request_id}
6. Next.js → Browser: Open SSE stream

7. Go Scraper: XREAD scrape:jobs → receives job
8. Scraper → Playwright: Load calendar URL for SEA→NRT June 2026
9. Playwright → Alaska: Navigate, intercept searchbff responses
10. Scraper: Normalize each date's data
11. Scraper → Redis: SET cache:SEA:NRT:2026-06-01 (60s TTL)
12. Scraper → Postgres: INSERT award_searches + award_flights
13. Scraper → Redis: PUBLISH scrape:results:{request_id} {date result}
14. (Repeat for each date in response)
15. Scraper → Redis: PUBLISH scrape:results:{request_id} {done}

16. Next.js receives Pub/Sub → SSE to browser
17. Browser renders calendar cells progressively
```

## 6. Database Schema (UML)

### Phase 1 (MVP)

```
┌──────────────────┐     ┌──────────────────┐
│    airports       │     │     routes        │
├──────────────────┤     ├──────────────────┤
│ code VARCHAR(3)  │PK   │ id SERIAL        │PK
│ name TEXT        │     │ origin_code      │FK → airports
│ city TEXT        │     │ dest_code        │FK → airports
│ country VARCHAR(2)│    │ is_active BOOL   │
│ region TEXT      │     └──────────────────┘
│ latitude FLOAT   │
│ longitude FLOAT  │     ┌──────────────────┐     ┌───────────────────┐
│ is_origin BOOL   │     │  award_searches  │     │  award_flights    │
└──────────────────┘     ├──────────────────┤     ├───────────────────┤
                         │ id SERIAL        │PK   │ id SERIAL         │PK
                         │ origin_code      │     │ search_id         │FK
                         │ dest_code        │     │ flight_number TEXT │
                         │ search_date DATE │     │ carrier_code      │
                         │ searched_at TS   │     │ carrier_name TEXT  │
                         │ raw_response JSONB│    │ origin VARCHAR(3)  │
                         │ status VARCHAR   │     │ destination        │
                         └────────┬─────────┘     │ departure_time TS  │
                                  │               │ arrival_time TS    │
                                  │ 1:N           │ duration INT (min) │
                                  └──────────────▶│ aircraft TEXT      │
                                                  │ cabin VARCHAR(20)  │
                                                  │ miles_cost INT     │
                                                  │ cash_cost FLOAT    │
                                                  │ seats_remaining INT│
                                                  │ booking_code       │
                                                  │ is_saver BOOL      │
                                                  │ is_direct BOOL     │
                                                  │ amenities JSONB    │
                                                  └───────────────────┘
```

### Phase 2 (Alerts) / Phase 3 (Personalization) — see `design-scraper-protocol.md`

## 7. Frontend

### Pages

| Route | Purpose |
|-------|---------|
| `/` | Landing — route picker, quick search |
| `/search` | Calendar heatmap + flight list |

### Components

```
Landing Page                    Search Page
┌──────────────────┐           ┌──────────────────────────────┐
│  RoutePicker     │           │  MonthNav (◀ June 2026 ▶)    │
│  ┌────────────┐  │           │                              │
│  │AirportSearch│  │    ┌────▶│  CalendarHeatmap             │
│  │ From: SEA  │  │    │     │  ┌──┬──┬──┬──┬──┬──┬──┐     │
│  └────────────┘  │    │     │  │  │  │██│░░│██│██│██│     │
│  ┌────────────┐  │    │     │  │  │  │25│--│30│25│35│     │
│  │AirportSearch│  │    │     │  ├──┼──┼──┼──┼──┼──┼──┤     │
│  │ To:   NRT  │  │    │     │  │██│░░│██│..│..│..│..│     │
│  └────────────┘  │    │     │  │40│--│25│  loading  │     │
│                  │    │     │  └──┴──┴──┴──┴──┴──┴──┘     │
│  [Search] ───────┼────┘     │                              │
│                  │          │  CabinFilter [Econ|Biz|1st]   │
└──────────────────┘          │                              │
                              │  FlightList (click a date)   │
                              │  ┌──────────────────────┐    │
                              │  │ FlightCard           │    │
                              │  │ AS 123  SEA→NRT      │    │
                              │  │ 10:30am → 2:30pm+1   │    │
                              │  │ 25,000 mi + $5.60    │    │
                              │  │ 5 seats · Economy    │    │
                              │  └──────────────────────┘    │
                              └──────────────────────────────┘
```

## 8. Project Structure

```
smart-booking/alaska/
├── apps/
│   ├── web/                      # Next.js frontend
│   │   ├── app/
│   │   │   ├── page.tsx          # Landing
│   │   │   ├── search/page.tsx   # Calendar + list
│   │   │   └── api/
│   │   │       ├── search/       # Redis/DB reads + job publish
│   │   │       └── airports/     # Autocomplete
│   │   ├── components/
│   │   │   ├── CalendarHeatmap.tsx
│   │   │   ├── FlightList.tsx
│   │   │   ├── FlightCard.tsx
│   │   │   ├── RoutePicker.tsx
│   │   │   ├── AirportSearch.tsx
│   │   │   ├── MonthNav.tsx
│   │   │   └── CabinFilter.tsx
│   │   └── lib/
│   │       ├── redis.ts          # Redis client (ioredis)
│   │       ├── db.ts             # Drizzle client
│   │       └── types.ts
│   │
│   └── scraper/                  # Go scraper subsystem
│       ├── go.mod
│       ├── go.sum
│       ├── cmd/
│       │   └── scraper/
│       │       └── main.go       # Entry point
│       ├── internal/
│       │   ├── config/
│       │   │   └── config.go     # Env config
│       │   ├── dispatcher/
│       │   │   └── dispatcher.go # Redis stream consumer, dedup
│       │   ├── pool/
│       │   │   └── pool.go       # Browser worker pool
│       │   ├── worker/
│       │   │   └── worker.go     # Single browser worker
│       │   ├── processor/
│       │   │   └── processor.go  # Normalize, persist, publish
│       │   ├── scraping/
│       │   │   ├── alaska.go     # Alaska-specific scrape logic
│       │   │   └── antibot.go    # Anti-detection utilities
│       │   ├── protocol/
│       │   │   ├── messages.go   # Message schemas
│       │   │   └── redis.go      # Redis stream/pubsub ops
│       │   └── storage/
│       │       ├── cache.go      # Redis cache ops
│       │       └── db.go         # Postgres ops (pgx)
│       ├── pkg/
│       │   └── types/
│       │       └── alaska.go     # AlaskaResponse types
│       ├── Dockerfile
│       └── Makefile
│
├── packages/
│   └── shared/                   # Shared constants
│       ├── airports.ts           # Airport code database
│       └── routes.ts             # Supported route pairs
│
├── db/
│   ├── schema.ts                 # Drizzle schema
│   ├── migrations/
│   └── seed.ts                   # Seed airports + routes
│
├── docker-compose.yml            # Redis + scraper for local dev
├── turbo.json
├── package.json
└── .env.example
```

## 9. Phase Roadmap

| Phase | Features | Tech |
|-------|----------|------|
| **Phase 1 (MVP)** | Alaska award search, calendar heatmap, flight list | Next.js + Go + Redis + Supabase |
| **Phase 2** | Email alerts, more airlines | Add alert tables, cron, email service |
| **Phase 3** | User accounts, personalization, credit cards, hotels | Supabase Auth, pgvector |

## 10. Open Items

- [ ] Test calendar URL: does it return full month data in 1 page load?
- [ ] Verify playwright-go can intercept searchbff/V3 responses
- [ ] Tune optimal batch size (1 URL per month vs per week vs per day)
- [ ] Set up docker-compose.yml for local Redis
- [ ] Scaffold Go module and Next.js project
