import { Fragment, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useRoutedSelection } from '@/hooks/use-routed-selection';
import { ItemDetail } from './ItemDetail';
import { RoadmapView } from './RoadmapView';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CircleDashed, CircleDot, CircleCheckBig, Ban, PauseCircle, Eye, Plus, Columns3, List, Map, Grid3x3 } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { FormDialog, SelectField, TextField } from '@/components/form-dialog';
import { Dialog, DialogBody, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { toast } from '@/components/toast';
import { cn } from '@/lib/utils';

/*
 * Plan — the work a project has decided to do.
 *
 * An item is intent, in a sentence somebody would say out loud. What the
 * software must then do is a specification and whether it does it is a test;
 * both are things an item points at, so the links here are how the rest of
 * Applad is reached from the decision that caused it.
 *
 * Closed work is out of the list by default. A backlog showing everything ever
 * finished is not a backlog.
 */

interface Link {
  $id: string;
  kind: string;
  ref: string;
  label?: string;
}

interface Item {
  assigneeId?: string;
  $id: string;
  title: string;
  body: string;
  status: string;
  priority: string;
  labels: string[];
  links: Link[];
  closedAt?: string;
}

const STATUS = {
  todo: { label: 'Todo', icon: CircleDashed, color: 'var(--text-muted)' },
  in_progress: { label: 'In progress', icon: CircleDot, color: '#3B82F6' },
  in_review: { label: 'In review', icon: Eye, color: '#A78BFA' },
  blocked: { label: 'Blocked', icon: PauseCircle, color: '#F59E0B' },
  done: { label: 'Done', icon: CircleCheckBig, color: '#22C55E' },
  cancelled: { label: 'Cancelled', icon: Ban, color: 'var(--text-subtle)' },
} as const;

const PRIORITY_COLOR: Record<string, string> = {
  urgent: '#EF4444',
  high: '#F59E0B',
  medium: 'var(--text-muted)',
  low: 'var(--text-subtle)',
};

// The order work moves through, left to right and top to bottom. It read
// in_progress → blocked → todo, which is the order of attention rather than
// of progress, and on a board that runs the stages backwards.
const ORDER = ['todo', 'in_progress', 'in_review', 'blocked', 'done', 'cancelled'];

/** Who work is assigned to, by id, for the labels on rows and cards. */
function useAssigneeNames() {
  const { data } = useQuery({
    queryKey: ['plan-assignees'],
    queryFn: async () =>
      ((await api.get('/plan/assignees')).data as {
        assignees: { $id: string; name: string; kind: string }[];
      }).assignees ?? [],
  });
  const byId: Record<string, { name: string; kind: string }> = {};
  for (const a of data ?? []) byId[a.$id] = { name: a.name, kind: a.kind };
  return byId;
}

export function PlanPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const qc = useQueryClient();
  // Which item is open belongs in the address, so a refresh does not throw
  // somebody out of what they were reading.
  const selection = useRoutedSelection('plan', 'itemId');
  const assignees = useAssigneeNames();
  const [showClosed, setShowClosed] = useState(false);
  const [creating, setCreating] = useState(false);
  // The matrix that turns impact and urgency into a priority is a project-wide
  // policy, so it is edited from the page rather than from any one item.
  const [matrixOpen, setMatrixOpen] = useState(false);
  // Set when a milestone is opened from the roadmap: the list then shows that
  // milestone's work rather than everything.
  const [milestone, setMilestone] = useState<string | null>(null);
  // Remembered, because which view suits you is a preference about how you
  // work rather than something to re-choose on every visit.
  const [view, setView] = useState<'list' | 'board' | 'roadmap'>(
    () => (localStorage.getItem('applad_plan_view') as 'list' | 'board' | 'roadmap') ?? 'list',
  );
  const chooseView = (v: 'list' | 'board' | 'roadmap') => {
    setView(v);
    localStorage.setItem('applad_plan_view', v);
  };

  const query = useQuery({
    queryKey: ['plan-items', projectId, showClosed, milestone],
    queryFn: async () =>
      (
        (
          await api.get('/plan/items', {
            params: {
              includeClosed: showClosed || undefined,
              milestoneId: milestone ?? undefined,
            },
          })
        ).data as { items: Item[] }
      ).items ?? [],
  });

  const milestonesQuery = useQuery({
    queryKey: ['plan-milestones'],
    queryFn: async () =>
      ((await api.get('/plan/milestones')).data as { milestones: { $id: string; name: string }[] })
        .milestones ?? [],
  });
  const milestones = milestonesQuery.data ?? [];

  const items = query.data ?? [];
  const grouped = ORDER.map((status) => ({
    status,
    items: items.filter((i) => i.status === status),
  })).filter((g) => g.items.length > 0);

  const setStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.patch(`/plan/items/${id}`, { status }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['plan-items'] }),
    onError: (e) => toast.error(friendlyError(e)),
  });

  if (selection.id) {
    return <ItemDetail itemId={selection.id} onBack={selection.clear} />;
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Plan</h1>
          <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
            What this project has decided to do, and what each decision led to.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setMatrixOpen(true)}>
            <Grid3x3 size={14} />
            Priority matrix
          </Button>
          <Button onClick={() => setCreating(true)}>
            <Plus size={14} />
            New item
          </Button>
        </div>
      </div>

      <div className="flex items-center gap-3">
        <label
          className={cn(
            'flex w-fit cursor-pointer items-center gap-2 text-[length:var(--text-label)] text-text-secondary',
            view === 'roadmap' && 'invisible',
          )}
        >
        <input
          type="checkbox"
          checked={showClosed}
          onChange={(e) => setShowClosed(e.target.checked)}
          className="accent-[var(--color-accent)]"
        />
          Show closed work
        </label>

        {view !== 'roadmap' && milestones.length > 0 && (
          <select
            value={milestone ?? ''}
            onChange={(e) => setMilestone(e.target.value || null)}
            className="h-8 rounded-[var(--radius)] border border-field-border bg-field-fill px-2 text-[length:var(--text-label)] text-text-secondary"
          >
            <option value="">All milestones</option>
            {milestones.map((m) => (
              <option key={m.$id} value={m.$id}>
                {m.name}
              </option>
            ))}
          </select>
        )}

        <div className="ml-auto flex h-8 items-center overflow-hidden rounded-[var(--radius)] border border-field-border bg-field-fill">
          {([
            ['list', List],
            ['board', Columns3],
            ['roadmap', Map],
          ] as const).map(([value, Icon], i) => (
            <div key={value} className="flex h-full">
              {i > 0 && <div className="w-px bg-field-border" />}
              <button
                type="button"
                onClick={() => chooseView(value)}
                aria-label={`${value} view`}
                className={cn(
                  'flex h-full w-8 items-center justify-center transition-colors',
                  view === value
                    ? 'bg-fill-active text-text-primary'
                    : 'text-text-subtle hover:bg-fill-hover hover:text-text-secondary',
                )}
              >
                <Icon size={14} />
              </button>
            </div>
          ))}
        </div>
      </div>

      {view === 'roadmap' ? (
        <RoadmapView onOpenItem={selection.select} />
      ) : query.isLoading ? (
        <div className="py-10 text-center text-[length:var(--text-body)] text-text-muted">
          Loading…
        </div>
      ) : query.error ? (
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={CircleDashed}
          title="Nothing planned yet"
          subtitle="Add the first thing this project has decided to do."
        />
      ) : view === 'board' ? (
        <BoardView
          items={items}
          showClosed={showClosed}
          onStatus={(id, status) => setStatus.mutate({ id, status })}
          onOpen={selection.select}
          assignees={assignees}
        />
      ) : (
        <div className="flex flex-col gap-6">
          {grouped.map((group) => {
            const meta = STATUS[group.status as keyof typeof STATUS];
            return (
              <div key={group.status} className="flex flex-col gap-2">
                <div className="flex items-center gap-2">
                  <meta.icon size={14} style={{ color: meta.color }} />
                  <span className="text-[length:var(--text-label)] font-medium text-text-primary">
                    {meta.label}
                  </span>
                  <span className="text-[length:var(--text-caption)] text-text-subtle">
                    {group.items.length}
                  </span>
                </div>
                <div className="flex flex-col gap-1.5">
                  {group.items.map((item) => (
                    <ItemRow
                      key={item.$id}
                      item={item}
                      onStatus={(status) => setStatus.mutate({ id: item.$id, status })}
                      onOpen={() => selection.select(item.$id)}
                      assignee={item.assigneeId ? assignees[item.assigneeId] : undefined}
                    />
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      )}

      <CreateItemDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={() => qc.invalidateQueries({ queryKey: ['plan-items'] })}
      />

      <MatrixDialog open={matrixOpen} onOpenChange={setMatrixOpen} />
    </div>
  );
}

/*
 * The priority matrix, editable.
 *
 * Impact and urgency are answered per item; what those two answers resolve to
 * is a policy this dialog owns. A cell is a select, and changing it PUTs the
 * one cell — the grid is not saved as a whole, so two people editing different
 * cells do not clobber each other. Change and defect keep separate grids
 * because the same pair of answers should not mean the same urgency for
 * building something wanted and fixing something broken.
 */
const MATRIX_KINDS = ['change', 'defect'] as const;
const MATRIX_PRIORITIES = ['low', 'medium', 'high', 'urgent'];
const MATRIX_LEVELS = [3, 2, 1];

function MatrixDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
}) {
  const qc = useQueryClient();
  const [kind, setKind] = useState<(typeof MATRIX_KINDS)[number]>('change');

  const { data: cells = [] } = useQuery({
    queryKey: ['plan-matrix', kind],
    queryFn: async () =>
      ((await api.get('/plan/matrix', { params: { kind } })).data as {
        cells: { impact: number; urgency: number; priority: string }[];
      }).cells ?? [],
    enabled: open,
  });

  const set = useMutation({
    mutationFn: (body: { kind: string; impact: number; urgency: number; priority: string }) =>
      api.put('/plan/matrix', body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['plan-matrix', kind] });
      // Derived priorities on items follow from the grid, so refresh the list.
      qc.invalidateQueries({ queryKey: ['plan-items'] });
    },
    onError: (e) => toast.error(friendlyError(e)),
  });

  const priorityAt = (impact: number, urgency: number) =>
    cells.find((c) => c.impact === impact && c.urgency === urgency)?.priority ?? 'medium';

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent width={520}>
        <DialogHeader>
          <DialogTitle>Priority matrix</DialogTitle>
          <DialogDescription>
            How an item's impact and urgency resolve to its priority. Change a cell to set the
            policy for this project.
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <div className="mb-4 flex h-8 w-fit items-center overflow-hidden rounded-[var(--radius)] border border-field-border bg-field-fill">
            {MATRIX_KINDS.map((k, i) => (
              <div key={k} className="flex h-full">
                {i > 0 && <div className="w-px bg-field-border" />}
                <button
                  type="button"
                  onClick={() => setKind(k)}
                  className={cn(
                    'flex h-full items-center px-3 text-[length:var(--text-label)] capitalize transition-colors',
                    kind === k
                      ? 'bg-fill-active text-text-primary'
                      : 'text-text-subtle hover:bg-fill-hover hover:text-text-secondary',
                  )}
                >
                  {k}
                </button>
              </div>
            ))}
          </div>

          <div className="grid grid-cols-[auto_repeat(3,1fr)] gap-1.5">
            <div />
            {MATRIX_LEVELS.map((u) => (
              <div
                key={u}
                className="text-center text-[length:var(--text-caption)] text-text-subtle"
              >
                Urgency {LEVEL_LABEL[u]}
              </div>
            ))}

            {MATRIX_LEVELS.map((impact) => (
              <Fragment key={impact}>
                <div className="flex items-center pr-2 text-[length:var(--text-caption)] text-text-subtle">
                  Impact {LEVEL_LABEL[impact]}
                </div>
                {MATRIX_LEVELS.map((urgency) => {
                  const priority = priorityAt(impact, urgency);
                  return (
                    <select
                      key={urgency}
                      value={priority}
                      onChange={(e) =>
                        set.mutate({ kind, impact, urgency, priority: e.target.value })
                      }
                      className="w-full rounded-[var(--radius)] border border-field-border bg-field-fill px-2 py-1.5 text-center text-[length:var(--text-label)] font-medium capitalize"
                      style={{ color: PRIORITY_COLOR[priority] }}
                    >
                      {MATRIX_PRIORITIES.map((p) => (
                        <option key={p} value={p} className="text-text-primary">
                          {p}
                        </option>
                      ))}
                    </select>
                  );
                })}
              </Fragment>
            ))}
          </div>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

/*
 * The same work as columns.
 *
 * A list answers "what is outstanding"; a board answers "where is everything".
 * Dragging a card is the same edit as changing the select in the list — one
 * PATCH of one field — so the two views cannot disagree about what a move
 * means.
 *
 * Closed columns appear only when closed work is asked for, or a board of
 * a long-lived project is mostly a monument to finished things.
 */
function BoardView({
  items,
  showClosed,
  onStatus,
  onOpen,
  assignees,
}: {
  items: Item[];
  showClosed: boolean;
  onStatus: (id: string, status: string) => void;
  onOpen: (id: string) => void;
  assignees: Record<string, { name: string; kind: string }>;
}) {
  const [dragging, setDragging] = useState<string | null>(null);
  const [over, setOver] = useState<string | null>(null);

  const columns = ORDER.filter(
    (s) => showClosed || (s !== 'done' && s !== 'cancelled'),
  );

  return (
    <div className="flex gap-3 overflow-x-auto pb-2">
      {columns.map((status) => {
        const meta = STATUS[status as keyof typeof STATUS];
        const column = items.filter((i) => i.status === status);
        return (
          <div
            key={status}
            onDragOver={(e) => {
              e.preventDefault();
              setOver(status);
            }}
            onDragLeave={() => setOver((s) => (s === status ? null : s))}
            onDrop={() => {
              if (dragging) onStatus(dragging, status);
              setDragging(null);
              setOver(null);
            }}
            className={cn(
              'flex w-[280px] shrink-0 flex-col gap-2 rounded-[var(--radius-10)] border p-3 transition-colors',
              over === status
                ? 'border-[var(--color-accent)] bg-fill-hover'
                : 'border-border bg-surface-alt',
            )}
          >
            <div className="flex items-center gap-2">
              <meta.icon size={13} style={{ color: meta.color }} />
              <span className="text-[length:var(--text-label)] font-medium text-text-primary">
                {meta.label}
              </span>
              <span className="text-[length:var(--text-caption)] text-text-subtle">
                {column.length}
              </span>
            </div>

            {column.map((item) => (
              <div
                key={item.$id}
                draggable
                onClick={() => onOpen(item.$id)}
                onDragStart={() => setDragging(item.$id)}
                onDragEnd={() => {
                  setDragging(null);
                  setOver(null);
                }}
                className={cn(
                  'flex cursor-grab flex-col gap-2 rounded-[var(--radius)] border border-border bg-surface p-3 active:cursor-grabbing',
                  dragging === item.$id && 'opacity-50',
                )}
              >
                <span className="text-[length:var(--text-body)] text-text-primary">
                  {item.title}
                </span>
                <div className="flex flex-wrap items-center gap-1.5">
                  <span
                    className="text-[length:var(--text-caption)] font-medium"
                    style={{ color: PRIORITY_COLOR[item.priority] }}
                  >
                    {item.priority}
                  </span>
                  {item.assigneeId && assignees[item.assigneeId] && (
                    <span className="rounded-[var(--radius-sm)] border border-border bg-fill px-1.5 py-0.5 text-[length:var(--text-caption)] text-text-secondary">
                      {assignees[item.assigneeId].kind === 'ai'
                        ? 'Applad'
                        : assignees[item.assigneeId].name}
                    </span>
                  )}
                  {item.labels.map((label) => (
                    <span
                      key={label}
                      className="rounded-[var(--radius-sm)] border border-border bg-fill px-1.5 py-0.5 text-[length:var(--text-caption)] text-text-secondary"
                    >
                      {label}
                    </span>
                  ))}
                  {item.links.map((link) => (
                    <span
                      key={link.$id}
                      title={`${link.kind}: ${link.ref}`}
                      className="max-w-[150px] truncate rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)]"
                      style={{
                        backgroundColor: 'color-mix(in srgb, var(--color-accent) 12%, transparent)',
                        color: 'var(--color-accent)',
                      }}
                    >
                      {link.kind}
                    </span>
                  ))}
                </div>
              </div>
            ))}

            {column.length === 0 && (
              <div className="py-6 text-center text-[length:var(--text-caption)] text-text-subtle">
                Nothing here
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

function ItemRow({
  item,
  onStatus,
  onOpen,
  assignee,
}: {
  item: Item;
  onStatus: (status: string) => void;
  onOpen: () => void;
  assignee?: { name: string; kind: string };
}) {
  const closed = item.status === 'done' || item.status === 'cancelled';
  return (
    <div className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-surface px-4 py-3 transition-colors hover:bg-fill-hover">
      <select
        value={item.status}
        onChange={(e) => onStatus(e.target.value)}
        className="cursor-pointer rounded-[var(--radius-sm)] border border-field-border bg-fill px-2 py-1 text-[length:var(--text-caption)] text-text-secondary"
      >
        {Object.entries(STATUS).map(([value, meta]) => (
          <option key={value} value={value}>
            {meta.label}
          </option>
        ))}
      </select>

      <button
        onClick={onOpen}
        className={cn(
          'flex-1 truncate text-left text-[length:var(--text-body)] hover:underline',
          closed ? 'text-text-muted line-through' : 'text-text-primary',
        )}
      >
        {item.title}
      </button>

      {item.labels.map((label) => (
        <span
          key={label}
          className="rounded-[var(--radius-sm)] border border-border bg-fill px-1.5 py-0.5 text-[length:var(--text-caption)] text-text-secondary"
        >
          {label}
        </span>
      ))}

      {/* What this decision led to. A plan item with no links is a decision
          nobody has acted on yet, which is worth being able to see. */}
      {item.links.map((link) => (
        <span
          key={link.$id}
          title={`${link.kind}: ${link.ref}`}
          className="max-w-[180px] truncate rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)]"
          style={{ backgroundColor: 'color-mix(in srgb, var(--color-accent) 12%, transparent)', color: 'var(--color-accent)' }}
        >
          {link.kind} · {link.label || link.ref}
        </span>
      ))}

      {assignee && (
        <span
          className="rounded-[var(--radius-sm)] border border-border bg-fill px-1.5 py-0.5 text-[length:var(--text-caption)] text-text-secondary"
          title={assignee.kind === 'ai' ? 'Assigned to Applad' : `Assigned to ${assignee.name}`}
        >
          {assignee.kind === 'ai' ? 'Applad' : assignee.name}
        </span>
      )}

      <span
        className="text-[length:var(--text-caption)] font-medium"
        style={{ color: PRIORITY_COLOR[item.priority] }}
      >
        {item.priority}
      </span>
    </div>
  );
}

function CreateItemDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onCreated: () => void;
}) {
  const [title, setTitle] = useState('');
  // Two answers rather than a verdict: the grid decides the priority, so
  // nobody has to weigh "how much does this matter" against "how soon" in
  // their head and report the average.
  const kind = 'change';
  const [impact, setImpact] = useState(2);
  const [urgency, setUrgency] = useState(2);
  const [saving, setSaving] = useState(false);

  // The anchors and the grid come from the server, so what "medium" means and
  // what it resolves to are never a second copy that can drift.
  const { data: meta } = useQuery({
    queryKey: ['plan-meta'],
    queryFn: async () =>
      (await api.get('/plan/meta')).data as {
        hints: Record<string, { impact: Record<string, string>; urgency: Record<string, string> }>;
      },
  });

  const { data: cells = [] } = useQuery({
    queryKey: ['plan-matrix', kind],
    queryFn: async () =>
      ((await api.get('/plan/matrix', { params: { kind } })).data as {
        cells: { impact: number; urgency: number; priority: string }[];
      }).cells ?? [],
  });

  const hints = meta?.hints?.[kind];
  const derived = cells.find((c) => c.impact === impact && c.urgency === urgency)?.priority;

  const submit = async () => {
    setSaving(true);
    try {
      const created = await api.post('/plan/items', { title: title.trim(), kind });
      const id = created.data.$id ?? created.data.id;
      await api.post(`/plan/items/${id}/rate`, { impact, urgency });
      setTitle('');
      setImpact(2);
      setUrgency(2);
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
      title="New item"
      subtitle="Something this project has decided to do"
      submitLabel="Add"
      loading={saving}
      submitDisabled={!title.trim()}
      onSubmit={submit}
    >
      <TextField
        label="Title"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder="e.g. Add promotions to checkout"
        autoFocus
      />

      <SelectField
        label="Impact"
        hint="How much it matters"
        value={String(impact)}
        onChange={(v) => setImpact(Number(v))}
        options={[3, 2, 1].map((l) => ({
          value: String(l),
          label: `${LEVEL_LABEL[l]} — ${hints?.impact?.[String(l)] ?? ''}`,
        }))}
      />
      <SelectField
        label="Urgency"
        hint="How soon it is needed"
        value={String(urgency)}
        onChange={(v) => setUrgency(Number(v))}
        options={[3, 2, 1].map((l) => ({
          value: String(l),
          label: `${LEVEL_LABEL[l]} — ${hints?.urgency?.[String(l)] ?? ''}`,
        }))}
      />

      <div className="flex items-center gap-2 rounded-[var(--radius)] border border-border bg-fill px-3 py-2">
        <span className="text-[length:var(--text-label)] text-text-secondary">Priority</span>
        <span
          className="rounded-[var(--radius-sm)] px-2 py-0.5 text-[length:var(--text-label)] font-medium"
          style={{
            backgroundColor: `color-mix(in srgb, ${PRIORITY_COLOR[derived ?? 'medium']} 18%, transparent)`,
            color: PRIORITY_COLOR[derived ?? 'medium'],
          }}
        >
          {derived ?? '—'}
        </span>
        <span className="ml-auto text-[length:var(--text-caption)] text-text-subtle">
          from this project's matrix
        </span>
      </div>
    </FormDialog>
  );
}

export const LEVEL_LABEL: Record<number, string> = { 3: 'High', 2: 'Medium', 1: 'Low' };
