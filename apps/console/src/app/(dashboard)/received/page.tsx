"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";

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
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-2">Received</h1>
      <p className="text-sm text-zinc-500 mb-8">Inbound mail via SES receipt → Raisin</p>
      {error && <p className="text-sm text-red-400 mb-4">{error}</p>}
      <div className="overflow-hidden rounded-lg border border-zinc-800">
        <table className="w-full text-sm">
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
            {list.length === 0 && (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-zinc-500">
                  No inbound mail yet
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
