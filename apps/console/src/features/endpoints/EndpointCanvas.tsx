import { useCallback } from 'react';
import {
  Background,
  BackgroundVariant,
  BaseEdge,
  type Connection,
  Controls,
  type EdgeChange,
  EdgeLabelRenderer,
  type EdgeProps,
  type EdgeTypes,
  getSmoothStepPath,
  type NodeChange,
  type OnBeforeDelete,
  ReactFlow,
  ReactFlowProvider,
  type NodeTypes,
  useReactFlow,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { FlowNode } from './FlowNode';
import type { EpEdge, EpNode } from './graph';

const nodeTypes: NodeTypes = { flow: FlowNode };

// A connector that shows a ✕ button at its midpoint to unlink it. The button
// appears on hover of the edge (and stays while hovered), so unlinking is
// discoverable without having to find and select a thin line.
function DeletableEdge(props: EdgeProps) {
  const { id, sourceX, sourceY, targetX, targetY, sourcePosition, targetPosition, markerEnd, style } = props;
  const { deleteElements } = useReactFlow();
  const [path, labelX, labelY] = getSmoothStepPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
  });
  return (
    <>
      <BaseEdge id={id} path={path} markerEnd={markerEnd} style={style} />
      <EdgeLabelRenderer>
        <button
          type="button"
          title="Unlink"
          className="ep-edge-x nodrag nopan"
          style={{ transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)` }}
          onClick={(e) => {
            e.stopPropagation();
            void deleteElements({ edges: [{ id }] });
          }}
        >
          ×
        </button>
      </EdgeLabelRenderer>
    </>
  );
}

const edgeTypes: EdgeTypes = { deletable: DeletableEdge };

const defaultEdgeOptions = {
  type: 'deletable',
  style: { stroke: 'color-mix(in srgb, var(--color-text-secondary) 55%, transparent)', strokeWidth: 1.5 },
};

/**
 * The endpoint graph as a vertical flowchart. Presentational: state lives in the
 * builder, so the node library and inspector share one source of truth.
 */
export function EndpointCanvas({
  nodes,
  edges,
  onNodesChange,
  onEdgesChange,
  onConnect,
  onNodeClick,
  onPaneClick,
  onBeforeDelete,
}: {
  nodes: EpNode[];
  edges: EpEdge[];
  onNodesChange: (changes: NodeChange<EpNode>[]) => void;
  onEdgesChange: (changes: EdgeChange<EpEdge>[]) => void;
  onConnect: (c: Connection) => void;
  onNodeClick: (id: string) => void;
  onPaneClick: () => void;
  onBeforeDelete: OnBeforeDelete<EpNode, EpEdge>;
}) {
  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: EpNode) => onNodeClick(node.id),
    [onNodeClick],
  );

  return (
    <div className="h-full w-full">
      <ReactFlowProvider>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          defaultEdgeOptions={defaultEdgeOptions}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onNodeClick={handleNodeClick}
          onPaneClick={onPaneClick}
          onBeforeDelete={onBeforeDelete}
          deleteKeyCode={['Delete', 'Backspace']}
          snapToGrid
          snapGrid={[10, 10]}
          minZoom={0.3}
          maxZoom={2}
          fitView
          fitViewOptions={{ padding: 0.3, maxZoom: 1 }}
          proOptions={{ hideAttribution: true }}
          colorMode="dark"
        >
          <Background variant={BackgroundVariant.Dots} gap={22} size={1} />
          <Controls showInteractive={false} />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
}
