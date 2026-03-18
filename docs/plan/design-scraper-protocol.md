# Scraper Subsystem & Communication Protocol Design

Last updated: 2026-03-17

## 1. Architectural Shift

**Before:** FE → Scraper (request-response proxy) → Alaska → DB
**After:** FE and Scraper are fully decoupled subsystems communicating via a message bus

```
┌──────────────────────┐          ┌──────────────────────┐
│   Frontend (Next.js) │          │  Scraper (Rust)       │
│                      │          │                       │
│  Reads from:         │          │  Reads from:          │
│  • Redis (cache)     │          │  • Message bus (jobs)  │
│  • Postgres (DB)     │          │                       │
│                      │          │  Writes to:           │
│  Writes to:          │          │  • Redis (cache)       │
│  • Message bus (jobs)│          │  • Postgres (DB)       │
│                      │          │  • Message bus (done)  │
└──────────┬───────────┘          └──────────┬────────────┘
           │                                 │
           └──────────┬──┬───────────────────┘
                      │  │
              ┌───────▼──▼────────┐
              │   Message Bus     │
              │   (Redis Streams) │
              └───────────────────┘
                      │
              ┌───────▼───────────┐
              │   Postgres        │
              │   (Supabase)      │
              └───────────────────┘
              ┌───────────────────┐
              │   Redis           │
              │   (Cache + Bus)   │
              └───────────────────┘
```

## 2. Communication Protocol Options

### Option A: Redis Streams + Pub/Sub (Recommended)

Redis serves dual purpose: **cache** (hot reads) + **message bus** (job coordination).

| Channel | Type | Purpose |
|---------|------|---------|
| `scrape:jobs` | Redis Stream | FE publishes scrape requests |
| `scrape:results:{request_id}` | Redis Pub/Sub | Scraper publishes per-date results |
| `scrape:status:{request_id}` | Redis Key (TTL) | Job status tracking |
| `cache:{origin}:{dest}:{date}` | Redis Key (60s TTL) | Cached flight results |

**Pros:** Fast (sub-ms pub/sub), persistent streams (replay on crash), built-in consumer groups for scaling, serves as cache too.
**Cons:** Need Redis (Upstash free tier: 10K commands/day, or local Redis).

### Option B: Postgres LISTEN/NOTIFY + Jobs Table

Use Postgres as both storage and message bus.

| Component | Implementation |
|-----------|---------------|
| Job queue | `scrape_jobs` table (status: pending → processing → completed → failed) |
| Real-time | LISTEN/NOTIFY on channel `scrape_results` |
| Status | Poll `scrape_jobs` table |

**Pros:** No extra infra (Supabase already exists).
**Cons:** NOTIFY payload capped at 8KB. LISTEN requires persistent connection. Higher latency than Redis. Polling fallback needed.

### Option C: NATS

Lightweight message broker with request-reply pattern.

**Pros:** Purpose-built for this, very fast, built-in request-reply.
**Cons:** Another service to run and maintain.

### Option D: Unix Domain Sockets / IPC

Direct inter-process communication on same host.

**Pros:** Zero-dependency, fastest possible, perfect for self-hosted MVP.
**Cons:** Same-host only, doesn't scale to multi-host.

### Recommendation

**MVP: Option A (Redis)** — serves as both cache and message bus. Single dependency, well-supported in both Rust and Node, free tier available (Upstash or local).

**Alternative MVP: Option D (Unix sockets)** if we want zero external dependencies for local dev. Can layer Redis on top later.

## 3. Actor-Based Scraper Design (Rust)

### 3.1 Actor Hierarchy

```
┌─────────────────────────────────────────────────────┐
│                  Supervisor Actor                    │
│  • Monitors all child actors                        │
│  • Restarts on failure (let-it-crash)               │
│  • Manages graceful shutdown                        │
└──────────────────────┬──────────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         ▼             ▼             ▼
┌──────────────┐ ┌──────────┐ ┌───────────────┐
│ Dispatcher   │ │ Pool     │ │ Result        │
│ Actor        │ │ Manager  │ │ Processor     │
│              │ │ Actor    │ │ Actor         │
│ • Consumes   │ │          │ │               │
│   job stream │ │ • Manages│ │ • Normalizes  │
│ • Validates  │ │   N      │ │   raw data    │
│ • Dedup jobs │ │   browser│ │ • Writes to   │
│ • Enqueues   │ │   actors │ │   Postgres    │
│   to pool    │ │ • Health │ │ • Writes to   │
│              │ │   checks │ │   Redis cache │
│              │ │ • Auto-  │ │ • Publishes   │
│              │ │   scale  │ │   completion  │
└──────┬───────┘ └────┬─────┘ └───────┬───────┘
       │              │               │
       │         ┌────┴────┐          │
       │         ▼         ▼          │
       │  ┌───────────┐ ┌───────────┐ │
       │  │ Browser   │ │ Browser   │ │
       │  │ Actor #1  │ │ Actor #N  │ │
       │  │           │ │           │ │
       │  │ • Owns 1  │ │ • Owns 1  │ │
       │  │   Chrome  │ │   Chrome  │ │
       │  │   context │ │   context │ │
       │  │ • Handles │ │ • Handles │ │
       │  │   1 scrape│ │   1 scrape│ │
       │  │   at a    │ │   at a    │ │
       │  │   time    │ │   time    │ │
       │  │ • Anti-bot│ │ • Anti-bot│ │
       │  │   per ctx │ │   per ctx │ │
       │  └───────────┘ └───────────┘ │
       │                              │
       └──────────────────────────────┘
```

