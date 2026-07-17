import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { Activity, AlertTriangle, Timer, Zap } from 'lucide-react';
import { api } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { useTabIndex } from '@/hooks/use-tab-param';
import { PageTabs } from '@/components/page-tabs';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { StatusChip } from '@/components/status-chip';
import {
  FormDialog,
  FormField,
  SelectField,
  TextField,
} from '@/components/form-dialog';
import { Textarea } from '@/components/ui/textarea';
import {
  RUNTIME_OPTIONS,
  runtimeById,
  shortDate,
} from './runtimes';
import { SourceTypeToggle, type SourceType } from './SourceTypeToggle';
import { FunctionDetail } from './FunctionDetail';

const LIST_TABS = ['Functions', 'Usage'];

const COLUMNS: DataTableColumn[] = [
  { key: 'name', label: 'Name', flex: 4, sortable: true },
  { key: 'runtime', label: 'Runtime', flex: 2 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'updatedAt', label: 'Updated', flex: 2 },
];

export function FunctionsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [tab, setTab] = useTabIndex(LIST_TABS);
  const [selected, setSelected] = useState<Row | null>(null);

  const list = useResourceList({
    endpoint: '/functions',
    itemsKey: 'functions',
    scope: [projectId],
  });

  if (selected) {
    return (
      <FunctionDetail
        fn={selected}
        onChange={setSelected}
        onBack={() => setSelected(null)}
        onDeleted={() => {
          setSelected(null);
          list.refetch();
        }}
      />
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Functions</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Deploy serverless functions that run on demand in isolated containers
        </p>
      </div>

      <PageTabs tabs={LIST_TABS} selected={tab} onChange={setTab} />

      {tab === 0 ? (
        <FunctionsTab list={list} onOpen={setSelected} />
      ) : (
        <UsageTab />
      )}
    </div>
  );
}

function FunctionsTab({
  list,
  onOpen,
}: {
  list: ReturnType<typeof useResourceList<Row>>;
  onOpen: (row: Row) => void;
}) {
  const [creating, setCreating] = useState(false);

  return (
    <>
      <DataTable
        columns={COLUMNS}
        rows={list.rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'name':
              return String(row['name'] ?? '');
            case 'runtime':
              return String(row['runtime'] ?? '');
            case 'status':
              return String(row['status'] ?? 'active');
            case 'updatedAt':
              return shortDate(row['updatedAt']);
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'status') return <StatusChip label={String(row['status'] ?? 'active')} />;
          if (key === 'runtime') return runtimeById(String(row['runtime'] ?? 'custom')).label;
          return undefined;
        }}
        rowIcon={(row) => runtimeById(String(row['runtime'] ?? 'custom')).icon}
        onRowClick={onOpen}
        onDeleteRow={async (row) => {
          await api.delete(`/functions/${String(row['$id'])}`);
          list.refetch();
        }}
        gridCard={(row) => <FunctionGridCard fn={row} />}
        persistKey="functions_view"
        createLabel="Create function"
        onCreate={() => setCreating(true)}
        searchHint="Search functions…"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="functions"
        filters={[
          { key: 'runtime', label: 'Runtime', options: RUNTIME_OPTIONS },
          {
            key: 'status',
            label: 'Status',
            options: ['active', 'inactive', 'building', 'failed'].map((s) => ({
              value: s,
              label: s,
            })),
          },
        ]}
        filterValues={list.filters}
        onFiltersChange={list.setFilters}
        emptyIcon={Zap}
        emptyTitle="No functions yet"
        emptySubtitle="Write backend logic that runs on demand in any language."
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      <CreateFunctionDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={() => list.refetch()}
      />
    </>
  );
}

function FunctionGridCard({ fn }: { fn: Row }) {
  const name = String(fn['name'] ?? 'Unnamed');
  const runtime = runtimeById(String(fn['runtime'] ?? 'custom'));
  const status = String(fn['status'] ?? 'inactive');
  const Icon = runtime.icon;
  return (
    <div className="flex h-full flex-col gap-3 rounded-[var(--radius-10)] border border-border bg-surface p-4 transition-colors hover:border-field-border hover:bg-fill-hover">
      <div className="flex h-9 w-9 items-center justify-center rounded-[var(--radius)] bg-[color-mix(in_srgb,var(--color-accent)_10%,transparent)] text-[var(--color-accent)]">
        <Icon size={16} />
      </div>
      <div className="min-w-0">
        <div className="truncate text-[length:var(--text-body)] font-medium text-text-primary">
          {name}
        </div>
        <div className="text-[length:var(--text-caption)] text-text-secondary">{runtime.label}</div>
      </div>
      <div className="mt-auto flex items-center justify-between">
        <StatusChip label={status} />
        <span className="text-[length:var(--text-caption)] text-text-subtle">
          {shortDate(fn['updatedAt'])}
        </span>
      </div>
    </div>
  );
}

function CreateFunctionDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState('');
  const [runtime, setRuntime] = useState('node-20');
  const [entrypoint, setEntrypoint] = useState('index.js');
  const [timeout, setTimeout] = useState('15');
  const [sourceType, setSourceType] = useState<SourceType>('inline');
  const [source, setSource] = useState('');
  const [repository, setRepository] = useState('');
  const [branch, setBranch] = useState('main');
  const [saving, setSaving] = useState(false);

  const reset = () => {
    setName('');
    setRuntime('node-20');
    setEntrypoint('index.js');
    setTimeout('15');
    setSourceType('inline');
    setSource('');
    setRepository('');
    setBranch('main');
  };

  const submit = async () => {
    setSaving(true);
    try {
      await api.post('/functions', {
        name: name.trim(),
        runtime,
        entrypoint: entrypoint.trim(),
        timeout: Number.parseInt(timeout, 10) || 15,
        sourceType,
        source: sourceType === 'inline' ? source : '',
        repository: sourceType === 'git' ? repository.trim() : '',
        branch: sourceType === 'git' ? branch.trim() : '',
      });
      onOpenChange(false);
      reset();
      onCreated();
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Create function"
      subtitle="Deploy serverless backend logic"
      submitLabel="Create"
      loading={saving}
      submitDisabled={!name.trim()}
      width={480}
      onSubmit={submit}
    >
      <TextField
        label="Name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="my-function"
        autoFocus
      />
      <SelectField label="Runtime" value={runtime} onChange={setRuntime} options={RUNTIME_OPTIONS} />
      <TextField
        label="Entrypoint"
        value={entrypoint}
        onChange={(e) => setEntrypoint(e.target.value)}
        placeholder="index.js"
      />
      <TextField
        label="Timeout (seconds)"
        type="number"
        value={timeout}
        onChange={(e) => setTimeout(e.target.value)}
        placeholder="15"
      />
      <FormField label="Source">
        <SourceTypeToggle value={sourceType} onChange={setSourceType} />
      </FormField>
      {sourceType === 'inline' ? (
        <FormField label="Source code">
          <Textarea
            value={source}
            onChange={(e) => setSource(e.target.value)}
            placeholder="// your code here"
            rows={6}
            className="font-[family-name:var(--font-mono)]"
          />
        </FormField>
      ) : (
        <>
          <TextField
            label="Repository URL"
            value={repository}
            onChange={(e) => setRepository(e.target.value)}
            placeholder="https://github.com/user/repo"
          />
          <TextField
            label="Branch"
            value={branch}
            onChange={(e) => setBranch(e.target.value)}
            placeholder="main"
          />
        </>
      )}
    </FormDialog>
  );
}

function UsageTab() {
  const stats: [string, typeof Activity, string][] = [
    ['Total executions', Activity, '—'],
    ['Avg duration', Timer, '—'],
    ['Failure rate', AlertTriangle, '—'],
  ];
  return (
    <div className="flex flex-col gap-6">
      <div>
        <div className="text-[length:var(--text-title)] font-semibold text-text-primary">Usage</div>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Function execution metrics for the past 30 days.
        </p>
      </div>
      <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
        {stats.map(([label, Icon, value]) => (
          <div key={label} className="rounded-[var(--radius-10)] border border-border bg-surface p-5">
            <Icon size={16} className="text-text-secondary" />
            <div className="mt-3 text-[length:var(--text-h2)] font-bold text-text-primary">{value}</div>
            <div className="mt-1 text-[length:var(--text-label)] text-text-secondary">{label}</div>
          </div>
        ))}
      </div>
      <div className="flex h-44 items-center justify-center rounded-[var(--radius)] border border-border bg-surface text-[length:var(--text-body)] text-text-subtle">
        Usage charts coming soon
      </div>
    </div>
  );
}
