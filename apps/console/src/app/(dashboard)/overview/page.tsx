"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
import Link from "next/link";
import { EmptyState, PageHeader, StatusChip, TableSkeleton } from "@/components/ui";
import { formatAbsolute, formatRelative } from "@/lib/format";

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

type Email = {
  id: string;
  from: string;
  to: string[];
  subject: string;
  status: string;
  created_at: string;
};

function formatCount(n: number | undefined) {
  if (n === undefined) return "—";
  return n.toLocaleString();
}

export default function OverviewPage() {
  const router = useRouter();
  const [usage, setUsage] = useState<Usage | null>(null);
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [emails, setEmails] = useState<Email[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!localStorage.getItem("raisin_team_token")) {
      router.replace("/login");
      return;
    }
    Promise.all([
      apiFetch<Usage>("/usage"),
      apiFetch<Metrics>("/emails/metrics"),
      apiFetch<{ data: Email[] }>("/emails"),
    ])
      .then(([u, m, e]) => {
        setUsage(u);
        setMetrics(m);
        setEmails((e.data ?? []).slice(0, 12));
      })
      .catch((err) => setError(err.message));
  }, [router]);

  const usedPct = usage ? Math.min(100, (usage.emails_sent / Math.max(1, usage.quota)) * 100) : 0;
  const summary = metrics
    ? `${formatCount(metrics.sent)} sent · ${formatCount(metrics.delivered)} delivered · ${formatCount(metrics.opened)} opened · ${formatCount(metrics.clicked)} clicked`
    : "Sending health for this workspace";

  return (
    <div>
      <PageHeader
        title="Overview"
        description={summary}
        actions={
          <div className="flex items-center gap-2">
            <Link href="/domains" className="btn-secondary">
              Add domain
            </Link>
            <Link href="/emails" className="btn-primary">
              View emails
            </Link>
          </div>
        }
      />

      {error && (
        <div className="mb-5 rounded-2xl border border-red-900/50 bg-red-950/40 px-3 py-2.5 text-[13px] text-red-300">
          {error}. Is the API running? (default http://localhost:18080)
        </div>
      )}

      {usage && (
        <div className="mb-6 flex max-w-xl items-center gap-3">
          <div className="h-1 flex-1 overflow-hidden rounded-full bg-white/[0.06]">
            <div className="h-full rounded-full bg-orange-500" style={{ width: `${usedPct}%` }} />
          </div>
          <div className="shrink-0 text-[12px] tabular-nums text-[var(--muted)]">
            {usage.emails_sent.toLocaleString()} / {usage.quota.toLocaleString()}
            <span className="ml-1.5 capitalize text-[var(--muted-2)]">{usage.billing_status}</span>
          </div>
        </div>
      )}

      {emails === null ? (
        <TableSkeleton />
      ) : emails.length === 0 ? (
        <EmptyState
          title="No mail yet"
          hint="Compose a send from Emails, or hit the API with your team key. Recent messages will land here."
        />
      ) : (
        <div className="data-table">
          <table className="w-full">
            <thead>
              <tr>
                <th>Subject</th>
                <th>To</th>
                <th>Status</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              {emails.map((e) => (
                <tr key={e.id}>
                  <td className="text-zinc-100">
                    <Link href={`/emails/${e.id}`} className="hover:text-orange-300">
                      {e.subject || "(no subject)"}
                    </Link>
                  </td>
                  <td className="text-[var(--muted)]">{e.to?.join(", ")}</td>
                  <td>
                    <StatusChip status={e.status} />
                  </td>
                  <td className="tabular-nums text-[var(--muted)]" title={formatAbsolute(e.created_at)}>
                    {formatRelative(e.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
