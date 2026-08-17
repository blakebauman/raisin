"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

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

  useEffect(() => {
    apiFetch("/usage").then(setUsage).catch(console.error);
    apiFetch<Team>("/team").then(setTeam).catch(console.error);
  }, []);

  async function checkout() {
    const res = await apiFetch<{ url: string }>("/billing/checkout", { method: "POST", body: "{}" });
    if (res.url) window.location.href = res.url;
  }

  async function toggleTestMode() {
    if (!team) return;
    setSaving(true);
    try {
      const next = await apiFetch<Team>("/team", {
        method: "PATCH",
        body: JSON.stringify({ test_mode: !team.test_mode }),
      });
      setTeam(next);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Settings</h1>

      <section className="rounded-lg border border-zinc-800 bg-[#141417]/70 p-5 max-w-lg mb-6">
        <h2 className="text-sm font-medium text-zinc-200">Sending mode</h2>
        <p className="mt-2 text-sm text-zinc-400">
          Test mode allows sends without a verified domain. Turn it off for production.
        </p>
        {team && (
          <div className="mt-4 flex items-center justify-between gap-4">
            <div>
              <div className="text-sm text-zinc-100">{team.name}</div>
              <div className="text-xs text-zinc-500 mt-0.5">
                {team.test_mode ? "Test mode on" : "Production mode"}
              </div>
            </div>
            <button
              type="button"
              disabled={saving}
              onClick={toggleTestMode}
              className={`rounded-md px-4 py-2 text-sm font-medium ${
                team.test_mode
                  ? "bg-zinc-700 text-zinc-100"
                  : "bg-orange-500 text-black"
              }`}
            >
              {saving ? "Saving…" : team.test_mode ? "Disable test mode" : "Enable test mode"}
            </button>
          </div>
        )}
      </section>

      <section className="rounded-lg border border-zinc-800 bg-[#141417]/70 p-5 max-w-lg">
        <h2 className="text-sm font-medium text-zinc-200">Billing</h2>
        {usage && (
          <p className="mt-2 text-sm text-zinc-400">
            {usage.emails_sent}/{usage.quota} this period · status {usage.billing_status}
          </p>
        )}
        <button
          onClick={checkout}
          className="mt-4 rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black"
        >
          Upgrade with Stripe
        </button>
        <p className="mt-3 text-xs text-zinc-500">Requires STRIPE_SECRET_KEY on the API.</p>
      </section>
      <button
        className="mt-8 text-sm text-zinc-500 hover:text-zinc-200"
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
