import type { NormalizedFlight, NormalizedFare } from "@/lib/types";

interface FlightCardProps {
  flight: NormalizedFlight;
}

export default function FlightCard({ flight }: FlightCardProps) {
  function formatTime(isoTime: string): string {
    try {
      const d = new Date(isoTime);
      return d.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit", hour12: true });
    } catch {
      return isoTime;
    }
  }

  function formatDuration(mins: number): string {
    const h = Math.floor(mins / 60);
    const m = mins % 60;
    return `${h}h ${m}m`;
  }

  function cabinBadge(fare: NormalizedFare) {
    const colors: Record<string, string> = {
      economy: "bg-blue-900/50 text-blue-300",
      business: "bg-purple-900/50 text-purple-300",
      first: "bg-amber-900/50 text-amber-300",
    };
    return colors[fare.cabin] || colors.economy;
  }

  return (
    <div className="bg-gray-800 rounded-lg border border-gray-700 p-4 hover:border-gray-500 transition-colors">
      {/* Flight header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <span className="font-mono font-bold text-white">{flight.flightNumber}</span>
          <span className="text-gray-500 text-sm">{flight.carrier.name}</span>
        </div>
        {flight.isDirect && (
          <span className="text-xs bg-green-900/50 text-green-400 px-2 py-0.5 rounded">Direct</span>
        )}
      </div>

      {/* Times */}
      <div className="flex items-center gap-4 mb-3">
        <div className="text-center">
          <div className="text-lg font-semibold text-white">{formatTime(flight.departure.time)}</div>
          <div className="text-xs text-gray-500">{flight.departure.airport}</div>
        </div>
        <div className="flex-1 flex flex-col items-center">
          <div className="text-xs text-gray-500">{formatDuration(flight.duration)}</div>
          <div className="w-full h-px bg-gray-600 my-1 relative">
            <div className="absolute right-0 top-1/2 -translate-y-1/2 text-gray-500">▸</div>
          </div>
          {flight.aircraft && (
            <div className="text-xs text-gray-600">{flight.aircraft}</div>
          )}
        </div>
        <div className="text-center">
          <div className="text-lg font-semibold text-white">{formatTime(flight.arrival.time)}</div>
          <div className="text-xs text-gray-500">{flight.arrival.airport}</div>
        </div>
      </div>

      {/* Fares */}
      <div className="flex flex-wrap gap-2">
        {flight.fares
          .sort((a, b) => a.miles - b.miles)
          .map((fare, i) => (
            <div
              key={i}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm ${cabinBadge(fare)}`}
            >
              <span className="font-semibold">{(fare.miles / 1000).toFixed(0)}K mi</span>
              <span className="text-xs opacity-75">+ ${fare.cash.toFixed(2)}</span>
              {fare.seatsRemaining > 0 && fare.seatsRemaining <= 5 && (
                <span className="text-xs opacity-60">{fare.seatsRemaining} left</span>
              )}
              {fare.isSaver && (
                <span className="text-[10px] bg-green-800 text-green-300 px-1 rounded">SAVER</span>
              )}
            </div>
          ))}
      </div>

      {/* Amenities */}
      {flight.amenities && flight.amenities.length > 0 && (
        <div className="flex gap-2 mt-2">
          {flight.amenities.map((a) => (
            <span key={a} className="text-xs text-gray-600">{a}</span>
          ))}
        </div>
      )}
    </div>
  );
}
