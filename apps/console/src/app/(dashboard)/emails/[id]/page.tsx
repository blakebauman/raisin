"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { EmptyState, Msg, PageHeader, SectionLabel, StatusChip } from "@/components/ui";

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

export default function EmailDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const [email, setEmail] = useState<Email | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [atts, setAtts] = useState<Attachment[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

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
        setEvents(ev.data ?? []);
        setAtts(a.data ?? []);
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : "failed to load");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id]);

  async function reload() {
    const [e, ev, a] = await Promise.all([
      apiFetch<Email>(`/emails/${id}`),
      apiFetch<{ data: Event[] }>(`/emails/${id}/events`),
      apiFetch<{ data: Attachment[] }>(`/emails/${id}/attachments`),
    ]);
    setEmail(e);
    setEvents(ev.data ?? []);
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
    return <p className="text-sm text-red-400">{error}</p>;
  }
  if (!email) {
    return <p className="text-sm text-zinc-500">Loading…</p>;
  }

  const canCancel = email.status === "queued" || email.status === "scheduled";

  return (
    <div>
      <Link href="/emails" className="text-xs text-zinc-500 hover:text-zinc-300">
        ← Emails
      </Link>
      <div className="mt-3">
        <PageHeader
          title={email.subject || "(no subject)"}
          description={`${email.from} → ${email.to?.join(", ") ?? ""}`}
          actions={
            <div className="flex items-center gap-2">
              <StatusChip status={email.status} />
              {canCancel && (
                <button type="button" disabled={busy} onClick={cancel} className="btn-secondary text-xs">
                  {busy ? "Canceling…" : "Cancel send"}
                </button>
              )}
            </div>
          }
        />
      </div>
      <Msg tone="error">{error}</Msg>

      <div className="mt-2 grid gap-6 lg:grid-cols-2">
        <section className="rounded-lg border border-zinc-800 bg-[var(--panel)]/70 p-5">
          <SectionLabel>Timeline</SectionLabel>
          {events.length === 0 ? (
            <EmptyState title="No events yet" hint="Delivery and engagement events appear here." />
          ) : (
            <ol className="space-y-3">
              {events.map((ev) => (
                <li key={ev.id} className="flex gap-3 text-sm">
                  <div className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-orange-500" />
                  <div>
                    <div className="font-mono text-xs text-zinc-100">{ev.type}</div>
                    <div className="text-xs tabular-nums text-zinc-500">
                      {new Date(ev.created_at).toLocaleString()}
                    </div>
                  </div>
                </li>
              ))}
            </ol>
          )}
        </section>

        <section className="rounded-lg border border-zinc-800 bg-[var(--panel)]/70 p-5">
          <SectionLabel>Preview</SectionLabel>
          {email.html ? (
            <iframe
              title="preview"
              className="min-h-64 w-full rounded border border-zinc-800 bg-white"
              srcDoc={email.html}
            />
          ) : (
            <pre className="whitespace-pre-wrap text-xs text-zinc-400">{email.text}</pre>
          )}
          {atts.length > 0 && (
            <div className="mt-4">
              <h3 className="mb-2 text-xs text-zinc-500">Attachments</h3>
              <ul className="space-y-1 text-sm text-zinc-300">
                {atts.map((a) => (
                  <li key={a.id} className="font-mono text-xs">
                    {a.filename} ({a.size_bytes}b)
                  </li>
                ))}
              </ul>
            </div>
          )}
          {email.provider_message_id && (
            <p className="mt-4 font-mono text-xs text-zinc-600">msg: {email.provider_message_id}</p>
          )}
        </section>
      </div>
    </div>
  );
}
