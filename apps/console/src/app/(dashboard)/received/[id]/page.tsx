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

export default function ReceivedDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const [mail, setMail] = useState<Received | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!id) return;
    apiFetch<Received>(`/emails/received/${id}`)
      .then(setMail)
      .catch((e) => setError(e.message));
  }, [id]);

  if (error) {
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
    </div>
  );
}
