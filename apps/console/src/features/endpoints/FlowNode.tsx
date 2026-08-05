import { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { BLOCK_DEFS, type BlockShape, type BlockType, blockSummary } from './blockDefs';

export interface FlowNodeData extends Record<string, unknown> {
  blockType: BlockType;
  label: string;
  config: Record<string, unknown>;
  /** Test trace status: completed | failed | skipped. */
  status?: string;
}

/** The bounding box each shape is drawn within. React Flow measures this. */
export const NODE_W = 210;
export const NODE_H: Record<BlockShape, number> = {
  terminator: 60,
  process: 74,
  io: 74,
  decision: 128,
};

const HANDLE_STYLE: React.CSSProperties = {
  width: 11,
  height: 11,
  background: 'var(--color-surface)',
  border: '1.5px solid var(--color-text-secondary)',
};

function statusColor(status?: string): string | null {
  if (status === 'completed') return '#35c07f';
  if (status === 'failed') return 'var(--color-danger)';
  if (status === 'skipped') return '#64748b';
  return null;
}

function FlowNodeComponent({ data, selected }: NodeProps) {
  const d = data as FlowNodeData;
  const def = BLOCK_DEFS[d.blockType];
  const Icon = def.icon;
  const shape = def.shape;
  const accent = def.accent;
  const w = NODE_W;
  const h = NODE_H[shape];
  const status = statusColor(d.status);

  const isStart = d.blockType === 'endpoint_handler';
  const isEnd = d.blockType === 'endpoint_response';
  const summary = blockSummary({ id: '', type: d.blockType, label: d.label, config: d.config });

  // A colored outline that survives clip-path: an accent-filled shape with a
  // slightly inset surface shape on top. Selection brightens the outline.
  const outlineColor = selected ? accent : 'var(--color-border)';
  const outlineW = selected ? 2 : 1.5;

  const clip = CLIP[shape];
  const shapeWrap: React.CSSProperties = clip
    ? { clipPath: clip, background: outlineColor }
    : {
        borderRadius: shape === 'terminator' ? 9999 : 'var(--radius-10)',
        border: `${outlineW}px solid ${outlineColor}`,
        background: 'var(--color-surface)',
      };
  const inner: React.CSSProperties = clip
    ? {
        clipPath: clip,
        background: 'var(--color-surface)',
        position: 'absolute',
        inset: outlineW,
      }
    : {};

  return (
    <div style={{ width: w, height: h, position: 'relative' }}>
      {/* Input handle on top (all but Start). */}
      {!isStart && <Handle type="target" position={Position.Top} style={HANDLE_STYLE} />}

      <div style={{ width: '100%', height: '100%', position: 'relative', ...shapeWrap }}>
        {clip && <div style={inner} />}
        <div
          className="absolute inset-0 flex flex-col items-center justify-center px-3 text-center"
          style={shape === 'process' || shape === 'io' ? { alignItems: 'flex-start', textAlign: 'left', paddingLeft: 44 } : undefined}
        >
          <div className="flex items-center gap-1.5">
            {(shape === 'terminator' || shape === 'decision') && (
              <Icon size={13} style={{ color: accent }} />
            )}
            <span className="truncate text-[length:var(--text-body)] font-semibold text-text-primary">
              {d.label || def.label}
            </span>
          </div>
          {summary && shape !== 'decision' && (
            <span className="mt-0.5 max-w-full truncate text-[length:var(--text-2xs)] text-text-subtle">
              {summary}
            </span>
          )}
          {shape === 'decision' && (
            <span className="mt-0.5 text-[length:var(--text-2xs)] text-text-subtle">
              {String(d.config.field || 'branch')}
            </span>
          )}
        </div>

        {/* Left accent bar + icon chip for rectangular/parallelogram steps. */}
        {(shape === 'process' || shape === 'io') && (
          <div
            className="absolute left-3 top-1/2 flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-[var(--radius-7)]"
            style={{ background: `color-mix(in srgb, ${accent} 14%, transparent)`, color: accent }}
          >
            <Icon size={14} />
          </div>
        )}

        {/* A colored kind pill, echoing the reference designer. */}
        {shape === 'terminator' && (
          <span
            className="absolute right-2 top-1.5 rounded-full px-1.5 text-[length:var(--text-2xs)] font-semibold uppercase tracking-wide"
            style={{ color: accent, background: `color-mix(in srgb, ${accent} 14%, transparent)` }}
          >
            {isStart ? 'start' : isEnd ? 'end' : ''}
          </span>
        )}
      </div>

      {/* Status dot from a test run. */}
      {status && (
        <span
          className="absolute -right-1 -top-1 h-3 w-3 rounded-full border-2"
          style={{ background: status, borderColor: 'var(--color-surface)' }}
        />
      )}

      {/* Output handle(s) on the bottom. A decision has true/false; End has none. */}
      {def.shape === 'decision' ? (
        <>
          <Handle
            id="true"
            type="source"
            position={Position.Bottom}
            style={{ ...HANDLE_STYLE, left: '32%' }}
          />
          <Handle
            id="false"
            type="source"
            position={Position.Bottom}
            style={{ ...HANDLE_STYLE, left: '68%' }}
          />
          <span className="absolute -bottom-4 left-[24%] text-[length:var(--text-2xs)] text-[#35c07f]">
            true
          </span>
          <span className="absolute -bottom-4 left-[60%] text-[length:var(--text-2xs)] text-[var(--color-danger)]">
            false
          </span>
        </>
      ) : (
        !isEnd && <Handle type="source" position={Position.Bottom} style={HANDLE_STYLE} />
      )}
    </div>
  );
}

/** clip-path polygons for the non-rectangular flowchart shapes. */
const CLIP: Record<BlockShape, string | null> = {
  terminator: null,
  process: null,
  io: 'polygon(13% 0, 100% 0, 87% 100%, 0 100%)',
  decision: 'polygon(50% 0, 100% 50%, 50% 100%, 0 50%)',
};

export const FlowNode = memo(FlowNodeComponent);
