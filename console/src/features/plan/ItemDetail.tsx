import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { ChevronLeft, Bug, GitCommitHorizontal, Plus, Trash2, Check, Pencil } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { RichTextEditor } from '@/components/rich-text-editor';
import { toast } from '@/components/toast';
import { cn } from '@/lib/utils';
import { AnchoredChoice } from './PlanPage';

/*
 * One item, and everything agreed about it.
 *
 * Laid out as a document with its properties beside it: prose is read at a
 * comfortable measure, and the fields that describe the work — what state it
 * is in, when it is due, which milestone it belongs to — sit in a column that
 * does not push the reading wider. The page filled the whole window before,
 * so a sentence ran the width of a monitor and nothing could be changed
 * without leaving.
 */

interface Criterion {
  $id: string;
  text: string;
  specRef: string;
  met: boolean;
}

interface Comment {
  $id: string;
  body: string;
  $createdAt: string;
}

interface Event {
  $id: string;
  field: string;
  oldValue: string;
  newValue: string;
  $createdAt: string;
}

const FIELD_LABEL: Record<string, string> = {
  status: 'status',
  priority: 'priority',
  kind: 'kind',
  milestone: 'milestone',
  targetDate: 'target date',
  title: 'title',
  assignee: 'assignee',
};

const LEVEL_NAME: Record<number, string> = { 1: 'low', 2: 'medium', 3: 'high' };

const PRIORITY_TONE: Record<string, string> = {
  urgent: '#EF4444',
  high: '#F59E0B',
  medium: 'var(--text-muted)',
  low: 'var(--text-subtle)',
};

interface Item {
  $id: string;
  title: string;
  body: string;
  status: string;
  priority: string;
  kind: string;
  milestoneId?: string;
  impact?: number;
  urgency?: number;
  priorityIsManual: boolean;
  targetDate?: string;
  labels: string[];
  links: { $id: string; kind: string; ref: string }[];
}

const STATUSES = ['todo', 'in_progress', 'in_review', 'blocked', 'done', 'cancelled'];
const PRIORITIES = ['low', 'medium', 'high', 'urgent'];
const KINDS = ['change', 'defect'];

const LABEL: Record<string, string> = {
  todo: 'Todo',
  in_progress: 'In progress',
  in_review: 'In review',
  blocked: 'Blocked',
  done: 'Done',
  cancelled: 'Cancelled',
};

