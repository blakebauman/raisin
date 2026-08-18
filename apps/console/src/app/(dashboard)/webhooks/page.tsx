"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import {
  EmptyState,
  Field,
  FormPanel,
  Msg,
  PageHeader,
  SecretCallout,
  StatusChip,
} from "@/components/ui";

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
  const [endpoint, setEndpoint] = useState("");
  const [selected, setSelected] = useState<string | null>(null);
  const [events, setEvents] = useState<WebhookEvent[]>([]);
  const [attempts, setAttempts] = useState<Attempt[]>([]);
  const [selectedEvent, setSelectedEvent] = useState<string | null>(null);
  const [createdSecret, setCreatedSecret] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [msg, setMsg] = useState("");
  const [open, setOpen] = useState(false);

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
    try {
      const res = await apiFetch<{ signing_secret?: string }>("/webhooks", {
        method: "POST",
        body: JSON.stringify({
          endpoint,
          events: DEFAULT_EVENTS,
        }),
      });
      if (res.signing_secret) setCreatedSecret(res.signing_secret);
      setEndpoint("");
      setOpen(false);
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "create failed");
    }
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
      <PageHeader
        title="Webhooks"
        description="Push delivery events to your endpoints."
        actions={
          <button type="button" className="btn-primary" onClick={() => setOpen((v) => !v)}>
            {open ? "Cancel" : "Add endpoint"}
          </button>
        }
      />

      {open && (
        <FormPanel onSubmit={create} title="New endpoint">
          <Field label="URL" htmlFor="wh-url" hint="Receives signed POSTs for sent, delivered, bounce, open, and click.">
            <input
              id="wh-url"
              className="field"
              type="url"
              required
              value={endpoint}
              onChange={(e) => setEndpoint(e.target.value)}
              placeholder="https://example.com/webhooks"
              autoFocus
            />
          </Field>
          <button type="submit" className="btn-primary w-fit">
            Add endpoint
          </button>
        </FormPanel>
      )}

      <Msg tone="error">{msg}</Msg>

      {createdSecret && (
        <SecretCallout>
          Signing secret — copy now, shown once:{" "}
          <code className="break-all font-mono text-orange-300">{createdSecret}</code>
        </SecretCallout>
      )}

      {list.length === 0 ? (
        <EmptyState title="No webhooks yet" hint="Add an HTTPS endpoint to receive delivery events." />
      ) : (
        <ul className="mb-10 border-t border-[var(--border)]">
          {list.map((w) => (
            <li key={w.id} className="list-row items-start">
              <button type="button" className="min-w-0 flex-1 text-left" onClick={() => loadEvents(w.id)}>
                <div className="truncate text-zinc-100">{w.endpoint}</div>
                <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-zinc-500">
                  <StatusChip status={(w.enabled ?? true) ? "active" : "canceled"} />
                  <span className="truncate">{(w.events || []).join(", ")}</span>
                </div>
              </button>
              <div className="flex shrink-0 gap-2">
                <button
                  type="button"
                  disabled={busy === w.id}
                  onClick={() => toggleEnabled(w)}
                  className="btn-secondary text-xs"
                >
                  {(w.enabled ?? true) ? "Disable" : "Enable"}
                </button>
                <button
                  type="button"
                  disabled={busy === w.id}
                  onClick={() => remove(w.id)}
                  className="btn-danger text-xs"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {selected && (
        <section className="max-w-3xl">
          <h2 className="mb-3 text-sm font-medium text-zinc-300">Recent deliveries</h2>
          {events.length === 0 ? (
            <EmptyState title="No events yet" hint="Events appear after Raisin delivers to this endpoint." />
          ) : (
            <ul className="border-t border-[var(--border)]">
              {events.map((ev) => (
                <li key={ev.id} className="border-b border-[var(--border)] py-2.5 text-[13px]">
                  <button
                    type="button"
                    className="w-full text-left"
                    onClick={() => loadAttempts(selected, ev.id)}
                  >
                    <div className="flex justify-between gap-4">
                      <span className="text-zinc-100">{ev.type}</span>
                      <span className="text-xs tabular-nums text-zinc-500">
                        {new Date(ev.created_at).toLocaleString()}
                      </span>
                    </div>
                  </button>
                  {selectedEvent === ev.id && attempts.length > 0 && (
                    <ul className="mt-3 space-y-1 border-t border-[var(--border)] pt-3 text-[12px] text-[var(--muted)]">
                      {attempts.map((a) => (
                        <li key={a.id} className="flex justify-between gap-2">
                          <span>
                            {a.success ? "ok" : "fail"}
                            {a.status_code != null ? ` · ${a.status_code}` : ""}
                            {a.error ? ` · ${a.error}` : ""}
                          </span>
                          <span className="tabular-nums">{new Date(a.attempted_at).toLocaleString()}</span>
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
