import { useCallback, useEffect, useRef, useState } from 'react';
import Editor from '@monaco-editor/react';
import {
  ArrowUpRight,
  BadgeCheck,
  ChevronDown,
  ChevronRight,
  Copy,
  CopyPlus,
  Database,
  FileJson,
  FileSpreadsheet,
  Play,
  Plus,
} from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/form-dialog';
import { useMonacoTheme } from '@/stores/theme';
import { MetricPill, str, type Json } from './shared';

interface SchemaColumn {
  name: string;
  type: string;
  required: boolean;
}
interface SchemaTable {
  name: string;
  columns: SchemaColumn[];
}

const SCHEMA_QUERY = `select
  table_name,
  column_name,
  data_type,
  is_nullable,
  udt_name
from information_schema.columns
where table_schema = current_schema()
order by table_name, ordinal_position`;

export function SqlConsole({
  dbId,
  tables,
  onOpenTable,
}: {
  dbId: string;
  tables: Json[];
  onOpenTable: (id: string, name: string) => void;
}) {
  const monacoTheme = useMonacoTheme();
  const [sql, setSql] = useState('');
  const [result, setResult] = useState<Json | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const [writeAllowed, setWriteAllowed] = useState(false);
  const [confirmWrite, setConfirmWrite] = useState(false);
  const [history, setHistory] = useState<string[]>([]);
  const [schema, setSchema] = useState<SchemaTable[]>([]);
  const [schemaLoading, setSchemaLoading] = useState(false);
  const [schemaError, setSchemaError] = useState<string | null>(null);
  const sqlRef = useRef(sql);
  sqlRef.current = sql;

  const tableIdByName: Record<string, string> = {};
  for (const t of tables) tableIdByName[str(t['name'])] = str(t['$id']);

  const loadSchema = useCallback(async () => {
    setSchemaLoading(true);
    setSchemaError(null);
    try {
      const res = await api.post(`/databases/${dbId}/sql`, {
        statement: SCHEMA_QUERY,
        writeAllowed: false,
      });
      const payload = res.data as Json;
      const rows = (payload['rows'] as Json[] | undefined) ?? [];
      const grouped = new Map<string, SchemaColumn[]>();
      for (const row of rows) {
        const tableName = str(row['table_name']);
        const columnName = str(row['column_name']);
        if (!tableName || !columnName) continue;
        const cols = grouped.get(tableName) ?? [];
        cols.push({
          name: columnName,
          type: str(row['data_type']) || str(row['udt_name']) || 'text',
          required: str(row['is_nullable']).toUpperCase() === 'NO',
        });
        grouped.set(tableName, cols);
      }
      const next = Array.from(grouped.entries())
        .map(([name, columns]) => ({ name, columns }))
        .sort((a, b) => a.name.localeCompare(b.name));
      setSchema(next);
    } catch (e) {
      setSchemaError(friendlyError(e));
    } finally {
      setSchemaLoading(false);
    }
  }, [dbId]);

  useEffect(() => {
    void loadSchema();
  }, [loadSchema]);

  const runSQL = useCallback(async () => {
    const statement = sqlRef.current.trim();
    if (!statement) {
      setError('SQL statement is required.');
      return;
    }
    setRunning(true);
    setError(null);
    try {
      const res = await api.post(`/databases/${dbId}/sql`, { statement, writeAllowed });
      setResult(res.data as Json);
      setHistory((prev) => [statement, ...prev.filter((s) => s !== statement)].slice(0, 12));
    } catch (e) {
      setResult(null);
      setError(friendlyError(e));
    } finally {
      setRunning(false);
    }
  }, [dbId, writeAllowed]);

  const insertIdentifier = (identifier: string) => {
    setSql((prev) => {
      const needsSpace = prev.length > 0 && !/\s$/.test(prev);
      return `${prev}${needsSpace ? ' ' : ''}${identifier}`;
    });
  };

  const copySQL = async () => {
    try {
      await navigator.clipboard.writeText(sql);
    } catch {
      /* clipboard unavailable */
    }
  };

  const columns = ((result?.['columns'] as string[] | undefined) ?? []).slice();
  const rows = (result?.['rows'] as Json[] | undefined) ?? [];

  return (
    <div className="flex flex-col gap-4 lg:flex-row lg:items-start">
      <div className="flex min-w-0 flex-1 flex-col gap-4">
        {/* Editor card */}
        <div className="rounded-[var(--radius-10)] border border-border bg-surface p-4">
          <div className="flex items-center gap-2">
            <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
              SQL editor
            </div>
            <div className="ml-auto flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={copySQL}>
                <Copy size={14} />
                Copy
              </Button>
              <Button size="sm" onClick={runSQL} loading={running} disabled={running}>
                {!running && <Play size={14} />}
                {running ? 'Running…' : 'Run query'}
              </Button>
            </div>
          </div>
          <p className="mt-2 text-[length:var(--text-label)] text-text-muted">
            DDL is blocked. Queries run read-only by default with project and user context applied.
          </p>

          <div className="mt-4 overflow-hidden rounded-[var(--radius)] border border-border">
            <Editor
              height="320px"
              theme={monacoTheme}
              defaultLanguage="sql"
              value={sql}
              onChange={(v) => setSql(v ?? '')}
              onMount={(editor, monaco) => {
                editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
                  void runSQL();
                });
              }}
              options={{
                minimap: { enabled: false },
                fontSize: 13,
                lineHeight: 21,
                wordWrap: 'on',
                automaticLayout: true,
                scrollBeyondLastLine: false,
                quickSuggestions: true,
              }}
            />
          </div>

          <div className="mt-4 flex items-center gap-3">
            <Switch
              checked={writeAllowed}
              disabled={running}
              onCheckedChange={(v) => {
                if (!v) setWriteAllowed(false);
                else setConfirmWrite(true);
              }}
            />
            <span className="text-[length:var(--text-body)] text-text-primary">Allow writes</span>
            <span
              className="text-[length:var(--text-label)]"
              style={{ color: writeAllowed ? '#FFD166' : 'var(--text-muted)' }}
            >
              {writeAllowed
                ? 'INSERT, UPDATE and DELETE are allowed.'
                : 'Transaction is forced read-only.'}
            </span>
          </div>
          <p className="mt-2 text-[length:var(--text-caption)] text-text-subtle">
            Use the schema browser to insert tables and columns, then run with Cmd+Enter or
            Ctrl+Enter.
          </p>
        </div>

        {/* Results card */}
        <div className="rounded-[var(--radius-10)] border border-border bg-surface p-4">
          <div className="flex items-center gap-2">
            <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
              Results
            </div>
            {result && (
              <div className="ml-auto flex items-center gap-2">
                {columns.length > 0 && rows.length > 0 && (
                  <>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => downloadResults('json', dbId, columns, rows)}
                    >
                      <FileJson size={14} />
                      JSON
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => downloadResults('csv', dbId, columns, rows)}
                    >
                      <FileSpreadsheet size={14} />
                      CSV
                    </Button>
                  </>
                )}
                <MetricPill label={`${str(result['rowCount'] ?? 0)} rows`} />
                <MetricPill label={`${str(result['executionMs'] ?? 0)} ms`} />
              </div>
            )}
          </div>

          <div className="mt-4 min-h-[220px]">
            {error ? (
              <pre className="overflow-x-auto whitespace-pre-wrap rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] p-4 font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-[var(--color-danger)]">
                {error}
              </pre>
            ) : !result ? (
              <EmptyState
                icon={Database}
                title="No results yet"
                subtitle="Run a statement to inspect rows or affected counts."
                actionLabel="Run query"
                onAction={runSQL}
              />
            ) : rows.length === 0 ? (
              <SummaryView result={result} />
            ) : (
              <ResultsGrid columns={columns} rows={rows} />
            )}
          </div>
        </div>
      </div>

      {/* Right rail: schema browser + recent queries */}
      <div className="flex w-full flex-col gap-4 lg:w-[300px] lg:shrink-0">
        <SchemaBrowser
          schema={schema}
          loading={schemaLoading}
          error={schemaError}
          tableIdByName={tableIdByName}
          onOpenTable={onOpenTable}
          onInsert={insertIdentifier}
        />
        <div className="rounded-[var(--radius-10)] border border-border bg-surface p-4">
          <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
            Recent queries
          </div>
          {history.length === 0 ? (
            <p className="mt-3 text-[length:var(--text-label)] leading-relaxed text-text-subtle">
              Recent statements from this session will appear here.
            </p>
          ) : (
            <div className="mt-3 flex flex-col gap-2">
              {history.map((statement, i) => (
                <button
                  key={i}
                  type="button"
                  onClick={() => setSql(statement)}
                  className="rounded-[var(--radius)] border border-border bg-fill p-3 text-left font-[family-name:var(--font-mono)] text-[length:var(--text-label)] leading-relaxed text-text-primary"
                >
                  <span className="line-clamp-4">{statement}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      <ConfirmDialog
        open={confirmWrite}
        onOpenChange={setConfirmWrite}
        title="Enable write mode"
        message="Write mode allows INSERT, UPDATE, and DELETE statements in this database. Schema changes are still blocked. Enable only when you intend to mutate data."
        confirmLabel="Enable writes"
        onConfirm={() => {
          setWriteAllowed(true);
          setConfirmWrite(false);
        }}
      />
    </div>
  );
}

function SchemaBrowser({
  schema,
  loading,
  error,
  tableIdByName,
  onOpenTable,
  onInsert,
}: {
  schema: SchemaTable[];
  loading: boolean;
  error: string | null;
  tableIdByName: Record<string, string>;
  onOpenTable: (id: string, name: string) => void;
  onInsert: (identifier: string) => void;
}) {
  return (
    <div className="rounded-[var(--radius-10)] border border-border bg-surface p-4">
      <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
        Schema browser
      </div>
      <p className="mt-1 text-[length:var(--text-label)] leading-relaxed text-text-muted">
        Inspect tables and columns while writing queries.
      </p>
      <div className="mt-3">
        {loading ? (
          <div className="py-6 text-center text-[length:var(--text-label)] text-text-subtle">
            Loading schema…
          </div>
        ) : error ? (
          <div className="text-[length:var(--text-label)] text-[var(--color-danger)]">
            Error: {error}
          </div>
        ) : schema.length === 0 ? (
          <p className="text-[length:var(--text-label)] leading-relaxed text-text-subtle">
            No schema metadata available yet.
          </p>
        ) : (
          <div className="flex flex-col gap-2.5">
            {schema.map((table) => (
              <SchemaTableRow
                key={table.name}
                table={table}
                tableId={tableIdByName[table.name] ?? ''}
                onOpenTable={onOpenTable}
                onInsert={onInsert}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function SchemaTableRow({
  table,
  tableId,
  onOpenTable,
  onInsert,
}: {
  table: SchemaTable;
  tableId: string;
  onOpenTable: (id: string, name: string) => void;
  onInsert: (identifier: string) => void;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div className="rounded-[var(--radius)] border border-border bg-fill">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-2.5 text-left"
      >
        {open ? (
          <ChevronDown size={14} className="text-text-secondary" />
        ) : (
          <ChevronRight size={14} className="text-text-muted" />
        )}
        <div className="min-w-0">
          <div className="truncate text-[length:var(--text-body)] font-semibold text-text-primary">
            {table.name}
          </div>
          <div className="font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-muted">
            information_schema
          </div>
        </div>
      </button>
      {open && (
        <div className="px-3 pb-3">
          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              onClick={() => onInsert(table.name)}
              className="inline-flex items-center gap-1.5 text-[length:var(--text-label)] text-text-secondary hover:text-text-primary"
            >
              <CopyPlus size={14} />
              Insert table
            </button>
            <button
              type="button"
              disabled={!tableId}
              onClick={() => tableId && onOpenTable(tableId, table.name)}
              className="inline-flex items-center gap-1.5 text-[length:var(--text-label)] text-text-secondary hover:text-text-primary disabled:opacity-40"
            >
              <ArrowUpRight size={14} />
              Open table
            </button>
          </div>
          <div className="mt-2 flex flex-col gap-2">
            {table.columns.map((column) => (
              <div
                key={column.name}
                className="flex items-start gap-2 rounded-[var(--radius)] bg-surface-alt p-2.5"
              >
                <Database size={14} className="mt-0.5 text-[var(--color-accent)]" />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="min-w-0 flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-primary">
                      {column.name}
                    </span>
                    <button
                      type="button"
                      onClick={() => onInsert(column.name)}
                      className="text-text-secondary hover:text-text-primary"
                      aria-label={`Insert ${column.name}`}
                    >
                      <Plus size={14} />
                    </button>
                  </div>
                  <div className="mt-1 flex flex-wrap gap-1.5">
                    <MetricPill label={column.type} />
                    {column.required && <MetricPill label="required" />}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function ResultsGrid({ columns, rows }: { columns: string[]; rows: Json[] }) {
  const display = columns.length === 0 && rows.length > 0 ? Object.keys(rows[0]) : columns;
  return (
    <div className="overflow-x-auto rounded-[var(--radius)] border border-border">
      <table className="w-full border-collapse text-left">
        <thead>
          <tr className="bg-fill">
            {display.map((c) => (
              <th
                key={c}
                className="min-w-[180px] px-3 py-2.5 text-[length:var(--text-label)] font-semibold text-text-muted"
              >
                {c}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i} className="border-t border-border">
              {display.map((c) => (
                <td
                  key={c}
                  className="min-w-[180px] max-w-[320px] truncate px-3 py-2.5 font-[family-name:var(--font-mono)] text-[length:var(--text-label)] text-text-primary"
                >
                  {formatValue(row[c])}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function SummaryView({ result }: { result: Json }) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <BadgeCheck size={28} className="text-status-success" />
      <div className="mt-3 text-[length:var(--text-subhead)] font-semibold text-text-primary">
        Statement completed
      </div>
      <div className="mt-1 text-[length:var(--text-label)] text-text-muted">
        {str(result['rowCount'] ?? 0)} rows affected in {str(result['executionMs'] ?? 0)} ms.
      </div>
    </div>
  );
}

function formatValue(value: unknown): string {
  if (value == null) return 'NULL';
  if (typeof value === 'object') return JSON.stringify(value, null, 2);
  return String(value);
}

function csvCell(value: unknown): string {
  const normalized =
    value == null ? '' : typeof value === 'object' ? JSON.stringify(value) : String(value);
  return `"${normalized.replace(/"/g, '""')}"`;
}

function downloadResults(format: 'json' | 'csv', dbId: string, columns: string[], rows: Json[]) {
  if (rows.length === 0 || columns.length === 0) return;
  const safeId = dbId.replace(/[^a-zA-Z0-9_-]/g, '_');
  const timestamp = new Date().toISOString().replace(/:/g, '-');
  let content: string;
  let mime: string;
  let ext: string;
  if (format === 'csv') {
    const lines = [columns.map(csvCell).join(',')];
    for (const row of rows) lines.push(columns.map((c) => csvCell(row[c])).join(','));
    content = lines.join('\n');
    mime = 'text/csv;charset=utf-8';
    ext = 'csv';
  } else {
    content = JSON.stringify(rows, null, 2);
    mime = 'application/json;charset=utf-8';
    ext = 'json';
  }
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `${safeId}_query_${timestamp}.${ext}`;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}
