import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, Bug, GitCommitHorizontal, Plus, Trash2, Check } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { toast } from '@/components/toast';
import { cn } from '@/lib/utils';

/*
 * One item, and everything agreed about it.
 *
 * Acceptance criteria are the constraints the work was agreed against, and
 * they are the reason this page exists: they are written before anybody has
 * decided how the behaviour will be expressed, and a Rule in a specification
 * is one of them made executable. A criterion showing no rule is visibly
 * still an intention, which is the state worth being able to see.
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
  createdAt?: string;
  $createdAt: string;
}

interface Item {
  $id: string;
  title: string;
  body: string;
  status: string;
  priority: string;
  kind: string;
  labels: string[];
  links: { $id: string; kind: string; ref: string }[];
}

export function ItemDetail({ itemId, onBack }: { itemId: string; onBack: () => void }) {
  const qc = useQueryClient();

  const item = useQuery({
    queryKey: ['plan-item', itemId],
    queryFn: async () => (await api.get(`/plan/items/${itemId}`)).data as Item,
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

  const refreshCriteria = () => qc.invalidateQueries({ queryKey: ['plan-criteria', itemId] });

  const [newCriterion, setNewCriterion] = useState('');
  const [newComment, setNewComment] = useState('');

  const addCriterion = useMutation({
    mutationFn: () => api.post(`/plan/items/${itemId}/criteria`, { text: newCriterion.trim() }),
    onSuccess: () => {
      setNewCriterion('');
      refreshCriteria();
    },
    onError: (e) => toast.error(friendlyError(e)),
  });

  const setMet = useMutation({
    mutationFn: ({ id, met }: { id: string; met: boolean }) =>
      api.patch(`/plan/criteria/${id}`, { met }),
    onSuccess: refreshCriteria,
    onError: (e) => toast.error(friendlyError(e)),
  });

  const removeCriterion = useMutation({
    mutationFn: (id: string) => api.delete(`/plan/criteria/${id}`),
    onSuccess: refreshCriteria,
    onError: (e) => toast.error(friendlyError(e)),
  });

  const addComment = useMutation({
    mutationFn: () => api.post(`/plan/items/${itemId}/comments`, { body: newComment.trim() }),
    onSuccess: () => {
      setNewComment('');
      qc.invalidateQueries({ queryKey: ['plan-comments', itemId] });
    },
    onError: (e) => toast.error(friendlyError(e)),
  });

  const data = item.data;
  const list = criteria.data ?? [];
  const specified = list.filter((c) => c.specRef.trim() !== '').length;

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <button
          onClick={onBack}
          className="mb-1 flex items-center gap-1 text-[length:var(--text-label)] text-text-muted transition-colors hover:text-text-primary"
        >
          <ChevronLeft size={14} />
          Plan
        </button>
        <div className="flex items-center gap-2">
          {data?.kind === 'defect' ? (
            <Bug size={18} style={{ color: '#EF4444' }} />
          ) : (
            <GitCommitHorizontal size={18} className="text-text-muted" />
          )}
          <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
            {data?.title ?? 'Item'}
          </h1>
        </div>
        {data?.body && (
          <p className="mt-2 max-w-[70ch] whitespace-pre-wrap text-[length:var(--text-body)] text-text-secondary">
            {data.body}
          </p>
        )}
      </div>

      <section className="flex flex-col gap-3">
        <div className="flex items-baseline gap-2">
          <h2 className="text-[length:var(--text-control)] font-medium text-text-primary">
            Acceptance criteria
          </h2>
          {list.length > 0 && (
            /* Two counts, because they say different things: what is agreed,
               and how much of it has become behaviour anything can check. */
            <span className="text-[length:var(--text-caption)] text-text-subtle">
              {specified} of {list.length} specified
            </span>
          )}
        </div>

        <div className="flex flex-col gap-1.5">
          {list.map((c) => (
            <div
              key={c.$id}
              className="group flex items-start gap-3 rounded-[var(--radius)] border border-border bg-surface px-4 py-3"
            >
              <button
                onClick={() => setMet.mutate({ id: c.$id, met: !c.met })}
                disabled={c.specRef.trim() !== ''}
                title={
                  c.specRef
                    ? 'A specification decides this one — the rule it became is checked by its own tests.'
                    : 'Not yet expressed as behaviour; mark by hand until it is.'
                }
                className={cn(
                  'mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-[3px] border transition-colors',
                  c.met || c.specRef
                    ? 'border-transparent bg-[var(--color-accent)] text-white'
                    : 'border-field-border hover:border-text-muted',
                  c.specRef && 'cursor-default opacity-70',
                )}
              >
                {(c.met || !!c.specRef) && <Check size={11} />}
              </button>

              <span className="flex-1 text-[length:var(--text-body)] text-text-primary">
                {c.text}
              </span>

              {c.specRef ? (
                <span
                  className="max-w-[280px] truncate rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-caption)]"
                  style={{
                    backgroundColor: 'color-mix(in srgb, var(--color-accent) 12%, transparent)',
                    color: 'var(--color-accent)',
                  }}
                  title={c.specRef}
                >
                  {c.specRef}
                </span>
              ) : (
                <span className="text-[length:var(--text-caption)] text-text-subtle">
                  not specified
                </span>
              )}

              <button
                onClick={() => removeCriterion.mutate(c.$id)}
                className="opacity-0 transition-opacity group-hover:opacity-100"
                aria-label="Remove criterion"
              >
                <Trash2 size={13} className="text-text-subtle hover:text-[#EF4444]" />
              </button>
            </div>
          ))}

          {list.length === 0 && !criteria.isLoading && (
            <p className="text-[length:var(--text-body)] text-text-subtle">
              Nothing agreed yet. Criteria are the constraints this work has to satisfy — they
              become the rules of its specification.
            </p>
          )}
        </div>

        <div className="flex gap-2">
          <input
            value={newCriterion}
            onChange={(e) => setNewCriterion(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && newCriterion.trim()) addCriterion.mutate();
            }}
            placeholder="e.g. A discount never takes an order below zero"
            className="flex-1 rounded-[var(--radius)] border border-field-border bg-field-fill px-3 py-2 text-[length:var(--text-body)] text-text-primary placeholder:text-text-subtle"
          />
          <Button
            size="sm"
            onClick={() => addCriterion.mutate()}
            disabled={!newCriterion.trim() || addCriterion.isPending}
          >
            <Plus size={14} />
            Add
          </Button>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-[length:var(--text-control)] font-medium text-text-primary">
          Discussion
        </h2>

        {(comments.data ?? []).map((c) => (
          <div
            key={c.$id}
            className="rounded-[var(--radius)] border border-border bg-surface px-4 py-3"
          >
            <p className="whitespace-pre-wrap text-[length:var(--text-body)] text-text-primary">
              {c.body}
            </p>
            <span className="mt-1 block text-[length:var(--text-caption)] text-text-subtle">
              {new Date(c.$createdAt).toLocaleString()}
            </span>
          </div>
        ))}

        {(comments.data ?? []).length === 0 && !comments.isLoading && (
          <p className="text-[length:var(--text-body)] text-text-subtle">
            No discussion yet.
          </p>
        )}

        <div className="flex gap-2">
          <textarea
            value={newComment}
            onChange={(e) => setNewComment(e.target.value)}
            rows={2}
            placeholder="Add a note — why this was decided, what it depends on…"
            className="flex-1 resize-y rounded-[var(--radius)] border border-field-border bg-field-fill px-3 py-2 text-[length:var(--text-body)] text-text-primary placeholder:text-text-subtle"
          />
          <Button
            size="sm"
            onClick={() => addComment.mutate()}
            disabled={!newComment.trim() || addComment.isPending}
          >
            Post
          </Button>
        </div>
      </section>
    </div>
  );
}
