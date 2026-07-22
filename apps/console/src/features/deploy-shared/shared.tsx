import { type ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, Loader2, Upload, type LucideIcon } from 'lucide-react';
import { api } from '@/api/client';
import { IdText } from '@/components/id-text';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

/* Shared bits for the Mobile and Desktop deploy-target pages. Both are thin
 * compositions over `/deploy/targets`; the detail tabs (Overview / Builds /
 * Signing / Distribution / Settings) reuse these primitives. */

export type Target = Record<string, unknown>;

export function fmtDate(v: unknown): string {
  if (v == null) return '—';
  return String(v).split('T')[0];
}

export function TabSpinner() {
  return (
    <div className="flex justify-center p-12">
      <Loader2 className="animate-spin text-[var(--color-accent)]" size={24} />
    </div>
  );
}

export function DetailHeader({ title, onBack }: { title: string; onBack: () => void }) {
  return (
    <div className="flex items-center gap-2">
      <button
        type="button"
        onClick={onBack}
        className="rounded-[var(--radius-6)] p-1 text-text-secondary transition-colors hover:bg-fill hover:text-text-primary"
        aria-label="Back"
      >
        <ArrowLeft size={18} />
      </button>
      <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">{title}</h1>
    </div>
  );
}

export function InfoCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex-1 rounded-[var(--radius)] border border-border bg-surface p-4">
      <div className="text-[length:var(--text-label)] text-text-subtle">{label}</div>
      <div className="mt-1.5 text-[length:var(--text-control)] font-medium text-text-primary">
        {value}
      </div>
    </div>
  );
}

export function SettingRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex pb-4">
      <div className="w-[140px] shrink-0 text-[length:var(--text-body)] text-text-subtle">
        {label}
      </div>
      <div className="flex-1 text-[length:var(--text-body)] text-text-primary">{value}</div>
    </div>
  );
}

export function DangerZone({ description, onDelete }: { description: string; onDelete: () => void }) {
  return (
    <div
      className="rounded-[var(--radius)] border p-5"
      style={{ borderColor: 'color-mix(in srgb, var(--color-danger) 30%, transparent)' }}
    >
      <div className="text-[length:var(--text-control)] font-semibold text-[var(--color-danger)]">
        Danger zone
      </div>
      <div className="mt-2 text-[length:var(--text-body)] text-text-subtle">{description}</div>
      <Button
        variant="outline"
        size="sm"
        onClick={onDelete}
        className="mt-3 border-[var(--color-danger)] text-[var(--color-danger)] hover:bg-[color-mix(in_srgb,var(--color-danger)_10%,transparent)] hover:text-[var(--color-danger)]"
      >
        Delete app
      </Button>
    </div>
  );
}

/* Reads /deploy/releases?targetId= and renders the build history list. Shared by
 * Mobile and Desktop Builds tabs. */
export function BuildsTab({ targetId }: { targetId: string }) {
  const q = useQuery({
    queryKey: ['deploy-releases', targetId],
    queryFn: async () => {
      const res = await api.get('/deploy/releases', { params: { targetId } });
      return res.data as Record<string, unknown>;
    },
  });

  if (q.isLoading) return <TabSpinner />;
  const releases = (q.data?.['releases'] as Record<string, unknown>[] | undefined) ?? [];
  if (releases.length === 0) {
    return (
      <div className="p-8 text-center text-[length:var(--text-body)] text-text-secondary">
        No builds yet
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-2">
      {releases.map((r, i) => {
        const status = String(r['status'] ?? 'pending');
        const color =
          status === 'completed'
            ? 'var(--status-success)'
            : status === 'failed'
              ? 'var(--status-danger)'
              : 'var(--status-warning)';
        return (
          <div
            key={String(r['$id'] ?? i)}
            className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-surface p-3.5"
          >
            <span
              className="h-2 w-2 shrink-0 rounded-full"
              style={{ backgroundColor: color }}
            />
            <div className="min-w-0 flex-1">
              <IdText id={String(r['$id'] ?? '')} fontSize={12} />
              <div className="text-[length:var(--text-caption)] text-text-subtle">
                {String(r['triggerType'] ?? 'manual')} • {String(r['durationMs'] ?? 0)}ms
              </div>
            </div>
            <span className="text-[length:var(--text-label)]" style={{ color }}>
              {status}
            </span>
          </div>
        );
      })}
    </div>
  );
}

export function SigningUploadCard({
  title,
  description,
  buttonLabel,
}: {
  title: string;
  description: string;
  buttonLabel: string;
}) {
  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-5">
      <div className="text-[length:var(--text-control)] text-text-primary">{title}</div>
      <div className="mt-1 text-[length:var(--text-body)] text-text-secondary">{description}</div>
      <Button variant="outline" size="sm" className="mt-3">
        <Upload size={14} />
        {buttonLabel}
      </Button>
    </div>
  );
}

export function ConfigSection({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-5">
      <div className="text-[length:var(--text-control)] text-text-primary">{title}</div>
      <div className="mt-1 text-[length:var(--text-body)] text-text-subtle">{description}</div>
      <div className="mt-4 flex flex-col">{children}</div>
    </div>
  );
}

export function ConfigField({ label, value }: { label: string; value: string }) {
  const empty = value.trim() === '';
  return (
    <div className="flex items-center pb-2.5">
      <div className="w-[180px] shrink-0 text-[length:var(--text-body)] text-text-subtle">
        {label}
      </div>
      <div
        className={cn(
          'flex-1 rounded-[var(--radius-6)] border border-field-border bg-fill px-3 py-2 text-[length:var(--text-body)]',
          empty ? 'text-text-subtle' : 'text-text-primary',
        )}
      >
        {empty ? '—' : value}
      </div>
    </div>
  );
}

/* Selectable chip used in the create dialogs (platform / framework choices). */
export function ChoiceChip({
  label,
  icon: Icon,
  active,
  onClick,
  className,
}: {
  label: string;
  icon?: LucideIcon;
  active: boolean;
  onClick: () => void;
  className?: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex flex-col items-center justify-center gap-1.5 rounded-[var(--radius)] border px-4 py-3 transition-colors',
        active
          ? 'border-[var(--color-accent)] bg-[color-mix(in_srgb,var(--color-accent)_15%,transparent)]'
          : 'border-border bg-surface hover:border-field-border',
        className,
      )}
    >
      {Icon && (
        <Icon
          size={18}
          className={active ? 'text-[var(--color-accent)]' : 'text-text-secondary'}
        />
      )}
      <span
        className={cn(
          'text-[length:var(--text-body)]',
          active ? 'font-semibold text-text-primary' : 'text-text-secondary',
        )}
      >
        {label}
      </span>
    </button>
  );
}
