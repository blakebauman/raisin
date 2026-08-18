"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { EmptyState, Msg, PageHeader, SectionLabel, StatusChip } from "@/components/ui";

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
  const [msgTone, setMsgTone] = useState<"muted" | "error" | "ok">("muted");
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
    load().catch((e) => {
      setMsg(e.message);
      setMsgTone("error");
    });
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
      setMsgTone("ok");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "save failed");
      setMsgTone("error");
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
      setMsgTone("error");
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
      setMsgTone("error");
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
      setMsgTone("error");
    } finally {
      setBusy(false);
    }
  }

  const memberIds = new Set(memberSegs.map((s) => s.id));

  if (!contact) {
    return (
      <div>
        <Link href="/contacts" className="text-xs text-zinc-500 hover:text-zinc-300">
          ← Contacts
        </Link>
        <p className="mt-6 text-sm text-zinc-500">{msg || "Loading…"}</p>
      </div>
    );
  }

  return (
    <div className="max-w-3xl">
      <Link href="/contacts" className="text-xs text-zinc-500 hover:text-zinc-300">
        ← Contacts
      </Link>
      <div className="mt-3">
        <PageHeader
          title={contact.email}
          description={`id ${contact.id.slice(0, 8)}…`}
          actions={<StatusChip status={contact.unsubscribed ? "unsubscribed" : "subscribed"} />}
        />
      </div>

      <Msg tone={msgTone}>{msg}</Msg>

      <form onSubmit={saveProfile} className="mb-10 grid gap-3">
        <SectionLabel>Profile</SectionLabel>
        <div className="grid gap-3 sm:grid-cols-2">
          <input
            className="field"
            placeholder="First name"
            value={firstName}
            onChange={(e) => setFirstName(e.target.value)}
          />
          <input
            className="field"
            placeholder="Last name"
            value={lastName}
            onChange={(e) => setLastName(e.target.value)}
          />
        </div>
        {properties.length > 0 && (
          <div className="grid gap-3 sm:grid-cols-2">
            {properties.map((p) => (
              <input
                key={p.id}
                className="field"
                placeholder={`${p.key} (${p.type})`}
                type={p.type === "number" ? "number" : "text"}
                value={propValues[p.key] ?? ""}
                onChange={(e) => setPropValues((prev) => ({ ...prev, [p.key]: e.target.value }))}
              />
            ))}
          </div>
        )}
        <div className="flex flex-wrap gap-2">
          <button type="submit" disabled={busy} className="btn-primary">
            Save
          </button>
          <button type="button" disabled={busy} onClick={toggleUnsub} className="btn-secondary">
            {contact.unsubscribed ? "Re-subscribe" : "Unsubscribe"}
          </button>
        </div>
      </form>

      <section className="mb-10">
        <SectionLabel>Segments</SectionLabel>
        {segments.length === 0 ? (
          <EmptyState title="No segments yet" hint="Create segments from the contacts page." />
        ) : (
          <ul className="border-t border-[var(--border)]">
            {segments.map((s) => {
              const on = memberIds.has(s.id);
              return (
                <li key={s.id} className="list-row">
                  <span className="text-zinc-200">{s.name}</span>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => toggleSegment(s.id, on)}
                    className="text-xs text-orange-400 hover:underline disabled:opacity-50"
                  >
                    {on ? "Remove" : "Add"}
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <section>
        <SectionLabel>Topics</SectionLabel>
        {contactTopics.length === 0 ? (
          <EmptyState title="No topics yet" hint="Create topics under Audience → Topics." />
        ) : (
          <ul className="border-t border-[var(--border)]">
            {contactTopics.map((t) => (
              <li key={t.topic_id} className="list-row">
                <span className="text-zinc-200">
                  {t.name ?? t.topic_id}
                  <span className="ml-2 text-xs text-zinc-500">
                    {t.subscribed ? "subscribed" : "opted out"}
                  </span>
                </span>
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => setTopic(t.topic_id, !t.subscribed)}
                  className="text-xs text-orange-400 hover:underline disabled:opacity-50"
                >
                  {t.subscribed ? "Opt out" : "Subscribe"}
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
