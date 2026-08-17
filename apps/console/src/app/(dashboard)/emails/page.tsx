"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";

type Email = {
  id: string;
  from: string;
  to: string[];
  subject: string;
  status: string;
  created_at: string;
};

export default function EmailsPage() {
  const [emails, setEmails] = useState<Email[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [from, setFrom] = useState("Acme <hello@acme.test>");
  const [to, setTo] = useState("you@example.com");
  const [subject, setSubject] = useState("Hello from Raisin");
  const [html, setHtml] = useState("<p>It works on <strong>raisin.run</strong>.</p>");
  const [msg, setMsg] = useState("");
  const [loadingMore, setLoadingMore] = useState(false);

  const load = useCallback(async (c?: string | null, append = false) => {
    const q = c ? `?cursor=${encodeURIComponent(c)}` : "";
    const res = await apiFetch<{ data: Email[]; next_cursor?: string }>(`/emails${q}`);
    setEmails((prev) => (append ? [...prev, ...(res.data ?? [])] : res.data ?? []));
    setNextCursor(res.next_cursor ?? null);
  }, []);

  useEffect(() => {
    load(null, false).catch((e) => setMsg(e.message));
  }, [load]);

  async function send(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/emails", {
        method: "POST",
        body: JSON.stringify({ from, to: [to], subject, html }),
      });
      setMsg("Queued");
      await load(null, false);
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "send failed");
    }
  }

  async function loadMore() {
    if (!nextCursor) return;
    setLoadingMore(true);
    try {
      await load(nextCursor, true);
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "load failed");
    } finally {
      setLoadingMore(false);
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Emails</h1>

      <form onSubmit={send} className="mb-10 grid gap-3 max-w-xl">
        <input className="field" value={from} onChange={(e) => setFrom(e.target.value)} placeholder="From" />
        <input className="field" value={to} onChange={(e) => setTo(e.target.value)} placeholder="To" />
        <input className="field" value={subject} onChange={(e) => setSubject(e.target.value)} placeholder="Subject" />
        <textarea className="field min-h-24 font-mono text-xs" value={html} onChange={(e) => setHtml(e.target.value)} />
        <button type="submit" className="w-fit rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Send email
        </button>
        {msg && <p className="text-sm text-zinc-400">{msg}</p>}
      </form>

      <div className="overflow-hidden rounded-lg border border-zinc-800">
        <table className="w-full text-sm">
          <thead className="bg-zinc-900/60 text-left text-xs uppercase tracking-wider text-zinc-500">
            <tr>
              <th className="px-4 py-3">Subject</th>
              <th className="px-4 py-3">To</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Created</th>
            </tr>
          </thead>
          <tbody>
            {emails.map((e) => (
              <tr key={e.id} className="border-t border-zinc-800/80 hover:bg-zinc-900/40">
                <td className="px-4 py-3 text-zinc-100">
                  <Link href={`/emails/${e.id}`} className="hover:text-orange-300">
                    {e.subject || "(no subject)"}
                  </Link>
                </td>
                <td className="px-4 py-3 text-zinc-400">{e.to?.join(", ")}</td>
                <td className="px-4 py-3">
                  <span className="rounded bg-zinc-800 px-2 py-0.5 text-xs text-orange-300">{e.status}</span>
                </td>
                <td className="px-4 py-3 text-zinc-500 tabular-nums">
                  {new Date(e.created_at).toLocaleString()}
                </td>
              </tr>
            ))}
            {emails.length === 0 && (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-zinc-500">
                  No emails yet
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {nextCursor && (
        <button
          type="button"
          disabled={loadingMore}
          onClick={loadMore}
          className="mt-4 rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:bg-zinc-900 disabled:opacity-50"
        >
          {loadingMore ? "Loading…" : "Load more"}
        </button>
      )}

      <style jsx>{`
        .field {
          width: 100%;
          border-radius: 0.375rem;
          border: 1px solid #3f3f46;
          background: #09090b;
          padding: 0.625rem 0.75rem;
          font-size: 0.875rem;
          outline: none;
        }
        .field:focus {
          border-color: rgba(249, 115, 22, 0.6);
        }
      `}</style>
    </div>
  );
}
