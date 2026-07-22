import { Fragment, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CalendarDays, Plus, Trash2, Flag, ChevronRight } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { ConfirmDialog, FormDialog, TextField } from '@/components/form-dialog';
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

interface PlanItem {
  $id: string;
  title: string;
  status: string;
  priority: string;
  kind: string;
  targetDate?: string;
  criteriaCount: number;
  criteriaSpecified: number;
}

const STATUS_DOT: Record<string, string> = {
  todo: 'var(--text-muted)',
  in_progress: '#3B82F6',
  in_review: '#A78BFA',
  blocked: '#F59E0B',
  done: '#22C55E',
  cancelled: 'var(--text-subtle)',
};

const PRIORITY_TONE: Record<string, string> = {
  urgent: '#EF4444',
  high: '#F59E0B',
  medium: 'var(--text-muted)',
  low: 'var(--text-subtle)',
};

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

export function RoadmapView({ onOpenItem }: { onOpenItem: (id: string) => void }) {
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
  // Deleting a milestone is not undoable and other people's work points at
  // it, so it asks for the name — the same confirmation the rest of the
  // console uses for things that cannot be taken back.
  const [confirming, setConfirming] = useState<Milestone | null>(null);
  // Opening a milestone shows its work underneath it, rather than sending
  // somebody to another view: the timeline is the context that makes the
  // items mean anything, and leaving it to read them throws that away.
  const [expanded, setExpanded] = useState<string | null>(null);

  const milestones = query.data ?? [];
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
        /*
         * One row per milestone, carrying both halves.
         *
         * This was a timeline above a list of the same milestones: the bars
         * said when and the rows said how much, and neither was complete on
         * its own. A row now holds its name, date and counts on the left and
         * its bar in the shared track on the right, so a milestone is read
         * once.
         */
        <div className="overflow-hidden rounded-[var(--radius-10)] border border-border bg-surface">
          <div className="flex border-b border-border px-4 py-2">
            <div className="w-[300px] shrink-0" />
            <div className="relative h-4 flex-1">
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
          </div>

          {milestones.map((m) => {
            const p = m.progress;
            const when = whenLabel(m.targetDate, m.completedAt);
            const hasItems = p.total > 0;
            const donePct = hasItems ? (p.done / p.total) * 100 : 0;
            const unspecified = p.criteria - p.specified;
            const at = m.targetDate
              ? positionOf(new Date(m.targetDate).getTime(), span.start, span.end)
              : 0;
            const overdue = !m.completedAt && m.targetDate && daysUntil(m.targetDate) < 0;

            return (
              <Fragment key={m.$id}>
              <div
                role="button"
                tabIndex={0}
                onClick={() => setExpanded(expanded === m.$id ? null : m.$id)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    setExpanded(expanded === m.$id ? null : m.$id);
                  }
                }}
                className="group flex cursor-pointer items-center gap-4 border-b border-border px-4 py-3 hover:bg-fill-hover"
              >
                <div className="flex w-[300px] shrink-0 flex-col gap-1">
                  <div className="flex items-center gap-2">
                    <span className="flex items-center gap-1 truncate text-[length:var(--text-body)] font-medium text-text-primary">
                      <ChevronRight
                        size={13}
                        className={cn(
                          'shrink-0 text-text-subtle transition-transform',
                          expanded === m.$id && 'rotate-90',
                        )}
                      />
                      {m.name}
                    </span>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        setConfirming(m);
                      }}
                      className="opacity-0 transition-opacity group-hover:opacity-100"
                      aria-label="Delete milestone"
                    >
                      <Trash2 size={12} className="text-text-subtle hover:text-[#EF4444]" />
                    </button>
                  </div>

                  <div className="flex flex-wrap items-center gap-x-2 text-[length:var(--text-caption)]">
                    {m.targetDate && (
                      <span className="flex items-center gap-1 text-text-muted">
                        <CalendarDays size={11} />
                        {new Date(m.targetDate).toLocaleDateString(undefined, {
                          day: 'numeric',
                          month: 'short',
                        })}
                      </span>
                    )}
                    <span style={{ color: when.tone }}>{when.text}</span>
                    <span className="text-text-secondary">
                      {hasItems ? `${p.done}/${p.total} done` : 'no items'}
                    </span>
                    {p.criteria > 0 && unspecified > 0 && (
                      <span
                        style={{ color: '#F59E0B' }}
                        title="Acceptance criteria not yet expressed as rules in a specification"
                      >
                        {unspecified} unspecified
                      </span>
                    )}
                  </div>
                </div>

                {/* The track. A bar ends on the date it is aimed at, shaded by
                    how much of it is done. */}
                <div className="relative h-8 flex-1">
                  <div
                    className="pointer-events-none absolute -top-3 bottom-[-0.75rem] w-px opacity-60"
                    style={{ left: `${todayAt}%`, backgroundColor: 'var(--color-accent)' }}
                  />
                  {m.targetDate ? (
                    <div
                      className="absolute inset-y-1 left-0 flex items-center overflow-hidden rounded-[var(--radius-sm)] border transition-transform group-hover:scale-y-110"
                      style={{
                        width: `${Math.max(at, 4)}%`,
                        borderColor: overdue ? '#EF4444' : 'var(--border)',
                        backgroundColor: 'var(--fill)',
                      }}
                      title={`${p.done} of ${p.total} done`}
                    >
                      <div
                        className="absolute inset-y-0 left-0"
                        style={{
                          width: `${donePct}%`,
                          backgroundColor: 'color-mix(in srgb, #22C55E 40%, transparent)',
                        }}
                      />
                    </div>
                  ) : (
                    <span className="flex h-full items-center text-[length:var(--text-caption)] text-text-subtle">
                      Not on the roadmap — no target date
                    </span>
                  )}
                </div>
              </div>

              {expanded === m.$id && (
                <MilestoneItems
                  milestoneId={m.$id}
                  onOpenItem={onOpenItem}
                  span={span}
                  todayAt={todayAt}
                  milestoneAt={at}
                />
              )}
              </Fragment>
            );
          })}
        </div>
      )}

      <ConfirmDialog
        open={confirming !== null}
        onOpenChange={(o) => !o && setConfirming(null)}
        title="Delete milestone"
        message={
          <>
            <b>{confirming?.name}</b> will be deleted. Its items are not — they stay in the
            backlog, no longer aimed at a date.
          </>
        }
        confirmText={confirming?.name}
        confirmTextLabel="Type the milestone name to confirm"
        loading={remove.isPending}
        onConfirm={() => {
          if (confirming) remove.mutate(confirming.$id);
          setConfirming(null);
        }}
      />

      <CreateMilestoneDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={() => qc.invalidateQueries({ queryKey: ['plan-milestones'] })}
      />
    </div>
  );
}

