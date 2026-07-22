import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, Plus, X } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { FormDialog, ConfirmDialog, TextField, FormField } from '@/components/form-dialog';
import { ChipGroup, str, type Json } from './shared';

type RelType = 'oneToOne' | 'oneToMany' | 'manyToOne' | 'manyToMany';

const REL_TYPES: { value: RelType; label: string }[] = [
  { value: 'oneToOne', label: 'oneToOne' },
  { value: 'oneToMany', label: 'oneToMany' },
  { value: 'manyToOne', label: 'manyToOne' },
  { value: 'manyToMany', label: 'manyToMany' },
];

export function RelationshipsPanel({ dbId, tableId }: { dbId: string; tableId: string }) {
  const qc = useQueryClient();
  const base = `/databases/${dbId}/tables/${tableId}`;
  const [creating, setCreating] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [key, setKey] = useState('');
  const [relatedTableId, setRelatedTableId] = useState('');
  const [type, setType] = useState<RelType>('oneToMany');
  const [saving, setSaving] = useState(false);

  const query = useQuery({
    queryKey: ['db-rels', dbId, tableId],
    queryFn: async () => {
      const res = await api.get(`${base}/relationships`);
      return (res.data as { relationships?: Json[] }).relationships ?? [];
    },
  });
  const rels = query.data ?? [];
  const invalidate = () => qc.invalidateQueries({ queryKey: ['db-rels', dbId, tableId] });

  const create = async () => {
    if (!key.trim() || !relatedTableId.trim()) return;
    setSaving(true);
    try {
      await api.post(`${base}/columns/relationship`, {
        key: key.trim(),
        relatedTableId: relatedTableId.trim(),
        type,
      });
      setCreating(false);
      setKey('');
      setRelatedTableId('');
      setType('oneToMany');
      invalidate();
    } finally {
      setSaving(false);
    }
  };

  const del = async () => {
    if (!deleteTarget) return;
    await api.delete(`${base}/relationships/${deleteTarget}`);
    setDeleteTarget(null);
    invalidate();
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-end">
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus size={14} />
          Create relationship
        </Button>
      </div>

      {query.error ? (
        <ErrorState error={query.error} onRetry={query.refetch} />
      ) : query.isLoading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-subtle">Loading…</div>
      ) : rels.length === 0 ? (
        <EmptyState
          icon={Link}
          title="No relationships"
          subtitle="Define relationships between tables."
          actionLabel="Create relationship"
          onAction={() => setCreating(true)}
        />
      ) : (
        <div className="overflow-x-auto rounded-[var(--radius-10)] border border-border">
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-border text-[length:var(--text-label)] font-medium text-text-muted">
                <th className="px-4 py-2.5">Key</th>
                <th className="px-4 py-2.5">Type</th>
                <th className="px-4 py-2.5">Related Table</th>
                <th className="px-4 py-2.5">On Delete</th>
                <th className="w-10" />
              </tr>
            </thead>
            <tbody>
              {rels.map((r) => {
                const k = str(r['key']);
                const id = str(r['$id']) || k;
                return (
                  <tr key={id} className="group border-b border-[var(--fill)] last:border-0 hover:bg-fill">
                    <td className="px-4 py-3 text-[length:var(--text-body)] text-text-primary">{k}</td>
                    <td className="px-4 py-3 text-[length:var(--text-body)] text-text-muted">{str(r['type'])}</td>
                    <td className="px-4 py-3 text-[length:var(--text-body)] text-text-muted">
                      {str(r['relatedTableId'])}
                    </td>
                    <td className="px-4 py-3 text-[length:var(--text-body)] text-text-muted">
                      {str(r['onDelete']) || 'setNull'}
                    </td>
                    <td className="px-2 py-3">
                      <button
                        type="button"
                        onClick={() => setDeleteTarget(id)}
                        className="rounded-[var(--radius-6)] p-1.5 text-text-subtle opacity-0 transition-all hover:bg-fill hover:text-[var(--color-danger)] group-hover:opacity-100"
                        aria-label="Delete relationship"
                      >
                        <X size={14} />
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
        title="Create relationship"
        submitLabel="Create"
        loading={saving}
        submitDisabled={!key.trim() || !relatedTableId.trim()}
        onSubmit={create}
      >
        <TextField label="Key" value={key} onChange={(e) => setKey(e.target.value)} placeholder="relationship_name" autoFocus />
        <TextField
          label="Related table ID"
          value={relatedTableId}
          onChange={(e) => setRelatedTableId(e.target.value)}
          placeholder="table_id"
        />
        <FormField label="Type">
          <ChipGroup options={REL_TYPES} value={type} onChange={setType} />
        </FormField>
      </FormDialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        title="Delete relationship"
        message="Are you sure? This action cannot be undone."
        onConfirm={del}
      />
    </div>
  );
}
