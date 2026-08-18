"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { EmptyState, Msg, PageHeader, SectionLabel, StatusChip } from "@/components/ui";
import { streamEmailEvents, type LiveEmailEvent } from "@/lib/events-stream";

function typeTone(type: string) {
  if (type.includes("bounce") || type.includes("fail") || type.includes("complain")) {
    return "text-red-300";
  }
  if (type.includes("click") || type.includes("open")) return "text-orange-300";
  if (type.includes("deliver")) return "text-emerald-300";
  return "text-zinc-200";
}

function emailIdFrom(ev: LiveEmailEvent): string | null {
  if (ev.data && typeof ev.data === "object" && "email_id" in ev.data) {
    const id = (ev.data as { email_id?: unknown }).email_id;
    return typeof id === "string" ? id : null;
  }
  return null;
}

export default function ActivityPage() {
  const [events, setEvents] = useState<LiveEmailEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState("");
  const lastId = useRef<string | undefined>(undefined);
  const seen = useRef(new Set<string>());

  useEffect(() => {
    const ac = new AbortController();
    let cancelled = false;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;

    const connect = () => {
      if (cancelled) return;
      setConnected(false);
      streamEmailEvents({
        lastEventId: lastId.current,
        signal: ac.signal,
        onOpen: () => {
          if (!cancelled) {
            setConnected(true);
            setError("");
          }
        },
        onEvent: (ev) => {
          if (seen.current.has(ev.id)) return;
          seen.current.add(ev.id);
          lastId.current = ev.id;
          setEvents((prev) => [ev, ...prev].slice(0, 200));
        },
      })
        .then(() => {
          if (!cancelled) {
            setConnected(false);
            retryTimer = setTimeout(connect, 1500);
          }
        })
        .catch((e: unknown) => {
          if (cancelled || (e instanceof DOMException && e.name === "AbortError")) return;
          setConnected(false);
          setError(e instanceof Error ? e.message : "stream failed");
          retryTimer = setTimeout(connect, 2500);
        });
    };

    connect();
    return () => {
      cancelled = true;
      ac.abort();
      if (retryTimer) clearTimeout(retryTimer);
    };
  }, []);

  return (
    <div>
      <PageHeader
        title="Activity"
        description="Live delivery and engagement events for this team. Opens and clicks appear as soon as trackers fire."
        actions={
          <span
            className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11px] uppercase tracking-wide ${
              connected
                ? "border-emerald-800/80 bg-emerald-950/40 text-emerald-300"
                : "border-zinc-700 bg-zinc-900 text-zinc-500"
            }`}
          >
            <span
              className={`h-1.5 w-1.5 rounded-full ${connected ? "bg-emerald-400" : "bg-zinc-600"}`}
            />
            {connected ? "Live" : "Reconnecting"}
          </span>
        }
      />
      <Msg tone="error">{error}</Msg>

      <section className="mt-4 rounded-lg border border-zinc-800 bg-[var(--panel)]/70 p-5">
        <SectionLabel>Event stream</SectionLabel>
        {events.length === 0 ? (
          <EmptyState
            title="Waiting for events"
            hint="Send a message, then open or click it — events appear here in realtime."
          />
        ) : (
          <ul className="divide-y divide-zinc-800/80">
            {events.map((ev) => {
              const eid = emailIdFrom(ev);
              return (
                <li key={ev.id} className="flex flex-wrap items-baseline gap-x-3 gap-y-1 py-3 text-sm">
                  <span className={`font-mono text-xs ${typeTone(ev.type)}`}>{ev.type}</span>
                  <span className="text-xs tabular-nums text-zinc-500">
                    {ev.created_at ? new Date(ev.created_at).toLocaleString() : "—"}
                  </span>
                  {eid ? (
                    <Link
                      href={`/emails/${eid}`}
                      className="font-mono text-xs text-zinc-400 hover:text-orange-300"
                    >
                      {eid.slice(0, 8)}…
                    </Link>
                  ) : null}
                  <StatusChip status={ev.type.replace(/^email\./, "")} />
                </li>
              );
            })}
          </ul>
        )}
      </section>
    </div>
  );
}
