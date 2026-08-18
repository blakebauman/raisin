"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { formatAbsolute, formatRelative } from "@/lib/format";
import {
  BackLink,
  EmptyState,
  MailFrame,
  Msg,
  PropertyRow,
  TableSkeleton,
} from "@/components/ui";

type Received = {
  id: string;
  from: string;
  to: string[];
  subject: string;
  html?: string;
  text?: string;
  created_at: string;
};

type Attachment = {
  id: string;
  filename: string;
  content_type: string;
  size_bytes: number;
};

export default function ReceivedDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const [mail, setMail] = useState<Received | null>(null);
  const [atts, setAtts] = useState<Attachment[]>([]);
  const [error, setError] = useState("");
  const [downloading, setDownloading] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    Promise.all([
      apiFetch<Received>(`/emails/received/${id}`),
      apiFetch<{ data: Attachment[] }>(`/emails/received/${id}/attachments`).catch(() => ({ data: [] })),
    ])
      .then(([m, a]) => {
        setMail(m);
        setAtts(a.data ?? []);
      })
      .catch((e) => setError(e.message));
  }, [id]);

  async function download(a: Attachment) {
    setDownloading(a.id);
    try {
      const res = await apiFetch<{ content: string; filename: string; content_type: string }>(
        `/emails/received/${id}/attachments/${a.id}`,
      );
      const bin = atob(res.content);
      const bytes = new Uint8Array(bin.length);
      for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
      const blob = new Blob([bytes], { type: res.content_type || "application/octet-stream" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = res.filename || a.filename;
      link.click();
      URL.revokeObjectURL(url);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "download failed");
    } finally {
      setDownloading(null);
    }
  }

  if (error && !mail) {
    return <p className="text-[13px] text-red-400">{error}</p>;
  }
  if (!mail) {
    return <TableSkeleton rows={10} />;
  }

  return (
    <div>
      <BackLink href="/received">Received</BackLink>
      <h1 className="mt-3 text-[17px] font-medium tracking-tight text-zinc-50 text-balance">
        {mail.subject || "(no subject)"}
      </h1>
      <Msg tone="error">{error}</Msg>

      <div className="mt-5 grid items-start gap-8 lg:grid-cols-[minmax(0,1fr)_15.5rem]">
        <div className="min-w-0">
          {mail.html ? (
            <MailFrame html={mail.html} />
          ) : mail.text ? (
            <pre className="whitespace-pre-wrap text-[13px] text-[var(--muted)]">{mail.text}</pre>
          ) : (
            <EmptyState title="Empty body" />
          )}

          {atts.length > 0 && (
            <div className="mt-8">
              <h2 className="mb-1 text-[13px] font-medium text-zinc-300">Attachments</h2>
              <ul className="border-t border-[var(--border)]">
                {atts.map((a) => (
                  <li key={a.id} className="list-row">
                    <span className="min-w-0 text-[13px] text-zinc-300">
                      {a.filename}{" "}
                      <span className="text-[12px] text-[var(--muted)]">
                        ({a.content_type}, {a.size_bytes} B)
                      </span>
                    </span>
                    <button
                      type="button"
                      disabled={downloading === a.id}
                      onClick={() => download(a)}
                      className="btn-secondary"
                    >
                      {downloading === a.id ? "…" : "Download"}
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>

        <aside>
          <dl>
            <PropertyRow label="From">
              <span className="break-all">{mail.from}</span>
            </PropertyRow>
            <PropertyRow label="To">
              <span className="break-all">{(mail.to || []).join(", ") || "—"}</span>
            </PropertyRow>
            <PropertyRow label="Received">
              <span title={formatAbsolute(mail.created_at)}>{formatRelative(mail.created_at)}</span>
            </PropertyRow>
          </dl>
        </aside>
      </div>
    </div>
  );
}
