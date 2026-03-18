# Data Source Research — 2026-03-17

## Key Finding: Alaska Direct JSON API

Alaska has an internal BFF (Backend For Frontend) API that returns structured JSON. **No browser automation needed for basic search.**

### Endpoint
```
GET https://www.alaskaair.com/searchbff/V3/search
```

### Parameters
| Param | Example | Description |
|-------|---------|-------------|
| `origins` | `SEA` | IATA origin code |
| `destinations` | `NRT` | IATA destination code |
| `dates` | `2026-06-01` | Departure date |
| `numADTs` | `1` | Number of adult passengers |
| `fareView` | `as_awards` | Must be `as_awards` for mile pricing |
| `sessionID` | (empty) | Optional session ID |
| `solutionSetIDs` | (empty) | Optional |
| `solutionIDs` | (empty) | Optional |

### Response Structure (TypeScript)
```typescript
type AlaskaResponse = {
  departureStation: string
  arrivalStation: string
  slices?: {
    id: number
    origin: string
    destination: string
    duration: number
    segments: {
      publishingCarrier: { carrierCode: string; carrierFullName: string; flightNumber: number }
      displayCarrier: { carrierCode: string; carrierFullName: string; flightNumber: number }
      departureStation: string
      arrivalStation: string
      aircraftCode: string
      aircraft: string
      duration: number
      departureTime: string    // ISO datetime
      arrivalTime: string      // ISO datetime
      nextDayArrival: boolean
      performance: { canceledPercentage: number; percentOntime: number; ... }[]
      amenities: string[]      // e.g. ["Wi-Fi"]
    }[]
    fares: Record<string, {
      grandTotal: number       // Cash portion (taxes/fees)
      milesPoints: number      // Miles cost
      seatsRemaining: number
      discount: boolean
      mixedCabin: boolean
      cabins: string[]         // "FIRST" | "MAIN" | "SAVER" | "COACH" | "BUSINESS"
      bookingCodes: string[]
      refundable: boolean
    }>
  }[]
  env: string
  qpxcSessionID: string
  advisories: any[]
}
```

### Risks
- No official documentation — endpoint could change without notice
- Potential bot protection (Akamai or similar)
- May require browser-like headers/cookies
- TOS violation risk

### Source
Reverse-engineered by [AwardWiz](https://github.com/lg/awardwiz) (archived Sept 2024, 123 stars)

---

## Alternative: seats.aero API

### Overview
Pre-scraped cached award data across 24 airline programs including Alaska.

### Pricing
- **Pro:** $9.99/mo or $99.99/yr
- **1,000 API calls/day** (resets midnight UTC)
- **Commercial use requires separate agreement**

### Endpoints
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/partnerapi/search` | GET | Cached search by origin/dest/dates |
| `/partnerapi/availability` | GET | Bulk availability for one program |
| `/partnerapi/trips/{id}` | GET | Flight-level detail |
| `/partnerapi/routes` | GET | List monitored routes |
| `/partnerapi/live` | POST | Real-time search (commercial only) |

### Auth
`Partner-Authorization: Bearer pro_xxxxxxxxxxxxxxxxxxxxx`

### Data Fields
Per cabin class (Y/W/J/F): availability boolean, miles cost, remaining seats, operating airlines, direct flight flag. Plus route metadata, data freshness timestamps.

### Pros
- Legal (authorized API vs scraping)
- 24 airline programs, 70,000+ routes
- Well-documented (OpenAPI 3.0)
- Cheap ($10/mo)

### Cons
- Cached data (not real-time)
- 1,000 req/day cap
- Commercial use prohibited without approval
- Attribution required

### Source
[seats.aero Developer Hub](https://developers.seats.aero/)

---

## Other Options Evaluated (Not Recommended)

| Option | Why Not |
|--------|---------|
| Alaska Official Developer Portal | No award search endpoints |
| ExpertFlyer | No API, declining data coverage |
| AwardFares | No public API |
| Duffel API | Revenue flights only, no award search |
| Milez.biz | Published charts only, no live availability |
| AwardWiz (fork) | Archived, high maintenance |
| Flightplan (fork) | Abandoned, Puppeteer-based, likely broken |
| curl-impersonate | Can't execute JS, only works if API is fully reverse-engineered |

---

## Recommendation

**Hybrid approach:**
1. **Primary:** Direct API (`searchbff/V3`) for real-time Alaska searches
2. **Fallback:** seats.aero API for when direct API fails + multi-airline expansion
3. **Future:** Playwright as last resort for airlines with no API/data provider

This gives real-time data for free, with seats.aero as a reliable cached fallback at $10/mo.
