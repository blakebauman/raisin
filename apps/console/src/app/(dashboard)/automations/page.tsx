"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

type StepDraft = {
  type: "wait" | "send_email";
  seconds: string;
  from: string;
  subject: string;
  html: string;
};

type Automation = {
  id: string;
  name: string;
  trigger_type: string;
  enabled: boolean;
  steps?: { position: number; type: string; config: unknown }[];
};

type Run = {
  id: string;
  automation_id: string;
  status: string;
  current_step: number;
  created_at: string;
};

const emptyWait = (): StepDraft => ({
  type: "wait",
  seconds: "60",
  from: "",
  subject: "",
  html: "",
});

const emptySend = (): StepDraft => ({
  type: "send_email",
  seconds: "",
  from: "Acme <hello@acme.test>",
  subject: "Welcome",
  html: "<p>Thanks for joining.</p>",
});

export default function AutomationsPage() {
  const [list, setList] = useState<Automation[]>([]);
  const [name, setName] = useState("Welcome sequence");
  const [trigger, setTrigger] = useState("contact.created");
  const [steps, setSteps] = useState<StepDraft[]>([emptyWait(), emptySend()]);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [runsFor, setRunsFor] = useState<string | null>(null);
  const [runs, setRuns] = useState<Run[]>([]);

  async function load() {
    const res = await apiFetch<{ data: Automation[] }>("/automations");
    setList(res.data ?? []);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  function updateStep(i: number, patch: Partial<StepDraft>) {
    setSteps((prev) => prev.map((s, idx) => (idx === i ? { ...s, ...patch } : s)));
  }

  function buildStepsPayload() {
    return steps.map((s) => {
      if (s.type === "wait") {
        return { type: "wait", config: { seconds: Number(s.seconds) || 0 } };
      }
      return {
        type: "send_email",
        config: { from: s.from, subject: s.subject, html: s.html },
      };
    });
  }

  function loadIntoEditor(a: Automation) {
    setEditingId(a.id);
    setName(a.name);
    setTrigger(a.trigger_type);
    const drafts: StepDraft[] = (a.steps || []).map((st) => {
      const cfg = (st.config || {}) as Record<string, unknown>;
      if (st.type === "wait") {
        return {
          type: "wait" as const,
          seconds: String(cfg.seconds ?? 60),
          from: "",
          subject: "",
          html: "",
        };
      }
      return {
        type: "send_email" as const,
        seconds: "",
        from: String(cfg.from ?? ""),
        subject: String(cfg.subject ?? ""),
        html: String(cfg.html ?? ""),
      };
    });
    setSteps(drafts.length ? drafts : [emptyWait()]);
    setMsg(`Editing ${a.name}`);
  }

  function resetEditor() {
    setEditingId(null);
    setName("Welcome sequence");
    setTrigger("contact.created");
    setSteps([emptyWait(), emptySend()]);
  }

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      if (editingId) {
        await apiFetch(`/automations/${editingId}`, {
          method: "PATCH",
          body: JSON.stringify({
            name,
            trigger_type: trigger,
            steps: buildStepsPayload(),
          }),
        });
        setMsg("Automation updated");
        resetEditor();
      } else {
        await apiFetch("/automations", {
          method: "POST",
          body: JSON.stringify({
            name,
            trigger_type: trigger,
            steps: buildStepsPayload(),
          }),
        });
        setMsg("Automation created (disabled until you enable it)");
      }
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
      if (runsFor === id) {
        setRunsFor(null);
        setRuns([]);
      }
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
    } finally {
      setBusy(null);
    }
  }

  async function loadRuns(id: string) {
    setBusy(id);
    setRunsFor(id);
    try {
      const res = await apiFetch<{ data: Run[] }>(`/automations/${id}/runs`);
      setRuns(res.data ?? []);
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
      setRuns([]);
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

        <div className="space-y-3 border border-zinc-800 rounded-lg p-3">
          <div className="flex items-center justify-between">
            <span className="text-sm text-zinc-300">Steps</span>
            <div className="flex gap-2">
              <button
                type="button"
                className="text-xs rounded-md border border-zinc-700 px-2 py-1 hover:bg-zinc-900"
                onClick={() => setSteps((s) => [...s, emptyWait()])}
              >
                + Wait
              </button>
              <button
                type="button"
                className="text-xs rounded-md border border-zinc-700 px-2 py-1 hover:bg-zinc-900"
                onClick={() => setSteps((s) => [...s, emptySend()])}
              >
                + Send email
              </button>
            </div>
          </div>
          {steps.map((s, i) => (
            <div key={i} className="grid gap-2 rounded-md bg-zinc-950/60 p-3 border border-zinc-800">
              <div className="flex items-center justify-between gap-2">
                <select
                  className="rounded-md border border-zinc-700 bg-zinc-950 px-2 py-1.5 text-xs"
                  value={s.type}
                  onChange={(e) =>
                    updateStep(i, e.target.value === "wait" ? emptyWait() : emptySend())
                  }
                >
                  <option value="wait">wait</option>
                  <option value="send_email">send_email</option>
                </select>
                <button
                  type="button"
                  className="text-xs text-red-400 hover:underline disabled:opacity-40"
                  disabled={steps.length <= 1}
                  onClick={() => setSteps((prev) => prev.filter((_, idx) => idx !== i))}
                >
                  Remove
                </button>
              </div>
              {s.type === "wait" ? (
                <label className="text-xs text-zinc-500 grid gap-1">
                  Seconds
                  <input
                    className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100"
                    value={s.seconds}
                    onChange={(e) => updateStep(i, { seconds: e.target.value })}
                    inputMode="numeric"
                  />
                </label>
              ) : (
                <>
                  <input
                    className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
                    placeholder="From"
                    value={s.from}
                    onChange={(e) => updateStep(i, { from: e.target.value })}
                  />
                  <input
                    className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
                    placeholder="Subject"
                    value={s.subject}
                    onChange={(e) => updateStep(i, { subject: e.target.value })}
                  />
                  <textarea
                    className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm min-h-[72px]"
                    placeholder="HTML body"
                    value={s.html}
                    onChange={(e) => updateStep(i, { html: e.target.value })}
                  />
                </>
              )}
            </div>
          ))}
        </div>

        <button type="submit" className="w-fit rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
          {editingId ? "Save changes" : "Create"}
        </button>
        {editingId && (
          <button
            type="button"
            onClick={resetEditor}
            className="w-fit rounded-md border border-zinc-700 px-4 py-2 text-sm hover:bg-zinc-900"
          >
            Cancel edit
          </button>
        )}
        {msg && <p className="text-sm text-zinc-400">{msg}</p>}
      </form>
      <ul className="space-y-2">
        {list.map((a) => (
          <li key={a.id} className="rounded-lg border border-zinc-800 px-4 py-3 text-sm">
            <div className="flex items-center justify-between gap-4">
              <div>
                <div className="text-zinc-100">{a.name}</div>
                <div className="text-xs text-zinc-500 mt-0.5">
                  {a.trigger_type} · {a.enabled ? "enabled" : "disabled"} · {(a.steps || []).length} steps
                </div>
                {(a.steps || []).length > 0 && (
                  <ol className="mt-2 text-xs text-zinc-500 list-decimal list-inside space-y-0.5">
                    {a.steps!.map((st, idx) => (
                      <li key={idx}>
                        {st.type}
                        {st.type === "wait" && typeof st.config === "object" && st.config && "seconds" in st.config
                          ? ` (${(st.config as { seconds?: number }).seconds}s)`
                          : ""}
                      </li>
                    ))}
                  </ol>
                )}
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  disabled={busy === a.id}
                  onClick={() => loadIntoEditor(a)}
                  className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs hover:bg-zinc-900 disabled:opacity-50"
                >
                  Edit
                </button>
                <button
                  type="button"
                  disabled={busy === a.id}
                  onClick={() => loadRuns(a.id)}
                  className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs hover:bg-zinc-900 disabled:opacity-50"
                >
                  Runs
                </button>
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
            </div>
            {runsFor === a.id && (
              <div className="mt-3 border-t border-zinc-800 pt-3">
                {runs.length === 0 ? (
                  <p className="text-xs text-zinc-500">No runs yet.</p>
                ) : (
                  <ul className="space-y-1">
                    {runs.map((r) => (
                      <li key={r.id} className="flex justify-between text-xs text-zinc-400 font-mono">
                        <span>
                          {r.status} · step {r.current_step}
                        </span>
                        <span>{new Date(r.created_at).toLocaleString()}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
