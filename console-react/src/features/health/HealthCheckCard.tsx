import { StatusChip, type StatusVariant } from '@/components/status-chip';
import type { HealthCheck } from './useHealth';

const STATUS_VARIANT: Record<string, StatusVariant> = {
  pass: 'success',
  warn: 'warning',
  fail: 'danger',
};

export function statusVariantFor(status: string): StatusVariant {
  return STATUS_VARIANT[status] ?? 'danger';
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-3 pt-2">
      <span className="w-[72px] shrink-0 text-[length:var(--text-label)] text-text-subtle">{label}</span>
      <span className="min-w-0 flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-secondary">
        {value}
      </span>
    </div>
  );
}

export function HealthCheckCard({ check }: { check: HealthCheck }) {
  const color =
    check.status === 'pass'
      ? 'var(--status-success)'
      : check.status === 'warn'
        ? 'var(--status-warning)'
        : 'var(--status-danger)';

  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-[18px]">
      <div className="flex items-center gap-2.5">
        <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: color }} />
        <span className="min-w-0 flex-1 truncate text-[length:var(--text-subhead)] font-semibold text-text-primary">
          {check.label}
        </span>
        <StatusChip label={check.status} variant={statusVariantFor(check.status)} />
      </div>

      <div className="mt-3">
        <Metric label="Endpoint" value={check.path} />
        <Metric label="Ping" value={`${check.ping} ms`} />
      </div>

      {check.error && (
        <div className="mt-3 rounded-[var(--radius)] bg-status-danger/10 p-3 text-[length:var(--text-label)] leading-relaxed text-status-danger">
          {check.error}
        </div>
      )}
    </div>
  );
}
