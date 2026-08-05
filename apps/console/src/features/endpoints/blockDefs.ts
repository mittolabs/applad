import {
  ArrowRightLeft,
  Code2,
  Database,
  Filter,
  GitBranch,
  Globe,
  LogIn,
  type LucideIcon,
  Mail,
  Send,
  Wand2,
} from 'lucide-react';

/*
 * Block (node) definitions for the endpoint builder. An endpoint graph is the
 * same nodes+edges shape a workflow uses; the builder edits it as an ordered
 * list of blocks and derives a linear edge chain, with condition blocks naming
 * their own branch targets. Kept deliberately small: handler, data, transform,
 * condition, response. Logic/integration parity with workflows can grow later.
 */

export type BlockType =
  | 'endpoint_handler'
  | 'endpoint_data'
  | 'set_variable'
  | 'edit_fields'
  | 'if_condition'
  | 'filter'
  | 'http_request'
  | 'applad_functions'
  | 'applad_messaging'
  | 'endpoint_response';

/** Flowchart symbol a node is drawn as, so the graph reads by shape:
 *  terminator = start/end (stadium), process = a step (rectangle),
 *  decision = a branch (diamond), io = data in/out (parallelogram). */
export type BlockShape = 'terminator' | 'process' | 'decision' | 'io';

export interface Block {
  id: string;
  type: BlockType;
  label: string;
  config: Record<string, unknown>;
  /** Canvas coordinates, persisted so a saved graph reopens laid out. */
  position?: { x: number; y: number };
}

export interface BlockDef {
  type: BlockType;
  label: string;
  blurb: string;
  icon: LucideIcon;
  accent: string;
  shape: BlockShape;
  category: string;
  /** A fresh config for a newly added block. */
  defaults: () => Record<string, unknown>;
  /** Handler is the fixed entry point: one per endpoint, not added or removed. */
  fixed?: boolean;
}

export const BLOCK_DEFS: Record<BlockType, BlockDef> = {
  endpoint_handler: {
    type: 'endpoint_handler',
    label: 'Request',
    blurb: 'Where the request enters. Reads method, path, query, headers and body.',
    icon: LogIn,
    accent: '#6f8cff',
    shape: 'terminator',
    category: 'Start & end',
    defaults: () => ({}),
    fixed: true,
  },
  endpoint_data: {
    type: 'endpoint_data',
    label: 'Data',
    blurb: 'Read or write a database table. Runs as the caller unless you turn off Apply rules.',
    icon: Database,
    accent: '#7c5cff',
    shape: 'process',
    category: 'Steps',
    defaults: () => ({
      action: 'create',
      databaseId: '',
      tableId: '',
      rowId: '',
      data: {},
      applyRules: true,
    }),
  },
  set_variable: {
    type: 'set_variable',
    label: 'Set variable',
    blurb: 'Compute a value from the request or an earlier block and name it.',
    icon: ArrowRightLeft,
    accent: '#35c07f',
    shape: 'io',
    category: 'Steps',
    defaults: () => ({ key: '', value: '' }),
  },
  edit_fields: {
    type: 'edit_fields',
    label: 'Transform',
    blurb: 'Set one or more named values from templates for later blocks to use.',
    icon: Wand2,
    accent: '#35c07f',
    shape: 'io',
    category: 'Steps',
    defaults: () => ({ fields: {} }),
  },
  if_condition: {
    type: 'if_condition',
    label: 'Condition',
    blurb: 'Branch on a value. Each branch names the block to run (or skip).',
    icon: GitBranch,
    accent: '#f5a623',
    shape: 'decision',
    category: 'Logic',
    defaults: () => ({ field: '', operator: 'eq', value: '', trueBranch: '', falseBranch: '' }),
  },
  filter: {
    type: 'filter',
    label: 'Filter',
    blurb: 'Stop the run unless a condition holds. Everything downstream is skipped on no match.',
    icon: Filter,
    accent: '#f5a623',
    shape: 'io',
    category: 'Logic',
    defaults: () => ({ field: '', operator: 'eq', value: '' }),
  },
  http_request: {
    type: 'http_request',
    label: 'HTTP request',
    blurb: 'Call an external API. SSRF-guarded, so it cannot reach internal services.',
    icon: Globe,
    accent: '#6f8cff',
    shape: 'process',
    category: 'Integrations',
    defaults: () => ({ method: 'GET', url: '', body: '', headers: {} }),
  },
  applad_functions: {
    type: 'applad_functions',
    label: 'Call function',
    blurb: 'Invoke a deployed function and use its result. Needs a project API key.',
    icon: Code2,
    accent: '#7c5cff',
    shape: 'process',
    category: 'Integrations',
    defaults: () => ({ action: 'invoke', targetId: '', data: {}, apiKey: '' }),
  },
  applad_messaging: {
    type: 'applad_messaging',
    label: 'Send email',
    blurb: 'Send an email through your messaging config. Needs a project API key.',
    icon: Mail,
    accent: '#7c5cff',
    shape: 'process',
    category: 'Integrations',
    defaults: () => ({ action: 'send_email', to: '', subject: '', body: '', apiKey: '' }),
  },
  endpoint_response: {
    type: 'endpoint_response',
    label: 'Response',
    blurb: 'Produce the HTTP response and end the run. The first response reached wins.',
    icon: Send,
    accent: '#35c07f',
    shape: 'terminator',
    category: 'Start & end',
    defaults: () => ({ status: 200, mode: 'json', body: '', bodyField: '' }),
  },
};

