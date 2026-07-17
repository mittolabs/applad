import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { File as FileIcon, FolderClosed, HardDrive } from 'lucide-react';
import { api } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { useTabIndex } from '@/hooks/use-tab-param';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { PageTabs } from '@/components/page-tabs';
import { IdText } from '@/components/id-text';
import { FormDialog, TextField } from '@/components/form-dialog';
import { BucketDetailView } from './BucketDetailView';
import { FileDetailView } from './FileDetailView';
import { fmtDate } from './helpers';

const TABS = ['Buckets', 'Usage'];

const COLUMNS: DataTableColumn[] = [
  { key: '$id', label: 'Bucket ID', flex: 3 },
  { key: 'name', label: 'Name', flex: 3, sortable: true },
  { key: 'createdAt', label: 'Created', flex: 2 },
  { key: 'updatedAt', label: 'Updated', flex: 2 },
];

export function StoragePage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [tab, setTab] = useTabIndex(TABS);
  const [selectedBucketId, setSelectedBucketId] = useState<string | null>(null);
  const [selectedFileId, setSelectedFileId] = useState<string | null>(null);

  if (selectedBucketId && selectedFileId) {
    return (
      <FileDetailView
        bucketId={selectedBucketId}
        fileId={selectedFileId}
        onBack={() => setSelectedFileId(null)}
      />
    );
  }

  if (selectedBucketId) {
    return (
      <BucketDetailView
        bucketId={selectedBucketId}
        onBack={() => setSelectedBucketId(null)}
        onFileSelect={setSelectedFileId}
      />
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Storage</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Store and serve files with buckets, image transforms and access policies
        </p>
      </div>

      <PageTabs tabs={TABS} selected={tab} onChange={setTab} />

      {tab === 0 ? (
        <BucketsTab projectId={projectId} onSelect={setSelectedBucketId} />
      ) : (
        <UsageTab />
      )}
    </div>
  );
}

function BucketsTab({
  projectId,
  onSelect,
}: {
  projectId: string | undefined;
  onSelect: (id: string) => void;
}) {
  const qc = useQueryClient();
  const list = useResourceList({
    endpoint: '/storage/buckets',
    itemsKey: 'buckets',
    scope: [projectId],
  });

  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');

  const create = useMutation({
    mutationFn: () =>
      api.post('/storage/buckets', {
        bucketId: 'unique()',
        name: name.trim(),
        permissions: [],
        allowedFileExtensions: [],
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['/storage/buckets'] });
      setCreating(false);
      setName('');
    },
  });

  const getCellValue = (row: Row, key: string) => {
    switch (key) {
      case 'name':
        return (row['name'] as string) ?? '';
      case 'createdAt':
        return fmtDate(row['createdAt'] ?? row['$createdAt']);
      case 'updatedAt':
        return fmtDate(row['updatedAt'] ?? row['$updatedAt']);
      default:
        return String(row[key] ?? '');
    }
  };

  return (
    <>
      <DataTable
        columns={COLUMNS}
        rows={list.rows}
        getCellValue={getCellValue}
        rowIcon={() => FolderClosed}
        onRowClick={(row) => onSelect(String(row['$id']))}
        onDeleteRow={async (row) => {
          await api.delete(`/storage/buckets/${row['$id']}`);
          list.refetch();
        }}
        gridCard={(row) => <BucketGridCard bucket={row} />}
        persistKey="storage-buckets"
        createLabel="Create bucket"
        onCreate={() => setCreating(true)}
        searchHint="Search buckets"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="Buckets"
        emptyIcon={FolderClosed}
        emptyTitle="No buckets"
        emptySubtitle="Create a bucket to start storing files"
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      <FormDialog
        open={creating}
        onOpenChange={setCreating}
        title="Create bucket"
        subtitle="Storage buckets organize your files"
        submitLabel="Create"
        loading={create.isPending}
        submitDisabled={!name.trim()}
        onSubmit={() => create.mutate()}
      >
        <TextField
          label="Bucket name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. user-avatars"
          autoFocus
        />
      </FormDialog>
    </>
  );
}

function BucketGridCard({ bucket }: { bucket: Row }) {
  const id = String(bucket['$id'] ?? '');
  const name = (bucket['name'] as string) ?? '';
  return (
    <div className="flex h-full flex-col gap-3 rounded-[var(--radius-10)] border border-border bg-surface p-4 transition-colors hover:border-[color-mix(in_srgb,var(--color-accent)_35%,var(--border))] hover:bg-fill-hover">
      <FolderClosed size={20} className="text-[var(--color-accent)]" />
      <div className="truncate text-[length:var(--text-control)] font-medium text-text-primary">
        {name}
      </div>
      <div className="mt-auto self-start rounded-[var(--radius-6)] border border-border bg-fill px-2 py-1">
        <IdText id={id} fontSize={11} />
      </div>
    </div>
  );
}

function UsageTab() {
  const cards = [
    { label: 'Total buckets', value: '—', icon: FolderClosed },
    { label: 'Total files', value: '—', icon: FileIcon },
    { label: 'Storage used', value: '—', icon: HardDrive },
  ];
  return (
    <div className="flex flex-col gap-6">
      <div>
        <div className="text-[length:var(--text-title)] font-semibold text-text-primary">Usage</div>
        <div className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Storage activity for the past 30 days.
        </div>
      </div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {cards.map(({ label, value, icon: Icon }) => (
          <div key={label} className="rounded-[var(--radius)] border border-border bg-surface p-5">
            <Icon size={16} className="text-text-secondary" />
            <div className="mt-3 text-[length:var(--text-h2)] font-bold text-text-primary">
              {value}
            </div>
            <div className="mt-1 text-[length:var(--text-label)] text-text-secondary">{label}</div>
          </div>
        ))}
      </div>
      <div className="flex h-48 items-center justify-center rounded-[var(--radius)] border border-border bg-surface text-[length:var(--text-body)] text-text-subtle">
        Usage charts coming soon
      </div>
    </div>
  );
}
