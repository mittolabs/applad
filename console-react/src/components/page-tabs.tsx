import { cn } from '@/lib/utils';

/* Ports page_tabs.dart — horizontal text tabs with an animated 2px underline
 * on the active tab, a 1px border under the whole row.
 * label 14; active w500/text-primary + text-primary underline; inactive
 * w400/text-muted; 24px gap between tabs; underline animates 150ms. */
export function PageTabs({
  tabs,
  selected,
  onChange,
  className,
}: {
  tabs: string[];
  selected: number;
  onChange: (index: number) => void;
  className?: string;
}) {
  return (
    <div className={cn('flex border-b border-border', className)}>
      {tabs.map((tab, i) => {
        const active = i === selected;
        return (
          <button
            key={tab}
            type="button"
            onClick={() => onChange(i)}
            className={cn(
              'mr-6 -mb-px cursor-pointer border-b-2 pb-2.5 text-[length:var(--text-control)] transition-colors',
              active
                ? 'border-[var(--text-primary)] font-medium text-text-primary'
                : 'border-transparent font-normal text-text-muted hover:text-text-secondary',
            )}
          >
            {tab}
          </button>
        );
      })}
    </div>
  );
}