### 3.2 Actor Responsibilities

| Actor | Responsibility | State |
|-------|---------------|-------|
| **Supervisor** | Monitor children, restart on crash, shutdown | Child actor handles |
| **Dispatcher** | Consume Redis stream `scrape:jobs`, deduplicate, route to pool | Seen job IDs (dedup set) |
| **PoolManager** | Maintain N BrowserActors, work queue, load balancing | Available/busy actor list |
| **BrowserActor** | Own one Chrome context, execute one scrape at a time, apply anti-bot | Browser context, UA, viewport |
| **ResultProcessor** | Normalize raw AlaskaResponse, write DB + cache, publish completion | DB connection pool |

### 3.3 Rust Technology Choices

| Concern | Library | Why |
|---------|---------|-----|
| Async runtime | `tokio` | Industry standard, required by most Rust async libs |
| Actor framework | `tokio` channels (mpsc/broadcast) | Lightweight, no heavy framework needed |
| Browser control | `chromiumoxide` | Rust-native CDP (Chrome DevTools Protocol) client |
| Redis client | `redis-rs` + `deadpool-redis` | Async Redis with connection pooling |
| Postgres client | `sqlx` | Async, compile-time checked queries |
| HTTP (health endpoint) | `axum` | Lightweight, tokio-native |
| Serialization | `serde` + `serde_json` | Standard Rust serialization |
| Retry/backoff | `backon` | Ergonomic retry with backoff strategies |
| Logging | `tracing` | Structured async-aware logging |

**Why not Playwright in Rust?** Playwright is Node-only. But `chromiumoxide` provides the same capabilities we need:
- Launch/control headless Chrome
- Intercept network responses (CDP `Network.responseReceived`)
- Block domains (CDP `Network.setBlockedURLs`)
- Set viewport, user-agent, cookies

### 3.4 Browser Control via CDP (chromiumoxide)

```
chromiumoxide approach:

1. Launch Chrome with: --headless --disable-gpu --no-sandbox
2. Create browser context (isolated session)
3. CDP: Network.enable → listen for responseReceived events
4. CDP: Network.setBlockedURLs → block tracking domains
5. CDP: Page.navigate → go to Alaska search URL
6. Wait for Network.responseReceived where URL matches "searchbff/V3"
7. CDP: Network.getResponseBody → extract JSON
8. Parse AlaskaResponse → normalize → emit to ResultProcessor
9. Close page (keep context alive for reuse)
```

## 4. Communication Protocol Specification

### 4.1 Message Schemas

#### Scrape Request (FE → Redis Stream → Scraper)

```json
{
  "request_id": "uuid-v4",
  "origin": "SEA",
  "destination": "NRT",
  "dates": ["2026-06-01", "2026-06-02", "..."],
  "priority": "normal",
  "requested_at": "2026-03-17T20:30:00Z",
  "callback_channel": "scrape:results:uuid-v4"
}
```

**Stream:** `scrape:jobs`
**Consumer group:** `scraper-workers`

#### Scrape Result (Scraper → Redis Pub/Sub → FE)

```json
{
  "request_id": "uuid-v4",
  "date": "2026-06-01",
  "status": "success",
  "origin": "SEA",
  "destination": "NRT",
  "cheapest": {
    "cabin": "economy",
    "miles": 25000,
    "cash": 5.60
  },
  "flight_count": 3,
  "scraped_at": "2026-03-17T20:30:05Z"
}
```

**Channel:** `scrape:results:{request_id}`

#### Job Status (Redis Key)

```json
{
  "request_id": "uuid-v4",
  "status": "processing",
  "total_dates": 30,
  "completed_dates": 12,
  "failed_dates": 0,
  "started_at": "2026-03-17T20:30:00Z"
}
```

**Key:** `scrape:status:{request_id}` (TTL: 5 minutes)

### 4.2 Protocol Flow

