import { NextRequest, NextResponse } from "next/server";

const API_URL =
  process.env.RAISIN_API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:18080";

async function proxy(req: NextRequest, path: string[]) {
  const url = new URL(req.url);
  const target = `${API_URL.replace(/\/$/, "")}/${path.join("/")}${url.search}`;
  const headers = new Headers();
  const auth = req.headers.get("authorization");
  const team = req.headers.get("x-team-token");
  const idem = req.headers.get("idempotency-key");
  if (auth) headers.set("Authorization", auth);
  if (team) headers.set("X-Team-Token", team);
  if (idem) headers.set("Idempotency-Key", idem);
  headers.set("User-Agent", "raisin-console-proxy/0.1.0");
  headers.set("Content-Type", "application/json");

  const init: RequestInit = { method: req.method, headers };
  if (req.method !== "GET" && req.method !== "HEAD") {
    init.body = await req.text();
  }
  const res = await fetch(target, init);
  const body = await res.text();
  return new NextResponse(body, {
    status: res.status,
    headers: { "Content-Type": res.headers.get("Content-Type") ?? "application/json" },
  });
}

export async function GET(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function POST(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function PATCH(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function DELETE(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function PUT(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
