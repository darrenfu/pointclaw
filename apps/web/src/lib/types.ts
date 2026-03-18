// Shared types matching the Go scraper output

export interface NormalizedFlight {
  flightNumber: string;
  carrier: { code: string; name: string };
  departure: { airport: string; time: string };
  arrival: { airport: string; time: string };
  duration: number; // minutes
  aircraft: string;
  isDirect: boolean;
  fares: NormalizedFare[];
  amenities: string[];
}

export interface NormalizedFare {
  cabin: "economy" | "business" | "first";
  miles: number;
  cash: number;
  seatsRemaining: number;
  bookingCode: string;
  isSaver: boolean;
}

export interface DateResult {
  date: string; // "2026-06-01"
  status: "success" | "no_flights" | "error" | "loading";
  cheapest: { cabin: string; miles: number; cash: number } | null;
  flightCount: number;
}

export interface CalendarData {
  [date: string]: DateResult;
}

export interface Airport {
  code: string;
  name: string;
  city: string;
  country: string;
  region: string;
}

export interface SearchParams {
  origin: string;
  destination: string;
  month: string; // "2026-06"
}
