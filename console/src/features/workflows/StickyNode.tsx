import { memo, useEffect, useRef, useState } from 'react';
import { useReactFlow, type NodeProps } from '@xyflow/react';
import { ORANGE } from './nodeDefs';
import type { WfNodeData } from './WorkflowNode';

/**
 * Canvas-local annotation node. Not persisted in the save payload (filtered out
 * by `toSaveNodes` on `node.type === 'sticky'`) — matches the Flutter behaviour
 * where sticky notes live only in the editor session. Reuses the `WfNodeData`
 * shape (label doubles as the note text) so the parent node state stays a single
 * uniform array.
 */
function StickyNodeComponent({ id, data, selected }: NodeProps) {
  const nodeData = data as WfNodeData;
  const { setNodes } = useReactFlow();
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(nodeData.label ?? '');
  const ref = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    setText(nodeData.label ?? '');
  }, [nodeData.label]);

  useEffect(() => {
    if (editing) ref.current?.focus();
  }, [editing]);

  const commit = () => {
    setEditing(false);
    setNodes((ns) =>
      ns.map((n) => (n.id === id ? { ...n, data: { ...n.data, label: text } } : n)),
    );
  };

  return (
    <div
      onDoubleClick={() => setEditing(true)}
      className="rounded-[var(--radius-6)] p-2"
      style={{
        width: 200,
        minHeight: 100,
        background: `color-mix(in srgb, ${ORANGE} 18%, transparent)`,
        border: `1px solid color-mix(in srgb, ${ORANGE} ${selected ? 70 : 40}%, transparent)`,
      }}
    >
      {editing ? (
        <textarea
          ref={ref}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              e.stopPropagation();
              commit();
            }
          }}
          className="nodrag h-[84px] w-full resize-none bg-transparent text-[length:var(--text-body)] outline-none"
          style={{ color: `color-mix(in srgb, ${ORANGE} 85%, white)` }}
        />
      ) : (
        <div
          className="whitespace-pre-wrap break-words text-[length:var(--text-body)]"
          style={{ color: `color-mix(in srgb, ${ORANGE} 85%, white)` }}
        >
          {text || 'Note'}
        </div>
      )}
    </div>
  );
}

export const StickyNode = memo(StickyNodeComponent);
