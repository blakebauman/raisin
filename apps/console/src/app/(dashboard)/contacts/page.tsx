"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";
import { EmptyState, Field, Msg, PageHeader, SectionLabel } from "@/components/ui";

type Contact = {
  id: string;
  email: string;
  first_name?: string | null;
  unsubscribed?: boolean;
  properties?: Record<string, unknown>;
};

type Segment = { id: string; name: string };
type Property = { id: string; key: string; type: string };

export default function ContactsPage() {
  const [list, setList] = useState<Contact[]>([]);
  const [segments, setSegments] = useState<Segment[]>([]);
  const [properties, setProperties] = useState<Property[]>([]);
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [segmentName, setSegmentName] = useState("");
  const [selectedSegment, setSelectedSegment] = useState("");
  const [propKey, setPropKey] = useState("");
  const [propType, setPropType] = useState("string");
  const [propValues, setPropValues] = useState<Record<string, string>>({});
  const [msg, setMsg] = useState("");
  const [msgTone, setMsgTone] = useState<"muted" | "error" | "ok">("muted");
  const [setupOpen, setSetupOpen] = useState(false);

  async function load() {
    const [c, s, p] = await Promise.all([
      apiFetch<{ data: Contact[] }>("/contacts"),
      apiFetch<{ data: Segment[] }>("/segments"),
      apiFetch<{ data: Property[] }>("/contact-properties"),
    ]);
    setList(c.data ?? []);
    setSegments(s.data ?? []);
    setProperties(p.data ?? []);
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
      const propertiesPayload: Record<string, string | number> = {};
      for (const p of properties) {
        const raw = (propValues[p.key] ?? "").trim();
        if (!raw) continue;
        propertiesPayload[p.key] = p.type === "number" ? Number(raw) : raw;
      }
      const contact = await apiFetch<{ id: string }>("/contacts", {
        method: "POST",
        body: JSON.stringify({
          email,
          first_name: firstName || undefined,
          properties: Object.keys(propertiesPayload).length ? propertiesPayload : undefined,
        }),
      });
      if (selectedSegment && contact.id) {
        await apiFetch(`/contacts/${contact.id}/segments/${selectedSegment}`, {
          method: "POST",
          body: "{}",
        });
      }
      setEmail("");
      setFirstName("");
      setPropValues({});
      setMsg("Contact added");
      setMsgTone("ok");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
      setMsgTone("error");
    }
  }

  async function createSegment(e: React.FormEvent) {
    e.preventDefault();
    try {
      await apiFetch("/segments", { method: "POST", body: JSON.stringify({ name: segmentName }) });
      setSegmentName("");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "segment failed");
      setMsgTone("error");
    }
  }

  async function createProperty(e: React.FormEvent) {
    e.preventDefault();
    try {
      await apiFetch("/contact-properties", {
        method: "POST",
        body: JSON.stringify({ key: propKey, type: propType }),
      });
      setPropKey("");
      setPropType("string");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "property failed");
      setMsgTone("error");
    }
  }

  async function removeProperty(id: string) {
    try {
      await apiFetch(`/contact-properties/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "delete property failed");
      setMsgTone("error");
    }
  }

  async function remove(id: string) {
    try {
      await apiFetch(`/contacts/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "delete failed");
      setMsgTone("error");
    }
  }

  return (
    <div>
      <PageHeader
        title="Contacts"
        description="Audience records with properties, segments, and topic preferences."
        actions={
          <button type="button" className="btn-secondary" onClick={() => setSetupOpen((v) => !v)}>
            {setupOpen ? "Hide setup" : "Segments & properties"}
          </button>
        }
      />

      <form onSubmit={create} className="form-panel mb-6 max-w-2xl">
        <SectionLabel>Add contact</SectionLabel>
        <div className="flex flex-wrap items-end gap-2">
          <Field label="Email" htmlFor="c-email" className="min-w-[12rem] flex-1">
            <input
              id="c-email"
              className="field"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </Field>
          <Field label="First name" htmlFor="c-first" className="w-36">
            <input
              id="c-first"
              className="field"
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
            />
          </Field>
          <Field label="Segment" htmlFor="c-seg" className="w-40">
            <select
              id="c-seg"
              className="field"
              value={selectedSegment}
              onChange={(e) => setSelectedSegment(e.target.value)}
            >
              <option value="">None</option>
              {segments.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </Field>
          <button type="submit" className="btn-primary">
            Add contact
          </button>
        </div>
        {properties.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {properties.map((p) => (
              <Field key={p.id} label={p.key} htmlFor={`c-prop-${p.id}`} className="w-40">
                <input
                  id={`c-prop-${p.id}`}
                  className="field"
                  type={p.type === "number" ? "number" : "text"}
                  value={propValues[p.key] ?? ""}
                  onChange={(e) => setPropValues((prev) => ({ ...prev, [p.key]: e.target.value }))}
                />
              </Field>
            ))}
          </div>
        )}
      </form>

      {setupOpen && (
        <div className="form-panel mb-8 max-w-2xl">
          <SectionLabel>Segments & properties</SectionLabel>
          <form onSubmit={createSegment} className="flex flex-wrap items-end gap-2">
            <Field label="Segment name" htmlFor="seg-name" className="min-w-[10rem] flex-1">
              <input
                id="seg-name"
                className="field"
                value={segmentName}
                onChange={(e) => setSegmentName(e.target.value)}
                required
              />
            </Field>
            <button type="submit" className="btn-secondary">
              Create segment
            </button>
          </form>
          <form onSubmit={createProperty} className="flex flex-wrap items-end gap-2">
            <Field label="Property key" htmlFor="prop-key" className="min-w-[10rem] flex-1">
              <input
                id="prop-key"
                className="field"
                value={propKey}
                onChange={(e) => setPropKey(e.target.value)}
                required
              />
            </Field>
            <Field label="Type" htmlFor="prop-type" className="w-28">
              <select
                id="prop-type"
                className="field"
                value={propType}
                onChange={(e) => setPropType(e.target.value)}
              >
                <option value="string">string</option>
                <option value="number">number</option>
              </select>
            </Field>
            <button type="submit" className="btn-secondary">
              Add property
            </button>
          </form>
          {(properties.length > 0 || segments.length > 0) && (
            <div className="flex flex-wrap gap-2">
              {segments.map((s) => (
                <span key={s.id} className="rounded-md border border-zinc-800 px-2.5 py-1 text-xs text-zinc-400">
                  Segment · {s.name}
                </span>
              ))}
              {properties.map((p) => (
                <span
                  key={p.id}
                  className="inline-flex items-center gap-2 rounded-md border border-zinc-800 px-2.5 py-1 text-xs text-zinc-300"
                >
                  {p.key} · {p.type}
                  <button type="button" onClick={() => removeProperty(p.id)} className="text-red-400 hover:underline">
                    Remove
                  </button>
                </span>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="mb-3">
        <Msg tone={msgTone}>{msg}</Msg>
      </div>

      <SectionLabel>Directory</SectionLabel>
      {list.length === 0 ? (
        <EmptyState title="No contacts yet" hint="Add an email above, or create contacts via the API." />
      ) : (
        <ul className="space-y-1.5">
          {list.map((c) => (
            <li key={c.id} className="list-row">
              <Link href={`/contacts/${c.id}`} className="min-w-0 hover:text-orange-400">
                <div className="truncate text-zinc-100">{c.email}</div>
                <div className="mt-0.5 truncate text-xs text-zinc-500">
                  {[c.first_name, c.unsubscribed ? "unsubscribed" : null]
                    .filter(Boolean)
                    .join(" · ")}
                  {c.properties && Object.keys(c.properties).length > 0
                    ? ` · ${Object.entries(c.properties)
                        .slice(0, 3)
                        .map(([k, v]) => `${k}=${String(v)}`)
                        .join(", ")}`
                    : ""}
                </div>
              </Link>
              <button type="button" onClick={() => remove(c.id)} className="btn-danger shrink-0 px-2 py-1 text-xs">
                Delete
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