export function ItemDetail({ itemId, onBack }: { itemId: string; onBack: () => void }) {
  const qc = useQueryClient();

  const item = useQuery({
    queryKey: ['plan-item', itemId],
    queryFn: async () => (await api.get(`/plan/items/${itemId}`)).data as Item,
  });

  const milestones = useQuery({
    queryKey: ['plan-milestones'],
    queryFn: async () =>
      ((await api.get('/plan/milestones')).data as { milestones: { $id: string; name: string }[] })
        .milestones ?? [],
  });

  const criteria = useQuery({
    queryKey: ['plan-criteria', itemId],
    queryFn: async () =>
      ((await api.get(`/plan/items/${itemId}/criteria`)).data as { criteria: Criterion[] })
        .criteria ?? [],
  });

  const comments = useQuery({
    queryKey: ['plan-comments', itemId],
    queryFn: async () =>
      ((await api.get(`/plan/items/${itemId}/comments`)).data as { comments: Comment[] })
        .comments ?? [],
  });

  // Every property change is the same request, so they share one mutation.
  const patch = useMutation({
    mutationFn: (body: Record<string, unknown>) => api.patch(`/plan/items/${itemId}`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['plan-item', itemId] });
      qc.invalidateQueries({ queryKey: ['plan-items'] });
      qc.invalidateQueries({ queryKey: ['plan-milestones'] });
    },
    onError: (e) => toast.error(friendlyError(e)),
  });

  const data = item.data;
  const list = criteria.data ?? [];
  const specified = list.filter((c) => c.specRef.trim() !== '').length;

  return (
    <div className="mx-auto flex w-full max-w-[1080px] flex-col gap-6 p-6 md:p-8">
      <button
        onClick={onBack}
        className="flex w-fit items-center gap-1 text-[length:var(--text-label)] text-text-muted transition-colors hover:text-text-primary"
      >
        <ChevronLeft size={14} />
        Plan
      </button>

      <div className="flex flex-col gap-8 lg:flex-row lg:items-start lg:justify-center">
        {/* The document. Capped at a readable measure rather than the width of
            whatever monitor it is opened on. */}
        <div className="flex w-full min-w-0 max-w-[720px] flex-1 flex-col gap-8">
          {data && (
            <EditableTitle
              kind={data.kind}
              value={data.title}
              onSave={(title) => patch.mutate({ title })}
            />
          )}

          {data && (
            <EditableBody value={data.body} onSave={(body) => patch.mutate({ body })} />
          )}

          <PriorityAssessment itemId={itemId} item={data} />

          <section className="flex flex-col gap-3">
            <div className="flex items-baseline gap-2">
              <h2 className="text-[length:var(--text-control)] font-medium text-text-primary">
                Acceptance criteria
              </h2>
              {list.length > 0 && (
                /* Two counts, because they say different things: what is
                   agreed, and how much of it has become behaviour anything
                   can check. */
                <span className="text-[length:var(--text-caption)] text-text-subtle">
                  {specified} of {list.length} specified
                </span>
              )}
            </div>

            <CriteriaList itemId={itemId} criteria={list} />
          </section>

          <ActivitySection itemId={itemId} comments={comments.data ?? []} />
        </div>

        {/* The properties. A column, so changing one does not mean leaving the
            page and the reading measure stays where it is. */}
        <aside className="flex w-full shrink-0 flex-col gap-3.5 rounded-[var(--radius)] border border-border bg-surface p-3.5 lg:w-[240px]">
          <Property label="Status">
            <Choice
              value={data?.status ?? 'todo'}
              options={STATUSES.map((s) => ({ value: s, label: LABEL[s] ?? s }))}
              onChange={(status) => patch.mutate({ status })}
            />
          </Property>

          <Property label="Impact and urgency">
            <div className="flex flex-col gap-1 text-[length:var(--text-label)] text-text-primary">
              <span>
                Impact:{' '}
                <span className="text-text-secondary">
                  {data?.impact ? LEVEL_NAME[data.impact] : 'not assessed'}
                </span>
              </span>
              <span>
                Urgency:{' '}
                <span className="text-text-secondary">
                  {data?.urgency ? LEVEL_NAME[data.urgency] : 'not assessed'}
                </span>
              </span>
            </div>
          </Property>

          <Property
            label="Priority"
            hint={
              data?.priorityIsManual
                ? 'Set directly. Answer the assessment to let the matrix decide instead.'
                : 'Follows from the assessment. Change an answer to change this.'
            }
          >
            {data?.priorityIsManual ? (
              <Choice
                value={data?.priority ?? 'medium'}
                options={PRIORITIES.map((p) => ({ value: p, label: p }))}
                onChange={(priority) => patch.mutate({ priority })}
              />
            ) : (
              <div className="flex items-center gap-2">
                <span
                  className="rounded-[var(--radius-sm)] px-2 py-1 text-[length:var(--text-label)] font-medium"
                  style={{
                    backgroundColor: `color-mix(in srgb, ${PRIORITY_TONE[data?.priority ?? 'medium']} 15%, transparent)`,
                    color: PRIORITY_TONE[data?.priority ?? 'medium'],
                  }}
                >
                  {data?.priority}
                </span>
                <button
                  onClick={() => patch.mutate({ priority: data?.priority })}
                  className="text-[length:var(--text-caption)] text-text-subtle hover:text-text-primary"
                  title="Stop deriving it and set it by hand"
                >
                  override
                </button>
              </div>
            )}
          </Property>

          <Property label="Kind" hint="A defect is a promise already broken.">
            <Choice
              value={data?.kind ?? 'change'}
              options={KINDS.map((k) => ({ value: k, label: k }))}
              onChange={(kind) => patch.mutate({ kind })}
            />
          </Property>

          <Property label="Milestone">
            <Choice
              value={data?.milestoneId ?? ''}
              options={[
                { value: '', label: 'None' },
                ...(milestones.data ?? []).map((m) => ({ value: m.$id, label: m.name })),
              ]}
              onChange={(milestoneId) => patch.mutate({ milestoneId })}
            />
          </Property>

          <Property label="Target date">
            <input
              type="date"
              value={data?.targetDate ? data.targetDate.slice(0, 10) : ''}
              onChange={(e) => patch.mutate({ targetDate: e.target.value })}
              className="w-full rounded-[var(--radius)] border border-field-border bg-field-fill px-2 py-1.5 text-[length:var(--text-label)] text-text-primary"
            />
          </Property>

          {(data?.links ?? []).length > 0 && (
            <Property label="Links">
              <div className="flex flex-col gap-1">
                {data?.links.map((l) => (
                  <span
                    key={l.$id}
                    title={l.ref}
                    className="truncate rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)]"
                    style={{
                      backgroundColor: 'color-mix(in srgb, var(--color-accent) 12%, transparent)',
                      color: 'var(--color-accent)',
                    }}
                  >
                    {l.kind} · {l.ref}
                  </span>
                ))}
              </div>
            </Property>
          )}
        </aside>
      </div>
    </div>
  );
}

