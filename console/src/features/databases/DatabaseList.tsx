import { useState } from 'react';
import { Database, Rows3, Table2 } from 'lucide-react';
import { api } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { DataTable, type DataTableColumn } from '@/components/data-table';
import { FormDialog, TextField } from '@/components/form-dialog';
import { fmtDate, str, type Json } from './shared';

const COLUMNS: DataTableColumn[] = [
  { key: '$id', label: 'Database ID', flex: 3 },
  { key: 'name', label: 'Name', flex: 3, sortable: true },
  { key: 'created', label: 'Created', flex: 2 },
];

/* The databases list only — the page header + Databases/Usage tabs live in
 * DatabasesPage, so this renders just the table (no duplicate h1/tabs). */
export function DatabaseList({
  projectId,
  onSelectDb,
}: {
  projectId?: string;
  onSelectDb: (id: string, name: string) => void;
}) {
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [saving, setSaving] = useState(false);

  const list = useResourceList<Json>({
    endpoint: '/databases',
    itemsKey: 'databases',
    scope: [projectId],
  });

  const create = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      await api.post('/databases', { name: name.trim() });
      setCreating(false);
      setName('');
      list.refetch();
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <DataTable
        columns={COLUMNS}
        rows={list.rows}
        getCellValue={(row, key) =>
          key === 'created' ? fmtDate(row['createdAt'] ?? row['$createdAt']) : str(row[key])
        }
        rowIcon={() => Database}
        onRowClick={(row) => onSelectDb(str(row['$id']), str(row['name']) || str(row['$id']))}
        onDeleteRow={async (row) => {
          await api.delete(`/databases/${str(row['$id'])}`);
          list.refetch();
        }}
        requireTypedConfirm
        deleteMessage="This cannot be undone. Every table in this database and all their rows are permanently deleted."
        createLabel="Create database"
        onCreate={() => setCreating(true)}
        searchHint="Search by name or ID"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="Databases"
        emptyIcon={Database}
        emptyTitle="No databases"
        emptySubtitle="Create a database to get started"
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      <FormDialog
        open={creating}
        onOpenChange={setCreating}
        title="Create database"
        submitLabel="Create"
        loading={saving}
        submitDisabled={!name.trim()}
        onSubmit={create}
      >
        <TextField
          label="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Enter database name"
          autoFocus
        />
      </FormDialog>
    </>
  );
}

/* Usage tab content (stat cards + chart placeholder) — rendered by DatabasesPage
 * under its own Usage tab. */
export function DatabaseUsageTab() {
  const cards = [
    { label: 'Total databases', icon: Database },
    { label: 'Total tables', icon: Table2 },
    { label: 'Total rows', icon: Rows3 },
  ] as const;
  return (
    <div className="flex flex-col gap-6">
      <p className="text-[length:var(--text-body)] text-text-secondary">
        Database activity for the past 30 days.
      </p>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {cards.map(({ label, icon: Icon }) => (
          <div key={label} className="rounded-[var(--radius-10)] border border-border bg-surface p-5">
            <Icon size={16} className="text-text-secondary" />
            <div className="mt-3 text-[length:var(--text-h2)] font-bold text-text-primary">—</div>
            <div className="mt-1 text-[length:var(--text-label)] text-text-secondary">{label}</div>
          </div>
        ))}
      </div>
      <div className="flex h-48 items-center justify-center rounded-[var(--radius-10)] border border-border bg-surface text-[length:var(--text-body)] text-text-subtle">
        Usage charts coming soon
      </div>
    </div>
  );
}
