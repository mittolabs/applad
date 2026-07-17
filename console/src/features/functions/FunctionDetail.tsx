import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import Editor from '@monaco-editor/react';
import { Activity, ChevronLeft, Play, Plus, Trash2, Variable } from 'lucide-react';
import { api } from '@/api/client';
import { useMonacoTheme } from '@/stores/theme';
import { PageTabs } from '@/components/page-tabs';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { StatusChip } from '@/components/status-chip';
import { EmptyState } from '@/components/empty-state';
import { CodeBlock } from '@/components/code-block';
import { IdText } from '@/components/id-text';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
} from '@/components/ui/dialog';
import {
  ConfirmDialog,
  FormDialog,
  SelectField,
  TextField,
} from '@/components/form-dialog';
import {
  RUNTIME_OPTIONS,
  relativeTime,
  runtimeById,
} from './runtimes';
import { SourceTypeToggle, type SourceType } from './SourceTypeToggle';

const DETAIL_TABS = ['Executions', 'Variables', 'Settings'];

export function FunctionDetail({
  fn,
  onChange,
  onBack,
  onDeleted,
}: {
  fn: Row;
  onChange: (fn: Row) => void;
  onBack: () => void;
  onDeleted: () => void;
}) {
  const [tab, setTab] = useState(0);
  const [running, setRunning] = useState(false);
  const id = String(fn['$id']);
  const name = String(fn['name'] ?? 'Function');

  const executions = useQuery({
    queryKey: ['function-executions', id],
    queryFn: async () => {
      const res = await api.get(`/functions/${id}/executions`, { params: { limit: 25 } });
      return res.data as Record<string, unknown>;
    },
  });

  const run = async () => {
    setRunning(true);
    try {
      await api.post(`/functions/${id}/executions`, {});
      await executions.refetch();
    } finally {
      setRunning(false);
    }
  };

  return (
    <div className="flex flex-col gap-4 p-6 md:p-8">
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1 text-[length:var(--text-body)] text-text-secondary transition-colors hover:text-text-primary"
        >
          <ChevronLeft size={16} />
          Functions
        </button>
        <span className="text-[length:var(--text-body)] text-text-subtle">/</span>
        <span className="text-[length:var(--text-body)] font-medium text-text-primary">{name}</span>
        <div className="ml-auto">
          <Button size="sm" loading={running} onClick={run}>
            <Play size={14} />
            Execute
          </Button>
        </div>
      </div>

      <PageTabs tabs={DETAIL_TABS} selected={tab} onChange={setTab} />

      {tab === 0 && <ExecutionsTab query={executions} onRun={run} running={running} />}
      {tab === 1 && <VariablesTab fn={fn} onChange={onChange} />}
      {tab === 2 && <SettingsTab fn={fn} onChange={onChange} onDeleted={onDeleted} />}
    </div>
  );
}

// ── Executions ────────────────────────────────────────────────────────────────

const EXECUTION_COLUMNS: DataTableColumn[] = [
  { key: '$id', label: 'Execution ID', flex: 3 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'duration', label: 'Duration', flex: 2 },
  { key: 'triggered', label: 'Triggered', flex: 3 },
];

function ExecutionsTab({
  query,
  onRun,
  running,
}: {
  query: ReturnType<typeof useQuery<Record<string, unknown>>>;
  onRun: () => void;
  running: boolean;
}) {
  const rows = ((query.data?.['executions'] as Row[] | undefined) ?? []);
  const [selected, setSelected] = useState<Row | null>(null);

  return (
    <>
      <DataTable
        columns={EXECUTION_COLUMNS}
        rows={rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'status':
              return String(row['status'] ?? 'pending');
            case 'duration':
              return `${Math.round(Number(row['duration'] ?? 0))} ms`;
            case 'triggered':
              return relativeTime(String(row['$createdAt'] ?? ''));
            default:
              return '';
          }
        }}
        cellRender={(row, key) =>
          key === 'status' ? <StatusChip label={String(row['status'] ?? 'pending')} /> : undefined
        }
        rowIcon={() => Activity}
        onRowClick={(row) => setSelected(row)}
        createLabel={running ? 'Running…' : 'Run'}
        onCreate={onRun}
        emptyIcon={Activity}
        emptyTitle="No executions yet"
        emptySubtitle="Trigger your first execution above."
        loading={query.isLoading}
        error={query.error}
        onRetry={() => query.refetch()}
      />

      <ExecutionDialog execution={selected} onClose={() => setSelected(null)} />
    </>
  );
}

