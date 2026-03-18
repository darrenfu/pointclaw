"use client";

import { useState, useRef, useEffect, useId } from "react";
import { searchAirports } from "@/lib/airports";
import type { Airport } from "@/lib/types";

interface AirportSearchProps {
  label: string;
  value: string;
  onChange: (code: string) => void;
  filterCodes?: string[]; // only show these codes
}

export default function AirportSearch({ label, value, onChange, filterCodes }: AirportSearchProps) {
  const [query, setQuery] = useState("");
  const [isOpen, setIsOpen] = useState(false);
  const [results, setResults] = useState<Airport[]>([]);
  const [selectedAirport, setSelectedAirport] = useState<Airport | null>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const inputId = useId();

  useEffect(() => {
    if (query.length > 0) {
      let airports = searchAirports(query);
      if (filterCodes) {
        airports = airports.filter((a) => filterCodes.includes(a.code));
      }
      setResults(airports);
      setIsOpen(airports.length > 0);
    } else {
      setResults([]);
      setIsOpen(false);
    }
  }, [query, filterCodes]);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  function handleSelect(airport: Airport) {
    setSelectedAirport(airport);
    setQuery("");
    setIsOpen(false);
    onChange(airport.code);
  }

  function handleClear() {
    setSelectedAirport(null);
    setQuery("");
    onChange("");
  }

  return (
    <div ref={wrapperRef} className="relative">
      <label htmlFor={inputId} className="block text-sm font-medium text-gray-400 mb-1">{label}</label>
      {selectedAirport ? (
        <div className="flex items-center gap-2 bg-gray-800 border border-gray-600 rounded-lg px-3 py-2">
          <span className="font-mono font-bold text-white">{selectedAirport.code}</span>
          <span className="text-gray-400 text-sm truncate">{selectedAirport.city}</span>
          <button onClick={handleClear} className="ml-auto text-gray-500 hover:text-white text-lg">&times;</button>
        </div>
      ) : (
        <input
          id={inputId}
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="City or airport code..."
          className="w-full bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
        />
      )}
      {isOpen && (
        <ul className="absolute z-50 mt-1 w-full bg-gray-800 border border-gray-600 rounded-lg shadow-xl max-h-60 overflow-auto">
          {results.map((airport) => (
            <li
              key={airport.code}
              onClick={() => handleSelect(airport)}
              className="px-3 py-2 hover:bg-gray-700 cursor-pointer flex items-center gap-2"
            >
              <span className="font-mono font-bold text-blue-400 w-10">{airport.code}</span>
              <span className="text-white">{airport.city}</span>
              <span className="text-gray-500 text-sm ml-auto">{airport.country}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
