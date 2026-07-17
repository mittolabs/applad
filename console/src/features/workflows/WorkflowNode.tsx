import { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { defFor, ACCENT, GREEN, RED } from './nodeDefs';

export interface WfNodeData extends Record<string, unknown> {
  nodeType: string;
  label: string;
  config: Record<string, unknown>;
  disabled: boolean;
  onError?: string;
  /** Execution status: running | completed | failed | skipped */
  status?: string;
}

const NODE_W = 220;
const NODE_H = 72;

function statusColor(status: string): string {
  switch (status) {
    case 'completed':
      return GREEN;
    case 'failed':
      return RED;
    case 'skipped':
      return '#64748B';
    default:
      return ACCENT;
  }
}

function WorkflowNodeComponent({ id, data, selected }: NodeProps) {
  const d = defFor(data.nodeType as string);
  const nodeData = data as WfNodeData;
  const Icon = d.icon;
  const disabled = nodeData.disabled === true;
  const status = nodeData.status;
  const inputs = d.inputs;
  const outputs = d.outputs;
  const labels = d.outputLabels;

  return (
    <div
      className="relative flex items-center rounded-[var(--radius-10)] border"
      style={{
        width: NODE_W,
        height: NODE_H,
        background: disabled ? 'var(--color-surface-alt, #111114)' : 'var(--color-surface)',
        borderColor: selected ? d.color : 'var(--color-border)',
        borderWidth: selected ? 1.5 : 1,
        boxShadow: '0 4px 12px rgba(0,0,0,0.25)',
        opacity: disabled ? 0.7 : 1,
      }}
    >
      {/* Left color bar */}
      <div
        className="absolute left-0 top-0 h-full w-1 rounded-l-[var(--radius-10)]"
        style={{ background: disabled ? '#333333' : d.color }}
      />

      {/* Icon */}
      <div
        className="ml-4 flex h-8 w-8 shrink-0 items-center justify-center rounded-[var(--radius-7)]"
        style={{ background: `color-mix(in srgb, ${d.color} 12%, transparent)`, color: d.color }}
      >
        <Icon size={15} />
      </div>

      {/* Label + type */}
      <div className="ml-3 min-w-0 flex-1 pr-3">
        <div
          className="truncate text-[length:var(--text-body)] font-medium"
          style={{ color: disabled ? 'var(--color-text-subtle)' : 'var(--color-text-primary)' }}
        >
          {nodeData.label || d.label}
        </div>
        <div className="truncate text-[length:var(--text-caption)] text-text-subtle">
          {nodeData.nodeType}
        </div>
      </div>

      {/* Disabled strikethrough */}
      {disabled && (
        <div
          className="pointer-events-none absolute left-0 top-1/2 h-px w-full"
          style={{ background: 'rgba(239,68,68,0.3)' }}
        />
      )}

      {/* Execution status badge */}
      {status && (
        <div
          className="absolute -right-1 -top-1 h-3 w-3 rounded-full border-2"
          style={{ background: statusColor(status), borderColor: 'var(--color-surface)' }}
        />
      )}

      {/* Input handles */}
      {inputs > 0 &&
        Array.from({ length: inputs }).map((_, i) => (
          <Handle
            key={`in-${i}`}
            id={inputs > 1 ? `i${i}` : undefined}
            type="target"
            position={Position.Left}
            style={{
              top: `${((i + 1) / (inputs + 1)) * 100}%`,
              background: 'var(--color-surface)',
              border: '1.5px solid var(--color-text-secondary)',
              width: 12,
              height: 12,
            }}
          />
        ))}

      {/* Output handles (with labels for multi-output) */}
      {Array.from({ length: outputs }).map((_, i) => (
        <div key={`out-${i}`}>
          <Handle
            id={outputs > 1 ? `o${i}` : undefined}
            type="source"
            position={Position.Right}
            style={{
              top: `${((i + 1) / (outputs + 1)) * 100}%`,
              background: ACCENT,
              border: '1.5px solid rgba(255,255,255,0.3)',
              width: 12,
              height: 12,
            }}
          />
          {outputs > 1 && labels && labels[i] && (
            <span
              className="pointer-events-none absolute text-[length:var(--text-2xs)] text-text-subtle"
              style={{ right: -6, top: `calc(${((i + 1) / (outputs + 1)) * 100}% - 6px)`, transform: 'translateX(100%)' }}
            >
              {labels[i]}
            </span>
          )}
        </div>
      ))}

      {/* keep id referenced for React devtools clarity */}
      <span data-node-id={id} className="hidden" />
    </div>
  );
}

export const WorkflowNode = memo(WorkflowNodeComponent);