function Property({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[length:var(--text-caption)] text-text-subtle">{label}</span>
      {children}
      {hint && <span className="text-[length:var(--text-caption)] text-text-subtle">{hint}</span>}
    </div>
  );
}

function Choice({
  value,
  options,
  onChange,
}: {
  value: string;
  options: { value: string; label: string }[];
  onChange: (v: string) => void;
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="w-full rounded-[var(--radius)] border border-field-border bg-field-fill px-2 py-1.5 text-[length:var(--text-label)] text-text-primary"
    >
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  );
}

/* The title edits where it sits — retyping it in a dialog would be a worse
   version of clicking it. */
function EditableTitle({
  kind,
  value,
  onSave,
}: {
  kind: string;
  value: string;
  onSave: (v: string) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);

  useEffect(() => setDraft(value), [value]);

  const commit = () => {
    setEditing(false);
    if (draft.trim() && draft !== value) onSave(draft.trim());
  };

  return (
    <div className="flex items-start gap-2">
      {kind === 'defect' ? (
        <Bug size={18} className="mt-1.5 shrink-0" style={{ color: '#EF4444' }} />
      ) : (
        <GitCommitHorizontal size={18} className="mt-1.5 shrink-0 text-text-muted" />
      )}

      {editing ? (
        <input
          autoFocus
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Enter') commit();
            if (e.key === 'Escape') {
              setDraft(value);
              setEditing(false);
            }
          }}
          className="flex-1 rounded-[var(--radius)] border border-field-border bg-field-fill px-2 py-1 text-[length:var(--text-h1)] font-semibold text-text-primary"
        />
      ) : (
        <button
          onClick={() => setEditing(true)}
          className="group flex flex-1 items-center gap-2 text-left"
        >
          <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">{value}</h1>
          <Pencil
            size={13}
            className="shrink-0 text-text-subtle opacity-0 transition-opacity group-hover:opacity-100"
          />
        </button>
      )}
    </div>
  );
}

/* Markdown, written and previewed with the console's own editor. Read mode is
   the rendered version, because that is what the writing is for. */
function EditableBody({ value, onSave }: { value: string; onSave: (v: string) => void }) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);

  useEffect(() => setDraft(value), [value]);

  if (!editing) {
    return (
      <button
        onClick={() => setEditing(true)}
        className="group rounded-[var(--radius)] px-1 py-1 text-left transition-colors hover:bg-fill-hover"
      >
        {value ? (
          <div className="prose-plan text-[length:var(--text-body)] text-text-secondary">
            <Markdown remarkPlugins={[remarkGfm]}>{value}</Markdown>
          </div>
        ) : (
          <span className="text-[length:var(--text-body)] text-text-subtle">
            Add a description — what this is, and why it was decided.
          </span>
        )}
      </button>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <RichTextEditor value={draft} onChange={setDraft} minRows={8} />
      <div className="flex justify-end gap-2">
        <Button
          size="sm"
          variant="ghost"
          onClick={() => {
            setDraft(value);
            setEditing(false);
          }}
        >
          Cancel
        </Button>
        <Button
          size="sm"
          onClick={() => {
            onSave(draft);
            setEditing(false);
          }}
        >
          Save
        </Button>
      </div>
    </div>
  );
}

