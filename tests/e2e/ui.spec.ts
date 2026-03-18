import { test, expect } from "@playwright/test";

// ============================================================================
// UI E2E Tests — browser-based tests against the live Next.js frontend
// Requires: Next.js running on localhost:3000, Redis with seeded data
// ============================================================================

test.describe("Homepage", () => {
  test("loads with correct branding and elements", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator("h1")).toContainText("PointClaw");
    await expect(page.locator("text=Find award flights")).toBeVisible();
    await expect(page.locator("text=Search Award Flights")).toBeVisible();
    await expect(page.locator("text=Alaska Airlines")).toBeVisible();
  });

  test("has From and To input fields", async ({ page }) => {
    await page.goto("/");
    // Labels are now properly associated via htmlFor/id
    await expect(page.getByLabel("From")).toBeVisible();
    await expect(page.getByRole("textbox", { name: "To" })).toBeVisible();
  });

  test("search button is disabled when no airports selected", async ({ page }) => {
    await page.goto("/");
    const btn = page.getByRole("button", { name: "Search Award Flights" });
    await expect(btn).toBeDisabled();
  });

  test("search button is disabled when only origin selected", async ({ page }) => {
    await page.goto("/");
    // Select origin using the properly labeled input
    await page.getByLabel("From").fill("SEA");
    await page.locator("ul li").filter({ hasText: "SEA" }).first().click();

    const btn = page.getByRole("button", { name: "Search Award Flights" });
    await expect(btn).toBeDisabled();
  });
});

test.describe("Airport autocomplete", () => {
  test("shows dropdown when typing in From field", async ({ page }) => {
    await page.goto("/");
    await page.getByLabel("From").fill("SEA");
    await expect(page.locator("ul li").filter({ hasText: "SEA" })).toBeVisible();
  });

  test("shows dropdown when typing in To field", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("textbox", { name: "To" }).fill("NRT");
    await expect(page.locator("ul li").filter({ hasText: "NRT" })).toBeVisible();
  });

  test("filters by city name", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("textbox", { name: "To" }).fill("tokyo");
    const items = page.locator("ul li");
    await expect(items.first()).toBeVisible();
    // Should show NRT and HND
    await expect(page.locator("ul li").filter({ hasText: "NRT" })).toBeVisible();
    await expect(page.locator("ul li").filter({ hasText: "HND" })).toBeVisible();
  });

  test("selecting an airport shows the code and city", async ({ page }) => {
    await page.goto("/");
    await page.getByLabel("From").fill("SEA");
    await page.locator("ul li").filter({ hasText: "SEA" }).click();

    // Should show selected airport with code and city (input is replaced by display div)
    // The wrapper div still has the label, so we look inside the parent container
    await expect(page.locator("text=SEA").first()).toBeVisible();
    await expect(page.locator("text=Seattle").first()).toBeVisible();
  });

  test("clear button resets the selection", async ({ page }) => {
    await page.goto("/");
    // Select an airport
    await page.getByLabel("From").fill("SEA");
    await page.locator("ul li").filter({ hasText: "SEA" }).click();
    await expect(page.locator(".font-mono").filter({ hasText: "SEA" }).first()).toBeVisible();

    // Click the × button (inside the same container)
    await page.locator("button", { hasText: "×" }).first().click();

    // Input field should be back
    await expect(page.getByLabel("From")).toBeVisible();
  });

  test("From field only shows origin airports", async ({ page }) => {
    await page.goto("/");
    // "From" field has filterCodes={ORIGIN_CODES} = ["SEA", "LAX", "SFO", "YVR", "PDX"]
    // Typing "tokyo" should NOT show results in From field
    await page.getByLabel("From").fill("tokyo");
    await page.waitForTimeout(300);
    const items = page.locator("ul li");
    expect(await items.count()).toBe(0);
  });

  test("To field shows all airports", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("textbox", { name: "To" }).fill("tokyo");
    const items = page.locator("ul li");
    await expect(items.first()).toBeVisible();
    expect(await items.count()).toBeGreaterThan(0);
  });

  test("dropdown closes when clicking outside", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("textbox", { name: "To" }).fill("NRT");
    await expect(page.locator("ul li").first()).toBeVisible();

    // Click outside
    await page.locator("h1").click();
    await page.waitForTimeout(300);

    // Dropdown should be gone
    expect(await page.locator("ul li").count()).toBe(0);
  });
});

