"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import { EmptyState, Field, FormRow, Msg, PageHeader, StatusChip } from "@/components/ui";

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
      <PageHeader title="Dedicated IPs" description="Warmup and assign sending pools to domains." />
      <FormRow onSubmit={create}>
        <Field label="Pool name" htmlFor="ip-name" className="min-w-[12rem] flex-1">
          <input
            id="ip-name"
            className="field"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </Field>
        <Field label="Region" htmlFor="ip-region" className="w-40">
          <select
            id="ip-region"
            className="field"
            value={region}
            onChange={(e) => setRegion(e.target.value)}
          >
            {(regions.length ? regions : ["us-east-1"]).map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </select>
        </Field>
        <button type="submit" className="btn-primary">
          Request pool
        </button>
      </FormRow>
      <Msg>{msg}</Msg>
      {list.length === 0 ? (
        <EmptyState title="No IP pools" hint="Request a dedicated pool when you need warmed sending IPs." />
      ) : (
      <ul className="border-t border-[var(--border)]">
        {list.map((p) => (
          <li key={p.id} className="border-b border-[var(--border)] py-3 text-[13px]">
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-zinc-100">{p.name}</span>
                  <StatusChip status={p.status} />
                </div>
                <div className="mt-1 text-xs text-zinc-500">
                  {p.region}
                  {p.ips?.[0] ? ` · ${p.ips[0].address}` : ""}
                </div>
                {p.warmup && (
                  <div className="mt-1 text-xs text-zinc-500">
                    Warmup day {p.warmup.day_index + 1}: {p.warmup.sent_today}/{p.warmup.daily_cap} sent
                  </div>
                )}
                <div className="mt-2 flex flex-wrap items-center gap-2">
                  <select
                    className="field w-auto text-xs"
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
                    className="btn-secondary text-xs"
                  >
                    Assign
                  </button>
                </div>
              </div>
              <div className="flex shrink-0 gap-2">
                {p.status === "paused" ? (
                  <button
                    type="button"
                    disabled={busy === p.id}
                    onClick={() => resume(p.id)}
                    className="btn-secondary text-xs"
                  >
                    Resume
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={busy === p.id}
                    onClick={() => pause(p.id)}
                    className="btn-secondary text-xs"
                  >
                    Pause
                  </button>
                )}
                <button
                  type="button"
                  disabled={busy === p.id}
                  onClick={() => remove(p.id)}
                  className="btn-danger text-xs"
                >
                  Delete
                </button>
              </div>
            </div>
          </li>
        ))}
      </ul>
      )}
    </div>
  );
}
