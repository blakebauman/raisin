"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { apiFetch } from "@/lib/api";

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
        `/emails/received/${id}/attachments/${a.id}`
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
    return <p className="text-red-400 text-sm">{error}</p>;
  }
  if (!mail) {
    return <p className="text-zinc-500 text-sm">Loading…</p>;
  }

  return (
    <div>
      <Link href="/received" className="text-xs text-zinc-500 hover:text-zinc-300">
        ← Received
      </Link>
      <h1 className="mt-3 font-[family-name:var(--font-display)] text-4xl text-zinc-50">
        {mail.subject || "(no subject)"}
      </h1>
      <div className="mt-2 flex flex-wrap gap-3 text-sm text-zinc-400">
        <span>{mail.from}</span>
        <span>→ {(mail.to || []).join(", ")}</span>
        <span className="tabular-nums text-zinc-500">
          {new Date(mail.created_at).toLocaleString()}
        </span>
      </div>

      {atts.length > 0 && (
        <section className="mt-8 max-w-3xl">
          <h2 className="text-xs uppercase tracking-wider text-zinc-500 mb-3">Attachments</h2>
          <ul className="space-y-2">
            {atts.map((a) => (
              <li key={a.id} className="flex items-center justify-between gap-4 text-sm text-zinc-300">
                <span>
                  {a.filename}{" "}
                  <span className="text-xs text-zinc-500">
                    ({a.content_type}, {a.size_bytes} B)
                  </span>
                </span>
                <button
                  type="button"
                  disabled={downloading === a.id}
                  onClick={() => download(a)}
                  className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs hover:bg-zinc-900 disabled:opacity-50"
                >
                  {downloading === a.id ? "…" : "Download"}
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="mt-8 rounded-lg border border-zinc-800 bg-[#141417]/70 p-5 max-w-3xl">
        <h2 className="text-xs uppercase tracking-wider text-zinc-500 mb-3">Body</h2>
        {mail.html ? (
          <iframe
            title="preview"
            className="w-full min-h-80 rounded border border-zinc-800 bg-white"
            srcDoc={mail.html}
          />
        ) : (
          <pre className="text-xs text-zinc-400 whitespace-pre-wrap">{mail.text || "(empty)"}</pre>
        )}
      </section>
      {error && <p className="mt-4 text-sm text-red-400">{error}</p>}
    </div>
  );
}
