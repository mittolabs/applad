import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Globe, Smartphone, Monitor } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { FormDialog, FormField, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';

/*
 * Choosing what to record against.
 *
 * Deployed sites are offered by name rather than asked for as a URL, because
 * the thing somebody wants to test is almost always something they just
 * deployed. Device targets are listed but not yet selectable: an Android
 * emulator needs virtualisation the host may not have, and an iOS simulator
 * needs a Mac, so neither can be promised from here.
 */

export interface StudioSession {
  $id: string;
  target: string;
  status: string;
}

type Platform = 'web' | 'android' | 'ios';

export function RecordDialog({
  open,
  onOpenChange,
  onStarted,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onStarted: (s: StudioSession) => void;
}) {
  const { projectId } = useParams<{ projectId: string }>();
  const [platform, setPlatform] = useState<Platform>('web');
  const [target, setTarget] = useState('');
  const [busy, setBusy] = useState(false);

  // The project's deployed sites, so a target is picked rather than typed.
  const { data: sites = [] } = useQuery({
    queryKey: ['deploy-targets', projectId],
    queryFn: async () => {
      const res = await api.get('/deploy/targets');
      return ((res.data as { targets?: Record<string, unknown>[] }).targets ?? []).filter(
        (t) => String(t.type ?? 'web') === 'web',
      );
    },
    enabled: open,
  });

  useEffect(() => {
    if (open) {
      setPlatform('web');
      setTarget('');
      setBusy(false);
    }
  }, [open]);

  const start = async () => {
    setBusy(true);
    try {
      const res = await api.post('/studio/sessions', { target: target.trim() });
      onOpenChange(false);
      onStarted(res.data as StudioSession);
    } catch (e) {
      toast.error(friendlyError(e));
      setBusy(false);
    }
  };

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Record a flow"
      subtitle="Use your app; Applad writes the test from what you do."
      submitLabel="Start recording"
      loading={busy}
      submitDisabled={platform !== 'web' || !target.trim()}
      onSubmit={start}
      width={540}
    >
      <FormField label="Platform">
        <div className="flex gap-2">
          {(
            [
              { id: 'web', label: 'Web', icon: Globe, ready: true },
              { id: 'android', label: 'Android', icon: Smartphone, ready: false },
              { id: 'ios', label: 'iOS', icon: Monitor, ready: false },
            ] as const
          ).map((p) => {
            const Icon = p.icon;
            const selected = platform === p.id;
            return (
              <button
                key={p.id}
                type="button"
                onClick={() => setPlatform(p.id)}
                className="flex flex-1 flex-col items-center gap-1.5 rounded-[var(--radius)] border py-3 text-[length:var(--text-caption)] transition-colors"
                style={
                  selected
                    ? {
                        borderColor: 'color-mix(in srgb, var(--color-accent) 40%, transparent)',
                        backgroundColor: 'color-mix(in srgb, var(--color-accent) 10%, transparent)',
                        color: 'var(--color-accent)',
                      }
                    : { borderColor: 'var(--border)', color: 'var(--text-muted)' }
                }
              >
                <Icon size={20} />
                {p.label}
                {!p.ready && (
                  <span className="text-[length:var(--text-caption)] text-text-subtle">soon</span>
                )}
              </button>
            );
          })}
        </div>
      </FormField>

      {platform !== 'web' ? (
        <p className="text-[length:var(--text-caption)] text-text-muted">
          {platform === 'android'
            ? 'Android recording needs an emulator, which requires hardware virtualisation on the host.'
            : 'iOS recording needs a simulator, which only runs on Apple hardware.'}{' '}
          Recorded steps compile to Maestro already, so this is a runner away.
        </p>
      ) : (
        <>
          {sites.length > 0 && (
            <FormField label="Deployed sites">
              <div className="flex flex-wrap gap-1.5">
                {sites.map((s) => {
                  const url = `http://applad-site-${slug(String(s.name ?? ''))}`;
                  return (
                    <button
                      key={String(s.$id)}
                      type="button"
                      onClick={() => setTarget(url)}
                      className="rounded-[var(--radius-6)] border border-border px-2.5 py-1 text-[length:var(--text-caption)] text-text-muted transition-colors hover:text-text-primary"
                    >
                      {String(s.name)}
                    </button>
                  );
                })}
              </div>
            </FormField>
          )}
          <TextField
            label="URL"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            placeholder="http://applad-site-the-range"
            hint="A deployed app is reached by its container name from inside Applad; any URL the browser can reach also works."
          />
        </>
      )}
    </FormDialog>
  );
}

// Mirrors the subdomain the deploy executor derives from a target's name.
function slug(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '');
}
