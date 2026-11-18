import type { Health } from "@/lib/types";
import { timeAgo } from "@/lib/format";

export default function HealthBadge({ health }: { health: Health }) {
  if (!health || health.checks === 0) {
    return <span className="badge badge-muted" title="No probe has completed yet">unverified</span>;
  }
  const cls = !health.last_ok ? "badge-bad" : health.uptime_pct >= 95 ? "badge-good" : "badge-warn";
  return (
    <span className={`badge ${cls}`} title={`${health.checks} checks, p50 ${health.p50_ms} ms`}>
      {health.last_ok ? "verified" : "down"} {timeAgo(health.last_verified)} · {health.uptime_pct.toFixed(0)}% up
    </span>
  );
}
