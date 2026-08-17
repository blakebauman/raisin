"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { apiFetch } from "@/lib/api";

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

  useEffect(() => {
    if (!id) return;
    Promise.all([
      apiFetch<Email>(`/emails/${id}`),
      apiFetch<{ data: Event[] }>(`/emails/${id}/events`),
      apiFetch<{ data: Attachment[] }>(`/emails/${id}/attachments`),
    ])
      .then(([e, ev, a]) => {
        setEmail(e);
        setEvents(ev.data ?? []);
        setAtts(a.data ?? []);
      })
      .catch((e) => setError(e.message));
  }, [id]);

  if (error) {
    return <p className="text-red-400 text-sm">{error}</p>;
  }
  if (!email) {
    return <p className="text-zinc-500 text-sm">Loading…</p>;
  }

  return (
    <div>
      <Link href="/emails" className="text-xs text-zinc-500 hover:text-zinc-300">
        ← Emails
      </Link>
      <h1 className="mt-3 font-[family-name:var(--font-display)] text-4xl text-zinc-50">
        {email.subject || "(no subject)"}
      </h1>
      <div className="mt-2 flex flex-wrap gap-3 text-sm text-zinc-400">
        <span>{email.from}</span>
        <span>→ {email.to?.join(", ")}</span>
        <span className="rounded bg-zinc-800 px-2 py-0.5 text-xs text-orange-300">{email.status}</span>
      </div>

      <div className="mt-8 grid gap-6 lg:grid-cols-2">
        <section className="rounded-lg border border-zinc-800 bg-[#141417]/70 p-5">
          <h2 className="text-xs uppercase tracking-wider text-zinc-500 mb-3">Timeline</h2>
          <ol className="space-y-3">
            {events.map((ev) => (
              <li key={ev.id} className="flex gap-3 text-sm">
                <div className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-orange-500" />
                <div>
                  <div className="text-zinc-100 font-mono text-xs">{ev.type}</div>
                  <div className="text-zinc-500 text-xs tabular-nums">
                    {new Date(ev.created_at).toLocaleString()}
                  </div>
                </div>
              </li>
            ))}
            {events.length === 0 && <p className="text-sm text-zinc-500">No events yet</p>}
          </ol>
        </section>

        <section className="rounded-lg border border-zinc-800 bg-[#141417]/70 p-5">
          <h2 className="text-xs uppercase tracking-wider text-zinc-500 mb-3">Preview</h2>
          {email.html ? (
            <iframe
              title="preview"
              className="w-full min-h-64 rounded border border-zinc-800 bg-white"
              srcDoc={email.html}
            />
          ) : (
            <pre className="text-xs text-zinc-400 whitespace-pre-wrap">{email.text}</pre>
          )}
          {atts.length > 0 && (
            <div className="mt-4">
              <h3 className="text-xs text-zinc-500 mb-2">Attachments</h3>
              <ul className="text-sm text-zinc-300 space-y-1">
                {atts.map((a) => (
                  <li key={a.id} className="font-mono text-xs">
                    {a.filename} ({a.size_bytes}b)
                  </li>
                ))}
              </ul>
            </div>
          )}
          {email.provider_message_id && (
            <p className="mt-4 text-xs text-zinc-600 font-mono">msg: {email.provider_message_id}</p>
          )}
        </section>
      </div>
    </div>
  );
}
