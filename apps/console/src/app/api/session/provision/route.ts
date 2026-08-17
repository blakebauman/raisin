import { NextRequest, NextResponse } from "next/server";
import { auth } from "@/lib/auth";
import { headers } from "next/headers";

const API_URL =
  process.env.RAISIN_API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:18080";
const JWT_SECRET = process.env.JWT_SECRET ?? "dev-jwt-secret-change-me-in-production";

/** Mint a Raisin team JWT for the current Better Auth session. */
export async function POST(_req: NextRequest) {
  const session = await auth.api.getSession({ headers: await headers() });
  if (!session?.user?.email) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }

  const res = await fetch(`${API_URL}/console/provision`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "User-Agent": "raisin-console" },
    body: JSON.stringify({
      secret: JWT_SECRET,
      email: session.user.email,
      name: session.user.name ?? session.user.email,
    }),
  });
  if (!res.ok) {
    const text = await res.text();
    return NextResponse.json({ error: text }, { status: 500 });
  }
  const data = await res.json();
  return NextResponse.json(data);
}
