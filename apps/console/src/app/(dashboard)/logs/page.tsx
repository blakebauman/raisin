"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import { EmptyState, Msg, PageHeader, SectionLabel } from "@/components/ui";

type APILog = {
  id: string;
  method: string;
  path: string;
  status_code: number;
  duration_ms: number;
  request_id?: string;
  created_at: string;
};

function statusTone(code: number) {
  if (code >= 500) return "text-red-300";
  if (code >= 400) return "text-amber-200";
  return "text-emerald-300";
}

export default function LogsPage() {
  const [list, setList] = useState<APILog[]>([]);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<APILog | null>(null);

  useEffect(() => {
    apiFetch<{ data: APILog[] }>("/logs")
      .then((r) => setList(r.data ?? []))
      .catch((e) => setError(e.message));
  }, []);

  async function openDetail(id: string) {
    setError("");
    try {
      const log = await apiFetch<APILog>(`/logs/${id}`);
      setSelected(log);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "failed to load log");
    }
  }

  return (
    <div>
      <PageHeader title="API Logs" description="Recent authenticated API requests for this team." />
      <Msg tone="error">{error}</Msg>

      {list.length === 0 && !error ? (
        <EmptyState title="No API logs yet" hint="Authenticated requests will show up here." />
      ) : (
        <div className="data-table">
          <table className="w-full">
            <thead>
              <tr>
                <th>Method</th>
                <th>Path</th>
                <th>Status</th>
                <th>ms</th>
                <th>When</th>
              </tr>
            </thead>
            <tbody>
              {list.map((l) => (
                <tr
                  key={l.id}
                  className={`cursor-pointer ${selected?.id === l.id ? "bg-zinc-900/60" : ""}`}
                  onClick={() => openDetail(l.id)}
                >
                  <td className="font-mono text-xs">{l.method}</td>
                  <td className="font-mono text-xs text-zinc-400">{l.path}</td>
                  <td className={`tabular-nums ${statusTone(l.status_code)}`}>{l.status_code}</td>
                  <td className="tabular-nums text-zinc-500">{l.duration_ms}</td>
                  <td className="text-xs tabular-nums text-zinc-500">
                    {l.created_at ? new Date(l.created_at).toLocaleString() : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {selected && (
        <section className="mt-6 max-w-2xl rounded-lg border border-zinc-800 bg-[var(--panel)]/70 p-5 text-sm">
          <div className="flex justify-between gap-4">
            <SectionLabel>Request</SectionLabel>
            <button type="button" className="btn-ghost text-xs" onClick={() => setSelected(null)}>
              Close
            </button>
          </div>
          <h2 className="font-mono text-sm text-zinc-200">
            {selected.method} {selected.path}
          </h2>
          <dl className="mt-4 grid grid-cols-2 gap-3 text-xs">
            <div>
              <dt className="text-zinc-500">Status</dt>
              <dd className={`tabular-nums ${statusTone(selected.status_code)}`}>{selected.status_code}</dd>
            </div>
            <div>
              <dt className="text-zinc-500">Duration</dt>
              <dd className="tabular-nums text-zinc-200">{selected.duration_ms} ms</dd>
            </div>
            <div>
              <dt className="text-zinc-500">Request ID</dt>
              <dd className="break-all font-mono text-zinc-300">{selected.request_id || "—"}</dd>
            </div>
            <div>
              <dt className="text-zinc-500">Time</dt>
              <dd className="tabular-nums text-zinc-200">
                {selected.created_at ? new Date(selected.created_at).toLocaleString() : "—"}
              </dd>
            </div>
          </dl>
        </section>
      )}
    </div>
  );
}
