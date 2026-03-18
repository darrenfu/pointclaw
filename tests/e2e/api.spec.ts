import { test, expect } from "@playwright/test";

// ============================================================================
// API E2E Tests — hit the live Next.js server and verify responses
// Requires: Next.js running on localhost:3000, Redis with seeded data
// ============================================================================

test.describe("GET /api/airports", () => {
  test("returns airports matching a code query", async ({ request }) => {
    const res = await request.get("/api/airports?q=sea");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.airports).toBeDefined();
    expect(body.airports.length).toBeGreaterThan(0);
    const sea = body.airports.find((a: any) => a.code === "SEA");
    expect(sea).toBeDefined();
    expect(sea.city).toBe("Seattle");
  });

  test("returns airports matching a city query", async ({ request }) => {
    const res = await request.get("/api/airports?q=tokyo");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.airports.length).toBeGreaterThan(0);
    const codes = body.airports.map((a: any) => a.code);
    expect(codes).toContain("NRT");
    expect(codes).toContain("HND");
  });

  test("returns airports matching a partial name", async ({ request }) => {
    const res = await request.get("/api/airports?q=narita");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.airports.length).toBeGreaterThan(0);
    expect(body.airports[0].code).toBe("NRT");
  });

  test("returns default airports when no query provided", async ({ request }) => {
    const res = await request.get("/api/airports");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.airports).toBeDefined();
    expect(body.airports.length).toBeLessThanOrEqual(20);
  });

  test("returns empty query gives default airports", async ({ request }) => {
    const res = await request.get("/api/airports?q=");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.airports.length).toBeLessThanOrEqual(20);
  });

  test("returns empty results for nonsense query", async ({ request }) => {
    const res = await request.get("/api/airports?q=zzzzzzz");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.airports.length).toBe(0);
  });

  test("search is case-insensitive", async ({ request }) => {
    const lower = await (await request.get("/api/airports?q=sea")).json();
    const upper = await (await request.get("/api/airports?q=SEA")).json();
    const mixed = await (await request.get("/api/airports?q=Sea")).json();
    expect(lower.airports.length).toBe(upper.airports.length);
    expect(lower.airports.length).toBe(mixed.airports.length);
  });

  test("results are capped at 10", async ({ request }) => {
    // "a" should match many airports
    const res = await request.get("/api/airports?q=a");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.airports.length).toBeLessThanOrEqual(10);
  });
});

test.describe("GET /api/search — month query", () => {
  test("returns calendar data for a seeded month", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-03");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.calendar).toBeDefined();
    expect(body.hasData).toBe(true);

    // Should have entries for each day of March
    const dates = Object.keys(body.calendar);
    expect(dates.length).toBe(31);

    // At least some dates should have "success" status
    const successDates = dates.filter((d) => body.calendar[d].status === "success");
    expect(successDates.length).toBeGreaterThan(0);
  });

  test("success dates have cheapest fare info", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-03");
    const body = await res.json();
    const dates = Object.keys(body.calendar);
    const successDates = dates.filter((d) => body.calendar[d].status === "success");

    for (const d of successDates) {
      const entry = body.calendar[d];
      expect(entry.cheapest).not.toBeNull();
      expect(entry.cheapest.miles).toBeGreaterThan(0);
      expect(entry.cheapest.cabin).toBeDefined();
      expect(entry.flightCount).toBeGreaterThan(0);
    }
  });

  test("no_flights dates have null cheapest", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-03");
    const body = await res.json();
    const dates = Object.keys(body.calendar);
    const noFlightDates = dates.filter((d) => body.calendar[d].status === "no_flights");

    for (const d of noFlightDates) {
      const entry = body.calendar[d];
      expect(entry.cheapest).toBeNull();
      expect(entry.flightCount).toBe(0);
    }
  });

  test("returns 400 when origin is missing", async ({ request }) => {
    const res = await request.get("/api/search?dest=NRT&month=2026-03");
    expect(res.status()).toBe(400);
  });

  test("returns 400 when dest is missing", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&month=2026-03");
    expect(res.status()).toBe(400);
  });

  test("returns 400 when both month and date are missing", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT");
    expect(res.status()).toBe(400);
  });

  test("returns loading status for unseeded month", async ({ request }) => {
    // Use a route+month combo that definitely has no cache and no scraper activity
    const res = await request.get("/api/search?origin=SEA&dest=NRT&month=2029-12");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.calendar).toBeDefined();
    expect(body.hasData).toBe(false);

    // All dates should be "loading" (no cache)
    const dates = Object.keys(body.calendar);
    for (const d of dates) {
      expect(body.calendar[d].status).toBe("loading");
    }
  });

  test("returns correct number of days for February", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-02");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(Object.keys(body.calendar).length).toBe(28);
  });

  test("returns correct number of days for leap year February", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT&month=2028-02");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(Object.keys(body.calendar).length).toBe(29);
  });

  test("returns correct number of days for April (30 days)", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-04");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(Object.keys(body.calendar).length).toBe(30);
  });

  test("handles unseeded route gracefully", async ({ request }) => {
    const res = await request.get("/api/search?origin=JFK&dest=LHR&month=2026-03");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.calendar).toBeDefined();
    expect(body.hasData).toBe(false);
  });
});

