import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { AlignJustify, Columns3, KeyRound, MoreHorizontal, Plus, Upload } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import { IdText } from '@/components/id-text';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { SearchListHeader } from '@/components/search-list';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { FormDialog, ConfirmDialog, TextField, TextAreaField } from '@/components/form-dialog';
import { columnTypeIcon, localFilter, str, type Json } from './shared';

export function RowsPanel({
  dbId,
  tableId,
  onCreateColumn,
}: {
  dbId: string;
  tableId: string;
  onCreateColumn: () => void;
}) {
  const qc = useQueryClient();
  const base = `/databases/${dbId}/tables/${tableId}`;
  const [search, setSearch] = useState('');
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<Json | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const columnsQuery = useQuery({
    queryKey: ['db-columns', dbId, tableId],
    queryFn: async () => {
      const res = await api.get(`${base}/columns`);
      return (res.data as { columns?: Json[] }).columns ?? [];
    },
  });
  const columns = columnsQuery.data ?? [];

  const rowsQuery = useQuery({
    queryKey: ['db-rows', dbId, tableId],
    queryFn: async () => {
      const res = await api.get(`${base}/rows`, { params: { limit: 100 } });
      const data = res.data as { documents?: Json[]; rows?: Json[] };
      return data.documents ?? data.rows ?? [];
    },
  });

  const invalidateRows = () => qc.invalidateQueries({ queryKey: ['db-rows', dbId, tableId] });

  const rows = localFilter(rowsQuery.data ?? [], search);
  const colKeys = columns.map((c) => str(c['key'])).filter(Boolean);
  const displayKeys = ['$id', ...colKeys];

  const del = async () => {
    if (!deleteTarget) return;
    await api.delete(`${base}/rows/${deleteTarget}`);
    setDeleteTarget(null);
    invalidateRows();
  };

  const loading = rowsQuery.isLoading || columnsQuery.isLoading;
  const error = rowsQuery.error ?? columnsQuery.error;

  return (
    <div className="flex flex-col gap-4">
      <SearchListHeader
        searchHint="Search rows"
        value={search}
        onChange={setSearch}
        trailing={
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" disabled>
              <Upload size={14} />
              Import CSV
            </Button>
            <Button size="sm" onClick={() => setCreating(true)}>
              <Plus size={14} />
              Create row
            </Button>
          </div>
        }
      />

      {error ? (
        <ErrorState error={error} onRetry={rowsQuery.refetch} />
      ) : loading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-subtle">
          Loading…
        </div>
      ) : rows.length === 0 && columns.length === 0 ? (
        <EmptyState
          icon={Columns3}
          title="You have no columns yet"
          subtitle="Create columns to define your data schema."
          actionLabel="Create column"
          onAction={onCreateColumn}
        />
      ) : rows.length === 0 ? (
        <EmptyState
          icon={AlignJustify}
          title="No rows yet"
          subtitle="Create a row or import data from CSV."
          actionLabel="Create row"
          onAction={() => setCreating(true)}
        />
      ) : (
        <div className="overflow-x-auto rounded-[var(--radius-10)] border border-border">
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-border">
                {displayKeys.map((k) => (
                  <th
                    key={k}
                    className="px-4 py-2.5 text-[length:var(--text-label)] font-medium text-text-muted"
                  >
                    <span className="inline-flex items-center gap-1.5">
                      {k === '$id' ? (
                        <KeyRound size={12} className="text-text-subtle" />
                      ) : (
                        <ColumnIcon columns={columns} colKey={k} />
                      )}
                      {k}
                    </span>
                  </th>
                ))}
                <th className="w-10" />
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => {
                const rowId = str(row['$id']);
                const data = (row['data'] as Json | undefined) ?? row;
                return (
                  <tr key={rowId} className="group border-b border-[var(--fill)] last:border-0 hover:bg-fill">
                    {displayKeys.map((k) => (
                      <td
                        key={k}
                        className="max-w-[280px] truncate px-4 py-3 font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-primary"
                      >
                        {k === '$id' ? (
                          <IdText id={rowId} fontSize={12} />
                        ) : (
                          str(data[k]) || '-'
                        )}
                      </td>
                    ))}
                    <td className="px-2 py-3">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <button
                            type="button"
                            className="rounded-[var(--radius-6)] p-1.5 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-text-primary group-hover:opacity-100"
                            aria-label="Row actions"
                          >
                            <MoreHorizontal size={14} />
                          </button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onSelect={() => setEditing(row)}>Edit</DropdownMenuItem>
                          <DropdownMenuItem destructive onSelect={() => setDeleteTarget(rowId)}>
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {creating && (
        <RowDialog
          mode="create"
          base={base}
          columns={columns}
          onClose={() => setCreating(false)}
          onSaved={() => {
            setCreating(false);
            invalidateRows();
          }}
        />
      )}
      {editing && (
        <RowDialog
          mode="edit"
          base={base}
          columns={columns}
          row={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            invalidateRows();
          }}
        />
      )}

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        title="Delete row"
        message="Are you sure? This action cannot be undone."
        onConfirm={del}
      />
    </div>
  );
}

function ColumnIcon({ columns, colKey }: { columns: Json[]; colKey: string }) {
  const col = columns.find((c) => str(c['key']) === colKey);
  const Icon = columnTypeIcon(str(col?.['type']));
  return <Icon size={12} className="text-[var(--color-accent)]" />;
}

function RowDialog({
  mode,
  base,
  columns,
  row,
  onClose,
  onSaved,
}: {
  mode: 'create' | 'edit';
  base: string;
  columns: Json[];
  row?: Json;
  onClose: () => void;
  onSaved: () => void;
}) {
  const data = (row?.['data'] as Json | undefined) ?? row ?? {};
  const rowId = str(row?.['$id']);
  const colKeys = columns.map((c) => str(c['key'])).filter(Boolean);

  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(colKeys.map((k) => [k, str(data[k])])),
  );
  const [json, setJson] = useState(() =>
    colKeys.length === 0 ? JSON.stringify(data, null, 2) : '{}',
  );
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    setSaving(true);
    try {
      let payload: Json;
      if (colKeys.length === 0) {
        payload = json.trim() ? (JSON.parse(json) as Json) : {};
      } else {
        payload = {};
        for (const k of colKeys) {
          const v = values[k]?.trim() ?? '';
          if (mode === 'edit' || v) payload[k] = v;
        }
      }
      if (mode === 'create') {
        await api.post(`${base}/rows`, { data: payload });
      } else {
        await api.patch(`${base}/rows/${rowId}`, { data: payload });
      }
      onSaved();
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={mode === 'create' ? 'Create row' : 'Edit row'}
      submitLabel={mode === 'create' ? 'Create' : 'Save'}
      loading={saving}
      width={520}
      onSubmit={submit}
    >
      {colKeys.length === 0 ? (
        <TextAreaField
          label="Data (JSON)"
          value={json}
          onChange={(e) => setJson(e.target.value)}
          rows={6}
          placeholder='{"key": "value"}'
        />
      ) : (
        colKeys.map((k) => (
          <TextField
            key={k}
            label={k}
            value={values[k] ?? ''}
            onChange={(e) => setValues((prev) => ({ ...prev, [k]: e.target.value }))}
            placeholder={`Enter ${k}`}
          />
        ))
      )}
    </FormDialog>
  );
}
