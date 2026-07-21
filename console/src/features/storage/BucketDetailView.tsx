import { useRef, useState } from 'react';
import { useTabIndex } from '@/hooks/use-tab-param';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, File as FileIcon, FolderClosed } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { PageTabs } from '@/components/page-tabs';
import { IdText } from '@/components/id-text';
import { BucketSettings } from './BucketSettings';
import { formatSize, mimeIcon, timeAgo } from './helpers';

const TABS = ['Files', 'Usage', 'Settings'];

const FILE_COLUMNS: DataTableColumn[] = [
  { key: 'name', label: 'Filename', flex: 4, sortable: true },
  { key: 'mimeType', label: 'Type', flex: 2 },
  { key: 'size', label: 'Size', flex: 1 },
  { key: 'createdAt', label: 'Created', flex: 2 },
];

export function BucketDetailView({
  bucketId,
  onBack,
  onFileSelect,
}: {
  bucketId: string;
  onBack: () => void;
  onFileSelect: (fileId: string) => void;
}) {
  // In the URL so a refresh stays on the tab somebody was reading.
  const [tab, setTab] = useTabIndex(TABS, undefined, 'view');

  const { data: bucket } = useQuery({
    queryKey: ['storage-bucket', bucketId],
    queryFn: async () => {
      const res = await api.get(`/storage/buckets/${bucketId}`);
      return res.data as Record<string, unknown>;
    },
  });

  const bucketName = (bucket?.['name'] as string) ?? bucketId;

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={onBack}
          className="text-text-muted transition-colors hover:text-text-primary"
          aria-label="Back"
        >
          <ArrowLeft size={20} />
        </button>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
          {bucketName}
        </h1>
        <div className="flex items-center gap-1.5">
          <FolderClosed size={13} className="text-text-subtle" />
          <IdText id={bucketId} />
        </div>
      </div>

      <PageTabs tabs={TABS} selected={tab} onChange={setTab} />

      {tab === 0 && <FilesTab bucketId={bucketId} onFileSelect={onFileSelect} />}
      {tab === 1 && <BucketUsageTab />}
      {tab === 2 && <BucketSettings bucketId={bucketId} bucket={bucket} onDeleted={onBack} />}
    </div>
  );
}

function FilesTab({
  bucketId,
  onFileSelect,
}: {
  bucketId: string;
  onFileSelect: (fileId: string) => void;
}) {
  const qc = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const list = useResourceList({
    endpoint: `/storage/buckets/${bucketId}/files`,
    itemsKey: 'files',
    scope: [bucketId],
    defaultPerPage: 100,
  });

  const invalidate = () => {
    list.refetch();
    qc.invalidateQueries({ queryKey: ['storage-files', bucketId] });
  };

  const upload = useMutation({
    mutationFn: (file: File) => {
      const form = new FormData();
      // Appwrite-style server assigns an id when fileId is "unique()".
      form.append('fileId', 'unique()');
      form.append('file', file);
      return api.post(`/storage/buckets/${bucketId}/files`, form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
    },
    onSuccess: invalidate,
    onError: (e) => setUploadError(friendlyError(e)),
  });

  const onPick = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (file) {
      setUploadError(null);
      upload.mutate(file);
    }
  };

  const getCellValue = (row: Row, key: string) => {
    switch (key) {
      case 'name':
        return (row['name'] as string) ?? 'Untitled';
      case 'mimeType':
        return (row['mimeType'] as string) ?? '';
      case 'size':
        return formatSize(row['sizeOriginal']);
      case 'createdAt':
        return timeAgo(row['createdAt'] ?? row['$createdAt']);
      default:
        return String(row[key] ?? '');
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <input ref={fileInputRef} type="file" className="hidden" onChange={onPick} />
      {uploadError && (
        <div className="rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-danger)_30%,var(--border))] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[length:var(--text-body)] text-[var(--color-danger)]">
          {uploadError}
        </div>
      )}
      <DataTable
        columns={FILE_COLUMNS}
        rows={list.rows}
        getCellValue={getCellValue}
        rowIcon={(row) => mimeIcon((row['mimeType'] as string) ?? '').icon}
        rowIconColor={(row) => mimeIcon((row['mimeType'] as string) ?? '').color}
        onRowClick={(row) => onFileSelect(String(row['$id']))}
        onDeleteRow={async (row) => {
          await api.delete(`/storage/buckets/${bucketId}/files/${row['$id']}`);
          invalidate();
        }}
        createLabel={upload.isPending ? 'Uploading…' : 'Upload file'}
        onCreate={() => !upload.isPending && fileInputRef.current?.click()}
        searchHint="Search files"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="Files"
        emptyIcon={FileIcon}
        emptyTitle="No files"
        emptySubtitle="Upload a file to this bucket"
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />
    </div>
  );
}

function BucketUsageTab() {
  const cards = [
    { label: 'Total Files', value: '—' },
    { label: 'Storage Used', value: '—' },
    { label: 'Bandwidth', value: '—' },
  ];
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
      {cards.map((c) => (
        <div key={c.label} className="rounded-[var(--radius)] border border-border bg-surface p-5">
          <div className="text-[length:var(--text-h2)] font-bold text-text-primary">{c.value}</div>
          <div className="mt-1 text-[length:var(--text-label)] text-text-muted">{c.label}</div>
        </div>
      ))}
    </div>
  );
}
