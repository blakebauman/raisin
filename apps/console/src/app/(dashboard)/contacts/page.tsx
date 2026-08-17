"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

export default function ContactsPage() {
  const [list, setList] = useState<any[]>([]);
  const [segments, setSegments] = useState<any[]>([]);
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [segmentName, setSegmentName] = useState("");
  const [selectedSegment, setSelectedSegment] = useState("");
  const [msg, setMsg] = useState("");

  async function load() {
    const [c, s] = await Promise.all([
      apiFetch<{ data: any[] }>("/contacts"),
      apiFetch<{ data: any[] }>("/segments"),
    ]);
    setList(c.data ?? []);
    setSegments(s.data ?? []);
  }

  useEffect(() => {
    load().catch((e) => setMsg(e.message));
  }, []);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setMsg("");
    try {
      const contact = await apiFetch<{ id: string }>("/contacts", {
        method: "POST",
        body: JSON.stringify({
          email,
          first_name: firstName || undefined,
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

      <form onSubmit={create} className="mb-6 flex flex-wrap gap-2 max-w-2xl">
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
      </form>

      <form onSubmit={createSegment} className="mb-8 flex gap-2 max-w-md">
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
            <div>
              <span>{c.email}</span>
              {c.first_name && <span className="ml-2 text-zinc-500">{c.first_name}</span>}
              {c.unsubscribed && <span className="ml-2 text-xs text-zinc-500">unsubscribed</span>}
            </div>
            <button type="button" onClick={() => remove(c.id)} className="text-xs text-red-400 hover:underline">
              Delete
            </button>
          </li>
        ))}
        {list.length === 0 && <p className="text-sm text-zinc-500">No contacts yet</p>}
      </ul>
    </div>
  );
}
