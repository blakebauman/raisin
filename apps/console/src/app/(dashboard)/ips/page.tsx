"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

type Pool = {
  id: string;
  name: string;
  region: string;
  status: string;
  ips?: { address: string }[];
  warmup?: { day_index: number; daily_cap: number; sent_today: number };
};

type Domain = { id: string; name: string; ip_pool_id?: string | null };

export default function IPsPage() {
  const [list, setList] = useState<Pool[]>([]);
  const [domains, setDomains] = useState<Domain[]>([]);
  const [name, setName] = useState("Primary");
  const [region, setRegion] = useState("us-east-1");
  const [regions, setRegions] = useState<string[]>([]);
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [assignDomain, setAssignDomain] = useState<Record<string, string>>({});

  async function load() {
    const [p, r, d] = await Promise.all([
      apiFetch<{ data: Pool[] }>("/ip-pools"),
      apiFetch<{ data: string[] }>("/domains/regions"),
      apiFetch<{ data: Domain[] }>("/domains"),
    ]);
    setList(p.data ?? []);
    setRegions(r.data ?? []);
    setDomains(d.data ?? []);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/ip-pools", { method: "POST", body: JSON.stringify({ name, region }) });
      setMsg("Pool provisioning / warming");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
    }
  }

  async function pause(id: string) {
    setBusy(id);
    try {
      await apiFetch(`/ip-pools/${id}/pause`, { method: "POST", body: "{}" });
      await load();
    } finally {
      setBusy(null);
    }
  }

  async function resume(id: string) {
    setBusy(id);
    try {
      await apiFetch(`/ip-pools/${id}/resume`, { method: "POST", body: "{}" });
      await load();
    } finally {
      setBusy(null);
    }
  }

  async function remove(id: string) {
    setBusy(id);
    try {
      await apiFetch(`/ip-pools/${id}`, { method: "DELETE" });
      await load();
    } finally {
      setBusy(null);
    }
  }

  async function assign(poolId: string) {
    const domainId = assignDomain[poolId];
    if (!domainId) {
      setMsg("Pick a domain to assign");
      return;
    }
    setBusy(poolId);
    setMsg("");
    try {
      await apiFetch(`/ip-pools/${poolId}/assign-domain`, {
        method: "POST",
        body: JSON.stringify({ domain_id: domainId }),
      });
      setMsg("Domain assigned to pool");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "assign failed");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Dedicated IPs</h1>
      <form onSubmit={create} className="mb-10 flex flex-wrap gap-2 max-w-xl">
        <input
          className="flex-1 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Pool name"
        />
        <select
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          value={region}
          onChange={(e) => setRegion(e.target.value)}
        >
          {(regions.length ? regions : ["us-east-1"]).map((r) => (
            <option key={r} value={r}>
              {r}
            </option>
          ))}
        </select>
        <button type="submit" className="rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Request pool
        </button>
      </form>
      {msg && <p className="mb-4 text-sm text-zinc-400">{msg}</p>}
      <ul className="space-y-3">
        {list.map((p) => (
          <li key={p.id} className="rounded-lg border border-zinc-800 px-4 py-3 text-sm">
            <div className="flex items-start justify-between gap-4">
              <div>
                <div className="text-zinc-100">{p.name}</div>
                <div className="text-xs text-zinc-500 mt-1">
                  {p.region} · {p.status}
                  {p.ips?.[0] ? ` · ${p.ips[0].address}` : ""}
                </div>
                {p.warmup && (
                  <div className="text-xs text-zinc-500 mt-1">
                    Warmup day {p.warmup.day_index + 1}: {p.warmup.sent_today}/{p.warmup.daily_cap} sent
                  </div>
                )}
                <div className="mt-2 flex flex-wrap gap-2 items-center">
                  <select
                    className="rounded border border-zinc-700 bg-zinc-950 px-2 py-1 text-xs"
                    value={assignDomain[p.id] || ""}
                    onChange={(e) => setAssignDomain({ ...assignDomain, [p.id]: e.target.value })}
                  >
                    <option value="">Assign domain…</option>
                    {domains.map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.name}
                        {d.ip_pool_id === p.id ? " (current)" : ""}
                      </option>
                    ))}
                  </select>
                  <button
                    type="button"
                    disabled={busy === p.id}
                    onClick={() => assign(p.id)}
                    className="rounded-md border border-zinc-700 px-2 py-1 text-xs"
                  >
                    Assign
                  </button>
                </div>
              </div>
              <div className="flex gap-2">
                {p.status === "paused" ? (
                  <button
                    type="button"
                    disabled={busy === p.id}
                    onClick={() => resume(p.id)}
                    className="rounded-md border border-zinc-700 px-2 py-1 text-xs"
                  >
                    Resume
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={busy === p.id}
                    onClick={() => pause(p.id)}
                    className="rounded-md border border-zinc-700 px-2 py-1 text-xs"
                  >
                    Pause
                  </button>
                )}
                <button
                  type="button"
                  disabled={busy === p.id}
                  onClick={() => remove(p.id)}
                  className="rounded-md border border-zinc-700 px-2 py-1 text-xs text-red-400"
                >
                  Delete
                </button>
              </div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
