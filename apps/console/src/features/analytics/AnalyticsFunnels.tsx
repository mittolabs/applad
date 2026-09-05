import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Filter, Loader2, Plus, Trash2 } from 'lucide-react';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { FormDialog, TextField } from '@/components/form-dialog';
import { Button } from '@/components/ui/button';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import {
  ACCENT,
  Bar,
  RED,
  SectionTitle,
  asRows,
  fmtNum,
  num,
  tint,
  useAnalyticsResource,
} from './analytics-shared';

/* AnalyticsFunnels — funnel definitions and their step-by-step conversion.
 * A funnel is a list of event names; the API counts distinct users who reached
 * each step over the window and reports the drop between consecutive steps. */

export function AnalyticsFunnels({ projectId }: { projectId?: string }) {
  const query = useAnalyticsResource('/analytics/funnels', projectId);
  const [creating, setCreating] = useState(false);
  const funnels = asRows(query.data?.funnels);

  if (query.isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="h-6 w-6 animate-spin" style={{ color: ACCENT }} />
      </div>
    );
  }
  if (query.error) {
    return (
      <div className="px-6 md:px-8">
        <ErrorState error={query.error} onRetry={query.refetch} />
      </div>
    );
  }

  return (
    <div className="overflow-y-auto px-6 py-6 md:px-8">
      <div className="flex items-center justify-between">
        <SectionTitle title={`${funnels.length} ${funnels.length === 1 ? 'funnel' : 'funnels'}`} />
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus size={14} />
          New funnel
        </Button>
      </div>

      <div className="mt-4">
        {funnels.length === 0 ? (
          <EmptyState
            icon={Filter}
            title="No funnels yet"
            subtitle="Name two or more events in order and this shows where people drop out."
            actionLabel="New funnel"
            onAction={() => setCreating(true)}
          />
        ) : (
          <div className="flex flex-col gap-3">
            {funnels.map((f) => (
              <FunnelCard
                key={String(f.$id)}
                funnel={f}
                projectId={projectId}
                onDeleted={() => query.refetch()}
              />
            ))}
          </div>
        )}
      </div>

      <CreateFunnelDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={() => query.refetch()}
      />
    </div>
  );
}

function FunnelCard({
  funnel,
  projectId,
  onDeleted,
}: {
  funnel: Record<string, unknown>;
  projectId?: string;
  onDeleted: () => void;
}) {
  const funnelId = String(funnel.$id ?? '');
  const analysis = useQuery({
    queryKey: ['analytics', 'funnel', funnelId, projectId],
    enabled: !!projectId && !!funnelId,
    queryFn: async () => {
      const res = await api.get(`/analytics/funnels/${funnelId}/analyze`);
      return res.data as Record<string, unknown>;
    },
  });

  const steps = asRows(analysis.data?.steps);
  const entry = steps.length > 0 ? num(steps[0].count) : 0;

  const remove = async () => {
    try {
      await api.delete(`/analytics/funnels/${funnelId}`);
      onDeleted();
    } catch (e) {
      toast.error(friendlyError(e));
    }
  };

  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-4">
      <div className="flex items-center gap-2">
        <span className="flex-1 truncate text-[length:var(--text-control)] font-semibold text-text-primary">
          {String(funnel.name ?? 'Funnel')}
        </span>
        {entry > 0 && steps.length > 1 && (
          <span className="text-[length:var(--text-caption)] text-text-subtle">
            {((num(steps[steps.length - 1].count) / entry) * 100).toFixed(1)}% end to end
          </span>
        )}
        <button
          type="button"
          onClick={remove}
          aria-label="Delete funnel"
          className="rounded-[var(--radius-sm)] p-1.5 text-text-subtle transition-colors hover:text-[color:var(--status-danger)]"
        >
          <Trash2 size={14} />
        </button>
      </div>

      <div className="mt-3.5">
        {analysis.isLoading ? (
          <div className="py-4 text-center text-[length:var(--text-body)] text-text-muted">
            Analysing…
          </div>
        ) : steps.length === 0 ? (
          <div className="py-4 text-center text-[length:var(--text-body)] text-text-muted">
            No one has reached these steps in the last 7 days.
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {steps.map((s, i) => {
              // Conversion is against the previous step; the drop is what the
              // reader is actually looking for, so name it rather than making
              // them subtract from 100.
              const pct = entry > 0 ? (num(s.count) / entry) * 100 : 0;
              const drop = i === 0 ? 0 : 100 - num(s.conversionRate);
              return (
                <div key={i}>
                  <div className="flex items-center gap-2">
                    <span className="flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-primary">
                      {String(s.step ?? '')}
                    </span>
                    <span className="text-[length:var(--text-label)] text-text-secondary">
                      {fmtNum(s.count ?? 0)}
                    </span>
                    {i > 0 && drop > 0.05 && (
                      <span
                        className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)] font-medium"
                        style={{ color: RED, backgroundColor: tint(RED, 10) }}
                      >
                        −{drop.toFixed(1)}%
                      </span>
                    )}
                  </div>
                  <div className="mt-1.5">
                    <Bar pct={pct} color={ACCENT} />
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

function CreateFunnelDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState('');
  const [steps, setSteps] = useState<string[]>(['', '']);
  const [saving, setSaving] = useState(false);

  const reset = () => {
    setName('');
    setSteps(['', '']);
  };

  const filled = steps.map((s) => s.trim()).filter(Boolean);

  const submit = async () => {
    setSaving(true);
    try {
      await api.post('/analytics/funnels', { name: name.trim(), steps: filled });
      onOpenChange(false);
      reset();
      onCreated();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) reset();
      }}
      title="New funnel"
      subtitle="Name the events in the order people should reach them."
      submitLabel="Create"
      loading={saving}
      submitDisabled={!name.trim() || filled.length < 2}
      onSubmit={submit}
    >
      <TextField
        label="Name"
        placeholder="Signup to first post"
        value={name}
        onChange={(e) => setName(e.target.value)}
        autoFocus
      />
      {steps.map((s, i) => (
        <div key={i} className="flex items-end gap-2">
          <div className="flex-1">
            <TextField
              label={i === 0 ? 'Steps' : undefined}
              placeholder={i === 0 ? 'account_created' : 'post_published'}
              value={s}
              onChange={(e) =>
                setSteps((prev) => prev.map((v, j) => (j === i ? e.target.value : v)))
              }
            />
          </div>
          {steps.length > 2 && (
            <button
              type="button"
              aria-label="Remove step"
              onClick={() => setSteps((prev) => prev.filter((_, j) => j !== i))}
              className="mb-1.5 rounded-[var(--radius-sm)] p-1.5 text-text-subtle transition-colors hover:text-[color:var(--status-danger)]"
            >
              <Trash2 size={14} />
            </button>
          )}
        </div>
      ))}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="self-start"
        onClick={() => setSteps((prev) => [...prev, ''])}
      >
        <Plus size={14} />
        Add step
      </Button>
    </FormDialog>
  );
}
