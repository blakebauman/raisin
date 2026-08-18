"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { apiFetch } from "@/lib/api";
import {
  EmptyState,
  Field,
  FormRow,
  Msg,
  PageHeader,
  SectionLabel,
  SecretCallout,
  StatusChip,
} from "@/components/ui";

type Team = {
  id: string;
  name: string;
  slug: string;
  test_mode: boolean;
  monthly_quota: number;
  billing_status: string;
};

type Member = {
  id: string;
  user_id: string;
  email: string;
  name: string | null;
  role: string;
  created_at: string;
};

type Invite = {
  id: string;
  email: string;
  role: string;
  expires_at: string;
  created_at: string;
  invite_url?: string;
  token?: string;
};

type Plans = {
  configured: boolean;
  name: string;
  interval: string;
  has_annual: boolean;
  monthly_quota: number;
  checkout_ready: boolean;
  portal_ready: boolean;
};

function SettingsInner() {
  const params = useSearchParams();
  const billingFlash = params.get("billing");
  const [usage, setUsage] = useState<any>(null);
  const [plans, setPlans] = useState<Plans | null>(null);
  const [team, setTeam] = useState<Team | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const [invites, setInvites] = useState<Invite[]>([]);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<"member" | "admin">("member");
  const [createdInvite, setCreatedInvite] = useState<Invite | null>(null);
  const [interval, setInterval] = useState<"month" | "year">("month");
  const [saving, setSaving] = useState(false);
  const [busy, setBusy] = useState(false);
  const [billingBusy, setBillingBusy] = useState(false);
  const [msg, setMsg] = useState("");
  const [ok, setOk] = useState("");
  const [canManage, setCanManage] = useState(true);

  async function loadTeam() {
    const next = await apiFetch<Team>("/team");
    setTeam(next);
  }

  async function loadMembers() {
    const res = await apiFetch<{ data: Member[] }>("/team/members");
    setMembers(res.data ?? []);
  }

  async function loadInvites() {
    try {
      const res = await apiFetch<{ data: Invite[] }>("/team/invites");
      setInvites(res.data ?? []);
      setCanManage(true);
    } catch {
      setInvites([]);
      setCanManage(false);
    }
  }

  useEffect(() => {
    if (billingFlash === "success") setOk("Checkout complete — Pro activates when Stripe confirms payment.");
    if (billingFlash === "canceled") setMsg("Checkout canceled.");
  }, [billingFlash]);

  useEffect(() => {
    apiFetch("/usage").then(setUsage).catch((e) => setMsg(e.message));
    apiFetch<Plans>("/billing/plans").then(setPlans).catch(() => undefined);
    loadTeam().catch((e) => setMsg(e.message));
    loadMembers().catch((e) => setMsg(e.message));
    loadInvites().catch(() => undefined);
  }, []);

  async function checkout() {
    setBillingBusy(true);
    setMsg("");
    try {
      const res = await apiFetch<{ url: string }>("/billing/checkout", {
        method: "POST",
        body: JSON.stringify({
          interval,
          success_url: `${window.location.origin}/settings?billing=success`,
          cancel_url: `${window.location.origin}/settings?billing=canceled`,
        }),
      });
      if (res.url) window.location.href = res.url;
    } catch (e: unknown) {
      setMsg(e instanceof Error ? e.message : "checkout failed");
    } finally {
      setBillingBusy(false);
    }
  }

  async function openPortal() {
    setBillingBusy(true);
    setMsg("");
    try {
      const res = await apiFetch<{ url: string }>("/billing/portal", {
        method: "POST",
        body: JSON.stringify({ return_url: `${window.location.origin}/settings` }),
      });
      if (res.url) window.location.href = res.url;
    } catch (e: unknown) {
      setMsg(e instanceof Error ? e.message : "portal failed");
    } finally {
      setBillingBusy(false);
    }
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

  async function createInvite(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setMsg("");
    setOk("");
    setCreatedInvite(null);
    try {
      const inv = await apiFetch<Invite>("/team/invites", {
        method: "POST",
        body: JSON.stringify({ email: inviteEmail, role: inviteRole }),
      });
      setCreatedInvite(inv);
      setInviteEmail("");
      setInviteRole("member");
      setOk(`Invite created for ${inv.email}`);
      await loadInvites();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "invite failed");
    } finally {
      setBusy(false);
    }
  }

  async function revokeInvite(id: string) {
    setMsg("");
    try {
      await apiFetch(`/team/invites/${id}`, { method: "DELETE" });
      setOk("Invite revoked");
      if (createdInvite?.id === id) setCreatedInvite(null);
      await loadInvites();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "revoke failed");
    }
  }

  async function removeMember(id: string) {
    setMsg("");
    try {
      await apiFetch(`/team/members/${id}`, { method: "DELETE" });
      setOk("Member removed");
      await loadMembers();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "remove failed");
    }
  }

  async function copyInviteLink() {
    const url =
      createdInvite?.token != null
        ? `${window.location.origin}/invite?token=${createdInvite.token}`
        : createdInvite?.invite_url;
    if (!url) return;
    try {
      await navigator.clipboard.writeText(url);
      setOk("Invite link copied");
    } catch {
      setMsg("Could not copy link");
    }
  }

  const status = usage?.billing_status ?? team?.billing_status ?? "free";
  const isPaid = status === "active" || status === "past_due";

  return (
    <div>
      <PageHeader title="Settings" description="Team profile, members, billing, and workspace options." />
      <Msg tone="error">{msg}</Msg>
      <Msg tone="ok">{ok}</Msg>

      <section className="mb-8 max-w-lg">
        <SectionLabel>Sending mode</SectionLabel>
        <p className="text-[13px] text-[var(--muted)]">
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

      <section className="mb-8 max-w-2xl border-t border-[var(--border)] pt-6">
        <SectionLabel>Members</SectionLabel>
        <p className="mb-3 text-[13px] text-[var(--muted)]">People with console access to this team.</p>
        {members.length === 0 ? (
          <EmptyState title="No members yet" />
        ) : (
          <div className="border-t border-[var(--border)]">
            {members.map((m) => (
              <div
                key={m.id}
                className="flex h-10 items-center justify-between gap-3 border-b border-[var(--border)] text-[13px]"
              >
                <div className="min-w-0 truncate">
                  <span className="text-zinc-100">{m.name || m.email}</span>
                  {m.name ? <span className="ml-2 text-zinc-500">{m.email}</span> : null}
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <StatusChip status={m.role} />
                  {m.role !== "owner" && canManage ? (
                    <button type="button" className="btn-ghost px-2" onClick={() => removeMember(m.id)}>
                      Remove
                    </button>
                  ) : null}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {canManage && (
        <section className="mb-8 max-w-2xl border-t border-[var(--border)] pt-6">
          <SectionLabel>Invites</SectionLabel>
          <p className="mb-3 text-[13px] text-[var(--muted)]">
            Invite teammates by email. Share the link — they sign in with that address to join.
          </p>

          {(createdInvite?.token || createdInvite?.invite_url) && (
            <SecretCallout>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="min-w-0 break-all font-mono text-[12px]">
                  {createdInvite.token
                    ? `${typeof window !== "undefined" ? window.location.origin : ""}/invite?token=${createdInvite.token}`
                    : createdInvite.invite_url}
                </span>
                <button type="button" className="btn-secondary shrink-0" onClick={copyInviteLink}>
                  Copy link
                </button>
              </div>
            </SecretCallout>
          )}

          <FormRow onSubmit={createInvite}>
            <Field label="Email" htmlFor="invite-email" className="min-w-0 flex-1">
              <input
                id="invite-email"
                type="email"
                required
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                placeholder="teammate@company.com"
                className="input"
              />
            </Field>
            <Field label="Role" htmlFor="invite-role">
              <select
                id="invite-role"
                value={inviteRole}
                onChange={(e) => setInviteRole(e.target.value as "member" | "admin")}
                className="input"
              >
                <option value="member">member</option>
                <option value="admin">admin</option>
              </select>
            </Field>
            <button type="submit" disabled={busy} className="btn-primary self-end">
              {busy ? "Sending…" : "Invite"}
            </button>
          </FormRow>

          {invites.length > 0 && (
            <div className="mt-4 border-t border-[var(--border)]">
              {invites.map((inv) => (
                <div
                  key={inv.id}
                  className="flex h-10 items-center justify-between gap-3 border-b border-[var(--border)] text-[13px]"
                >
                  <div className="min-w-0 truncate">
                    <span className="text-zinc-100">{inv.email}</span>
                    <span className="ml-2 text-zinc-500">{inv.role}</span>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <StatusChip status="pending" />
                    <button type="button" className="btn-ghost px-2" onClick={() => revokeInvite(inv.id)}>
                      Revoke
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>
      )}

      <section className="max-w-lg border-t border-[var(--border)] pt-6">
        <SectionLabel>Billing</SectionLabel>
        {usage && (
          <p className="text-[13px] text-[var(--muted)]">
            {usage.emails_sent.toLocaleString()}/{usage.quota.toLocaleString()} this period ·{" "}
            <span className="capitalize">{usage.billing_status}</span>
            {plans?.configured ? ` · ${plans.name}` : null}
          </p>
        )}
        {plans?.configured && !isPaid && (
          <p className="mt-2 text-[13px] text-zinc-400">
            Upgrade to {plans.name} for {plans.monthly_quota.toLocaleString()} emails/month.
          </p>
        )}
        <div className="mt-4 flex flex-wrap items-end gap-3">
          {!isPaid && plans?.has_annual && (
            <Field label="Billing period" htmlFor="billing-interval">
              <select
                id="billing-interval"
                className="input"
                value={interval}
                onChange={(e) => setInterval(e.target.value as "month" | "year")}
              >
                <option value="month">Monthly</option>
                <option value="year">Annual</option>
              </select>
            </Field>
          )}
          {!isPaid && (
            <button
              type="button"
              onClick={checkout}
              disabled={billingBusy || plans?.checkout_ready === false}
              className="btn-primary"
            >
              {billingBusy ? "Redirecting…" : "Upgrade to Pro"}
            </button>
          )}
          {isPaid && (
            <button type="button" onClick={openPortal} disabled={billingBusy} className="btn-secondary">
              {billingBusy ? "Redirecting…" : "Manage billing"}
            </button>
          )}
        </div>
        {plans && !plans.checkout_ready && (
          <p className="mt-3 text-xs text-zinc-500">
            Set <code className="text-zinc-400">STRIPE_SECRET_KEY</code> and{" "}
            <code className="text-zinc-400">STRIPE_PRICE_PRO</code> on the API to enable checkout.
          </p>
        )}
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

export default function SettingsPage() {
  return (
    <Suspense fallback={<div className="text-[13px] text-[var(--muted)]">Loading…</div>}>
      <SettingsInner />
    </Suspense>
  );
}
