import { NextRequest, NextResponse } from "next/server";

const API_URL =
  process.env.RAISIN_API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:18080";
const JWT_SECRET = process.env.JWT_SECRET ?? "dev-jwt-secret-change-me-in-production";
const DEMO_TEAM_ID = "00000000-0000-0000-0000-000000000001";
const DEMO_USER_ID = "00000000-0000-0000-0000-000000000002";

export async function POST(req: NextRequest) {
  const body = await req.json().catch(() => ({}));
  const email = (body as { email?: string }).email ?? "demo@raisin.run";

  // Dev/demo auth: mint a team JWT via the API
  const res = await fetch(`${API_URL}/console/token`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "User-Agent": "raisin-console" },
    body: JSON.stringify({
      team_id: DEMO_TEAM_ID,
      user_id: DEMO_USER_ID,
      secret: JWT_SECRET,
      email,
    }),
  });
  if (!res.ok) {
    const text = await res.text();
    return NextResponse.json({ error: text }, { status: 500 });
  }
  const data = await res.json();
  return NextResponse.json({ token: data.token, email });
}
