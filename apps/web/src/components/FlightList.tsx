"use client";

import { useState } from "react";
import type { NormalizedFlight } from "@/lib/types";
import FlightCard from "./FlightCard";

interface FlightListProps {
  flights: NormalizedFlight[];
  date: string;
  origin: string;
  destination: string;
}

type SortKey = "miles" | "departure" | "duration";
type CabinFilter = "all" | "economy" | "business" | "first";

export default function FlightList({ flights, date, origin, destination }: FlightListProps) {
  const [sortBy, setSortBy] = useState<SortKey>("miles");
  const [cabinFilter, setCabinFilter] = useState<CabinFilter>("all");

  const filtered = flights.filter((f) => {
    if (cabinFilter === "all") return true;
    return f.fares.some((fare) => fare.cabin === cabinFilter);
  });

  const sorted = [...filtered].sort((a, b) => {
    switch (sortBy) {
      case "miles":
        const aMin = Math.min(...a.fares.map((f) => f.miles));
        const bMin = Math.min(...b.fares.map((f) => f.miles));
        return aMin - bMin;
      case "departure":
        return a.departure.time.localeCompare(b.departure.time);
      case "duration":
        return a.duration - b.duration;
      default:
        return 0;
    }
  });

  const formatDate = (d: string) => {
    try {
      return new Date(d + "T00:00:00").toLocaleDateString("en-US", {
        weekday: "long",
        month: "long",
        day: "numeric",
        year: "numeric",
      });
    } catch {
      return d;
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-white">
          {origin} → {destination} &middot; {formatDate(date)}
        </h3>
        <span className="text-sm text-gray-500">{sorted.length} flight{sorted.length !== 1 ? "s" : ""}</span>
      </div>

      {/* Controls */}
      <div className="flex flex-wrap gap-2 mb-4">
        <div className="flex items-center gap-1 text-sm">
          <span className="text-gray-500">Sort:</span>
          {(["miles", "departure", "duration"] as SortKey[]).map((key) => (
            <button
              key={key}
              onClick={() => setSortBy(key)}
              className={`px-2 py-1 rounded text-xs ${
                sortBy === key
                  ? "bg-blue-600 text-white"
                  : "bg-gray-800 text-gray-400 hover:text-white"
              }`}
            >
              {key === "miles" ? "Price" : key.charAt(0).toUpperCase() + key.slice(1)}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-1 text-sm">
          <span className="text-gray-500">Cabin:</span>
          {(["all", "economy", "business", "first"] as CabinFilter[]).map((cabin) => (
            <button
              key={cabin}
              onClick={() => setCabinFilter(cabin)}
              className={`px-2 py-1 rounded text-xs ${
                cabinFilter === cabin
                  ? "bg-blue-600 text-white"
                  : "bg-gray-800 text-gray-400 hover:text-white"
              }`}
            >
              {cabin === "all" ? "All" : cabin.charAt(0).toUpperCase() + cabin.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* Flight cards */}
      <div className="space-y-3">
        {sorted.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No flights match your filters</p>
        ) : (
          sorted.map((flight) => <FlightCard key={flight.flightNumber} flight={flight} />)
        )}
      </div>
    </div>
  );
}
