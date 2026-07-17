import { Activity, type LucideIcon, UserPlus, Users } from 'lucide-react';

const STATS: { label: string; value: string; icon: LucideIcon }[] = [
  { label: 'Total users', value: '—', icon: Users },
  { label: 'Active sessions', value: '—', icon: Activity },
  { label: 'New signups (30d)', value: '—', icon: UserPlus },
];

export function UsageTab() {
  return (
    <div className="pb-8">
      <h2 className="text-[length:var(--text-title)] font-semibold text-text-primary">Usage</h2>
      <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
        Authentication activity for the past 30 days.
      </p>

      <div className="mt-6 grid grid-cols-1 gap-3 sm:grid-cols-3">
        {STATS.map(({ label, value, icon: Icon }) => (
          <div key={label} className="rounded-[var(--radius)] border border-border bg-surface p-5">
            <Icon size={16} className="text-text-secondary" />
            <div className="mt-3 text-[length:var(--text-h2)] font-bold text-text-primary">{value}</div>
            <div className="mt-1 text-[length:var(--text-label)] text-text-secondary">{label}</div>
          </div>
        ))}
      </div>

      <div className="mt-6 flex h-52 items-center justify-center rounded-[var(--radius)] border border-border bg-surface">
        <span className="text-[length:var(--text-body)] text-text-subtle">Usage charts coming soon</span>
      </div>
    </div>
  );
}
