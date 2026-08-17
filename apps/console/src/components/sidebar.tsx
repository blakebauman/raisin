"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Mail,
  Globe,
  Key,
  Webhook,
  Users,
  FileText,
  Megaphone,
  ScrollText,
  Settings,
  Ban,
  Inbox,
  Workflow,
  Server,
  Shield,
  Tags,
} from "lucide-react";

const nav = [
  { href: "/overview", label: "Overview", icon: LayoutDashboard },
  { href: "/emails", label: "Emails", icon: Mail },
  { href: "/received", label: "Received", icon: Inbox },
  { href: "/domains", label: "Domains", icon: Globe },
  { href: "/automations", label: "Automations", icon: Workflow },
  { href: "/ips", label: "Dedicated IPs", icon: Server },
  { href: "/api-keys", label: "API Keys", icon: Key },
  { href: "/oauth", label: "OAuth Apps", icon: Shield },
  { href: "/webhooks", label: "Webhooks", icon: Webhook },
  { href: "/contacts", label: "Contacts", icon: Users },
  { href: "/topics", label: "Topics", icon: Tags },
  { href: "/suppressions", label: "Suppressions", icon: Ban },
  { href: "/templates", label: "Templates", icon: FileText },
  { href: "/broadcasts", label: "Broadcasts", icon: Megaphone },
  { href: "/logs", label: "Logs", icon: ScrollText },
  { href: "/settings", label: "Settings", icon: Settings },
];

export function Sidebar() {
  const path = usePathname();
  return (
    <aside className="w-56 shrink-0 border-r border-zinc-800/80 bg-[#0f0f12]/80 backdrop-blur px-3 py-6 flex flex-col gap-6">
      <Link href="/overview" className="px-2">
        <div className="font-[family-name:var(--font-display)] text-2xl tracking-tight text-zinc-50">
          Raisin
        </div>
        <div className="text-[11px] uppercase tracking-[0.18em] text-zinc-500 mt-1">
          raisin.run
        </div>
      </Link>
      <nav className="flex flex-col gap-0.5">
        {nav.map((item) => {
          const active = path === item.href || (item.href !== "/overview" && path.startsWith(item.href));
          const Icon = item.icon;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition ${
                active
                  ? "bg-zinc-800/80 text-orange-400"
                  : "text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900"
              }`}
            >
              <Icon size={16} strokeWidth={1.75} />
              {item.label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
