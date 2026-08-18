"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import { EmptyState, Field, FormPanel, Msg, PageHeader, StatusChip } from "@/components/ui";

type Broadcast = {
  id: string;
  subject: string;
  status: string;
  name?: string | null;
  segment_id?: string | null;
  topic_id?: string | null;
  sent_count?: number;
  failed_count?: number;
  scheduled_at?: string | null;
};

type Segment = { id: string; name: string };
type Topic = { id: string; name: string };

export default function BroadcastsPage() {
  const [list, setList] = useState<Broadcast[]>([]);
  const [segments, setSegments] = useState<Segment[]>([]);
  const [topics, setTopics] = useState<Topic[]>([]);
  const [from, setFrom] = useState("Acme <hello@acme.test>");
  const [subject, setSubject] = useState("Product update");
  const [html, setHtml] = useState("<p>Hello from Raisin.</p>");
  const [name, setName] = useState("");
  const [segmentId, setSegmentId] = useState("");
  const [topicId, setTopicId] = useState("");
  const [scheduledAt, setScheduledAt] = useState("");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [open, setOpen] = useState(false);

  async function load() {
    const [b, s, t] = await Promise.all([
      apiFetch<{ data: Broadcast[] }>("/broadcasts"),
      apiFetch<{ data: Segment[] }>("/segments"),
      apiFetch<{ data: Topic[] }>("/topics"),
    ]);
    setList(b.data ?? []);
    setSegments(s.data ?? []);
    setTopics(t.data ?? []);
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
          topic_id: topicId || undefined,
          scheduled_at: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
        }),
      });
      setMsg("Draft created");
      setOpen(false);
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "create failed");
    }
  }

  async function send(id: string, immediate = false) {
    setBusy(id);
    setMsg("");
    try {
      const res = await apiFetch<{ status: string }>(`/broadcasts/${id}/send`, {
        method: "POST",
        body: JSON.stringify(immediate ? { immediate: true } : {}),
      });
      setMsg(res.status === "scheduled" ? "Send scheduled" : "Send queued");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "send failed");
    } finally {
      setBusy(null);
    }
  }

  async function cancel(id: string) {
    setBusy(id);
    setMsg("");
    try {
      await apiFetch(`/broadcasts/${id}/cancel`, { method: "POST", body: "{}" });
      setMsg("Canceled");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "cancel failed");
    } finally {
      setBusy(null);
    }
  }

  async function remove(id: string) {
    setBusy(id);
    setMsg("");
    try {
      await apiFetch(`/broadcasts/${id}`, { method: "DELETE" });
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
        title="Broadcasts"
        description="Schedule and send audience campaigns."
        actions={
          <button type="button" className="btn-primary" onClick={() => setOpen((v) => !v)}>
            {open ? "Cancel" : "New draft"}
          </button>
        }
      />

      <Msg>{msg}</Msg>

      {open && (
        <FormPanel onSubmit={create} title="New broadcast">
          <Field label="Name" htmlFor="bc-name" hint="Optional internal label.">
            <input
              id="bc-name"
              className="field"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
          </Field>
          <Field label="From" htmlFor="bc-from">
            <input
              id="bc-from"
              className="field"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              required
            />
          </Field>
          <Field label="Subject" htmlFor="bc-subject">
            <input
              id="bc-subject"
              className="field"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              required
            />
          </Field>
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Audience" htmlFor="bc-segment">
              <select
                id="bc-segment"
                className="field"
                value={segmentId}
                onChange={(e) => setSegmentId(e.target.value)}
              >
                <option value="">All contacts</option>
                {segments.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Topic filter" htmlFor="bc-topic">
              <select
                id="bc-topic"
                className="field"
                value={topicId}
                onChange={(e) => setTopicId(e.target.value)}
              >
                <option value="">None</option>
                {topics.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
            </Field>
          </div>
          <Field label="Schedule" htmlFor="bc-when" hint="Leave empty to keep as a draft until you send.">
            <input
              id="bc-when"
              type="datetime-local"
              className="field"
              value={scheduledAt}
              onChange={(e) => setScheduledAt(e.target.value)}
            />
          </Field>
          <Field label="HTML" htmlFor="bc-html">
            <textarea
              id="bc-html"
              className="field min-h-24 font-mono text-xs"
              value={html}
              onChange={(e) => setHtml(e.target.value)}
              required
            />
          </Field>
          <button type="submit" className="btn-primary w-fit">
            Create draft
          </button>
        </FormPanel>
      )}

      {list.length === 0 ? (
        <EmptyState title="No broadcasts yet" hint="Create a draft, then send or schedule it." />
      ) : (
        <ul className="border-t border-[var(--border)]">
          {list.map((b) => (
            <li key={b.id} className="list-row">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-zinc-100">{b.subject}</span>
                  <StatusChip status={b.status} />
                </div>
                <div className="mt-0.5 text-xs text-zinc-500">
                  {typeof b.sent_count === "number" || typeof b.failed_count === "number"
                    ? `${b.sent_count ?? 0} sent / ${b.failed_count ?? 0} failed`
                    : ""}
                  {b.scheduled_at ? ` · at ${new Date(b.scheduled_at).toLocaleString()}` : ""}
                  {b.name ? ` · ${b.name}` : ""}
                  {b.topic_id ? " · topic" : ""}
                  {b.segment_id ? " · segment" : b.topic_id ? "" : " · all contacts"}
                </div>
              </div>
              <div className="flex shrink-0 gap-2">
                {(b.status === "draft" || b.status === "scheduled") && (
                  <button
                    type="button"
                    disabled={busy === b.id}
                    onClick={() => send(b.id, b.status === "scheduled")}
                    className="btn-secondary text-xs"
                  >
                    {busy === b.id ? "Sending…" : b.status === "scheduled" ? "Send now" : "Send"}
                  </button>
                )}
                {(b.status === "draft" || b.status === "queued" || b.status === "scheduled" || b.status === "sending") && (
                  <button
                    type="button"
                    disabled={busy === b.id}
                    onClick={() => cancel(b.id)}
                    className="btn-secondary text-xs"
                  >
                    Cancel
                  </button>
                )}
                <button
                  type="button"
                  disabled={busy === b.id}
                  onClick={() => remove(b.id)}
                  className="btn-danger text-xs"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
