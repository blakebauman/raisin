"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import { EmptyState, Field, FormPanel, Msg, PageHeader, SecretCallout } from "@/components/ui";

type Key = { id: string; name: string; prefix: string; permission: string; created_at: string };

export default function ApiKeysPage() {
  const [keys, setKeys] = useState<Key[]>([]);
  const [name, setName] = useState("");
  const [created, setCreated] = useState("");
  const [open, setOpen] = useState(false);
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    const res = await apiFetch<{ data: Key[] }>("/api-keys");
    setKeys(res.data ?? []);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setMsg("");
    try {
      const res = await apiFetch<{ token: string }>("/api-keys", {
        method: "POST",
        body: JSON.stringify({ name: name || "Production", permission: "full_access" }),
      });
      setCreated(res.token);
      setName("");
      setOpen(false);
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "create failed");
    } finally {
      setBusy(false);
    }
  }

  async function remove(id: string) {
    await apiFetch(`/api-keys/${id}`, { method: "DELETE" });
    await load();
  }

  return (
    <div>
      <PageHeader
        title="API Keys"
        description="Authenticate sends and management calls."
        actions={
          <button type="button" className="btn-primary" onClick={() => setOpen((v) => !v)}>
            {open ? "Cancel" : "Create key"}
          </button>
        }
      />

      {open && (
        <FormPanel onSubmit={create} title="New key">
          <Field label="Name" htmlFor="key-name" hint="Shown in the list; the token itself is only revealed once.">
            <input
              id="key-name"
              className="field"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Production"
              autoFocus
            />
          </Field>
          <button type="submit" disabled={busy} className="btn-primary w-fit">
            {busy ? "Creating…" : "Create key"}
          </button>
        </FormPanel>
      )}

      <Msg tone={msg.toLowerCase().includes("fail") ? "error" : "muted"}>{msg}</Msg>

      {created && (
        <SecretCallout>
          Copy now — shown once:{" "}
          <code className="font-mono text-orange-300 break-all">{created}</code>
        </SecretCallout>
      )}

      {keys.length === 0 ? (
        <EmptyState title="No API keys yet" hint="Create a key to send mail from your app." />
      ) : (
        <ul className="space-y-2">
          {keys.map((k) => (
            <li key={k.id} className="list-row">
              <div className="min-w-0">
                <div className="text-zinc-100">{k.name}</div>
                <div className="font-mono text-xs text-zinc-500">
                  {k.prefix}… · {k.permission}
                </div>
              </div>
              <button type="button" onClick={() => remove(k.id)} className="btn-danger text-xs">
                Delete
              </button>
            </li>
          ))}
        </ul>
      )}

      <p className="mt-6 text-xs text-zinc-500">
        Demo key: <code className="font-mono">ra_demo_00000000000000000000000000000000</code>
      </p>
    </div>
  );
}
