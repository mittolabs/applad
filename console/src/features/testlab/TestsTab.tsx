import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Circle, ShieldOff, Tag, Video } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/empty-state';
import { FormDialog, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';

/*
 * The catalogue: one row per behaviour, however it got here.
 *
 * Recorded flows appear the moment they are saved; tests written in the repo
 * appear the first time a run reports them. Both then carry a history, which
 * is the thing worth looking at — a test that has failed once looks nothing
 * like one that has been red for a week.
 */

interface Test {
  $id: string;
  name: string;
  suiteName: string;
  source: string;
  tags: string[];
  quarantined: boolean;
  lastStatus?: string;
  history?: string[];
}

const DOT: Record<string, string> = {
  passed: '#22C55E',
  failed: '#EF4444',
  errored: '#F59E0B',
  skipped: '#4B5563',
};

export function TestsTab({ projectId, onRecord }: { projectId?: string; onRecord: () => void }) {
  const qc = useQueryClient();
  const [tagging, setTagging] = useState<Test | null>(null);
  const [tagText, setTagText] = useState('');

  const { data: tests = [] } = useQuery({
    queryKey: ['tests', projectId],
    queryFn: async () => ((await api.get('/tests/tests')).data as { tests: Test[] }).tests ?? [],
  });

  const refresh = () => qc.invalidateQueries({ queryKey: ['tests', projectId] });

  const saveTags = async () => {
    if (!tagging) return;
    const tags = tagText
      .split(',')
      .map((t) => t.trim().replace(/^@/, ''))
      .filter(Boolean);
    try {
      await api.put(`/tests/tests/${tagging.$id}/tags`, { tags });
      setTagging(null);
      refresh();
    } catch (e) {
      toast.error(friendlyError(e));
    }
  };

  const toggleQuarantine = async (t: Test) => {
    try {
      await api.put(`/tests/tests/${t.$id}/quarantine`, { quarantined: !t.quarantined });
      toast.success(t.quarantined ? 'Back in the verdict' : 'Quarantined — it runs but no longer blocks');
      refresh();
    } catch (e) {
      toast.error(friendlyError(e));
    }
  };

  if (tests.length === 0) {
    return (
      <EmptyState
        icon={Video}
        title="No tests yet"
        subtitle="Record a flow by using your app, or run a suite you already have — anything a run reports appears here."
        actionLabel="Record a flow"
        onAction={onRecord}
      />
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <span className="text-[length:var(--text-label)] text-text-secondary">
          {tests.length} {tests.length === 1 ? 'test' : 'tests'}
        </span>
        <Button onClick={onRecord}>
          <Circle size={12} fill="currentColor" />
          Record a flow
        </Button>
      </div>

      <div className="flex flex-col gap-2">
        {tests.map((t) => (
          <div
            key={t.$id}
            className="flex items-center gap-4 rounded-[var(--radius)] border border-border bg-surface px-4 py-3"
          >
            <span
              className="h-2 w-2 shrink-0 rounded-full"
              style={{ backgroundColor: DOT[t.lastStatus ?? ''] ?? 'var(--text-subtle)' }}
            />

            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className="truncate text-[length:var(--text-body)] text-text-primary">{t.name}</span>
                {t.quarantined && (
                  <span
                    className="shrink-0 rounded-[var(--radius-6)] px-1.5 py-0.5 text-[length:var(--text-caption)]"
                    style={{ backgroundColor: '#F59E0B22', color: '#FBBF24' }}
                  >
                    quarantined
                  </span>
                )}
                {t.tags.map((tag) => (
                  <span
                    key={tag}
                    className="shrink-0 rounded-[var(--radius-6)] bg-fill px-1.5 py-0.5 text-[length:var(--text-caption)] text-text-muted"
                  >
                    @{tag}
                  </span>
                ))}
              </div>
              <div className="mt-0.5 text-[length:var(--text-caption)] text-text-subtle">
                {t.source === 'recorded' ? 'recorded' : t.suiteName}
              </div>
            </div>

            {/* Ten runs, newest on the right: the shape tells you whether this
                is broken or merely unreliable. */}
            <div className="flex shrink-0 items-center gap-[3px]">
              {[...(t.history ?? [])].reverse().map((h, i) => (
                <span
                  key={i}
                  title={h}
                  className="h-3.5 w-1.5 rounded-[1px]"
                  style={{ backgroundColor: DOT[h] ?? 'var(--fill)' }}
                />
              ))}
            </div>

            <Button
              variant="ghost"
              onClick={() => {
                setTagging(t);
                setTagText(t.tags.join(', '));
              }}
              aria-label="Tags"
            >
              <Tag size={13} />
            </Button>
            <Button variant="ghost" onClick={() => toggleQuarantine(t)} aria-label="Quarantine">
              <ShieldOff size={13} style={{ color: t.quarantined ? '#FBBF24' : undefined }} />
            </Button>
          </div>
        ))}
      </div>

      <FormDialog
        open={!!tagging}
        onOpenChange={(o) => !o && setTagging(null)}
        title="Tags"
        subtitle="Tags are how suites select tests: smoke, checkout, slow."
        submitLabel="Save"
        onSubmit={saveTags}
      >
        <TextField
          label="Tags"
          value={tagText}
          onChange={(e) => setTagText(e.target.value)}
          placeholder="smoke, checkout"
          autoFocus
          hint="Comma separated."
        />
      </FormDialog>
    </div>
  );
}
