"use client";

import { useEffect, useRef, useState } from "react";
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

function LogDetail({
  selected,
  loading,
  onClose,
}: {
  selected: APILog | null;
  loading: boolean;
  onClose: () => void;
}) {
  if (!selected && !loading) {
    return (
      <div className="rounded-lg border border-dashed border-zinc-800 px-4 py-8 text-center">
        <p className="text-sm text-zinc-400">Select a request to inspect status, timing, and request ID.</p>
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-zinc-800 bg-[var(--panel)]/80 p-5 text-sm shadow-lg shadow-black/20">
      <div className="flex items-start justify-between gap-3">
        <SectionLabel>Request</SectionLabel>
        {selected && (
          <button type="button" className="btn-ghost -mt-1 px-2 py-1 text-xs" onClick={onClose}>
            Close
          </button>
        )}
      </div>
      {loading && !selected ? (
        <p className="text-sm text-zinc-500">Loading…</p>
      ) : selected ? (
        <>
          <h2 className="break-all font-mono text-sm text-zinc-200">
            {selected.method} {selected.path}
          </h2>
          <dl className="mt-4 grid grid-cols-2 gap-3 text-xs lg:grid-cols-1">
            <div>
              <dt className="text-zinc-500">Status</dt>
              <dd className={`mt-0.5 tabular-nums ${statusTone(selected.status_code)}`}>
                {selected.status_code}
              </dd>
            </div>
            <div>
              <dt className="text-zinc-500">Duration</dt>
              <dd className="mt-0.5 tabular-nums text-zinc-200">{selected.duration_ms ?? "—"} ms</dd>
            </div>
            <div className="col-span-2 lg:col-span-1">
              <dt className="text-zinc-500">Request ID</dt>
              <dd className="mt-0.5 break-all font-mono text-zinc-300">{selected.request_id || "—"}</dd>
            </div>
            <div className="col-span-2 lg:col-span-1">
              <dt className="text-zinc-500">Time</dt>
              <dd className="mt-0.5 tabular-nums text-zinc-200">
                {selected.created_at ? new Date(selected.created_at).toLocaleString() : "—"}
              </dd>
            </div>
          </dl>
        </>
      ) : null}
    </div>
  );
}

export default function LogsPage() {
  const [list, setList] = useState<APILog[]>([]);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<APILog | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const detailRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    apiFetch<{ data: APILog[] }>("/logs")
      .then((r) => setList(r.data ?? []))
      .catch((e) => setError(e.message));
  }, []);

  async function openDetail(row: APILog) {
    setError("");
    setSelected(row);
    setLoadingDetail(true);
    requestAnimationFrame(() => {
      detailRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
    });
    try {
      const log = await apiFetch<APILog>(`/logs/${row.id}`);
      setSelected(log);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "failed to load log");
    } finally {
      setLoadingDetail(false);
    }
  }

  return (
    <div>
      <PageHeader title="API Logs" description="Recent authenticated API requests for this team." />
      <Msg tone="error">{error}</Msg>

      {list.length === 0 && !error ? (
        <EmptyState title="No API logs yet" hint="Authenticated requests will show up here." />
      ) : (
        <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(260px,20rem)] lg:gap-6">
          {/* Detail first in DOM so stacked (narrow) layouts show it above the table, not under it. */}
          <div ref={detailRef} className="order-1 lg:order-2 lg:sticky lg:top-8">
            <LogDetail selected={selected} loading={loadingDetail} onClose={() => setSelected(null)} />
          </div>

          <div className="order-2 min-w-0 lg:order-1">
            <div className="log-table data-table max-h-[min(70vh,40rem)] overflow-auto">
              <table className="w-full">
                <thead className="sticky top-0 z-10 bg-[var(--panel)]">
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
                      className={`cursor-pointer ${selected?.id === l.id ? "bg-zinc-800/80" : ""}`}
                      onClick={() => openDetail(l)}
                      aria-selected={selected?.id === l.id}
                    >
                      <td className="font-mono text-xs">{l.method}</td>
                      <td className="max-w-[12rem] truncate font-mono text-xs text-zinc-400 sm:max-w-[18rem] lg:max-w-none">
                        {l.path}
                      </td>
                      <td className={`tabular-nums ${statusTone(l.status_code)}`}>{l.status_code}</td>
                      <td className="tabular-nums text-zinc-500">{l.duration_ms ?? "—"}</td>
                      <td className="whitespace-nowrap text-xs tabular-nums text-zinc-500">
                        {l.created_at ? new Date(l.created_at).toLocaleString() : "—"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
