"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

type OAuthApp = {
  id: string;
  name: string;
  client_id: string;
  client_secret?: string;
  redirect_uris: string[];
  scopes: string[];
};

export default function OAuthAppsPage() {
  const [list, setList] = useState<OAuthApp[]>([]);
  const [name, setName] = useState("");
  const [redirect, setRedirect] = useState("http://localhost:3000/oauth/callback");
  const [createdSecret, setCreatedSecret] = useState("");
  const [msg, setMsg] = useState("");

  async function load() {
    const res = await apiFetch<{ data: OAuthApp[] }>("/oauth/apps");
    setList(res.data ?? []);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      const app = await apiFetch<OAuthApp>("/oauth/apps", {
        method: "POST",
        body: JSON.stringify({ name, redirect_uris: [redirect] }),
      });
      if (app.client_secret) setCreatedSecret(app.client_secret);
      setName("");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
    }
  }

  async function remove(id: string) {
    await apiFetch(`/oauth/apps/${id}`, { method: "DELETE" });
    await load();
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">OAuth Apps</h1>
      <p className="text-sm text-zinc-500 mb-6 max-w-xl">
        Third-party apps authorize against Raisin with scoped access tokens (`ra_atk_…`).
      </p>
      <form onSubmit={create} className="mb-8 grid gap-2 max-w-xl">
        <input
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="App name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
        <input
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="Redirect URI"
          value={redirect}
          onChange={(e) => setRedirect(e.target.value)}
          required
        />
        <button type="submit" className="w-fit rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Create app
        </button>
      </form>
      {createdSecret && (
        <div className="mb-6 rounded-md border border-orange-500/30 bg-orange-500/10 px-4 py-3 text-sm max-w-xl">
          Client secret (shown once): <code className="font-mono text-orange-300 break-all">{createdSecret}</code>
        </div>
      )}
      {msg && <p className="mb-4 text-sm text-red-400">{msg}</p>}
      <ul className="space-y-2">
        {list.map((a) => (
          <li key={a.id} className="rounded-lg border border-zinc-800 px-4 py-3 text-sm flex justify-between gap-4">
            <div>
              <div className="text-zinc-100">{a.name}</div>
              <div className="text-xs font-mono text-zinc-500 mt-1 break-all">{a.client_id}</div>
              <div className="text-xs text-zinc-600 mt-1">{(a.redirect_uris || []).join(", ")}</div>
            </div>
            <button
              type="button"
              onClick={() => remove(a.id)}
              className="h-fit rounded-md border border-zinc-700 px-2 py-1 text-xs text-red-400"
            >
              Delete
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
