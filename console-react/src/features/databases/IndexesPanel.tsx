import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ListOrdered, Plus } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { FormDialog, ConfirmDialog, TextField, FormField } from '@/components/form-dialog';
import { ChipGroup, formatIndexType, str, type Json } from './shared';

type IndexType = 'btree' | 'unique' | 'fulltext';

const INDEX_TYPES: { value: IndexType; label: string }[] = [
  { value: 'btree', label: 'B-tree' },
  { value: 'unique', label: 'Unique' },
  { value: 'fulltext', label: 'GIN full-text' },
];

export function IndexesPanel({ dbId, tableId }: { dbId: string; tableId: string }) {
  const qc = useQueryClient();
  const base = `/databases/${dbId}/tables/${tableId}`;
  const [creating, setCreating] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [key, setKey] = useState('');
  const [columns, setColumns] = useState('');
  const [type, setType] = useState<IndexType>('btree');
  const [saving, setSaving] = useState(false);

  const query = useQuery({
    queryKey: ['db-indexes', dbId, tableId],
    queryFn: async () => {
      const res = await api.get(`${base}/indexes`);
      return (res.data as { indexes?: Json[] }).indexes ?? [];
    },
  });
  const indexes = query.data ?? [];
  const invalidate = () => qc.invalidateQueries({ queryKey: ['db-indexes', dbId, tableId] });

  const create = async () => {
    if (!key.trim()) return;
    setSaving(true);
    try {
      const cols = columns
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      await api.post(`${base}/indexes`, {
        key: key.trim(),
        type,
        columns: cols,
        orders: cols.map(() => 'ASC'),
      });
      setCreating(false);
      setKey('');
      setColumns('');
      setType('btree');
      invalidate();
    } finally {
      setSaving(false);
    }
  };

  const del = async () => {
    if (!deleteTarget) return;
    await api.delete(`${base}/indexes/${deleteTarget}`);
    setDeleteTarget(null);
    invalidate();
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-end">
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus size={14} />
          Create index
        </Button>
      </div>

      {query.error ? (
        <ErrorState error={query.error} onRetry={query.refetch} />
      ) : query.isLoading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-subtle">Loading…</div>
      ) : indexes.length === 0 ? (
        <EmptyState
          icon={ListOrdered}
          title="No indexes"
          subtitle="Create indexes to optimize query performance."
          actionLabel="Create index"
          onAction={() => setCreating(true)}
        />
      ) : (
        <div className="overflow-x-auto rounded-[var(--radius-10)] border border-border">
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-border text-[length:var(--text-label)] font-medium text-text-muted">
                <th className="px-4 py-2.5">Key</th>
                <th className="px-4 py-2.5">Type</th>
                <th className="px-4 py-2.5">Columns</th>
                <th className="px-4 py-2.5">Orders</th>
                <th className="w-10" />
              </tr>
            </thead>
            <tbody>
              {indexes.map((idx) => {
                const k = str(idx['key']);
                const cols =
                  (idx['columns'] as unknown[] | undefined) ??
                  (idx['attributes'] as unknown[] | undefined) ??
                  [];
                const orders = (idx['orders'] as unknown[] | undefined) ?? [];
                return (
                  <tr key={k} className="group border-b border-[var(--fill)] last:border-0 hover:bg-fill">
                    <td className="px-4 py-3 text-[length:var(--text-body)] text-text-primary">{k}</td>
                    <td className="px-4 py-3 text-[length:var(--text-body)] text-text-muted">
                      {formatIndexType(str(idx['type']))}
                    </td>
                    <td className="px-4 py-3 text-[length:var(--text-label)] text-text-muted">
                      {cols.join(', ')}
                    </td>
                    <td className="px-4 py-3 text-[length:var(--text-label)] text-text-muted">
                      {orders.join(', ')}
                    </td>
                    <td className="px-2 py-3">
                      <button
                        type="button"
                        onClick={() => setDeleteTarget(k)}
                        className="rounded-[var(--radius-6)] p-1.5 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-[var(--color-danger)] group-hover:opacity-100"
                        aria-label="Delete index"
                      >
                        <Plus size={14} className="rotate-45" />
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <FormDialog
        open={creating}
        onOpenChange={setCreating}
        title="Create index"
        submitLabel="Create"
        loading={saving}
        submitDisabled={!key.trim()}
        onSubmit={create}
      >
        <TextField label="Key" value={key} onChange={(e) => setKey(e.target.value)} placeholder="index_name" autoFocus />
        <TextField
          label="Columns (comma separated)"
          value={columns}
          onChange={(e) => setColumns(e.target.value)}
          placeholder="column1, column2"
        />
        <FormField label="Type">
          <ChipGroup options={INDEX_TYPES} value={type} onChange={setType} />
        </FormField>
      </FormDialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        title="Delete index"
        message="Are you sure? This action cannot be undone."
        onConfirm={del}
      />
    </div>
  );
}