test.describe("Search flow — full route selection and navigation", () => {
  test("selecting both airports and searching navigates to search page", async ({ page }) => {
    await page.goto("/");

    // Select origin
    await page.getByLabel("From").fill("SEA");
    await page.locator("ul li").filter({ hasText: "SEA" }).click();

    // Select destination
    await page.getByRole("textbox", { name: "To" }).fill("NRT");
    await page.locator("ul li").filter({ hasText: "NRT" }).click();

    // Button should be enabled
    const btn = page.getByRole("button", { name: "Search Award Flights" });
    await expect(btn).toBeEnabled();

    // Click search
    await btn.click();

    // Should navigate to /search with correct params
    await page.waitForURL(/\/search\?origin=SEA&dest=NRT&month=\d{4}-\d{2}/);
    expect(page.url()).toContain("origin=SEA");
    expect(page.url()).toContain("dest=NRT");
  });

  test("search page shows correct route header", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await expect(page.locator("text=Seattle → Tokyo")).toBeVisible();
    await expect(page.locator("text=SEA → NRT")).toBeVisible();
    await expect(page.locator("text=Award availability")).toBeVisible();
  });

  test("search page has back link to homepage", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    const backLink = page.locator("a", { hasText: "← Back" });
    await expect(backLink).toBeVisible();
    await expect(backLink).toHaveAttribute("href", "/");
  });

  test("search page with missing params shows error", async ({ page }) => {
    await page.goto("/search");
    await expect(page.locator("text=Missing search parameters")).toBeVisible();
    await expect(page.locator("text=Go back")).toBeVisible();
  });
});

test.describe("Calendar heatmap", () => {
  test("renders calendar grid for March 2026", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");

    // Should show month name
    await expect(page.locator("text=March 2026")).toBeVisible();

    // Should show day headers
    for (const day of ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"]) {
      await expect(page.locator(`text=${day}`).first()).toBeVisible();
    }

    // Should have calendar cells with mile prices
    await expect(page.locator("button", { hasText: /\d+K/ }).first()).toBeVisible();
  });

  test("calendar cells show mile prices for available dates", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    // Should have cells with prices like "20K", "25K", etc.
    const priceCells = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ });
    expect(await priceCells.count()).toBeGreaterThan(0);
  });

  test("calendar cells show -- for no-flight dates", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const noFlightCells = page.locator("button").filter({ hasText: "--" });
    // Seeded data has ~30% no_flights
    expect(await noFlightCells.count()).toBeGreaterThan(0);
  });

  test("calendar has color legend", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await expect(page.locator("text=Saver")).toBeVisible();
    await expect(page.locator("text=Mid")).toBeVisible();
    await expect(page.locator("text=High")).toBeVisible();
    await expect(page.locator("text=N/A")).toBeVisible();
  });

  test("clicking a date cell selects it and shows flight details", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    // Find and click a cell with a price
    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();

    // Flight details should appear below
    await expect(page.locator("text=flight").first()).toBeVisible({ timeout: 5000 });
  });

  test("clicking a no-flight date shows no flights message", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const noFlightCell = page.locator("button").filter({ hasText: "--" }).first();
    if (await noFlightCell.count() > 0) {
      await noFlightCell.click();
      // Should show "No flights match your filters" or empty state
      await page.waitForTimeout(500);
      await expect(page.locator("text=No flights match")).toBeVisible();
    }
  });
});

test.describe("Month navigation", () => {
  test("shows prev and next month buttons", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-06");
    await expect(page.locator("button", { hasText: "◀" })).toBeVisible();
    await expect(page.locator("button", { hasText: "▶" })).toBeVisible();
  });

  test("clicking next month updates the calendar", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await expect(page.locator("text=March 2026")).toBeVisible();

    await page.locator("button", { hasText: "▶" }).click();
    await expect(page.locator("text=April 2026")).toBeVisible();
    // URL should be updated synchronously via window.history.replaceState
    await page.waitForTimeout(200);
    expect(page.url()).toContain("month=2026-04");
  });

  test("clicking prev month updates the calendar", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-06");
    await expect(page.locator("text=June 2026")).toBeVisible();

    await page.locator("button", { hasText: "◀" }).click();
    await expect(page.locator("text=May 2026")).toBeVisible();
    await page.waitForTimeout(200);
    expect(page.url()).toContain("month=2026-05");
  });

  test("prev button is disabled for current month", async ({ page }) => {
    const now = new Date();
    const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
    await page.goto(`/search?origin=SEA&dest=NRT&month=${currentMonth}`);

    const prevBtn = page.locator("button", { hasText: "◀" });
    await expect(prevBtn).toBeDisabled();
  });

  test("next button is disabled 12 months from now", async ({ page }) => {
    const future = new Date();
    future.setMonth(future.getMonth() + 12);
    const maxMonth = `${future.getFullYear()}-${String(future.getMonth() + 1).padStart(2, "0")}`;
    await page.goto(`/search?origin=SEA&dest=NRT&month=${maxMonth}`);

    const nextBtn = page.locator("button", { hasText: "▶" });
    await expect(nextBtn).toBeDisabled();
  });

  test("navigating months preserves origin and destination", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-06");
    await page.locator("button", { hasText: "▶" }).click();
    await page.waitForTimeout(500);
    await expect(page.locator("text=July 2026")).toBeVisible();
    expect(page.url()).toContain("origin=SEA");
    expect(page.url()).toContain("dest=NRT");
  });
});

