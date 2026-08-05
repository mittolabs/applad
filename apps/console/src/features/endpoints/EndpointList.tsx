import { useState } from 'react';
import { Workflow } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { StatusChip } from '@/components/status-chip';
import { FormDialog, TextField, SelectField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { AUTH_OPTIONS, HTTP_METHODS } from './blockDefs';

const COLUMNS: DataTableColumn[] = [
  { key: 'method', label: 'Method', flex: 1 },
  { key: 'path', label: 'Path', flex: 4, sortable: true },
  { key: 'name', label: 'Name', flex: 3 },
  { key: 'auth', label: 'Auth', flex: 2 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'updatedAt', label: 'Updated', flex: 2 },
];

const METHOD_COLOR: Record<string, string> = {
  GET: '#35c07f',
  POST: '#6f8cff',
  PUT: '#f5a623',
  PATCH: '#a08cff',
  DELETE: '#e5484d',
};

export function MethodBadge({ method }: { method: string }) {
  const color = METHOD_COLOR[method] ?? 'var(--color-text-muted)';
  return (
    <span
      className="inline-flex rounded-[var(--radius-sm)] px-1.5 py-0.5 font-mono text-[length:var(--text-caption)] font-semibold"
      style={{ color, backgroundColor: `color-mix(in srgb, ${color} 12%, transparent)` }}
    >
      {method}
    </span>
  );
}

function relativeTime(v: unknown): string {
  if (!v) return '';
  const dt = new Date(String(v));
  if (Number.isNaN(dt.getTime())) return String(v);
  const m = Math.floor((Date.now() - dt.getTime()) / 60000);
  if (m < 1) return 'just now';
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d}d ago`;
  return dt.toLocaleDateString();
}

export function EndpointList({
  projectId,
  onSelect,
}: {
  projectId: string | undefined;
  onSelect: (row: Row) => void;
}) {
  const list = useResourceList<Row>({
    endpoint: '/endpoints',
    itemsKey: 'endpoints',
    scope: [projectId],
  });
  const [creating, setCreating] = useState(false);

  const cellRender = (row: Row, key: string): React.ReactNode | undefined => {
    if (key === 'method') return <MethodBadge method={String(row.method ?? 'GET')} />;
    if (key === 'status') return <StatusChip label={String(row.status ?? 'draft')} />;
    if (key === 'path')
      return <span className="font-mono text-text-primary">{String(row.path ?? '')}</span>;
    if (key === 'auth')
      return <span className="capitalize text-text-secondary">{String(row.auth ?? 'public')}</span>;
    if (key === 'updatedAt') return relativeTime(row['$updatedAt'] ?? row.updatedAt);
    if (key === 'name')
      return <span className="text-text-secondary">{String(row.name ?? '')}</span>;
    return undefined;
  };

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Endpoints</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Compose a REST endpoint from blocks. No code, runs inline with no cold start.
        </p>
      </div>

      <DataTable
        columns={COLUMNS}
        rows={list.rows}
        emptyIcon={Workflow}
        emptyTitle="No endpoints yet"
        emptySubtitle="Create an endpoint to serve a REST route built from blocks."
        cellRender={cellRender}
        onRowClick={onSelect}
        onDeleteRow={async (row) => {
          try {
            await api.delete(`/endpoints/${row['$id']}`);
            toast.success('Endpoint deleted');
            list.refetch();
          } catch (e) {
            toast.error(friendlyError(e));
          }
        }}
        deleteTitle="Delete endpoint"
        deleteNameKey="path"
        createLabel="New endpoint"
        onCreate={() => setCreating(true)}
        searchHint="Search paths"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="endpoints"
      />

      <CreateEndpointDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={(row) => {
          list.refetch();
          onSelect(row);
        }}
      />
    </div>
  );
}

function CreateEndpointDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (row: Row) => void;
}) {
  const [method, setMethod] = useState('POST');
  const [path, setPath] = useState('');
  const [name, setName] = useState('');
  const [auth, setAuth] = useState('public');
  const [saving, setSaving] = useState(false);

  const reset = () => {
    setMethod('POST');
    setPath('');
    setName('');
    setAuth('public');
  };

  const submit = async () => {
    if (!path.trim()) return;
    setSaving(true);
    try {
      // A new endpoint starts with a request block and a 200 response, so the
      // builder opens on something runnable rather than an empty canvas.
      const res = await api.post('/endpoints', {
        method,
        path: path.trim(),
        name: name.trim(),
        auth,
        status: 'draft',
        nodes: [
          { id: 'n1', type: 'endpoint_handler', label: 'Request', config: {} },
          {
            id: 'n2',
            type: 'endpoint_response',
            label: 'Response',
            config: { status: 200, mode: 'json', body: '{"ok":true}' },
          },
        ],
        edges: [{ id: 'e1', source: 'n1', target: 'n2' }],
      });
      onOpenChange(false);
      reset();
      onCreated(res.data as Row);
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
      title="New endpoint"
      subtitle="Pick a method and path. You can wire the blocks next."
      submitLabel="Create"
      loading={saving}
      submitDisabled={!path.trim()}
      onSubmit={submit}
    >
      <SelectField
        label="Method"
        value={method}
        onChange={setMethod}
        options={HTTP_METHODS.map((m) => ({ value: m, label: m }))}
      />
      <TextField
        label="Path"
        value={path}
        onChange={(e) => setPath(e.target.value)}
        placeholder="/signup or /users/{id}"
        hint="Served at /v1/e/<path>. Use {name} for a path parameter."
        autoFocus
      />
      <TextField
        label="Name (optional)"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="Sign up"
      />
      <SelectField label="Who can call it" value={auth} onChange={setAuth} options={AUTH_OPTIONS} />
    </FormDialog>
  );
}
