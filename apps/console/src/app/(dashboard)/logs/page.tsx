"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

type APILog = {
  id: string;
  method: string;
  path: string;
  status_code: number;
  duration_ms: number;
  request_id?: string;
  created_at: string;
};

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
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">API Logs</h1>
      {error && <p className="mb-4 text-sm text-red-400">{error}</p>}
      <div className="overflow-hidden rounded-lg border border-zinc-800">
        <table className="w-full text-sm">
          <thead className="bg-zinc-900/60 text-left text-xs uppercase tracking-wider text-zinc-500">
            <tr>
              <th className="px-4 py-3">Method</th>
              <th className="px-4 py-3">Path</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">ms</th>
              <th className="px-4 py-3">When</th>
            </tr>
          </thead>
          <tbody>
            {list.map((l) => (
              <tr
                key={l.id}
                className="border-t border-zinc-800/80 hover:bg-zinc-900/40 cursor-pointer"
                onClick={() => openDetail(l.id)}
              >
                <td className="px-4 py-2 font-mono text-xs">{l.method}</td>
                <td className="px-4 py-2 font-mono text-xs text-zinc-400">{l.path}</td>
                <td className="px-4 py-2">{l.status_code}</td>
                <td className="px-4 py-2 text-zinc-500">{l.duration_ms}</td>
                <td className="px-4 py-2 text-zinc-500 tabular-nums text-xs">
                  {l.created_at ? new Date(l.created_at).toLocaleString() : "—"}
                </td>
              </tr>
            ))}
            {list.length === 0 && !error && (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-zinc-500">
                  No API logs yet
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {selected && (
        <section className="mt-6 rounded-lg border border-zinc-800 bg-[#141417]/70 p-5 max-w-2xl text-sm">
          <div className="flex justify-between gap-4">
            <h2 className="text-zinc-200">
              {selected.method} {selected.path}
            </h2>
            <button type="button" className="text-xs text-zinc-500 hover:text-zinc-300" onClick={() => setSelected(null)}>
              Close
            </button>
          </div>
          <dl className="mt-4 grid grid-cols-2 gap-3 text-xs">
            <div>
              <dt className="text-zinc-500">Status</dt>
              <dd className="text-zinc-200">{selected.status_code}</dd>
            </div>
            <div>
              <dt className="text-zinc-500">Duration</dt>
              <dd className="text-zinc-200">{selected.duration_ms} ms</dd>
            </div>
            <div>
              <dt className="text-zinc-500">Request ID</dt>
              <dd className="font-mono text-zinc-300 break-all">{selected.request_id || "—"}</dd>
            </div>
            <div>
              <dt className="text-zinc-500">Time</dt>
              <dd className="text-zinc-200">
                {selected.created_at ? new Date(selected.created_at).toLocaleString() : "—"}
              </dd>
            </div>
          </dl>
        </section>
      )}
    </div>
  );
}