function ExecutionDialog({
  execution,
  onClose,
}: {
  execution: Row | null;
  onClose: () => void;
}) {
  const status = String(execution?.['status'] ?? 'pending');
  const duration = Math.round(Number(execution?.['duration'] ?? 0));
  const createdAt = String(execution?.['$createdAt'] ?? '');
  const output = String(execution?.['output'] ?? '');
  const errors = String(execution?.['errors'] ?? '');

  return (
    <Dialog open={!!execution} onOpenChange={(o) => !o && onClose()}>
      <DialogContent width={640}>
        <DialogHeader>
          <DialogTitle>Execution</DialogTitle>
          {execution && (
            <div className="mt-1 flex flex-wrap items-center gap-3">
              <IdText id={String(execution['$id'] ?? '')} />
              <StatusChip label={status} />
              <span className="text-[length:var(--text-label)] text-text-secondary">
                {duration} ms
              </span>
              {createdAt && (
                <span className="text-[length:var(--text-label)] text-text-subtle">
                  {relativeTime(createdAt)}
                </span>
              )}
            </div>
          )}
        </DialogHeader>
        <DialogBody className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <span className="text-[length:var(--text-label)] font-medium text-text-secondary">
              Output
            </span>
            {output ? (
              <CodeBlock code={output} />
            ) : (
              <span className="text-[length:var(--text-body)] text-text-subtle">No output.</span>
            )}
          </div>
          <div className="flex flex-col gap-1.5">
            <span className="text-[length:var(--text-label)] font-medium text-text-secondary">
              Errors
            </span>
            {errors ? (
              <pre className="overflow-x-auto rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_5%,transparent)] px-3 py-2.5 font-[family-name:var(--font-mono)] text-[11.5px] leading-relaxed text-[var(--color-danger)]">
                {errors}
              </pre>
            ) : (
              <span className="text-[length:var(--text-body)] text-text-subtle">No errors.</span>
            )}
          </div>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

// ── Variables ─────────────────────────────────────────────────────────────────

function readVars(fn: Row): Record<string, string> {
  const raw = (fn['envVars'] as Record<string, unknown> | undefined) ?? {};
  return Object.fromEntries(Object.entries(raw).map(([k, v]) => [k, String(v)]));
}

function VariablesTab({ fn, onChange }: { fn: Row; onChange: (fn: Row) => void }) {
  const vars = readVars(fn);
  const entries = Object.entries(vars);
  const [adding, setAdding] = useState(false);
  const [key, setKey] = useState('');
  const [value, setValue] = useState('');
  const [saving, setSaving] = useState(false);

  const saveVars = async (next: Record<string, string>) => {
    const res = await api.put(`/functions/${String(fn['$id'])}`, { ...fn, envVars: next });
    onChange(res.data as Row);
  };

  const addVar = async () => {
    if (!key.trim()) return;
    setSaving(true);
    try {
      await saveVars({ ...vars, [key.trim()]: value });
      setAdding(false);
      setKey('');
      setValue('');
    } finally {
      setSaving(false);
    }
  };

  const deleteVar = async (k: string) => {
    const next = { ...vars };
    delete next[k];
    await saveVars(next);
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <span className="text-[length:var(--text-body)] text-text-secondary">
          {entries.length} variable{entries.length === 1 ? '' : 's'}
        </span>
        <Button variant="outline" size="sm" onClick={() => setAdding(true)}>
          <Plus size={14} />
          Add variable
        </Button>
      </div>

      {entries.length === 0 ? (
        <EmptyState
          icon={Variable}
          title="No variables yet"
          subtitle="Add environment variables for your function to use at runtime."
        />
      ) : (
        <div className="flex flex-col gap-2">
          {entries.map(([k, v]) => (
            <div
              key={k}
              className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-surface px-3.5 py-3"
            >
              <span className="flex-[2] truncate font-[family-name:var(--font-mono)] text-[length:var(--text-body)] font-medium text-text-primary">
                {k}
              </span>
              <span className="flex-[3] truncate font-[family-name:var(--font-mono)] text-[length:var(--text-body)] text-text-secondary">
                {v.length > 32 ? `${v.slice(0, 32)}…` : v}
              </span>
              <button
                type="button"
                onClick={() => deleteVar(k)}
                className="rounded-[var(--radius-6)] p-1.5 text-text-subtle transition-colors hover:bg-fill hover:text-[var(--color-danger)]"
                aria-label={`Delete ${k}`}
              >
                <Trash2 size={14} />
              </button>
            </div>
          ))}
        </div>
      )}

      <FormDialog
        open={adding}
        onOpenChange={setAdding}
        title="Add variable"
        submitLabel="Add"
        loading={saving}
        submitDisabled={!key.trim()}
        onSubmit={addVar}
      >
        <TextField
          label="Key"
          value={key}
          onChange={(e) => setKey(e.target.value)}
          placeholder="API_KEY"
          autoFocus
        />
        <TextField
          label="Value"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="your-secret-value"
        />
      </FormDialog>
    </div>
  );
}

// ── Settings ──────────────────────────────────────────────────────────────────

function SettingsTab({
  fn,
  onChange,
  onDeleted,
}: {
  fn: Row;
  onChange: (fn: Row) => void;
  onDeleted: () => void;
}) {
  const monacoTheme = useMonacoTheme();
  const id = String(fn['$id']);
  const [name, setName] = useState(String(fn['name'] ?? ''));
  const [runtime, setRuntime] = useState(String(fn['runtime'] ?? 'node-20'));
  const [entrypoint, setEntrypoint] = useState(String(fn['entrypoint'] ?? ''));
  const [timeout, setTimeout] = useState(String(fn['timeout'] ?? 15));
  const [cron, setCron] = useState(String(fn['cron'] ?? ''));
  const [sourceType, setSourceType] = useState<SourceType>(
    (fn['sourceType'] as SourceType | undefined) ?? 'inline',
  );
  const [source, setSource] = useState(String(fn['source'] ?? ''));
  const [repository, setRepository] = useState(String(fn['repository'] ?? ''));
  const [branch, setBranch] = useState(String(fn['branch'] ?? 'main'));
  const [saving, setSaving] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const save = async () => {
    setSaving(true);
    try {
      const res = await api.put(`/functions/${id}`, {
        name: name.trim(),
        runtime,
        entrypoint: entrypoint.trim(),
        timeout: Number.parseInt(timeout, 10) || 15,
        sourceType,
        source: sourceType === 'inline' ? source : '',
        repository: sourceType === 'git' ? repository.trim() : '',
        branch: sourceType === 'git' ? branch.trim() : '',
        cron: cron.trim(),
      });
      onChange(res.data as Row);
    } finally {
      setSaving(false);
    }
  };

  const del = async () => {
    setDeleting(true);
    try {
      await api.delete(`/functions/${id}`);
      onDeleted();
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="flex max-w-2xl flex-col gap-3">
      <SectionCard title="General" description="Basic function configuration.">
        <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} placeholder="my-function" />
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
        <TextField
          label="Schedule (cron)"
          value={cron}
          onChange={(e) => setCron(e.target.value)}
          placeholder="0 * * * *"
          hint="Standard 5-field cron expression (minute hour day month weekday). Leave empty for manual execution only."
        />
      </SectionCard>

      <SectionCard title="Source" description="Deploy from inline code or a GitHub repository.">
        <SourceTypeToggle value={sourceType} onChange={setSourceType} />
        {sourceType === 'inline' ? (
          <div className="overflow-hidden rounded-[var(--radius)] border border-field-border">
            <Editor
              height="320px"
              theme={monacoTheme}
              language={runtimeById(runtime).language}
              value={source}
              onChange={(v) => setSource(v ?? '')}
              options={{ minimap: { enabled: false }, fontSize: 13, scrollBeyondLastLine: false }}
            />
          </div>
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
      </SectionCard>

      <div>
        <Button loading={saving} onClick={save}>
          Save changes
        </Button>
      </div>

      <div className="mt-6 rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-danger)_20%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_5%,transparent)] p-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <div className="text-[length:var(--text-control)] font-semibold text-[var(--color-danger)]">
              Delete function
            </div>
            <div className="mt-0.5 text-[length:var(--text-label)] text-text-subtle">
              Permanently removes the function and all execution history.
            </div>
          </div>
          <Button variant="outline" onClick={() => setConfirmDelete(true)}>
            Delete
          </Button>
        </div>
      </div>

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Delete function"
        message="This will permanently delete the function and all its executions."
        loading={deleting}
        onConfirm={del}
      />
    </div>
  );
}

function SectionCard({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-4 rounded-[var(--radius-10)] border border-border bg-surface p-4">
      <div>
        <div className="text-[length:var(--text-control)] font-semibold text-text-primary">{title}</div>
        <div className="mt-0.5 text-[length:var(--text-label)] text-text-subtle">{description}</div>
      </div>
      {children}
    </div>
  );
}
