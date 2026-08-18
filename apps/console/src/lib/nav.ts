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
  type LucideIcon,
} from "lucide-react";

export type NavItem = { href: string; label: string; icon: LucideIcon; hint?: string };

export const navGroups: { label?: string; items: NavItem[] }[] = [
  {
    items: [{ href: "/overview", label: "Overview", icon: LayoutDashboard, hint: "Sending health and recent mail" }],
  },
  {
    label: "Mail",
    items: [
      { href: "/emails", label: "Emails", icon: Mail, hint: "Transactional sends" },
      { href: "/activity", label: "Activity", icon: Activity, hint: "Live delivery events" },
      { href: "/received", label: "Received", icon: Inbox, hint: "Inbound mail" },
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

export const allNavItems = navGroups.flatMap((g) => g.items);
