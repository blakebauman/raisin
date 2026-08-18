"use client";

import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { Msg, PageHeader, SectionLabel } from "@/components/ui";

type PublicApp = {
  name: string;
  client_id: string;
  redirect_uris: string[];
  scopes: string[];
};

export default function OAuthAuthorizePage() {
  const params = useSearchParams();
  const clientId = params.get("client_id") || "";
  const redirectURI = params.get("redirect_uri") || "";
  const scopeParam = params.get("scope") || "";
  const state = params.get("state") || "";

  const [app, setApp] = useState<PublicApp | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const requestedScopes = useMemo(
    () => (scopeParam ? scopeParam.split(/[,\s]+/).filter(Boolean) : app?.scopes || []),
    [scopeParam, app],
  );

  useEffect(() => {
    if (!clientId) {
      setError("client_id is required");
      return;
    }
    apiFetch<PublicApp>(`/oauth/apps/public?client_id=${encodeURIComponent(clientId)}`)
      .then((a) => setApp(a))
      .catch((e) => setError(e instanceof Error ? e.message : "unknown OAuth app"));
  }, [clientId]);

  async function approve() {
    setBusy(true);
    setError("");
    try {
      const res = await apiFetch<{ code: string }>("/oauth/authorize", {
        method: "POST",
        body: JSON.stringify({
          client_id: clientId,
          redirect_uri: redirectURI,
          scopes: requestedScopes,
        }),
      });
      const url = new URL(redirectURI);
      url.searchParams.set("code", res.code);
      if (state) url.searchParams.set("state", state);
      window.location.href = url.toString();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "authorize failed");
      setBusy(false);
    }
  }

  if (error && !app) {
    return <p className="text-sm text-red-400">{error}</p>;
  }
  if (!app) {
    return <p className="text-sm text-zinc-500">Loading app…</p>;
  }

  return (
    <div className="max-w-lg">
      <PageHeader
        title="Authorize"
        description={`${app.name} wants access to your Raisin team.`}
      />

      <div className="mb-6 rounded-lg border border-zinc-800 bg-[var(--panel)]/50 p-4">
        <SectionLabel>Scopes</SectionLabel>
        <ul className="space-y-1 font-mono text-sm text-zinc-300">
          {requestedScopes.map((s) => (
            <li key={s}>{s}</li>
          ))}
        </ul>
        <div className="mt-4 break-all text-xs text-zinc-500">Redirect: {redirectURI}</div>
      </div>

      <Msg tone="error">{error}</Msg>

      <div className="flex gap-2">
        <button
          type="button"
          disabled={busy || !redirectURI}
          onClick={approve}
          className="btn-primary"
        >
          {busy ? "Redirecting…" : "Allow access"}
        </button>
        <a href="/oauth" className="btn-secondary">
          Cancel
        </a>
      </div>
    </div>
  );
}
