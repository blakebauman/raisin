"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

type Webhook = {
  id: string;
  endpoint: string;
  events: string[];
  enabled?: boolean;
};

type WebhookEvent = {
  id: string;
  type: string;
  payload: unknown;
  created_at: string;
};

type Attempt = {
  id: string;
  status_code: number | null;
  success: boolean;
  error?: string | null;
  attempted_at: string;
};

const DEFAULT_EVENTS = ["email.sent", "email.delivered", "email.bounced", "email.opened", "email.clicked"];

export default function WebhooksPage() {
  const [list, setList] = useState<Webhook[]>([]);
  const [endpoint, setEndpoint] = useState("https://example.com/webhooks");
  const [selected, setSelected] = useState<string | null>(null);
  const [events, setEvents] = useState<WebhookEvent[]>([]);
  const [attempts, setAttempts] = useState<Attempt[]>([]);
  const [selectedEvent, setSelectedEvent] = useState<string | null>(null);
  const [createdSecret, setCreatedSecret] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [msg, setMsg] = useState("");

  async function load() {
    const res = await apiFetch<{ data: Webhook[] }>("/webhooks");
    setList(res.data ?? []);
  }
  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function loadEvents(id: string) {
    setSelected(id);
    setSelectedEvent(null);
    setAttempts([]);
    const res = await apiFetch<{ data: WebhookEvent[] }>(`/webhooks/${id}/events`);
    setEvents(res.data ?? []);
  }

  async function loadAttempts(webhookId: string, eventId: string) {
    setSelectedEvent(eventId);
    const res = await apiFetch<{ data: Attempt[] }>(
      `/webhooks/${webhookId}/events/${eventId}/attempts`,
    );
    setAttempts(res.data ?? []);
  }

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    const res = await apiFetch<{ signing_secret?: string }>("/webhooks", {
      method: "POST",
      body: JSON.stringify({
        endpoint,
        events: DEFAULT_EVENTS,
      }),
    });
    if (res.signing_secret) setCreatedSecret(res.signing_secret);
    await load();
  }

  async function toggleEnabled(w: Webhook) {
    setBusy(w.id);
    setMsg("");
    try {
      await apiFetch(`/webhooks/${w.id}`, {
        method: "PATCH",
        body: JSON.stringify({ enabled: !(w.enabled ?? true) }),
      });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "update failed");
    } finally {
      setBusy(null);
    }
  }

  async function remove(id: string) {
    setBusy(id);
    setMsg("");
    try {
      await apiFetch(`/webhooks/${id}`, { method: "DELETE" });
      if (selected === id) {
        setSelected(null);
        setEvents([]);
        setAttempts([]);
      }
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "delete failed");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Webhooks</h1>
      <form onSubmit={create} className="mb-6 flex gap-2 max-w-xl">
        <input
          className="flex-1 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          value={endpoint}
          onChange={(e) => setEndpoint(e.target.value)}
        />
        <button type="submit" className="rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Add
        </button>
      </form>
      {msg && <p className="mb-4 text-sm text-red-400">{msg}</p>}
      {createdSecret && (
        <div className="mb-6 rounded-md border border-orange-500/30 bg-orange-500/10 px-4 py-3 text-sm max-w-xl">
          Signing secret — copy now, shown once:{" "}
          <code className="font-mono text-orange-300 break-all">{createdSecret}</code>
        </div>
      )}
      <ul className="space-y-2 mb-10">
        {list.map((w) => (
          <li key={w.id} className="rounded-lg border border-zinc-800 px-4 py-3 text-sm">
            <div className="flex items-start justify-between gap-3">
              <button type="button" className="text-left flex-1 min-w-0" onClick={() => loadEvents(w.id)}>
                <div className="text-zinc-100 truncate">{w.endpoint}</div>
                <div className="text-xs text-zinc-500 mt-1">
                  {(w.enabled ?? true) ? "enabled" : "disabled"} · {(w.events || []).join(", ")}
                </div>
              </button>
              <div className="flex gap-2 shrink-0">
                <button
                  type="button"
                  disabled={busy === w.id}
                  onClick={() => toggleEnabled(w)}
                  className="rounded-md border border-zinc-700 px-2 py-1 text-xs hover:bg-zinc-900 disabled:opacity-50"
                >
                  {(w.enabled ?? true) ? "Disable" : "Enable"}
                </button>
                <button
                  type="button"
                  disabled={busy === w.id}
                  onClick={() => remove(w.id)}
                  className="rounded-md border border-zinc-700 px-2 py-1 text-xs text-red-400 hover:bg-zinc-900 disabled:opacity-50"
                >
                  Delete
                </button>
              </div>
            </div>
          </li>
        ))}
      </ul>

      {selected && (
        <section className="max-w-3xl">
          <h2 className="text-lg text-zinc-200 mb-3">Recent deliveries</h2>
          {events.length === 0 ? (
            <p className="text-sm text-zinc-500">No events yet.</p>
          ) : (
            <ul className="space-y-2">
              {events.map((ev) => (
                <li key={ev.id} className="rounded-lg border border-zinc-800 px-4 py-3 text-sm">
                  <button
                    type="button"
                    className="text-left w-full"
                    onClick={() => loadAttempts(selected, ev.id)}
                  >
                    <div className="flex justify-between gap-4">
                      <span className="text-zinc-100">{ev.type}</span>
                      <span className="text-xs text-zinc-500">
                        {new Date(ev.created_at).toLocaleString()}
                      </span>
                    </div>
                  </button>
                  {selectedEvent === ev.id && attempts.length > 0 && (
                    <ul className="mt-3 space-y-1 border-t border-zinc-800 pt-3 text-xs text-zinc-400">
                      {attempts.map((a) => (
                        <li key={a.id} className="flex justify-between gap-2">
                          <span>
                            {a.success ? "ok" : "fail"}
                            {a.status_code != null ? ` · ${a.status_code}` : ""}
                            {a.error ? ` · ${a.error}` : ""}
                          </span>
                          <span>{new Date(a.attempted_at).toLocaleString()}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              ))}
            </ul>
          )}
        </section>
      )}
    </div>
  );
}
