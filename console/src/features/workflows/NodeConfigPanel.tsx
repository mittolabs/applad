import { useState } from 'react';
import type { Edge, Node } from '@xyflow/react';
import {
  ArrowDownToLine,
  ArrowLeft,
  ArrowRight,
  ArrowUpFromLine,
  Clock,
  Copy,
  Pin,
  PinOff,
  Play,
  RefreshCw,
  ToggleLeft,
  ToggleRight,
  Trash2,
  Webhook,
  X,
} from 'lucide-react';
import { toast } from '@/components/toast';
import { defFor, GREEN, ORANGE, RED, TYPE_FIELDS, type FieldSpec } from './nodeDefs';
import type { WfNodeData } from './WorkflowNode';

const ACCENT_PURPLE = '#6C47FF';

export interface ExecEntry {
  input?: unknown;
  output?: unknown;
}

export function NodeConfigPanel({
  node,
  nodes,
  edges,
  wfId,
  triggerType,
  onTriggerTypeChange,
  triggerConfig,
  onTriggerConfigChange,
  webhookSecret,
  onRegenerateSecret,
  onLabelChange,
  onConfigChange,
  onOnErrorChange,
  onDelete,
  onToggleDisable,
  onTest,
  testing,
  onDeleteEdge,
  lastExecData,
  pinnedData,
  onTogglePin,
  onClose,
}: {
  node: Node<WfNodeData>;
  nodes: Node<WfNodeData>[];
  edges: Edge[];
  wfId: string;
  triggerType: string;
  onTriggerTypeChange: (t: string) => void;
  triggerConfig: Record<string, unknown>;
  onTriggerConfigChange: (key: string, value: string) => void;
  webhookSecret: string;
  onRegenerateSecret: () => void;
  onLabelChange: (label: string) => void;
  onConfigChange: (key: string, value: string) => void;
  onOnErrorChange: (value: string) => void;
  onDelete: () => void;
  onToggleDisable: () => void;
  onTest: () => void;
  testing: boolean;
  onDeleteEdge: (edgeId: string) => void;
  lastExecData: Record<string, ExecEntry>;
  pinnedData: Record<string, unknown>;
  onTogglePin: () => void;
  onClose: () => void;
}) {
  const [tab, setTab] = useState(0);
  const d = defFor(node.data.nodeType);
  const isTrigger = node.data.nodeType === 'trigger';
  const disabled = node.data.disabled === true;
  const Icon = d.icon;

  return (
    <div className="flex h-full w-[340px] shrink-0 flex-col border-l border-border bg-surface">
      {/* Header */}
      <div className="flex items-center gap-2.5 px-5 pb-3 pt-4">
        <div
          className="flex h-[30px] w-[30px] items-center justify-center rounded-[var(--radius-7)]"
          style={{ background: `color-mix(in srgb, ${d.color} 12%, transparent)`, color: d.color }}
        >
          <Icon size={14} />
        </div>
        <span className="flex-1 truncate text-[length:var(--text-control)] font-semibold text-text-primary">
          {d.label}
        </span>
        {!isTrigger && (
          <>
            <button
              type="button"
              title={disabled ? 'Enable' : 'Disable'}
              onClick={onToggleDisable}
              style={{ color: disabled ? RED : GREEN }}
            >
              {disabled ? <ToggleLeft size={16} /> : <ToggleRight size={16} />}
            </button>
            <button type="button" title="Delete" onClick={onDelete} className="text-[#ff5252]">
              <Trash2 size={15} />
            </button>
            <button
              type="button"
              title="Test this step"
              onClick={onTest}
              disabled={testing}
              style={{ color: GREEN }}
            >
              {testing ? <RefreshCw size={14} className="animate-spin" /> : <Play size={14} />}
            </button>
          </>
        )}
        <button type="button" onClick={onClose} className="text-text-secondary hover:text-text-primary">
          <X size={16} />
        </button>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-border">
        {['Settings', 'Input', 'Output'].map((label, i) => (
          <button
            key={label}
            type="button"
            onClick={() => setTab(i)}
            className="flex-1 border-b-2 py-2.5 text-center text-[length:var(--text-label)] transition-colors"
            style={{
              borderBottomColor: tab === i ? 'var(--color-accent)' : 'transparent',
              color: tab === i ? 'var(--color-text-primary)' : 'var(--color-text-subtle)',
              fontWeight: tab === i ? 500 : 400,
            }}
          >
            {label}
          </button>
        ))}
      </div>

      {/* Body */}
      <div className="min-h-0 flex-1 overflow-y-auto">
        {tab === 0 ? (
          <SettingsTab
            node={node}
            nodes={nodes}
            edges={edges}
            isTrigger={isTrigger}
            wfId={wfId}
            triggerType={triggerType}
            onTriggerTypeChange={onTriggerTypeChange}
            triggerConfig={triggerConfig}
            onTriggerConfigChange={onTriggerConfigChange}
            webhookSecret={webhookSecret}
            onRegenerateSecret={onRegenerateSecret}
            onLabelChange={onLabelChange}
            onConfigChange={onConfigChange}
            onOnErrorChange={onOnErrorChange}
            onDeleteEdge={onDeleteEdge}
            lastExecData={lastExecData}
          />
        ) : (
          <DataPreview
            nodeId={node.id}
            isInput={tab === 1}
            lastExecData={lastExecData}
            pinnedData={pinnedData}
            onTogglePin={onTogglePin}
          />
        )}
      </div>
    </div>
  );
}

