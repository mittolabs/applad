import { useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Layers, Plus, Trash2 } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import { Button } from '@/components/ui/button';
import { ConfirmDialog } from '@/components/form-dialog';
import { ErrorState } from '@/components/error-state';
import type { Row } from '@/components/data-table';
import { CreateEnvDialog } from './CreateEnvDialog';
import { EnvDetail } from './EnvDetail';
import { cn } from '@/lib/utils';

function envColor(slug: string): string {
  if (slug === 'production' || slug === 'prod') return '#10B981';
  if (slug === 'staging') return '#F59E0B';
  if (slug === 'development' || slug === 'dev') return '#8B5CF6';
  return '#9AA0B4';
}

export function EnvironmentsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [params, setParams] = useSearchParams();
  const selectedId = params.get('envId');

  const [creating, setCreating] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Row | null>(null);
  const [deleting, setDeleting] = useState(false);

  const envs = useQuery({
    queryKey: ['deploy-environments', projectId],
    queryFn: async () => {
      const res = await api.get('/deploy/environments');
      return (res.data as Record<string, unknown>)['environments'] as Row[] | undefined;
    },
  });

  const rows = envs.data ?? [];

  const select = (id: string | null) => {
    const next = new URLSearchParams(params);
    if (id) next.set('envId', id);
    else next.delete('envId');
    next.delete('tab');
    setParams(next, { replace: true });
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      const id = String(deleteTarget['$id']);
      await api.delete(`/deploy/environments/${id}`);
      toast.success('Environment deleted');
      if (selectedId === id) select(null);
      setDeleteTarget(null);
      envs.refetch();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center gap-4 px-6 pb-3 pt-5 md:px-8">
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Environments</h1>
        <Button className="ml-auto" size="sm" onClick={() => setCreating(true)}>
          <Plus size={16} />
          New environment
        </Button>
      </div>
      <div className="h-px bg-border" />

      <div className="flex min-h-0 flex-1">
        <div className="w-56 shrink-0 overflow-y-auto border-r border-border">
          {envs.error ? (
            <ErrorState error={envs.error} onRetry={() => envs.refetch()} />
          ) : envs.isLoading ? (
            <div className="p-4 text-[length:var(--text-body)] text-text-muted">Loading…</div>
          ) : rows.length === 0 ? (
            <div className="p-6 text-center text-[length:var(--text-body)] text-text-secondary">
              No environments yet
            </div>
          ) : (
            <ul className="py-2">
              {rows.map((env) => {
                const id = String(env['$id']);
                const name = String(env['name'] ?? env['slug'] ?? id);
                const slug = String(env['slug'] ?? '');
                const isDefault = env['isDefault'] === true;
                const selected = selectedId === id;
                return (
                  <li key={id}>
                    <button
                      type="button"
                      onClick={() => select(id)}
                      className={cn(
                        'group flex w-full items-center gap-2.5 px-4 py-2 text-left transition-colors',
                        selected ? 'bg-fill-active' : 'hover:bg-fill',
                      )}
                    >
                      <span
                        className="mt-0.5 h-2 w-2 shrink-0 rounded-full"
                        style={{ backgroundColor: envColor(slug) }}
                      />
                      <span className="flex min-w-0 flex-col">
                        <span
                          className={cn(
                            'truncate text-[length:var(--text-body)]',
                            selected
                              ? 'font-medium text-text-primary'
                              : 'text-text-secondary',
                          )}
                        >
                          {name}
                        </span>
                        {isDefault && (
                          <span className="text-[length:var(--text-caption)] text-text-subtle">
                            default
                          </span>
                        )}
                      </span>
                      <span
                        role="button"
                        tabIndex={0}
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteTarget(env);
                        }}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.stopPropagation();
                            setDeleteTarget(env);
                          }
                        }}
                        className="ml-auto rounded-[var(--radius-6)] p-1 text-text-subtle opacity-0 transition-all hover:text-[var(--color-danger)] group-hover:opacity-100"
                        aria-label="Delete environment"
                      >
                        <Trash2 size={14} />
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        <div className="min-w-0 flex-1 overflow-y-auto">
          {selectedId ? (
            <EnvDetail
              key={selectedId}
              envId={selectedId}
              onListChanged={() => envs.refetch()}
            />
          ) : (
            <EnvEmptyState />
          )}
        </div>
      </div>

      <CreateEnvDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={(id) => {
          envs.refetch();
          select(id);
        }}
      />

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        title="Delete environment"
        message="This will permanently remove the environment and its variables."
        loading={deleting}
        onConfirm={confirmDelete}
      />
    </div>
  );
}

function EnvEmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center text-center">
      <div className="flex h-16 w-16 items-center justify-center rounded-[var(--radius-12)] border border-border bg-surface text-text-secondary">
        <Layers size={32} />
      </div>
      <div className="mt-4 text-[length:var(--text-subhead)] font-medium text-text-primary">
        Select an environment
      </div>
      <div className="mt-1.5 text-[length:var(--text-body)] text-text-secondary">
        Choose from the left or create a new one.
      </div>
    </div>
  );
}
