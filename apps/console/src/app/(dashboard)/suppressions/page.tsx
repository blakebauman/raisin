"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import {
  EmptyState,
  Field,
  FormPanel,
  FormRow,
  Msg,
  PageHeader,
} from "@/components/ui";

export default function SuppressionsPage() {
  const [list, setList] = useState<any[]>([]);
  const [email, setEmail] = useState("");
  const [reason, setReason] = useState("manual");
  const [batch, setBatch] = useState("");
  const [msg, setMsg] = useState("");
  const [msgTone, setMsgTone] = useState<"muted" | "error" | "ok">("muted");
  const [batchOpen, setBatchOpen] = useState(false);

  async function load() {
    const res = await apiFetch<{ data: any[] }>("/suppressions");
    setList(res.data ?? []);
  }
  useEffect(() => {
    load().catch((e) => {
      setMsg(e.message);
      setMsgTone("error");
    });
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
      setMsg("Added");
      setMsgTone("ok");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
      setMsgTone("error");
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
      setBatchOpen(false);
      setMsg(`Added ${emails.length}`);
      setMsgTone("ok");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "batch failed");
      setMsgTone("error");
    }
  }

  async function remove(id: string) {
    try {
      await apiFetch(`/suppressions/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "remove failed");
      setMsgTone("error");
    }
  }

  return (
    <div>
      <PageHeader
        title="Suppressions"
        description="Addresses that must not receive mail."
        actions={
          <button type="button" className="btn-secondary" onClick={() => setBatchOpen((v) => !v)}>
            {batchOpen ? "Hide batch" : "Add batch"}
          </button>
        }
      />

      <Msg tone={msgTone}>{msg}</Msg>

      <FormRow onSubmit={add}>
        <Field label="Email" htmlFor="sup-email" className="min-w-[12rem] flex-1">
          <input
            id="sup-email"
            className="field"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </Field>
        <Field label="Reason" htmlFor="sup-reason" className="w-36">
          <select
            id="sup-reason"
            className="field"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          >
            <option value="manual">manual</option>
            <option value="bounce">bounce</option>
            <option value="complaint">complaint</option>
          </select>
        </Field>
        <button type="submit" className="btn-primary">
          Add address
        </button>
      </FormRow>

      {batchOpen && (
        <FormPanel onSubmit={addBatch} title="Batch import">
          <Field label="Addresses" htmlFor="sup-batch" hint="One email per line, or comma-separated.">
            <textarea
              id="sup-batch"
              className="field min-h-24 font-mono text-xs"
              value={batch}
              onChange={(e) => setBatch(e.target.value)}
              required
              autoFocus
            />
          </Field>
          <button type="submit" className="btn-secondary w-fit">
            Add batch
          </button>
        </FormPanel>
      )}

      {list.length === 0 ? (
        <EmptyState title="No suppressions" hint="Add addresses that must never be sent to." />
      ) : (
        <ul className="space-y-2">
          {list.map((s) => (
            <li key={s.id} className="list-row">
              <div className="min-w-0">
                <div className="text-zinc-100">{s.email}</div>
                <div className="text-xs text-zinc-500">
                  {s.reason}
                  {s.created_at ? ` · ${new Date(s.created_at).toLocaleString()}` : ""}
                </div>
              </div>
              <button type="button" onClick={() => remove(s.id)} className="btn-danger text-xs">
                Remove
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
