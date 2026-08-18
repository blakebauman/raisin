import type { FormEvent, ReactNode } from "react";

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
    <header className="mb-8 flex flex-wrap items-start justify-between gap-4">
      <div className="min-w-0">
        <h1 className="font-[family-name:var(--font-display)] text-3xl tracking-tight text-zinc-50 text-balance">
          {title}
        </h1>
        {description && (
          <p className="mt-1.5 max-w-2xl text-pretty text-sm text-zinc-500">{description}</p>
        )}
      </div>
      {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
    </header>
  );
}

export function SectionLabel({ children }: { children: ReactNode }) {
  return (
    <h2 className="mb-3 text-xs font-medium uppercase tracking-wider text-zinc-500">{children}</h2>
  );
}

export function EmptyState({ title, hint }: { title: string; hint?: string }) {
  return (
    <div className="rounded-lg border border-dashed border-zinc-800 px-6 py-10 text-center">
      <p className="text-sm text-zinc-300">{title}</p>
      {hint && <p className="mt-1.5 text-pretty text-xs text-zinc-500">{hint}</p>}
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
      ? "text-emerald-300 bg-emerald-950/50"
      : status === "failed" ||
          status === "bounced" ||
          status === "complained" ||
          status === "canceled" ||
          status === "unsubscribed"
        ? "text-red-300 bg-red-950/40"
        : status === "queued" ||
            status === "scheduled" ||
            status === "sending" ||
            status === "draft" ||
            status === "pending" ||
            status === "partial"
          ? "text-amber-200 bg-amber-950/40"
          : "text-orange-300 bg-zinc-800";
  return (
    <span className={`inline-flex rounded px-2 py-0.5 text-xs tabular-nums ${tone}`}>{status}</span>
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
    tone === "error" ? "text-red-400" : tone === "ok" ? "text-emerald-400" : "text-zinc-400";
  return <p className={`mb-4 text-sm ${cls}`}>{children}</p>;
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
      <label htmlFor={htmlFor} className="text-xs font-medium text-zinc-400">
        {label}
      </label>
      {children}
      {hint ? <p className="text-xs text-zinc-600">{hint}</p> : null}
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
      <form onSubmit={onSubmit} className="form-panel mb-8">
        {body}
      </form>
    );
  }
  return <div className="form-panel mb-8">{body}</div>;
}

/** Inline single-row create: input(s) + primary button aligned to end. */
export function FormRow({ children, onSubmit }: { children: ReactNode; onSubmit?: (e: FormEvent) => void }) {
  if (onSubmit) {
    return (
      <form onSubmit={onSubmit} className="form-row mb-6">
        {children}
      </form>
    );
  }
  return <div className="form-row mb-6">{children}</div>;
}

export function SecretCallout({ children }: { children: ReactNode }) {
  return (
    <div className="mb-6 max-w-xl rounded-md border border-orange-500/30 bg-orange-500/10 px-4 py-3 text-sm text-zinc-200">
      {children}
    </div>
  );
}