function CriteriaList({ itemId, criteria }: { itemId: string; criteria: Criterion[] }) {
  const qc = useQueryClient();
  const [text, setText] = useState('');
  const refresh = () => qc.invalidateQueries({ queryKey: ['plan-criteria', itemId] });

  const add = useMutation({
    mutationFn: () => api.post(`/plan/items/${itemId}/criteria`, { text: text.trim() }),
    onSuccess: () => {
      setText('');
      refresh();
    },
    onError: (e) => toast.error(friendlyError(e)),
  });

  const update = useMutation({
    mutationFn: ({ id, body }: { id: string; body: Record<string, unknown> }) =>
      api.patch(`/plan/criteria/${id}`, body),
    onSuccess: refresh,
    onError: (e) => toast.error(friendlyError(e)),
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.delete(`/plan/criteria/${id}`),
    onSuccess: refresh,
    onError: (e) => toast.error(friendlyError(e)),
  });

  return (
    <div className="flex flex-col gap-1.5">
      {criteria.map((c) => (
        <CriterionRow
          key={c.$id}
          criterion={c}
          onToggle={() => update.mutate({ id: c.$id, body: { met: !c.met } })}
          onSaveText={(t) => update.mutate({ id: c.$id, body: { text: t } })}
          onSaveSpec={(ref) => update.mutate({ id: c.$id, body: { specRef: ref } })}
          onDelete={() => remove.mutate(c.$id)}
        />
      ))}

      {criteria.length === 0 && (
        <p className="text-[length:var(--text-body)] text-text-subtle">
          Nothing agreed yet. Criteria are the constraints this work has to satisfy — they become
          the rules of its specification.
        </p>
      )}

      <div className="mt-1 flex gap-2">
        <input
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && text.trim()) add.mutate();
          }}
          placeholder="e.g. A discount never takes an order below zero"
          className="flex-1 rounded-[var(--radius)] border border-field-border bg-field-fill px-3 py-2 text-[length:var(--text-body)] text-text-primary placeholder:text-text-subtle"
        />
        <Button size="sm" onClick={() => add.mutate()} disabled={!text.trim() || add.isPending}>
          <Plus size={14} />
          Add
        </Button>
      </div>
    </div>
  );
}

function CriterionRow({
  criterion,
  onToggle,
  onSaveText,
  onSaveSpec,
  onDelete,
}: {
  criterion: Criterion;
  onToggle: () => void;
  onSaveText: (v: string) => void;
  onSaveSpec: (v: string) => void;
  onDelete: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(criterion.text);
  const [spec, setSpec] = useState(criterion.specRef);
  const [linking, setLinking] = useState(false);

  const specified = criterion.specRef.trim() !== '';

  return (
    <div className="group flex items-start gap-3 rounded-[var(--radius)] border border-border bg-surface px-3 py-2.5">
      <button
        onClick={onToggle}
        disabled={specified}
        title={
          specified
            ? 'A specification decides this one — the rule it became is checked by its own tests.'
            : 'Not yet expressed as behaviour; mark by hand until it is.'
        }
        className={cn(
          'mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-[3px] border transition-colors',
          criterion.met || specified
            ? 'border-transparent bg-[var(--color-accent)] text-white'
            : 'border-field-border hover:border-text-muted',
          specified && 'cursor-default opacity-70',
        )}
      >
        {(criterion.met || specified) && <Check size={11} />}
      </button>

      <div className="flex flex-1 flex-col gap-1">
        {editing ? (
          <input
            autoFocus
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onBlur={() => {
              setEditing(false);
              if (draft.trim() && draft !== criterion.text) onSaveText(draft.trim());
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') e.currentTarget.blur();
              if (e.key === 'Escape') {
                setDraft(criterion.text);
                setEditing(false);
              }
            }}
            className="rounded-[var(--radius-sm)] border border-field-border bg-field-fill px-2 py-1 text-[length:var(--text-body)] text-text-primary"
          />
        ) : (
          <button onClick={() => setEditing(true)} className="text-left">
            <span className="text-[length:var(--text-body)] text-text-primary">
              {criterion.text}
            </span>
          </button>
        )}

        {linking ? (
          <input
            autoFocus
            value={spec}
            onChange={(e) => setSpec(e.target.value)}
            onBlur={() => {
              setLinking(false);
              if (spec !== criterion.specRef) onSaveSpec(spec);
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') e.currentTarget.blur();
            }}
            placeholder="apply_discounts.feature#never-below-zero"
            className="rounded-[var(--radius-sm)] border border-field-border bg-field-fill px-2 py-1 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-primary"
          />
        ) : (
          <button onClick={() => setLinking(true)} className="w-fit text-left">
            {specified ? (
              <span
                className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)]"
                style={{
                  backgroundColor: 'color-mix(in srgb, var(--color-accent) 12%, transparent)',
                  color: 'var(--color-accent)',
                }}
              >
                {criterion.specRef}
              </span>
            ) : (
              <span className="text-[length:var(--text-caption)] text-text-subtle hover:text-text-muted">
                not specified — link a rule
              </span>
            )}
          </button>
        )}
      </div>

      <button
        onClick={onDelete}
        className="opacity-0 transition-opacity group-hover:opacity-100"
        aria-label="Remove criterion"
      >
        <Trash2 size={13} className="text-text-subtle hover:text-[#EF4444]" />
      </button>
    </div>
  );
}

