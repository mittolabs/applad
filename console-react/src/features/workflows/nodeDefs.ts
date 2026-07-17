import {
  ArrowDownToLine,
  ArrowRight,
  ArrowUpDown,
  BarChart3,
  BookOpen,
  Bot,
  Box,
  Braces,
  Calendar,
  Cloud,
  Code,
  Code2,
  CopyMinus,
  CreditCard,
  Database,
  FileJson,
  FileSearch,
  FileText,
  Filter,
  FolderClosed,
  GitBranch,
  GitCompare,
  Globe,
  HardDrive,
  Hash,
  Lock,
  type LucideIcon,
  Mail,
  Merge,
  MessageCircle,
  MessageSquare,
  Octagon,
  Pencil,
  Phone,
  Play,
  Plug,
  Repeat,
  Replace,
  Send,
  Sheet,
  Shield,
  Sigma,
  Sparkles,
  Split,
  SplitSquareVertical,
  Ticket,
  Timer,
  Users,
  Variable,
  Webhook,
  Workflow,
} from 'lucide-react';

// ── Shared color constants (match the Flutter source) ──
export const ACCENT = '#3472A4';
export const GREEN = '#10B981';
export const ORANGE = '#F59E0B';
export const RED = '#EF4444';

export interface NodeDef {
  type: string;
  label: string;
  description: string;
  icon: LucideIcon;
  color: string;
  category: string;
  outputs: number;
  inputs: number;
  outputLabels?: string[];
}

function def(
  type: string,
  label: string,
  description: string,
  icon: LucideIcon,
  color: string,
  opts: Partial<Pick<NodeDef, 'category' | 'outputs' | 'inputs' | 'outputLabels'>> = {},
): NodeDef {
  return {
    type,
    label,
    description,
    icon,
    color,
    category: opts.category ?? 'Core',
    outputs: opts.outputs ?? 1,
    inputs: opts.inputs ?? 1,
    outputLabels: opts.outputLabels,
  };
}

export const TRIGGER_DEF: NodeDef = def(
  'trigger',
  'Trigger',
  'Workflow start point',
  Play,
  GREEN,
  { category: 'Triggers', inputs: 0 },
);

export interface NodeCategory {
  name: string;
  description: string;
  icon: LucideIcon;
}

export const CATEGORIES: NodeCategory[] = [
  { name: 'Applad', description: 'Auth, Databases, Storage, Functions, Messaging', icon: Box },
  { name: 'AI', description: 'LLM agents, summarize, transform', icon: Sparkles },
  { name: 'Integrations', description: 'Connect to external services', icon: Plug },
  { name: 'Data transformation', description: 'Manipulate, filter or convert data', icon: Pencil },
  { name: 'Flow', description: 'Branch, merge or loop the flow', icon: GitBranch },
  { name: 'Core', description: 'HTTP requests, code, webhooks', icon: Box },
  { name: 'Triggers', description: 'Start your workflow', icon: Play },
];

