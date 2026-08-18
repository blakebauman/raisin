"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import {
  EmptyState,
  Field,
  FormPanel,
  Msg,
  PageHeader,
} from "@/components/ui";

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
  const [msgTone, setMsgTone] = useState<"muted" | "error" | "ok">("muted");
  const [busy, setBusy] = useState<string | null>(null);
  const [subTopicId, setSubTopicId] = useState("");
  const [subEmail, setSubEmail] = useState("");
  const [subAction, setSubAction] = useState<"subscribe" | "unsubscribe">("subscribe");
  const [createOpen, setCreateOpen] = useState(false);
  const [subOpen, setSubOpen] = useState(false);

  async function load() {
    const res = await apiFetch<{ data: Topic[] }>("/topics");
    const data = res.data ?? [];
    setList(data);
    if (!subTopicId && data[0]) setSubTopicId(data[0].id);
  }

  useEffect(() => {
    load().catch((e) => {
      setMsg(e.message);
      setMsgTone("error");
    });
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
      setCreateOpen(false);
      setMsg("Topic created");
      setMsgTone("ok");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
      setMsgTone("error");
    }
  }

  async function remove(id: string) {
    setBusy(id);
    try {
      await apiFetch(`/topics/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "delete failed");
      setMsgTone("error");
    } finally {
      setBusy(null);
    }
  }

  async function setSubscription(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      const contacts = await apiFetch<{ data: { id: string; email: string }[] }>("/contacts");
      const contact = (contacts.data ?? []).find(
        (c) => c.email.toLowerCase() === subEmail.trim().toLowerCase(),
      );
      if (!contact) {
        setMsg("Contact not found. Create them under Contacts first.");
        setMsgTone("error");
        return;
      }
      if (!subTopicId) {
        setMsg("Select a topic");
        setMsgTone("error");
        return;
      }
      await apiFetch(`/contacts/${contact.id}/topics/${subTopicId}`, {
        method: "PUT",
        body: JSON.stringify({ subscribed: subAction === "subscribe" }),
      });
      setMsg(subAction === "subscribe" ? "Subscribed" : "Unsubscribed from topic");
      setMsgTone("ok");
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "subscription failed");
      setMsgTone("error");
    }
  }

  return (
    <div>
      <PageHeader
        title="Topics"
        description="Subscription topics for marketing and product mail."
        actions={
          <div className="flex gap-2">
            <button type="button" className="btn-secondary" onClick={() => setSubOpen((v) => !v)}>
              {subOpen ? "Hide subscription" : "Manage subscription"}
            </button>
            <button type="button" className="btn-primary" onClick={() => setCreateOpen((v) => !v)}>
              {createOpen ? "Cancel" : "Create topic"}
            </button>
          </div>
        }
      />

      <Msg tone={msgTone}>{msg}</Msg>

      {createOpen && (
        <FormPanel onSubmit={create} title="New topic">
          <Field label="Name" htmlFor="topic-name">
            <input
              id="topic-name"
              className="field"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Newsletter"
              required
              autoFocus
            />
          </Field>
          <Field label="Description" htmlFor="topic-desc">
            <input
              id="topic-desc"
              className="field"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional"
            />
          </Field>
          <Field label="Default subscription" htmlFor="topic-def">
            <select
              id="topic-def"
              className="field"
              value={defSub}
              onChange={(e) => setDefSub(e.target.value)}
            >
              <option value="opt_in">Opt-in (must subscribe)</option>
              <option value="opt_out">Opt-out (subscribed until they leave)</option>
            </select>
          </Field>
          <button type="submit" className="btn-primary w-fit">
            Create topic
          </button>
        </FormPanel>
      )}

      {subOpen && (
        <FormPanel onSubmit={setSubscription} title="Manage subscription">
          <Field label="Topic" htmlFor="sub-topic">
            <select
              id="sub-topic"
              className="field"
              value={subTopicId}
              onChange={(e) => setSubTopicId(e.target.value)}
            >
              <option value="">Select topic</option>
              {list.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Contact email" htmlFor="sub-email">
            <input
              id="sub-email"
              className="field"
              type="email"
              value={subEmail}
              onChange={(e) => setSubEmail(e.target.value)}
              required
            />
          </Field>
          <Field label="Action" htmlFor="sub-action">
            <select
              id="sub-action"
              className="field"
              value={subAction}
              onChange={(e) => setSubAction(e.target.value as "subscribe" | "unsubscribe")}
            >
              <option value="subscribe">Subscribe</option>
              <option value="unsubscribe">Unsubscribe</option>
            </select>
          </Field>
          <button type="submit" className="btn-secondary w-fit">
            Update subscription
          </button>
        </FormPanel>
      )}

      {list.length === 0 ? (
        <EmptyState title="No topics yet" hint="Create a topic, then subscribe contacts to it." />
      ) : (
        <ul className="max-w-2xl border-t border-[var(--border)]">
          {list.map((t) => (
            <li key={t.id} className="list-row">
              <div className="min-w-0">
                <div className="text-zinc-100">{t.name}</div>
                <div className="mt-0.5 text-xs text-zinc-500">
                  {t.default_subscription}
                  {t.description ? ` · ${t.description}` : ""}
                </div>
              </div>
              <button
                type="button"
                disabled={busy === t.id}
                onClick={() => remove(t.id)}
                className="btn-danger text-xs"
              >
                Delete
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
