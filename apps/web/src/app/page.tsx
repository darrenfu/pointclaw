import RoutePicker from "@/components/RoutePicker";

export default function Home() {
  return (
    <main className="min-h-screen bg-gray-950 flex flex-col items-center justify-center px-4">
      <div className="text-center mb-10">
        <h1 className="text-5xl font-bold text-white mb-3">
          Point<span className="text-blue-400">Claw</span>
        </h1>
        <p className="text-gray-400 text-lg">
          Find award flights. See miles pricing across dates. Spot the saver fares.
        </p>
      </div>

      <RoutePicker />

      <div className="mt-16 text-center text-sm text-gray-600">
        <p>Searching Alaska Airlines award availability</p>
        <p className="mt-1">More airlines coming soon</p>
      </div>
    </main>
  );
}
