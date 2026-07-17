import { type ReactNode } from 'react';
import { ChevronLeft, ChevronRight, Search } from 'lucide-react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';
import { cn } from '@/lib/utils';

export const PER_PAGE_OPTIONS = [6, 12, 25];

export function totalPages(total: number, perPage: number) {
  return Math.min(Math.max(Math.ceil(total / perPage), 1), 999999);
}

/* Ports search_list.dart SearchListHeader — 280px search field + spacer +
 * optional trailing widget. Search fires on submit (Enter). */
export function SearchListHeader({
  searchHint = 'Search by name or ID',
  value,
  onChange,
  onSearch,
  trailing,
  className,
}: {
  searchHint?: string;
  value: string;
  onChange: (v: string) => void;
  onSearch?: () => void;
  trailing?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('flex items-center', className)}>
      <div className="relative w-[280px]">
        <Search
          size={16}
          className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-text-subtle"
        />
        <input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && onSearch?.()}
          placeholder={searchHint}
          className="h-[var(--control-h)] w-full rounded-[var(--radius)] border border-border bg-field-fill pl-8 pr-3 text-[length:var(--text-body)] text-text-primary placeholder:text-text-subtle focus:border-[var(--color-accent)] focus:outline-none"
        />
      </div>
      <div className="ml-auto">{trailing}</div>
    </div>
  );
}

/* Ports search_list.dart SearchListFooter — per-page dropdown (6/12/25) +
 * "<label> per page. Total: N" + Prev / current-page badge / Next. */
export function SearchListFooter({
  total,
  perPage,
  currentPage,
  onPrev,
  onNext,
  onPerPageChange,
  itemLabel = 'items',
  className,
}: {
  total: number;
  perPage: number;
  currentPage: number;
  onPrev: () => void;
  onNext: () => void;
  onPerPageChange: (n: number) => void;
  itemLabel?: string;
  className?: string;
}) {
  const pages = totalPages(total, perPage);
  const canPrev = currentPage > 1;
  const canNext = currentPage < pages;

  return (
    <div className={cn('flex items-center pt-3', className)}>
      <Select value={String(perPage)} onValueChange={(v) => onPerPageChange(Number(v))}>
        <SelectTrigger className="w-[64px] text-[length:var(--text-label)]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {PER_PAGE_OPTIONS.map((n) => (
            <SelectItem key={n} value={String(n)}>
              {n}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <span className="ml-3 text-[length:var(--text-label)] text-text-subtle">
        {itemLabel} per page. Total: {total}
      </span>

      <div className="ml-auto flex items-center gap-1.5">
        <PageNavButton dir="prev" enabled={canPrev} onClick={onPrev} />
        <div className="flex h-7 w-7 items-center justify-center rounded-[var(--radius-6)] border border-border bg-fill-active text-[length:var(--text-label)] font-medium text-text-primary">
          {currentPage}
        </div>
        <PageNavButton dir="next" enabled={canNext} onClick={onNext} />
      </div>
    </div>
  );
}

function PageNavButton({
  dir,
  enabled,
  onClick,
}: {
  dir: 'prev' | 'next';
  enabled: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={!enabled}
      onClick={onClick}
      className={cn(
        'flex items-center gap-0.5 text-[length:var(--text-label)] transition-colors',
        enabled
          ? 'cursor-pointer text-text-muted hover:text-text-secondary'
          : 'cursor-default text-text-subtle',
      )}
    >
      {dir === 'prev' && <ChevronLeft size={16} />}
      {dir === 'prev' ? 'Prev' : 'Next'}
      {dir === 'next' && <ChevronRight size={16} />}
    </button>
  );
}