function CfgLabel({ children }: { children: React.ReactNode }) {
  return (
    <div className="text-[length:var(--text-label)] font-medium text-text-secondary">{children}</div>
  );
}

function upstreamSuggestions(
  nodeId: string,
  nodes: Node<WfNodeData>[],
  edges: Edge[],
  lastExecData: Record<string, ExecEntry>,
): string[] {
  const byId = (id: string) => nodes.find((n) => n.id === id);
  const visited = new Set<string>();
  const queue = [nodeId];
  const upstream: string[] = [];
  while (queue.length) {
    const cur = queue.pop()!;
    if (visited.has(cur)) continue;
    visited.add(cur);
    for (const e of edges) {
      if (e.target === cur && !visited.has(e.source)) {
        upstream.push(e.source);
        queue.push(e.source);
      }
    }
  }

  const result: string[] = [];
  if (upstream.some((id) => byId(id)?.data.nodeType === 'trigger')) {
    result.push(
      '{{.trigger.body}}',
      '{{.trigger.body.email}}',
      '{{.trigger.body.name}}',
      '{{.trigger.headers}}',
      '{{.trigger.params}}',
    );
  }
  for (const id of upstream) {
    const n = byId(id);
    if (!n) continue;
    const label = (n.data.label || id).toLowerCase().replace(/ /g, '_');
    const out = lastExecData[id]?.output;
    if (out && typeof out === 'object' && !Array.isArray(out)) {
      for (const k of Object.keys(out as Record<string, unknown>).slice(0, 5)) {
        result.push(`{{.${label}.output.${k}}}`);
      }
    } else {
      result.push(`{{.${label}.output}}`);
    }
  }
  return result.slice(0, 5);
}

function SettingsTab({
  node,
  nodes,
  edges,
  isTrigger,
  wfId,
  triggerType,
  onTriggerTypeChange,
  triggerConfig,
  onTriggerConfigChange,
  webhookSecret,
  onRegenerateSecret,
  onLabelChange,
  onConfigChange,
  onOnErrorChange,
  onDeleteEdge,
  lastExecData,
}: {
  node: Node<WfNodeData>;
  nodes: Node<WfNodeData>[];
  edges: Edge[];
  isTrigger: boolean;
  wfId: string;
  triggerType: string;
  onTriggerTypeChange: (t: string) => void;
  triggerConfig: Record<string, unknown>;
  onTriggerConfigChange: (key: string, value: string) => void;
  webhookSecret: string;
  onRegenerateSecret: () => void;
  onLabelChange: (label: string) => void;
  onConfigChange: (key: string, value: string) => void;
  onOnErrorChange: (value: string) => void;
  onDeleteEdge: (edgeId: string) => void;
  lastExecData: Record<string, ExecEntry>;
}) {
  const config = node.data.config ?? {};
  const fields: FieldSpec[] = isTrigger ? [] : TYPE_FIELDS[node.data.nodeType] ?? [];

  return (
    <div className="flex flex-col gap-4 p-5">
      <div className="flex flex-col gap-1.5">
        <CfgLabel>Label</CfgLabel>
        <TextInput
          value={node.data.label ?? ''}
          placeholder="Node label"
          onChange={onLabelChange}
        />
      </div>

      {isTrigger ? (
        <TriggerSettings
          wfId={wfId}
          triggerType={triggerType}
          onTriggerTypeChange={onTriggerTypeChange}
          triggerConfig={triggerConfig}
          onTriggerConfigChange={onTriggerConfigChange}
          webhookSecret={webhookSecret}
          onRegenerateSecret={onRegenerateSecret}
        />
      ) : (
        <>
          {fields.map((f) => (
            <ConfigField
              key={f.key}
              spec={f}
              value={String(config[f.key] ?? '')}
              onChange={(v) => onConfigChange(f.key, v)}
              suggestions={
                f.expr ? upstreamSuggestions(node.id, nodes, edges, lastExecData) : []
              }
              onAppend={(s) => onConfigChange(f.key, `${String(config[f.key] ?? '')}${s}`)}
            />
          ))}

          <div className="h-px bg-border" />
          <div className="flex flex-col gap-2">
            <CfgLabel>On error</CfgLabel>
            <OnErrorSelect value={(node.data.onError as string) ?? 'stop'} onChange={onOnErrorChange} />
          </div>
        </>
      )}

      <div className="flex flex-col gap-2">
        <CfgLabel>Connections</CfgLabel>
        <ConnectionsList nodeId={node.id} nodes={nodes} edges={edges} onDeleteEdge={onDeleteEdge} />
      </div>
    </div>
  );
}

