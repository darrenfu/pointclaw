import type { Airport } from "./types";

// MVP airport database — origins + popular destinations
export const AIRPORTS: Airport[] = [
  // Origins
  { code: "SEA", name: "Seattle-Tacoma International", city: "Seattle", country: "US", region: "north-america" },
  { code: "LAX", name: "Los Angeles International", city: "Los Angeles", country: "US", region: "north-america" },
  { code: "SFO", name: "San Francisco International", city: "San Francisco", country: "US", region: "north-america" },
  { code: "YVR", name: "Vancouver International", city: "Vancouver", country: "CA", region: "north-america" },
  { code: "PDX", name: "Portland International", city: "Portland", country: "US", region: "north-america" },

  // Japan
  { code: "NRT", name: "Narita International", city: "Tokyo", country: "JP", region: "asia-pacific" },
  { code: "HND", name: "Haneda Airport", city: "Tokyo", country: "JP", region: "asia-pacific" },
  { code: "KIX", name: "Kansai International", city: "Osaka", country: "JP", region: "asia-pacific" },

  // Korea
  { code: "ICN", name: "Incheon International", city: "Seoul", country: "KR", region: "asia-pacific" },

  // SE Asia
  { code: "TPE", name: "Taiwan Taoyuan International", city: "Taipei", country: "TW", region: "asia-pacific" },
  { code: "HKG", name: "Hong Kong International", city: "Hong Kong", country: "HK", region: "asia-pacific" },
  { code: "SIN", name: "Changi Airport", city: "Singapore", country: "SG", region: "asia-pacific" },
  { code: "BKK", name: "Suvarnabhumi Airport", city: "Bangkok", country: "TH", region: "asia-pacific" },
  { code: "MNL", name: "Ninoy Aquino International", city: "Manila", country: "PH", region: "asia-pacific" },

  // Oceania
  { code: "SYD", name: "Sydney Airport", city: "Sydney", country: "AU", region: "oceania" },
  { code: "MEL", name: "Melbourne Airport", city: "Melbourne", country: "AU", region: "oceania" },
  { code: "AKL", name: "Auckland Airport", city: "Auckland", country: "NZ", region: "oceania" },
  { code: "NAN", name: "Nadi International", city: "Nadi", country: "FJ", region: "oceania" },

  // US
  { code: "JFK", name: "John F. Kennedy International", city: "New York", country: "US", region: "north-america" },
  { code: "BOS", name: "Logan International", city: "Boston", country: "US", region: "north-america" },
  { code: "MIA", name: "Miami International", city: "Miami", country: "US", region: "north-america" },
  { code: "ORD", name: "O'Hare International", city: "Chicago", country: "US", region: "north-america" },
  { code: "DFW", name: "Dallas/Fort Worth International", city: "Dallas", country: "US", region: "north-america" },
  { code: "DCA", name: "Ronald Reagan National", city: "Washington D.C.", country: "US", region: "north-america" },
  { code: "HNL", name: "Daniel K. Inouye International", city: "Honolulu", country: "US", region: "north-america" },
  { code: "ANC", name: "Ted Stevens Anchorage International", city: "Anchorage", country: "US", region: "north-america" },

  // Canada
  { code: "YYZ", name: "Toronto Pearson International", city: "Toronto", country: "CA", region: "north-america" },
  { code: "YUL", name: "Montréal-Trudeau International", city: "Montreal", country: "CA", region: "north-america" },

  // Islands
  { code: "OGG", name: "Kahului Airport", city: "Maui", country: "US", region: "islands" },
  { code: "KOA", name: "Ellison Onizuka Kona International", city: "Kona", country: "US", region: "islands" },
  { code: "LIH", name: "Lihue Airport", city: "Kauai", country: "US", region: "islands" },
  { code: "CUN", name: "Cancún International", city: "Cancún", country: "MX", region: "islands" },
  { code: "SJD", name: "Los Cabos International", city: "Cabo San Lucas", country: "MX", region: "islands" },
  { code: "PVR", name: "Gustavo Díaz Ordaz International", city: "Puerto Vallarta", country: "MX", region: "islands" },

  // Europe
  { code: "LHR", name: "Heathrow Airport", city: "London", country: "GB", region: "europe" },
  { code: "CDG", name: "Charles de Gaulle Airport", city: "Paris", country: "FR", region: "europe" },
  { code: "FRA", name: "Frankfurt Airport", city: "Frankfurt", country: "DE", region: "europe" },
  { code: "FCO", name: "Leonardo da Vinci International", city: "Rome", country: "IT", region: "europe" },
  { code: "BCN", name: "Barcelona-El Prat Airport", city: "Barcelona", country: "ES", region: "europe" },

  // South America
  { code: "LIM", name: "Jorge Chávez International", city: "Lima", country: "PE", region: "south-america" },
  { code: "BOG", name: "El Dorado International", city: "Bogotá", country: "CO", region: "south-america" },
  { code: "SCL", name: "Arturo Merino Benítez International", city: "Santiago", country: "CL", region: "south-america" },
  { code: "EZE", name: "Ministro Pistarini International", city: "Buenos Aires", country: "AR", region: "south-america" },
];

export const ORIGIN_CODES = ["SEA", "LAX", "SFO", "YVR", "PDX"];

export function getAirport(code: string): Airport | undefined {
  return AIRPORTS.find((a) => a.code === code);
}

export function searchAirports(query: string): Airport[] {
  const q = query.toLowerCase();
  return AIRPORTS.filter(
    (a) =>
      a.code.toLowerCase().includes(q) ||
      a.name.toLowerCase().includes(q) ||
      a.city.toLowerCase().includes(q)
  ).slice(0, 10);
}
