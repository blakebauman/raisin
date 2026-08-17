"use client";

import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { apiFetch } from "@/lib/api";

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
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-4">Authorize</h1>
      <p className="text-sm text-zinc-400 mb-6">
        <span className="text-zinc-100">{app.name}</span> wants access to your Raisin team.
      </p>
      <div className="rounded-lg border border-zinc-800 p-4 mb-6">
        <div className="text-xs uppercase tracking-wider text-zinc-500 mb-2">Scopes</div>
        <ul className="space-y-1 text-sm text-zinc-300 font-mono">
          {requestedScopes.map((s) => (
            <li key={s}>{s}</li>
          ))}
        </ul>
        <div className="mt-4 text-xs text-zinc-500 break-all">Redirect: {redirectURI}</div>
      </div>
      {error && <p className="mb-4 text-sm text-red-400">{error}</p>}
      <div className="flex gap-2">
        <button
          type="button"
          disabled={busy || !redirectURI}
          onClick={approve}
          className="rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black disabled:opacity-50"
        >
          {busy ? "Redirecting…" : "Allow"}
        </button>
        <a href="/oauth" className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300">
          Cancel
        </a>
      </div>
    </div>
  );
}
