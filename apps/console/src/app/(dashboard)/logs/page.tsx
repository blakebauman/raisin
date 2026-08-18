"use client";

import { useEffect, useRef, useState } from "react";
import { apiFetch } from "@/lib/api";
import { formatAbsolute, formatRelative } from "@/lib/format";
import { EmptyState, Msg, PageHeader, PropertyRow, TableSkeleton } from "@/components/ui";

type APILog = {
  id: string;
  method: string;
  status_code: number;
  duration_ms: number;
  path: string;
  request_id?: string;
  created_at: string;
};

function statusTone(code: number) {
  if (code >= 500) return "text-red-400";
  if (code >= 400) return "text-amber-300";
  return "text-emerald-400";
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
      <p className="py-8 text-[13px] text-[var(--muted)]">
        Select a request to inspect status, timing, and request ID.
      </p>
    );
  }

  if (loading && !selected) {
    return <p className="py-8 text-[13px] text-[var(--muted)]">Loading…</p>;
  }

  if (!selected) return null;

  return (
    <div>
      <div className="mb-3 flex items-center justify-between gap-3">
        <h2 className="break-all font-mono text-[13px] text-zinc-200">
          {selected.method} {selected.path}
        </h2>
        <button type="button" className="btn-ghost h-6 px-2" onClick={onClose}>
          Close
        </button>
      </div>
      <dl>
        <PropertyRow label="Status">
          <span className={`tabular-nums ${statusTone(selected.status_code)}`}>{selected.status_code}</span>
        </PropertyRow>
        <PropertyRow label="Duration">
          <span className="tabular-nums">{selected.duration_ms ?? "—"} ms</span>
        </PropertyRow>
        <PropertyRow label="Request ID">
          <span className="break-all font-mono text-[12px]">{selected.request_id || "—"}</span>
        </PropertyRow>
        <PropertyRow label="Time">
          <span title={formatAbsolute(selected.created_at)}>{formatRelative(selected.created_at)}</span>
        </PropertyRow>
      </dl>
    </div>
  );
}

export default function LogsPage() {
  const [list, setList] = useState<APILog[] | null>(null);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<APILog | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const detailRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    apiFetch<{ data: APILog[] }>("/logs")
      .then((r) => setList(r.data ?? []))
      .catch((e) => {
        setError(e.message);
        setList([]);
      });
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

      {list === null ? (
        <TableSkeleton />
      ) : list.length === 0 && !error ? (
        <EmptyState title="No API logs yet" hint="Authenticated requests will show up here." />
      ) : (
        <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_15.5rem]">
          <div ref={detailRef} className="order-1 min-w-0 lg:order-2 lg:sticky lg:top-6 lg:border-l lg:border-[var(--border)] lg:pl-5">
            <LogDetail selected={selected} loading={loadingDetail} onClose={() => setSelected(null)} />
          </div>

          <div className="order-2 min-w-0 lg:order-1">
            <div className="log-table data-table max-h-[min(70vh,40rem)] overflow-auto">
              <table className="w-full">
                <thead className="sticky top-0 z-10 bg-[var(--background)]">
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
                      className={`cursor-pointer ${selected?.id === l.id ? "bg-white/[0.06]" : ""}`}
                      onClick={() => openDetail(l)}
                      aria-selected={selected?.id === l.id}
                    >
                      <td className="font-mono text-[12px]">{l.method}</td>
                      <td className="max-w-[12rem] truncate font-mono text-[12px] text-[var(--muted)] sm:max-w-[18rem] lg:max-w-none">
                        {l.path}
                      </td>
                      <td className={`tabular-nums ${statusTone(l.status_code)}`}>{l.status_code}</td>
                      <td className="tabular-nums text-[var(--muted)]">{l.duration_ms ?? "—"}</td>
                      <td className="whitespace-nowrap text-[12px] tabular-nums text-[var(--muted)]" title={formatAbsolute(l.created_at)}>
                        {formatRelative(l.created_at)}
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
