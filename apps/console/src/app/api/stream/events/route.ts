import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

const API_URL =
  process.env.RAISIN_API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:18080";

/** Streaming proxy for GET /events/stream — must not buffer the response body. */
export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const target = `${API_URL.replace(/\/$/, "")}/events/stream${url.search}`;

  const headers = new Headers();
  const auth = req.headers.get("authorization");
  const team = req.headers.get("x-team-token");
  const last = req.headers.get("last-event-id");
  if (auth) headers.set("Authorization", auth);
  if (team) headers.set("X-Team-Token", team);
  if (last) headers.set("Last-Event-ID", last);
  headers.set("User-Agent", "raisin-console-proxy/0.1.0");
  headers.set("Accept", "text/event-stream");

  const res = await fetch(target, {
    method: "GET",
    headers,
    cache: "no-store",
  });

  if (!res.ok || !res.body) {
    const body = await res.text();
    return new NextResponse(body, {
      status: res.status,
      headers: { "Content-Type": res.headers.get("Content-Type") ?? "application/json" },
    });
  }

  return new NextResponse(res.body, {
    status: 200,
    headers: {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache, no-transform",
      Connection: "keep-alive",
      "X-Accel-Buffering": "no",
    },
  });
}
