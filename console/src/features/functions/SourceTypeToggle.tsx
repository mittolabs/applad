import { Code2, GitBranch } from 'lucide-react';
import { cn } from '@/lib/utils';

/* Ports `_SourceTypeBtn` — a two-option segmented toggle between inline code
 * and a GitHub repository, used by the create dialog and the settings tab. */
export type SourceType = 'inline' | 'git';

export function SourceTypeToggle({
  value,
  onChange,
}: {
  value: SourceType;
  onChange: (value: SourceType) => void;
}) {
  return (
    <div className="flex gap-2">
      <ToggleButton
        label="Inline code"
        icon={<Code2 size={14} />}
        selected={value === 'inline'}
        onClick={() => onChange('inline')}
      />
      <ToggleButton
        label="GitHub"
        icon={<GitBranch size={14} />}
        selected={value === 'git'}
        onClick={() => onChange('git')}
      />
    </div>
  );
}

function ToggleButton({
  label,
  icon,
  selected,
  onClick,
}: {
  label: string;
  icon: React.ReactNode;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-[var(--radius)] border px-3 py-2 text-[length:var(--text-body)] transition-colors',
        selected
          ? 'border-[var(--color-accent)] bg-[color-mix(in_srgb,var(--color-accent)_15%,transparent)] font-semibold text-[var(--color-accent)]'
          : 'border-field-border bg-field-fill text-text-secondary hover:text-text-primary',
      )}
    >
      {icon}
      {label}
    </button>
  );
}
