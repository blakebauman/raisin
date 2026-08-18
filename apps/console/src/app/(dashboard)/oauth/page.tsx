"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import {
  EmptyState,
  Field,
  FormPanel,
  Msg,
  PageHeader,
  SecretCallout,
} from "@/components/ui";

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
  const [redirect, setRedirect] = useState("http://localhost:3001/oauth/callback");
  const [createdSecret, setCreatedSecret] = useState("");
  const [msg, setMsg] = useState("");
  const [open, setOpen] = useState(false);

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
      setOpen(false);
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
      <PageHeader
        title="OAuth Apps"
        description="Third-party apps authorize against Raisin with scoped access tokens."
        actions={
          <button type="button" className="btn-primary" onClick={() => setOpen((v) => !v)}>
            {open ? "Cancel" : "Create app"}
          </button>
        }
      />

      {open && (
        <FormPanel onSubmit={create} title="New OAuth app">
          <Field label="App name" htmlFor="oauth-name">
            <input
              id="oauth-name"
              className="field"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              autoFocus
            />
          </Field>
          <Field label="Redirect URI" htmlFor="oauth-redirect" hint="Must match the URI used in the authorize request.">
            <input
              id="oauth-redirect"
              className="field"
              value={redirect}
              onChange={(e) => setRedirect(e.target.value)}
              required
            />
          </Field>
          <button type="submit" className="btn-primary w-fit">
            Create app
          </button>
        </FormPanel>
      )}

      {createdSecret && (
        <SecretCallout>
          Client secret (shown once):{" "}
          <code className="break-all font-mono text-orange-300">{createdSecret}</code>
          <div className="mt-2 text-xs text-zinc-400">
            Consent URL:{" "}
            <code className="font-mono">/oauth/authorize?client_id=…&redirect_uri=…</code>
          </div>
        </SecretCallout>
      )}

      <Msg tone="error">{msg}</Msg>

      {list.length === 0 ? (
        <EmptyState title="No OAuth apps" hint="Create an app when a third party needs scoped team access." />
      ) : (
        <ul className="space-y-2">
          {list.map((a) => (
            <li key={a.id} className="list-row items-start">
              <div className="min-w-0">
                <div className="text-zinc-100">{a.name}</div>
                <div className="mt-1 break-all font-mono text-xs text-zinc-500">{a.client_id}</div>
                <div className="mt-1 text-xs text-zinc-600">{(a.redirect_uris || []).join(", ")}</div>
              </div>
              <button type="button" onClick={() => remove(a.id)} className="btn-danger text-xs">
                Delete
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
