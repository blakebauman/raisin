"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { allNavItems } from "@/lib/nav";

export function CommandMenu({ open, onClose }: { open: boolean; onClose: () => void }) {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [index, setIndex] = useState(0);

  const results = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return allNavItems;
    return allNavItems.filter(
      (item) =>
        item.label.toLowerCase().includes(q) ||
        item.href.toLowerCase().includes(q) ||
        (item.hint ?? "").toLowerCase().includes(q),
    );
  }, [query]);

  useEffect(() => {
    if (!open) return;
    setQuery("");
    setIndex(0);
    const t = window.setTimeout(() => inputRef.current?.focus(), 10);
    return () => window.clearTimeout(t);
  }, [open]);

  useEffect(() => {
    setIndex(0);
  }, [query]);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
      } else if (e.key === "ArrowDown") {
        e.preventDefault();
        setIndex((i) => Math.min(i + 1, Math.max(results.length - 1, 0)));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter" && results[index]) {
        e.preventDefault();
        router.push(results[index].href);
        onClose();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose, results, index, router]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-[var(--z-modal)] flex items-start justify-center bg-black/55 px-4 pt-[18vh]"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Search console"
        className="w-full max-w-lg overflow-hidden rounded-2xl border border-[var(--border)] bg-[#121214] shadow-2xl shadow-black/50"
        onClick={(e) => e.stopPropagation()}
      >
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Go to a page…"
          className="h-11 w-full border-b border-[var(--border)] bg-transparent px-3.5 text-[15px] text-zinc-100 outline-none placeholder:text-[var(--muted)]"
        />
        <ul className="max-h-80 overflow-auto p-1">
          {results.length === 0 ? (
            <li className="px-3 py-6 text-[13px] text-[var(--muted)]">No matching pages</li>
          ) : (
            results.map((item, i) => {
              const Icon = item.icon;
              const active = i === index;
              return (
                <li key={item.href}>
                  <button
                    type="button"
                    onMouseEnter={() => setIndex(i)}
                    onClick={() => {
                      router.push(item.href);
                      onClose();
                    }}
                    className={`flex h-8 w-full items-center gap-2 rounded-full px-2.5 text-left text-[13px] ${
                      active ? "bg-white/[0.07] text-zinc-50" : "text-zinc-300"
                    }`}
                  >
                    <Icon size={15} strokeWidth={1.75} className={active ? "text-orange-400" : "text-[var(--muted)]"} />
                    <span className="flex-1">{item.label}</span>
                    {item.hint && <span className="truncate text-[12px] text-[var(--muted-2)]">{item.hint}</span>}
                  </button>
                </li>
              );
            })
          )}
        </ul>
      </div>
    </div>
  );
}
