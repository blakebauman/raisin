"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Search } from "lucide-react";
import { navGroups } from "@/lib/nav";
import { WorkspaceMark } from "@/components/ui";

export function Sidebar({
  open,
  onClose,
  onSearch,
}: {
  open: boolean;
  onClose: () => void;
  onSearch: () => void;
}) {
  const path = usePathname();
  const [email, setEmail] = useState<string | null>(null);

  useEffect(() => {
    setEmail(localStorage.getItem("raisin_email"));
  }, []);

  function signOut() {
    localStorage.clear();
    window.location.href = "/login";
  }

  return (
    <>
      <div
        className={`fixed inset-0 z-[var(--z-modal-backdrop)] bg-black/60 lg:hidden ${open ? "block" : "hidden"}`}
        onClick={onClose}
        aria-hidden={!open}
      />
      <aside
        className={`fixed inset-y-0 left-0 z-[var(--z-modal)] flex w-[244px] shrink-0 flex-col border-r border-[var(--border)] bg-[var(--panel-2)] transition-transform duration-200 ease-out lg:static lg:z-auto lg:translate-x-0 ${
          open ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <div className="flex h-12 items-center gap-2 px-3">
          <Link href="/overview" className="flex min-w-0 items-center gap-2 rounded-full px-1.5 py-1 hover:bg-white/[0.04]" onClick={onClose}>
            <WorkspaceMark />
            <span className="truncate text-[13px] font-medium text-zinc-50">Raisin</span>
          </Link>
        </div>

        <div className="px-3 pb-2">
          <button
            type="button"
            onClick={onSearch}
            className="flex h-8 w-full items-center gap-2 rounded-full border border-[var(--border)] bg-[#08090a] px-3 text-left text-[13px] text-[var(--muted)] transition hover:border-[var(--border-strong)] hover:text-zinc-300"
          >
            <Search size={14} strokeWidth={1.75} />
            <span className="flex-1">Search</span>
            <span className="kbd">⌘K</span>
          </button>
        </div>

        <nav className="flex flex-1 flex-col gap-3 overflow-y-auto px-3 pb-3">
          {navGroups.map((group) => (
            <div key={group.label ?? "top"} className="flex flex-col gap-px">
              {group.label && (
                <div className="px-2 pb-1 pt-1 text-[11px] font-medium text-[var(--muted-2)]">{group.label}</div>
              )}
              {group.items.map((item) => {
                const active =
                  path === item.href || (item.href !== "/overview" && path.startsWith(item.href));
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    onClick={onClose}
                    className={`flex h-8 items-center gap-2 rounded-full px-2.5 text-[13px] transition ${
                      active
                        ? "bg-white/[0.06] text-zinc-50"
                        : "text-[var(--muted)] hover:bg-white/[0.04] hover:text-zinc-200"
                    }`}
                  >
                    <Icon
                      size={16}
                      strokeWidth={1.75}
                      className={active ? "text-orange-400" : "opacity-80"}
                    />
                    {item.label}
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>

        <div className="border-t border-[var(--border)] p-3">
          <div className="flex items-center justify-between gap-2 px-1">
            <div className="min-w-0">
              <div className="truncate text-[12px] text-zinc-300">{email || "Workspace"}</div>
              <div className="text-[11px] text-[var(--muted-2)]">raisin.run</div>
            </div>
            <button type="button" onClick={signOut} className="btn-ghost h-6 px-2 text-[12px]">
              Sign out
            </button>
          </div>
        </div>
      </aside>
    </>
  );
}
