"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

type Key = { id: string; name: string; prefix: string; permission: string; created_at: string };

export default function ApiKeysPage() {
  const [keys, setKeys] = useState<Key[]>([]);
  const [name, setName] = useState("Production");
  const [created, setCreated] = useState("");

  async function load() {
    const res = await apiFetch<{ data: Key[] }>("/api-keys");
    setKeys(res.data ?? []);
  }

  useEffect(() => {
    load().catch(console.error);
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    const res = await apiFetch<{ token: string }>("/api-keys", {
      method: "POST",
      body: JSON.stringify({ name, permission: "full_access" }),
    });
    setCreated(res.token);
    await load();
  }

  async function remove(id: string) {
    await apiFetch(`/api-keys/${id}`, { method: "DELETE" });
    await load();
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">API Keys</h1>
      <form onSubmit={create} className="mb-6 flex gap-2 max-w-md">
        <input
          className="flex-1 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button type="submit" className="rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Create
        </button>
      </form>
      {created && (
        <div className="mb-6 rounded-md border border-orange-500/30 bg-orange-500/10 px-4 py-3 text-sm">
          Copy now — shown once: <code className="font-mono text-orange-300">{created}</code>
        </div>
      )}
      <ul className="space-y-2">
        {keys.map((k) => (
          <li key={k.id} className="flex items-center justify-between rounded-lg border border-zinc-800 px-4 py-3">
            <div>
              <div className="text-sm text-zinc-100">{k.name}</div>
              <div className="text-xs font-mono text-zinc-500">{k.prefix}… · {k.permission}</div>
            </div>
            <button onClick={() => remove(k.id)} className="text-xs text-red-400 hover:underline">
              Delete
            </button>
          </li>
        ))}
      </ul>
      <p className="mt-6 text-xs text-zinc-500">
        Demo key: <code className="font-mono">ra_demo_00000000000000000000000000000000</code>
      </p>
    </div>
  );
}
