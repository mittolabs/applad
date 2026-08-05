import { useState } from 'react';
import { Switch } from '@/components/ui/switch';
import { FormField, SelectField, TextAreaField, TextField } from '@/components/form-dialog';
import {
  BLOCK_DEFS,
  type Block,
  CONDITION_OPERATORS,
  DATA_ACTIONS,
  HTTP_METHODS,
  RESPONSE_MODES,
} from './blockDefs';

/** The right-hand config panel for the selected block. */
export function Inspector({
  block,
  onChange,
}: {
  block: Block | null;
  onChange: (patch: Record<string, unknown>) => void;
}) {
  if (!block) {
    return (
      <div className="flex h-full items-center justify-center p-8 text-center text-[length:var(--text-body)] text-text-muted">
        Select a block to configure it, or add one from the left.
      </div>
    );
  }
  const def = BLOCK_DEFS[block.type];
  const Icon = def.icon;
  return (
    <div className="flex flex-col gap-4 p-5">
      <div className="flex items-center gap-2.5">
        <span
          className="flex h-8 w-8 items-center justify-center rounded-[var(--radius-7)]"
          style={{ background: `color-mix(in srgb, ${def.accent} 14%, transparent)`, color: def.accent }}
        >
          <Icon size={16} />
        </span>
        <div>
          <div className="text-[length:var(--text-body)] font-semibold text-text-primary">
            {def.label}
          </div>
          <div className="text-[length:var(--text-caption)] text-text-muted">{def.blurb}</div>
        </div>
      </div>
      <BlockConfig block={block} onChange={onChange} />
    </div>
  );
}

