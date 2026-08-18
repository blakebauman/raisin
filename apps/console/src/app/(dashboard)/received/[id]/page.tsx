"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { EmptyState, Msg, PageHeader, SectionLabel } from "@/components/ui";

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
    return <p className="text-sm text-red-400">{error}</p>;
  }
  if (!mail) {
    return <p className="text-sm text-zinc-500">Loading…</p>;
  }

  return (
    <div>
      <Link href="/received" className="text-xs text-zinc-500 hover:text-zinc-300">
        ← Received
      </Link>
      <div className="mt-3">
        <PageHeader
          title={mail.subject || "(no subject)"}
          description={`${mail.from} → ${(mail.to || []).join(", ")} · ${new Date(mail.created_at).toLocaleString()}`}
        />
      </div>
      <Msg tone="error">{error}</Msg>

      {atts.length > 0 && (
        <section className="mb-8 max-w-3xl">
          <SectionLabel>Attachments</SectionLabel>
          <ul className="space-y-2">
            {atts.map((a) => (
              <li key={a.id} className="list-row">
                <span className="min-w-0 text-sm text-zinc-300">
                  {a.filename}{" "}
                  <span className="text-xs text-zinc-500">
                    ({a.content_type}, {a.size_bytes} B)
                  </span>
                </span>
                <button
                  type="button"
                  disabled={downloading === a.id}
                  onClick={() => download(a)}
                  className="btn-secondary text-xs"
                >
                  {downloading === a.id ? "…" : "Download"}
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="max-w-3xl rounded-lg border border-zinc-800 bg-[var(--panel)]/70 p-5">
        <SectionLabel>Body</SectionLabel>
        {mail.html ? (
          <iframe
            title="preview"
            className="min-h-80 w-full rounded border border-zinc-800 bg-white"
            srcDoc={mail.html}
          />
        ) : mail.text ? (
          <pre className="whitespace-pre-wrap text-xs text-zinc-400">{mail.text}</pre>
        ) : (
          <EmptyState title="Empty body" />
        )}
      </section>
    </div>
  );
}
