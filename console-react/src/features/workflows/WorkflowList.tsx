import { useState } from 'react';
import {
  AlertTriangle,
  BarChart3,
  Boxes,
  Clock,
  Database,
  GitBranch,
  LayoutTemplate,
  type LucideIcon,
  Mail,
  Play,
  Webhook,
} from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { StatusChip } from '@/components/status-chip';
import { FormDialog, TextField } from '@/components/form-dialog';
import { Dialog, DialogContent } from '@/components/ui/dialog';
import { toast } from '@/components/toast';
import { ACCENT } from './nodeDefs';

const COLUMNS: DataTableColumn[] = [
  { key: 'name', label: 'Name', flex: 4, sortable: true },
  { key: 'triggerType', label: 'Trigger', flex: 2 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'nodes', label: 'Nodes', flex: 1 },
  { key: 'updatedAt', label: 'Updated', flex: 2 },
];

function relativeTime(v: unknown): string {
  if (!v) return '';
  const dt = new Date(String(v));
  if (Number.isNaN(dt.getTime())) return String(v);
  const diff = Date.now() - dt.getTime();
  const m = Math.floor(diff / 60000);
  if (m < 1) return 'just now';
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d}d ago`;
  return dt.toLocaleDateString();
}

function triggerIcon(t: string): LucideIcon {
  return t === 'webhook' ? Webhook : t === 'cron' ? Clock : Play;
}

export function WorkflowList({
  projectId,
  onSelect,
}: {
  projectId: string | undefined;
  onSelect: (wf: Row) => void;
}) {
  const list = useResourceList<Row>({
    endpoint: '/workflows',
    itemsKey: 'workflows',
    scope: [projectId],
  });
  const [creating, setCreating] = useState(false);
  const [templatesOpen, setTemplatesOpen] = useState(false);

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Workflows</h1>
          <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
            Automate tasks with visual pipelines
          </p>
        </div>
        <button
          type="button"
          onClick={() => setTemplatesOpen(true)}
          className="flex items-center gap-2 rounded-[var(--radius)] px-3.5 py-2.5 text-[length:var(--text-body)] text-text-secondary hover:bg-fill hover:text-text-primary"
        >
          <LayoutTemplate size={14} />
          Templates
        </button>
      </div>

      <DataTable
        columns={COLUMNS}
        rows={list.rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'name':
              return String(row['name'] ?? 'Unnamed');
            case 'triggerType':
              return String(row['triggerType'] ?? 'manual');
            case 'status':
              return String(row['status'] ?? 'draft');
            case 'nodes':
              return String((row['nodes'] as unknown[] | undefined)?.length ?? 0);
            case 'updatedAt':
              return relativeTime(row['updatedAt']);
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'status') return <StatusChip label={String(row['status'] ?? 'draft')} />;
          if (key === 'triggerType') {
            const t = String(row['triggerType'] ?? 'manual');
            const Icon = triggerIcon(t);
            return (
              <span className="inline-flex items-center gap-1.5 text-text-secondary">
                <Icon size={12} className="text-text-muted" />
                {t}
              </span>
            );
          }
          return undefined;
        }}
        rowIcon={() => GitBranch}
        rowIconColor={() => ACCENT}
        onRowClick={onSelect}
        onDeleteRow={async (row) => {
          await api.delete(`/workflows/${String(row['$id'])}`);
          list.refetch();
        }}
        gridCard={(row) => <WorkflowGridCard wf={row} />}
        persistKey="workflows"
        createLabel="New Workflow"
        onCreate={() => setCreating(true)}
        searchHint="Search workflows…"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="workflows"
        emptyIcon={GitBranch}
        emptyTitle="No workflows yet"
        emptySubtitle="Create your first workflow to get started"
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      <CreateWorkflowDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={(wf) => {
          list.refetch();
          onSelect(wf);
        }}
      />
      <TemplatesDialog
        open={templatesOpen}
        onOpenChange={setTemplatesOpen}
        onCreated={(wf) => {
          list.refetch();
          onSelect(wf);
        }}
      />
    </div>
  );
}

function WorkflowGridCard({ wf }: { wf: Row }) {
  const name = String(wf['name'] ?? 'Unnamed');
  const status = String(wf['status'] ?? 'draft');
  const trigger = String(wf['triggerType'] ?? 'manual');
  const nc = (wf['nodes'] as unknown[] | undefined)?.length ?? 0;
  const TrigIcon = triggerIcon(trigger);
  return (
    <div className="flex h-full flex-col gap-3.5 rounded-[var(--radius-12)] border border-border bg-surface p-5 transition-colors hover:border-field-border hover:bg-fill-hover">
      <div className="flex items-center gap-3">
        <div
          className="flex h-9 w-9 items-center justify-center rounded-[var(--radius)]"
          style={{ background: `color-mix(in srgb, ${ACCENT} 12%, transparent)`, color: ACCENT }}
        >
          <GitBranch size={18} />
        </div>
        <span className="min-w-0 flex-1 truncate text-[length:var(--text-control)] font-semibold text-text-primary">
          {name}
        </span>
        <StatusChip label={status} />
      </div>
      <div className="flex items-center gap-2">
        <Chip icon={<TrigIcon size={12} />} label={trigger} />
        <Chip icon={<Boxes size={12} />} label={`${nc} nodes`} />
      </div>
    </div>
  );
}

function Chip({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-[var(--radius-6)] bg-fill px-2 py-1 text-[length:var(--text-caption)] text-text-subtle">
      {icon}
      {label}
    </span>
  );
}

function CreateWorkflowDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (wf: Row) => void;
}) {
  const [name, setName] = useState('');
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      const res = await api.post('/workflows', {
        name: name.trim(),
        description: '',
        triggerType: 'manual',
        nodes: [],
        edges: [],
      });
      onOpenChange(false);
      setName('');
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
      title="New Workflow"
      submitLabel="Create"
      loading={saving}
      submitDisabled={!name.trim()}
      onSubmit={submit}
    >
      <TextField
        label="Workflow name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="My workflow"
        autoFocus
      />
    </FormDialog>
  );
}

interface Template {
  name: string;
  description: string;
  icon: LucideIcon;
  triggerType: string;
  nodes: Record<string, unknown>[];
  edges: Record<string, unknown>[];
}

const TEMPLATES: Template[] = [
  {
    name: 'Welcome Email',
    description: 'Send a welcome email when a webhook fires',
    icon: Mail,
    triggerType: 'webhook',
    nodes: [
      { id: 'n1', type: 'trigger', label: 'Trigger', config: {}, position: { x: 100, y: 200 }, disabled: false },
      {
        id: 'n2',
        type: 'applad_messaging',
        label: 'Send Email',
        config: {
          action: 'send_email',
          to: '{{.trigger.body.email}}',
          subject: 'Welcome!',
          body: 'Hi {{.trigger.body.name}}, welcome!',
        },
        position: { x: 380, y: 200 },
        disabled: false,
      },
    ],
    edges: [{ id: 'e1', source: 'n1', target: 'n2' }],
  },
  {
    name: 'Daily Digest',
    description: 'Cron-triggered digest with AI summarization',
    icon: BarChart3,
    triggerType: 'cron',
    nodes: [
      { id: 'n1', type: 'trigger', label: 'Trigger', config: {}, position: { x: 100, y: 200 }, disabled: false },
      {
        id: 'n2',
        type: 'http_request',
        label: 'Fetch Data',
        config: { url: 'https://api.example.com/data', method: 'GET' },
        position: { x: 380, y: 200 },
        disabled: false,
      },
      {
        id: 'n3',
        type: 'ai_summarize',
        label: 'AI Summarize',
        config: { model: 'claude-sonnet-4-20250514', text: '{{.fetch_data.output.body}}' },
        position: { x: 660, y: 200 },
        disabled: false,
      },
      {
        id: 'n4',
        type: 'applad_messaging',
        label: 'Send Digest',
        config: {
          action: 'send_email',
          to: 'team@example.com',
          subject: 'Daily Digest',
          body: '{{.ai_summarize.output.summary}}',
        },
        position: { x: 940, y: 200 },
        disabled: false,
      },
    ],
    edges: [
      { id: 'e1', source: 'n1', target: 'n2' },
      { id: 'e2', source: 'n2', target: 'n3' },
      { id: 'e3', source: 'n3', target: 'n4' },
    ],
  },
  {
    name: 'Webhook to DB',
    description: 'Store incoming webhook data in a database',
    icon: Database,
    triggerType: 'webhook',
    nodes: [
      { id: 'n1', type: 'trigger', label: 'Trigger', config: {}, position: { x: 100, y: 200 }, disabled: false },
      {
        id: 'n2',
        type: 'edit_fields',
        label: 'Map Fields',
        config: { fields: '{"email": "{{.trigger.body.email}}", "name": "{{.trigger.body.name}}"}' },
        position: { x: 380, y: 200 },
        disabled: false,
      },
      {
        id: 'n3',
        type: 'applad_database',
        label: 'Save to DB',
        config: { action: 'create_document', data: '{{.map_fields.output}}' },
        position: { x: 660, y: 200 },
        disabled: false,
      },
    ],
    edges: [
      { id: 'e1', source: 'n1', target: 'n2' },
      { id: 'e2', source: 'n2', target: 'n3' },
    ],
  },
  {
    name: 'Error Alert',
    description: 'Catch errors and send a Slack notification',
    icon: AlertTriangle,
    triggerType: 'webhook',
    nodes: [
      { id: 'n1', type: 'trigger', label: 'Trigger', config: {}, position: { x: 100, y: 200 }, disabled: false },
      {
        id: 'n2',
        type: 'try_catch',
        label: 'Try / Catch',
        config: { tryNodes: '', catchTarget: 'n3' },
        position: { x: 380, y: 200 },
        disabled: false,
      },
      {
        id: 'n3',
        type: 'slack',
        label: 'Slack Alert',
        config: { message: 'Error: {{.trigger.body.error}}' },
        position: { x: 660, y: 300 },
        disabled: false,
      },
    ],
    edges: [
      { id: 'e1', source: 'n1', target: 'n2' },
      { id: 'e2', source: 'n2', target: 'n3' },
    ],
  },
];

function TemplatesDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (wf: Row) => void;
}) {
  const [busy, setBusy] = useState(false);

  const create = async (t: Template) => {
    setBusy(true);
    try {
      const res = await api.post('/workflows', {
        name: t.name,
        description: t.description,
        triggerType: t.triggerType,
        nodes: t.nodes,
        edges: t.edges,
      });
      onOpenChange(false);
      onCreated(res.data as Row);
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent width={600} showClose={false}>
        <div className="flex items-center gap-2.5 border-b border-border px-5 py-4">
          <LayoutTemplate size={16} style={{ color: ACCENT }} />
          <div className="flex-1">
            <div className="text-[length:var(--text-title)] font-semibold text-text-primary">
              Workflow Templates
            </div>
            <div className="text-[length:var(--text-body)] text-text-secondary">
              Start with a pre-built workflow
            </div>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3 p-5">
          {TEMPLATES.map((t) => {
            const Icon = t.icon;
            return (
              <button
                key={t.name}
                type="button"
                disabled={busy}
                onClick={() => create(t)}
                className="flex flex-col gap-2.5 rounded-[var(--radius-10)] border border-border bg-fill p-4 text-left transition-colors hover:bg-fill-hover disabled:opacity-60"
              >
                <div className="flex items-center">
                  <div
                    className="flex h-8 w-8 items-center justify-center rounded-[var(--radius-7)]"
                    style={{ background: `color-mix(in srgb, ${ACCENT} 12%, transparent)`, color: ACCENT }}
                  >
                    <Icon size={15} />
                  </div>
                  <span className="ml-auto rounded-full border border-border bg-surface px-2 py-0.5 text-[length:var(--text-2xs)] text-text-subtle">
                    {t.nodes.length} nodes
                  </span>
                </div>
                <div className="text-[length:var(--text-body)] font-semibold text-text-primary">
                  {t.name}
                </div>
                <div className="text-[length:var(--text-caption)] text-text-secondary">
                  {t.description}
                </div>
              </button>
            );
          })}
        </div>
      </DialogContent>
    </Dialog>
  );
}
