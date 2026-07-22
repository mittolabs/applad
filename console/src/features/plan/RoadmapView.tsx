import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CalendarDays, Plus, Trash2, Flag } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { FormDialog, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { cn } from '@/lib/utils';

/*
 * The roadmap.
 *
 * Every roadmap tool asks somebody to type a percentage, which is a number
 * about a feeling. This one counts: how many of a milestone's items are done,
 * and how many of the constraints those items were agreed against have become
 * behaviour anything can check. The second is the one that predicts trouble —
 * work can look finished while half of what was agreed was never expressed as
 * a rule, and nothing else in this class of tool can see that.
 *
 * A milestone with no items shows no progress rather than 0%, because those
 * are different states and only one of them means "nothing has happened".
 */

interface Progress {
  total: number;
  done: number;
  inProgress: number;
  blocked: number;
  criteria: number;
  specified: number;
}

interface Milestone {
  $id: string;
  name: string;
  description: string;
  targetDate?: string;
  completedAt?: string;
  progress: Progress;
}

/** Days from today, negative when the date has passed. */
function daysUntil(date: string): number {
  const target = new Date(date);
  const today = new Date();
  target.setHours(0, 0, 0, 0);
  today.setHours(0, 0, 0, 0);
  return Math.round((target.getTime() - today.getTime()) / 86_400_000);
}

/** How a date reads to a person, rather than as a timestamp. */
function whenLabel(date?: string, completed?: string): { text: string; tone: string } {
  if (completed) return { text: 'Completed', tone: '#22C55E' };
  if (!date) return { text: 'No target date', tone: 'var(--text-subtle)' };

  const days = daysUntil(date);
  if (days < 0) return { text: `${-days}d overdue`, tone: '#EF4444' };
  if (days === 0) return { text: 'Due today', tone: '#F59E0B' };
  if (days <= 7) return { text: `in ${days}d`, tone: '#F59E0B' };
  return { text: `in ${Math.round(days / 7)}w`, tone: 'var(--text-muted)' };
}

export function RoadmapView() {
  const qc = useQueryClient();
  const [creating, setCreating] = useState(false);

  const query = useQuery({
    queryKey: ['plan-milestones'],
    queryFn: async () =>
      ((await api.get('/plan/milestones')).data as { milestones: Milestone[] }).milestones ?? [],
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.delete(`/plan/milestones/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['plan-milestones'] }),
    onError: (e) => toast.error(friendlyError(e)),
  });

  const milestones = query.data ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <span className="text-[length:var(--text-control)] font-medium text-text-primary">
          Milestones
        </span>
        <Button size="sm" variant="outline" onClick={() => setCreating(true)}>
          <Plus size={14} />
          New milestone
        </Button>
      </div>

      {query.isLoading ? (
        <div className="py-10 text-center text-[length:var(--text-body)] text-text-muted">
          Loading…
        </div>
      ) : milestones.length === 0 ? (
        <div className="flex flex-col items-center gap-2 rounded-[var(--radius-10)] border border-border bg-surface py-12 text-center">
          <Flag size={20} className="text-text-subtle" />
          <div className="text-[length:var(--text-body)] text-text-primary">No milestones yet</div>
          <div className="max-w-[380px] text-[length:var(--text-caption)] text-text-subtle">
            A milestone is a date something is aimed at. Items belong to one, and its progress is
            counted from them rather than typed.
          </div>
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {milestones.map((m) => (
            <MilestoneRow key={m.$id} milestone={m} onDelete={() => remove.mutate(m.$id)} />
          ))}
        </div>
      )}

      <CreateMilestoneDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={() => qc.invalidateQueries({ queryKey: ['plan-milestones'] })}
      />
    </div>
  );
}

function MilestoneRow({
  milestone,
  onDelete,
}: {
  milestone: Milestone;
  onDelete: () => void;
}) {
  const p = milestone.progress;
  const when = whenLabel(milestone.targetDate, milestone.completedAt);
  const hasItems = p.total > 0;
  const donePct = hasItems ? Math.round((p.done / p.total) * 100) : 0;
  const activePct = hasItems ? Math.round((p.inProgress / p.total) * 100) : 0;

  // Agreed but never expressed as behaviour. This is the number that predicts
  // trouble, so it is stated rather than folded into a single percentage.
  const unspecified = p.criteria - p.specified;

  return (
    <div className="group flex flex-col gap-3 rounded-[var(--radius-10)] border border-border bg-surface px-4 py-3.5">
      <div className="flex items-center gap-3">
        <span className="flex-1 truncate text-[length:var(--text-body)] font-medium text-text-primary">
          {milestone.name}
        </span>

        {milestone.targetDate && (
          <span className="flex items-center gap-1.5 text-[length:var(--text-caption)] text-text-muted">
            <CalendarDays size={12} />
            {new Date(milestone.targetDate).toLocaleDateString(undefined, {
              day: 'numeric',
              month: 'short',
              year: 'numeric',
            })}
          </span>
        )}

        <span
          className="text-[length:var(--text-caption)] font-medium"
          style={{ color: when.tone }}
        >
          {when.text}
        </span>

        <button
          onClick={onDelete}
          className="opacity-0 transition-opacity group-hover:opacity-100"
          aria-label="Delete milestone"
        >
          <Trash2 size={13} className="text-text-subtle hover:text-[#EF4444]" />
        </button>
      </div>

      {/* The bar is the count, not an estimate. An empty milestone shows no
          bar at all rather than a 0% one — "nothing planned" and "nothing
          done" are different things to be told. */}
      {hasItems ? (
        <>
          <div className="flex h-1.5 overflow-hidden rounded-full bg-fill">
            <div style={{ width: `${donePct}%`, backgroundColor: '#22C55E' }} />
            <div style={{ width: `${activePct}%`, backgroundColor: '#3B82F6' }} />
          </div>

          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[length:var(--text-caption)]">
            <span className="text-text-secondary">
              {p.done} of {p.total} done
            </span>
            {p.inProgress > 0 && (
              <span style={{ color: '#3B82F6' }}>{p.inProgress} in progress</span>
            )}
            {p.blocked > 0 && <span style={{ color: '#F59E0B' }}>{p.blocked} blocked</span>}

            {p.criteria > 0 && (
              <span
                className={cn('ml-auto', unspecified > 0 ? '' : 'text-text-subtle')}
                style={unspecified > 0 ? { color: '#F59E0B' } : undefined}
                title="Acceptance criteria that have become rules in a specification"
              >
                {unspecified > 0
                  ? `${unspecified} of ${p.criteria} criteria still unspecified`
                  : `all ${p.criteria} criteria specified`}
              </span>
            )}
          </div>
        </>
      ) : (
        <span className="text-[length:var(--text-caption)] text-text-subtle">
          No items in this milestone yet
        </span>
      )}
    </div>
  );
}

function CreateMilestoneDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState('');
  const [targetDate, setTargetDate] = useState('');
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    setSaving(true);
    try {
      await api.post('/plan/milestones', { name: name.trim(), targetDate });
      setName('');
      setTargetDate('');
      onOpenChange(false);
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
      onOpenChange={onOpenChange}
      title="New milestone"
      subtitle="A date something is aimed at"
      submitLabel="Create"
      loading={saving}
      submitDisabled={!name.trim()}
      onSubmit={submit}
    >
      <TextField
        label="Name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="e.g. Checkout revamp"
        autoFocus
      />
      <TextField
        label="Target date"
        type="date"
        value={targetDate}
        onChange={(e) => setTargetDate(e.target.value)}
        hint="Optional. A milestone without one still groups work, it just has no place on the roadmap."
      />
    </FormDialog>
  );
}
