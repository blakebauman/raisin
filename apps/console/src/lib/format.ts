export function formatRelative(iso?: string | null): string {
  if (!iso) return "—";
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "—";
  const delta = Math.round((Date.now() - t) / 1000);
  const future = delta < 0;
  const n = Math.abs(delta);
  let label: string;
  if (n < 45) label = "just now";
  else if (n < 3600) label = `${Math.round(n / 60)}m`;
  else if (n < 86400) label = `${Math.round(n / 3600)}h`;
  else if (n < 86400 * 7) label = `${Math.round(n / 86400)}d`;
  else label = new Date(iso).toLocaleDateString();
  if (label === "just now") return label;
  return future ? `in ${label}` : label;
}

export function formatAbsolute(iso?: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString();
}
