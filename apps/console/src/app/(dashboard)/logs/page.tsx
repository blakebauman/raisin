"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

export default function LogsPage() {
  const [list, setList] = useState<any[]>([]);
  useEffect(() => {
    apiFetch<{ data: any[] }>("/logs")
      .then((r) => setList(r.data ?? []))
      .catch(console.error);
  }, []);
  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">API Logs</h1>
      <div className="overflow-hidden rounded-lg border border-zinc-800">
        <table className="w-full text-sm">
          <thead className="bg-zinc-900/60 text-left text-xs uppercase tracking-wider text-zinc-500">
            <tr>
              <th className="px-4 py-3">Method</th>
              <th className="px-4 py-3">Path</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">ms</th>
            </tr>
          </thead>
          <tbody>
            {list.map((l) => (
              <tr key={l.id} className="border-t border-zinc-800/80">
                <td className="px-4 py-2 font-mono text-xs">{l.method}</td>
                <td className="px-4 py-2 font-mono text-xs text-zinc-400">{l.path}</td>
                <td className="px-4 py-2">{l.status_code}</td>
                <td className="px-4 py-2 text-zinc-500">{l.duration_ms}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