function BlockConfig({
  block,
  onChange,
}: {
  block: Block;
  onChange: (patch: Record<string, unknown>) => void;
}) {
  const c = block.config;

  switch (block.type) {
    case 'endpoint_handler':
      return (
        <>
          <p className="text-[length:var(--text-body)] text-text-secondary">
            The request enters here. Later blocks read it as{' '}
            <code className="font-mono text-text-primary">request.body</code>,{' '}
            <code className="font-mono text-text-primary">request.query</code>,{' '}
            <code className="font-mono text-text-primary">request.params</code> and{' '}
            <code className="font-mono text-text-primary">request.headers</code>.
          </p>
          <TextField
            label="Redact these fields in stored history"
            value={csvFromList(c.redactFields)}
            onChange={(e) => onChange({ redactFields: listFromCsv(e.target.value) })}
            placeholder="ssn, internalNote"
            hint="Comma-separated body field names. Secret-named fields (password, token, card…) are always redacted."
          />
        </>
      );

    case 'endpoint_data':
      return (
        <>
          <SelectField
            label="Action"
            value={String(c.action ?? 'create')}
            onChange={(v) => onChange({ action: v })}
            options={DATA_ACTIONS}
          />
          <div className="grid grid-cols-2 gap-3">
            <TextField
              label="Database ID"
              value={String(c.databaseId ?? '')}
              onChange={(e) => onChange({ databaseId: e.target.value })}
              placeholder="app"
            />
            <TextField
              label="Table ID"
              value={String(c.tableId ?? '')}
              onChange={(e) => onChange({ tableId: e.target.value })}
              placeholder="users"
            />
          </div>
          {['get', 'update', 'delete'].includes(String(c.action)) && (
            <TextField
              label="Row ID"
              value={String(c.rowId ?? '')}
              onChange={(e) => onChange({ rowId: e.target.value })}
              placeholder="{{.request.params.id}}"
              hint="Templatable from the request or an earlier block."
            />
          )}
          {['create', 'update'].includes(String(c.action)) && (
            <JsonField
              label="Data"
              hint="Field values. String values are templated, e.g. {{.request.body.email}}."
              value={c.data ?? {}}
              onChange={(obj) => onChange({ data: obj })}
            />
          )}
          <div className="flex items-start justify-between gap-3 rounded-[var(--radius)] border border-border p-3">
            <div>
              <div className="text-[length:var(--text-body)] font-medium text-text-primary">
                Apply rules
              </div>
              <div className="text-[length:var(--text-caption)] text-text-muted">
                On, runs as the caller under row security. Off, runs as the service role with full access.
              </div>
            </div>
            <Switch checked={c.applyRules !== false} onCheckedChange={(v) => onChange({ applyRules: v })} />
          </div>
        </>
      );

    case 'set_variable':
      return (
        <>
          <TextField
            label="Name"
            value={String(c.key ?? '')}
            onChange={(e) => onChange({ key: e.target.value })}
            placeholder="fullName"
          />
          <TextField
            label="Value"
            value={String(c.value ?? '')}
            onChange={(e) => onChange({ value: e.target.value })}
            placeholder="{{.request.body.first}} {{.request.body.last}}"
          />
        </>
      );

    case 'if_condition':
      return (
        <>
          <TextField
            label="Field"
            value={String(c.field ?? '')}
            onChange={(e) => onChange({ field: e.target.value })}
            placeholder="request.body.email"
          />
          <div className="grid grid-cols-2 gap-3">
            <SelectField
              label="Operator"
              value={String(c.operator ?? 'eq')}
              onChange={(v) => onChange({ operator: v })}
              options={CONDITION_OPERATORS}
            />
            <TextField
              label="Value"
              value={String(c.value ?? '')}
              onChange={(e) => onChange({ value: e.target.value })}
              placeholder="active"
            />
          </div>
          <p className="rounded-[var(--radius)] border border-border bg-fill px-3 py-2 text-[length:var(--text-caption)] text-text-muted">
            Connect this block's <span className="text-[#35c07f]">true</span> and{' '}
            <span className="text-[var(--color-danger)]">false</span> outputs on the canvas to choose
            which block runs on each outcome.
          </p>
        </>
      );

    case 'endpoint_response':
      return (
        <>
          <div className="grid grid-cols-2 gap-3">
            <TextField
              label="Status code"
              type="number"
              value={String(c.status ?? 200)}
              onChange={(e) => onChange({ status: Number(e.target.value) || 200 })}
            />
            <SelectField
              label="Body type"
              value={String(c.mode ?? 'json')}
              onChange={(v) => onChange({ mode: v })}
              options={RESPONSE_MODES}
            />
          </div>
          <TextField
            label="Body from a block (optional)"
            value={String(c.bodyField ?? '')}
            onChange={(e) => onChange({ bodyField: e.target.value })}
            placeholder="a block id, e.g. n2"
            hint="Return an earlier block's output directly. Leave empty to use the body below."
          />
          <TextAreaField
            label="Body"
            value={String(c.body ?? '')}
            onChange={(e) => onChange({ body: e.target.value })}
            placeholder={'{"id":"{{.n2.$id}}","ok":true}'}
            rows={5}
            className="font-mono"
          />
        </>
      );

    case 'edit_fields':
      return (
        <JsonField
          label="Fields"
          hint="A JSON object of name → value. String values are templated, e.g. {{.request.body.email}}."
          value={c.fields ?? {}}
          onChange={(obj) => onChange({ fields: obj })}
        />
      );

    case 'filter':
      return (
        <>
          <TextField
            label="Field"
            value={String(c.field ?? '')}
            onChange={(e) => onChange({ field: e.target.value })}
            placeholder="request.body.status"
          />
          <div className="grid grid-cols-2 gap-3">
            <SelectField
              label="Keep when"
              value={String(c.operator ?? 'eq')}
              onChange={(v) => onChange({ operator: v })}
              options={CONDITION_OPERATORS}
            />
            <TextField
              label="Value"
              value={String(c.value ?? '')}
              onChange={(e) => onChange({ value: e.target.value })}
              placeholder="active"
            />
          </div>
          <p className="text-[length:var(--text-caption)] text-text-muted">
            If the condition fails, this block and everything after it is skipped.
          </p>
        </>
      );

    case 'http_request':
      return (
        <>
          <div className="grid grid-cols-[110px_1fr] gap-3">
            <SelectField
              label="Method"
              value={String(c.method ?? 'GET')}
              onChange={(v) => onChange({ method: v })}
              options={HTTP_METHODS.map((m) => ({ value: m, label: m }))}
            />
            <TextField
              label="URL"
              value={String(c.url ?? '')}
              onChange={(e) => onChange({ url: e.target.value })}
              placeholder="https://api.example.com/things"
            />
          </div>
          <TextAreaField
            label="Body (optional)"
            value={String(c.body ?? '')}
            onChange={(e) => onChange({ body: e.target.value })}
            placeholder={'{"key":"{{.request.body.value}}"}'}
            rows={4}
            className="font-mono"
          />
          <JsonField
            label="Headers (optional)"
            value={c.headers ?? {}}
            onChange={(obj) => onChange({ headers: obj })}
          />
          <p className="text-[length:var(--text-caption)] text-text-muted">
            Read the result as <code className="font-mono">{'{{.<blockId>.body}}'}</code> and{' '}
            <code className="font-mono">{'{{.<blockId>.statusCode}}'}</code>.
          </p>
        </>
      );

    case 'applad_functions':
      return (
        <>
          <TextField
            label="Function / target ID"
            value={String(c.targetId ?? '')}
            onChange={(e) => onChange({ targetId: e.target.value })}
            placeholder="fn_123"
          />
          <JsonField
            label="Input data"
            value={c.data ?? {}}
            onChange={(obj) => onChange({ data: obj })}
          />
          <TextField
            label="Project API key"
            value={String(c.apiKey ?? '')}
            onChange={(e) => onChange({ apiKey: e.target.value })}
            placeholder="applad_key_…"
            hint="Server-side call, so it needs a project API key. Store it in a Vault value and template it in."
          />
        </>
      );

    case 'applad_messaging':
      return (
        <>
          <TextField
            label="To"
            value={String(c.to ?? '')}
            onChange={(e) => onChange({ to: e.target.value })}
            placeholder="{{.request.body.email}}"
          />
          <TextField
            label="Subject"
            value={String(c.subject ?? '')}
            onChange={(e) => onChange({ subject: e.target.value })}
            placeholder="Welcome"
          />
          <TextAreaField
            label="Body"
            value={String(c.body ?? '')}
            onChange={(e) => onChange({ body: e.target.value })}
            rows={4}
          />
          <TextField
            label="Project API key"
            value={String(c.apiKey ?? '')}
            onChange={(e) => onChange({ apiKey: e.target.value })}
            placeholder="applad_key_…"
            hint="Sending is a server-side action, so it needs a project API key."
          />
        </>
      );

    default:
      return null;
  }
}

