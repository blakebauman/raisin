"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

export default function SuppressionsPage() {
  const [list, setList] = useState<any[]>([]);
  const [email, setEmail] = useState("");
  const [reason, setReason] = useState("manual");
  const [batch, setBatch] = useState("");
  const [msg, setMsg] = useState("");

  async function load() {
    const res = await apiFetch<{ data: any[] }>("/suppressions");
    setList(res.data ?? []);
  }
  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function add(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/suppressions", {
        method: "POST",
        body: JSON.stringify({ email, reason }),
      });
      setEmail("");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
    }
  }

  async function addBatch(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    const emails = batch
      .split(/[\n,]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (emails.length === 0) return;
    try {
      await apiFetch("/suppressions/batch", {
        method: "POST",
        body: JSON.stringify({ emails, reason }),
      });
      setBatch("");
      setMsg(`Added ${emails.length}`);
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "batch failed");
    }
  }

  async function remove(id: string) {
    try {
      await apiFetch(`/suppressions/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "remove failed");
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Suppressions</h1>
      {msg && <p className="mb-4 text-sm text-zinc-400">{msg}</p>}

      <form onSubmit={add} className="mb-4 flex flex-wrap gap-2 max-w-xl">
        <input
          className="flex-1 min-w-[12rem] rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="bounce@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <select
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        >
          <option value="manual">manual</option>
          <option value="bounce">bounce</option>
          <option value="complaint">complaint</option>
        </select>
        <button type="submit" className="rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Add
        </button>
      </form>

      <form onSubmit={addBatch} className="mb-8 max-w-xl">
        <textarea
          className="w-full min-h-20 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm font-mono"
          placeholder={"batch@example.com\nother@example.com"}
          value={batch}
          onChange={(e) => setBatch(e.target.value)}
        />
        <button type="submit" className="mt-2 rounded-md border border-zinc-700 px-4 py-2 text-sm hover:bg-zinc-900">
          Add batch
        </button>
      </form>

      <ul className="space-y-2">
        {list.map((s) => (
          <li key={s.id} className="flex items-center justify-between rounded-lg border border-zinc-800 px-4 py-3 text-sm">
            <div>
              <div className="text-zinc-100">{s.email}</div>
              <div className="text-xs text-zinc-500">
                {s.reason}
                {s.created_at ? ` · ${new Date(s.created_at).toLocaleString()}` : ""}
              </div>
            </div>
            <button type="button" onClick={() => remove(s.id)} className="text-xs text-red-400 hover:underline">
              Remove
            </button>
          </li>
        ))}
        {list.length === 0 && <p className="text-sm text-zinc-500">No suppressions</p>}
      </ul>
    </div>
  );
}
