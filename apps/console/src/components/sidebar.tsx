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
  Activity,
} from "lucide-react";

type NavItem = { href: string; label: string; icon: typeof Mail };

const groups: { label?: string; items: NavItem[] }[] = [
  {
    items: [{ href: "/overview", label: "Overview", icon: LayoutDashboard }],
  },
  {
    label: "Mail",
    items: [
      { href: "/emails", label: "Emails", icon: Mail },
      { href: "/activity", label: "Activity", icon: Activity },
      { href: "/received", label: "Received", icon: Inbox },
    ],
  },
  {
    label: "Audience",
    items: [
      { href: "/contacts", label: "Contacts", icon: Users },
      { href: "/topics", label: "Topics", icon: Tags },
      { href: "/suppressions", label: "Suppressions", icon: Ban },
    ],
  },
  {
    label: "Campaigns",
    items: [
      { href: "/templates", label: "Templates", icon: FileText },
      { href: "/broadcasts", label: "Broadcasts", icon: Megaphone },
      { href: "/automations", label: "Automations", icon: Workflow },
    ],
  },
  {
    label: "Platform",
    items: [
      { href: "/domains", label: "Domains", icon: Globe },
      { href: "/ips", label: "Dedicated IPs", icon: Server },
      { href: "/webhooks", label: "Webhooks", icon: Webhook },
      { href: "/api-keys", label: "API Keys", icon: Key },
      { href: "/oauth", label: "OAuth Apps", icon: Shield },
    ],
  },
  {
    label: "Account",
    items: [
      { href: "/logs", label: "API Logs", icon: ScrollText },
      { href: "/settings", label: "Settings", icon: Settings },
    ],
  },
];

export function Sidebar() {
  const path = usePathname();
  return (
    <aside className="flex w-56 shrink-0 flex-col gap-5 border-r border-zinc-800/80 bg-[var(--panel-2)]/90 px-3 py-5 backdrop-blur">
      <Link href="/overview" className="px-2.5">
        <div className="font-[family-name:var(--font-display)] text-2xl tracking-tight text-zinc-50">
          Raisin
        </div>
        <div className="mt-0.5 text-[11px] uppercase tracking-[0.16em] text-zinc-500">Console</div>
      </Link>
      <nav className="flex flex-1 flex-col gap-4 overflow-y-auto pb-4">
        {groups.map((group) => (
          <div key={group.label ?? "top"} className="flex flex-col gap-0.5">
            {group.label && (
              <div className="px-2.5 pb-1 text-[10px] font-medium uppercase tracking-wider text-zinc-600">
                {group.label}
              </div>
            )}
            {group.items.map((item) => {
              const active =
                path === item.href || (item.href !== "/overview" && path.startsWith(item.href));
              const Icon = item.icon;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm transition ${
                    active
                      ? "bg-zinc-800/90 text-orange-400"
                      : "text-zinc-400 hover:bg-zinc-900 hover:text-zinc-100"
                  }`}
                >
                  <Icon size={15} strokeWidth={1.75} className="opacity-90" />
                  {item.label}
                </Link>
              );
            })}
          </div>
        ))}
      </nav>
    </aside>
  );
}
