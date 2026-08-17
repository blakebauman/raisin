"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

export default function BroadcastsPage() {
  const [list, setList] = useState<any[]>([]);
  const [segments, setSegments] = useState<any[]>([]);
  const [from, setFrom] = useState("Acme <hello@acme.test>");
  const [subject, setSubject] = useState("Product update");
  const [html, setHtml] = useState("<p>Hello from Raisin.</p>");
  const [name, setName] = useState("");
  const [segmentId, setSegmentId] = useState("");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState<string | null>(null);

  async function load() {
    const [b, s] = await Promise.all([
      apiFetch<{ data: any[] }>("/broadcasts"),
      apiFetch<{ data: any[] }>("/segments"),
    ]);
    setList(b.data ?? []);
    setSegments(s.data ?? []);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/broadcasts", {
        method: "POST",
        body: JSON.stringify({
          name: name || undefined,
          from,
          subject,
          html,
          segment_id: segmentId || undefined,
        }),
      });
      setMsg("Draft created");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "create failed");
    }
  }

  async function send(id: string) {
    setBusy(id);
    setMsg("");
    try {
      await apiFetch(`/broadcasts/${id}/send`, { method: "POST", body: "{}" });
      setMsg("Send queued");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "send failed");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Broadcasts</h1>

      <form onSubmit={create} className="mb-10 grid gap-3 max-w-xl">
        <input
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="Name (optional)"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <input
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="From"
          value={from}
          onChange={(e) => setFrom(e.target.value)}
          required
        />
        <input
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="Subject"
          value={subject}
          onChange={(e) => setSubject(e.target.value)}
          required
        />
        <select
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          value={segmentId}
          onChange={(e) => setSegmentId(e.target.value)}
        >
          <option value="">All contacts</option>
          {segments.map((s) => (
            <option key={s.id} value={s.id}>
              Segment: {s.name}
            </option>
          ))}
        </select>
        <textarea
          className="min-h-24 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 font-mono text-xs"
          value={html}
          onChange={(e) => setHtml(e.target.value)}
        />
        <button type="submit" className="w-fit rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Create draft
        </button>
        {msg && <p className="text-sm text-zinc-400">{msg}</p>}
      </form>

      <ul className="space-y-2">
        {list.map((b) => (
          <li
            key={b.id}
            className="flex items-center justify-between gap-4 rounded-lg border border-zinc-800 px-4 py-3 text-sm"
          >
            <div>
              <div className="text-zinc-100">{b.subject}</div>
              <div className="text-xs text-zinc-500 mt-0.5">
                {b.status}
                {b.name ? ` · ${b.name}` : ""}
                {b.segment_id ? " · segment" : " · all contacts"}
              </div>
            </div>
            {b.status === "draft" && (
              <button
                type="button"
                disabled={busy === b.id}
                onClick={() => send(b.id)}
                className="shrink-0 rounded-md border border-zinc-700 px-3 py-1.5 text-xs hover:bg-zinc-900 disabled:opacity-50"
              >
                {busy === b.id ? "Sending…" : "Send"}
              </button>
            )}
          </li>
        ))}
        {list.length === 0 && <p className="text-sm text-zinc-500">No broadcasts yet.</p>}
      </ul>
    </div>
  );
}
