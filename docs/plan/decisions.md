# Smart Booking — Key Decisions

Last updated: 2026-03-17

## Decision Log

### D001: Target Airline (2026-03-17)
**Decision:** Start with Alaska Airlines only, expand later.
**Rationale:** Personal use case, Alaska has public award calendar search without login.

### D002: No User Auth for MVP (2026-03-17)
**Decision:** Anonymous users, no login/accounts in MVP.
**Rationale:** Simplify MVP scope. User accounts come in Phase 3 with personalization.

### D003: Tech Stack — FINAL (2026-03-17)
**Decision:**
- Frontend: React / Next.js / Tailwind / TanStack Query
- Scraper: **Go** (copying seats.aero's approach) + **playwright-go** (fallback: rod)
- Message Bus + Cache: **Redis** (local for MVP)
- Database: Postgres (Supabase) + pgvector + Drizzle ORM
- Hosting: Self-hosted for MVP
**Rationale:** Go is what seats.aero and Roame use for scraping infra. playwright-go gives Playwright API in Go. Redis serves dual purpose as cache and message bus. Industry-aligned stack.

### D004: Data Source — FINAL (2026-03-17)
**Decision:** Playwright only for MVP. No seats.aero integration yet.
**Approach:** Playwright browser automation → intercept `searchbff/V3/search` JSON response (real-time, free)
**Why not direct curl?** Tested — Alaska's searchbff returns HTML without browser context. Akamai bot protection requires JS execution.
**Future:** seats.aero API can be added later for multi-airline expansion.
**See:** `research-data-sources.md` for full analysis.

### D010: Hosting Architecture — UPDATED (2026-03-17)
**Decision:** Self-hosted for MVP. Decoupled scraper and serving layers.
- **Frontend (Next.js):** Self-hosted (local). Vercel only if free tier suffices.
- **Scraper service (Playwright + Node/TS):** Self-hosted (local). Railway only if free tier suffices.
- **Database:** Supabase (free tier: 500MB, 2 projects)
**Rationale:** Zero-cost MVP for development and e2e testing. Cloud hosting adopted later only if free.

### D005: Alerting Mechanism — DEFERRED (2026-03-17)
**Decision:** Moved to Phase 2.
**Rationale:** Focus MVP on core search experience. Alerts add complexity (cron, email service).

### D006: Caching Strategy (2026-03-17)
**Decision:** On-demand search + 1-minute cache invalidation. No background polling in MVP.
**Rationale:** Keep infrastructure simple. User triggers search, results cached briefly to avoid hammering the API.

### D007: Route Scope (2026-03-17)
**Decision:** 5 origins x ~35 destinations ≈ 175 route pairs.
**Origins:** SEA, LAX, SFO, YVR, PDX
**Destinations:** Japan (NRT, HND, KIX), Korea (ICN), SE Asia (TPE, HKG, SIN, BKK, MNL), Oceania (SYD, MEL, AKL, NAN), US (JFK, BOS, MIA, ORD, DFW, DCA, HNL, ANC), Canada (YYZ, YUL), Islands (OGG, KOA, LIH, CUN, SJD, PVR), Europe (LHR, CDG, FRA, FCO, BCN), South America (LIM, BOG, SCL, EZE)
**Includes:** Popular destinations for Alaska alliance airlines (JAL, AA, Cathay, Korean Air, etc.)

### D008: Search Depth (2026-03-17)
**Decision:** 12 months rolling search, month-by-month navigation.

### D009: UI Approach (2026-03-17)
**Decision:** Both calendar heatmap and list view. Focus on visual design.
**Calendar heatmap:** Color gradient by miles cost (green = cheap saver, yellow = mid, red = expensive). Gray/empty for unavailable dates.
**Rationale:** Calendar gives at-a-glance availability across dates; list gives detailed flight info.

### D011: Storage — Write-Through (2026-03-17)
**Decision:** Dual-layer — Redis cache (60s TTL) + Postgres (permanent). Scraper writes to both.
**Rationale:** Historical data for trends and Phase 2 alerts.

### D012: Scraper Language — Go (2026-03-17)
**Decision:** Go, copying seats.aero's stack. Actor pattern via goroutines + channels.
**Browser lib:** playwright-go first, rod as fallback.
**Rationale:** Industry standard for scraping infra (seats.aero, Roame both use Go). Goroutines are natural actors. Single binary deployment.

### D013: FE ↔ Scraper Decoupling (2026-03-17)
**Decision:** FE queries Redis + Postgres directly. Scraper is independent subsystem consuming Redis Stream jobs.
**Protocol:** Redis Streams (job queue) + Pub/Sub (results). SSE to browser.
**Rationale:** Clean separation. FE doesn't depend on scraper being up for cached reads.

### D014: Job Batching (2026-03-17)
**Decision:** Minimize job count and scrape URL count. Test calendar URL to see if 1 URL = 1 month of data.
**Approach:** Start with calendar URL, tune batch boundaries via testing.

### D015: Message Bus (2026-03-17)
**Decision:** Local Redis for MVP. Streams + Pub/Sub + cache keys.
**Rationale:** Zero cost, single dependency, serves as both cache and message bus.
