import { useCallback, useEffect, useMemo, useState } from 'react';
import { AlertTriangle, ArrowLeft } from 'lucide-react';
import {
  addEdge,
  type Connection,
  type EdgeChange,
  type NodeChange,
  useEdgesState,
  useNodesState,
} from '@xyflow/react';
import { api, friendlyError } from '@/api/client';
import type { Row } from '@/components/data-table';
import { Button } from '@/components/ui/button';
import { SelectField, TextAreaField, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { cn } from '@/lib/utils';
import { MethodBadge } from './EndpointList';
import { EndpointCanvas } from './EndpointCanvas';
import { Inspector } from './Inspector';
import { TestPanel } from './TestPanel';
import { type EpEdge, type EpNode, fromFlow, newFlowNode, toFlow } from './graph';
import { ADDABLE_BLOCKS, AUTH_OPTIONS, type Block, type BlockType, HTTP_METHODS } from './blockDefs';

function nextId(nodes: EpNode[]): string {
  let max = 0;
  for (const n of nodes) {
    const m = /^n(\d+)$/.exec(n.id);
    if (m) max = Math.max(max, Number(m[1]));
  }
  return `n${max + 1}`;
}

export function EndpointBuilder({
  endpoint,
  onBack,
  onSaved,
}: {
  endpoint: Row;
  onBack: () => void;
  onSaved: () => void;
}) {
  const id = String(endpoint['$id'] ?? endpoint.id ?? '');
  const initial = useMemo(() => toFlow(endpoint.nodes, endpoint.edges), [endpoint.nodes, endpoint.edges]);

  const [method, setMethod] = useState(String(endpoint.method ?? 'GET'));
  const [path, setPath] = useState(String(endpoint.path ?? '/'));
  const [name, setName] = useState(String(endpoint.name ?? ''));
  const [auth, setAuth] = useState(String(endpoint.auth ?? 'public'));
  const [status, setStatus] = useState(String(endpoint.status ?? 'draft'));
  const [schemaText, setSchemaText] = useState(() => {
    const s = endpoint.inputSchema as Record<string, unknown> | null | undefined;
    return s && Object.keys(s).length ? JSON.stringify(s, null, 2) : '';
  });

  const [nodes, setNodes, onNodesChangeRaw] = useNodesState<EpNode>(initial.nodes);
  const [edges, setEdges, onEdgesChangeRaw] = useEdgesState<EpEdge>(initial.edges);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [tab, setTab] = useState<'inspect' | 'test' | 'runs'>('inspect');
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);

  // Publish-time safety hints: patterns that expose data more widely than an
  // author may intend. Non-blocking; the author decides.
  const warnings = useMemo(() => {
    const w: string[] = [];
    const dataNodes = nodes.filter((n) => n.data.blockType === 'endpoint_data');
    if (auth === 'public' && dataNodes.some((n) => n.data.config.applyRules === false)) {
      w.push(
        'This endpoint is public and a Data block has Apply rules off, so anyone who calls it gets full table access. Confirm that is intended.',
      );
    }
    const templatesFromRequest = (v: unknown) => /\{\{[^}]*\.?request\./.test(String(v ?? ''));
    if (dataNodes.some((n) => templatesFromRequest(n.data.config.databaseId) || templatesFromRequest(n.data.config.tableId))) {
      w.push(
        'A Data block chooses its database or table from the request, letting callers target arbitrary tables. Prefer a fixed table or a value you validate.',
      );
    }
    return w;
  }, [nodes, auth]);

  const markDirty = () => setDirty(true);

  const onNodesChange = useCallback(
    (changes: NodeChange<EpNode>[]) => {
      onNodesChangeRaw(changes);
      for (const c of changes) {
        if (c.type === 'remove' && c.id === selectedId) setSelectedId(null);
        if (c.type === 'position' || c.type === 'remove' || c.type === 'add') markDirty();
      }
    },
    [onNodesChangeRaw, selectedId],
  );

  const onEdgesChange = useCallback(
    (changes: EdgeChange<EpEdge>[]) => {
      onEdgesChangeRaw(changes);
      if (changes.some((c) => c.type !== 'select')) markDirty();
    },
    [onEdgesChangeRaw],
  );

  const onConnect = useCallback(
    (c: Connection) => {
      // One source handle, one edge: replacing a wire re-points a branch rather
      // than stacking two. For a condition's true/false, this keeps each output
      // to a single target.
      setEdges((eds) => {
        const cleaned = eds.filter(
          (e) => !(e.source === c.source && e.sourceHandle === c.sourceHandle),
        );
        return addEdge({ ...c, type: 'deletable' }, cleaned);
      });
      markDirty();
    },
    [setEdges],
  );

  const addBlock = useCallback(
    (type: BlockType) => {
      setNodes((nds) => {
        const node = newFlowNode(type, nextId(nds), nds);
        setSelectedId(node.id);
        return [...nds, node];
      });
      setTab('inspect');
      markDirty();
    },
    [setNodes],
  );

  const patchSelected = useCallback(
    (patch: Record<string, unknown>) => {
      if (!selectedId) return;
      setNodes((nds) =>
        nds.map((n) =>
          n.id === selectedId ? { ...n, data: { ...n.data, config: { ...n.data.config, ...patch } } } : n,
        ),
      );
      markDirty();
    },
    [selectedId, setNodes],
  );

  const onBeforeDelete = useCallback(
    async ({ nodes: toDelete, edges: toDeleteEdges }: { nodes: EpNode[]; edges: EpEdge[] }) => {
      // The request handler is the fixed entry point; never delete it. But a
      // deletion of only edges (or only the handler) must still remove the
      // edges — returning false here would block unlinking entirely.
      const deletable = toDelete.filter((n) => n.data.blockType !== 'endpoint_handler');
      if (toDelete.length > 0 && deletable.length === 0 && toDeleteEdges.length === 0) {
        return false;
      }
      return { nodes: deletable, edges: toDeleteEdges };
    },
    [],
  );

  const save = useCallback(async (): Promise<boolean> => {
    if (!path.trim()) {
      toast.error('A path is required');
      return false;
    }
    let inputSchema: unknown = null;
    if (schemaText.trim()) {
      try {
        inputSchema = JSON.parse(schemaText);
      } catch {
        toast.error('Input validation schema is not valid JSON');
        return false;
      }
    }
    setSaving(true);
    try {
      const graph = fromFlow(nodes, edges);
      await api.put(`/endpoints/${id}`, {
        method,
        path: path.trim(),
        name: name.trim(),
        auth,
        status,
        inputSchema,
        nodes: graph.nodes,
        edges: graph.edges,
      });
      setDirty(false);
      onSaved();
      return true;
    } catch (e) {
      toast.error(friendlyError(e));
      return false;
    } finally {
      setSaving(false);
    }
  }, [nodes, edges, id, method, path, name, auth, status, schemaText, onSaved]);

  const selectedBlock: Block | null = useMemo(() => {
    const n = nodes.find((x) => x.id === selectedId);
    if (!n) return null;
    return { id: n.id, type: n.data.blockType, label: n.data.label, config: n.data.config };
  }, [nodes, selectedId]);

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-3 border-b border-border px-6 py-3 md:px-8">
        <button
          type="button"
          onClick={onBack}
          className="flex items-center gap-1.5 rounded-[var(--radius)] px-2 py-1.5 text-[length:var(--text-body)] text-text-secondary hover:bg-fill hover:text-text-primary"
        >
          <ArrowLeft size={15} />
          Endpoints
        </button>
        <div className="flex min-w-0 items-center gap-2">
          <MethodBadge method={method} />
          <span className="truncate font-mono text-[length:var(--text-body)] text-text-primary">
            {path}
          </span>
        </div>
        <div className="ml-auto flex items-center gap-3">
          {dirty && <span className="text-[length:var(--text-caption)] text-text-muted">Unsaved</span>}
          <div className="w-36">
            <SelectField
              value={status}
              onChange={(v) => {
                setStatus(v);
                markDirty();
              }}
              options={[
                { value: 'draft', label: 'Draft' },
                { value: 'published', label: 'Published' },
              ]}
            />
          </div>
          <Button onClick={() => void save()} loading={saving} disabled={!dirty && !saving}>
            Save
          </Button>
        </div>
      </div>

      {/* Non-blocking safety hints for risky patterns. */}
      {warnings.length > 0 && (
        <div className="flex flex-col gap-1 border-b border-border bg-[color-mix(in_srgb,var(--status-warning)_10%,transparent)] px-6 py-2.5 md:px-8">
          {warnings.map((wn, i) => (
            <div key={i} className="flex items-start gap-2 text-[length:var(--text-caption)] text-text-secondary">
              <AlertTriangle size={13} className="mt-0.5 shrink-0 text-[var(--status-warning)]" />
              <span>{wn}</span>
            </div>
          ))}
        </div>
      )}

      {/* Body: library | canvas | inspector/test */}
      <div className="grid min-h-0 flex-1 grid-cols-1 md:grid-cols-[248px_1fr_360px]">
        {/* Left: node library + route settings */}
        <div className="flex flex-col gap-5 overflow-y-auto border-r border-border p-4">
          <section>
            <h2 className="mb-2 text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-muted">
              Add a block
            </h2>
            <NodesLibrary onAdd={addBlock} />
          </section>
          <section className="border-t border-border pt-4">
            <h2 className="mb-3 text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-muted">
              Route
            </h2>
            <div className="flex flex-col gap-3">
              <div className="grid grid-cols-[96px_1fr] gap-2">
                <SelectField
                  value={method}
                  onChange={(v) => {
                    setMethod(v);
                    markDirty();
                  }}
                  options={HTTP_METHODS.map((m) => ({ value: m, label: m }))}
                />
                <TextField
                  value={path}
                  onChange={(e) => {
                    setPath(e.target.value);
                    markDirty();
                  }}
                  placeholder="/signup"
                />
              </div>
              <TextField
                label="Name"
                value={name}
                onChange={(e) => {
                  setName(e.target.value);
                  markDirty();
                }}
                placeholder="Sign up"
              />
              <SelectField
                label="Who can call it"
                value={auth}
                onChange={(v) => {
                  setAuth(v);
                  markDirty();
                }}
                options={AUTH_OPTIONS}
              />
              <TextAreaField
                label="Input validation (optional)"
                value={schemaText}
                onChange={(e) => {
                  setSchemaText(e.target.value);
                  markDirty();
                }}
                rows={4}
                className="font-mono"
                placeholder={'{\n  "required": ["email"],\n  "properties": { "email": { "type": "string" } }\n}'}
                hint="A rejected body returns 400 before any block runs."
              />
            </div>
          </section>
        </div>

        {/* Center: the flowchart canvas */}
        <div className="min-h-[420px] bg-background">
          <EndpointCanvas
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={(nid) => {
              setSelectedId(nid);
              setTab('inspect');
            }}
            onPaneClick={() => setSelectedId(null)}
            onBeforeDelete={onBeforeDelete}
          />
        </div>

        {/* Right: inspect the selected block, or test the endpoint */}
        <div className="flex min-h-0 flex-col overflow-y-auto border-l border-border">
          <div className="flex gap-1 border-b border-border p-2">
            <TabButton active={tab === 'inspect'} onClick={() => setTab('inspect')}>
              Inspect
            </TabButton>
            <TabButton active={tab === 'test'} onClick={() => setTab('test')}>
              Test
            </TabButton>
            <TabButton active={tab === 'runs'} onClick={() => setTab('runs')}>
              Runs
            </TabButton>
          </div>
          {tab === 'inspect' && <Inspector block={selectedBlock} onChange={patchSelected} />}
          {tab === 'test' && (
            <TestPanel endpointId={id} method={method} path={path} dirty={dirty} onSaveFirst={save} />
          )}
          {tab === 'runs' && <RunsPanel endpointId={id} active={tab === 'runs'} />}
        </div>
      </div>
    </div>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'flex-1 rounded-[var(--radius)] px-3 py-1.5 text-[length:var(--text-body)] font-medium',
        active ? 'bg-fill text-text-primary' : 'text-text-secondary hover:bg-fill hover:text-text-primary',
      )}
    >
      {children}
    </button>
  );
}

