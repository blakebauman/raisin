"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

export default function TemplatesPage() {
  const [list, setList] = useState<any[]>([]);
  const [name, setName] = useState("Welcome");
  const [subject, setSubject] = useState("Welcome {{name}}");
  const [html, setHtml] = useState("<p>Hi {{name}}, welcome.</p>");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState<string | null>(null);

  async function load() {
    const res = await apiFetch<{ data: any[] }>("/templates");
    setList(res.data ?? []);
  }
  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/templates", {
        method: "POST",
        body: JSON.stringify({ name, subject, html }),
      });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "create failed");
    }
  }

  async function publish(id: string) {
    setBusy(id);
    setMsg("");
    try {
      await apiFetch(`/templates/${id}/publish`, { method: "POST", body: "{}" });
      setMsg("Published");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "publish failed");
    } finally {
      setBusy(null);
    }
  }

  async function remove(id: string) {
    setBusy(id);
    try {
      await apiFetch(`/templates/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "delete failed");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Templates</h1>
      <form onSubmit={create} className="mb-8 grid gap-2 max-w-xl">
        <input className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm" value={name} onChange={(e) => setName(e.target.value)} />
        <input className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm" value={subject} onChange={(e) => setSubject(e.target.value)} />
        <textarea className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm font-mono min-h-24" value={html} onChange={(e) => setHtml(e.target.value)} />
        <button type="submit" className="w-fit rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Create
        </button>
        {msg && <p className="text-sm text-zinc-400">{msg}</p>}
      </form>
      <ul className="space-y-2">
        {list.map((t) => (
          <li key={t.id} className="flex items-center justify-between gap-4 rounded-lg border border-zinc-800 px-4 py-3 text-sm">
            <div>
              <div className="text-zinc-100">{t.name}</div>
              <div className="text-xs text-zinc-500">{t.status}</div>
            </div>
            <div className="flex gap-3 shrink-0">
              {t.status !== "published" && (
                <button
                  type="button"
                  disabled={busy === t.id}
                  onClick={() => publish(t.id)}
                  className="text-xs text-orange-400 hover:underline disabled:opacity-50"
                >
                  Publish
                </button>
              )}
              <button
                type="button"
                disabled={busy === t.id}
                onClick={() => remove(t.id)}
                className="text-xs text-red-400 hover:underline disabled:opacity-50"
              >
                Delete
              </button>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
