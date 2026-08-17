"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

type Automation = {
  id: string;
  name: string;
  trigger_type: string;
  enabled: boolean;
  steps?: { position: number; type: string; config: unknown }[];
};

export default function AutomationsPage() {
  const [list, setList] = useState<Automation[]>([]);
  const [name, setName] = useState("Welcome sequence");
  const [trigger, setTrigger] = useState("contact.created");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState<string | null>(null);

  async function load() {
    const res = await apiFetch<{ data: Automation[] }>("/automations");
    setList(res.data ?? []);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/automations", {
        method: "POST",
        body: JSON.stringify({
          name,
          trigger_type: trigger,
          steps: [
            { type: "wait", config: { seconds: 60 } },
            {
              type: "send_email",
              config: {
                from: "Acme <hello@acme.test>",
                subject: "Welcome",
                html: "<p>Thanks for joining.</p>",
              },
            },
          ],
        }),
      });
      setMsg("Automation created (disabled until you enable it)");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
    }
  }

  async function toggle(a: Automation) {
    setBusy(a.id);
    try {
      await apiFetch(`/automations/${a.id}`, {
        method: "PATCH",
        body: JSON.stringify({ enabled: !a.enabled }),
      });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
    } finally {
      setBusy(null);
    }
  }

  async function remove(id: string) {
    setBusy(id);
    try {
      await apiFetch(`/automations/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Automations</h1>
      <form onSubmit={create} className="mb-10 grid gap-3 max-w-xl">
        <input
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Name"
        />
        <select
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          value={trigger}
          onChange={(e) => setTrigger(e.target.value)}
        >
          <option value="contact.created">contact.created</option>
          <option value="email.delivered">email.delivered</option>
          <option value="email.opened">email.opened</option>
          <option value="email.clicked">email.clicked</option>
          <option value="email.received">email.received</option>
        </select>
        <p className="text-xs text-zinc-500">Creates wait → send_email steps. Enable when ready.</p>
        <button type="submit" className="w-fit rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          Create
        </button>
        {msg && <p className="text-sm text-zinc-400">{msg}</p>}
      </form>
      <ul className="space-y-2">
        {list.map((a) => (
          <li key={a.id} className="flex items-center justify-between gap-4 rounded-lg border border-zinc-800 px-4 py-3 text-sm">
            <div>
              <div className="text-zinc-100">{a.name}</div>
              <div className="text-xs text-zinc-500 mt-0.5">
                {a.trigger_type} · {a.enabled ? "enabled" : "disabled"} · {(a.steps || []).length} steps
              </div>
            </div>
            <div className="flex gap-2">
              <button
                type="button"
                disabled={busy === a.id}
                onClick={() => toggle(a)}
                className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs hover:bg-zinc-900 disabled:opacity-50"
              >
                {a.enabled ? "Disable" : "Enable"}
              </button>
              <button
                type="button"
                disabled={busy === a.id}
                onClick={() => remove(a.id)}
                className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-red-400 hover:bg-zinc-900 disabled:opacity-50"
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
