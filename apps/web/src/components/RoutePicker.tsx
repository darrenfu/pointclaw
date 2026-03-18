"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import AirportSearch from "./AirportSearch";
import { ORIGIN_CODES } from "@/lib/airports";

export default function RoutePicker() {
  const [origin, setOrigin] = useState("");
  const [destination, setDestination] = useState("");
  const router = useRouter();

  function handleSearch() {
    if (!origin || !destination) return;
    const now = new Date();
    const month = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
    router.push(`/search?origin=${origin}&dest=${destination}&month=${month}`);
  }

  return (
    <div className="w-full max-w-lg mx-auto space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <AirportSearch
          label="From"
          value={origin}
          onChange={setOrigin}
          filterCodes={ORIGIN_CODES}
        />
        <AirportSearch
          label="To"
          value={destination}
          onChange={setDestination}
        />
      </div>
      <button
        onClick={handleSearch}
        disabled={!origin || !destination}
        className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white font-semibold py-3 rounded-lg transition-colors"
      >
        Search Award Flights
      </button>
    </div>
  );
}
