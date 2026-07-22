import { type ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';
import { api } from '@/api/client';
import type { useResourceList } from '@/hooks/use-resource-list';
import {
  DataTable,
  type DataTableColumn,
  type Row,
} from '@/components/data-table';

/* Shared list surface for deploy targets (Sites + Containers). A thin
 * composition over DataTable + useResourceList so both features share the
 * same search/pagination/delete plumbing against /deploy/targets. */

export function TargetList({
  list,
  columns,
  getCellValue,
  cellRender,
  rowIcon,
  onRowClick,
  onDeleted,
  createLabel,
  onCreate,
  itemLabel,
  searchHint,
  emptyIcon,
  emptyTitle,
  emptySubtitle,
}: {
  list: ReturnType<typeof useResourceList<Row>>;
  columns: DataTableColumn[];
  getCellValue: (row: Row, key: string) => string;
  cellRender?: (row: Row, key: string) => ReactNode | undefined;
  rowIcon: (row: Row) => LucideIcon;
  onRowClick: (row: Row) => void;
  onDeleted: () => void;
  createLabel: string;
  onCreate: () => void;
  itemLabel: string;
  searchHint: string;
  emptyIcon: LucideIcon;
  emptyTitle: string;
  emptySubtitle: string;
}) {
  return (
    <DataTable
      columns={columns}
      rows={list.rows}
      getCellValue={getCellValue}
      cellRender={cellRender}
      rowIcon={rowIcon}
      onRowClick={onRowClick}
      onDeleteRow={async (row) => {
        await api.delete(`/deploy/targets/${String(row['$id'] ?? row['id'] ?? '')}`);
        onDeleted();
      }}
      // Deleting one of these stops something the public is reaching, so the
      // name has to be typed.
      requireTypedConfirm
      deleteMessage="This cannot be undone. The site stops being served immediately, and its deployments and history are removed."
      createLabel={createLabel}
      onCreate={onCreate}
      searchHint={searchHint}
      searchValue={list.search}
      onSearchChange={list.setSearch}
      onSearch={list.runSearch}
      total={list.total}
      perPage={list.perPage}
      page={list.page}
      onPerPageChange={list.setPerPage}
      onPrev={list.prevPage}
      onNext={list.nextPage}
      itemLabel={itemLabel}
      emptyIcon={emptyIcon}
      emptyTitle={emptyTitle}
      emptySubtitle={emptySubtitle}
      loading={list.isLoading}
      error={list.error}
      onRetry={list.refetch}
    />
  );
}
