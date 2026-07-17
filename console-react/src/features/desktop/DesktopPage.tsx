import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Apple, Check, Laptop, Minus, Monitor, Plus, Terminal, type LucideIcon } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { PageTabs } from '@/components/page-tabs';
import { StatusChip } from '@/components/status-chip';
import { DeployCreateEntry, type CreateEntryResult } from '@/components/deploy-create-entry';
import { ConfirmDialog, FormDialog, FormField, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import {
  BuildsTab,
  ChoiceChip,
  ConfigField,
  ConfigSection,
  DangerZone,
  DetailHeader,
  InfoCard,
  SettingRow,
  SigningUploadCard,
  TabSpinner,
  fmtDate,
  type Target,
} from '../deploy-shared/shared';

const COLUMNS: DataTableColumn[] = [
  { key: 'name', label: 'Name', flex: 4, sortable: true },
  { key: 'platforms', label: 'Platforms', flex: 2 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'updatedAt', label: 'Updated', flex: 2 },
];

const DETAIL_TABS = ['Overview', 'Builds', 'Signing', 'Distribution', 'Settings'];

function platformLabel(platform: string): string {
  switch (platform) {
    case 'macos':
      return 'macOS';
    case 'windows':
      return 'Windows';
    case 'linux':
      return 'Linux';
    default:
      return 'Cross-platform';
  }
}

function platformsLabel(row: Row): string {
  const parts: string[] = [];
  if (row['macEnabled'] === true) parts.push('macOS');
  if (row['windowsEnabled'] === true) parts.push('Windows');
  if (row['linuxEnabled'] === true) parts.push('Linux');
  return parts.length === 0 ? '—' : parts.join(', ');
}

export function DesktopPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const list = useResourceList({
    endpoint: '/deploy/targets',
    itemsKey: 'targets',
    params: { type: 'desktop', projectId },
    scope: [projectId],
    defaultPerPage: 6,
  });

  if (selectedId) {
    return (
      <DesktopDetail
        id={selectedId}
        onBack={() => setSelectedId(null)}
        onDeleted={() => {
          setSelectedId(null);
          list.refetch();
        }}
      />
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
          Desktop Apps
        </h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Build for macOS, Windows, and Linux from source.
        </p>
      </div>

      <DesktopList list={list} onOpen={setSelectedId} />
    </div>
  );
}

function DesktopList({
  list,
  onOpen,
}: {
  list: ReturnType<typeof useResourceList<Row>>;
  onOpen: (id: string) => void;
}) {
  const [entryOpen, setEntryOpen] = useState(false);
  const [pending, setPending] = useState<CreateEntryResult | null>(null);

  return (
    <>
      <DataTable
        columns={COLUMNS}
        rows={list.rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'name':
              return String(row['name'] ?? '');
            case 'platforms':
              return platformsLabel(row);
            case 'status':
              return String(row['status'] ?? 'active');
            case 'updatedAt':
              return fmtDate(row['updatedAt'] ?? row['$updatedAt']);
            default:
              return '';
          }
        }}
        cellRender={(row, key) =>
          key === 'status' ? <StatusChip label={String(row['status'] ?? 'active')} /> : undefined
        }
        rowIcon={() => Monitor}
        onRowClick={(row) => onOpen(String(row['$id'] ?? row['id'] ?? ''))}
        onDeleteRow={async (row) => {
          await api.delete(`/deploy/targets/${String(row['$id'])}`);
          list.refetch();
        }}
        createLabel="Create app"
        onCreate={() => setEntryOpen(true)}
        searchHint="Search apps…"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="apps"
        emptyIcon={Plus}
        emptyTitle="No desktop apps yet"
        emptySubtitle="Build for macOS, Windows, and Linux from source."
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      <DeployCreateEntry
        open={entryOpen}
        onOpenChange={setEntryOpen}
        category="desktop"
        title="Create Desktop App"
        subtitle="Choose how to get started"
        onResult={(r) => setPending(r)}
      />

      <CreateDesktopDialog
        result={pending}
        onClose={() => setPending(null)}
        onCreated={() => {
          setPending(null);
          list.refetch();
        }}
      />
    </>
  );
}