function ConfigField({
  spec,
  value,
  onChange,
  suggestions,
  onAppend,
}: {
  spec: FieldSpec;
  value: string;
  onChange: (v: string) => void;
  suggestions: string[];
  onAppend: (s: string) => void;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <CfgLabel>{spec.label}</CfgLabel>
      {spec.lines && spec.lines > 1 ? (
        <TextArea value={value} placeholder={spec.hint} rows={spec.lines} onChange={onChange} />
      ) : (
        <TextInput value={value} placeholder={spec.hint} onChange={onChange} />
      )}
      {spec.expr && suggestions.length > 0 && (
        <div className="flex gap-1 overflow-x-auto pb-1">
          {suggestions.map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => onAppend(s)}
              className="shrink-0 rounded-[var(--radius-sm)] border border-border bg-fill px-1.5 py-0.5 text-[length:var(--text-2xs)] text-text-subtle hover:text-text-secondary"
            >
              {s}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function OnErrorSelect({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const labels: Record<string, string> = {
    stop: 'Stop workflow',
    continue: 'Continue (skip node)',
    error_output: 'Route to error output',
  };
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="rounded-[var(--radius-6)] border border-border bg-fill px-3 py-2 text-[length:var(--text-body)] text-text-primary outline-none focus:border-[var(--color-accent)]"
    >
      {Object.entries(labels).map(([k, v]) => (
        <option key={k} value={k}>
          {v}
        </option>
      ))}
    </select>
  );
}

function ConnectionsList({
  nodeId,
  nodes,
  edges,
  onDeleteEdge,
}: {
  nodeId: string;
  nodes: Node<WfNodeData>[];
  edges: Edge[];
  onDeleteEdge: (id: string) => void;
}) {
  const byId = (id: string) => nodes.find((n) => n.id === id);
  const ins = edges.filter((e) => e.target === nodeId);
  const outs = edges.filter((e) => e.source === nodeId);
  if (ins.length === 0 && outs.length === 0) {
    return <div className="text-[length:var(--text-body)] text-text-subtle">No connections</div>;
  }
  return (
    <div className="flex flex-col gap-1.5">
      {ins.map((e) => (
        <ConnChip
          key={e.id}
          icon={<ArrowLeft size={13} />}
          label={`From: ${byId(e.source)?.data.label ?? '?'}`}
          onDelete={() => onDeleteEdge(e.id)}
        />
      ))}
      {outs.map((e) => (
        <ConnChip
          key={e.id}
          icon={<ArrowRight size={13} />}
          label={`To: ${byId(e.target)?.data.label ?? '?'}`}
          onDelete={() => onDeleteEdge(e.id)}
        />
      ))}
    </div>
  );
}

function ConnChip({
  icon,
  label,
  onDelete,
}: {
  icon: React.ReactNode;
  label: string;
  onDelete: () => void;
}) {
  return (
    <div className="flex items-center gap-2 rounded-[var(--radius-6)] border border-border bg-fill px-2.5 py-1.5">
      <span className="text-text-subtle">{icon}</span>
      <span className="flex-1 truncate text-[length:var(--text-label)] text-text-secondary">
        {label}
      </span>
      <button type="button" onClick={onDelete} className="text-text-subtle hover:text-text-secondary">
        <X size={12} />
      </button>
    </div>
  );
}

// ── Trigger settings ──

function TriggerSettings({
  wfId,
  triggerType,
  onTriggerTypeChange,
  triggerConfig,
  onTriggerConfigChange,
  webhookSecret,
  onRegenerateSecret,
}: {
  wfId: string;
  triggerType: string;
  onTriggerTypeChange: (t: string) => void;
  triggerConfig: Record<string, unknown>;
  onTriggerConfigChange: (key: string, value: string) => void;
  webhookSecret: string;
  onRegenerateSecret: () => void;
}) {
  const cron = String(triggerConfig['cron'] ?? '');
  return (
    <>
      <div className="h-px bg-border" />
      <div className="flex flex-col gap-2">
        <CfgLabel>Trigger type</CfgLabel>
        <div className="flex gap-2">
          <TrigChip
            active={triggerType === 'manual'}
            icon={<Play size={12} />}
            label="Manual"
            onClick={() => onTriggerTypeChange('manual')}
          />
          <TrigChip
            active={triggerType === 'webhook'}
            icon={<Webhook size={12} />}
            label="Webhook"
            onClick={() => onTriggerTypeChange('webhook')}
          />
          <TrigChip
            active={triggerType === 'cron'}
            icon={<Clock size={12} />}
            label="Schedule"
            onClick={() => onTriggerTypeChange('cron')}
          />
        </div>
      </div>

      {triggerType === 'manual' && (
        <p className="text-[length:var(--text-label)] text-text-secondary">
          Run this workflow manually from the dashboard or via the API.
        </p>
      )}

      {triggerType === 'webhook' && (
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <CfgLabel>Webhook URL</CfgLabel>
            <CopyRow text={`${window.location.origin}/v1/workflows/webhooks/${wfId}`} />
            <p className="text-[length:var(--text-caption)] text-text-secondary">
              Send POST requests to this URL to trigger the workflow.
            </p>
          </div>
          <div className="flex flex-col gap-1.5">
            <div className="flex items-center justify-between">
              <CfgLabel>Signing secret</CfgLabel>
              <button
                type="button"
                onClick={onRegenerateSecret}
                className="flex items-center gap-1 text-[length:var(--text-caption)] text-text-secondary hover:text-text-primary"
              >
                <RefreshCw size={11} />
                Regenerate
              </button>
            </div>
            {webhookSecret ? (
              <CopyRow text={webhookSecret} />
            ) : (
              <p className="text-[length:var(--text-caption)] text-text-secondary">
                Save the workflow once to generate a signing secret.
              </p>
            )}
            <p className="text-[length:var(--text-caption)] text-text-secondary">
              Verify requests with X-Applad-Signature: hex(hmac-sha256(secret, body)).
            </p>
          </div>
        </div>
      )}

      {triggerType === 'cron' && (
        <div className="flex flex-col gap-1.5">
          <CfgLabel>Cron expression</CfgLabel>
          <TextInput
            value={cron}
            placeholder="*/5 * * * *"
            onChange={(v) => onTriggerConfigChange('cron', v)}
          />
          <p className="text-[length:var(--text-caption)] text-text-secondary">{describeCron(cron)}</p>
          <p className="text-[length:var(--text-caption)] text-text-subtle">
            Format: minute hour day month weekday (UTC)
          </p>
        </div>
      )}
    </>
  );
}

function TrigChip({
  active,
  icon,
  label,
  onClick,
}: {
  active: boolean;
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex items-center gap-1.5 rounded-[var(--radius-6)] border px-2.5 py-1.5 text-[length:var(--text-caption)]"
      style={{
        borderColor: active ? ACCENT_PURPLE : 'var(--color-border)',
        background: active ? `color-mix(in srgb, ${ACCENT_PURPLE} 15%, transparent)` : 'transparent',
        color: active ? ACCENT_PURPLE : 'var(--color-text-secondary)',
        fontWeight: active ? 600 : 400,
      }}
    >
      {icon}
      {label}
    </button>
  );
}

function CopyRow({ text }: { text: string }) {
  return (
    <div className="flex items-center gap-2 rounded-[var(--radius-6)] border border-border bg-background px-2.5 py-2">
      <span className="flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-primary">
        {text}
      </span>
      <button
        type="button"
        onClick={() => {
          navigator.clipboard.writeText(text);
          toast.success('Copied');
        }}
        className="text-text-secondary hover:text-text-primary"
      >
        <Copy size={13} />
      </button>
    </div>
  );
}

function describeCron(expr: string): string {
  const parts = expr.trim().split(/\s+/);
  if (parts.length !== 5) {
    return expr.trim() === '' ? 'Enter a cron expression above.' : 'Invalid expression (need 5 fields).';
  }
  const [min, hr, dom, mon, dow] = parts;
  if (min === '*' && hr === '*' && dom === '*' && mon === '*' && dow === '*') return 'Every minute';
  if (min.startsWith('*/') && hr === '*' && dom === '*' && mon === '*' && dow === '*')
    return `Every ${min.slice(2)} minutes`;
  if (min === '0' && hr.startsWith('*/') && dom === '*' && mon === '*' && dow === '*')
    return `Every ${hr.slice(2)} hours`;
  if (min === '0' && hr === '0' && dom === '*' && mon === '*' && dow === '*')
    return 'Daily at midnight UTC';
  if (min === '0' && dom === '*' && mon === '*' && dow === '*') return `Daily at ${hr}:00 UTC`;
  if (min === '0' && hr === '0' && dom === '1' && mon === '*' && dow === '*')
    return 'Monthly on the 1st at midnight UTC';
  return expr;
}

// ── Data preview (Input/Output tabs) ──

function DataPreview({
  nodeId,
  isInput,
  lastExecData,
  pinnedData,
  onTogglePin,
}: {
  nodeId: string;
  isInput: boolean;
  lastExecData: Record<string, ExecEntry>;
  pinnedData: Record<string, unknown>;
  onTogglePin: () => void;
}) {
  const entry = lastExecData[nodeId];
  const pinned = pinnedData[nodeId];
  const display = isInput ? entry?.input : pinned ?? entry?.output;

  if (display === undefined || display === null) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 p-8 text-center">
        {isInput ? (
          <ArrowDownToLine size={24} className="text-text-subtle" />
        ) : (
          <ArrowUpFromLine size={24} className="text-text-subtle" />
        )}
        <p className="text-[length:var(--text-label)] text-text-subtle">
          Execute the workflow to see {isInput ? 'input' : 'output'} data
        </p>
      </div>
    );
  }

  const jsonStr = (() => {
    try {
      return JSON.stringify(display, null, 2);
    } catch {
      return String(display);
    }
  })();

  return (
    <div className="flex h-full flex-col">
      {!isInput && entry?.output !== undefined && (
        <div className="flex items-center gap-2 px-3 pt-2">
          {pinned !== undefined && (
            <span
              className="rounded-[var(--radius-sm)] px-1.5 py-0.5 text-[length:var(--text-2xs)] font-semibold"
              style={{ background: `color-mix(in srgb, ${ORANGE} 15%, transparent)`, color: ORANGE }}
            >
              PINNED
            </span>
          )}
          <div className="flex-1" />
          <button
            type="button"
            onClick={onTogglePin}
            style={{ color: pinned !== undefined ? ORANGE : 'var(--color-text-subtle)' }}
          >
            {pinned !== undefined ? <PinOff size={14} /> : <Pin size={14} />}
          </button>
        </div>
      )}
      <pre className="min-h-0 flex-1 overflow-auto whitespace-pre-wrap break-words p-3 font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-secondary">
        {jsonStr}
      </pre>
    </div>
  );
}

// ── Small styled inputs ──

function TextInput({
  value,
  placeholder,
  onChange,
}: {
  value: string;
  placeholder?: string;
  onChange: (v: string) => void;
}) {
  return (
    <input
      value={value}
      placeholder={placeholder}
      onChange={(e) => onChange(e.target.value)}
      className="rounded-[var(--radius-6)] border border-border bg-fill px-3 py-2 text-[length:var(--text-body)] text-text-primary outline-none placeholder:text-text-subtle focus:border-[var(--color-accent)]"
    />
  );
}

function TextArea({
  value,
  placeholder,
  rows,
  onChange,
}: {
  value: string;
  placeholder?: string;
  rows: number;
  onChange: (v: string) => void;
}) {
  return (
    <textarea
      value={value}
      placeholder={placeholder}
      rows={rows}
      onChange={(e) => onChange(e.target.value)}
      className="resize-y rounded-[var(--radius-6)] border border-border bg-fill px-3 py-2 font-[family-name:var(--font-mono)] text-[length:var(--text-body)] text-text-primary outline-none placeholder:text-text-subtle focus:border-[var(--color-accent)]"
    />
  );
}