/*
 * What people said, and what changed.
 *
 * Two different questions — "why is this being done this way" and "who moved
 * it to blocked on Tuesday" — so they are tabs rather than one stream. The
 * composer stays closed until there is something to say: a permanently open
 * editor is the loudest thing on a page that is mostly for reading.
 */
function ActivitySection({ itemId, comments }: { itemId: string; comments: Comment[] }) {
  const [tab, setTab] = useState<'comments' | 'history'>('comments');

  const activity = useQuery({
    queryKey: ['plan-activity', itemId],
    queryFn: async () =>
      ((await api.get(`/plan/items/${itemId}/activity`)).data as { activity: Event[] })
        .activity ?? [],
    enabled: tab === 'history',
  });

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center gap-1 border-b border-border">
        {([
          ['comments', `Comments${comments.length ? ` (${comments.length})` : ''}`],
          ['history', 'History'],
        ] as const).map(([value, label]) => (
          <button
            key={value}
            onClick={() => setTab(value)}
            className={cn(
              '-mb-px border-b-2 px-3 py-2 text-[length:var(--text-label)] transition-colors',
              tab === value
                ? 'border-[var(--color-accent)] text-text-primary'
                : 'border-transparent text-text-muted hover:text-text-primary',
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {tab === 'comments' ? (
        <Discussion itemId={itemId} comments={comments} />
      ) : (
        <History events={activity.data ?? []} loading={activity.isLoading} />
      )}
    </section>
  );
}

function History({ events, loading }: { events: Event[]; loading: boolean }) {
  if (loading) {
    return <p className="text-[length:var(--text-body)] text-text-muted">Loading…</p>;
  }
  if (events.length === 0) {
    return (
      <p className="text-[length:var(--text-body)] text-text-subtle">
        Nothing has changed since this was created.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {events.map((e) => (
        <div key={e.$id} className="flex items-baseline gap-2 text-[length:var(--text-body)]">
          <span className="text-text-secondary">
            {e.field === 'body' ? (
              <>edited the description</>
            ) : (
              <>
                changed <span className="text-text-primary">{FIELD_LABEL[e.field] ?? e.field}</span>
                {e.oldValue && (
                  <>
                    {' '}from <span className="text-text-primary">{e.oldValue}</span>
                  </>
                )}
                {e.newValue && (
                  <>
                    {' '}to <span className="text-text-primary">{e.newValue}</span>
                  </>
                )}
              </>
            )}
          </span>
          <span className="ml-auto shrink-0 text-[length:var(--text-caption)] text-text-subtle">
            {new Date(e.$createdAt).toLocaleString()}
          </span>
        </div>
      ))}
    </div>
  );
}

function Discussion({ itemId, comments }: { itemId: string; comments: Comment[] }) {
  const qc = useQueryClient();
  const [body, setBody] = useState('');
  const [composing, setComposing] = useState(false);

  const add = useMutation({
    mutationFn: () => api.post(`/plan/items/${itemId}/comments`, { body: body.trim() }),
    onSuccess: () => {
      setBody('');
      setComposing(false);
      qc.invalidateQueries({ queryKey: ['plan-comments', itemId] });
    },
    onError: (e) => toast.error(friendlyError(e)),
  });

  return (
    <div className="flex flex-col gap-3">
      {comments.map((c) => (
        <div key={c.$id} className="rounded-[var(--radius)] border border-border bg-surface px-4 py-3">
          <div className="prose-plan text-[length:var(--text-body)] text-text-primary">
            <Markdown remarkPlugins={[remarkGfm]}>{c.body}</Markdown>
          </div>
          <span className="mt-1 block text-[length:var(--text-caption)] text-text-subtle">
            {new Date(c.$createdAt).toLocaleString()}
          </span>
        </div>
      ))}

      {comments.length === 0 && !composing && (
        <p className="text-[length:var(--text-body)] text-text-subtle">No comments yet.</p>
      )}

      {composing ? (
        <div className="flex flex-col gap-2">
          <RichTextEditor
            value={body}
            onChange={setBody}
            minRows={4}
            placeholder="Add a comment — why this was decided, what it depends on…"
          />
          {/* Actions right, primary last: the eye ends where the decision is. */}
          <div className="flex justify-end gap-2">
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                setBody('');
                setComposing(false);
              }}
            >
              Cancel
            </Button>
            <Button size="sm" onClick={() => add.mutate()} disabled={!body.trim() || add.isPending}>
              Comment
            </Button>
          </div>
        </div>
      ) : (
        <button
          onClick={() => setComposing(true)}
          className="w-full rounded-[var(--radius)] border border-field-border bg-field-fill px-3 py-2 text-left text-[length:var(--text-body)] text-text-subtle transition-colors hover:border-text-subtle"
        >
          Add a comment…
        </button>
      )}
    </div>
  );
}

/*
 * What people said, and what changed.
 *
 * Two different questions — "why is this being done this way" and "who moved
 * it to blocked on Tuesday" — so they are tabs rather than one stream. The
 * composer stays closed until there is something to say: a permanently open
 * editor is the loudest thing on a page that is mostly for reading.
 */

/*
 * Impact and urgency, anchored.
 *
 * A level with no anchor is a vibe: two people pick "medium" for different
 * reasons and the scale stops sorting anything. And the anchors differ by
 * kind — fixing something broken has workarounds and blocked people, building
 * something wanted has neither. Asking a change whether a workaround exists
 * is how a field gets answered at random.
 */
function PriorityAssessment({ itemId, item }: { itemId: string; item?: Item }) {
  const qc = useQueryClient();

  const { data: meta } = useQuery({
    queryKey: ['plan-meta'],
    queryFn: async () =>
      (await api.get('/plan/meta')).data as {
        hints: Record<string, { impact: Record<string, string>; urgency: Record<string, string> }>;
      },
  });

  const rate = useMutation({
    mutationFn: (body: { impact: number; urgency: number }) =>
      api.post(`/plan/items/${itemId}/rate`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['plan-item', itemId] });
      qc.invalidateQueries({ queryKey: ['plan-items'] });
      qc.invalidateQueries({ queryKey: ['plan-activity', itemId] });
    },
    onError: (e) => toast.error(friendlyError(e)),
  });

  const hints = meta?.hints?.[item?.kind ?? 'change'];

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-baseline gap-2">
        <h2 className="text-[length:var(--text-control)] font-medium text-text-primary">
          Priority
        </h2>
        <span className="text-[length:var(--text-caption)] text-text-subtle">
          {item?.priorityIsManual
            ? 'set directly'
            : item?.impact
              ? `impact ${LEVEL_NAME[item.impact]} · urgency ${LEVEL_NAME[item.urgency ?? 1]} → ${item.priority}`
              : 'not assessed'}
        </span>
      </div>

      <div className="grid gap-4 rounded-[var(--radius)] border border-border bg-surface p-4 sm:grid-cols-2">
        <AnchoredChoice
          label="Impact"
          caption="How much it matters"
          value={item?.impact ?? 0}
          onChange={(impact) => rate.mutate({ impact, urgency: item?.urgency ?? 2 })}
          hints={hints?.impact}
        />
        <AnchoredChoice
          label="Urgency"
          caption="How soon it is needed"
          value={item?.urgency ?? 0}
          onChange={(urgency) => rate.mutate({ impact: item?.impact ?? 2, urgency })}
          hints={hints?.urgency}
        />
      </div>
    </section>
  );
}
