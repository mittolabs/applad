import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  Panel,
  ReactFlow,
  ReactFlowProvider,
  addEdge,
  useEdgesState,
  useNodesState,
  useReactFlow,
  useViewport,
  type Connection,
  type Edge,
  type FinalConnectionState,
  type Node,
  type NodeTypes,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import {
  ArrowLeft,
  ChevronDown,
  History,
  Play,
  Plus,
  Redo2,
  Map as MapIcon,
  Save,
  StickyNote,
  Terminal,
  Undo2,
} from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { defFor, GREEN, ORANGE } from './nodeDefs';
import { WorkflowNode, type WfNodeData } from './WorkflowNode';
import { StickyNode } from './StickyNode';
import { NodePalette } from './NodePalette';
import { NodeConfigPanel, type ExecEntry } from './NodeConfigPanel';
import { ExecutionsPanel, type Execution, type LogEntry } from './ExecutionsPanel';
import { LiveLogsPanel } from './LiveLogsPanel';
import { CanvasContextMenu, type ContextMenuState } from './CanvasContextMenu';

type WfNode = Node<WfNodeData>;

const nodeTypes: NodeTypes = { wf: WorkflowNode, sticky: StickyNode };

interface Snapshot {
  nodes: WfNode[];
  edges: Edge[];
}

// ── Model <-> React Flow conversion ──

function toRfNodes(raw: Record<string, unknown>[]): WfNode[] {
  return raw.map((n, i) => {
    let type = String(n['type'] ?? '');
    let config = { ...((n['config'] as Record<string, unknown>) ?? {}) };
    // Reverse the save-time trigger transform (set_variable{_trigger} -> trigger).
    if (type === 'set_variable' && config['key'] === '_trigger') {
      type = String(config['value'] ?? 'trigger');
      config = {};
    }
    const pos = (n['position'] as { x?: number; y?: number } | undefined) ?? {};
    return {
      id: String(n['id'] ?? `node_${i}`),
      type: 'wf',
      position: {
        x: typeof pos.x === 'number' ? pos.x : 300 + i * 280,
        y: typeof pos.y === 'number' ? pos.y : 300,
      },
      data: {
        nodeType: type,
        label: String(n['label'] ?? defFor(type).label),
        config,
        disabled: n['disabled'] === true,
        onError: (n['onError'] as string | undefined) ?? undefined,
      },
    } satisfies WfNode;
  });
}

function toRfEdges(raw: Record<string, unknown>[]): Edge[] {
  return raw.map((e, i) => ({
    id: String(e['id'] ?? `e_${i}`),
    source: String(e['source'] ?? ''),
    target: String(e['target'] ?? ''),
  }));
}

function toSaveNodes(nodes: WfNode[]): Record<string, unknown>[] {
  // Sticky notes are editor-local annotations — never persisted (matches Flutter).
  return nodes.filter((n) => n.type !== 'sticky').map((n) => {
    const type = n.data.nodeType;
    const base: Record<string, unknown> = {
      id: n.id,
      type,
      label: n.data.label,
      config: n.data.config,
      position: { x: n.position.x, y: n.position.y },
    };
    if (n.data.onError) base['onError'] = n.data.onError;
    // Triggers are persisted as a set_variable marker for backend compatibility.
    if (type === 'trigger' || type.startsWith('trigger_')) {
      base['type'] = 'set_variable';
      base['config'] = { key: '_trigger', value: type };
    }
    return base;
  });
}

function toSaveEdges(edges: Edge[]): Record<string, unknown>[] {
  return edges.map((e) => ({ id: e.id, source: e.source, target: e.target }));
}

export function WorkflowBuilder({
  workflow,
  onBack,
  onSaved,
}: {
  workflow: Record<string, unknown>;
  onBack: () => void;
  onSaved: () => void;
}) {
  return (
    <ReactFlowProvider>
      <BuilderInner workflow={workflow} onBack={onBack} onSaved={onSaved} />
    </ReactFlowProvider>
  );
}

/* Bottom-right canvas controls — ports workflow_builder.dart `_zoomControls` +
 * `_minimap` toggle: a live zoom % (click to reset to 100%) and a minimap toggle.
 * Reads live zoom via useViewport, so it must render inside the ReactFlow context. */