const PLATFORM_CHOICES: { value: string; label: string; icon: LucideIcon }[] = [
  { value: 'macos', label: 'macOS', icon: Apple },
  { value: 'windows', label: 'Windows', icon: Monitor },
  { value: 'linux', label: 'Linux', icon: Terminal },
  { value: 'cross-platform', label: 'Cross-platform', icon: Laptop },
];

const FRAMEWORK_CHOICES: { value: string; label: string }[] = [
  { value: 'flutter', label: 'Flutter' },
  { value: 'electron', label: 'Electron' },
  { value: 'tauri', label: 'Tauri' },
];

function CreateDesktopDialog({
  result,
  onClose,
  onCreated,
}: {
  result: CreateEntryResult | null;
  onClose: () => void;
  onCreated: () => void;
}) {
  const prefillName =
    (result?.choice === 'template' && (result.templateConfig?.['name'] as string)) ||
    (result?.choice === 'repository' && (result.repoConfig?.['name'] as string)) ||
    '';
  const prefillFramework =
    (result?.choice === 'template' && (result.templateConfig?.['framework'] as string)) || 'flutter';
  const prefillRepo =
    (result?.choice === 'repository' &&
      ((result.repoConfig?.['cloneUrl'] as string) ?? (result.repoConfig?.['url'] as string))) ||
    '';
  const prefillBranch =
    (result?.choice === 'repository' && (result.repoConfig?.['defaultBranch'] as string)) || 'main';
  const sourceType =
    result?.choice === 'template' ? 'template' : result?.choice === 'repository' ? 'git' : 'upload';

  const [name, setName] = useState('');
  const [platform, setPlatform] = useState('cross-platform');
  const [framework, setFramework] = useState('flutter');
  const [saving, setSaving] = useState(false);

  const [seenResult, setSeenResult] = useState<CreateEntryResult | null>(null);
  if (result && result !== seenResult) {
    setSeenResult(result);
    setName(prefillName);
    setPlatform('cross-platform');
    setFramework(prefillFramework);
  }

  const submit = async () => {
    setSaving(true);
    try {
      await api.post('/deploy/targets', {
        name: name.trim(),
        type: 'desktop',
        platform,
        framework,
        source: sourceType,
        repository: prefillRepo,
        branch: prefillBranch,
        ...(result?.templateConfig ? { templateId: result.templateConfig['$id'] } : {}),
      });
      toast.success('Desktop app created');
      onCreated();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open={result != null}
      onOpenChange={(o) => !o && onClose()}
      title="Create Desktop App"
      subtitle="Configure your desktop application"
      submitLabel="Create"
      loading={saving}
      submitDisabled={!name.trim()}
      width={500}
      onSubmit={submit}
    >
      <TextField
        label="App name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="my-desktop-app"
        autoFocus
      />
      <FormField label="Platform">
        <div className="flex flex-wrap gap-2">
          {PLATFORM_CHOICES.map((p) => (
            <ChoiceChip
              key={p.value}
              label={p.label}
              icon={p.icon}
              active={platform === p.value}
              onClick={() => setPlatform(p.value)}
              className="w-[110px]"
            />
          ))}
        </div>
      </FormField>
      <FormField label="Framework">
        <div className="flex flex-wrap gap-2">
          {FRAMEWORK_CHOICES.map((f) => (
            <ChoiceChip
              key={f.value}
              label={f.label}
              active={framework === f.value}
              onClick={() => setFramework(f.value)}
            />
          ))}
        </div>
      </FormField>
    </FormDialog>
  );
}

function DesktopDetail({
  id,
  onBack,
  onDeleted,
}: {
  id: string;
  onBack: () => void;
  onDeleted: () => void;
}) {
  const [tab, setTab] = useState(0);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const q = useQuery({
    queryKey: ['deploy-target', id],
    queryFn: async () => (await api.get(`/deploy/targets/${id}`)).data as Target,
  });

  if (q.isLoading || !q.data) {
    return (
      <div className="p-6 md:p-8">
        <TabSpinner />
      </div>
    );
  }

  const t = q.data;
  const platform = String(t['platform'] ?? 'cross-platform');

  const del = async () => {
    setDeleting(true);
    try {
      await api.delete(`/deploy/targets/${id}`);
      toast.success('App deleted');
      onDeleted();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setDeleting(false);
      setConfirmDelete(false);
    }
  };

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <DetailHeader title={String(t['name'] ?? '')} onBack={onBack} />
      <PageTabs tabs={DETAIL_TABS} selected={tab} onChange={setTab} />

      {tab === 0 && (
        <div className="flex flex-col gap-4">
          <div className="flex gap-4">
            <InfoCard label="Platform" value={platformLabel(platform)} />
            <InfoCard label="Framework" value={String(t['framework'] ?? '—')} />
            <InfoCard label="Build Type" value={String(t['buildType'] ?? 'release')} />
          </div>
          <div className="flex gap-4">
            <InfoCard label="Source" value={String(t['source'] ?? 'manual')} />
            <InfoCard label="Repository" value={String(t['repository'] ?? '—')} />
            <InfoCard label="Branch" value={String(t['branch'] ?? 'main')} />
          </div>
        </div>
      )}

      {tab === 1 && <BuildsTab targetId={id} />}

      {tab === 2 && <SigningContent t={t} platform={platform} />}

      {tab === 3 && <DistributionContent t={t} platform={platform} />}

      {tab === 4 && (
        <div className="flex flex-col gap-6">
          <div className="text-[length:var(--text-title)] font-semibold text-text-primary">
            App Settings
          </div>
          <div className="flex flex-col">
            <SettingRow label="Name" value={String(t['name'] ?? '')} />
            <SettingRow label="Platform" value={platformLabel(platform)} />
            <SettingRow label="Framework" value={String(t['framework'] ?? '—')} />
            <SettingRow label="Source" value={String(t['source'] ?? 'manual')} />
            <SettingRow label="Repository" value={String(t['repository'] ?? '—')} />
            <SettingRow label="Branch" value={String(t['branch'] ?? 'main')} />
          </div>
          <DangerZone
            description="Delete this app and all its builds."
            onDelete={() => setConfirmDelete(true)}
          />
        </div>
      )}

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={(o) => !o && setConfirmDelete(false)}
        title="Delete app"
        message="Delete this app and all its builds. This action cannot be undone."
        loading={deleting}
        onConfirm={del}
      />
    </div>
  );
}

