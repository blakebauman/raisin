"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
import Link from "next/link";
import { PageHeader } from "@/components/ui";

type Usage = {
  emails_sent: number;
  quota: number;
  remaining: number;
  billing_status: string;
};

type Metrics = {
  sent: number;
  delivered: number;
  bounced: number;
  opened: number;
  clicked: number;
};

export default function OverviewPage() {
  const router = useRouter();
  const [usage, setUsage] = useState<Usage | null>(null);
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!localStorage.getItem("raisin_team_token")) {
      router.replace("/login");
      return;
    }
    Promise.all([
      apiFetch<Usage>("/usage"),
      apiFetch<Metrics>("/emails/metrics"),
    ])
      .then(([u, m]) => {
        setUsage(u);
        setMetrics(m);
      })
      .catch((e) => setError(e.message));
  }, [router]);

  return (
    <div>
      <PageHeader
        title="Overview"
        description="Sending health for your team on raisin.run"
      />

      {error && (
        <div className="mb-6 rounded-md border border-red-900/50 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error} — is the API running? (default http://localhost:18080)
        </div>
      )}

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {[
          { label: "Sent", value: metrics?.sent ?? "—" },
          { label: "Delivered", value: metrics?.delivered ?? "—" },
          { label: "Opened", value: metrics?.opened ?? "—" },
          { label: "Clicked", value: metrics?.clicked ?? "—" },
        ].map((s) => (
          <div
            key={s.label}
            className="rounded-lg border border-zinc-800 bg-[var(--panel)]/70 px-5 py-4"
          >
            <div className="text-xs uppercase tracking-wider text-zinc-500">{s.label}</div>
            <div className="mt-2 text-3xl font-medium tabular-nums text-zinc-50">{s.value}</div>
          </div>
        ))}
      </div>

      {usage && (
        <div className="mt-6 rounded-lg border border-zinc-800 bg-[var(--panel)]/70 px-5 py-4">
          <div className="flex items-baseline justify-between">
            <div>
              <div className="text-xs uppercase tracking-wider text-zinc-500">Monthly usage</div>
              <div className="mt-2 text-lg text-zinc-100">
                {usage.emails_sent} / {usage.quota} emails
              </div>
            </div>
            <div className="text-sm capitalize text-zinc-500">{usage.billing_status}</div>
          </div>
          <div className="mt-4 h-1.5 overflow-hidden rounded-full bg-zinc-800">
            <div
              className="h-full rounded-full bg-orange-500"
              style={{ width: `${Math.min(100, (usage.emails_sent / Math.max(1, usage.quota)) * 100)}%` }}
            />
          </div>
        </div>
      )}

      <div className="mt-8 flex gap-3">
        <Link href="/emails" className="btn-primary">
          View emails
        </Link>
        <Link href="/domains" className="btn-secondary">
          Add domain
        </Link>
      </div>
    </div>
  );
}
