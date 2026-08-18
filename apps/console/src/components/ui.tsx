import type { FormEvent, ReactNode } from "react";
import Link from "next/link";
import { ChevronLeft } from "lucide-react";

export function PageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <header className="mb-5 flex flex-wrap items-center justify-between gap-3">
      <div className="min-w-0">
        <h1 className="text-[15px] font-medium tracking-tight text-zinc-50 text-balance">{title}</h1>
        {description && (
          <p className="mt-0.5 max-w-2xl text-pretty text-[13px] text-[var(--muted)]">{description}</p>
        )}
      </div>
      {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
    </header>
  );
}

export function SectionLabel({ children }: { children: ReactNode }) {
  return <h2 className="mb-2 text-[13px] font-medium text-zinc-300">{children}</h2>;
}

export function EmptyState({ title, hint }: { title: string; hint?: string }) {
  return (
    <div className="py-12">
      <p className="text-[13px] text-zinc-200">{title}</p>
      {hint && <p className="mt-1 max-w-md text-pretty text-[13px] text-[var(--muted)]">{hint}</p>}
    </div>
  );
}

export function StatusChip({ status }: { status: string }) {
  const tone =
    status === "delivered" ||
    status === "sent" ||
    status === "verified" ||
    status === "active" ||
    status === "published" ||
    status === "subscribed"
      ? "text-emerald-400 bg-emerald-400/10"
      : status === "failed" ||
          status === "bounced" ||
          status === "complained" ||
          status === "canceled" ||
          status === "unsubscribed"
        ? "text-red-400 bg-red-400/10"
        : status === "queued" ||
            status === "scheduled" ||
            status === "sending" ||
            status === "draft" ||
            status === "pending" ||
            status === "partial"
          ? "text-amber-300 bg-amber-400/10"
          : "text-orange-300 bg-orange-400/10";
  return (
    <span className={`inline-flex rounded-full px-2 py-0.5 text-[12px] tabular-nums ${tone}`}>{status}</span>
  );
}

export function Msg({
  children,
  tone = "muted",
}: {
  children: ReactNode;
  tone?: "muted" | "error" | "ok";
}) {
  if (!children) return null;
  const cls =
    tone === "error" ? "text-red-400" : tone === "ok" ? "text-emerald-400" : "text-[var(--muted)]";
  return <p className={`mb-4 text-[13px] ${cls}`}>{children}</p>;
}

/** Labeled control for stack forms. Prefer over placeholder-only inputs. */
export function Field({
  label,
  hint,
  htmlFor,
  children,
  className,
}: {
  label: string;
  hint?: string;
  htmlFor?: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={`grid gap-1.5 ${className ?? ""}`}>
      <label htmlFor={htmlFor} className="text-[12px] font-medium text-zinc-400">
        {label}
      </label>
      {children}
      {hint ? <p className="text-[12px] text-[var(--muted-2)]">{hint}</p> : null}
    </div>
  );
}

/** Bordered create/edit panel. Use when the form has 2+ fields. */
export function FormPanel({
  children,
  title,
  onSubmit,
}: {
  children: ReactNode;
  title?: string;
  onSubmit?: (e: FormEvent) => void;
}) {
  const body = (
    <>
      {title ? <SectionLabel>{title}</SectionLabel> : null}
      {children}
    </>
  );
  if (onSubmit) {
    return (
      <form onSubmit={onSubmit} className="form-panel">
        {body}
      </form>
    );
  }
  return <div className="form-panel">{body}</div>;
}

/** Inline single-row create: input(s) + primary button aligned to end. */
export function FormRow({ children, onSubmit }: { children: ReactNode; onSubmit?: (e: FormEvent) => void }) {
  if (onSubmit) {
    return (
      <form onSubmit={onSubmit} className="form-row">
        {children}
      </form>
    );
  }
  return <div className="form-row">{children}</div>;
}

export function SecretCallout({ children }: { children: ReactNode }) {
  return (
    <div className="mb-6 max-w-xl rounded-2xl border border-orange-500/25 bg-orange-500/10 px-3 py-2.5 text-[13px] text-zinc-200">
      {children}
    </div>
  );
}

export function WorkspaceMark({ size = 18 }: { size?: number }) {
  return (
    <span
      className="inline-flex shrink-0 items-center justify-center rounded-full bg-[var(--accent)] font-semibold text-[#140b04]"
      style={{ width: size, height: size, fontSize: Math.round(size * 0.58) }}
      aria-hidden
    >
      R
    </span>
  );
}

export function BackLink({ href, children }: { href: string; children: ReactNode }) {
  return (
    <Link
      href={href}
      className="inline-flex items-center gap-0.5 text-[12px] text-[var(--muted)] transition hover:text-zinc-200"
    >
      <ChevronLeft size={14} strokeWidth={1.75} />
      {children}
    </Link>
  );
}

export function LiveDot({ live, liveLabel = "Live", idleLabel = "Idle" }: { live: boolean; liveLabel?: string; idleLabel?: string }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[12px] ${
        live ? "bg-emerald-400/10 text-emerald-400" : "bg-white/[0.04] text-[var(--muted)]"
      }`}
    >
      <span className={`h-1.5 w-1.5 rounded-full ${live ? "bg-emerald-400" : "bg-zinc-600"}`} />
      {live ? liveLabel : idleLabel}
    </span>
  );
}

export function PropertyRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3 border-t border-[var(--border)] py-2 text-[13px] first:border-t-0">
      <dt className="shrink-0 text-[var(--muted)]">{label}</dt>
      <dd className="min-w-0 text-right text-zinc-200">{children}</dd>
    </div>
  );
}

export function TableSkeleton({ rows = 8 }: { rows?: number }) {
  return (
    <div className="border-t border-[var(--border)]" aria-hidden>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex h-9 items-center border-b border-[var(--border)] px-3">
          <div
            className="h-2.5 rounded-sm bg-white/[0.05]"
            style={{ width: `${42 + ((i * 17) % 38)}%` }}
          />
        </div>
      ))}
    </div>
  );
}

export function MailFrame({ html, title = "preview" }: { html: string; title?: string }) {
  return (
    <iframe
      title={title}
      className="min-h-72 w-full rounded-2xl border border-[var(--border)] bg-white"
      srcDoc={html}
    />
  );
}
