"use client";

interface MonthNavProps {
  month: string; // "2026-06"
  onChange: (month: string) => void;
}

export default function MonthNav({ month, onChange }: MonthNavProps) {
  const [year, mon] = month.split("-").map(Number);
  const date = new Date(year, mon - 1);
  const label = date.toLocaleDateString("en-US", { month: "long", year: "numeric" });

  const now = new Date();
  const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
  const maxDate = new Date(now);
  maxDate.setMonth(maxDate.getMonth() + 12);
  const maxMonth = `${maxDate.getFullYear()}-${String(maxDate.getMonth() + 1).padStart(2, "0")}`;

  function prev() {
    const d = new Date(year, mon - 2);
    const m = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
    if (m >= currentMonth) onChange(m);
  }

  function next() {
    const d = new Date(year, mon);
    const m = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
    if (m <= maxMonth) onChange(m);
  }

  const canPrev = month > currentMonth;
  const canNext = month < maxMonth;

  return (
    <div className="flex items-center justify-center gap-4">
      <button
        onClick={prev}
        disabled={!canPrev}
        className="px-3 py-1 text-xl text-gray-400 hover:text-white disabled:text-gray-700 disabled:cursor-not-allowed"
      >
        ◀
      </button>
      <h2 className="text-xl font-semibold text-white min-w-48 text-center">{label}</h2>
      <button
        onClick={next}
        disabled={!canNext}
        className="px-3 py-1 text-xl text-gray-400 hover:text-white disabled:text-gray-700 disabled:cursor-not-allowed"
      >
        ▶
      </button>
    </div>
  );
}
