import { NextRequest, NextResponse } from "next/server";
import { searchAirports, AIRPORTS } from "@/lib/airports";

// GET /api/airports?q=tok
export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url);
  const query = searchParams.get("q");

  if (!query || query.length === 0) {
    return NextResponse.json({ airports: AIRPORTS.slice(0, 20) });
  }

  const results = searchAirports(query);
  return NextResponse.json({ airports: results });
}
