"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

type Domain = {
  id: string;
  name: string;
  status: string;
  region: string;
  records: { record: string; name: string; type: string; value: string; status: string }[];
};

export default function DomainsPage() {
  const [domains, setDomains] = useState<Domain[]>([]);
  const [name, setName] = useState("");
  const [msg, setMsg] = useState("");
  const [verifying, setVerifying] = useState<string | null>(null);

  async function load() {
    const res = await apiFetch<{ data: Domain[] }>("/domains");
    setDomains(res.data ?? []);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/domains", { method: "POST", body: JSON.stringify({ name }) });
      setName("");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
    }
  }

  async function verify(id: string) {
    setMsg("");
    setVerifying(id);
    try {
      const d = await apiFetch<Domain>(`/domains/${id}/verify`, { method: "POST" });
      setMsg(d.status === "verified" ? `Verified ${d.name}` : `Still pending — check DNS for ${d.name}`);
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "verify failed");
    } finally {
      setVerifying(null);
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Domains</h1>
      <form onSubmit={create} className="mb-8 flex gap-2 max-w-lg">
        <input
          className="flex-1 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="example.com"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button type="submit" className="rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Add
        </button>
      </form>
      {msg && (
        <p className={`mb-4 text-sm ${msg.toLowerCase().includes("fail") || msg.toLowerCase().includes("error") ? "text-red-400" : "text-zinc-400"}`}>
          {msg}
        </p>
      )}
      <div className="space-y-4">
        {domains.map((d) => (
          <div key={d.id} className="rounded-lg border border-zinc-800 bg-[#141417]/70 p-5">
            <div className="flex items-center justify-between gap-4">
              <div>
                <div className="text-lg text-zinc-50">{d.name}</div>
                <div className="text-xs text-zinc-500 mt-1">
                  {d.region} ·{" "}
                  <span className={d.status === "verified" ? "text-emerald-400" : "text-amber-400"}>
                    {d.status}
                  </span>
                </div>
              </div>
              <button
                type="button"
                disabled={verifying === d.id}
                onClick={() => verify(d.id)}
                className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs hover:bg-zinc-900 disabled:opacity-50"
              >
                {verifying === d.id ? "Checking…" : "Verify DNS"}
              </button>
            </div>
            {d.records?.length > 0 && (
              <div className="mt-4 overflow-x-auto">
                <table className="w-full text-xs">
                  <thead className="text-zinc-500 text-left">
                    <tr>
                      <th className="py-1 pr-3">Purpose</th>
                      <th className="py-1 pr-3">Type</th>
                      <th className="py-1 pr-3">Name</th>
                      <th className="py-1 pr-3">Value</th>
                      <th className="py-1">Status</th>
                    </tr>
                  </thead>
                  <tbody className="font-mono text-zinc-300">
                    {d.records.map((r, i) => (
                      <tr key={i} className="border-t border-zinc-800/60">
                        <td className="py-2 pr-3 text-zinc-500">{r.record}</td>
                        <td className="py-2 pr-3">{r.type}</td>
                        <td className="py-2 pr-3">{r.name}</td>
                        <td className="py-2 pr-3 break-all">{r.value}</td>
                        <td className="py-2">
                          <span
                            className={
                              r.status === "verified"
                                ? "text-emerald-400"
                                : r.status === "pending"
                                  ? "text-amber-400"
                                  : "text-zinc-500"
                            }
                          >
                            {r.status || "—"}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
