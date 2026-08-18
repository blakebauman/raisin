"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";
import { EmptyState, PageHeader } from "@/components/ui";

export default function ReceivedPage() {
  const [list, setList] = useState<any[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<{ data: any[] }>("/emails/received")
      .then((r) => setList(r.data ?? []))
      .catch((e) => setError(e.message));
  }, []);

  return (
    <div>
      <PageHeader title="Received" description="Inbound mail captured for your domains via SES receipt." />
      {error && <p className="text-sm text-red-400 mb-4">{error}</p>}
      {list.length === 0 && !error ? (
        <EmptyState title="No inbound mail" hint="Forward or send to a verified receiving domain." />
      ) : (
      <div className="data-table">
        <table className="w-full">
          <thead className="bg-zinc-900/60 text-left text-xs uppercase tracking-wider text-zinc-500">
            <tr>
              <th className="px-4 py-3">From</th>
              <th className="px-4 py-3">Subject</th>
              <th className="px-4 py-3">To</th>
              <th className="px-4 py-3">Received</th>
            </tr>
          </thead>
          <tbody>
            {list.map((e) => (
              <tr key={e.id} className="border-t border-zinc-800/80 hover:bg-zinc-900/40">
                <td className="px-4 py-3 text-zinc-200">
                  <Link href={`/received/${e.id}`} className="hover:text-orange-400">
                    {e.from}
                  </Link>
                </td>
                <td className="px-4 py-3 text-zinc-100">
                  <Link href={`/received/${e.id}`} className="hover:text-orange-400">
                    {e.subject || "(no subject)"}
                  </Link>
                </td>
                <td className="px-4 py-3 text-zinc-500">{(e.to || []).join(", ")}</td>
                <td className="px-4 py-3 text-zinc-500 tabular-nums">
                  {new Date(e.created_at).toLocaleString()}
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
