"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import { Msg, PageHeader, SectionLabel, StatusChip } from "@/components/ui";

type Team = {
  id: string;
  name: string;
  slug: string;
  test_mode: boolean;
  monthly_quota: number;
  billing_status: string;
};

export default function SettingsPage() {
  const [usage, setUsage] = useState<any>(null);
  const [team, setTeam] = useState<Team | null>(null);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");

  useEffect(() => {
    apiFetch("/usage").then(setUsage).catch((e) => setMsg(e.message));
    apiFetch<Team>("/team").then(setTeam).catch((e) => setMsg(e.message));
  }, []);

  async function checkout() {
    const res = await apiFetch<{ url: string }>("/billing/checkout", { method: "POST", body: "{}" });
    if (res.url) window.location.href = res.url;
  }

  async function toggleTestMode() {
    if (!team) return;
    setSaving(true);
    setMsg("");
    try {
      const next = await apiFetch<Team>("/team", {
        method: "PATCH",
        body: JSON.stringify({ test_mode: !team.test_mode }),
      });
      setTeam(next);
    } catch (e: unknown) {
      setMsg(e instanceof Error ? e.message : "update failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <PageHeader title="Settings" description="Team profile, billing, and workspace options." />
      <Msg tone="error">{msg}</Msg>

      <section className="mb-6 max-w-lg rounded-lg border border-zinc-800 bg-[var(--panel)]/70 p-5">
        <SectionLabel>Sending mode</SectionLabel>
        <p className="text-sm text-zinc-400">
          Test mode allows sends without a verified domain. Turn it off for production.
        </p>
        {team && (
          <div className="mt-4 flex items-center justify-between gap-4">
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm text-zinc-100">{team.name}</span>
                <StatusChip status={team.test_mode ? "draft" : "active"} />
              </div>
              <div className="mt-0.5 text-xs text-zinc-500">
                {team.test_mode ? "Test mode on" : "Production mode"} · {team.slug}
              </div>
            </div>
            <button
              type="button"
              disabled={saving}
              onClick={toggleTestMode}
              className={team.test_mode ? "btn-secondary" : "btn-primary"}
            >
              {saving ? "Saving…" : team.test_mode ? "Disable test mode" : "Enable test mode"}
            </button>
          </div>
        )}
      </section>

      <section className="max-w-lg rounded-lg border border-zinc-800 bg-[var(--panel)]/70 p-5">
        <SectionLabel>Billing</SectionLabel>
        {usage && (
          <p className="text-sm text-zinc-400">
            {usage.emails_sent}/{usage.quota} this period · {usage.billing_status}
          </p>
        )}
        <button type="button" onClick={checkout} className="btn-primary mt-4">
          Upgrade with Stripe
        </button>
        <p className="mt-3 text-xs text-zinc-500">Requires STRIPE_SECRET_KEY on the API.</p>
      </section>

      <button
        type="button"
        className="btn-ghost mt-8 px-0"
        onClick={() => {
          localStorage.clear();
          window.location.href = "/login";
        }}
      >
        Sign out
      </button>
    </div>
  );
}
