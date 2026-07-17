import { type ReactNode, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Activity,
  ArrowRight,
  Clock,
  ExternalLink,
  GitBranch,
  Globe,
  HardDrive,
  Plus,
  RefreshCw,
  RotateCcw,
  Rocket,
  ScrollText,
  Timer,
  Trash2,
  Upload,
  type LucideIcon,
} from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { IdText } from '@/components/id-text';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { ConfirmDialog, FormDialog, FormField, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import type { Row } from '@/components/data-table';
import { TargetDetailScaffold } from '../deploy-shared/TargetDetailScaffold';
import { DeploymentsPanel, useReleases } from '../deploy-shared/DeploymentsPanel';
import { frameworkById } from '../deploy-shared/frameworks';
import {
  asNumber,
  formatBytes,
  formatDuration,
  formatNumber,
  formatTimestamp,
  rowId,
  timeAgo,
} from '../deploy-shared/format';

const DETAIL_TABS = ['Overview', 'Deployments', 'Logs', 'Domains', 'Usage', 'Settings'];

export function SiteDetail({
  site,
  onChange,
  onBack,
  onDeleted,
}: {
  site: Row;
  onChange: (site: Row) => void;
  onBack: () => void;
  onDeleted: () => void;
}) {
  const [tab, setTab] = useState(0);
  const siteId = rowId(site);
  const name = String(site['name'] ?? 'Untitled');

  return (
    <TargetDetailScaffold
      backLabel="Sites"
      onBack={onBack}
      name={name}
      id={siteId}
      tabs={DETAIL_TABS}
      tab={tab}
      onTab={setTab}
    >
      {tab === 0 && <OverviewTab site={site} siteId={siteId} onGoTab={setTab} />}
      {tab === 1 && <DeploymentsPanel targetId={siteId} />}
      {tab === 2 && <LogsTab siteId={siteId} />}
      {tab === 3 && <DomainsTab siteId={siteId} />}
      {tab === 4 && <UsageTab siteId={siteId} />}
      {tab === 5 && <SettingsTab site={site} siteId={siteId} onChange={onChange} onDeleted={onDeleted} />}
    </TargetDetailScaffold>
  );
}

// ── Overview ────────────────────────────────────────────────────────────────

function OverviewTab({ site, siteId, onGoTab }: { site: Row; siteId: string; onGoTab: (i: number) => void }) {
  const framework = frameworkById(String(site['framework'] ?? 'static'));
  const source = String(site['source'] ?? 'git');
  const repository = String(site['repository'] ?? '');
  const updatedAt = site['$updatedAt'] ?? site['updatedAt'] ?? '';

  const domains = useQuery({
    queryKey: ['site-domains', siteId],
    queryFn: async () => (await api.get(`/deploy/targets/${siteId}/domains`)).data as Record<string, unknown>,
  });
  const releases = useReleases(siteId);

  const domainCount = (domains.data?.['domains'] as unknown[] | undefined)?.length;
  const releaseCount = releases.data?.['total'];

  const [confirmRollback, setConfirmRollback] = useState(false);
  const [rollingBack, setRollingBack] = useState(false);

  // Candidate = the most recent successful release (matches Flutter — `success` only).
  const rollbackCandidate = ((releases.data?.['releases'] as Row[] | undefined) ?? []).find(
    (r) => r['status'] === 'success',
  );

  const requestRollback = () => {
    if (!rollbackCandidate) {
      toast.info('No successful deployment to roll back to');
      return;
    }
    setConfirmRollback(true);
  };

  const doRollback = async () => {
    if (!rollbackCandidate) return;
    setRollingBack(true);
    try {
      await api.post(`/deploy/releases/${rowId(rollbackCandidate)}/rollback`, {});
      toast.success('Rollback initiated');
      releases.refetch();
      setConfirmRollback(false);
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setRollingBack(false);
    }
  };

  const FrameworkIcon = framework.icon;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-6 lg:flex-row">
        <div className="flex h-[200px] w-full max-w-[340px] flex-col items-center justify-center gap-2 rounded-[var(--radius)] border border-border bg-surface text-text-subtle">
          <Globe size={40} />
          <span className="text-[length:var(--text-label)]">Site preview</span>
        </div>

        <div className="flex flex-1 flex-col gap-3">
          <InfoRow icon={Globe} label="Domain" value={String(site['domain'] ?? 'No domain assigned')} />
          <InfoRow icon={Clock} label="Last deployed" value={updatedAt ? timeAgo(updatedAt) || 'Never' : 'Never'} />
          <InfoRow
            icon={source === 'git' ? GitBranch : Upload}
            label="Source"
            value={repository || (source === 'git' ? 'Git repository' : 'Manual upload')}
          />
          <InfoRow icon={FrameworkIcon} label="Framework" value={framework.label} />
          <InfoRow icon={Timer} label="Build duration" value={formatDuration(site['buildDuration'])} />
          <InfoRow icon={HardDrive} label="Total size" value={formatBytes(site['totalSize'])} />

          <div className="mt-2 flex gap-2">
            <Button size="sm">
              <ExternalLink size={14} />
              Visit
            </Button>
            <Button variant="outline" size="sm" disabled={!releases.data} onClick={requestRollback}>
              <RotateCcw size={14} />
              Instant rollback
            </Button>
            <ConfirmDialog
              open={confirmRollback}
              onOpenChange={setConfirmRollback}
              title="Instant rollback"
              message="Roll back to the last successful deployment? This replaces the current live deployment."
              confirmLabel="Roll back"
              loading={rollingBack}
              onConfirm={doRollback}
            />
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        <SummaryCard
          icon={Globe}
          label="Domains"
          value={domainCount != null ? String(domainCount) : '--'}
          sublabel="View all domains"
          onClick={() => onGoTab(3)}
        />
        <SummaryCard
          icon={Rocket}
          label="Deployments"
          value={releaseCount != null ? String(releaseCount) : '--'}
          sublabel="View all deployments"
          onClick={() => onGoTab(1)}
        />
      </div>
    </div>
  );
}

