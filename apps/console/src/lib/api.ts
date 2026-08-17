export const DEMO_TEAM_ID = "00000000-0000-0000-0000-000000000001";

export async function getTeamToken(): Promise<string> {
  if (typeof window !== "undefined") {
    const t = localStorage.getItem("raisin_team_token");
    if (t) return t;
  }
  return "";
}

/** Browser calls go through Next.js /api/proxy (avoids CORS, keeps API URL server-side). */
export async function apiFetch<T = unknown>(
  path: string,
  init: RequestInit & { token?: string } = {}
): Promise<T> {
  const token = init.token ?? (await getTeamToken());
  const headers = new Headers(init.headers);
  headers.set("Content-Type", "application/json");
  headers.set("User-Agent", "raisin-console/0.1.0");
  if (token) headers.set("X-Team-Token", token);
  const { token: _t, ...rest } = init;
  const res = await fetch(`/api/proxy${path.startsWith("/") ? path : `/${path}`}`, {
    ...rest,
    headers,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error((err as { message?: string }).message ?? "request failed");
  }
  return res.json() as Promise<T>;
}
