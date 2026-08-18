import { WorkspaceMark } from "@/components/ui";

const rows = [
  { subject: "Welcome to Acme", to: "ava@example.com", status: "delivered", when: "2m" },
  { subject: "Password reset", to: "noah@example.com", status: "opened", when: "14m" },
  { subject: "Invoice 1842", to: "billing@orbit.dev", status: "clicked", when: "1h" },
  { subject: "Trial ending", to: "sam@north.io", status: "queued", when: "3h" },
  { subject: "Weekly digest", to: "team@acme.test", status: "bounced", when: "5h" },
];

const statusClass: Record<string, string> = {
  delivered: "text-emerald-400 bg-emerald-400/10",
  opened: "text-orange-300 bg-orange-400/10",
  clicked: "text-orange-300 bg-orange-400/10",
  queued: "text-amber-300 bg-amber-400/10",
  bounced: "text-red-400 bg-red-400/10",
};

export function ProductPreview() {
  return (
    <div
      aria-hidden
      className="overflow-hidden rounded-xl border border-white/15 bg-[#121316] shadow-[0_50px_120px_-36px_rgba(0,0,0,0.95)]"
    >
      <div className="flex min-h-[22rem] md:min-h-[26rem]">
        <div className="hidden w-[208px] shrink-0 border-r border-white/10 bg-[#0d0e10] sm:flex sm:flex-col">
          <div className="flex h-11 items-center gap-2 px-3">
            <WorkspaceMark size={16} />
            <span className="text-[13px] font-medium text-zinc-100">Raisin</span>
          </div>
          <div className="px-3 pb-2">
            <div className="flex h-8 items-center rounded-full border border-white/10 bg-black/40 px-3 text-[12px] text-zinc-500">
              Search
            </div>
          </div>
          <div className="flex flex-col gap-px px-3 text-[13px]">
            {["Overview", "Emails", "Activity", "Contacts", "Domains"].map((label) => (
              <div
                key={label}
                className={`flex h-8 items-center rounded-full px-2.5 ${
                  label === "Emails" ? "bg-white/10 text-zinc-50" : "text-zinc-500"
                }`}
              >
                {label}
              </div>
            ))}
          </div>
        </div>
        <div className="min-w-0 flex-1 bg-[#16171a]">
          <div className="flex h-11 items-center justify-between border-b border-white/10 px-4">
            <div>
              <div className="text-[13px] font-medium text-zinc-100">Emails</div>
              <div className="text-[12px] text-zinc-500">279 sent this period</div>
            </div>
            <div className="rounded-full bg-[var(--accent)] px-3 py-1 text-[12px] font-medium text-[#140b04]">
              Compose
            </div>
          </div>
          <div>
            {rows.map((row) => (
              <div
                key={row.subject}
                className="grid grid-cols-[1fr_auto] items-center gap-3 border-b border-white/10 px-4 py-2.5 sm:grid-cols-[minmax(0,1fr)_150px_96px_36px]"
              >
                <div className="truncate text-[13px] text-zinc-100">{row.subject}</div>
                <div className="hidden truncate text-[12px] text-zinc-500 sm:block">{row.to}</div>
                <div>
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-[12px] ${statusClass[row.status]}`}>
                    {row.status}
                  </span>
                </div>
                <div className="text-right text-[12px] tabular-nums text-zinc-500">{row.when}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
