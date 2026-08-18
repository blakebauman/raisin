import { getTeamToken } from "@/lib/api";

export type LiveEmailEvent = {
  id: string;
  type: string;
  created_at: string;
  data: unknown;
};

export type StreamEventsOptions = {
  types?: string[];
  emailId?: string;
  lastEventId?: string;
  signal?: AbortSignal;
  onEvent: (ev: LiveEmailEvent) => void;
  /** Fired once the SSE response is open (before any events). */
  onOpen?: () => void;
};

/**
 * Subscribe to team email events via SSE (through the streaming console proxy).
 * Uses fetch + ReadableStream so X-Team-Token auth works (EventSource cannot set headers).
 */
export async function streamEmailEvents(opts: StreamEventsOptions): Promise<void> {
  const token = await getTeamToken();
  const params = new URLSearchParams();
  if (opts.types?.length) params.set("types", opts.types.join(","));
  if (opts.emailId) params.set("email_id", opts.emailId);

  const headers: Record<string, string> = {
    "User-Agent": "raisin-console/0.1.0",
    Accept: "text/event-stream",
  };
  if (token) headers["X-Team-Token"] = token;
  if (opts.lastEventId) headers["Last-Event-ID"] = opts.lastEventId;

  const qs = params.toString();
  const res = await fetch(`/api/stream/events${qs ? `?${qs}` : ""}`, {
    method: "GET",
    headers,
    signal: opts.signal,
    cache: "no-store",
  });

  if (!res.ok || !res.body) {
    const errBody = await res.text().catch(() => res.statusText);
    throw new Error(errBody || `stream failed (${res.status})`);
  }

  opts.onOpen?.();

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });

    let sep: number;
    while ((sep = buf.indexOf("\n\n")) >= 0) {
      const raw = buf.slice(0, sep);
      buf = buf.slice(sep + 2);
      const ev = parseSSEFrame(raw);
      if (ev) opts.onEvent(ev);
    }
  }
}

function parseSSEFrame(raw: string): LiveEmailEvent | null {
  let data = "";
  for (const line of raw.split("\n")) {
    if (line.startsWith(":") || line.trim() === "") continue;
    if (line.startsWith("data:")) {
      data += (data ? "\n" : "") + line.slice(5).trimStart();
    }
  }
  if (!data) return null;
  try {
    return JSON.parse(data) as LiveEmailEvent;
  } catch {
    return null;
  }
}
