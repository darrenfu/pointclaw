"use client";

import { useState, useEffect, useCallback, Suspense } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import CalendarHeatmap from "@/components/CalendarHeatmap";
import FlightList from "@/components/FlightList";
import MonthNav from "@/components/MonthNav";
import { getAirport } from "@/lib/airports";
import type { CalendarData, NormalizedFlight } from "@/lib/types";

function SearchContent() {
  const searchParams = useSearchParams();
  const router = useRouter();

  const origin = searchParams.get("origin") || "";
  const destination = searchParams.get("dest") || "";
  const monthParam = searchParams.get("month") || "";

  const [month, setMonth] = useState(monthParam);
  const [calendarData, setCalendarData] = useState<CalendarData>({});
  const [selectedDate, setSelectedDate] = useState<string | null>(null);
  const [flights, setFlights] = useState<NormalizedFlight[]>([]);
  const [loading, setLoading] = useState(false);

  const originAirport = getAirport(origin);
  const destAirport = getAirport(destination);

  // Fetch calendar data when month changes
  const fetchCalendar = useCallback(async () => {
    if (!origin || !destination || !month) return;

    setLoading(true);
    setCalendarData({});
    setSelectedDate(null);
    setFlights([]);

    try {
      const res = await fetch(`/api/search?origin=${origin}&dest=${destination}&month=${month}`);
      if (res.ok) {
        const data = await res.json();
        setCalendarData(data.calendar || {});
      }
    } catch (err) {
      console.error("Failed to fetch calendar:", err);
    } finally {
      setLoading(false);
    }
  }, [origin, destination, month]);

  useEffect(() => {
    fetchCalendar();
  }, [fetchCalendar]);

  // Fetch flights for selected date
  async function handleSelectDate(date: string) {
    setSelectedDate(date);
    const result = calendarData[date];
    if (!result || result.status !== "success") {
      setFlights([]);
      return;
    }

    try {
      const res = await fetch(`/api/search?origin=${origin}&dest=${destination}&date=${date}`);
      if (res.ok) {
        const data = await res.json();
        setFlights(data.flights || []);
      }
    } catch (err) {
      console.error("Failed to fetch flights:", err);
      setFlights([]);
    }
  }

  function handleMonthChange(newMonth: string) {
    setMonth(newMonth);
    // Use replaceState for immediate URL sync (router.replace is async and may not update URL before assertions)
    const newUrl = `/search?origin=${origin}&dest=${destination}&month=${newMonth}`;
    window.history.replaceState(null, "", newUrl);
    router.replace(newUrl);
  }

  if (!origin || !destination) {
    return (
      <div className="min-h-screen bg-gray-950 flex items-center justify-center">
        <p className="text-gray-500">Missing search parameters. <a href="/" className="text-blue-400 hover:underline">Go back</a></p>
      </div>
    );
  }

  return (
    <main className="min-h-screen bg-gray-950 px-4 py-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <a href="/" className="text-blue-400 text-sm hover:underline mb-1 block">← Back</a>
            <h1 className="text-2xl font-bold text-white">
              {originAirport?.city || origin} → {destAirport?.city || destination}
            </h1>
            <p className="text-gray-500 text-sm">
              {origin} → {destination} &middot; Award availability
            </p>
          </div>
        </div>

        {/* Month navigation */}
        <div className="mb-6">
          <MonthNav month={month} onChange={handleMonthChange} />
        </div>

        {/* Calendar */}
        <div className="mb-8">
          {loading ? (
            <div className="bg-gray-900 rounded-xl p-8 border border-gray-700 text-center">
              <div className="animate-spin w-8 h-8 border-2 border-blue-400 border-t-transparent rounded-full mx-auto mb-3" />
              <p className="text-gray-500">Loading availability...</p>
            </div>
          ) : (
            <CalendarHeatmap
              month={month}
              data={calendarData}
              onSelectDate={handleSelectDate}
              selectedDate={selectedDate}
            />
          )}
        </div>

        {/* Flight list */}
        {selectedDate && (
          <FlightList
            flights={flights}
            date={selectedDate}
            origin={origin}
            destination={destination}
          />
        )}
      </div>
    </main>
  );
}

export default function SearchPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-gray-950 flex items-center justify-center">
        <p className="text-gray-500">Loading...</p>
      </div>
    }>
      <SearchContent />
    </Suspense>
  );
}
