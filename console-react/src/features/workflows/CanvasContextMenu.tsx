import { Ban, CheckSquare, Copy, Maximize2, Plus, Settings, Trash2 } from 'lucide-react';
import { RED } from './nodeDefs';

export interface ContextMenuState {
  x: number;
  y: number;
  /** Present when the menu was opened over a node. */
  nodeId?: string;
  /** True when the target node is a trigger (open-only actions). */
  isTrigger?: boolean;
}

/**
 * Right-click menu for the DAG canvas. Ports Flutter `_showContextMenu`:
 * over a node → Open settings / Duplicate / Toggle disable / Delete;
 * over empty canvas → Add node / Select all / Fit to view.
 */
export function CanvasContextMenu({
  menu,
  onClose,
  onOpen,
  onDuplicate,
  onToggleDisable,
  onDelete,
  onAddNode,
  onSelectAll,
  onFitView,
}: {
  menu: ContextMenuState;
  onClose: () => void;
  onOpen: (id: string) => void;
  onDuplicate: (id: string) => void;
  onToggleDisable: (id: string) => void;
  onDelete: (id: string) => void;
  onAddNode: () => void;
  onSelectAll: () => void;
  onFitView: () => void;
}) {
  const run = (fn: () => void) => {
    fn();
    onClose();
  };

  return (
    <>
      {/* Backdrop swallows the next click and closes the menu. */}
      <div className="fixed inset-0 z-40" onClick={onClose} onContextMenu={(e) => e.preventDefault()} />
      <div
        className="fixed z-50 min-w-[172px] overflow-hidden rounded-[var(--radius)] border border-border bg-surface py-1 shadow-lg"
        style={{ left: menu.x, top: menu.y }}
      >
        {menu.nodeId ? (
          <>
            <MenuItem icon={Settings} label="Open settings" onClick={() => run(() => onOpen(menu.nodeId!))} />
            {!menu.isTrigger && (
              <>
                <MenuItem icon={Copy} label="Duplicate" onClick={() => run(() => onDuplicate(menu.nodeId!))} />
                <MenuItem icon={Ban} label="Toggle disable" onClick={() => run(() => onToggleDisable(menu.nodeId!))} />
                <div className="my-1 h-px bg-border" />
                <MenuItem icon={Trash2} label="Delete" color={RED} onClick={() => run(() => onDelete(menu.nodeId!))} />
              </>
            )}
          </>
        ) : (
          <>
            <MenuItem icon={Plus} label="Add node" onClick={() => run(onAddNode)} />
            <MenuItem icon={CheckSquare} label="Select all" onClick={() => run(onSelectAll)} />
            <MenuItem icon={Maximize2} label="Fit to view" onClick={() => run(onFitView)} />
          </>
        )}
      </div>
    </>
  );
}

function MenuItem({
  icon: Icon,
  label,
  color,
  onClick,
}: {
  icon: typeof Settings;
  label: string;
  color?: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex w-full items-center gap-2.5 px-3 py-1.5 text-left text-[length:var(--text-body)] transition-colors hover:bg-fill"
      style={{ color: color ?? 'var(--color-text-secondary)' }}
    >
      <Icon size={15} />
      {label}
    </button>
  );
}