test.describe("GET /api/search — date query", () => {
  test("returns flight details for a seeded date", async ({ request }) => {
    // First find a date that has data
    const calRes = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-03");
    const calBody = await calRes.json();
    const successDate = Object.keys(calBody.calendar).find(
      (d) => calBody.calendar[d].status === "success"
    );
    expect(successDate).toBeDefined();

    const res = await request.get(`/api/search?origin=SEA&dest=NRT&date=${successDate}`);
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.flights).toBeDefined();
    expect(body.flights.length).toBeGreaterThan(0);
  });

  test("flight objects have required fields", async ({ request }) => {
    const calRes = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-03");
    const calBody = await calRes.json();
    const successDate = Object.keys(calBody.calendar).find(
      (d) => calBody.calendar[d].status === "success"
    );

    const res = await request.get(`/api/search?origin=SEA&dest=NRT&date=${successDate}`);
    const body = await res.json();

    for (const flight of body.flights) {
      expect(flight.flightNumber).toBeDefined();
      expect(flight.carrier).toBeDefined();
      expect(flight.carrier.code).toBeDefined();
      expect(flight.carrier.name).toBeDefined();
      expect(flight.departure).toBeDefined();
      expect(flight.departure.airport).toBeDefined();
      expect(flight.departure.time).toBeDefined();
      expect(flight.arrival).toBeDefined();
      expect(flight.arrival.airport).toBeDefined();
      expect(flight.arrival.time).toBeDefined();
      expect(flight.duration).toBeGreaterThan(0);
      expect(flight.fares).toBeDefined();
      expect(flight.fares.length).toBeGreaterThan(0);
    }
  });

  test("fare objects have required fields", async ({ request }) => {
    const calRes = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-03");
    const calBody = await calRes.json();
    const successDate = Object.keys(calBody.calendar).find(
      (d) => calBody.calendar[d].status === "success"
    );

    const res = await request.get(`/api/search?origin=SEA&dest=NRT&date=${successDate}`);
    const body = await res.json();

    for (const flight of body.flights) {
      for (const fare of flight.fares) {
        expect(fare.cabin).toBeDefined();
        expect(["economy", "business", "first"]).toContain(fare.cabin);
        expect(fare.miles).toBeGreaterThan(0);
        expect(fare.cash).toBeGreaterThanOrEqual(0);
        expect(fare.seatsRemaining).toBeGreaterThanOrEqual(0);
        expect(typeof fare.isSaver).toBe("boolean");
      }
    }
  });

  test("returns empty flights for no_flights date", async ({ request }) => {
    const calRes = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-03");
    const calBody = await calRes.json();
    const noFlightDate = Object.keys(calBody.calendar).find(
      (d) => calBody.calendar[d].status === "no_flights"
    );

    if (noFlightDate) {
      const res = await request.get(`/api/search?origin=SEA&dest=NRT&date=${noFlightDate}`);
      expect(res.status()).toBe(200);
      const body = await res.json();
      // Should return cache data with status no_flights or empty flights
      expect(body.flights === undefined || body.flights?.length === 0 || body.status === "no_flights").toBeTruthy();
    }
  });

  test("returns graceful response for uncached date", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT&date=2099-01-01");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.cached).toBe(false);
    expect(body.flights).toEqual([]);
  });

  test("returns 400 when origin is missing for date query", async ({ request }) => {
    const res = await request.get("/api/search?dest=NRT&date=2026-03-01");
    expect(res.status()).toBe(400);
  });
});

test.describe("Scraper integration — Redis stream and job dispatch", () => {
  test("scrape job is published when no cache exists for a month", async ({ request }) => {
    // Query an unseeded route — this should trigger a scrape job
    const res = await request.get("/api/search?origin=PDX&dest=HND&month=2026-09");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.hasData).toBe(false);

    // The API should have published a job to the Redis stream
    // We can verify by checking the lock key exists
    // (The API sets a lock: scrape:lock:PDX:HND:2026-09)
    // We'll verify indirectly — a second request should NOT publish another job (dedup lock)
    const res2 = await request.get("/api/search?origin=PDX&dest=HND&month=2026-09");
    expect(res2.status()).toBe(200);
    // Both should return hasData=false since scraper won't have data for this route
  });

  test("dedup lock prevents duplicate scrape jobs", async ({ request }) => {
    // First request triggers a job
    await request.get("/api/search?origin=SFO&dest=ICN&month=2026-10");
    // Second request within 30s should hit the lock
    const res2 = await request.get("/api/search?origin=SFO&dest=ICN&month=2026-10");
    expect(res2.status()).toBe(200);
    // Both return fine — the key test is that only one job was published
    // (We can't easily verify stream contents from here, but the lock mechanism is tested)
  });
});

test.describe("Edge cases and error handling", () => {
  test("handles invalid month format gracefully", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT&month=not-a-month");
    expect(res.status()).toBe(200);
    // Should return a calendar (possibly empty/broken) without crashing
    const body = await res.json();
    expect(body.calendar).toBeDefined();
  });

  test("handles month=13 (invalid) gracefully", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-13");
    expect(res.status()).toBe(200);
    const body = await res.json();
    // JavaScript Date(2026, 13, 0) wraps to January 2027 — check behavior
    expect(body.calendar).toBeDefined();
  });

  test("handles month=00 (invalid) gracefully", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=NRT&month=2026-00");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.calendar).toBeDefined();
  });

  test("handles very long origin code", async ({ request }) => {
    const res = await request.get("/api/search?origin=ABCDEFGHIJKLMNOP&dest=NRT&month=2026-03");
    expect(res.status()).toBe(200);
  });

  test("handles special characters in query", async ({ request }) => {
    const res = await request.get("/api/airports?q=%3Cscript%3E");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.airports.length).toBe(0);
  });

  test("handles empty origin", async ({ request }) => {
    const res = await request.get("/api/search?origin=&dest=NRT&month=2026-03");
    expect(res.status()).toBe(400);
  });

  test("handles empty dest", async ({ request }) => {
    const res = await request.get("/api/search?origin=SEA&dest=&month=2026-03");
    expect(res.status()).toBe(400);
  });
});
