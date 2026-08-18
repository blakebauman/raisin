"use client";

import { useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { streamEmailEvents } from "@/lib/events-stream";
import { formatAbsolute, formatRelative } from "@/lib/format";
import {
  BackLink,
  EmptyState,
  LiveDot,
  MailFrame,
  Msg,
  PropertyRow,
  StatusChip,
  TableSkeleton,
} from "@/components/ui";

type Email = {
  id: string;
  from: string;
  to: string[];
  subject: string;
  status: string;
  html?: string;
  text?: string;
  created_at: string;
  sent_at?: string;
  provider_message_id?: string;
};

type Event = { id: string; type: string; data: unknown; created_at: string };
type Attachment = { id: string; filename: string; content_type: string; size_bytes: number };

function eventDot(type: string) {
  if (type.includes("bounce") || type.includes("fail") || type.includes("complain")) return "bg-red-400";
  if (type.includes("deliver") || type.includes("sent")) return "bg-emerald-400";
  if (type.includes("open") || type.includes("click")) return "bg-orange-400";
  return "bg-zinc-500";
}

export default function EmailDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const [email, setEmail] = useState<Email | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [atts, setAtts] = useState<Attachment[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [live, setLive] = useState(false);
  const seen = useRef(new Set<string>());

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    (async () => {
      try {
        const [e, ev, a] = await Promise.all([
          apiFetch<Email>(`/emails/${id}`),
          apiFetch<{ data: Event[] }>(`/emails/${id}/events`),
          apiFetch<{ data: Attachment[] }>(`/emails/${id}/attachments`),
        ]);
        if (cancelled) return;
        setEmail(e);
        const list = ev.data ?? [];
        seen.current = new Set(list.map((x) => x.id));
        setEvents(list);
        setAtts(a.data ?? []);
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : "failed to load");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id]);

  useEffect(() => {
    if (!id) return;
    const ac = new AbortController();
    let cancelled = false;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;

    const connect = () => {
      if (cancelled) return;
      streamEmailEvents({
        emailId: id,
        signal: ac.signal,
        onOpen: () => {
          if (!cancelled) setLive(true);
        },
        onEvent: (ev) => {
          if (seen.current.has(ev.id)) return;
          seen.current.add(ev.id);
          setEvents((prev) => [
            ...prev,
            {
              id: ev.id,
              type: ev.type,
              data: ev.data,
              created_at: ev.created_at,
            },
          ]);
          if (
            ev.type === "email.delivered" ||
            ev.type === "email.bounced" ||
            ev.type === "email.complained" ||
            ev.type === "email.failed" ||
            ev.type === "email.sent"
          ) {
            apiFetch<Email>(`/emails/${id}`)
              .then((e) => setEmail(e))
              .catch(() => {});
          }
        },
      })
        .then(() => {
          if (!cancelled) {
            setLive(false);
            retryTimer = setTimeout(connect, 1500);
          }
        })
        .catch((e: unknown) => {
          if (cancelled || (e instanceof DOMException && e.name === "AbortError")) return;
          setLive(false);
          retryTimer = setTimeout(connect, 2500);
        });
    };

    connect();
    return () => {
      cancelled = true;
      ac.abort();
      if (retryTimer) clearTimeout(retryTimer);
    };
  }, [id]);

  async function reload() {
    const [e, ev, a] = await Promise.all([
      apiFetch<Email>(`/emails/${id}`),
      apiFetch<{ data: Event[] }>(`/emails/${id}/events`),
      apiFetch<{ data: Attachment[] }>(`/emails/${id}/attachments`),
    ]);
    setEmail(e);
    const list = ev.data ?? [];
    seen.current = new Set(list.map((x) => x.id));
    setEvents(list);
    setAtts(a.data ?? []);
  }

  async function cancel() {
    setBusy(true);
    setError("");
    try {
      await apiFetch(`/emails/${id}/cancel`, { method: "POST", body: "{}" });
      await reload();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "cancel failed");
    } finally {
      setBusy(false);
    }
  }

  if (error && !email) {
    return <p className="text-[13px] text-red-400">{error}</p>;
  }
  if (!email) {
    return <TableSkeleton rows={10} />;
  }

  const canCancel = email.status === "queued" || email.status === "scheduled";

  return (
    <div>
      <BackLink href="/emails">Emails</BackLink>

      <div className="mt-3 flex flex-wrap items-start justify-between gap-3">
        <h1 className="min-w-0 text-[17px] font-medium tracking-tight text-zinc-50 text-balance">
          {email.subject || "(no subject)"}
        </h1>
        <div className="flex shrink-0 items-center gap-2">
          <LiveDot live={live} />
          {canCancel && (
            <button type="button" disabled={busy} onClick={cancel} className="btn-secondary">
              {busy ? "Canceling…" : "Cancel send"}
            </button>
          )}
        </div>
      </div>
      <Msg tone="error">{error}</Msg>

      <div className="mt-5 grid items-start gap-8 lg:grid-cols-[minmax(0,1fr)_15.5rem]">
        <div className="min-w-0">
          {email.html ? (
            <MailFrame html={email.html} />
          ) : (
            <pre className="whitespace-pre-wrap text-[13px] text-[var(--muted)]">{email.text || "No body"}</pre>
          )}

          <h2 className="mt-8 mb-1 text-[13px] font-medium text-zinc-300">Activity</h2>
          {events.length === 0 ? (
            <EmptyState title="No events yet" hint="Delivery and engagement events appear here live." />
          ) : (
            <ol>
              {events.map((ev) => (
                <li key={ev.id} className="flex gap-3 border-t border-[var(--border)] py-2.5">
                  <div className={`mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full ${eventDot(ev.type)}`} />
                  <div className="min-w-0 flex-1">
                    <div className="font-mono text-[12px] text-zinc-100">{ev.type}</div>
                    <div className="text-[12px] tabular-nums text-[var(--muted)]" title={formatAbsolute(ev.created_at)}>
                      {formatRelative(ev.created_at)}
                    </div>
                  </div>
                </li>
              ))}
            </ol>
          )}
        </div>

        <aside>
          <dl>
            <PropertyRow label="Status">
              <StatusChip status={email.status} />
            </PropertyRow>
            <PropertyRow label="From">
              <span className="break-all">{email.from}</span>
            </PropertyRow>
            <PropertyRow label="To">
              <span className="break-all">{email.to?.join(", ") || "—"}</span>
            </PropertyRow>
            <PropertyRow label="Created">
              <span title={formatAbsolute(email.created_at)}>{formatRelative(email.created_at)}</span>
            </PropertyRow>
            {email.sent_at && (
              <PropertyRow label="Sent">
                <span title={formatAbsolute(email.sent_at)}>{formatRelative(email.sent_at)}</span>
              </PropertyRow>
            )}
            {email.provider_message_id && (
              <PropertyRow label="Message ID">
                <span className="break-all font-mono text-[11px] text-[var(--muted)]">{email.provider_message_id}</span>
              </PropertyRow>
            )}
          </dl>
          {atts.length > 0 && (
            <div className="mt-4 border-t border-[var(--border)] pt-3">
              <div className="mb-2 text-[13px] text-[var(--muted)]">Attachments</div>
              <ul className="space-y-1">
                {atts.map((a) => (
                  <li key={a.id} className="break-all font-mono text-[12px] text-zinc-300">
                    {a.filename}
                    <span className="text-[var(--muted)]"> ({a.size_bytes}b)</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </aside>
      </div>
    </div>
  );
}
