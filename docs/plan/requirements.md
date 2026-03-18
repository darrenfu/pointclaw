# Smart Booking — Requirements

Last updated: 2026-03-17

## Product Vision

A personalized travel booking recommendation engine — **seats.aero++**.

Consolidates multiple input factors (credit card perks, free hotel nights, airline mile balances, loyalty status) to recommend optimal end-to-end trip bookings across airlines and hotels.

## Differentiators vs seats.aero

- Support login-based airlines (JAL, ANA, etc.) that seats.aero can't access
- Personalized recommendations based on user's loyalty status, miles balances, credit card perks
- Smarter alerting (not flat/limited)
- End-to-end trip planning (flights + hotels), not just award search

## MVP (Phase 1) — Alaska Airlines Award Search

**Users:** Anonymous (no login/accounts)

**Features:**
1. Search for award availability on specific routes and dates
2. Display results with miles cost + taxes
3. Alert when award availability opens up on watched routes

**Scope:** Alaska Airlines only

## Phase 2 — Multi-Airline Expansion

- Roll up to more airlines with miles-based search
- Add airlines requiring login (JAL, ANA, etc.)

## Phase 3 — Personalization & Hotels

- User accounts with saved preferences
- Credit card perk integration (free nights, bonus categories)
- Mile point balance tracking
- Hotel award availability
- E2E trip recommendation engine

## Open Questions

- Data source for Alaska award availability? (scraping, API, third-party?)
- Tech stack preferences?
- Alerting mechanism? (email, push, browser notifications?)
- Hosting/deployment preferences?