```
┌────────┐     ┌─────────┐     ┌───────────┐     ┌──────────┐     ┌────────┐
│ Browser │     │ Next.js │     │   Redis   │     │  Scraper │     │ Alaska │
│         │     │   FE    │     │           │     │  (Rust)  │     │  .com  │
└────┬────┘     └────┬────┘     └─────┬─────┘     └────┬─────┘     └───┬────┘
     │               │               │                 │               │
     │  GET /search  │               │                 │               │
     │──────────────▶│               │                 │               │
     │               │               │                 │               │
     │               │ 1. Check cache│                 │               │
     │               │──────────────▶│                 │               │
     │               │               │                 │               │
     │               │   cache miss  │                 │               │
     │               │◀──────────────│                 │               │
     │               │               │                 │               │
     │               │ 2. Check DB   │                 │               │
     │               │──────────────▶│ (via Supabase)  │               │
     │               │   no recent   │                 │               │
     │               │◀──────────────│                 │               │
     │               │               │                 │               │
     │               │ 3. Publish    │                 │               │
     │               │   scrape job  │                 │               │
     │               │──────────────▶│ XADD scrape:jobs│               │
     │               │               │                 │               │
     │               │ 4. Subscribe  │                 │               │
     │               │   to results  │                 │               │
     │               │──────────────▶│ SUB scrape:results:{id}         │
     │               │               │                 │               │
     │  SSE stream   │               │                 │               │
     │◀ ─ ─ ─ ─ ─ ─ ─│ (open)       │                 │               │
     │               │               │  XREAD (consume)│               │
     │               │               │────────────────▶│               │
     │               │               │                 │               │
     │               │               │                 │  Navigate     │
     │               │               │                 │──────────────▶│
     │               │               │                 │  Intercept    │
     │               │               │                 │◀──────────────│
     │               │               │                 │               │
     │               │               │                 │ Normalize     │
     │               │               │                 │──┐            │
     │               │               │                 │  │            │
     │               │               │                 │◀─┘            │
     │               │               │                 │               │
     │               │               │  Write cache    │               │
     │               │               │◀────────────────│               │
     │               │               │                 │               │
     │               │               │  Write DB       │               │
     │               │               │◀────────────────│ (via sqlx)    │
     │               │               │                 │               │
     │               │               │  PUB result     │               │
     │               │  result event │◀────────────────│               │
     │               │◀──────────────│                 │               │
     │  SSE: date    │               │                 │               │
     │◀──────────────│               │                 │               │
     │               │               │                 │               │
     │  (repeat for each date)       │                 │               │
     │               │               │                 │               │
     │  SSE: done    │               │                 │               │
     │◀──────────────│               │                 │               │
```

### 4.3 FE Read Priority

When user requests a search, FE checks in order:

```
1. Redis cache (key: cache:{origin}:{dest}:{date})
   → Hit & fresh (< 60s)? Return immediately. Done.

2. Postgres (award_searches + award_flights WHERE searched_at > now() - 60s)
   → Recent result exists? Return from DB. Done.

3. Neither fresh? Publish scrape job + subscribe to results.
   → Stream results to browser via SSE as they arrive.
```

### 4.4 Deduplication

Multiple users searching the same route simultaneously should not trigger duplicate scrapes.

**Strategy:** Before publishing to `scrape:jobs`, FE checks:
- `SETNX scrape:lock:{origin}:{dest}:{date} {request_id} EX 30`
- If lock acquired → publish job
- If lock exists → subscribe to the existing request's result channel instead

The scraper also deduplicates at the Dispatcher level by maintaining a short-lived set of in-flight jobs.

## 5. Data Pipeline: Scrape → Process → Serve

```
┌───────────────────────────────────────────────────────────────┐
│                    SCRAPE PIPELINE                             │
│                                                               │
│  ┌─────────┐    ┌──────────┐    ┌─────────┐    ┌──────────┐ │
│  │  Raw     │───▶│ Validate │───▶│ Normal- │───▶│ Persist  │ │
│  │  Scrape  │    │ & Filter │    │  ize    │    │          │ │
│  └─────────┘    └──────────┘    └─────────┘    └──────────┘ │
│                                                    │    │     │
│                                              ┌─────┘    │     │
│                                              ▼          ▼     │
│                                         ┌────────┐ ┌───────┐ │
│                                         │ Redis  │ │Postgres│ │
│                                         │ Cache  │ │  DB   │ │
│                                         └────────┘ └───────┘ │
└───────────────────────────────────────────────────────────────┘

Stage 1: Raw Scrape (BrowserActor)
  Input:  (origin, dest, date)
  Output: AlaskaResponse (raw JSON from searchbff/V3)
  Notes:  CDP intercept, anti-bot applied

Stage 2: Validate & Filter (ResultProcessor)
  Input:  AlaskaResponse
  Rules:
  • Drop if no slices (no flights)
  • Drop segments where origin/dest don't match query
  • Flag connecting flights (segments > 1) separately
  Output: ValidatedResponse

Stage 3: Normalize (ResultProcessor)
  Input:  ValidatedResponse
  Transform:
  • Map cabin names: FIRST→first, BUSINESS→business, MAIN/COACH/SAVER→economy
  • Extract lowest fare per cabin class
  • Flag saver fares
  • Convert times to ISO 8601 with timezone
  • Calculate duration in minutes
  Output: NormalizedFlightResult

Stage 4: Persist (ResultProcessor)
  Input:  NormalizedFlightResult
  Actions:
  • INSERT into award_searches (with raw_response JSONB)
  • INSERT into award_flights (one row per flight × fare)
  • SET Redis cache key (60s TTL)
  • PUBLISH to scrape:results:{request_id}
```

