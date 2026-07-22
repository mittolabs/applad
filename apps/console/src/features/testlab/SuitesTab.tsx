import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Clock, Play, Plus, Rocket, Trash2 } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/empty-state';
import { FormDialog, FormField, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';

/*
 * Suites: which tests, and when.
 *
 * A suite is a selection over the catalogue plus its triggers — "smoke, on
 * every deploy". The target lives here rather than on the test, so the same
 * suite checks main and a branch instead of being duplicated per environment.
 */

interface Suite {
  $id: string;
  name: string;
  tags: string[];
  defaultTarget: string;
  runOnDeploy: boolean;
  cron?: string;
  testCount: number;
}

export function SuitesTab({ projectId }: { projectId?: string }) {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<Suite | null>(null);
  const [creating, setCreating] = useState(false);
  const [running, setRunning] = useState<string | null>(null);
  const [runTarget, setRunTarget] = useState<Suite | null>(null);
  const [targetText, setTargetText] = useState('');

  const [name, setName] = useState('');
  const [tags, setTags] = useState('');
  const [target, setTarget] = useState('');
  const [onDeploy, setOnDeploy] = useState(false);
  const [cron, setCron] = useState('');

  const { data: suites = [] } = useQuery({
    queryKey: ['test-suites', projectId],
    queryFn: async () => ((await api.get('/tests/suites')).data as { suites: Suite[] }).suites ?? [],
  });

  const refresh = () => qc.invalidateQueries({ queryKey: ['test-suites', projectId] });

  const openCreate = () => {
    setEditing(null);
    setName('');
    setTags('');
    setTarget('');
    setOnDeploy(false);
    setCron('');
    setCreating(true);
  };

  const openEdit = (s: Suite) => {
    setEditing(s);
    setName(s.name);
    setTags(s.tags.join(', '));
    setTarget(s.defaultTarget);
    setOnDeploy(s.runOnDeploy);
    setCron(s.cron ?? '');
    setCreating(true);
  };

  const save = async () => {
    const body = {
      name: name.trim(),
      tags: tags.split(',').map((t) => t.trim().replace(/^@/, '')).filter(Boolean),
      defaultTarget: target.trim(),
      runOnDeploy: onDeploy,
      cron: cron.trim(),
    };
    try {
      if (editing) await api.put(`/tests/suites/${editing.$id}`, body);
      else await api.post('/tests/suites', body);
      setCreating(false);
      refresh();
    } catch (e) {
      toast.error(friendlyError(e));
    }
  };

  const run = async (s: Suite, override?: string) => {
    setRunning(s.$id);
    try {
      await api.post(`/tests/suites/${s.$id}/run`, { actor: 'console', target: override ?? '' });
      toast.success(`Running ${s.name}`);
      setRunTarget(null);
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setRunning(null);
    }
  };

  const remove = async (s: Suite) => {
    try {
      await api.delete(`/tests/suites/${s.$id}`);
      refresh();
    } catch (e) {
      toast.error(friendlyError(e));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <span className="text-[length:var(--text-label)] text-text-secondary">
          {suites.length} {suites.length === 1 ? 'suite' : 'suites'}
        </span>
        <Button onClick={openCreate}>
          <Plus size={14} />
          New suite
        </Button>
      </div>

      {suites.length === 0 ? (
        <EmptyState
          title="No suites yet"
          subtitle="A suite selects tests by tag and says when they run — on every deploy, on a schedule, or when you ask."
          actionLabel="New suite"
          onAction={openCreate}
        />
      ) : (
        <div className="flex flex-col gap-2">
          {suites.map((s) => (
            <div
              key={s.$id}
              className="flex items-center gap-4 rounded-[var(--radius)] border border-border bg-surface px-4 py-3"
            >
              <button onClick={() => openEdit(s)} className="min-w-0 flex-1 text-left">
                <div className="text-[length:var(--text-body)] text-text-primary">{s.name}</div>
                <div className="mt-0.5 flex items-center gap-2 text-[length:var(--text-caption)] text-text-muted">
                  <span>
                    {s.testCount} {s.testCount === 1 ? 'test' : 'tests'}
                    {s.tags.length > 0 && ` · ${s.tags.map((t) => '@' + t).join(' ')}`}
                  </span>
                  {s.runOnDeploy && (
                    <span className="flex items-center gap-1" style={{ color: 'var(--color-accent)' }}>
                      <Rocket size={11} />
                      on deploy
                    </span>
                  )}
                  {s.cron && (
                    <span className="flex items-center gap-1">
                      <Clock size={11} />
                      {s.cron}
                    </span>
                  )}
                </div>
              </button>

              <Button
                variant="outline"
                loading={running === s.$id}
                onClick={() => {
                  setRunTarget(s);
                  setTargetText(s.defaultTarget);
                }}
              >
                <Play size={13} />
                Run
              </Button>
              <Button variant="ghost" onClick={() => remove(s)} aria-label="Delete suite">
                <Trash2 size={13} />
              </Button>
            </div>
          ))}
        </div>
      )}

      <FormDialog
        open={creating}
        onOpenChange={setCreating}
        title={editing ? 'Edit suite' : 'New suite'}
        subtitle="Which tests, and when they run."
        submitLabel="Save"
        submitDisabled={!name.trim()}
        onSubmit={save}
        width={520}
      >
        <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} placeholder="smoke" autoFocus />
        <TextField
          label="Tags"
          value={tags}
          onChange={(e) => setTags(e.target.value)}
          placeholder="smoke"
          hint="Comma separated. Leave empty to include every test."
        />
        <TextField
          label="Default target"
          value={target}
          onChange={(e) => setTarget(e.target.value)}
          placeholder="http://applad-site-the-range"
          hint="What it runs against unless a run says otherwise."
        />
        <FormField label="Triggers">
          <label className="flex items-center gap-2 text-[length:var(--text-label)] text-text-secondary">
            <input type="checkbox" checked={onDeploy} onChange={(e) => setOnDeploy(e.target.checked)} />
            Run after every successful deploy
          </label>
        </FormField>
        <TextField
          label="Schedule"
          value={cron}
          onChange={(e) => setCron(e.target.value)}
          placeholder="0 6 * * *"
          hint="Standard 5-field cron. Prefix with CRON_TZ=Africa/Nairobi for a timezone. Leave empty for no schedule."
        />
      </FormDialog>

      <FormDialog
        open={!!runTarget}
        onOpenChange={(o) => !o && setRunTarget(null)}
        title={`Run ${runTarget?.name ?? ''}`}
        subtitle="Point it wherever you need — main, a branch, anywhere reachable."
        submitLabel="Run"
        onSubmit={() => runTarget && run(runTarget, targetText.trim())}
      >
        <TextField
          label="Target"
          value={targetText}
          onChange={(e) => setTargetText(e.target.value)}
          placeholder="http://applad-site-the-range"
          autoFocus
        />
      </FormDialog>
    </div>
  );
}
