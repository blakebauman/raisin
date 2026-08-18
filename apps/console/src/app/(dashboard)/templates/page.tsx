"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import {
  EmptyState,
  Field,
  FormPanel,
  Msg,
  PageHeader,
  StatusChip,
} from "@/components/ui";

export default function TemplatesPage() {
  const [list, setList] = useState<any[]>([]);
  const [name, setName] = useState("Welcome");
  const [subject, setSubject] = useState("Welcome {{name}}");
  const [html, setHtml] = useState("<p>Hi {{name}}, welcome.</p>");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [open, setOpen] = useState(false);

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
      setOpen(false);
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
      <PageHeader
        title="Templates"
        description="Reusable HTML for transactional and campaign mail."
        actions={
          <button type="button" className="btn-primary" onClick={() => setOpen((v) => !v)}>
            {open ? "Cancel" : "New template"}
          </button>
        }
      />

      {open && (
        <FormPanel onSubmit={create} title="New template">
          <Field label="Name" htmlFor="tpl-name">
            <input
              id="tpl-name"
              className="field"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              autoFocus
            />
          </Field>
          <Field label="Subject" htmlFor="tpl-subject" hint="Supports {{variables}}.">
            <input
              id="tpl-subject"
              className="field"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              required
            />
          </Field>
          <Field label="HTML" htmlFor="tpl-html">
            <textarea
              id="tpl-html"
              className="field min-h-24 font-mono text-xs"
              value={html}
              onChange={(e) => setHtml(e.target.value)}
              required
            />
          </Field>
          <button type="submit" className="btn-primary w-fit">
            Create template
          </button>
        </FormPanel>
      )}

      <Msg>{msg}</Msg>

      {list.length === 0 ? (
        <EmptyState title="No templates yet" hint="Create a draft, then edit and publish." />
      ) : (
        <ul className="space-y-2">
          {list.map((t) => (
            <li key={t.id} className="list-row">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-zinc-100">{t.name}</span>
                  <StatusChip status={t.status} />
                </div>
              </div>
              <div className="flex shrink-0 gap-2">
                <a href={`/templates/${t.id}/edit`} className="btn-secondary text-xs">
                  Edit
                </a>
                {t.status !== "published" && (
                  <button
                    type="button"
                    disabled={busy === t.id}
                    onClick={() => publish(t.id)}
                    className="btn-secondary text-xs"
                  >
                    Publish
                  </button>
                )}
                <button
                  type="button"
                  disabled={busy === t.id}
                  onClick={() => remove(t.id)}
                  className="btn-danger text-xs"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
