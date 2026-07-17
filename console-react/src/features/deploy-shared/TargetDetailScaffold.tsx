import { type ReactNode } from 'react';
import { ChevronLeft } from 'lucide-react';
import { PageTabs } from '@/components/page-tabs';
import { IdText } from '@/components/id-text';

/* Shared detail scaffold for deploy targets (Sites + Containers).
 * Renders the back-breadcrumb header (with an ID badge) + the tab bar,
 * then the active tab body supplied by the caller. Ports the detail header
 * in sites_page.dart / containers_page.dart. */

export function TargetDetailScaffold({
  backLabel,
  onBack,
  name,
  id,
  tabs,
  tab,
  onTab,
  headerAction,
  children,
}: {
  backLabel: string;
  onBack: () => void;
  name: string;
  id: string;
  tabs: string[];
  tab: number;
  onTab: (index: number) => void;
  headerAction?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-4 p-6 md:p-8">
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1 text-[length:var(--text-control)] text-text-muted transition-colors hover:text-text-primary"
        >
          <ChevronLeft size={16} />
          {backLabel}
        </button>
        <span className="text-[length:var(--text-body)] text-text-subtle">/</span>
        <span className="min-w-0 flex-1 truncate text-[length:var(--text-h1)] font-semibold text-text-primary">
          {name}
        </span>
        {id && (
          <span className="rounded-[var(--radius-sm)] border border-border bg-fill px-2 py-[3px]">
            <IdText id={id} fontSize={11} />
          </span>
        )}
        {headerAction}
      </div>

      <PageTabs tabs={tabs} selected={tab} onChange={onTab} />

      <div>{children}</div>
    </div>
  );
}