function SigningContent({ t, platform }: { t: Target; platform: string }) {
  const showMac = platform === 'macos' || platform === 'cross-platform';
  const showWin = platform === 'windows' || platform === 'cross-platform';
  const showLinux = platform === 'linux' || platform === 'cross-platform';
  return (
    <div className="flex flex-col gap-4">
      <div className="text-[length:var(--text-title)] font-semibold text-text-primary">
        Code Signing
      </div>
      {showMac && (
        <>
          <SigningUploadCard
            title="macOS Certificate"
            description="Upload your Developer ID certificate (.p12) for macOS code signing"
            buttonLabel="Upload .p12 certificate"
          />
          <ConfigSection
            title="Apple Developer Team ID"
            description="Required for notarization and distribution outside the Mac App Store"
          >
            <ConfigField label="Team ID" value={String(t['appleTeamId'] ?? '')} />
          </ConfigSection>
          <ConfigSection
            title="Notarization"
            description="Apple notarization ensures your app is checked for malicious content"
          >
            <ConfigField label="Apple ID" value={String(t['appleId'] ?? '')} />
            <ConfigField
              label="App-specific password"
              value={t['notarizationPassword'] != null ? '********' : ''}
            />
          </ConfigSection>
        </>
      )}
      {showWin && (
        <SigningUploadCard
          title="Windows Code Signing Certificate"
          description="Upload your code signing certificate (.pfx) for Windows executables"
          buttonLabel="Upload .pfx certificate"
        />
      )}
      {showLinux && (
        <ConfigSection
          title="Linux GPG Signing"
          description="Configure GPG key for signing Linux packages"
        >
          <ConfigField label="GPG Key ID" value={String(t['gpgKeyId'] ?? '')} />
          <ConfigField
            label="GPG Key Server"
            value={String(t['gpgKeyServer'] ?? 'hkps://keys.openpgp.org')}
          />
        </ConfigSection>
      )}
    </div>
  );
}

