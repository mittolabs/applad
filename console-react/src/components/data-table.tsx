import { type ReactNode, useMemo, useState } from 'react';
import {
  type ColumnDef,
  type SortingState,
  type VisibilityState,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table';
import {
  ArrowDown,
  ArrowUp,
  ChevronsUpDown,
  Columns3,
  Filter as FilterIcon,
  LayoutGrid,
  List as ListIcon,
  type LucideIcon,
  Plus,
  Trash2,
} from 'lucide-react';
import { Button } from './ui/button';
import { Checkbox } from './ui/checkbox';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from './ui/popover';
import { SearchListFooter, SearchListHeader } from './search-list';
import { EmptyState } from './empty-state';
import { ErrorState } from './error-state';
import { IdText } from './id-text';
import { ConfirmDialog } from './form-dialog';
import { cn } from '@/lib/utils';

export type Row = Record<string, unknown>;

export interface DataTableColumn {
  key: string;
  label: string;
  /** Flex weight for column width (default 1). */
  flex?: number;
  sortable?: boolean;
  defaultVisible?: boolean;
}

export interface DataTableFilter {
  key: string;
  label: string;
  options: { value: string; label: string }[];
}

export interface DataTableProps {
  columns: DataTableColumn[];
  rows: Row[];
  /** String value for a cell (default row[key]). Used for sort + default render. */
  getCellValue?: (row: Row, key: string) => string;
  /** Custom cell renderer. The "$id" column auto-renders <IdText> unless overridden. */
  cellRender?: (row: Row, key: string) => ReactNode | undefined;
  rowIcon?: (row: Row) => LucideIcon | undefined;
  rowIconColor?: (row: Row) => string | undefined;
  onRowClick?: (row: Row) => void;
  onDeleteRow?: (row: Row) => Promise<void> | void;
  /** Title of the delete-confirm dialog (default "Delete item"). */
  deleteTitle?: string;
  /** Body copy of the delete-confirm dialog. Falls back to a generic warning. */
  deleteMessage?: string;

  createLabel?: string;
  onCreate?: () => void;
  createWidget?: ReactNode;

  // Search + pagination (owned by parent, typically via useResourceList).
  searchHint?: string;
  searchValue?: string;
  onSearchChange?: (v: string) => void;
  onSearch?: () => void;
  total?: number;
  perPage?: number;
  page?: number;
  onPerPageChange?: (n: number) => void;
  onPrev?: () => void;
  onNext?: () => void;
  itemLabel?: string;

  filters?: DataTableFilter[];
  filterValues?: Record<string, string | null>;
  onFiltersChange?: (values: Record<string, string | null>) => void;

  gridCard?: (row: Row) => ReactNode;
  persistKey?: string;

  emptyIcon?: LucideIcon;
  emptyTitle?: string;
  emptySubtitle?: string;

  loading?: boolean;
  error?: unknown;
  onRetry?: () => void;
  rowKey?: (row: Row) => string;
}

function defaultRowKey(row: Row): string {
  return String(row['$id'] ?? row['id'] ?? JSON.stringify(row));
}

export function DataTable(props: DataTableProps) {
  const {
    columns,
    rows,
    getCellValue = (row, key) => String(row[key] ?? ''),
    cellRender,
    rowIcon,
    rowIconColor,
    onRowClick,
    onDeleteRow,
    deleteTitle = 'Delete item',
    deleteMessage = 'This action cannot be undone. Are you sure you want to delete this item?',
    createLabel,
    onCreate,
    createWidget,
    searchHint,
    searchValue = '',
    onSearchChange,
    onSearch,
    total = rows.length,
    perPage = 12,
    page = 1,
    onPerPageChange,
    onPrev,
    onNext,
    itemLabel = 'items',
    filters,
    filterValues = {},
    onFiltersChange,
    gridCard,
    persistKey,
    emptyIcon,
    emptyTitle = 'Nothing here yet',
    emptySubtitle,
    loading,
    error,
    onRetry,
    rowKey = defaultRowKey,
  } = props;

  const [sorting, setSorting] = useState<SortingState>([]);
  const [visibility, setVisibility] = useState<VisibilityState>(() =>
    Object.fromEntries(columns.map((c) => [c.key, c.defaultVisible !== false])),
  );
  const [view, setView] = useState<'list' | 'grid'>(() => {
    if (!persistKey || !gridCard) return 'list';
    return (localStorage.getItem(`applad_view_${persistKey}`) as 'list' | 'grid') ?? 'list';
  });
  const [pendingDelete, setPendingDelete] = useState<Row | null>(null);
  const [deleting, setDeleting] = useState(false);

  const setViewPersist = (v: 'list' | 'grid') => {
    setView(v);
    if (persistKey) localStorage.setItem(`applad_view_${persistKey}`, v);
  };

  const tanstackColumns = useMemo<ColumnDef<Row>[]>(
    () =>
      columns.map((c) => ({
        id: c.key,
        accessorFn: (row) => getCellValue(row, c.key),
        header: c.label,
        enableSorting: c.sortable ?? false,
        meta: { flex: c.flex ?? 1 },
      })),
    [columns, getCellValue],
  );

  const table = useReactTable({
    data: rows,
    columns: tanstackColumns,
    state: { sorting, columnVisibility: visibility },
    onSortingChange: setSorting,
    onColumnVisibilityChange: setVisibility,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  const confirmDelete = async () => {
    if (!pendingDelete || !onDeleteRow) return;
    setDeleting(true);
    try {
      await onDeleteRow(pendingDelete);
      setPendingDelete(null);
    } finally {
      setDeleting(false);
    }
  };

  const renderCell = (row: Row, key: string): ReactNode => {
    const custom = cellRender?.(row, key);
    if (custom !== undefined) return custom;
    if (key === '$id' || key === 'id') return <IdText id={String(row[key] ?? '')} />;
    return getCellValue(row, key);
  };

  const trailing = (
    <div className="flex items-center gap-2">
      {filters && filters.length > 0 && onFiltersChange && (
        <FilterButton
          filters={filters}
          values={filterValues}
          onChange={onFiltersChange}
        />
      )}
      <ColumnVisibilityButton columns={columns} table={table} />
      {gridCard && persistKey && (
        <ViewToggle view={view} onChange={setViewPersist} />
      )}
      {createWidget ??
        (createLabel && onCreate && (
          <Button size="sm" onClick={onCreate}>
            <Plus size={14} />
            {createLabel}
          </Button>
        ))}
    </div>
  );

  return (
    <div className="flex flex-col gap-4">
      <SearchListHeader
        searchHint={searchHint}
        value={searchValue}
        onChange={(v) => onSearchChange?.(v)}
        onSearch={onSearch}
        trailing={trailing}
      />

      {error ? (
        <ErrorState error={error} onRetry={onRetry} />
      ) : loading ? (
        <TableSkeleton />
      ) : rows.length === 0 ? (
        <EmptyState
          icon={emptyIcon}
          title={emptyTitle}
          subtitle={emptySubtitle}
          actionLabel={createLabel}
          onAction={onCreate}
        />
      ) : view === 'grid' && gridCard ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-3">
          {rows.map((row) => (
            <div
              key={rowKey(row)}
              onClick={() => onRowClick?.(row)}
              className={cn(onRowClick && 'cursor-pointer')}
            >
              {gridCard(row)}
            </div>
          ))}
        </div>
      ) : (
        <div className="overflow-x-auto rounded-[var(--radius-10)] border border-border">
          <table className="w-full border-collapse text-left">
            <thead>
              {table.getHeaderGroups().map((hg) => (
                <tr key={hg.id} className="border-b border-border">
                  {hg.headers.map((header) => {
                    const col = columns.find((c) => c.key === header.column.id);
                    const sortable = col?.sortable;
                    const sorted = header.column.getIsSorted();
                    return (
                      <th
                        key={header.id}
                        style={{ width: `${(col?.flex ?? 1) * 100}px` }}
                        className="px-4 py-2.5 text-[length:var(--text-label)] font-medium text-text-muted"
                      >
                        <button
                          type="button"
                          disabled={!sortable}
                          onClick={header.column.getToggleSortingHandler()}
                          className={cn(
                            'inline-flex items-center gap-1',
                            sortable && 'cursor-pointer hover:text-text-secondary',
                          )}
                        >
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {sortable &&
                            (sorted === 'asc' ? (
                              <ArrowUp size={12} />
                            ) : sorted === 'desc' ? (
                              <ArrowDown size={12} />
                            ) : (
                              <ChevronsUpDown size={12} className="text-text-subtle" />
                            ))}
                        </button>
                      </th>
                    );
                  })}
                  {onDeleteRow && <th className="w-10" />}
                </tr>
              ))}
            </thead>
            <tbody>
              {table.getRowModel().rows.map((r) => {
                const row = r.original;
                const RowIcon = rowIcon?.(row);
                return (
                  <tr
                    key={r.id}
                    onClick={() => onRowClick?.(row)}
                    className={cn(
                      'group border-b border-[var(--fill)] last:border-0 transition-colors hover:bg-fill',
                      onRowClick && 'cursor-pointer',
                    )}
                  >
                    {r.getVisibleCells().map((cell, ci) => (
                      <td
                        key={cell.id}
                        className="px-4 py-3 text-[length:var(--text-body)] text-text-secondary align-middle"
                      >
                        <div className="flex items-center gap-2">
                          {ci === 0 && RowIcon && (
                            <RowIcon
                              size={14}
                              style={{ color: rowIconColor?.(row) ?? 'var(--color-accent)' }}
                            />
                          )}
                          {renderCell(row, cell.column.id)}
                        </div>
                      </td>
                    ))}
                    {onDeleteRow && (
                      <td className="px-2 py-3">
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            setPendingDelete(row);
                          }}
                          className="rounded-[var(--radius-6)] p-1.5 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-[var(--color-danger)] group-hover:opacity-100"
                          aria-label="Delete row"
                        >
                          <Trash2 size={14} />
                        </button>
                      </td>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {!error && !loading && rows.length > 0 && onPerPageChange && (
        <SearchListFooter
          total={total}
          perPage={perPage}
          currentPage={page}
          onPrev={() => onPrev?.()}
          onNext={() => onNext?.()}
          onPerPageChange={onPerPageChange}
          itemLabel={itemLabel}
        />
      )}

      <ConfirmDialog
        open={pendingDelete !== null}
        onOpenChange={(o) => !o && setPendingDelete(null)}
        title={deleteTitle}
        message={deleteMessage}
        loading={deleting}
        onConfirm={confirmDelete}
      />
    </div>
  );
}

/* Active-filter override for the toolbar chip (accent@10% bg / accent@35% border),
 * ports app_data_table.dart _ToolbarChip active state. */
const TOOLBAR_CHIP_ACTIVE =
  'border-[color-mix(in_srgb,var(--color-accent)_35%,transparent)] bg-[color-mix(in_srgb,var(--color-accent)_10%,transparent)] text-text-primary hover:bg-[color-mix(in_srgb,var(--color-accent)_10%,transparent)]';

function ColumnVisibilityButton({
  columns,
  table,
}: {
  columns: DataTableColumn[];
  table: ReturnType<typeof useReactTable<Row>>;
}) {
  const visibleCount = table.getVisibleLeafColumns().length;
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="toolbar" size="sm" className="gap-1.5 px-2.5">
          <Columns3 size={14} />
          {visibleCount}
        </Button>
      </PopoverTrigger>
      {/* Column visibility — ports app_data_table.dart column menu: a "Columns"
       * header + a labelled checkbox per column. At least one column must stay
       * visible, so the last checked box can't be unchecked. */}
      <PopoverContent align="end" className="w-56 p-2">
        <div className="px-1.5 pb-2 pt-1 text-[length:var(--text-caption)] font-medium text-text-muted">
          Columns
        </div>
        <div className="flex flex-col">
          {columns.map((c) => {
            const col = table.getColumn(c.key);
            if (!col) return null;
            const checked = col.getIsVisible();
            const isLast = checked && visibleCount <= 1;
            return (
              <label
                key={c.key}
                className={cn(
                  'flex items-center gap-2.5 rounded-[var(--radius-6)] px-1.5 py-2 text-[length:var(--text-body)] text-text-primary transition-colors',
                  isLast ? 'cursor-not-allowed opacity-60' : 'cursor-pointer hover:bg-fill',
                )}
              >
                <Checkbox
                  checked={checked}
                  disabled={isLast}
                  onCheckedChange={(v) => col.toggleVisibility(!!v)}
                />
                {c.label}
              </label>
            );
          })}
        </div>
      </PopoverContent>
    </Popover>
  );
}

function FilterButton({
  filters,
  values,
  onChange,
}: {
  filters: DataTableFilter[];
  values: Record<string, string | null>;
  onChange: (values: Record<string, string | null>) => void;
}) {
  const activeCount = Object.values(values).filter(Boolean).length;
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="toolbar"
          size="sm"
          className={cn('gap-1.5 px-2.5', activeCount > 0 && TOOLBAR_CHIP_ACTIVE)}
        >
          <FilterIcon size={14} />
          Filter
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-64 p-3">
        <div className="flex flex-col gap-3">
          {filters.map((f) => (
            <div key={f.key} className="flex flex-col gap-1.5">
              <span className="text-[length:var(--text-label)] font-medium text-text-secondary">
                {f.label}
              </span>
              <div className="flex flex-wrap gap-1.5">
                <FilterChip
                  label="All"
                  active={!values[f.key]}
                  onClick={() => onChange({ ...values, [f.key]: null })}
                />
                {f.options.map((o) => (
                  <FilterChip
                    key={o.value}
                    label={o.label}
                    active={values[f.key] === o.value}
                    onClick={() => onChange({ ...values, [f.key]: o.value })}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

function FilterChip({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'rounded-[var(--radius-6)] border px-2 py-1 text-[length:var(--text-caption)] transition-colors',
        active
          ? 'border-[var(--color-accent)] bg-fill-active text-text-primary'
          : 'border-border text-text-muted hover:text-text-secondary',
      )}
    >
      {label}
    </button>
  );
}

function ViewToggle({
  view,
  onChange,
}: {
  view: 'list' | 'grid';
  onChange: (v: 'list' | 'grid') => void;
}) {
  return (
    <div className="flex h-8 items-center overflow-hidden rounded-[var(--radius)] border border-field-border bg-field-fill">
      {(['list', 'grid'] as const).map((v, i) => (
        <div key={v} className="flex h-full">
          {i === 1 && <div className="w-px bg-field-border" />}
          <button
            type="button"
            onClick={() => onChange(v)}
            className={cn(
              'flex h-full w-8 items-center justify-center transition-colors',
              view === v
                ? 'bg-fill-active text-text-primary'
                : 'text-text-subtle hover:bg-fill-hover hover:text-text-secondary',
            )}
            aria-label={`${v} view`}
          >
            {v === 'list' ? <ListIcon size={14} /> : <LayoutGrid size={14} />}
          </button>
        </div>
      ))}
    </div>
  );
}

function TableSkeleton() {
  return (
    <div className="overflow-hidden rounded-[var(--radius-10)] border border-border">
      {Array.from({ length: 6 }).map((_, i) => (
        <div
          key={i}
          className="flex items-center gap-4 border-b border-[var(--fill)] px-4 py-3 last:border-0"
        >
          <div className="h-4 w-4 animate-pulse rounded bg-fill-active" />
          <div className="h-4 flex-1 animate-pulse rounded bg-fill-active" />
          <div className="h-4 w-24 animate-pulse rounded bg-fill-active" />
        </div>
      ))}
    </div>
  );
}
