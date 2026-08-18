"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import { EmptyState, Field, FormPanel, Msg, PageHeader, StatusChip } from "@/components/ui";

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
  const [open, setOpen] = useState(false);

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
    setOpen(true);
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
    setOpen(false);
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
        resetEditor();
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
      <PageHeader
        title="Automations"
        description="Trigger workflows from contact and email events."
        actions={
          <button
            type="button"
            className="btn-primary"
            onClick={() => {
              if (open) resetEditor();
              else setOpen(true);
            }}
          >
            {open ? "Cancel" : "New automation"}
          </button>
        }
      />

      <Msg>{msg}</Msg>

      {open && (
        <FormPanel onSubmit={create} title={editingId ? "Edit automation" : "New automation"}>
          <Field label="Name" htmlFor="auto-name">
            <input
              id="auto-name"
              className="field"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              autoFocus
            />
          </Field>
          <Field label="Trigger" htmlFor="auto-trigger">
            <select
              id="auto-trigger"
              className="field"
              value={trigger}
              onChange={(e) => setTrigger(e.target.value)}
            >
              <option value="contact.created">contact.created</option>
              <option value="email.delivered">email.delivered</option>
              <option value="email.opened">email.opened</option>
              <option value="email.clicked">email.clicked</option>
              <option value="email.received">email.received</option>
            </select>
          </Field>

          <div className="space-y-3 rounded-lg border border-zinc-800 p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium uppercase tracking-wider text-zinc-500">Steps</span>
              <div className="flex gap-2">
                <button
                  type="button"
                  className="btn-secondary text-xs"
                  onClick={() => setSteps((s) => [...s, emptyWait()])}
                >
                  Add wait
                </button>
                <button
                  type="button"
                  className="btn-secondary text-xs"
                  onClick={() => setSteps((s) => [...s, emptySend()])}
                >
                  Add send
                </button>
              </div>
            </div>
            {steps.map((s, i) => (
              <div key={i} className="grid gap-2 rounded-md border border-zinc-800 bg-zinc-950/60 p-3">
                <div className="flex items-center justify-between gap-2">
                  <select
                    className="field w-auto"
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
                    className="btn-danger text-xs"
                    disabled={steps.length <= 1}
                    onClick={() => setSteps((prev) => prev.filter((_, idx) => idx !== i))}
                  >
                    Remove
                  </button>
                </div>
                {s.type === "wait" ? (
                  <Field label="Seconds" htmlFor={`auto-wait-${i}`}>
                    <input
                      id={`auto-wait-${i}`}
                      className="field"
                      value={s.seconds}
                      onChange={(e) => updateStep(i, { seconds: e.target.value })}
                      inputMode="numeric"
                    />
                  </Field>
                ) : (
                  <>
                    <Field label="From" htmlFor={`auto-from-${i}`}>
                      <input
                        id={`auto-from-${i}`}
                        className="field"
                        value={s.from}
                        onChange={(e) => updateStep(i, { from: e.target.value })}
                      />
                    </Field>
                    <Field label="Subject" htmlFor={`auto-subj-${i}`}>
                      <input
                        id={`auto-subj-${i}`}
                        className="field"
                        value={s.subject}
                        onChange={(e) => updateStep(i, { subject: e.target.value })}
                      />
                    </Field>
                    <Field label="HTML" htmlFor={`auto-html-${i}`}>
                      <textarea
                        id={`auto-html-${i}`}
                        className="field min-h-[72px] font-mono text-xs"
                        value={s.html}
                        onChange={(e) => updateStep(i, { html: e.target.value })}
                      />
                    </Field>
                  </>
                )}
              </div>
            ))}
          </div>

          <div className="flex flex-wrap gap-2">
            <button type="submit" className="btn-primary">
              {editingId ? "Save changes" : "Create automation"}
            </button>
            {editingId && (
              <button type="button" onClick={resetEditor} className="btn-secondary">
                Cancel edit
              </button>
            )}
          </div>
        </FormPanel>
      )}

      {list.length === 0 ? (
        <EmptyState title="No automations yet" hint="Create a workflow from a contact or email event." />
      ) : (
      <ul className="space-y-2">
        {list.map((a) => (
          <li key={a.id} className="rounded-lg border border-zinc-800 px-4 py-3 text-sm">
            <div className="flex items-center justify-between gap-4">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-zinc-100">{a.name}</span>
                  <StatusChip status={a.enabled ? "active" : "draft"} />
                </div>
                <div className="mt-0.5 text-xs text-zinc-500">
                  {a.trigger_type} · {(a.steps || []).length} steps
                </div>
                {(a.steps || []).length > 0 && (
                  <ol className="mt-2 list-inside list-decimal space-y-0.5 text-xs text-zinc-500">
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
              <div className="flex shrink-0 gap-2">
                <button
                  type="button"
                  disabled={busy === a.id}
                  onClick={() => loadIntoEditor(a)}
                  className="btn-secondary text-xs"
                >
                  Edit
                </button>
                <button
                  type="button"
                  disabled={busy === a.id}
                  onClick={() => loadRuns(a.id)}
                  className="btn-secondary text-xs"
                >
                  Runs
                </button>
                <button
                  type="button"
                  disabled={busy === a.id}
                  onClick={() => toggle(a)}
                  className="btn-secondary text-xs"
                >
                  {a.enabled ? "Disable" : "Enable"}
                </button>
                <button
                  type="button"
                  disabled={busy === a.id}
                  onClick={() => remove(a.id)}
                  className="btn-danger text-xs"
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
                      <li key={r.id} className="flex justify-between font-mono text-xs text-zinc-400">
                        <span>
                          {r.status} · step {r.current_step}
                        </span>
                        <span className="tabular-nums">{new Date(r.created_at).toLocaleString()}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}
          </li>
        ))}
      </ul>
      )}
    </div>
  );
}
