import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  AlertTriangle,
  ChevronRight,
  ExternalLink,
  GitBranch,
  Link2,
  Plus,
  Rocket,
  Terminal,
  Upload,
  UploadCloud,
} from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { api, friendlyError } from '@/api/client';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { StatusChip } from '@/components/status-chip';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { FormDialog, SelectField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { ShellSnippets } from './ShellSnippets';

// ── Formatters (port _DeploymentReleasesView statics) ─────────────────────────

function fmtDur(ms?: number): string {
  if (!ms) return '—';
  const s = Math.floor(ms / 1000);
  if (s < 60) return `${s}s`;
  return `${Math.floor(s / 60)}m ${s % 60}s`;
}

function fmtSize(bytes?: number): string {
  if (!bytes) return '—';
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)}KB`;
  return `${(bytes / 1048576).toFixed(1)}MB`;
}

const MONTHS = ['', 'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
function fmtDateLong(raw: unknown): string {
  if (!raw) return '—';
  const d = new Date(String(raw));
  if (Number.isNaN(d.getTime())) return '—';
  return `${MONTHS[d.getMonth() + 1]} ${d.getDate()}, ${d.getFullYear()}`;
}

function cap(s: string): string {
  return s ? s[0].toUpperCase() + s.slice(1) : s;
}

const PER_PAGE = 12;

// ── Entry ─────────────────────────────────────────────────────────────────────

export function PlatformDeploymentTab({
  platform,
  projectId,
  onChange,
}: {
  platform: Row;
  projectId: string;
  onChange: (next: Row) => void;
}) {
  const platformId = String(platform['$id'] ?? platform['id'] ?? '');
  const linkedTargetId = String(platform['deployTargetId'] ?? '');
  const [connecting, setConnecting] = useState(false);

  const targetQuery = useQuery({
    queryKey: ['deploy-target', linkedTargetId],
    enabled: !!linkedTargetId,
    queryFn: async () => {
      const res = await api.get(`/deploy/targets/${linkedTargetId}`);
      return res.data as Record<string, unknown>;
    },
    retry: false,
  });

  const setTargetId = async (targetId: string) => {
    await api.patch(`/projects/${projectId}/platforms/${platformId}`, { deployTargetId: targetId });
    onChange({ ...platform, deployTargetId: targetId });
  };

  if (!linkedTargetId) {
    return (
      <>
        <NotConnected onConnect={() => setConnecting(true)} />
        <ConnectDeploymentDialog
          open={connecting}
          onOpenChange={setConnecting}
          projectId={projectId}
          onSelect={async (id) => {
            try {
              await setTargetId(id);
              setConnecting(false);
              toast.success('Deployment connected');
            } catch (e) {
              toast.error(friendlyError(e));
            }
          }}
        />
      </>
    );
  }

  if (targetQuery.isLoading) {
    return <div className="py-16 text-center text-[length:var(--text-body)] text-text-subtle">Loading…</div>;
  }

  if (targetQuery.isError || !targetQuery.data) {
    return (
      <TargetNotFound
        onRemove={async () => {
          try {
            await setTargetId('');
            toast.success('Connection removed');
          } catch (e) {
            toast.error(friendlyError(e));
          }
        }}
      />
    );
  }

  return (
    <ReleasesView
      targetId={linkedTargetId}
      onDisconnect={async () => {
        try {
          await setTargetId('');
          toast.success('Deployment disconnected');
        } catch (e) {
          toast.error(friendlyError(e));
        }
      }}
    />
  );
}

// ── Empty / error states ───────────────────────────────────────────────────────

function NotConnected({ onConnect }: { onConnect: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
      <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-[var(--radius-12)] border border-border bg-surface text-text-subtle">
        <Rocket size={24} />
      </div>
      <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
        No deployment connected
      </div>
      <div className="mt-1.5 max-w-sm text-[length:var(--text-body)] text-text-secondary">
        Connect a deploy target to enable builds and track deployment status directly from this platform.
      </div>
      <Button className="mt-5" onClick={onConnect}>
        <Link2 size={14} />
        Connect deployment
      </Button>
    </div>
  );
}

function TargetNotFound({ onRemove }: { onRemove: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
      <AlertTriangle size={28} className="text-text-subtle" />
      <div className="mt-3 text-[length:var(--text-control)] font-medium text-text-primary">
        Deployment target not found
      </div>
      <div className="mt-1 text-[length:var(--text-label)] text-text-secondary">
        It may have been deleted.
      </div>
      <Button variant="outline" className="mt-4" onClick={onRemove}>
        Remove connection
      </Button>
    </div>
  );
}

// ── Releases view ───────────────────────────────────────────────────────────────

const RELEASE_COLUMNS: DataTableColumn[] = [
  { key: '$id', label: 'Deployment ID', flex: 3 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'durationMs', label: 'Build duration', flex: 2 },
  { key: 'size', label: 'Total size', flex: 2 },
  { key: 'source', label: 'Source', flex: 2 },
  { key: '$createdAt', label: 'Updated', flex: 2 },
];

function ReleasesView({ targetId, onDisconnect }: { targetId: string; onDisconnect: () => void }) {
  const [page, setPage] = useState(1);
  const [dialog, setDialog] = useState<'git' | 'cli' | 'manual' | null>(null);

  const query = useQuery({
    queryKey: ['deploy-target-releases', targetId, page],
    queryFn: async () => {
      const [statsRes, relRes] = await Promise.all([
        api.get(`/deploy/targets/${targetId}/stats`).then((r) => r.data).catch(() => ({})),
        api
          .get(`/deploy/targets/${targetId}/releases`, {
            params: { limit: PER_PAGE, offset: (page - 1) * PER_PAGE },
          })
          .then((r) => r.data)
          .catch(() => ({})),
      ]);
      const rel = (relRes ?? {}) as Record<string, unknown>;
      const releases = ((rel['releases'] ?? rel['executions']) as Row[] | undefined) ?? [];
      return {
        stats: (statsRes ?? {}) as Record<string, unknown>,
        releases,
        total: Number(rel['total'] ?? releases.length),
      };
    },
  });

  const stats = query.data?.stats ?? {};
  const releases = query.data?.releases ?? [];
  const total = query.data?.total ?? 0;

  const statItems: [string, string][] = [
    ['Builds', String(stats['totalBuilds'] ?? total)],
    ['Successful', String(stats['successful'] ?? '—')],
    ['Failed', String(stats['failed'] ?? '—')],
    ['Avg build time', fmtDur(stats['avgBuildTimeMs'] as number | undefined)],
  ];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-6">
        {query.data &&
          statItems.map(([label, value]) => (
            <div key={label} className="flex flex-col">
              <span className="text-[length:var(--text-caption)] text-text-subtle">{label}</span>
              <span className="text-[length:var(--text-control)] font-semibold text-text-primary">
                {value}
              </span>
            </div>
          ))}
        <div className="ml-auto flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={onDisconnect}>
            Disconnect
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm">
                <Plus size={14} />
                Create deployment
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => setDialog('git')}>
                <GitBranch size={14} />
                Git
                <span className="ml-auto rounded-[var(--radius-sm)] bg-[color-mix(in_srgb,var(--color-accent-2)_12%,transparent)] px-1.5 py-0.5 text-[length:var(--text-2xs)] text-[var(--color-accent-2)]">
                  Recommended
                </span>
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => setDialog('cli')}>
                <Terminal size={14} />
                CLI
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => setDialog('manual')}>
                <Upload size={14} />
                Manual
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <DataTable
        columns={RELEASE_COLUMNS}
        rows={releases}
        getCellValue={(row, key) => {
          switch (key) {
            case '$id':
              return String(row['$id'] ?? '');
            case 'status':
              return String(row['status'] ?? '');
            case 'durationMs':
              return fmtDur(row['durationMs'] as number | undefined);
            case 'size':
              return fmtSize(row['artifactSize'] as number | undefined);
            case 'source':
              return String(row['triggerType'] ?? '—');
            case '$createdAt':
              return fmtDateLong(row['$createdAt'] ?? row['createdAt']);
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'status') return <StatusChip label={String(row['status'] ?? '')} />;
          if (key === 'source') {
            const src = String(row['triggerType'] ?? '');
            const Icon = src === 'git' ? GitBranch : src === 'cli' ? Terminal : Upload;
            return (
              <span className="inline-flex items-center gap-1.5 text-text-secondary">
                <Icon size={13} className="text-text-subtle" />
                {cap(src || 'manual')}
              </span>
            );
          }
          return undefined;
        }}
        rowIcon={() => Rocket}
        total={total}
        perPage={PER_PAGE}
        page={page}
        onPerPageChange={() => {}}
        onPrev={() => setPage((p) => Math.max(1, p - 1))}
        onNext={() => setPage((p) => p + 1)}
        itemLabel="deployments"
        emptyIcon={Rocket}
        emptyTitle="No deployments yet"
        emptySubtitle='Click "Create deployment" to trigger your first build.'
        loading={query.isLoading}
        error={query.error}
        onRetry={() => query.refetch()}
      />

      <GitDeployDialog
        open={dialog === 'git'}
        onOpenChange={(o) => !o && setDialog(null)}
        targetId={targetId}
        onCreated={() => {
          setDialog(null);
          query.refetch();
        }}
      />
      <CliDeployDialog open={dialog === 'cli'} onOpenChange={(o) => !o && setDialog(null)} targetId={targetId} />
      <ManualDeployDialog open={dialog === 'manual'} onOpenChange={(o) => !o && setDialog(null)} />
    </div>
  );
}

// ── Connect deployment dialog ────────────────────────────────────────────────────

function ConnectDeploymentDialog({
  open,
  onOpenChange,
  projectId,
  onSelect,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
  onSelect: (targetId: string) => void;
}) {
  const navigate = useNavigate();
  const query = useQuery({
    queryKey: ['deploy-targets', projectId],
    enabled: open,
    queryFn: async () => {
      const res = await api.get('/deploy/targets');
      return ((res.data as Record<string, unknown>)['targets'] as Row[] | undefined) ?? [];
    },
  });
  const targets = query.data ?? [];

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Connect deployment"
      subtitle="Link a deploy target to enable automatic builds"
      width={460}
    >
      {query.isLoading ? (
        <div className="py-6 text-center text-[length:var(--text-body)] text-text-subtle">Loading…</div>
      ) : targets.length === 0 ? (
        <div className="flex flex-col items-center py-4 text-center">
          <Rocket size={32} className="text-text-subtle" />
          <div className="mt-3 text-[length:var(--text-body)] font-medium text-text-primary">
            No deploy targets found
          </div>
          <div className="mt-1 text-[length:var(--text-label)] text-text-secondary">
            You need to create a deploy target first.
          </div>
          <Button
            size="sm"
            className="mt-3.5"
            onClick={() => {
              onOpenChange(false);
              navigate(`/project/${projectId}/deploy`);
            }}
          >
            <ExternalLink size={13} />
            Go to Deploy
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {targets.map((t) => {
            const id = String(t['$id'] ?? t['id'] ?? '');
            const name = String(t['name'] ?? id);
            const type = String(t['type'] ?? '');
            return (
              <button
                key={id}
                type="button"
                onClick={() => onSelect(id)}
                className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-surface p-3.5 text-left transition-colors hover:border-field-border hover:bg-fill-hover"
              >
                <span className="flex h-8 w-8 items-center justify-center rounded-[var(--radius-6)] bg-[color-mix(in_srgb,var(--color-accent)_10%,transparent)] text-[var(--color-accent)]">
                  <Rocket size={14} />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[length:var(--text-body)] font-medium text-text-primary">
                    {name}
                  </span>
                  {type && (
                    <span className="block truncate text-[length:var(--text-caption)] text-text-secondary">
                      {type}
                    </span>
                  )}
                </span>
                <ChevronRight size={14} className="text-text-subtle" />
              </button>
            );
          })}
        </div>
      )}
    </FormDialog>
  );
}

// ── Create deployment dialogs ────────────────────────────────────────────────────

function GitDeployDialog({
  open,
  onOpenChange,
  targetId,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  targetId: string;
  onCreated: () => void;
}) {
  const [pipelineId, setPipelineId] = useState('');
  const [activate, setActivate] = useState(true);
  const [triggering, setTriggering] = useState(false);

  const query = useQuery({
    queryKey: ['deploy-pipelines', targetId],
    enabled: open,
    queryFn: async () => {
      // Pipelines are a project-wide resource (GET /deploy/pipelines); there is
      // no per-target route, so fetch them all and filter to this target.
      const res = await api.get('/deploy/pipelines');
      const list = ((res.data as Record<string, unknown>)['pipelines'] as Row[] | undefined) ?? [];
      return list.filter((p) => String(p['targetId'] ?? '') === targetId);
    },
  });
  const pipelines = query.data ?? [];
  const selected = pipelineId || String(pipelines[0]?.['$id'] ?? '');
  const repo = String(pipelines.find((p) => String(p['$id']) === selected)?.['sourceURL'] ?? '');

  const trigger = async () => {
    if (!selected) return;
    setTriggering(true);
    try {
      await api.post(`/deploy/pipelines/${selected}/trigger`, { triggerType: 'manual', activate });
      toast.success('Deployment triggered');
      onCreated();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setTriggering(false);
    }
  };

  const empty = !query.isLoading && pipelines.length === 0;

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Create Git deployment"
      subtitle="Trigger a build from a connected Git pipeline."
      submitLabel="Create"
      loading={triggering}
      submitDisabled={!selected}
      onSubmit={empty ? undefined : trigger}
      width={460}
    >
      {query.isLoading ? (
        <div className="py-6 text-center text-[length:var(--text-body)] text-text-subtle">Loading…</div>
      ) : empty ? (
        <div className="flex flex-col items-center py-4 text-center">
          <GitBranch size={28} className="text-text-subtle" />
          <div className="mt-2.5 text-[length:var(--text-body)] text-text-secondary">
            No Git pipelines configured.
          </div>
          <div className="mt-0.5 text-[length:var(--text-label)] text-text-subtle">
            Connect a repository in the Deploy section first.
          </div>
        </div>
      ) : (
        <>
          <div className="flex items-center gap-2 rounded-[var(--radius)] border border-border bg-fill px-3 py-2.5">
            <GitBranch size={14} className="text-text-subtle" />
            <span className="truncate text-[length:var(--text-body)] font-medium text-text-primary">
              {repo || 'Git repository'}
            </span>
          </div>
          <SelectField
            label="Production branch"
            value={selected}
            onChange={setPipelineId}
            options={pipelines.map((p) => ({
              value: String(p['$id'] ?? ''),
              label: String(p['branch'] ?? 'main'),
            }))}
          />
          <label className="flex cursor-pointer items-start gap-2.5">
            <Checkbox checked={activate} onCheckedChange={(v) => setActivate(!!v)} className="mt-0.5" />
            <span>
              <span className="block text-[length:var(--text-body)] font-medium text-text-primary">
                Activate deployment after build
              </span>
              <span className="block text-[length:var(--text-caption)] text-text-subtle">
                This deployment will automatically activate after the build completes. If unchecked, it will
                remain inactive.
              </span>
            </span>
          </label>
        </>
      )}
    </FormDialog>
  );
}

function CliDeployDialog({
  open,
  onOpenChange,
  targetId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  targetId: string;
}) {
  const snippets = {
    unix: `applad deploy \\\n  --target-id ${targetId} \\\n  --activate`,
    cmd: `applad deploy ^\n  --target-id ${targetId} ^\n  --activate`,
    powershell: `applad deploy \`\n  --target-id ${targetId} \`\n  --activate`,
  };
  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Create CLI deployment"
      subtitle="Deploy by running the Applad CLI in your project folder."
      width={520}
    >
      <ShellSnippets snippets={snippets} />
      <p className="text-[length:var(--text-caption)] text-text-subtle">
        If it&apos;s your first time using the CLI, install it with{' '}
        <code className="font-[family-name:var(--font-mono)]">npm install -g @applad/cli</code> before running
        the deploy command.
      </p>
    </FormDialog>
  );
}

function ManualDeployDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Create manual deployment"
      subtitle="Upload a tar.gz file containing your project source code."
      width={480}
    >
      <div className="flex h-36 flex-col items-center justify-center rounded-[var(--radius)] border border-border bg-fill text-center">
        <UploadCloud size={32} className="text-text-subtle" />
        <div className="mt-2.5 text-[length:var(--text-body)] text-text-secondary">
          Drag and drop file here or click to upload
        </div>
        <div className="mt-1 text-[length:var(--text-caption)] text-text-subtle">Max file size: 30MB</div>
      </div>
    </FormDialog>
  );
}
