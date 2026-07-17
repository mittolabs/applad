import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Plus, Smartphone, Tablet } from 'lucide-react';
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
  { key: 'buildType', label: 'Type', flex: 2 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'updatedAt', label: 'Updated', flex: 2 },
];

const DETAIL_TABS = ['Overview', 'Builds', 'Signing', 'Settings'];

export function MobilePage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const list = useResourceList({
    endpoint: '/deploy/targets',
    itemsKey: 'targets',
    params: { type: 'mobile' },
    scope: [projectId],
  });

  if (selectedId) {
    return (
      <MobileDetail
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
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Mobile Apps</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Build Android APK/AAB and iOS IPA from source.
        </p>
      </div>

      <MobileList list={list} onOpen={setSelectedId} />
    </div>
  );
}

function MobileList({
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
            case 'buildType':
              return String(row['buildType'] ?? '');
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
        rowIcon={(row) => (String(row['buildType'] ?? '') === 'ipa' ? Tablet : Smartphone)}
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
        emptyTitle="No mobile apps yet"
        emptySubtitle="Build Android APK/AAB and iOS IPA from source."
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      <DeployCreateEntry
        open={entryOpen}
        onOpenChange={setEntryOpen}
        category="mobile"
        title="Create Mobile App"
        subtitle="Choose how to get started"
        onResult={(r) => setPending(r)}
      />

      <CreateMobileDialog
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

function CreateMobileDialog({
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
  const sourceType =
    result?.choice === 'template' ? 'template' : result?.choice === 'repository' ? 'git' : 'upload';

  const [name, setName] = useState('');
  const [platform, setPlatform] = useState<'android' | 'ios'>('android');
  const [saving, setSaving] = useState(false);

  // Re-seed the name each time a new entry result opens the dialog.
  const [seenResult, setSeenResult] = useState<CreateEntryResult | null>(null);
  if (result && result !== seenResult) {
    setSeenResult(result);
    setName(prefillName);
    setPlatform('android');
  }

  const submit = async () => {
    setSaving(true);
    try {
      const repo = result?.repoConfig;
      await api.post('/deploy/targets', {
        name: name.trim(),
        type: 'mobile',
        buildType: platform === 'ios' ? 'ipa' : 'apk',
        source: sourceType,
        ...(result?.templateConfig ? { templateId: result.templateConfig['$id'] } : {}),
        ...(repo
          ? {
              repository: (repo['cloneUrl'] as string) ?? (repo['url'] as string) ?? '',
              branch: (repo['defaultBranch'] as string) ?? 'main',
            }
          : {}),
      });
      toast.success('Mobile app created');
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
      title="Create Mobile App"
      subtitle="Build for Android or iOS"
      submitLabel="Create"
      loading={saving}
      submitDisabled={!name.trim()}
      onSubmit={submit}
    >
      <TextField
        label="App name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="my-app"
        autoFocus
      />
      <FormField label="Platform">
        <div className="grid grid-cols-2 gap-2">
          <ChoiceChip
            label="Android"
            icon={Smartphone}
            active={platform === 'android'}
            onClick={() => setPlatform('android')}
          />
          <ChoiceChip
            label="iOS"
            icon={Tablet}
            active={platform === 'ios'}
            onClick={() => setPlatform('ios')}
          />
        </div>
      </FormField>
    </FormDialog>
  );
}

function MobileDetail({
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
  const buildType = String(t['buildType'] ?? 'apk');
  const platformLabel = buildType === 'ipa' ? 'iOS' : 'Android';

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
        <div className="flex flex-col gap-6">
          <div className="flex gap-4">
            <InfoCard label="Platform" value={platformLabel} />
            <InfoCard label="Build Type" value={buildType} />
            <InfoCard label="Runtime" value={String(t['runtime'] ?? '—')} />
          </div>
          <div>
            <div className="text-[length:var(--text-title)] font-semibold text-text-primary">
              Distribution
            </div>
            <div className="mt-3 rounded-[var(--radius)] border border-border bg-surface p-4 text-[length:var(--text-body)] text-text-secondary">
              Store publishing and distribution can be configured in Settings.
            </div>
          </div>
        </div>
      )}

      {tab === 1 && <BuildsTab targetId={id} />}

      {tab === 2 && (
        <div className="flex flex-col gap-4">
          <div className="text-[length:var(--text-title)] font-semibold text-text-primary">
            Code Signing
          </div>
          <SigningUploadCard
            title="Android Keystore"
            description="Upload your keystore file for signed APK/AAB builds"
            buttonLabel="Upload keystore"
          />
          <SigningUploadCard
            title="iOS Provisioning"
            description="Upload provisioning profile + P12 certificate for IPA builds"
            buttonLabel="Upload profile"
          />
        </div>
      )}

      {tab === 3 && (
        <div className="flex flex-col gap-6">
          <div className="text-[length:var(--text-title)] font-semibold text-text-primary">
            App Settings
          </div>
          <div className="flex flex-col">
            <SettingRow label="Name" value={String(t['name'] ?? '')} />
            <SettingRow label="Platform" value={platformLabel} />
            <SettingRow label="Build Type" value={buildType} />
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
