"use client";

import { useEffect, useState } from "react";
import { Menu } from "lucide-react";
import { Sidebar } from "@/components/sidebar";
import { CommandMenu } from "@/components/command-menu";
import { WorkspaceMark } from "@/components/ui";

export function Shell({ children }: { children: React.ReactNode }) {
  const [navOpen, setNavOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setSearchOpen((v) => !v);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return (
    <div className="flex h-dvh overflow-hidden bg-[var(--background)]">
      <Sidebar open={navOpen} onClose={() => setNavOpen(false)} onSearch={() => setSearchOpen(true)} />
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex h-11 shrink-0 items-center gap-2 border-b border-[var(--border)] px-3 lg:hidden">
          <button
            type="button"
            className="btn-ghost h-8 w-8 px-0"
            aria-label="Open navigation"
            onClick={() => setNavOpen(true)}
          >
            <Menu size={16} />
          </button>
          <WorkspaceMark size={16} />
          <span className="text-[13px] font-medium">Raisin</span>
        </div>
        <main className="min-h-0 flex-1 overflow-auto">
          <div className="px-5 py-5 lg:px-8 lg:py-6">{children}</div>
        </main>
      </div>
      <CommandMenu open={searchOpen} onClose={() => setSearchOpen(false)} />
    </div>
  );
}