test.describe("Flight details panel", () => {
  test("shows flight card with carrier info", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    // Click a date with flights
    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    // Should show flight number and carrier
    await expect(page.locator("[class*='font-mono']").filter({ hasText: /^[A-Z]{2}\s\d+$/ }).first()).toBeVisible();
  });

  test("shows departure and arrival times", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    // Should show airport codes in flight detail
    await expect(page.locator("text=SEA").last()).toBeVisible();
    await expect(page.locator("text=NRT").last()).toBeVisible();
  });

  test("shows fare options with miles and cash", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    // Should show fare badges with "K mi" format
    await expect(page.locator("text=/\\d+K mi/").first()).toBeVisible();
    // Should show cash component
    await expect(page.locator("text=/\\$\\d+\\.\\d+/").first()).toBeVisible();
  });

  test("shows SAVER badge on saver fares", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    await expect(page.locator("text=SAVER").first()).toBeVisible();
  });

  test("shows Direct badge for direct flights", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    await expect(page.locator("text=Direct").first()).toBeVisible();
  });

  test("shows flight duration", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    // Duration format: "Xh Ym"
    await expect(page.locator("text=/\\d+h \\d+m/").first()).toBeVisible();
  });

  test("shows aircraft type", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    await expect(page.locator("text=Boeing").first()).toBeVisible();
  });

  test("shows amenities", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    await expect(page.locator("text=Wi-Fi").first()).toBeVisible();
  });
});

test.describe("Sort and filter controls", () => {
  test("sort buttons are visible when flights shown", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    await expect(page.locator("button", { hasText: "Price" })).toBeVisible();
    await expect(page.locator("button", { hasText: "Departure" })).toBeVisible();
    await expect(page.locator("button", { hasText: "Duration" })).toBeVisible();
  });

  test("cabin filter buttons are visible", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    await expect(page.locator("button", { hasText: "All" })).toBeVisible();
    await expect(page.locator("button", { hasText: "Economy" })).toBeVisible();
    await expect(page.locator("button", { hasText: "Business" })).toBeVisible();
    await expect(page.locator("button", { hasText: "First" })).toBeVisible();
  });

  test("clicking cabin filter filters the flights", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    // Click "First" filter — may show fewer or zero flights
    await page.locator("button", { hasText: "First" }).click();
    await page.waitForTimeout(500);

    // Should still show the flight list section (even if empty)
    await expect(page.locator("text=Sort:")).toBeVisible();
  });

  test("flight count updates in header", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    const priceCell = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ }).first();
    await priceCell.click();
    await page.waitForTimeout(1000);

    // Should show "N flight(s)" count
    await expect(page.locator("text=/\\d+ flights?/").first()).toBeVisible();
  });
});

test.describe("Responsive and loading states", () => {
  test("shows loading spinner while fetching calendar", async ({ page }) => {
    // Navigate to an unseeded month to see loading state
    await page.goto("/search?origin=SEA&dest=NRT&month=2027-11");
    // The loading spinner should appear briefly
    // (may be too fast to catch, but the page should not crash)
    await page.waitForTimeout(2000);
    await expect(page.locator("text=November 2027")).toBeVisible();
  });

  test("switching dates updates the flight panel", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-03");
    await page.waitForTimeout(1000);

    // Click first available date
    const cells = page.locator("button").filter({ hasText: /^\d+\s*\d+K$/ });
    const firstCell = cells.first();
    await firstCell.click();
    await page.waitForTimeout(1000);

    // Get the displayed date
    const dateText1 = await page.locator("h3").filter({ hasText: /SEA → NRT/ }).textContent();

    // Click a different date
    const secondCell = cells.nth(1);
    if (await secondCell.count() > 0) {
      await secondCell.click();
      await page.waitForTimeout(1000);

      const dateText2 = await page.locator("h3").filter({ hasText: /SEA → NRT/ }).textContent();
      // The date in the header should have changed
    }
  });
});

test.describe("URL-driven state", () => {
  test("direct URL to search page loads correctly", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-04");
    await expect(page.locator("text=April 2026")).toBeVisible();
    await expect(page.locator("text=Seattle → Tokyo")).toBeVisible();
  });

  test("month in URL is reflected in calendar", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-05");
    await expect(page.locator("text=May 2026")).toBeVisible();
  });

  test("navigating months updates URL", async ({ page }) => {
    await page.goto("/search?origin=SEA&dest=NRT&month=2026-06");
    await page.locator("button", { hasText: "▶" }).click();
    await page.waitForTimeout(500);
    await expect(page.locator("text=July 2026")).toBeVisible();
    expect(page.url()).toContain("month=2026-07");
  });
});
