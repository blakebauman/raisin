"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";
import { EmptyState, PageHeader, TableSkeleton } from "@/components/ui";
import { formatAbsolute, formatRelative } from "@/lib/format";

export default function ReceivedPage() {
  const [list, setList] = useState<any[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<{ data: any[] }>("/emails/received")
      .then((r) => setList(r.data ?? []))
      .catch((e) => {
        setError(e.message);
        setList([]);
      });
  }, []);

  return (
    <div>
      <PageHeader title="Received" description="Inbound mail captured for your domains via SES receipt." />
      {error && <p className="mb-4 text-[13px] text-red-400">{error}</p>}
      {list === null ? (
        <TableSkeleton />
      ) : list.length === 0 && !error ? (
        <EmptyState title="No inbound mail" hint="Forward or send to a verified receiving domain." />
      ) : (
      <div className="data-table">
        <table className="w-full">
          <thead>
            <tr>
              <th>From</th>
              <th>Subject</th>
              <th>To</th>
              <th>Received</th>
            </tr>
          </thead>
          <tbody>
            {list.map((e) => (
              <tr key={e.id}>
                <td className="text-zinc-200">
                  <Link href={`/received/${e.id}`} className="hover:text-orange-300">
                    {e.from}
                  </Link>
                </td>
                <td>
                  <Link href={`/received/${e.id}`} className="hover:text-orange-300">
                    {e.subject || "(no subject)"}
                  </Link>
                </td>
                <td className="text-[var(--muted)]">{(e.to || []).join(", ")}</td>
                <td className="tabular-nums text-[var(--muted)]" title={formatAbsolute(e.created_at)}>
                  {formatRelative(e.created_at)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      )}
    </div>
  );
}
