import {
  Braces,
  Clock,
  HelpCircle,
  Hash,
  type LucideIcon,
  Percent,
  Tag,
  ToggleLeft,
  Type,
  User,
  Users,
} from 'lucide-react';
import { AppBadge } from '@/components/app-badge';

/* Shared types + presentational helpers for the Feature Flags feature. */

export type Flag = Record<string, unknown>;

export const ACCENT = 'var(--color-accent)';
export const FLAG_TYPES = ['boolean', 'string', 'number', 'json'] as const;

const TYPE_ICON: Record<string, LucideIcon> = {
  boolean: ToggleLeft,
  string: Type,
  number: Hash,
  json: Braces,
};

/** Accent-tinted badge for a flag's value type. */
export function FlagTypeBadge({ type }: { type: string }) {
  const Icon = TYPE_ICON[type] ?? HelpCircle;
  return (
    <AppBadge
      label={type}
      color={ACCENT}
      icon={<Icon />}
    />
  );
}

interface RuleTypeStyle {
  color: string;
  icon: LucideIcon;
}

const RULE_TYPE_STYLE: Record<string, RuleTypeStyle> = {
  percentage: { color: '#8B5CF6', icon: Percent },
  attribute: { color: '#0EA5E9', icon: Tag },
  user: { color: '#0EA5E9', icon: User },
  team: { color: '#F59E0B', icon: Users },
  segment: { color: '#F59E0B', icon: Users },
  schedule: { color: '#22C55E', icon: Clock },
};

/** Colored badge for a targeting-rule type. */
export function RuleTypeBadge({ type }: { type: string }) {
  const style = RULE_TYPE_STYLE[type] ?? { color: 'var(--status-neutral)', icon: HelpCircle };
  const Icon = style.icon;
  return <AppBadge label={type} color={style.color} icon={<Icon />} />;
}

/** Format a timestamp as YYYY-MM-DD HH:MM, or 'N/A'. */
export function formatDate(ts: unknown): string {
  if (ts == null) return 'N/A';
  const str = String(ts);
  if (!str) return 'N/A';
  const dt = new Date(str);
  if (Number.isNaN(dt.getTime())) return str;
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${dt.getFullYear()}-${pad(dt.getMonth() + 1)}-${pad(dt.getDate())} ${pad(dt.getHours())}:${pad(dt.getMinutes())}`;
}

/** Compact number formatting (1.2K / 3.4M). */
export function formatNumber(n: unknown): string {
  const num = Number(n ?? 0);
  if (!Number.isFinite(num)) return '0';
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`;
  if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`;
  return String(num);
}
