"use client";

import type { CalendarData, DateResult } from "@/lib/types";

interface CalendarHeatmapProps {
  month: string; // "2026-06"
  data: CalendarData;
  onSelectDate: (date: string) => void;
  selectedDate: string | null;
}

export default function CalendarHeatmap({ month, data, onSelectDate, selectedDate }: CalendarHeatmapProps) {
  const [year, mon] = month.split("-").map(Number);
  const firstDay = new Date(year, mon - 1, 1);
  const daysInMonth = new Date(year, mon, 0).getDate();
  const startDow = firstDay.getDay(); // 0=Sun

  const dayHeaders = ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"];

  // Build cells: leading empty + day cells
  const cells: (number | null)[] = [];
  for (let i = 0; i < startDow; i++) cells.push(null);
  for (let d = 1; d <= daysInMonth; d++) cells.push(d);

  function getDateStr(day: number): string {
    return `${year}-${String(mon).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
  }

  function getCellColor(result: DateResult | undefined): string {
    if (!result || result.status === "loading") return "bg-gray-800 animate-pulse";
    if (result.status === "error") return "bg-red-900/30";
    if (result.status === "no_flights" || !result.cheapest) return "bg-gray-800";

    const miles = result.cheapest.miles;
    // Color scale based on miles cost
    if (miles <= 15000) return "bg-green-600";
    if (miles <= 25000) return "bg-green-500";
    if (miles <= 35000) return "bg-green-400/80";
    if (miles <= 50000) return "bg-yellow-500/80";
    if (miles <= 75000) return "bg-orange-500/80";
    return "bg-red-500/80";
  }

  function formatMiles(miles: number): string {
    if (miles >= 1000) return `${Math.round(miles / 1000)}K`;
    return String(miles);
  }

  return (
    <div className="bg-gray-900 rounded-xl p-4 border border-gray-700">
      {/* Day headers */}
      <div className="grid grid-cols-7 gap-1 mb-2">
        {dayHeaders.map((d) => (
          <div key={d} className="text-center text-xs text-gray-500 font-medium py-1">
            {d}
          </div>
        ))}
      </div>

      {/* Calendar grid */}
      <div className="grid grid-cols-7 gap-1">
        {cells.map((day, i) => {
          if (day === null) {
            return <div key={`empty-${i}`} className="aspect-square" />;
          }

          const dateStr = getDateStr(day);
          const result = data[dateStr];
          const isSelected = dateStr === selectedDate;
          const hasData = result && result.status === "success" && result.cheapest;
          const cellColor = getCellColor(result);

          return (
            <button
              key={dateStr}
              onClick={() => onSelectDate(dateStr)}
              className={`
                aspect-square rounded-lg flex flex-col items-center justify-center text-xs transition-all
                ${cellColor}
                ${isSelected ? "ring-2 ring-blue-400 ring-offset-1 ring-offset-gray-900" : ""}
                ${hasData ? "hover:brightness-110 cursor-pointer" : "cursor-default"}
              `}
            >
              <span className={`font-medium ${hasData ? "text-white" : "text-gray-500"}`}>
                {day}
              </span>
              {hasData && result.cheapest && (
                <span className="text-[10px] font-bold text-white/90">
                  {formatMiles(result.cheapest.miles)}
                </span>
              )}
              {result?.status === "no_flights" && (
                <span className="text-[10px] text-gray-600">--</span>
              )}
            </button>
          );
        })}
      </div>

      {/* Legend */}
      <div className="flex items-center justify-center gap-2 mt-4 text-xs text-gray-500">
        <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-green-600" /> Saver</span>
        <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-yellow-500/80" /> Mid</span>
        <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-red-500/80" /> High</span>
        <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-gray-800 border border-gray-700" /> N/A</span>
      </div>
    </div>
  );
}