function ZoomPanel({
  minimapOn,
  onToggleMinimap,
  onResetZoom,
}: {
  minimapOn: boolean;
  onToggleMinimap: () => void;
  onResetZoom: () => void;
}) {
  const { zoom } = useViewport();
  return (
    <div className="flex items-center gap-1 rounded-[var(--radius-8)] border border-border bg-surface p-1 shadow-sm">
      <button
        type="button"
        onClick={onResetZoom}
        title="Reset zoom to 100%"
        className="min-w-[52px] rounded-[var(--radius-6)] px-2 py-1 text-center text-[length:var(--text-caption)] font-medium text-text-secondary tabular-nums hover:bg-fill hover:text-text-primary"
      >
        {Math.round(zoom * 100)}%
      </button>
      <button
        type="button"
        onClick={onToggleMinimap}
        title={minimapOn ? 'Hide minimap' : 'Show minimap'}
        className={`rounded-[var(--radius-6)] p-1.5 hover:bg-fill ${
          minimapOn ? 'text-[var(--color-accent)]' : 'text-text-subtle hover:text-text-primary'
        }`}
      >
        <MapIcon size={15} />
      </button>
    </div>
  );
}

function BuilderInner({
  workflow,
  onBack,
  onSaved,
}: {
  workflow: Record<string, unknown>;
  onBack: () => void;
  onSaved: () => void;
}) {
  const wfId = String(workflow['$id']);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const { screenToFlowPosition, fitView, zoomTo } = useReactFlow();
  const [showMinimap, setShowMinimap] = useState(true);

  const [nodes, setNodes, onNodesChange] = useNodesState<WfNode>(
    toRfNodes((workflow['nodes'] as Record<string, unknown>[]) ?? []),
  );
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>(
    toRfEdges((workflow['edges'] as Record<string, unknown>[]) ?? []),
  );

  const [name, setName] = useState(String(workflow['name'] ?? 'Unnamed'));
  const [status, setStatus] = useState(String(workflow['status'] ?? 'draft'));
  const [triggerType, setTriggerType] = useState(String(workflow['triggerType'] ?? 'manual'));
  const [triggerConfig, setTriggerConfig] = useState<Record<string, unknown>>(
    (workflow['triggerConfig'] as Record<string, unknown>) ?? {},
  );
  const [webhookSecret, setWebhookSecret] = useState(String(workflow['webhookSecret'] ?? ''));

  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [configNodeId, setConfigNodeId] = useState<string | null>(null);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [pendingConnect, setPendingConnect] = useState<string | null>(null);
  const [execsOpen, setExecsOpen] = useState(false);
  const [testing, setTesting] = useState(false);
  const [lastExecData, setLastExecData] = useState<Record<string, ExecEntry>>({});
  const [pinnedData, setPinnedData] = useState<Record<string, unknown>>({});
  const [logsOpen, setLogsOpen] = useState(false);
  const [execLogs, setExecLogs] = useState<LogEntry[]>([]);
  const [menu, setMenu] = useState<ContextMenuState | null>(null);

  // ── Undo / redo ──
  const undoStack = useRef<Snapshot[]>([]);
  const redoStack = useRef<Snapshot[]>([]);
  const [, forceRender] = useState(0);
  const pushUndo = useCallback(() => {
    undoStack.current.push({ nodes, edges });
    redoStack.current = [];
    forceRender((n) => n + 1);
  }, [nodes, edges]);

  const applyUndo = useCallback(() => {
    const snap = undoStack.current.pop();
    if (!snap) return;
    redoStack.current.push({ nodes, edges });
    setNodes(snap.nodes);
    setEdges(snap.edges);
    setDirty(true);
    forceRender((n) => n + 1);
  }, [nodes, edges, setNodes, setEdges]);

  const applyRedo = useCallback(() => {
    const snap = redoStack.current.pop();
    if (!snap) return;
    undoStack.current.push({ nodes, edges });
    setNodes(snap.nodes);
    setEdges(snap.edges);
    setDirty(true);
    forceRender((n) => n + 1);
  }, [nodes, edges, setNodes, setEdges]);

  const setNodeData = useCallback(
    (id: string, patch: Partial<WfNodeData>) => {
      setNodes((ns) =>
        ns.map((n) => (n.id === id ? { ...n, data: { ...n.data, ...patch } } : n)),
      );
      setDirty(true);
    },
    [setNodes],
  );

  // ── Add node ──
  const addNode = useCallback(
    (type: string, position: { x: number; y: number }, connectFrom?: string | null) => {
      pushUndo();
      const id = `node_${Date.now()}`;
      const d = defFor(type);
      const newNode: WfNode = {
        id,
        type: 'wf',
        position: { x: Math.round(position.x / 20) * 20, y: Math.round(position.y / 20) * 20 },
        data: { nodeType: type, label: d.label, config: {}, disabled: false },
      };
      setNodes((ns) => [...ns, newNode]);
      if (connectFrom) {
        setEdges((es) => [...es, { id: `e_${Date.now()}`, source: connectFrom, target: id }]);
      }
      setDirty(true);
    },
    [pushUndo, setNodes, setEdges],
  );

  const openPaletteCenter = useCallback(() => {
    setPendingConnect(null);
    setPaletteOpen(true);
  }, []);

  const handlePaletteAdd = useCallback(
    (type: string) => {
      const rect = wrapperRef.current?.getBoundingClientRect();
      const center = rect
        ? screenToFlowPosition({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
        : { x: 400, y: 300 };
      addNode(type, center, pendingConnect);
      setPaletteOpen(false);
      setPendingConnect(null);
    },
    [addNode, pendingConnect, screenToFlowPosition],
  );

  // ── Connections ──
  const onConnect = useCallback(
    (params: Connection) => {
      if (params.source === params.target) return;
      pushUndo();
      setEdges((es) =>
        addEdge({ ...params, id: `e_${Date.now()}` }, es).filter(
          (e, i, arr) =>
            arr.findIndex((x) => x.source === e.source && x.target === e.target) === i,
        ),
      );
      setDirty(true);
    },
    [pushUndo, setEdges],
  );

  const connectStart = useRef<string | null>(null);
  const onConnectStart = useCallback((_: unknown, params: { nodeId: string | null }) => {
    connectStart.current = params.nodeId;
  }, []);

  const onConnectEnd = useCallback(
    (event: MouseEvent | TouchEvent, connectionState: FinalConnectionState) => {
      if (!connectionState.isValid && connectStart.current) {
        const { clientX, clientY } =
          'changedTouches' in event ? event.changedTouches[0] : (event as MouseEvent);
        const pos = screenToFlowPosition({ x: clientX, y: clientY });
        // stash the flow position for the palette add via a temp node placeholder
        setPendingConnect(connectStart.current);
        pendingPosRef.current = pos;
        setPaletteOpen(true);
      }
      connectStart.current = null;
    },
    [screenToFlowPosition],
  );
  const pendingPosRef = useRef<{ x: number; y: number } | null>(null);

  // Override palette add to honor a drop position when present.
  const handlePaletteAddResolved = useCallback(
    (type: string) => {
      if (pendingPosRef.current) {
        addNode(type, pendingPosRef.current, pendingConnect);
        pendingPosRef.current = null;
        setPaletteOpen(false);
        setPendingConnect(null);
        return;
      }
      handlePaletteAdd(type);
    },
    [addNode, pendingConnect, handlePaletteAdd],
  );

  const deleteEdge = useCallback(
    (edgeId: string) => {
      pushUndo();
      setEdges((es) => es.filter((e) => e.id !== edgeId));
      setDirty(true);
    },
    [pushUndo, setEdges],
  );

  // ── Node interactions ──
  const onNodeClick = useCallback((_: unknown, node: WfNode) => {
    if (node.type === 'sticky') return; // sticky notes have no config panel
    setConfigNodeId(node.id);
    setPaletteOpen(false);
  }, []);

  // ── Right-click context menus ──
  const onNodeContextMenu = useCallback(
    (event: React.MouseEvent, node: WfNode) => {
      event.preventDefault();
      if (node.type === 'sticky') {
        setMenu(null);
        return;
      }
      setMenu({ x: event.clientX, y: event.clientY, nodeId: node.id, isTrigger: node.data.nodeType === 'trigger' });
    },
    [],
  );

  const onPaneContextMenu = useCallback(
    (event: React.MouseEvent | MouseEvent) => {
      event.preventDefault();
      setMenu({ x: event.clientX, y: event.clientY });
    },
    [],
  );

  const openPaletteAt = useCallback(
    (client: { x: number; y: number }) => {
      pendingPosRef.current = screenToFlowPosition({ x: client.x, y: client.y });
      setPendingConnect(null);
      setConfigNodeId(null);
      setPaletteOpen(true);
    },
    [screenToFlowPosition],
  );

  const onNodeDragStop = useCallback(() => {
    setDirty(true);
  }, []);

  // Protect trigger nodes from deletion.
  const onBeforeDelete = useCallback(
    async ({ nodes: toDelete, edges: edgesToDelete }: { nodes: WfNode[]; edges: Edge[] }) => {
      const deletable = toDelete.filter((n) => n.data.nodeType !== 'trigger');
      if (deletable.length === 0 && edgesToDelete.length === 0) return false;
      pushUndo();
      setDirty(true);
      if (configNodeId && deletable.some((n) => n.id === configNodeId)) setConfigNodeId(null);
      return { nodes: deletable, edges: edgesToDelete };
    },
    [pushUndo, configNodeId],
  );

  const toggleDisable = useCallback(
    (id: string) => {
      pushUndo();
      setNodes((ns) =>
        ns.map((n) =>
          n.id === id && n.data.nodeType !== 'trigger'
            ? { ...n, data: { ...n.data, disabled: !n.data.disabled } }
            : n,
        ),
      );
      setDirty(true);
    },
    [pushUndo, setNodes],
  );

  const deleteNode = useCallback(
    (id: string) => {
      pushUndo();
      setNodes((ns) => ns.filter((n) => n.id !== id || n.data.nodeType === 'trigger'));
      setEdges((es) => es.filter((e) => e.source !== id && e.target !== id));
      setConfigNodeId(null);
      setDirty(true);
    },
    [pushUndo, setNodes, setEdges],
  );

  // ── Duplicate node(s) + their internal edges (offset 40,40) ──
  const duplicateNodes = useCallback(
    (ids: string[]) => {
      const idSet = new Set(ids);
      const toDup = nodes.filter(
        (n) => idSet.has(n.id) && n.type !== 'sticky' && n.data.nodeType !== 'trigger',
      );
      if (toDup.length === 0) return;
      pushUndo();
      const ts = Date.now();
      const idMap = new Map<string, string>();
      const clones = toDup.map((n, i) => {
        const nid = `node_${ts}_${i}`;
        idMap.set(n.id, nid);
        return {
          ...n,
          id: nid,
          selected: true,
          position: { x: n.position.x + 40, y: n.position.y + 40 },
          data: { ...n.data, config: { ...n.data.config }, status: undefined },
        } satisfies WfNode;
      });
      const clonedEdges: Edge[] = edges
        .filter((e) => idMap.has(e.source) && idMap.has(e.target))
        .map((e, i) => ({
          id: `e_${ts}_${i}`,
          source: idMap.get(e.source)!,
          target: idMap.get(e.target)!,
        }));
      setNodes((ns) => [...ns.map((n) => ({ ...n, selected: false })), ...clones]);
      if (clonedEdges.length) setEdges((es) => [...es, ...clonedEdges]);
      setDirty(true);
    },
    [nodes, edges, pushUndo, setNodes, setEdges],
  );

  const selectAllNodes = useCallback(() => {
    setNodes((ns) => ns.map((n) => ({ ...n, selected: true })));
  }, [setNodes]);

  const toggleDisableSelection = useCallback(() => {
    const sel = nodes.filter(
      (n) => n.selected && n.type !== 'sticky' && n.data.nodeType !== 'trigger',
    );
    if (sel.length === 0) return;
    pushUndo();
    const selIds = new Set(sel.map((n) => n.id));
    setNodes((ns) =>
      ns.map((n) =>
        selIds.has(n.id) ? { ...n, data: { ...n.data, disabled: !n.data.disabled } } : n,
      ),
    );
    setDirty(true);
  }, [nodes, pushUndo, setNodes]);

  // ── Sticky notes (canvas-local, not persisted) ──
  const addSticky = useCallback(
    (position?: { x: number; y: number }) => {
      const rect = wrapperRef.current?.getBoundingClientRect();
      const pos =
        position ??
        (rect
          ? screenToFlowPosition({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
          : { x: 400, y: 300 });
      const id = `note_${Date.now()}`;
      setNodes((ns) => [
        ...ns,
        {
          id,
          type: 'sticky',
          position: { x: Math.round(pos.x / 20) * 20, y: Math.round(pos.y / 20) * 20 },
          data: { nodeType: '__sticky', label: 'Note', config: {}, disabled: false },
        } satisfies WfNode,
      ]);
    },
    [screenToFlowPosition, setNodes],
  );

  // ── Save ──
  const save = useCallback(async () => {
    setSaving(true);
    try {
      const res = await api.put(`/workflows/${wfId}`, {
        name,
        description: workflow['description'] ?? '',
        status,
        triggerType,
        triggerConfig,
        nodes: toSaveNodes(nodes),
        edges: toSaveEdges(edges),
      });
      const data = res.data as Record<string, unknown> | undefined;
      if (data && typeof data['webhookSecret'] === 'string') {
        setWebhookSecret(data['webhookSecret'] as string);
      }
      setDirty(false);
      onSaved();
      toast.success('Workflow saved');
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSaving(false);
    }
  }, [wfId, name, workflow, status, triggerType, triggerConfig, nodes, edges, onSaved]);

  // ── Execute ──
  const execute = useCallback(async () => {
    setNodes((ns) => ns.map((n) => ({ ...n, data: { ...n.data, status: 'running' } })));
    try {
      const res = await api.post(`/workflows/${wfId}/execute`, { triggerData: {} });
      const exec = res.data as Execution | undefined;
      const logs = exec?.logs ?? [];
      const statuses = new Map<string, string>();
      const execData: Record<string, ExecEntry> = {};
      for (const log of logs) {
        const nid = log.nodeId ?? '';
        statuses.set(nid, log.status ?? 'completed');
        if (log.input !== undefined || log.output !== undefined) {
          execData[nid] = { input: log.input, output: log.output };
        }
      }
      setNodes((ns) =>
        ns.map((n) => ({ ...n, data: { ...n.data, status: statuses.get(n.id) } })),
      );
      if (Object.keys(execData).length) setLastExecData((d) => ({ ...d, ...execData }));
      setExecLogs(logs);
      setLogsOpen(true);
      toast.success('Workflow executed');
    } catch (e) {
      setNodes((ns) => ns.map((n) => ({ ...n, data: { ...n.data, status: undefined } })));
      toast.error(friendlyError(e));
    }
  }, [wfId, setNodes]);

  // ── Node test ──
  const testNode = useCallback(
    async (nodeId: string) => {
      setTesting(true);
      try {
        const input = pinnedData[nodeId] ?? lastExecData[nodeId]?.input ?? {};
        const res = await api.post(`/workflows/${wfId}/nodes/${nodeId}/test`, { input });
        const data = res.data as Record<string, unknown> | undefined;
        const output = data?.['output'] ?? data;
        setLastExecData((d) => ({ ...d, [nodeId]: { input, output } }));
      } catch (e) {
        setLastExecData((d) => ({
          ...d,
          [nodeId]: { input: {}, output: { error: friendlyError(e) } },
        }));
      } finally {
        setTesting(false);
      }
    },
    [wfId, pinnedData, lastExecData],
  );

  // ── Regenerate webhook secret ──
  const regenerateSecret = useCallback(async () => {
    try {
      const res = await api.post(`/workflows/${wfId}/webhook-secret`);
      const secret = (res.data as Record<string, unknown> | undefined)?.['webhookSecret'];
      if (typeof secret === 'string') {
        setWebhookSecret(secret);
        toast.success('Secret regenerated');
      }
    } catch (e) {
      toast.error(friendlyError(e));
    }
  }, [wfId]);

  // ── Load a past execution into the canvas ──
  const loadExecution = useCallback(
    (exec: Execution) => {
      const logs = exec.logs ?? [];
      const statuses = new Map<string, string>();
      const execData: Record<string, ExecEntry> = {};
      for (const log of logs) {
        const nid = log.nodeId ?? '';
        statuses.set(nid, log.status ?? 'completed');
        if (log.input !== undefined || log.output !== undefined) {
          execData[nid] = { input: log.input, output: log.output };
        }
      }
      setNodes((ns) => ns.map((n) => ({ ...n, data: { ...n.data, status: statuses.get(n.id) } })));
      if (Object.keys(execData).length) setLastExecData(execData);
      setExecLogs(logs);
      setLogsOpen(true);
    },
    [setNodes],
  );

  const closePanels = useCallback(() => {
    setMenu(null);
    setPaletteOpen(false);
    setConfigNodeId(null);
    setLogsOpen(false);
    setNodes((ns) => (ns.some((n) => n.selected) ? ns.map((n) => ({ ...n, selected: false })) : ns));
  }, [setNodes]);

  // ── Keyboard shortcuts ──
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      const typing =
        ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) || target.isContentEditable;
      const mod = e.metaKey || e.ctrlKey;
      const key = e.key.toLowerCase();
      if (mod && key === 's') {
        e.preventDefault();
        save();
      } else if (mod && key === 'z' && !typing) {
        e.preventDefault();
        if (e.shiftKey) applyRedo();
        else applyUndo();
      } else if (mod && key === 'd' && !typing) {
        e.preventDefault();
        duplicateNodes(nodes.filter((n) => n.selected).map((n) => n.id));
      } else if (mod && key === 'a' && !typing) {
        e.preventDefault();
        selectAllNodes();
      } else if (key === 'escape') {
        closePanels();
      } else if (!typing && !mod && key === 'd') {
        e.preventDefault();
        toggleDisableSelection();
      } else if (!typing && !mod && key === 'tab') {
        e.preventDefault();
        openPaletteCenter();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [
    save,
    applyUndo,
    applyRedo,
    duplicateNodes,
    selectAllNodes,
    toggleDisableSelection,
    closePanels,
    openPaletteCenter,
    nodes,
  ]);

  const configNode = useMemo(
    () => nodes.find((n) => n.id === configNodeId) ?? null,
    [nodes, configNodeId],
  );

  return (
    <div className="flex h-full flex-col bg-background">
      {/* Toolbar */}
      <div className="flex h-[52px] shrink-0 items-center gap-2 border-b border-border px-4">
        <button
          type="button"
          onClick={onBack}
          title="Back"
          className="text-text-secondary hover:text-text-primary"
        >
          <ArrowLeft size={18} />
        </button>
        <input
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setDirty(true);
          }}
          className="w-56 bg-transparent px-1 py-1.5 text-[length:var(--text-subhead)] font-semibold text-text-primary outline-none"
        />
        <StatusSelect
          status={status}
          onChange={(s) => {
            setStatus(s);
            setDirty(true);
          }}
        />

        <div className="ml-auto flex items-center gap-1">
          <button
            type="button"
            title="Undo"
            disabled={undoStack.current.length === 0}
            onClick={applyUndo}
            className="p-1.5 text-text-secondary hover:text-text-primary disabled:text-text-subtle"
          >
            <Undo2 size={15} />
          </button>
          <button
            type="button"
            title="Redo"
            disabled={redoStack.current.length === 0}
            onClick={applyRedo}
            className="p-1.5 text-text-secondary hover:text-text-primary disabled:text-text-subtle"
          >
            <Redo2 size={15} />
          </button>
          <div className="mx-1 h-6 w-px bg-border" />
          <Button variant="ghost" size="sm" onClick={openPaletteCenter}>
            <Plus size={15} />
            Add node
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title="Add sticky note"
            onClick={() => addSticky()}
          >
            <StickyNote size={15} />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            title="Toggle logs"
            onClick={() => setLogsOpen((v) => !v)}
            style={logsOpen ? { color: 'var(--color-accent)' } : undefined}
          >
            <Terminal size={15} />
          </Button>
          <Button variant="ghost" size="sm" onClick={() => setExecsOpen(true)}>
            <History size={15} />
            History
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={execute}
            style={{ color: GREEN, borderColor: GREEN }}
          >
            <Play size={14} />
            Execute
          </Button>
          <Button variant="primary" size="sm" loading={saving} onClick={save}>
            <Save size={14} />
            {dirty ? 'Save*' : 'Saved'}
          </Button>
        </div>
      </div>

      {/* Canvas + panels */}
      <div className="flex min-h-0 flex-1">
        <div ref={wrapperRef} className="flex min-w-0 flex-1 flex-col">
          <div className="min-h-0 flex-1">
            <ReactFlow
              nodes={nodes}
              edges={edges}
              nodeTypes={nodeTypes}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onConnectStart={onConnectStart}
              onConnectEnd={onConnectEnd}
              onNodeClick={onNodeClick}
              onNodeContextMenu={onNodeContextMenu}
              onPaneContextMenu={onPaneContextMenu}
              onNodeDragStop={onNodeDragStop}
              onBeforeDelete={onBeforeDelete}
              onPaneClick={() => {
                setConfigNodeId(null);
                setPaletteOpen(false);
                setMenu(null);
              }}
              deleteKeyCode={['Delete', 'Backspace']}
              snapToGrid
              snapGrid={[20, 20]}
              minZoom={0.15}
              maxZoom={3}
              fitView
              proOptions={{ hideAttribution: true }}
              colorMode="dark"
            >
              <Background variant={BackgroundVariant.Dots} gap={24} size={1} />
              <Controls showInteractive={false} />
              {showMinimap && (
                <MiniMap
                  pannable
                  zoomable
                  nodeColor={(n) =>
                    n.type === 'sticky' ? ORANGE : defFor((n.data as WfNodeData).nodeType).color
                  }
                  maskColor="rgba(0,0,0,0.6)"
                  style={{ background: 'var(--color-surface)' }}
                />
              )}
              <Panel position="bottom-right">
                <ZoomPanel
                  minimapOn={showMinimap}
                  onToggleMinimap={() => setShowMinimap((v) => !v)}
                  onResetZoom={() => zoomTo(1, { duration: 200 })}
                />
              </Panel>
            </ReactFlow>
          </div>
          {logsOpen && <LiveLogsPanel logs={execLogs} onClose={() => setLogsOpen(false)} />}
        </div>

        {configNode && (
          <NodeConfigPanel
            node={configNode}
            nodes={nodes}
            edges={edges}
            wfId={wfId}
            triggerType={triggerType}
            onTriggerTypeChange={(t) => {
              setTriggerType(t);
              setDirty(true);
            }}
            triggerConfig={triggerConfig}
            onTriggerConfigChange={(key, value) => {
              setTriggerConfig((c) => ({ ...c, [key]: value }));
              setDirty(true);
            }}
            webhookSecret={webhookSecret}
            onRegenerateSecret={regenerateSecret}
            onLabelChange={(label) => setNodeData(configNode.id, { label })}
            onConfigChange={(key, value) =>
              setNodeData(configNode.id, { config: { ...configNode.data.config, [key]: value } })
            }
            onOnErrorChange={(value) => setNodeData(configNode.id, { onError: value })}
            onDelete={() => deleteNode(configNode.id)}
            onToggleDisable={() => toggleDisable(configNode.id)}
            onTest={() => testNode(configNode.id)}
            testing={testing}
            onDeleteEdge={deleteEdge}
            lastExecData={lastExecData}
            pinnedData={pinnedData}
            onTogglePin={() =>
              setPinnedData((p) => {
                const next = { ...p };
                if (configNode.id in next) delete next[configNode.id];
                else next[configNode.id] = lastExecData[configNode.id]?.output;
                return next;
              })
            }
            onClose={() => setConfigNodeId(null)}
          />
        )}

        {paletteOpen && (
          <NodePalette onAdd={handlePaletteAddResolved} onClose={() => setPaletteOpen(false)} />
        )}
      </div>

      {menu && (
        <CanvasContextMenu
          menu={menu}
          onClose={() => setMenu(null)}
          onOpen={(id) => {
            setConfigNodeId(id);
            setPaletteOpen(false);
          }}
          onDuplicate={(id) => duplicateNodes([id])}
          onToggleDisable={toggleDisable}
          onDelete={deleteNode}
          onAddNode={() => openPaletteAt({ x: menu.x, y: menu.y })}
          onSelectAll={selectAllNodes}
          onFitView={() => fitView({ duration: 300 })}
        />
      )}

      <ExecutionsPanel
        wfId={wfId}
        open={execsOpen}
        onOpenChange={setExecsOpen}
        onLoad={loadExecution}
      />
    </div>
  );
}

function StatusSelect({ status, onChange }: { status: string; onChange: (s: string) => void }) {
  const color = status === 'active' ? GREEN : status === 'paused' ? ORANGE : '#64748B';
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className="flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[length:var(--text-label)] font-medium"
          style={{
            color,
            borderColor: `color-mix(in srgb, ${color} 30%, transparent)`,
            background: `color-mix(in srgb, ${color} 12%, transparent)`,
          }}
        >
          <span className="h-1.5 w-1.5 rounded-full" style={{ background: color }} />
          {status}
          <ChevronDown size={12} />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        {['draft', 'active', 'paused'].map((s) => (
          <DropdownMenuItem key={s} onClick={() => onChange(s)}>
            {s}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