export const ALL_NODE_DEFS: NodeDef[] = [
  // ── Flow ──
  def('if_condition', 'IF', 'Branch on condition', GitBranch, '#F97316', {
    category: 'Flow',
    outputs: 2,
    outputLabels: ['true', 'false'],
  }),
  def('switch', 'Switch', 'Route to multiple branches', Split, '#F97316', {
    category: 'Flow',
    outputs: 3,
    outputLabels: ['1', '2', 'default'],
  }),
  def('merge', 'Merge', 'Combine data from branches', Merge, '#F97316', {
    category: 'Flow',
    inputs: 2,
  }),
  def('loop', 'Loop', 'Iterate over items', Repeat, '#F97316', { category: 'Flow' }),
  def('wait', 'Wait', 'Pause execution', Timer, '#64748B', { category: 'Flow' }),
  def('no_operation', 'No Operation', 'Pass-through', ArrowRight, '#64748B', { category: 'Flow' }),
  def('execute_sub_workflow', 'Sub-Workflow', 'Call another workflow', Workflow, '#F97316', {
    category: 'Flow',
  }),
  def('filter', 'Filter', 'Keep items matching a condition', Filter, '#F97316', {
    category: 'Flow',
  }),

  // ── Core ──
  def('http_request', 'HTTP Request', 'Make an API call', Globe, '#8B5CF6', { category: 'Core' }),
  def('code', 'Code', 'Run template expression', Code2, '#06B6D4', { category: 'Core' }),
  def('javascript', 'JavaScript', 'Run JS-like code with helpers', Braces, '#F7DF1E', {
    category: 'Core',
  }),
  def('send_email', 'Send Email', 'Send via SMTP', Mail, '#EC4899', { category: 'Core' }),
  def('set_variable', 'Set Variable', 'Set a context value', Variable, '#F59E0B', {
    category: 'Core',
  }),
  def('delay', 'Delay', 'Wait before continuing', Timer, '#64748B', { category: 'Core' }),

  // ── Data transformation ──
  def('edit_fields', 'Edit Fields', 'Set multiple fields at once', Pencil, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('aggregate', 'Aggregate', 'Count, sum, avg, min, max', Sigma, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('summarize', 'Summarize', 'Group and count items', BarChart3, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('limit', 'Limit', 'Restrict number of items', ArrowDownToLine, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('split_out', 'Split Out', 'Split array into items', SplitSquareVertical, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('remove_duplicates', 'Remove Duplicates', 'Deduplicate items', CopyMinus, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('date_time', 'Date & Time', 'Format or manipulate dates', Calendar, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('convert_to_json', 'Convert to JSON', 'Serialize to JSON string', FileJson, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('extract_from_json', 'Extract from JSON', 'Parse JSON field', FileSearch, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('html_parse', 'HTML', 'Work with HTML content', Code, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('crypto', 'Crypto', 'Hash or encode data', Lock, '#06B6D4', {
    category: 'Data transformation',
  }),

  // ── Error handling (grouped under Flow) ──
  def('try_catch', 'Try / Catch', 'Handle errors gracefully', Shield, '#EF4444', {
    category: 'Flow',
    outputs: 2,
    outputLabels: ['success', 'error'],
  }),
  def('stop_and_error', 'Stop and Error', 'Fail workflow with message', Octagon, '#EF4444', {
    category: 'Flow',
  }),

  // ── AI ──
  def('ai_transform', 'AI Transform', 'Transform data with LLM', Sparkles, '#8B5CF6', {
    category: 'AI',
  }),
  def('ai_agent', 'AI Agent', 'Multi-step LLM agent with tools', Bot, '#8B5CF6', {
    category: 'AI',
  }),
  def('ai_summarize', 'AI Summarize', 'Summarize text content', FileText, '#8B5CF6', {
    category: 'AI',
  }),

  // ── Integrations ──
  def('slack', 'Slack', 'Send Slack messages', Hash, '#4A154B', { category: 'Integrations' }),
  def('discord', 'Discord', 'Send Discord messages', MessageCircle, '#5865F2', {
    category: 'Integrations',
  }),
  def('telegram', 'Telegram', 'Send Telegram messages', Send, '#26A5E4', {
    category: 'Integrations',
  }),
  def('github', 'GitHub', 'Create issues, PRs', GitBranch, '#E6EDF3', { category: 'Integrations' }),
  def('google_sheets', 'Google Sheets', 'Read/write spreadsheets', Sheet, '#34A853', {
    category: 'Integrations',
  }),
  def('notion', 'Notion', 'Query databases, create pages', BookOpen, '#FFFFFF', {
    category: 'Integrations',
  }),
  def('stripe', 'Stripe', 'Charges, customers, payments', CreditCard, '#635BFF', {
    category: 'Integrations',
  }),
  def('twilio_sms', 'Twilio SMS', 'Send SMS messages', Phone, '#F22F46', {
    category: 'Integrations',
  }),
  def('postgres_query', 'PostgreSQL', 'Run SQL queries', Database, '#336791', {
    category: 'Integrations',
  }),
  def('mysql_query', 'MySQL', 'Run SQL queries', Database, '#4479A1', {
    category: 'Integrations',
  }),
  def('redis_command', 'Redis', 'Run Redis commands', Database, '#DC382D', {
    category: 'Integrations',
  }),
  def('s3', 'AWS S3', 'Get, put, list objects', Cloud, '#FF9900', { category: 'Integrations' }),
  def('sendgrid', 'SendGrid', 'Send transactional emails', Mail, '#1A82E2', {
    category: 'Integrations',
  }),
  def('jira', 'Jira', 'Create and manage issues', Ticket, '#0052CC', {
    category: 'Integrations',
  }),

  // ── Applad-native ──
  def('applad_auth', 'Applad Auth', 'Manage users and sessions', Users, ACCENT, {
    category: 'Applad',
  }),
  def('applad_database', 'Applad Database', 'CRUD documents in collections', Database, ACCENT, {
    category: 'Applad',
  }),
  def('applad_storage', 'Applad Storage', 'Manage files in buckets', FolderClosed, ACCENT, {
    category: 'Applad',
  }),
  def('applad_functions', 'Applad Functions', 'Invoke serverless targets', Sparkles, ACCENT, {
    category: 'Applad',
  }),
  def('applad_messaging', 'Applad Messaging', 'Send email, SMS, push', MessageSquare, ACCENT, {
    category: 'Applad',
  }),

  // ── Additional data transformation ──
  def('sort', 'Sort', 'Sort items by field', ArrowUpDown, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('rename_keys', 'Rename Keys', 'Rename fields on items', Replace, '#06B6D4', {
    category: 'Data transformation',
  }),
  def('compare_datasets', 'Compare Datasets', 'Find added/removed/unchanged', GitCompare, '#06B6D4', {
    category: 'Data transformation',
  }),

  // ── Triggers ──
  def('trigger_manual', 'Manual', 'Run workflow manually from dashboard or API', Play, GREEN, {
    category: 'Triggers',
    inputs: 0,
  }),
  def('trigger_webhook', 'Webhook', 'Trigger via an incoming HTTP request', Webhook, GREEN, {
    category: 'Triggers',
    inputs: 0,
  }),
  def('trigger_schedule', 'Schedule', 'Run on a cron schedule', Timer, GREEN, {
    category: 'Triggers',
    inputs: 0,
  }),
  def('trigger_database', 'Database Event', 'Fire on row insert, update or delete', Database, GREEN, {
    category: 'Triggers',
    inputs: 0,
  }),
  def('trigger_auth', 'Auth Event', 'Fire on user signup, login or deletion', Users, GREEN, {
    category: 'Triggers',
    inputs: 0,
  }),
  def('trigger_storage', 'Storage Event', 'Fire on file upload or deletion', HardDrive, GREEN, {
    category: 'Triggers',
    inputs: 0,
  }),
  def('trigger_messaging', 'Messaging Event', 'Fire when a message is sent', Mail, GREEN, {
    category: 'Triggers',
    inputs: 0,
  }),
];

const BY_TYPE = new Map<string, NodeDef>(ALL_NODE_DEFS.map((d) => [d.type, d]));

export function defFor(type: string | undefined): NodeDef {
  const t = type ?? '';
  if (t === 'trigger') return TRIGGER_DEF;
  const found = BY_TYPE.get(t);
  if (found) return found;
  if (t.startsWith('trigger_')) return TRIGGER_DEF;
  return BY_TYPE.get('http_request')!;
}

// ── Type-specific configuration fields (ports Flutter `_typeFields`) ──

export interface FieldSpec {
  key: string;
  label: string;
  hint?: string;
  lines?: number;
  expr?: boolean;
}

export const TYPE_FIELDS: Record<string, FieldSpec[]> = {
  http_request: [
    { key: 'url', label: 'URL', hint: 'https://api.example.com', expr: true },
    { key: 'method', label: 'Method', hint: 'GET' },
    { key: 'body', label: 'Body (JSON)', hint: '{}', lines: 3, expr: true },
  ],
  send_email: [
    { key: 'to', label: 'To', expr: true },
    { key: 'subject', label: 'Subject', expr: true },
    { key: 'body', label: 'Body', lines: 3, expr: true },
  ],
  set_variable: [
    { key: 'key', label: 'Key' },
    { key: 'value', label: 'Value', expr: true },
  ],
  code: [
    { key: 'expression', label: 'Expression', hint: '{{.trigger.name}}', lines: 4, expr: true },
  ],
  if_condition: [
    { key: 'field', label: 'Field', hint: 'trigger.status', expr: true },
    { key: 'operator', label: 'Operator', hint: 'eq, neq, contains' },
    { key: 'value', label: 'Value', expr: true },
  ],
  delay: [{ key: 'durationMs', label: 'Duration (ms)', hint: '1000' }],
  switch: [
    { key: 'field', label: 'Field', hint: 'trigger.status' },
    { key: 'case1', label: 'Case 1 value' },
    { key: 'case2', label: 'Case 2 value' },
    { key: 'defaultTarget', label: 'Default target node ID' },
  ],
  merge: [],
  loop: [
    { key: 'items', label: 'Items field', hint: 'trigger.items' },
    { key: 'loopVariable', label: 'Loop variable name', hint: 'item' },
  ],
  wait: [{ key: 'seconds', label: 'Wait seconds', hint: '5' }],
  no_operation: [],
  execute_sub_workflow: [{ key: 'workflowId', label: 'Sub-workflow ID' }],
  filter: [
    { key: 'field', label: 'Field', hint: 'trigger.status' },
    { key: 'operator', label: 'Operator', hint: 'eq, neq, contains' },
    { key: 'value', label: 'Value' },
  ],
  edit_fields: [
    { key: 'fields', label: 'Fields (JSON)', hint: '{"key": "value"}', lines: 4, expr: true },
  ],
  aggregate: [
    { key: 'field', label: 'Array field', hint: 'trigger.items' },
    { key: 'operation', label: 'Operation', hint: 'count, sum, min, max, avg' },
  ],
  summarize: [
    { key: 'field', label: 'Field', hint: 'trigger.items' },
    { key: 'groupBy', label: 'Group by', hint: 'status' },
  ],
  limit: [{ key: 'count', label: 'Max items', hint: '10' }],
  split_out: [{ key: 'field', label: 'Array field', hint: 'trigger.items' }],
  remove_duplicates: [{ key: 'field', label: 'Dedup by field', hint: 'email' }],
  date_time: [
    { key: 'operation', label: 'Operation', hint: 'now, format, parse, add' },
    { key: 'format', label: 'Format', hint: '2006-01-02T15:04:05Z' },
    { key: 'value', label: 'Value' },
    { key: 'duration', label: 'Duration', hint: '24h' },
  ],
  convert_to_json: [{ key: 'data', label: 'Data field', hint: 'trigger.payload' }],
  extract_from_json: [
    { key: 'json', label: 'JSON string field' },
    { key: 'path', label: 'JSON path', hint: 'data.name' },
  ],
  html_parse: [
    { key: 'html', label: 'HTML content' },
    { key: 'selector', label: 'Selector' },
  ],
  crypto: [
    { key: 'operation', label: 'Operation', hint: 'md5, sha256, base64_encode, base64_decode' },
    { key: 'input', label: 'Input' },
  ],
  slack: [
    { key: 'webhookUrl', label: 'Webhook URL' },
    { key: 'message', label: 'Message', lines: 3 },
  ],
  discord: [
    { key: 'webhookUrl', label: 'Webhook URL' },
    { key: 'message', label: 'Message', lines: 3 },
  ],
  telegram: [
    { key: 'botToken', label: 'Bot Token' },
    { key: 'chatId', label: 'Chat ID' },
    { key: 'message', label: 'Message', lines: 3 },
  ],
  github: [
    { key: 'token', label: 'Personal Access Token' },
    { key: 'owner', label: 'Owner' },
    { key: 'repo', label: 'Repository' },
    { key: 'action', label: 'Action', hint: 'create_issue' },
    { key: 'title', label: 'Title' },
    { key: 'body', label: 'Body', lines: 3 },
  ],
  javascript: [
    { key: 'code', label: 'Code', hint: '{{json_stringify .trigger}}', lines: 8, expr: true },
  ],
  try_catch: [
    { key: 'tryNodes', label: 'Try node IDs (comma-separated)' },
    { key: 'catchTarget', label: 'Catch target node ID' },
  ],
  stop_and_error: [
    { key: 'message', label: 'Error message', hint: 'Something went wrong' },
  ],
  ai_transform: [
    { key: 'model', label: 'Model', hint: 'claude-sonnet-4-20250514' },
    { key: 'prompt', label: 'Prompt', hint: 'Transform this data...', lines: 4, expr: true },
    { key: 'userMessage', label: 'User message', hint: '{{.trigger.body.message}}', expr: true },
    { key: 'apiKey', label: 'API Key' },
  ],
  ai_agent: [
    { key: 'model', label: 'Model', hint: 'claude-sonnet-4-20250514' },
    { key: 'systemPrompt', label: 'System prompt', lines: 3 },
    { key: 'userMessage', label: 'User message', hint: '{{.trigger.body.message}}', expr: true },
    { key: 'tools', label: 'Tools (JSON)', hint: '[]', lines: 3 },
    { key: 'apiKey', label: 'API Key' },
    { key: 'maxSteps', label: 'Max steps', hint: '5' },
  ],
  ai_summarize: [
    { key: 'model', label: 'Model', hint: 'claude-sonnet-4-20250514' },
    { key: 'text', label: 'Text to summarize', lines: 3 },
    { key: 'maxLength', label: 'Max length', hint: '200' },
    { key: 'apiKey', label: 'API Key' },
  ],
  google_sheets: [
    { key: 'accessToken', label: 'Access Token' },
    { key: 'spreadsheetId', label: 'Spreadsheet ID' },
    { key: 'range', label: 'Range', hint: 'Sheet1!A1:D10' },
    { key: 'action', label: 'Action', hint: 'read or append' },
    { key: 'values', label: 'Values (JSON)', hint: '[[]]', lines: 2 },
  ],
  notion: [
    { key: 'apiKey', label: 'Integration Token' },
    { key: 'action', label: 'Action', hint: 'query_database or create_page' },
    { key: 'databaseId', label: 'Database ID' },
    { key: 'properties', label: 'Properties (JSON)', hint: '{}', lines: 3 },
  ],
  stripe: [
    { key: 'apiKey', label: 'Secret Key' },
    { key: 'action', label: 'Action', hint: 'create_charge' },
    { key: 'amount', label: 'Amount (cents)' },
    { key: 'currency', label: 'Currency', hint: 'usd' },
    { key: 'email', label: 'Customer email' },
  ],
  twilio_sms: [
    { key: 'accountSid', label: 'Account SID' },
    { key: 'authToken', label: 'Auth Token' },
    { key: 'from', label: 'From number' },
    { key: 'to', label: 'To number' },
    { key: 'body', label: 'Message', lines: 2 },
  ],
  postgres_query: [
    { key: 'connectionUrl', label: 'Connection URL' },
    { key: 'query', label: 'SQL Query', lines: 4 },
  ],
  mysql_query: [
    { key: 'connectionUrl', label: 'Connection URL' },
    { key: 'query', label: 'SQL Query', lines: 4 },
  ],
  redis_command: [
    { key: 'connectionUrl', label: 'Connection URL' },
    { key: 'command', label: 'Command', hint: 'GET mykey' },
  ],
  s3: [
    { key: 'accessKeyId', label: 'Access Key ID' },
    { key: 'secretAccessKey', label: 'Secret Access Key' },
    { key: 'region', label: 'Region', hint: 'us-east-1' },
    { key: 'bucket', label: 'Bucket' },
    { key: 'key', label: 'Object Key' },
    { key: 'action', label: 'Action', hint: 'get, put, or list' },
  ],
  sendgrid: [
    { key: 'apiKey', label: 'API Key' },
    { key: 'to', label: 'To' },
    { key: 'from', label: 'From' },
    { key: 'subject', label: 'Subject' },
    { key: 'body', label: 'Body', lines: 3 },
  ],
  jira: [
    { key: 'domain', label: 'Domain', hint: 'yourcompany' },
    { key: 'email', label: 'Email' },
    { key: 'apiToken', label: 'API Token' },
    { key: 'action', label: 'Action', hint: 'create_issue' },
    { key: 'projectKey', label: 'Project Key' },
    { key: 'summary', label: 'Summary' },
    { key: 'description', label: 'Description', lines: 3 },
  ],
  applad_auth: [
    {
      key: 'action',
      label: 'Action',
      hint: 'create_user, get_user, list_users, update_user, delete_user',
    },
    { key: 'email', label: 'Email', expr: true },
    { key: 'password', label: 'Password' },
    { key: 'name', label: 'Name', expr: true },
    { key: 'userId', label: 'User ID', expr: true },
  ],
  applad_database: [
    {
      key: 'action',
      label: 'Action',
      hint: 'create_document, get_document, list_documents, update_document, delete_document',
    },
    { key: 'databaseId', label: 'Database ID' },
    { key: 'collectionId', label: 'Collection ID' },
    { key: 'documentId', label: 'Document ID', expr: true },
    { key: 'data', label: 'Data (JSON)', hint: '{}', lines: 4, expr: true },
  ],
  applad_storage: [
    { key: 'action', label: 'Action', hint: 'list_files, get_file, delete_file' },
    { key: 'bucketId', label: 'Bucket ID' },
    { key: 'fileId', label: 'File ID' },
  ],
  applad_functions: [
    { key: 'action', label: 'Action', hint: 'invoke, list_executions' },
    { key: 'targetId', label: 'Target ID' },
    { key: 'data', label: 'Request Data (JSON)', hint: '{}', lines: 3 },
  ],
  applad_messaging: [
    { key: 'action', label: 'Action', hint: 'send_email, send_sms, send_push' },
    { key: 'to', label: 'To', expr: true },
    { key: 'subject', label: 'Subject', expr: true },
    { key: 'body', label: 'Body', lines: 3, expr: true },
    { key: 'title', label: 'Title (push only)', expr: true },
  ],
  sort: [
    { key: 'items', label: 'Items field', hint: 'trigger.items' },
    { key: 'field', label: 'Sort by field' },
    { key: 'order', label: 'Order', hint: 'asc or desc' },
  ],
  rename_keys: [
    { key: 'mapping', label: 'Mapping (JSON)', hint: '{"oldKey": "newKey"}', lines: 3 },
  ],
  compare_datasets: [
    { key: 'input1', label: 'Input 1 field', hint: 'trigger.oldData' },
    { key: 'input2', label: 'Input 2 field', hint: 'trigger.newData' },
    { key: 'keyField', label: 'Key field', hint: 'id' },
  ],
};

// ── Palette category groupings (ports the drill-down sections) ──

export interface PaletteSection {
  title: string;
  types: string[];
}

export const PALETTE_SECTIONS: Record<string, PaletteSection[]> = {
  'Data transformation': [
    { title: 'Popular', types: ['edit_fields', 'code', 'date_time', 'aggregate'] },
    { title: 'Add or remove items', types: ['filter', 'limit', 'remove_duplicates', 'split_out'] },
    { title: 'Combine items', types: ['aggregate', 'merge', 'summarize'] },
    { title: 'Convert data', types: ['convert_to_json', 'extract_from_json', 'crypto', 'html_parse'] },
  ],
  Flow: [
    { title: 'Popular', types: ['if_condition', 'switch', 'merge', 'loop'] },
    { title: 'Error handling', types: ['try_catch', 'stop_and_error'] },
    { title: 'Other', types: ['wait', 'filter', 'no_operation', 'execute_sub_workflow'] },
  ],
  Integrations: [
    { title: 'Communication', types: ['slack', 'discord', 'telegram', 'sendgrid', 'twilio_sms'] },
    { title: 'Developer tools', types: ['github', 'jira', 'notion'] },
    { title: 'Databases', types: ['postgres_query', 'mysql_query', 'redis_command'] },
    { title: 'Cloud & payments', types: ['s3', 'stripe', 'google_sheets'] },
  ],
  AI: [
    { title: 'Transform & generate', types: ['ai_transform', 'ai_summarize'] },
    { title: 'Agents', types: ['ai_agent'] },
  ],
  Triggers: [
    { title: 'Scheduling', types: ['trigger_manual', 'trigger_schedule'] },
    { title: 'HTTP', types: ['trigger_webhook'] },
    {
      title: 'Applad events',
      types: ['trigger_database', 'trigger_auth', 'trigger_storage', 'trigger_messaging'],
    },
  ],
};