/*
 * A milestone's work, shown under it.
 *
 * Only fetched when somebody opens one — a roadmap of twenty milestones
 * should not ask for twenty lists of items to draw two bars.
 */
function MilestoneItems({
  milestoneId,
  onOpenItem,
  span,
  todayAt,
  milestoneAt,
}: {
  milestoneId: string;
  onOpenItem: (id: string) => void;
  span: { start: number; end: number };
  todayAt: number;
  milestoneAt: number;
}) {
  const { data, isLoading } = useQuery({
    queryKey: ['plan-items', 'milestone', milestoneId],
    queryFn: async () =>
      (
        (
          await api.get('/plan/items', { params: { milestoneId, includeClosed: true } })
        ).data as { items: PlanItem[] }
      ).items ?? [],
  });

  const items = data ?? [];

  return (
    <div className="border-b border-border bg-surface-alt last:border-0">
      {isLoading ? (
        <div className="py-3 pl-9 text-[length:var(--text-caption)] text-text-muted">Loading…</div>
      ) : items.length === 0 ? (
        <div className="py-3 pl-9 text-[length:var(--text-caption)] text-text-subtle">
          No items in this milestone yet.
        </div>
      ) : (
        items.map((item) => {
          const closed = item.status === 'done' || item.status === 'cancelled';
          const due = item.targetDate ? daysUntil(item.targetDate) : null;
          // An item is placed at its own date when it has one, and at its
          // milestone's when it does not: work with no date of its own is
          // still due by the thing that contains it, and hiding it from the
          // track would say it is due never.
          const at = item.targetDate
            ? positionOf(new Date(item.targetDate).getTime(), span.start, span.end)
            : milestoneAt;
          const unspecified = item.criteriaCount - item.criteriaSpecified;

          return (
            <button
              key={item.$id}
              onClick={() => onOpenItem(item.$id)}
              className="flex w-full items-center gap-4 px-4 py-2 text-left hover:bg-fill-hover"
            >
              <div className="flex w-[300px] shrink-0 flex-col gap-0.5 pl-5">
                <div className="flex items-center gap-2">
                  <span
                    className="h-1.5 w-1.5 shrink-0 rounded-full"
                    style={{ backgroundColor: STATUS_DOT[item.status] ?? 'var(--text-subtle)' }}
                  />
                  <span
                    className={cn(
                      'truncate text-[length:var(--text-body)]',
                      closed ? 'text-text-muted line-through' : 'text-text-primary',
                    )}
                  >
                    {item.title}
                  </span>
                </div>

                <div className="flex flex-wrap items-center gap-x-2 pl-3.5 text-[length:var(--text-caption)]">
                  <span style={{ color: PRIORITY_TONE[item.priority] }}>{item.priority}</span>
                  {item.criteriaCount > 0 && (
                    <span
                      style={unspecified > 0 ? { color: '#F59E0B' } : { color: 'var(--text-subtle)' }}
                      title="Acceptance criteria expressed as rules in a specification"
                    >
                      {item.criteriaSpecified}/{item.criteriaCount} specified
                    </span>
                  )}
                  {item.kind === 'defect' && <span style={{ color: '#EF4444' }}>defect</span>}
                </div>
              </div>

              {/* Placed on the same axis as the milestone above, so an item
                  due after the thing that contains it is visible as such. */}
              <div className="relative h-7 flex-1">
                <div
                  className="pointer-events-none absolute inset-y-0 w-px opacity-40"
                  style={{ left: `${todayAt}%`, backgroundColor: 'var(--color-accent)' }}
                />
                <span
                  className="absolute top-1/2 flex -translate-y-1/2 items-center gap-1.5 whitespace-nowrap"
                  style={{ left: `calc(${at}% - 3px)` }}
                >
                  <span
                    className="h-1.5 w-1.5 rotate-45"
                    style={{
                      backgroundColor: closed
                        ? '#22C55E'
                        : due !== null && due < 0
                          ? '#EF4444'
                          : 'var(--text-muted)',
                    }}
                  />
                  {item.targetDate && (
                    <span
                      className="text-[length:var(--text-caption)]"
                      style={{ color: due !== null && due < 0 ? '#EF4444' : 'var(--text-subtle)' }}
                    >
                      {closed
                        ? 'done'
                        : due !== null && due < 0
                          ? `${-due}d late`
                          : new Date(item.targetDate).toLocaleDateString(undefined, {
                              day: 'numeric',
                              month: 'short',
                            })}
                    </span>
                  )}
                </span>
              </div>
            </button>
          );
        })
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