function DistributionContent({ t, platform }: { t: Target; platform: string }) {
  const showMac = platform === 'macos' || platform === 'cross-platform';
  const showWin = platform === 'windows' || platform === 'cross-platform';
  const showLinux = platform === 'linux' || platform === 'cross-platform';
  return (
    <div className="flex flex-col gap-4">
      <div className="text-[length:var(--text-title)] font-semibold text-text-primary">
        Distribution
      </div>
      {showMac && (
        <DistributionSection icon={Apple} title="macOS Distribution">
          <DistributionItem
            title="DMG Installer"
            description="Create a .dmg disk image for drag-and-drop install"
            enabled={t['dmgEnabled'] === true}
          />
          <DistributionItem
            title="PKG Installer"
            description="Create a .pkg installer package"
            enabled={t['pkgEnabled'] === true}
          />
          <DistributionItem
            title="Homebrew Cask"
            description="Publish to a Homebrew tap for `brew install` support"
            enabled={t['homebrewEnabled'] === true}
          />
        </DistributionSection>
      )}
      {showWin && (
        <DistributionSection icon={Monitor} title="Windows Distribution">
          <DistributionItem
            title="MSIX Package"
            description="Create MSIX package for modern Windows deployment"
            enabled={t['msixEnabled'] === true}
          />
          <DistributionItem
            title="NSIS Installer"
            description="Create a traditional .exe installer with NSIS"
            enabled={t['nsisEnabled'] === true}
          />
          <DistributionItem
            title="Microsoft Store"
            description="Publish to the Microsoft Store"
            enabled={t['msStoreEnabled'] === true}
          />
        </DistributionSection>
      )}
      {showLinux && (
        <DistributionSection icon={Terminal} title="Linux Distribution">
          <DistributionItem
            title="DEB Package"
            description="Create .deb package for Debian/Ubuntu"
            enabled={t['debEnabled'] === true}
          />
          <DistributionItem
            title="RPM Package"
            description="Create .rpm package for Fedora/RHEL"
            enabled={t['rpmEnabled'] === true}
          />
          <DistributionItem
            title="AppImage"
            description="Create portable AppImage binary"
            enabled={t['appImageEnabled'] === true}
          />
          <DistributionItem
            title="Flatpak"
            description="Create Flatpak package for Flathub"
            enabled={t['flatpakEnabled'] === true}
          />
          <DistributionItem
            title="Snap"
            description="Create Snap package for the Snap Store"
            enabled={t['snapEnabled'] === true}
          />
        </DistributionSection>
      )}
    </div>
  );
}

function DistributionSection({
  icon: Icon,
  title,
  children,
}: {
  icon: LucideIcon;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-[var(--radius)] border border-border bg-surface p-5">
      <div className="flex items-center gap-2">
        <Icon size={16} className="text-[var(--color-accent)]" />
        <div className="text-[length:var(--text-control)] font-semibold text-text-primary">
          {title}
        </div>
      </div>
      <div className="mt-4 flex flex-col gap-3">{children}</div>
    </div>
  );
}

function DistributionItem({
  title,
  description,
  enabled,
}: {
  title: string;
  description: string;
  enabled: boolean;
}) {
  return (
    <div className="flex items-center gap-3">
      <div
        className="flex h-8 w-8 shrink-0 items-center justify-center rounded-[var(--radius-6)]"
        style={{
          backgroundColor: enabled
            ? 'color-mix(in srgb, var(--status-success) 10%, transparent)'
            : 'color-mix(in srgb, var(--color-text-subtle) 5%, transparent)',
          color: enabled ? 'var(--status-success)' : 'var(--color-text-subtle)',
        }}
      >
        {enabled ? <Check size={14} /> : <Minus size={14} />}
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-[length:var(--text-body)] font-medium text-text-primary">{title}</div>
        <div className="text-[length:var(--text-caption)] text-text-subtle">{description}</div>
      </div>
      <button
        type="button"
        className="text-[length:var(--text-label)] font-medium"
        style={{ color: enabled ? 'var(--status-success)' : 'var(--color-accent)' }}
      >
        {enabled ? 'Configured' : 'Configure'}
      </button>
    </div>
  );
}
