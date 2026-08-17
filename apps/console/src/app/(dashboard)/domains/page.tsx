"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

type Domain = {
  id: string;
  name: string;
  status: string;
  region: string;
  open_tracking?: boolean;
  click_tracking?: boolean;
  claimed_at?: string | null;
  receiving_enabled?: boolean;
  bimi_svg_url?: string | null;
  records: { record: string; name: string; type: string; value: string; status: string }[];
};

export default function DomainsPage() {
  const [domains, setDomains] = useState<Domain[]>([]);
  const [name, setName] = useState("");
  const [region, setRegion] = useState("us-east-1");
  const [regions, setRegions] = useState<string[]>([]);
  const [msg, setMsg] = useState("");
  const [verifying, setVerifying] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  async function load() {
    const [d, r] = await Promise.all([
      apiFetch<{ data: Domain[] }>("/domains"),
      apiFetch<{ data: string[] }>("/domains/regions"),
    ]);
    setDomains(d.data ?? []);
    setRegions(r.data ?? []);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/domains", { method: "POST", body: JSON.stringify({ name, region }) });
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

  async function remove(id: string) {
    setBusy(id);
    setMsg("");
    try {
      await apiFetch(`/domains/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "delete failed");
    } finally {
      setBusy(null);
    }
  }

  async function toggleTracking(d: Domain, field: "open_tracking" | "click_tracking") {
    setBusy(d.id);
    setMsg("");
    try {
      await apiFetch(`/domains/${d.id}`, {
        method: "PATCH",
        body: JSON.stringify({ [field]: !(d[field] ?? true) }),
      });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "update failed");
    } finally {
      setBusy(null);
    }
  }

  async function claim(id: string) {
    setBusy(id);
    try {
      const d = await apiFetch<Domain>(`/domains/${id}/claim`, { method: "POST", body: "{}" });
      const txt = d.records?.find((r) => r.record === "Claim");
      setMsg(
        txt
          ? `Add TXT ${txt.name} = ${txt.value}, then click Confirm claim`
          : "Claim started — add the TXT record, then confirm",
      );
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "claim failed");
    } finally {
      setBusy(null);
    }
  }

  async function confirmClaim(id: string) {
    setBusy(id);
    try {
      await apiFetch(`/domains/${id}/claim/confirm`, { method: "POST", body: "{}" });
      setMsg("Domain claimed");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "confirm failed");
    } finally {
      setBusy(null);
    }
  }

  async function setReceiving(d: Domain, enabled: boolean) {
    setBusy(d.id);
    try {
      await apiFetch(`/domains/${d.id}/receiving`, {
        method: "POST",
        body: JSON.stringify({ enabled }),
      });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "receiving update failed");
    } finally {
      setBusy(null);
    }
  }

  async function setBIMI(d: Domain) {
    const svg = window.prompt("BIMI SVG URL", d.bimi_svg_url || "https://example.com/logo.svg");
    if (!svg) return;
    setBusy(d.id);
    try {
      await apiFetch(`/domains/${d.id}/bimi`, {
        method: "POST",
        body: JSON.stringify({ svg_url: svg }),
      });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "BIMI failed");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Domains</h1>
      <form onSubmit={create} className="mb-8 flex flex-wrap gap-2 max-w-2xl">
        <input
          className="flex-1 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="example.com"
          value={name}
          onChange={(e) => setName(e.target.value)}
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
              <div className="flex gap-2 shrink-0 flex-wrap justify-end">
                <button
                  type="button"
                  disabled={verifying === d.id}
                  onClick={() => verify(d.id)}
                  className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs hover:bg-zinc-900 disabled:opacity-50"
                >
                  {verifying === d.id ? "Checking…" : "Verify DNS"}
                </button>
                {d.status === "verified" && !d.claimed_at && (
                  <>
                    <button
                      type="button"
                      disabled={busy === d.id}
                      onClick={() => claim(d.id)}
                      className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs hover:bg-zinc-900 disabled:opacity-50"
                    >
                      Start claim
                    </button>
                    <button
                      type="button"
                      disabled={busy === d.id}
                      onClick={() => confirmClaim(d.id)}
                      className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs hover:bg-zinc-900 disabled:opacity-50"
                    >
                      Confirm claim
                    </button>
                  </>
                )}
                <button
                  type="button"
                  disabled={busy === d.id}
                  onClick={() => setBIMI(d)}
                  className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs hover:bg-zinc-900 disabled:opacity-50"
                >
                  BIMI
                </button>
                <button
                  type="button"
                  disabled={busy === d.id}
                  onClick={() => remove(d.id)}
                  className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-red-400 hover:bg-zinc-900 disabled:opacity-50"
                >
                  Delete
                </button>
              </div>
            </div>
            <div className="mt-3 flex flex-wrap gap-4 text-xs text-zinc-400">
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  className="rounded border-zinc-600"
                  checked={d.open_tracking ?? true}
                  disabled={busy === d.id}
                  onChange={() => toggleTracking(d, "open_tracking")}
                />
                Open tracking
              </label>
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  className="rounded border-zinc-600"
                  checked={d.click_tracking ?? true}
                  disabled={busy === d.id}
                  onChange={() => toggleTracking(d, "click_tracking")}
                />
                Click tracking
              </label>
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  className="rounded border-zinc-600"
                  checked={d.receiving_enabled ?? false}
                  disabled={busy === d.id || d.status !== "verified"}
                  onChange={() => setReceiving(d, !(d.receiving_enabled ?? false))}
                />
                Inbound receiving
              </label>
              {d.claimed_at && <span className="text-emerald-400">Claimed</span>}
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
