import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Table2 } from 'lucide-react';
import { api } from '@/api/client';
import { DataTable, type DataTableColumn } from '@/components/data-table';
import { FormDialog, TextField } from '@/components/form-dialog';
import { fmtDate, localFilter, str, type Json } from './shared';

const COLUMNS: DataTableColumn[] = [
  { key: '$id', label: 'Table ID', flex: 3 },
  { key: 'name', label: 'Name', flex: 3, sortable: true },
  { key: 'created', label: 'Created', flex: 2 },
];

export function TablesPanel({
  dbId,
  onSelectTable,
}: {
  dbId: string;
  onSelectTable: (id: string, name: string) => void;
}) {
  const qc = useQueryClient();
  const [search, setSearch] = useState('');
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [saving, setSaving] = useState(false);

  const query = useQuery({
    queryKey: ['db-tables', dbId],
    queryFn: async () => {
      const res = await api.get(`/databases/${dbId}/tables`);
      return (res.data as { tables?: Json[] }).tables ?? [];
    },
  });

  const rows = localFilter(query.data ?? [], search);

  const create = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      await api.post(`/databases/${dbId}/tables`, { name: name.trim() });
      setCreating(false);
      setName('');
      qc.invalidateQueries({ queryKey: ['db-tables', dbId] });
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <DataTable
        columns={COLUMNS}
        rows={rows}
        getCellValue={(row, key) =>
          key === 'created' ? fmtDate(row['createdAt'] ?? row['$createdAt']) : str(row[key])
        }
        rowIcon={() => Table2}
        onRowClick={(row) => onSelectTable(str(row['$id']), str(row['name']) || str(row['$id']))}
        onDeleteRow={async (row) => {
          await api.delete(`/databases/${dbId}/tables/${str(row['$id'])}`);
          qc.invalidateQueries({ queryKey: ['db-tables', dbId] });
        }}
        requireTypedConfirm
        deleteMessage="This cannot be undone. The table and every row in it are permanently deleted."
        createLabel="Create table"
        onCreate={() => setCreating(true)}
        searchHint="Search by name or ID"
        searchValue={search}
        onSearchChange={setSearch}
        emptyIcon={Table2}
        emptyTitle="No tables yet"
        emptySubtitle="Create, organize, and query structured data with Tables."
        loading={query.isLoading}
        error={query.error}
        onRetry={query.refetch}
      />

      <FormDialog
        open={creating}
        onOpenChange={setCreating}
        title="Create table"
        submitLabel="Create"
        loading={saving}
        submitDisabled={!name.trim()}
        onSubmit={create}
      >
        <TextField
          label="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Enter table name"
          autoFocus
        />
      </FormDialog>
    </>
  );
}
