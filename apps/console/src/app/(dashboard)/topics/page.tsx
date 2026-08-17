"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

type Topic = {
  id: string;
  name: string;
  description?: string | null;
  default_subscription: string;
  created_at: string;
};

export default function TopicsPage() {
  const [list, setList] = useState<Topic[]>([]);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [defSub, setDefSub] = useState("opt_in");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState<string | null>(null);

  async function load() {
    const res = await apiFetch<{ data: Topic[] }>("/topics");
    setList(res.data ?? []);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/topics", {
        method: "POST",
        body: JSON.stringify({
          name,
          description: description || undefined,
          default_subscription: defSub,
        }),
      });
      setName("");
      setDescription("");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
    }
  }

  async function remove(id: string) {
    setBusy(id);
    try {
      await apiFetch(`/topics/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "delete failed");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Topics</h1>
      {msg && <p className="mb-4 text-sm text-red-400">{msg}</p>}

      <form onSubmit={create} className="mb-10 grid gap-3 max-w-xl">
        <input
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="Newsletter"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
        <input
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="Description (optional)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <select
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          value={defSub}
          onChange={(e) => setDefSub(e.target.value)}
        >
          <option value="opt_in">Default: opt-in</option>
          <option value="opt_out">Default: opt-out</option>
        </select>
        <button type="submit" className="w-fit rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Create topic
        </button>
      </form>

      <ul className="space-y-2 max-w-2xl">
        {list.length === 0 ? (
          <li className="text-sm text-zinc-500">No topics yet.</li>
        ) : (
          list.map((t) => (
            <li
              key={t.id}
              className="flex items-center justify-between gap-4 rounded-lg border border-zinc-800 px-4 py-3 text-sm"
            >
              <div>
                <div className="text-zinc-100">{t.name}</div>
                <div className="text-xs text-zinc-500 mt-0.5">
                  {t.default_subscription}
                  {t.description ? ` · ${t.description}` : ""}
                </div>
              </div>
              <button
                type="button"
                disabled={busy === t.id}
                onClick={() => remove(t.id)}
                className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-red-400 hover:bg-zinc-900 disabled:opacity-50"
              >
                Delete
              </button>
            </li>
          ))
        )}
      </ul>
    </div>
  );
}
