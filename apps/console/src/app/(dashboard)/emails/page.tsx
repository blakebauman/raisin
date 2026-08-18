"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";
import { EmptyState, Field, FormPanel, Msg, PageHeader, StatusChip, TableSkeleton } from "@/components/ui";
import { formatAbsolute, formatRelative } from "@/lib/format";

type Email = {
  id: string;
  from: string;
  to: string[];
  subject: string;
  status: string;
  created_at: string;
};

export default function EmailsPage() {
  const [emails, setEmails] = useState<Email[] | null>(null);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [from, setFrom] = useState("Acme <hello@acme.test>");
  const [to, setTo] = useState("you@example.com");
  const [subject, setSubject] = useState("Hello from Raisin");
  const [html, setHtml] = useState("<p>It works on <strong>raisin.run</strong>.</p>");
  const [msg, setMsg] = useState("");
  const [msgTone, setMsgTone] = useState<"muted" | "error" | "ok">("muted");
  const [loadingMore, setLoadingMore] = useState(false);
  const [showCompose, setShowCompose] = useState(false);

  const load = useCallback(async (c?: string | null, append = false) => {
    const q = c ? `?cursor=${encodeURIComponent(c)}` : "";
    const res = await apiFetch<{ data: Email[]; next_cursor?: string }>(`/emails${q}`);
    setEmails((prev) => (append ? [...(prev ?? []), ...(res.data ?? [])] : res.data ?? []));
    setNextCursor(res.next_cursor ?? null);
  }, []);

  useEffect(() => {
    load(null, false).catch((e) => {
      setMsg(e.message);
      setMsgTone("error");
    });
  }, [load]);

  async function send(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/emails", {
        method: "POST",
        body: JSON.stringify({ from, to: [to], subject, html }),
      });
      setMsg("Queued for send");
      setMsgTone("ok");
      setShowCompose(false);
      await load(null, false);
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "send failed");
      setMsgTone("error");
    }
  }

  async function loadMore() {
    if (!nextCursor) return;
    setLoadingMore(true);
    try {
      await load(nextCursor, true);
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "load failed");
      setMsgTone("error");
    } finally {
      setLoadingMore(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Emails"
        description="Transactional sends for this team. Open a row for events, attachments, and cancel."
        actions={
          <button type="button" className="btn-primary" onClick={() => setShowCompose((v) => !v)}>
            {showCompose ? "Hide compose" : "Compose"}
          </button>
        }
      />

      {showCompose && (
        <FormPanel onSubmit={send} title="Compose">
          <Field label="From" htmlFor="em-from">
            <input
              id="em-from"
              className="field"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              required
              autoFocus
            />
          </Field>
          <Field label="To" htmlFor="em-to">
            <input
              id="em-to"
              className="field"
              type="email"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              required
            />
          </Field>
          <Field label="Subject" htmlFor="em-subject">
            <input
              id="em-subject"
              className="field"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              required
            />
          </Field>
          <Field label="HTML" htmlFor="em-html">
            <textarea
              id="em-html"
              className="field min-h-24 font-mono text-xs"
              value={html}
              onChange={(e) => setHtml(e.target.value)}
              required
            />
          </Field>
          <button type="submit" className="btn-primary w-fit">
            Send email
          </button>
        </FormPanel>
      )}

      <Msg tone={msgTone}>{msg}</Msg>

      {emails === null ? (
        <TableSkeleton />
      ) : emails.length === 0 ? (
        <EmptyState title="No emails yet" hint="Compose a send, or hit the API with your team key." />
      ) : (
        <div className="data-table">
          <table className="w-full">
            <thead>
              <tr>
                <th>Subject</th>
                <th>To</th>
                <th>Status</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              {emails.map((e) => (
                <tr key={e.id}>
                  <td className="text-zinc-100">
                    <Link href={`/emails/${e.id}`} className="hover:text-orange-300">
                      {e.subject || "(no subject)"}
                    </Link>
                  </td>
                  <td className="text-[var(--muted)]">{e.to?.join(", ")}</td>
                  <td>
                    <StatusChip status={e.status} />
                  </td>
                  <td className="tabular-nums text-[var(--muted)]" title={formatAbsolute(e.created_at)}>
                    {formatRelative(e.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {nextCursor && (
        <button type="button" disabled={loadingMore} onClick={loadMore} className="btn-secondary mt-4">
          {loadingMore ? "Loading…" : "Load more"}
        </button>
      )}
    </div>
  );
}