function csvFromList(v: unknown): string {
  return Array.isArray(v) ? v.join(', ') : '';
}

function listFromCsv(s: string): string[] {
  return s
    .split(',')
    .map((x) => x.trim())
    .filter(Boolean);
}

/** A JSON object editor backed by a text buffer, so mid-edit invalid JSON does
 * not throw away the field. Commits to the parent only on a valid parse. */
function JsonField({
  label,
  hint,
  value,
  onChange,
}: {
  label: string;
  hint?: string;
  value: unknown;
  onChange: (obj: Record<string, unknown>) => void;
}) {
  const [text, setText] = useState(() => JSON.stringify(value ?? {}, null, 2));
  const [error, setError] = useState('');
  return (
    <FormField label={label} hint={error ? undefined : hint} error={error}>
      <textarea
        value={text}
        rows={5}
        spellCheck={false}
        onChange={(e) => {
          setText(e.target.value);
          try {
            const parsed = e.target.value.trim() ? JSON.parse(e.target.value) : {};
            if (typeof parsed !== 'object' || Array.isArray(parsed)) {
              setError('Expected a JSON object');
              return;
            }
            setError('');
            onChange(parsed as Record<string, unknown>);
          } catch {
            setError('Invalid JSON');
          }
        }}
        className="w-full rounded-[var(--radius)] border border-border bg-background px-3 py-2 font-mono text-[length:var(--text-caption)] text-text-primary outline-none focus:border-[var(--color-accent)]"
      />
    </FormField>
  );
}
