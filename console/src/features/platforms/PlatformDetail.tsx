import { useState } from 'react';
import { ArrowLeft } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { PageTabs } from '@/components/page-tabs';
import type { Row } from '@/components/data-table';
import { IdText } from '@/components/id-text';
import { CodeBlock } from '@/components/code-block';
import { Button } from '@/components/ui/button';
import { ConfirmDialog, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { ShellSnippets } from './ShellSnippets';
import { PlatformDeploymentTab } from './PlatformDeploymentTab';
import {
  fmtDate,
  identityHint,
  identityLabel,
  identityValue,
  platformId,
  typeBadgeColor,
  typeIconFor,
  typeLabel,
} from './platform-utils';

const DETAIL_TABS = ['Overview', 'Deployment', 'Settings'];

export function PlatformDetail({
  platform,
  projectId,
  onBack,
  onChange,
  onDeleted,
}: {
  platform: Row;
  projectId: string;
  onBack: () => void;
  onChange: (next: Row) => void;
  onDeleted: () => void;
}) {
  const [tab, setTab] = useState(0);
  const type = String(platform['type'] ?? 'web');
  const TypeIcon = typeIconFor(type);
  const name = String(platform['name'] ?? '');
  const identity = identityValue(platform);
  const accent = typeBadgeColor(type);

  return (
    <div className="flex flex-col gap-4 p-6 md:p-8">
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={onBack}
          className="rounded-[var(--radius-6)] p-1 text-text-secondary transition-colors hover:bg-fill hover:text-text-primary"
          aria-label="Back"
        >
          <ArrowLeft size={18} />
        </button>
        <span
          className="flex h-8 w-8 items-center justify-center rounded-[var(--radius)]"
          style={{ backgroundColor: `color-mix(in srgb, ${accent} 12%, transparent)`, color: accent }}
        >
          <TypeIcon size={16} />
        </span>
        <div className="min-w-0">
          <div className="truncate text-[length:var(--text-title)] font-semibold text-text-primary">
            {name || 'Platform'}
          </div>
          {identity && (
            <div className="truncate font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-secondary">
              {identity}
            </div>
          )}
        </div>
      </div>

      <PageTabs tabs={DETAIL_TABS} selected={tab} onChange={setTab} />

      {tab === 0 && <OverviewTab platform={platform} projectId={projectId} type={type} />}
      {tab === 1 && (
        <PlatformDeploymentTab platform={platform} projectId={projectId} onChange={onChange} />
      )}
      {tab === 2 && (
        <SettingsTab
          platform={platform}
          projectId={projectId}
          type={type}
          onChange={onChange}
          onDeleted={onDeleted}
        />
      )}
    </div>
  );
}

// ── Overview ─────────────────────────────────────────────────────────────────────

function OverviewTab({ platform, projectId, type }: { platform: Row; projectId: string; type: string }) {
  const id = platformId(platform);
  const identity = identityValue(platform);
  const created = fmtDate(platform['$createdAt'] ?? platform['createdAt']);

  const sdkSnippet = `import { Applad } from '@applad/js';

const applad = new Applad()
  .setEndpoint('https://your-domain.com/v1')
  .setProject('${projectId}')
  .setPlatform('${id}');`;

  const cliSnippets = {
    unix: `npm install -g @applad/cli\n\napplad deploy \\\n  --platform-id ${id} \\\n  --activate`,
    cmd: `npm install -g @applad/cli\n\napplad deploy ^\n  --platform-id ${id} ^\n  --activate`,
    powershell: `npm install -g @applad/cli\n\napplad deploy \`\n  --platform-id ${id} \`\n  --activate`,
  };

  return (
    <div className="flex max-w-3xl flex-col gap-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <InfoCard label="Type" value={typeLabel(type)} />
        <InfoCard label={identityLabel(type)} value={identity || '—'} />
        <InfoCard label="Registered" value={created} />
      </div>

      <div className="rounded-[var(--radius-10)] border border-border bg-surface p-4">
        <div className="text-[length:var(--text-label)] text-text-subtle">Platform ID</div>
        <div className="mt-2">
          <IdText id={id} />
        </div>
        <div className="mt-1.5 text-[length:var(--text-caption)] text-text-subtle">
          Use this ID when initialising the SDK to identify and restrict API access to this platform.
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <div className="text-[length:var(--text-control)] font-semibold text-text-primary">Initialise the SDK</div>
        <CodeBlock code={sdkSnippet} language="javascript" />
      </div>

      <div className="flex flex-col gap-2">
        <div className="text-[length:var(--text-control)] font-semibold text-text-primary">Install the CLI</div>
        <ShellSnippets snippets={cliSnippets} />
      </div>
    </div>
  );
}

function InfoCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[var(--radius-10)] border border-border bg-surface p-4">
      <div className="text-[length:var(--text-label)] text-text-subtle">{label}</div>
      <div className="mt-1.5 truncate text-[length:var(--text-control)] font-medium text-text-primary">
        {value}
      </div>
    </div>
  );
}

// ── Settings ─────────────────────────────────────────────────────────────────────

function SettingsTab({
  platform,
  projectId,
  type,
  onChange,
  onDeleted,
}: {
  platform: Row;
  projectId: string;
  type: string;
  onChange: (next: Row) => void;
  onDeleted: () => void;
}) {
  const id = platformId(platform);
  const [name, setName] = useState(String(platform['name'] ?? ''));
  const [hostname, setHostname] = useState(identityValue(platform));
  const [saving, setSaving] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const save = async () => {
    setSaving(true);
    try {
      await api.patch(`/projects/${projectId}/platforms/${id}`, {
        name: name.trim(),
        hostname: hostname.trim(),
      });
      onChange({ ...platform, name: name.trim(), hostname: hostname.trim() });
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
      await api.delete(`/projects/${projectId}/platforms/${id}`);
      onDeleted();
    } catch (e) {
      toast.error(friendlyError(e));
      setDeleting(false);
    }
  };

  return (
    <div className="flex max-w-xl flex-col gap-6">
      <div className="flex flex-col gap-4">
        <div className="text-[length:var(--text-title)] font-semibold text-text-primary">Settings</div>
        <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} placeholder="My app" />
        <TextField
          label={identityLabel(type)}
          value={hostname}
          onChange={(e) => setHostname(e.target.value)}
          placeholder={identityHint(type)}
        />
        <div>
          <Button loading={saving} onClick={save}>
            Save changes
          </Button>
        </div>
      </div>

      <div className="rounded-[var(--radius-10)] border border-[color-mix(in_srgb,var(--color-danger)_30%,transparent)] bg-surface p-5">
        <div className="text-[length:var(--text-control)] font-semibold text-[var(--color-danger)]">
          Danger zone
        </div>
        <div className="mt-2 text-[length:var(--text-body)] text-text-subtle">
          Remove this platform. API access from this platform will be revoked.
        </div>
        <Button variant="outline" className="mt-3" onClick={() => setConfirmDelete(true)}>
          Remove platform
        </Button>
      </div>

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Remove platform"
        message="This platform will be removed and API access revoked. This cannot be undone."
        confirmLabel="Remove"
        loading={deleting}
        onConfirm={del}
      />
    </div>
  );
}
