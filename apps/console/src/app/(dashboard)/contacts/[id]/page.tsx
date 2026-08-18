"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch } from "@/lib/api";

type Contact = {
  id: string;
  email: string;
  first_name?: string | null;
  last_name?: string | null;
  unsubscribed?: boolean;
  properties?: Record<string, unknown>;
};

type Segment = { id: string; name: string };
type ContactTopic = {
  topic_id: string;
  name?: string;
  subscribed: boolean;
  default_subscription?: string;
};
type Property = { id: string; key: string; type: string };

export default function ContactDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const [contact, setContact] = useState<Contact | null>(null);
  const [segments, setSegments] = useState<Segment[]>([]);
  const [memberSegs, setMemberSegs] = useState<Segment[]>([]);
  const [contactTopics, setContactTopics] = useState<ContactTopic[]>([]);
  const [properties, setProperties] = useState<Property[]>([]);
  const [propValues, setPropValues] = useState<Record<string, string>>({});
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    const [c, allSeg, mem, ctops, props] = await Promise.all([
      apiFetch<Contact>(`/contacts/${id}`),
      apiFetch<{ data: Segment[] }>("/segments"),
      apiFetch<{ data: Segment[] }>(`/contacts/${id}/segments`),
      apiFetch<{ data: ContactTopic[] }>(`/contacts/${id}/topics`),
      apiFetch<{ data: Property[] }>("/contact-properties"),
    ]);
    setContact(c);
    setFirstName(c.first_name ?? "");
    setLastName(c.last_name ?? "");
    setSegments(allSeg.data ?? []);
    setMemberSegs(mem.data ?? []);
    setContactTopics(ctops.data ?? []);
    setProperties(props.data ?? []);
    const vals: Record<string, string> = {};
    for (const p of props.data ?? []) {
      const v = c.properties?.[p.key];
      vals[p.key] = v == null ? "" : String(v);
    }
    setPropValues(vals);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, [id]);

  async function saveProfile(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setMsg("");
    try {
      const propertiesPayload: Record<string, string | number> = {};
      for (const p of properties) {
        const raw = (propValues[p.key] ?? "").trim();
        if (!raw) continue;
        propertiesPayload[p.key] = p.type === "number" ? Number(raw) : raw;
      }
      await apiFetch(`/contacts/${id}`, {
        method: "PATCH",
        body: JSON.stringify({
          first_name: firstName,
          last_name: lastName,
          properties: propertiesPayload,
        }),
      });
      setMsg("Saved");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "save failed");
    } finally {
      setBusy(false);
    }
  }

  async function toggleUnsub() {
    if (!contact) return;
    setBusy(true);
    try {
      await apiFetch(`/contacts/${id}`, {
        method: "PATCH",
        body: JSON.stringify({ unsubscribed: !contact.unsubscribed }),
      });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "update failed");
    } finally {
      setBusy(false);
    }
  }

  async function toggleSegment(segId: string, inSeg: boolean) {
    setBusy(true);
    try {
      await apiFetch(`/contacts/${id}/segments/${segId}`, {
        method: inSeg ? "DELETE" : "POST",
        body: inSeg ? undefined : "{}",
      });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "segment update failed");
    } finally {
      setBusy(false);
    }
  }

  async function setTopic(topicId: string, subscribed: boolean) {
    setBusy(true);
    try {
      await apiFetch(`/contacts/${id}/topics/${topicId}`, {
        method: "PUT",
        body: JSON.stringify({ subscribed }),
      });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "topic update failed");
    } finally {
      setBusy(false);
    }
  }

  const memberIds = new Set(memberSegs.map((s) => s.id));

  if (!contact) {
    return (
      <div>
        <p className="text-sm text-zinc-500">{msg || "Loading…"}</p>
      </div>
    );
  }

  return (
    <div className="max-w-3xl">
      <Link href="/contacts" className="text-xs text-zinc-500 hover:text-zinc-300">
        ← Contacts
      </Link>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mt-3 mb-2 text-zinc-50">
        {contact.email}
      </h1>
      <p className="text-sm text-zinc-500 mb-8">
        {contact.unsubscribed ? "Unsubscribed" : "Subscribed"} · id {contact.id.slice(0, 8)}…
      </p>
      {msg && <p className="mb-4 text-sm text-zinc-400">{msg}</p>}

      <form onSubmit={saveProfile} className="mb-10 grid gap-3">
        <div className="flex flex-wrap gap-2">
          <input
            className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
            placeholder="First name"
            value={firstName}
            onChange={(e) => setFirstName(e.target.value)}
          />
          <input
            className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
            placeholder="Last name"
            value={lastName}
            onChange={(e) => setLastName(e.target.value)}
          />
          <button
            type="button"
            disabled={busy}
            onClick={toggleUnsub}
            className="rounded-md border border-zinc-700 px-3 py-2 text-xs hover:bg-zinc-900"
          >
            {contact.unsubscribed ? "Re-subscribe" : "Unsubscribe"}
          </button>
        </div>
        {properties.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {properties.map((p) => (
              <input
                key={p.id}
                className="w-40 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
                placeholder={`${p.key} (${p.type})`}
                type={p.type === "number" ? "number" : "text"}
                value={propValues[p.key] ?? ""}
                onChange={(e) => setPropValues((prev) => ({ ...prev, [p.key]: e.target.value }))}
              />
            ))}
          </div>
        )}
        <button
          type="submit"
          disabled={busy}
          className="w-fit rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black disabled:opacity-50"
        >
          Save
        </button>
      </form>

      <section className="mb-10">
        <h2 className="text-sm uppercase tracking-wider text-zinc-500 mb-3">Segments</h2>
        <ul className="space-y-2">
          {segments.map((s) => {
            const on = memberIds.has(s.id);
            return (
              <li key={s.id} className="flex items-center justify-between rounded-md border border-zinc-800 px-3 py-2 text-sm">
                <span>{s.name}</span>
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => toggleSegment(s.id, on)}
                  className="text-xs text-orange-400 hover:underline"
                >
                  {on ? "Remove" : "Add"}
                </button>
              </li>
            );
          })}
          {segments.length === 0 && <p className="text-sm text-zinc-500">No segments yet</p>}
        </ul>
      </section>

      <section>
        <h2 className="text-sm uppercase tracking-wider text-zinc-500 mb-3">Topics</h2>
        <ul className="space-y-2">
          {contactTopics.map((t) => (
            <li key={t.topic_id} className="flex items-center justify-between rounded-md border border-zinc-800 px-3 py-2 text-sm">
              <span>
                {t.name ?? t.topic_id}
                <span className="ml-2 text-xs text-zinc-500">{t.subscribed ? "subscribed" : "opted out"}</span>
              </span>
              <button
                type="button"
                disabled={busy}
                onClick={() => setTopic(t.topic_id, !t.subscribed)}
                className="text-xs text-orange-400 hover:underline"
              >
                {t.subscribed ? "Opt out" : "Subscribe"}
              </button>
            </li>
          ))}
          {contactTopics.length === 0 && <p className="text-sm text-zinc-500">No topics yet</p>}
        </ul>
      </section>
    </div>
  );
}
