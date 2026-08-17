"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
import Link from "next/link";

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
      <header className="mb-10">
        <h1 className="font-[family-name:var(--font-display)] text-4xl text-zinc-50">
          Overview
        </h1>
        <p className="mt-2 text-sm text-zinc-500">
          Sending health for your team on raisin.run
        </p>
      </header>

      {error && (
        <div className="mb-6 rounded-md border border-red-900/50 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error} — is the API running? (default http://localhost:18080)
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[
          { label: "Sent", value: metrics?.sent ?? "—" },
          { label: "Delivered", value: metrics?.delivered ?? "—" },
          { label: "Opened", value: metrics?.opened ?? "—" },
          { label: "Clicked", value: metrics?.clicked ?? "—" },
        ].map((s) => (
          <div
            key={s.label}
            className="rounded-lg border border-zinc-800 bg-[#141417]/70 px-5 py-4"
          >
            <div className="text-xs uppercase tracking-wider text-zinc-500">{s.label}</div>
            <div className="mt-2 text-3xl font-medium tabular-nums text-zinc-50">{s.value}</div>
          </div>
        ))}
      </div>

      {usage && (
        <div className="mt-8 rounded-lg border border-zinc-800 bg-[#141417]/70 px-5 py-4">
          <div className="flex items-baseline justify-between">
            <div>
              <div className="text-xs uppercase tracking-wider text-zinc-500">Monthly usage</div>
              <div className="mt-2 text-lg text-zinc-100">
                {usage.emails_sent} / {usage.quota} emails
              </div>
            </div>
            <div className="text-sm text-zinc-500 capitalize">{usage.billing_status}</div>
          </div>
          <div className="mt-4 h-1.5 overflow-hidden rounded-full bg-zinc-800">
            <div
              className="h-full rounded-full bg-orange-500"
              style={{ width: `${Math.min(100, (usage.emails_sent / Math.max(1, usage.quota)) * 100)}%` }}
            />
          </div>
        </div>
      )}

      <div className="mt-10 flex gap-3">
        <Link
          href="/emails"
          className="rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black hover:bg-orange-400"
        >
          View emails
        </Link>
        <Link
          href="/domains"
          className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-200 hover:bg-zinc-900"
        >
          Add domain
        </Link>
      </div>
    </div>
  );
}
