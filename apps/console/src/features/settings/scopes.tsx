import { useState } from 'react';
import { Check, ChevronDown, ChevronUp, Copy } from 'lucide-react';
import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';

/* Shared API-key scope + expiry primitives, ported from settings_page.dart /
 * api_key_detail_page.dart (_kScopeGroups, _kExpiryOptions, scope selector,
 * copy button). Reused by both the create dialog and the detail page. */

export const SCOPE_GROUPS: Record<string, string[]> = {
  Auth: ['auth.read', 'auth.write'],
  Databases: ['databases.read', 'databases.write'],
  Storage: ['storage.read', 'storage.write'],
  Functions: ['functions.read', 'functions.write', 'functions.execute'],
  Messaging: ['messaging.read', 'messaging.write'],
  Deploy: ['deploy.read', 'deploy.write'],
  Workflows: ['workflows.read', 'workflows.write', 'workflows.execute'],
};

export const EXPIRY_OPTIONS: { value: string; label: string }[] = [
  { value: 'never', label: 'Never' },
  { value: '1d', label: '1 Day' },
  { value: '7d', label: '7 Days' },
  { value: '30d', label: '30 Days' },
  { value: '90d', label: '90 Days' },
  { value: '1y', label: '1 Year' },
  { value: 'custom', label: 'Custom date…' },
];

const MONTHS = [
  'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
  'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
];

const DAY_MS = 86_400_000;

/** Compute the ISO expiry from a preset key + optional custom yyyy-mm-dd. */
export function expiresAtIso(expiry: string, customDate: string): string | null {
  if (expiry === 'never') return null;
  if (expiry === 'custom') {
    if (!customDate) return null;
    return new Date(`${customDate}T23:59:59`).toISOString();
  }
  const days: Record<string, number> = { '1d': 1, '7d': 7, '30d': 30, '90d': 90, '1y': 365 };
  const d = days[expiry];
  if (!d) return null;
  return new Date(Date.now() + d * DAY_MS).toISOString();
}

/** "Your key will expire on Mon D, YYYY" preview, or null for never. */
export function expiryPreview(expiry: string, customDate: string): string | null {
  const iso = expiresAtIso(expiry, customDate);
  if (!iso) return null;
  const t = new Date(iso);
  return `Your key will expire on ${MONTHS[t.getMonth()]} ${t.getDate()}, ${t.getFullYear()}`;
}

/** Format an ISO timestamp as "Mon D, YYYY", or null when empty/invalid. */
export function formatLongDate(iso: string | null | undefined): string | null {
  if (!iso) return null;
  const t = new Date(iso);
  if (Number.isNaN(t.getTime())) return null;
  return `${MONTHS[t.getMonth()]} ${t.getDate()}, ${t.getFullYear()}`;
}

// --- Scope selector ---------------------------------------------------------

export function ScopeGroups({
  selected,
  onToggleScope,
  onToggleGroup,
}: {
  selected: Set<string>;
  onToggleScope: (scope: string) => void;
  onToggleGroup: (group: string) => void;
}) {
  return (
    <div className="flex flex-col">
      {Object.entries(SCOPE_GROUPS).map(([group, scopes]) => (
        <ScopeGroupRow
          key={group}
          group={group}
          scopes={scopes}
          selected={selected}
          onToggleScope={onToggleScope}
          onToggleGroup={onToggleGroup}
        />
      ))}
    </div>
  );
}

function ScopeGroupRow({
  group,
  scopes,
  selected,
  onToggleScope,
  onToggleGroup,
}: {
  group: string;
  scopes: string[];
  selected: Set<string>;
  onToggleScope: (scope: string) => void;
  onToggleGroup: (group: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const selectedCount = scopes.filter((s) => selected.has(s)).length;
  const checkState: boolean | 'indeterminate' =
    selectedCount === scopes.length ? true : selectedCount > 0 ? 'indeterminate' : false;

  return (
    <div className="mb-1 overflow-hidden rounded-[var(--radius)] border border-border bg-fill">
      <div
        role="button"
        tabIndex={0}
        onClick={() => setExpanded((v) => !v)}
        onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && setExpanded((v) => !v)}
        className="flex cursor-pointer items-center gap-2 px-3 py-2.5"
      >
        <span onClick={(e) => e.stopPropagation()} className="flex items-center">
          <Checkbox
            checked={checkState}
            onCheckedChange={() => onToggleGroup(group)}
            aria-label={`Toggle ${group} scopes`}
          />
        </span>
        <span className="text-[length:var(--text-body)] font-medium text-text-primary">
          {group}
        </span>
        <span className="text-[length:var(--text-label)] text-text-subtle">
          {selectedCount} {selectedCount === 1 ? 'Scope' : 'Scopes'}
        </span>
        <span className="ml-auto text-text-subtle">
          {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
        </span>
      </div>
      {expanded && (
        <div className="border-t border-border">
          {scopes.map((scope) => (
            <label
              key={scope}
              className="flex cursor-pointer items-center gap-2 py-2 pl-10 pr-3"
            >
              <Checkbox
                checked={selected.has(scope)}
                onCheckedChange={() => onToggleScope(scope)}
              />
              <span className="font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-secondary">
                {scope}
              </span>
            </label>
          ))}
        </div>
      )}
    </div>
  );
}

// --- Copy button ------------------------------------------------------------

/** Icon copy button with a 2s green-tick feedback (ports _CopyButton). */
export function CopyButton({ text, size = 14 }: { text: string; size?: number }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard unavailable */
    }
  };
  return (
    <button
      type="button"
      onClick={copy}
      className={cn(
        'transition-colors',
        copied ? 'text-status-success' : 'text-text-subtle hover:text-text-primary',
      )}
      aria-label="Copy"
    >
      {copied ? <Check size={size} /> : <Copy size={size} />}
    </button>
  );
}
