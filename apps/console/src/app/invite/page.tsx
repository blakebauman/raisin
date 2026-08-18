"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { Msg, WorkspaceMark } from "@/components/ui";

type Preview = {
  team_name: string;
  email: string;
  role: string;
  expires_at: string;
  status: string;
};

function InviteInner() {
  const router = useRouter();
  const params = useSearchParams();
  const token = params.get("token")?.trim() ?? "";
  const [preview, setPreview] = useState<Preview | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [signedIn, setSignedIn] = useState(false);

  useEffect(() => {
    setSignedIn(!!localStorage.getItem("raisin_team_token"));
  }, []);

  useEffect(() => {
    if (!token) {
      setError("Missing invite token");
      setLoading(false);
      return;
    }
    fetch(`/api/proxy/team/invites/preview?token=${encodeURIComponent(token)}`, {
      headers: { "User-Agent": "raisin-console/0.1.0" },
    })
      .then(async (res) => {
        const data = await res.json();
        if (!res.ok) throw new Error(data.message ?? "invite not found");
        setPreview(data as Preview);
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : "failed to load invite"))
      .finally(() => setLoading(false));
  }, [token]);

  async function accept() {
    if (!token) return;
    setBusy(true);
    setError("");
    try {
      const res = await apiFetch<{ token: string; email: string }>("/team/invites/accept", {
        method: "POST",
        body: JSON.stringify({ token }),
      });
      localStorage.setItem("raisin_team_token", res.token);
      if (res.email) localStorage.setItem("raisin_email", res.email);
      router.replace("/overview");
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "accept failed");
    } finally {
      setBusy(false);
    }
  }

  const canAccept = preview?.status === "pending" && signedIn;
  const loginHref = `/login?next=${encodeURIComponent(`/invite?token=${token}`)}`;

  return (
    <div className="flex min-h-screen flex-col items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="mb-8 flex items-center gap-2">
          <WorkspaceMark size={22} />
          <span className="text-[15px] font-medium text-zinc-100">raisin</span>
        </div>
        <h1 className="text-[15px] font-medium text-zinc-50">Team invite</h1>
        <Msg tone="error">{error}</Msg>
        {loading ? (
          <p className="mt-3 text-[13px] text-[var(--muted)]">Loading invite…</p>
        ) : preview ? (
          <div className="mt-4 space-y-3 text-[13px]">
            <p className="text-zinc-300">
              You&apos;ve been invited to <span className="text-zinc-100">{preview.team_name}</span> as{" "}
              <span className="text-zinc-100">{preview.role}</span>.
            </p>
            <p className="text-[var(--muted)]">
              Sign in as <span className="text-zinc-200">{preview.email}</span> to accept.
            </p>
            <p className="text-[12px] text-zinc-500">Status: {preview.status}</p>
            {canAccept ? (
              <button type="button" className="btn-primary mt-2" disabled={busy} onClick={accept}>
                {busy ? "Joining…" : "Accept invite"}
              </button>
            ) : preview.status === "pending" ? (
              <Link href={loginHref} className="btn-primary mt-2 inline-flex">
                Sign in to accept
              </Link>
            ) : null}
          </div>
        ) : null}
      </div>
    </div>
  );
}

export default function InvitePage() {
  return (
    <Suspense fallback={<div className="p-8 text-[13px] text-[var(--muted)]">Loading…</div>}>
      <InviteInner />
    </Suspense>
  );
}