/** Blocks a user may add (handler is fixed, so excluded). */
export const ADDABLE_BLOCKS: BlockDef[] = Object.values(BLOCK_DEFS).filter((d) => !d.fixed);

export const HTTP_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'] as const;

export const AUTH_OPTIONS: { value: string; label: string }[] = [
  { value: 'public', label: 'Public (anyone)' },
  { value: 'session', label: 'Session (signed-in user)' },
  { value: 'api_key', label: 'API key' },
  { value: 'either', label: 'Session or API key' },
];

export const DATA_ACTIONS: { value: string; label: string }[] = [
  { value: 'create', label: 'Create row' },
  { value: 'get', label: 'Get row' },
  { value: 'list', label: 'List / query rows' },
  { value: 'update', label: 'Update row' },
  { value: 'delete', label: 'Delete row' },
];

export const CONDITION_OPERATORS: { value: string; label: string }[] = [
  { value: 'eq', label: 'equals' },
  { value: 'neq', label: 'does not equal' },
  { value: 'contains', label: 'contains' },
  { value: 'starts_with', label: 'starts with' },
  { value: 'empty', label: 'is empty' },
  { value: 'not_empty', label: 'is not empty' },
];

export const RESPONSE_MODES: { value: string; label: string }[] = [
  { value: 'json', label: 'JSON' },
  { value: 'text', label: 'Text' },
  { value: 'html', label: 'HTML' },
  { value: 'error', label: 'Error (JSON)' },
];

/** A one-line summary of a block for the list card. */
export function blockSummary(block: Block): string {
  const c = block.config;
  switch (block.type) {
    case 'endpoint_handler':
      return 'request.body, request.query, request.params';
    case 'endpoint_data': {
      const action = String(c.action ?? 'create');
      const table = String(c.tableId ?? '') || 'a table';
      const scoped = c.applyRules === false ? ' as service' : '';
      return `${action} in ${table}${scoped}`;
    }
    case 'set_variable':
      return c.key ? `${c.key} = ${truncate(String(c.value ?? ''))}` : 'name a value';
    case 'edit_fields': {
      const n = c.fields && typeof c.fields === 'object' ? Object.keys(c.fields as object).length : 0;
      return n ? `${n} field${n === 1 ? '' : 's'}` : 'set fields';
    }
    case 'if_condition':
      return c.field ? `${c.field} ${c.operator} ${truncate(String(c.value ?? ''))}` : 'branch on a value';
    case 'filter':
      return c.field ? `keep if ${c.field} ${c.operator} ${truncate(String(c.value ?? ''))}` : 'gate the run';
    case 'http_request':
      return `${String(c.method ?? 'GET')} ${truncate(String(c.url ?? '')) || 'a URL'}`;
    case 'applad_functions':
      return `invoke ${String(c.targetId ?? '') || 'a function'}`;
    case 'applad_messaging':
      return `email ${truncate(String(c.to ?? '')) || 'a recipient'}`;
    case 'endpoint_response':
      return `${c.status ?? 200} · ${String(c.mode ?? 'json')}`;
    default:
      return '';
  }
}

function truncate(s: string, n = 24): string {
  return s.length > n ? `${s.slice(0, n)}…` : s;
}
