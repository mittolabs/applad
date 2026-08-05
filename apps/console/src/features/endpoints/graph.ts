import type { Edge, Node } from '@xyflow/react';
import { BLOCK_DEFS, type Block, type BlockType } from './blockDefs';
import { type FlowNodeData, NODE_H } from './FlowNode';

export type EpNode = Node<FlowNodeData>;
export type EpEdge = Edge;

interface BackendEdge {
  id?: string;
  source: string;
  target: string;
  condition?: string;
}

/** Backend graph → React Flow nodes/edges. Missing positions get a vertical
 * auto-layout so a graph authored before positions existed still opens tidy. */
export function toFlow(rawNodes: unknown, rawEdges: unknown): { nodes: EpNode[]; edges: EpEdge[] } {
  const nodes = Array.isArray(rawNodes) ? (rawNodes as Block[]) : [];
  const edges = Array.isArray(rawEdges) ? (rawEdges as BackendEdge[]) : [];

  let y = 60;
  const rfNodes: EpNode[] = nodes.map((n) => {
    const def = BLOCK_DEFS[n.type];
    const pos =
      n.position && typeof n.position.x === 'number'
        ? { x: n.position.x, y: n.position.y }
        : { x: 300, y: y };
    y = pos.y + (def ? NODE_H[def.shape] : 80) + 70;
    return {
      id: n.id,
      type: 'flow',
      position: pos,
      data: {
        blockType: n.type,
        label: n.label || def?.label || n.type,
        config: (n.config as Record<string, unknown>) ?? {},
      },
    };
  });

  const rfEdges: EpEdge[] = edges.map((e, i) => ({
    id: e.id || `e${i + 1}`,
    type: 'deletable',
    source: e.source,
    target: e.target,
    sourceHandle: e.condition === 'true' || e.condition === 'false' ? e.condition : undefined,
  }));

  return { nodes: rfNodes, edges: rfEdges };
}

/** React Flow nodes/edges → backend graph. Persists positions and, for a
 * condition, resolves its true/false output edges into the config branch
 * targets the executor reads. */
export function fromFlow(nodes: EpNode[], edges: EpEdge[]): { nodes: Block[]; edges: BackendEdge[] } {
  const blocks: Block[] = nodes.map((n) => ({
    id: n.id,
    type: n.data.blockType,
    label: n.data.label,
    config: { ...n.data.config },
    position: { x: Math.round(n.position.x), y: Math.round(n.position.y) },
  }));

  for (const b of blocks) {
    if (b.type === 'if_condition') {
      const t = edges.find((e) => e.source === b.id && e.sourceHandle === 'true');
      const f = edges.find((e) => e.source === b.id && e.sourceHandle === 'false');
      b.config = { ...b.config, trueBranch: t?.target ?? '', falseBranch: f?.target ?? '' };
    }
  }

  const backendEdges: BackendEdge[] = edges.map((e, i) => ({
    id: e.id || `e${i + 1}`,
    source: e.source,
    target: e.target,
    condition: (e.sourceHandle as string) ?? '',
  }));

  return { nodes: blocks, edges: backendEdges };
}

/** A fresh React Flow node for a newly added block, placed below the lowest
 * node so it lands in view rather than on top of an existing one. */
export function newFlowNode(type: BlockType, id: string, nodes: EpNode[]): EpNode {
  const def = BLOCK_DEFS[type];
  const lowest = nodes.reduce((max, n) => Math.max(max, n.position.y), 0);
  const anchorX = nodes[0]?.position.x ?? 300;
  return {
    id,
    type: 'flow',
    position: { x: anchorX, y: nodes.length ? lowest + 150 : 60 },
    data: { blockType: type, label: def.label, config: def.defaults() },
  };
}
