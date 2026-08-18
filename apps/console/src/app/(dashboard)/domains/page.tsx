"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import { EmptyState, Field, FormRow, Msg, PageHeader, StatusChip } from "@/components/ui";

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
      <PageHeader title="Domains" description="Verify sending domains and receiving setup." />
      <FormRow onSubmit={create}>
        <Field label="Domain" htmlFor="dom-name" className="min-w-[12rem] flex-1">
          <input
            id="dom-name"
            className="field"
            placeholder="example.com"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </Field>
        <Field label="Region" htmlFor="dom-region" className="w-40">
          <select
            id="dom-region"
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
          Add domain
        </button>
      </FormRow>
      <Msg
        tone={
          msg.toLowerCase().includes("fail") || msg.toLowerCase().includes("error") ? "error" : "muted"
        }
      >
        {msg}
      </Msg>
      <div>
        {domains.length === 0 ? (
          <EmptyState title="No domains yet" hint="Add a domain, then verify DNS before sending in production." />
        ) : (
          domains.map((d) => (
          <div key={d.id} className="border-t border-[var(--border)] py-4">
            <div className="flex items-center justify-between gap-4">
              <div>
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-lg text-zinc-50">{d.name}</span>
                  <StatusChip status={d.status} />
                </div>
                <div className="mt-1 text-xs text-zinc-500">{d.region}</div>
              </div>
              <div className="flex shrink-0 flex-wrap justify-end gap-2">
                <button
                  type="button"
                  disabled={verifying === d.id}
                  onClick={() => verify(d.id)}
                  className="btn-secondary text-xs"
                >
                  {verifying === d.id ? "Checking…" : "Verify DNS"}
                </button>
                {d.status === "verified" && !d.claimed_at && (
                  <>
                    <button
                      type="button"
                      disabled={busy === d.id}
                      onClick={() => claim(d.id)}
                      className="btn-secondary text-xs"
                    >
                      Start claim
                    </button>
                    <button
                      type="button"
                      disabled={busy === d.id}
                      onClick={() => confirmClaim(d.id)}
                      className="btn-secondary text-xs"
                    >
                      Confirm claim
                    </button>
                  </>
                )}
                <button
                  type="button"
                  disabled={busy === d.id}
                  onClick={() => setBIMI(d)}
                  className="btn-secondary text-xs"
                >
                  BIMI
                </button>
                <button
                  type="button"
                  disabled={busy === d.id}
                  onClick={() => remove(d.id)}
                  className="btn-danger text-xs"
                >
                  Delete
                </button>
              </div>
            </div>
            <div className="mt-3 flex flex-wrap gap-4 text-xs text-zinc-400">
              <label className="inline-flex cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  className="rounded-sm border-zinc-600"
                  checked={d.open_tracking ?? true}
                  disabled={busy === d.id}
                  onChange={() => toggleTracking(d, "open_tracking")}
                />
                Open tracking
              </label>
              <label className="inline-flex cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  className="rounded-sm border-zinc-600"
                  checked={d.click_tracking ?? true}
                  disabled={busy === d.id}
                  onChange={() => toggleTracking(d, "click_tracking")}
                />
                Click tracking
              </label>
              <label className="inline-flex cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  className="rounded-sm border-zinc-600"
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
                  <thead className="text-left text-zinc-500">
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
                      <tr key={i} className="border-t border-[var(--border)]">
                        <td className="py-2 pr-3 text-zinc-500">{r.record}</td>
                        <td className="py-2 pr-3">{r.type}</td>
                        <td className="py-2 pr-3">{r.name}</td>
                        <td className="break-all py-2 pr-3">{r.value}</td>
                        <td className="py-2">
                          <StatusChip status={r.status || "pending"} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
          ))
        )}
      </div>
    </div>
  );
}
