import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Hash, Plus, Search } from 'lucide-react';
import { api } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { FormDialog, TextField } from '@/components/form-dialog';
import { ErrorState } from '@/components/error-state';

interface Topic extends Record<string, unknown> {
  name?: string;
  subscribers?: unknown[];
}

export function TopicsTab({ projectId }: { projectId: string | undefined }) {
  const [search, setSearch] = useState('');
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');

  const query = useQuery({
    queryKey: ['/messaging/topics', projectId],
    queryFn: async () => {
      const res = await api.get('/messaging/topics');
      return (res.data as { topics?: Topic[] }).topics ?? [];
    },
  });

  const create = useMutation({
    mutationFn: async () => {
      await api.post('/messaging/topics', { name: name.trim() });
    },
    onSuccess: () => {
      setCreating(false);
      setName('');
      void query.refetch();
    },
  });

  const topics = query.data ?? [];
  const q = search.trim().toLowerCase();
  const filtered = q
    ? topics.filter((t) => String(t.name ?? '').toLowerCase().includes(q))
    : topics;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <div className="relative w-[280px]">
          <Search
            size={15}
            className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-subtle"
          />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search topics"
            className="pl-9"
          />
        </div>
        <span className="flex-1" />
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus size={15} />
          Create topic
        </Button>
      </div>

      {query.error ? (
        <ErrorState error={query.error} onRetry={() => void query.refetch()} />
      ) : query.isLoading ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-muted">
          Loading…
        </div>
      ) : topics.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-[var(--radius)] border border-border bg-surface px-6 py-16 text-center">
          <div className="text-[length:var(--text-subhead)] font-medium text-text-primary">
            Create your first topic
          </div>
          <div className="mt-2 text-[length:var(--text-body)] text-text-secondary">
            Group targets and broadcast messages to all of them at once.
          </div>
          <Button variant="outline" className="mt-5" onClick={() => setCreating(true)}>
            Create topic
          </Button>
        </div>
      ) : filtered.length === 0 ? (
        <div className="py-16 text-center text-[length:var(--text-body)] text-text-muted">
          No topics match "{search}"
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {filtered.map((t, i) => {
            const subs = Array.isArray(t.subscribers) ? t.subscribers.length : 0;
            return (
              <div
                key={String(t.$id ?? t.name ?? i)}
                className="flex items-center gap-2 rounded-[var(--radius)] border border-border bg-surface p-4"
              >
                <Hash size={14} className="text-text-subtle" />
                <span className="text-[length:var(--text-control)] text-text-primary">
                  {String(t.name ?? '')}
                </span>
                <span className="flex-1" />
                <span className="text-[length:var(--text-label)] text-text-subtle">
                  {subs} subscriber{subs === 1 ? '' : 's'}
                </span>
              </div>
            );
          })}
        </div>
      )}

      <FormDialog
        open={creating}
        onOpenChange={(o) => {
          setCreating(o);
          if (!o) setName('');
        }}
        title="Create topic"
        subtitle="Group targets for broadcast messaging"
        submitLabel="Create"
        loading={create.isPending}
        submitDisabled={!name.trim()}
        onSubmit={() => create.mutate()}
      >
        <TextField
          label="Topic name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="my-topic"
          autoFocus
        />
      </FormDialog>
    </div>
  );
}