## 6. Postgres Schema for Job Tracking

```
┌──────────────────────────────────────────────────┐
│ scrape_jobs                                       │
├──────────────────────────────────────────────────┤
│ id              UUID PRIMARY KEY                  │
│ origin_code     VARCHAR(3) NOT NULL               │
│ dest_code       VARCHAR(3) NOT NULL               │
│ search_dates    DATE[] NOT NULL                   │
│ status          VARCHAR(20) DEFAULT 'pending'     │
│                 (pending/processing/completed/    │
│                  partial/failed)                   │
│ total_dates     INTEGER                           │
│ completed_dates INTEGER DEFAULT 0                 │
│ failed_dates    INTEGER DEFAULT 0                 │
│ requested_at    TIMESTAMP DEFAULT now()           │
│ started_at      TIMESTAMP                         │
│ completed_at    TIMESTAMP                         │
│ error_message   TEXT                              │
└──────────────────────────────────────────────────┘

  scrape_jobs ──1:N──▶ award_searches ──1:N──▶ award_flights
```

## 7. Error Handling & Resilience

| Scenario | Handling |
|----------|---------|
| Chrome crash | Supervisor restarts BrowserActor, replaces context |
| Alaska blocks request | Exponential backoff, rotate UA/viewport, circuit breaker |
| Redis down | FE falls back to polling Postgres directly |
| Postgres down | Scraper buffers results in Redis, retries DB write |
| Scraper process crash | Supervisor restarts; unacked Redis stream messages get re-delivered |
| Duplicate requests | Redis SETNX lock + Dispatcher dedup set |
| Timeout (no response in 30s) | BrowserActor aborts, marks job failed, returns to pool |
| Stale cache entry | TTL-based eviction (60s), no manual invalidation needed |

## 8. Rust Project Structure

```
smart-booking/alaska/
├── apps/
│   ├── web/                      # Next.js frontend (unchanged)
│   │
│   └── scraper/                  # Rust scraper subsystem
│       ├── Cargo.toml
│       ├── src/
│       │   ├── main.rs           # Entry point, supervisor setup
│       │   ├── config.rs         # Environment config
│       │   ├── actors/
│       │   │   ├── mod.rs
│       │   │   ├── supervisor.rs # Top-level supervisor
│       │   │   ├── dispatcher.rs # Redis stream consumer
│       │   │   ├── pool.rs       # Browser pool manager
│       │   │   ├── browser.rs    # Single browser actor
│       │   │   └── processor.rs  # Result normalization + persistence
│       │   ├── scraping/
│       │   │   ├── mod.rs
│       │   │   ├── alaska.rs     # Alaska-specific scrape logic
│       │   │   ├── anti_bot.rs   # Anti-detection utilities
│       │   │   └── types.rs      # AlaskaResponse types
│       │   ├── storage/
│       │   │   ├── mod.rs
│       │   │   ├── cache.rs      # Redis cache operations
│       │   │   ├── db.rs         # Postgres operations (sqlx)
│       │   │   └── models.rs     # DB models
│       │   └── protocol/
│       │       ├── mod.rs
│       │       ├── messages.rs   # Message schemas
│       │       └── streams.rs    # Redis stream operations
│       ├── Dockerfile
│       └── tests/
│           ├── integration/
│           └── unit/
```

## 9. Open Questions

1. **Redis hosting for MVP** — Local Redis? Upstash free tier (10K commands/day)?
2. **chromiumoxide maturity** — Need to verify it can reliably intercept XHR responses and handle Alaska's Akamai protection. Fallback: Rust orchestrates, spawns Playwright (Node) as subprocess.
3. **Compile-time SQL (sqlx)** — Requires DB connection at build time. Use `sqlx::query!` with offline mode for CI.
4. **Calendar month: batch or individual?** — Should FE publish 30 individual jobs or 1 batch job with 30 dates?
