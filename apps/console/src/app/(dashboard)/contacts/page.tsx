"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch } from "@/lib/api";

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
    load().catch((e) => setMsg(e.message));
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
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "failed");
    }
  }

  async function createSegment(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      await apiFetch("/segments", {
        method: "POST",
        body: JSON.stringify({ name: segmentName }),
      });
      setSegmentName("");
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "segment failed");
    }
  }

  async function createProperty(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
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
    }
  }

  async function removeProperty(id: string) {
    try {
      await apiFetch(`/contact-properties/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "delete property failed");
    }
  }

  async function remove(id: string) {
    try {
      await apiFetch(`/contacts/${id}`, { method: "DELETE" });
      await load();
    } catch (err: unknown) {
      setMsg(err instanceof Error ? err.message : "delete failed");
    }
  }

  return (
    <div>
      <h1 className="font-[family-name:var(--font-display)] text-4xl mb-8">Contacts</h1>
      {msg && <p className="mb-4 text-sm text-red-400">{msg}</p>}

      <form onSubmit={create} className="mb-6 grid gap-2 max-w-2xl">
        <div className="flex flex-wrap gap-2">
          <input
            className="flex-1 min-w-[12rem] rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
            placeholder="user@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
          <input
            className="w-36 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
            placeholder="First name"
            value={firstName}
            onChange={(e) => setFirstName(e.target.value)}
          />
          <select
            className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
            value={selectedSegment}
            onChange={(e) => setSelectedSegment(e.target.value)}
          >
            <option value="">No segment</option>
            {segments.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
          <button type="submit" className="rounded-md bg-orange-500 px-4 py-2 text-sm font-medium text-black">
            Add
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
      </form>

      <form onSubmit={createSegment} className="mb-4 flex gap-2 max-w-md">
        <input
          className="flex-1 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="New segment name"
          value={segmentName}
          onChange={(e) => setSegmentName(e.target.value)}
          required
        />
        <button type="submit" className="rounded-md border border-zinc-700 px-4 py-2 text-sm hover:bg-zinc-900">
          Create segment
        </button>
      </form>

      <form onSubmit={createProperty} className="mb-8 flex flex-wrap gap-2 max-w-xl">
        <input
          className="flex-1 min-w-[10rem] rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          placeholder="Property key (e.g. plan)"
          value={propKey}
          onChange={(e) => setPropKey(e.target.value)}
          required
        />
        <select
          className="rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm"
          value={propType}
          onChange={(e) => setPropType(e.target.value)}
        >
          <option value="string">string</option>
          <option value="number">number</option>
        </select>
        <button type="submit" className="rounded-md border border-zinc-700 px-4 py-2 text-sm hover:bg-zinc-900">
          Add property
        </button>
      </form>

      {properties.length > 0 && (
        <ul className="mb-6 flex flex-wrap gap-2">
          {properties.map((p) => (
            <li
              key={p.id}
              className="flex items-center gap-2 rounded-md border border-zinc-800 px-3 py-1.5 text-xs text-zinc-300"
            >
              <span>
                {p.key} · {p.type}
              </span>
              <button type="button" onClick={() => removeProperty(p.id)} className="text-red-400 hover:underline">
                Remove
              </button>
            </li>
          ))}
        </ul>
      )}

      {segments.length > 0 && (
        <p className="mb-4 text-xs text-zinc-500">
          Segments: {segments.map((s) => s.name).join(", ")}
        </p>
      )}

      <ul className="space-y-2">
        {list.map((c) => (
          <li
            key={c.id}
            className="flex items-center justify-between rounded-lg border border-zinc-800 px-4 py-3 text-sm text-zinc-200"
          >
            <Link href={`/contacts/${c.id}`} className="min-w-0 hover:text-orange-400">
              <span>{c.email}</span>
              {c.first_name && <span className="ml-2 text-zinc-500">{c.first_name}</span>}
              {c.unsubscribed && <span className="ml-2 text-xs text-zinc-500">unsubscribed</span>}
              {c.properties && Object.keys(c.properties).length > 0 && (
                <div className="mt-1 text-xs text-zinc-500">
                  {Object.entries(c.properties)
                    .map(([k, v]) => `${k}=${String(v)}`)
                    .join(" · ")}
                </div>
              )}
            </Link>
            <button type="button" onClick={() => remove(c.id)} className="text-xs text-red-400 hover:underline shrink-0 ml-4">
              Delete
            </button>
          </li>
        ))}
        {list.length === 0 && <p className="text-sm text-zinc-500">No contacts yet</p>}
      </ul>
    </div>
  );
}