function InfoRow({ icon: Icon, label, value }: { icon: LucideIcon; label: string; value: string }) {
  return (
    <div className="flex items-start gap-2">
      <Icon size={14} className="mt-0.5 text-text-subtle" />
      <span className="w-28 shrink-0 text-[length:var(--text-body)] text-text-muted">{label}</span>
      <span className="min-w-0 flex-1 truncate text-[length:var(--text-body)] text-text-primary">{value}</span>
    </div>
  );
}

function SummaryCard({
  icon: Icon,
  label,
  value,
  sublabel,
  onClick,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
  sublabel: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-surface p-4 text-left transition-colors hover:bg-fill-hover"
    >
      <Icon size={20} className="text-[var(--color-accent)]" />
      <div>
        <div className="text-[length:var(--text-label)] text-text-muted">{label}</div>
        <div className="text-[length:var(--text-h1)] font-semibold text-text-primary">{value}</div>
      </div>
      <div className="ml-auto flex items-center gap-1 text-[length:var(--text-label)] text-[var(--color-accent)]">
        {sublabel}
        <ArrowRight size={12} />
      </div>
    </button>
  );
}

// ── Logs ────────────────────────────────────────────────────────────────────

function LogsTab({ siteId }: { siteId: string }) {
  const logs = useQuery({
    queryKey: ['site-logs', siteId],
    queryFn: async () => (await api.get(`/deploy/targets/${siteId}/logs`)).data as Record<string, unknown>,
  });
  const rows = (logs.data?.['logs'] as Row[] | undefined) ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <ScrollText size={16} className="text-text-muted" />
        <span className="text-[length:var(--text-control)] font-medium text-text-primary">Access Logs</span>
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => logs.refetch()}>
          <RefreshCw size={14} />
          Refresh
        </Button>
      </div>

      {logs.isLoading ? (
        <div className="py-10 text-center text-[length:var(--text-body)] text-text-muted">Loading…</div>
      ) : logs.error ? (
        <ErrorState error={logs.error} onRetry={() => logs.refetch()} />
      ) : rows.length === 0 ? (
        <EmptyState icon={ScrollText} title="No logs yet" />
      ) : (
        <div className="overflow-x-auto rounded-[var(--radius)] border border-border">
          <table className="w-full min-w-[720px] text-[length:var(--text-body)]">
            <thead>
              <tr className="border-b border-border bg-surface text-left text-text-muted">
                <th className="px-4 py-2.5 font-semibold">Log ID</th>
                <th className="px-4 py-2.5 font-semibold">Path</th>
                <th className="px-4 py-2.5 font-semibold">Method</th>
                <th className="px-4 py-2.5 font-semibold">Status</th>
                <th className="px-4 py-2.5 font-semibold">Duration</th>
                <th className="px-4 py-2.5 font-semibold">Created</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((log) => {
                const code = Math.trunc(asNumber(log['statusCode']));
                return (
                  <tr key={rowId(log)} className="border-b border-border last:border-0">
                    <td className="px-4 py-2.5">
                      <IdText id={rowId(log)} fontSize={12} />
                    </td>
                    <td className="px-4 py-2.5 text-text-primary">{String(log['path'] ?? '/')}</td>
                    <td className="px-4 py-2.5">
                      <span className="rounded-[var(--radius-sm)] bg-fill px-1.5 py-0.5 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] font-semibold text-text-muted">
                        {String(log['method'] ?? 'GET')}
                      </span>
                    </td>
                    <td className="px-4 py-2.5">
                      <HttpStatusBadge code={code} />
                    </td>
                    <td className="px-4 py-2.5 text-text-muted">{asNumber(log['duration'])}ms</td>
                    <td className="px-4 py-2.5 text-text-muted">
                      {formatTimestamp(log['$createdAt'] ?? log['createdAt'])}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function HttpStatusBadge({ code }: { code: number }) {
  let color = 'var(--status-neutral)';
  if (code >= 200 && code < 300) color = 'var(--status-success)';
  else if (code >= 400 && code < 500) color = 'var(--status-warning)';
  else if (code >= 500) color = 'var(--status-danger)';
  return (
    <span
      className="rounded-[var(--radius-sm)] px-1.5 py-0.5 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] font-semibold"
      style={{ color, backgroundColor: `color-mix(in srgb, ${color} 12%, transparent)` }}
    >
      {code}
    </span>
  );
}

// ── Domains ─────────────────────────────────────────────────────────────────

const DOMAIN_TARGET_TYPES: { value: string; label: string; icon: LucideIcon }[] = [
  { value: 'active_deployment', label: 'Active deployment', icon: Rocket },
  { value: 'git_branch', label: 'Git branch', icon: GitBranch },
  { value: 'redirect', label: 'Redirect', icon: ExternalLink },
];

function DomainsTab({ siteId }: { siteId: string }) {
  const domains = useQuery({
    queryKey: ['site-domains', siteId],
    queryFn: async () => (await api.get(`/deploy/targets/${siteId}/domains`)).data as Record<string, unknown>,
  });
  const rows = (domains.data?.['domains'] as Row[] | undefined) ?? [];
  const [adding, setAdding] = useState(false);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <Globe size={16} className="text-text-muted" />
        <span className="text-[length:var(--text-control)] font-medium text-text-primary">Domains</span>
        <Button size="sm" className="ml-auto" onClick={() => setAdding(true)}>
          <Plus size={14} />
          Add domain
        </Button>
      </div>

      {domains.isLoading ? (
        <div className="py-10 text-center text-[length:var(--text-body)] text-text-muted">Loading…</div>
      ) : domains.error ? (
        <ErrorState error={domains.error} onRetry={() => domains.refetch()} />
      ) : rows.length === 0 ? (
        <EmptyState
          icon={Globe}
          title="No custom domains"
          actionLabel="Add domain"
          onAction={() => setAdding(true)}
        />
      ) : (
        <div className="flex flex-col gap-2">
          {rows.map((d) => {
            const targetType = String(d['targetType'] ?? 'active_deployment');
            let targetLabel: string;
            let TargetIcon: LucideIcon;
            switch (targetType) {
              case 'active_deployment':
                targetLabel = 'Active deployment';
                TargetIcon = Rocket;
                break;
              case 'git_branch':
                targetLabel = `Branch: ${String(d['targetValue'] ?? 'main')}`;
                TargetIcon = GitBranch;
                break;
              case 'redirect':
                targetLabel = `Redirect: ${String(d['targetValue'] ?? '')}`;
                TargetIcon = ExternalLink;
                break;
              default:
                targetLabel = targetType;
                TargetIcon = ExternalLink;
            }
            return (
              <div
                key={rowId(d) || String(d['domain'])}
                className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-surface px-4 py-3"
              >
                <Globe size={14} className="text-[var(--color-accent)]" />
                <span className="flex-1 text-[length:var(--text-body)] text-text-primary">{String(d['domain'] ?? '')}</span>
                <TargetIcon size={14} className="text-text-subtle" />
                <span className="text-[length:var(--text-body)] text-text-muted">{targetLabel}</span>
              </div>
            );
          })}
        </div>
      )}

      <AddDomainDialog open={adding} onOpenChange={setAdding} siteId={siteId} onAdded={() => domains.refetch()} />
    </div>
  );
}

function AddDomainDialog({
  open,
  onOpenChange,
  siteId,
  onAdded,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  siteId: string;
  onAdded: () => void;
}) {
  const [domain, setDomain] = useState('');
  const [targetType, setTargetType] = useState('active_deployment');
  const [targetValue, setTargetValue] = useState('');
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    setSaving(true);
    try {
      await api.post(`/deploy/targets/${siteId}/domains`, {
        domain: domain.trim(),
        targetType,
        targetValue: targetValue.trim(),
      });
      onOpenChange(false);
      setDomain('');
      setTargetValue('');
      setTargetType('active_deployment');
      onAdded();
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
      title="Add domain"
      submitLabel="Add"
      loading={saving}
      submitDisabled={!domain.trim()}
      onSubmit={submit}
    >
      <TextField label="Domain" value={domain} onChange={(e) => setDomain(e.target.value)} placeholder="example.com" autoFocus />
      <FormField label="Target type">
        <div className="flex flex-wrap gap-2">
          {DOMAIN_TARGET_TYPES.map((t) => (
            <ChoiceChip
              key={t.value}
              icon={t.icon}
              label={t.label}
              selected={targetType === t.value}
              onClick={() => setTargetType(t.value)}
            />
          ))}
        </div>
      </FormField>
      {targetType !== 'active_deployment' && (
        <TextField
          label={targetType === 'git_branch' ? 'Branch name' : 'Redirect URL'}
          value={targetValue}
          onChange={(e) => setTargetValue(e.target.value)}
          placeholder={targetType === 'git_branch' ? 'main' : 'https://example.com'}
        />
      )}
    </FormDialog>
  );
}

export function ChoiceChip({
  icon: Icon,
  label,
  selected,
  onClick,
}: {
  icon: LucideIcon;
  label: string;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex items-center gap-2 rounded-[var(--radius)] border px-3 py-2 text-[length:var(--text-body)] transition-colors"
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
      <Icon size={14} />
      {label}
    </button>
  );
}

// ── Usage ───────────────────────────────────────────────────────────────────

function UsageTab({ siteId }: { siteId: string }) {
  const [range, setRange] = useState('30d');
  const stats = useQuery({
    queryKey: ['site-stats', siteId, range],
    queryFn: async () =>
      (await api.get(`/deploy/targets/${siteId}/stats`, { params: { range } })).data as Record<string, unknown>,
  });
  const data = stats.data;

  return (
    <div className="flex flex-col gap-5">
      <div className="flex items-center">
        <span className="text-[length:var(--text-control)] font-medium text-text-primary">Usage</span>
        <div className="ml-auto flex items-center rounded-[var(--radius)] border border-border bg-surface p-0.5">
          {['24h', '7d', '30d'].map((r) => (
            <button
              key={r}
              type="button"
              onClick={() => setRange(r)}
              className="rounded-[var(--radius-6)] px-3 py-1 text-[length:var(--text-label)] transition-colors"
              style={
                r === range
                  ? { backgroundColor: 'color-mix(in srgb, var(--color-accent) 15%, transparent)', color: 'var(--color-accent)' }
                  : { color: 'var(--text-muted)' }
              }
            >
              {r}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
        <StatCard icon={Activity} label="Requests" value={data ? formatNumber(data['requests']) : '--'} />
        <StatCard icon={HardDrive} label="Bandwidth" value={data ? formatBytes(data['bandwidth']) : '--'} />
        <StatCard icon={Timer} label="Build minutes" value={data ? formatDuration(data['buildMinutes']) : '--'} />
      </div>
    </div>
  );
}

export function StatCard({ icon: Icon, label, value }: { icon: LucideIcon; label: string; value: string }) {
  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-4">
      <Icon size={18} className="text-[var(--color-accent)]" />
      <div className="mt-2.5 text-[length:var(--text-label)] text-text-muted">{label}</div>
      <div className="mt-1 text-[length:var(--text-h2)] font-semibold text-text-primary">{value}</div>
    </div>
  );
}

// ── Settings ────────────────────────────────────────────────────────────────

function SettingsTab({
  site,
  siteId,
  onChange,
  onDeleted,
}: {
  site: Row;
  siteId: string;
  onChange: (site: Row) => void;
  onDeleted: () => void;
}) {
  const framework = frameworkById(String(site['framework'] ?? 'static'));
  const [name, setName] = useState(String(site['name'] ?? ''));
  const [repository, setRepository] = useState(String(site['repository'] ?? ''));
  const [installCommand, setInstallCommand] = useState(String(site['installCommand'] ?? ''));
  const [buildCommand, setBuildCommand] = useState(String(site['buildCommand'] ?? ''));
  const [outputDirectory, setOutputDirectory] = useState(String(site['outputDirectory'] ?? ''));
  const initialEnv = (site['environmentVariables'] as Record<string, unknown> | undefined) ?? {};
  const [envVars, setEnvVars] = useState<{ key: string; value: string }[]>(
    Object.entries(initialEnv).map(([k, v]) => ({ key: k, value: String(v) })),
  );
  const [saving, setSaving] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const save = async () => {
    setSaving(true);
    try {
      const envMap: Record<string, string> = {};
      for (const { key, value } of envVars) if (key.trim()) envMap[key] = value;
      const res = await api.put(`/deploy/targets/${siteId}`, {
        name,
        repository,
        buildCommand,
        outputDirectory,
        installCommand,
        environmentVariables: envMap,
      });
      onChange(res.data as Row);
      toast.success('Changes saved');
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSaving(false);
    }
  };

  const del = async () => {
    setDeleting(true);
    try {
      await api.delete(`/deploy/targets/${siteId}`);
      onDeleted();
    } catch (e) {
      toast.error(friendlyError(e));
      setDeleting(false);
    }
  };

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <Section title="General">
        <TextField label="Site name" value={name} onChange={(e) => setName(e.target.value)} placeholder="my-site" />
        <FormField label="Framework">
          <div className="text-[length:var(--text-body)] text-text-primary">{framework.label}</div>
        </FormField>
        <TextField
          label="Git repository"
          value={repository}
          onChange={(e) => setRepository(e.target.value)}
          placeholder="https://github.com/user/repo"
        />
      </Section>

      <Section title="Build configuration">
        <TextField label="Install command" value={installCommand} onChange={(e) => setInstallCommand(e.target.value)} placeholder="npm install" />
        <TextField label="Build command" value={buildCommand} onChange={(e) => setBuildCommand(e.target.value)} placeholder="npm run build" />
        <TextField label="Output directory" value={outputDirectory} onChange={(e) => setOutputDirectory(e.target.value)} placeholder="dist" />
      </Section>

      <Section title="Environment variables">
        <div className="flex flex-col gap-2">
          {envVars.map((pair, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <Input
                value={pair.key}
                placeholder="KEY"
                className="flex-1 font-[family-name:var(--font-mono)]"
                onChange={(e) => setEnvVars((v) => v.map((p, i) => (i === idx ? { ...p, key: e.target.value } : p)))}
              />
              <Input
                value={pair.value}
                placeholder="VALUE"
                className="flex-1 font-[family-name:var(--font-mono)]"
                onChange={(e) => setEnvVars((v) => v.map((p, i) => (i === idx ? { ...p, value: e.target.value } : p)))}
              />
              <button
                type="button"
                onClick={() => setEnvVars((v) => v.filter((_, i) => i !== idx))}
                className="rounded-[var(--radius-6)] p-1.5 text-text-muted transition-colors hover:bg-fill hover:text-[var(--color-danger)]"
                aria-label="Remove variable"
              >
                <Trash2 size={14} />
              </button>
            </div>
          ))}
          <button
            type="button"
            onClick={() => setEnvVars((v) => [...v, { key: '', value: '' }])}
            className="flex w-fit items-center gap-1.5 text-[length:var(--text-body)] text-[var(--color-accent)]"
          >
            <Plus size={14} />
            Add variable
          </button>
        </div>
      </Section>

      <div className="flex justify-end">
        <Button loading={saving} onClick={save}>
          Save changes
        </Button>
      </div>

      <div className="rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-danger)_20%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_5%,transparent)] p-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <div className="text-[length:var(--text-control)] font-semibold text-[var(--color-danger)]">Delete this site</div>
            <div className="mt-0.5 text-[length:var(--text-label)] text-text-subtle">
              Once deleted, all deployments, domains, and logs will be permanently removed.
            </div>
          </div>
          <Button variant="outline" onClick={() => setConfirmDelete(true)}>
            Delete site
          </Button>
        </div>
      </div>

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Delete site"
        message={`Are you sure you want to delete "${String(site['name'] ?? '')}"? This will remove all deployments, domains, and data. This action cannot be undone.`}
        loading={deleting}
        onConfirm={del}
      />
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-3">
      <div className="text-[length:var(--text-control)] font-semibold text-text-primary">{title}</div>
      <div className="flex flex-col gap-3 rounded-[var(--radius)] border border-border bg-surface p-5">{children}</div>
    </div>
  );
}