interface RunRow {
  $id: string;
  status: string;
  method: string;
  path: string;
  statusCode: number;
  durationMs: number;
  $createdAt: string;
  request: Record<string, unknown>;
  response: unknown;
  logs: { nodeType: string; label: string; status: string; durationMs: number; error?: string }[];
}

/** Execution history for real (and test) calls. Fetches when the tab opens. */
function RunsPanel({ endpointId, active }: { endpointId: string; active: boolean }) {
  const [runs, setRuns] = useState<RunRow[] | null>(null);
  const [openId, setOpenId] = useState<string | null>(null);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setError('');
    try {
      const res = await api.get(`/endpoints/${endpointId}/executions?limit=50`);
      setRuns((res.data.executions ?? []) as RunRow[]);
    } catch (e) {
      setError(friendlyError(e));
    }
  }, [endpointId]);

  useEffect(() => {
    if (active) void load();
  }, [active, load]);

  return (
    <div className="flex flex-col gap-3 p-5">
      <div className="flex items-center justify-between">
        <span className="text-[length:var(--text-caption)] font-semibold uppercase tracking-wide text-text-muted">
          Recent runs
        </span>
        <button
          type="button"
          onClick={() => void load()}
          className="text-[length:var(--text-caption)] text-text-secondary hover:text-text-primary"
        >
          Refresh
        </button>
      </div>

      {error && <p className="text-[length:var(--text-caption)] text-[var(--color-danger)]">{error}</p>}
      {runs && runs.length === 0 && (
        <p className="text-[length:var(--text-body)] text-text-muted">
          No calls yet. Publish the endpoint and call it, or run a test.
        </p>
      )}

      <div className="flex flex-col gap-1.5">
        {runs?.map((r) => (
          <div key={r.$id} className="rounded-[var(--radius)] border border-border">
            <button
              type="button"
              onClick={() => setOpenId(openId === r.$id ? null : r.$id)}
              className="flex w-full items-center gap-2 px-3 py-2 text-left"
            >
              <span
                className={cn(
                  'rounded-[var(--radius-sm)] px-1.5 font-mono text-[length:var(--text-2xs)] font-semibold',
                  r.statusCode >= 400
                    ? 'bg-[color-mix(in_srgb,var(--color-danger)_14%,transparent)] text-[var(--color-danger)]'
                    : 'bg-[color-mix(in_srgb,var(--status-success)_14%,transparent)] text-[var(--status-success)]',
                )}
              >
                {r.statusCode}
              </span>
              <span className="font-mono text-[length:var(--text-caption)] text-text-primary">{r.method}</span>
              <span className="truncate font-mono text-[length:var(--text-caption)] text-text-muted">{r.path}</span>
              <span className="ml-auto text-[length:var(--text-2xs)] text-text-muted">{r.durationMs}ms</span>
              <span className="text-[length:var(--text-2xs)] text-text-subtle">{relTime(r.$createdAt)}</span>
            </button>
            {openId === r.$id && (
              <div className="flex flex-col gap-3 border-t border-border px-3 py-3">
                <RunSection title="Request">
                  <pre className="max-h-40 overflow-auto rounded-[var(--radius)] border border-border bg-background p-2 font-mono text-[length:var(--text-2xs)] text-text-primary">
                    {JSON.stringify(r.request, null, 2)}
                  </pre>
                </RunSection>
                <RunSection title="Response">
                  <pre className="max-h-40 overflow-auto rounded-[var(--radius)] border border-border bg-background p-2 font-mono text-[length:var(--text-2xs)] text-text-primary">
                    {JSON.stringify(r.response, null, 2)}
                  </pre>
                </RunSection>
                <RunSection title="Trace">
                  <div className="flex flex-col gap-1">
                    {r.logs?.map((l, i) => (
                      <div key={i} className="flex items-center gap-2 text-[length:var(--text-2xs)]">
                        <span
                          className={cn(
                            'h-1.5 w-1.5 shrink-0 rounded-full',
                            l.status === 'failed'
                              ? 'bg-[var(--color-danger)]'
                              : l.status === 'skipped'
                                ? 'bg-fill-active'
                                : 'bg-[var(--status-success)]',
                          )}
                        />
                        <span className="text-text-primary">{l.label || l.nodeType}</span>
                        <span className="ml-auto text-text-muted">{l.durationMs}ms</span>
                      </div>
                    ))}
                  </div>
                </RunSection>
              </div>
            )}
          </div>
        ))}
      </div>
      <p className="text-[length:var(--text-2xs)] text-text-subtle">
        Secret-named fields (password, token, card…) are redacted in stored history.
      </p>
    </div>
  );
}

function RunSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1 text-[length:var(--text-2xs)] font-semibold uppercase tracking-wide text-text-muted">
        {title}
      </div>
      {children}
    </div>
  );
}

function relTime(v: string): string {
  const dt = new Date(v);
  if (Number.isNaN(dt.getTime())) return '';
  const m = Math.floor((Date.now() - dt.getTime()) / 60000);
  if (m < 1) return 'just now';
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

function NodesLibrary({ onAdd }: { onAdd: (type: BlockType) => void }) {
  // Group the addable blocks by their category, like the reference designer.
  const groups = useMemo(() => {
    const byCat = new Map<string, typeof ADDABLE_BLOCKS>();
    for (const def of ADDABLE_BLOCKS) {
      const arr = byCat.get(def.category) ?? [];
      arr.push(def);
      byCat.set(def.category, arr);
    }
    return Array.from(byCat.entries());
  }, []);

  return (
    <div className="flex flex-col gap-3">
      {groups.map(([category, defs]) => (
        <div key={category} className="flex flex-col gap-1">
          <span className="text-[length:var(--text-2xs)] font-medium uppercase tracking-wide text-text-subtle">
            {category}
          </span>
          {defs.map((def) => {
            const Icon = def.icon;
            return (
              <button
                key={def.type}
                type="button"
                onClick={() => onAdd(def.type)}
                className="flex items-start gap-2.5 rounded-[var(--radius)] border border-border bg-surface px-2.5 py-2 text-left transition-colors hover:border-field-border hover:bg-fill"
              >
                <span className="mt-0.5 shrink-0" style={{ color: def.accent }}>
                  <Icon size={15} />
                </span>
                <span className="min-w-0">
                  <span className="block text-[length:var(--text-body)] font-medium text-text-primary">
                    {def.label}
                  </span>
                  <span className="block text-[length:var(--text-caption)] leading-snug text-text-muted">
                    {def.blurb}
                  </span>
                </span>
              </button>
            );
          })}
        </div>
      ))}
    </div>
  );
}
