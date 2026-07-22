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

/*
 * The window the roadmap draws: from today (or the earliest target, if
 * something is already overdue) to a month past the furthest one. Everything
 * is positioned in this span, which is what makes it a roadmap rather than a
 * list with dates written on it.
 */
function timeSpan(milestones: Milestone[]): { start: number; end: number; months: Date[] } {
  const dates = milestones
    .map((m) => (m.targetDate ? new Date(m.targetDate).getTime() : 0))
    .filter(Boolean);

  const now = Date.now();
  const start = Math.min(now, ...(dates.length ? dates : [now]));
  const furthest = Math.max(now, ...(dates.length ? dates : [now]));
  // A month of headroom, so the last milestone is not flush against the edge.
  const end = furthest + 30 * 86_400_000;

  const months: Date[] = [];
  const cursor = new Date(start);
  cursor.setDate(1);
  while (cursor.getTime() <= end) {
    months.push(new Date(cursor));
    cursor.setMonth(cursor.getMonth() + 1);
  }
  return { start, end, months };
}

/** Where a moment sits across the span, as a percentage. */
function positionOf(time: number, start: number, end: number): number {
  if (end <= start) return 0;
  return Math.min(100, Math.max(0, ((time - start) / (end - start)) * 100));
}

export function RoadmapView({ onOpenMilestone }: { onOpenMilestone: (id: string) => void }) {
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
  const dated = milestones.filter((m) => m.targetDate);
  const span = timeSpan(milestones);
  const todayAt = positionOf(Date.now(), span.start, span.end);

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
        <div className="flex flex-col gap-4">
          {dated.length > 0 && (
            <div className="relative overflow-hidden rounded-[var(--radius-10)] border border-border bg-surface p-4 pt-3">
              {/* The axis. Months across the top, today marked, and each
                  milestone drawn where its date actually falls. */}
              <div className="relative mb-2 h-5">
                {span.months.map((month) => (
                  <span
                    key={month.toISOString()}
                    className="absolute text-[length:var(--text-caption)] text-text-subtle"
                    style={{ left: `${positionOf(month.getTime(), span.start, span.end)}%` }}
                  >
                    {month.toLocaleDateString(undefined, { month: 'short' })}
                  </span>
                ))}
              </div>

              <div className="relative flex flex-col gap-2 pb-1">
                <div
                  className="pointer-events-none absolute bottom-0 top-0 w-px"
                  style={{ left: `${todayAt}%`, backgroundColor: 'var(--color-accent)' }}
                  title="Today"
                />
                {dated.map((m) => {
                  const at = positionOf(new Date(m.targetDate!).getTime(), span.start, span.end);
                  const p = m.progress;
                  const donePct = p.total > 0 ? (p.done / p.total) * 100 : 0;
                  const overdue = !m.completedAt && daysUntil(m.targetDate!) < 0;
                  return (
                    <button
                      key={m.$id}
                      onClick={() => onOpenMilestone(m.$id)}
                      className="relative h-7 w-full text-left"
                      title={`${m.name} — ${p.done} of ${p.total} done`}
                    >
                      <div
                        className="absolute top-0 flex h-7 items-center overflow-hidden rounded-[var(--radius-sm)] border transition-colors hover:brightness-125"
                        style={{
                          left: 0,
                          width: `${Math.max(at, 8)}%`,
                          borderColor: overdue ? '#EF4444' : 'var(--border)',
                          backgroundColor: 'var(--fill)',
                        }}
                      >
                        <div
                          className="absolute inset-y-0 left-0"
                          style={{
                            width: `${donePct}%`,
                            backgroundColor: 'color-mix(in srgb, #22C55E 35%, transparent)',
                          }}
                        />
                        <span className="relative truncate px-2 text-[length:var(--text-caption)] text-text-primary">
                          {m.name}
                        </span>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          <div className="flex flex-col gap-2">
            {milestones.map((m) => (
              <MilestoneRow
                key={m.$id}
                milestone={m}
                onOpen={() => onOpenMilestone(m.$id)}
                onDelete={() => remove.mutate(m.$id)}
              />
            ))}
          </div>
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
  onOpen,
  onDelete,
}: {
  milestone: Milestone;
  onOpen: () => void;
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
        <button
          onClick={onOpen}
          className="flex-1 truncate text-left text-[length:var(--text-body)] font-medium text-text-primary hover:underline"
        >
          {milestone.name}
        </button>

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
