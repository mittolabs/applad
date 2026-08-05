import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { DownloadCloud, Loader2, CheckCircle2, XCircle, Ban } from 'lucide-react';
import { api } from '../../api/client';

/* Migrations — import a project's data from another platform into this project.
 * Mirrors Appwrite's Migrations tab: pick a source, enter credentials, validate
 * (a pre-flight count), then start. Progress is polled from the migration
 * record while a job is running. */

type GroupKey = 'auth' | 'databases' | 'storage';

type Counts = Record<string, { total: number; done: number; error: number; warning: number; skip: number }>;

interface Migration {
  id: string;
  sourceType: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  groups: string[];
  counts: Counts;
  error?: string;
  createdAt: string;
}

interface Field {
  key: string;
  label: string;
  placeholder: string;
  secret?: boolean;
  multiline?: boolean;
  optional?: boolean;
}

interface SourceDef {
  id: string;
  label: string;
  available: boolean;
  fields: Field[];
}

const SOURCES: SourceDef[] = [
  {
    id: 'applad',
    label: 'Applad (this or another instance)',
    available: true,
    fields: [
      { key: 'endpoint', label: 'Instance URL', placeholder: 'https://api.other-instance.io — leave blank for this instance', optional: true },
      { key: 'sourceProjectId', label: 'Source project ID', placeholder: 'the project to import from' },
      { key: 'sourceApiKey', label: 'Source API key', placeholder: 'an API key for that project', secret: true },
    ],
  },
  {
    id: 'appwrite',
    label: 'Appwrite',
    available: true,
    fields: [
      { key: 'endpoint', label: 'Endpoint', placeholder: 'https://cloud.appwrite.io/v1' },
      { key: 'projectId', label: 'Project ID', placeholder: 'Appwrite project ID' },
      { key: 'apiKey', label: 'API key', placeholder: 'a server API key', secret: true },
    ],
  },
  {
    id: 'supabase',
    label: 'Supabase',
    available: true,
    fields: [
      { key: 'host', label: 'Database host', placeholder: 'from the connection pooler string' },
      { key: 'port', label: 'Port', placeholder: '6543', optional: true },
      { key: 'user', label: 'Database user', placeholder: 'postgres.<ref>' },
      { key: 'password', label: 'Database password', placeholder: 'database password', secret: true },
      { key: 'projectUrl', label: 'Project URL', placeholder: 'https://<ref>.supabase.co (for storage)', optional: true },
      { key: 'serviceKey', label: 'service_role key', placeholder: 'for storage', secret: true, optional: true },
    ],
  },
  {
    id: 'nhost',
    label: 'NHost',
    available: true,
    fields: [
      { key: 'host', label: 'Database host', placeholder: 'Postgres host' },
      { key: 'port', label: 'Port', placeholder: '5432', optional: true },
      { key: 'user', label: 'Database user', placeholder: 'postgres' },
      { key: 'password', label: 'Database password', placeholder: 'database password', secret: true },
      { key: 'storageUrl', label: 'Storage URL', placeholder: 'https://<sub>.storage.<region>.nhost.run (for files)', optional: true },
      { key: 'adminSecret', label: 'Hasura admin secret', placeholder: 'for files', secret: true, optional: true },
    ],
  },
  {
    id: 'firebase',
    label: 'Firebase',
    available: true,
    fields: [
      { key: 'serviceAccount', label: 'Service account JSON', placeholder: 'paste the service-account JSON', multiline: true },
      { key: 'signerKey', label: 'Password hash: signer key', placeholder: 'base64 (Auth → password hash params)', optional: true },
      { key: 'saltSeparator', label: 'Password hash: salt separator', placeholder: 'base64', optional: true },
      { key: 'rounds', label: 'Password hash: rounds', placeholder: '8', optional: true },
      { key: 'memCost', label: 'Password hash: mem cost', placeholder: '14', optional: true },
    ],
  },
];

const ALL_GROUPS: { key: GroupKey; label: string }[] = [
  { key: 'auth', label: 'Users & teams' },
  { key: 'databases', label: 'Databases' },
  { key: 'storage', label: 'Storage' },
];

function progressOf(m: Migration): number {
  let total = 0;
  let done = 0;
  for (const g of Object.values(m.counts ?? {})) {
    total += g.total ?? 0;
    done += (g.done ?? 0) + (g.error ?? 0) + (g.skip ?? 0) + (g.warning ?? 0);
  }
  if (total === 0) return m.status === 'completed' ? 100 : 0;
  return Math.min(100, Math.round((done / total) * 100));
}

const statusStyle: Record<Migration['status'], string> = {
  pending: 'text-zinc-400',
  running: 'text-[#6C47FF]',
  completed: 'text-emerald-400',
  failed: 'text-red-400',
  cancelled: 'text-zinc-500',
};

export function MigrationsPage() {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [source, setSource] = useState<SourceDef>(SOURCES[0]);
  const [creds, setCreds] = useState<Record<string, string>>({});
  const [groups, setGroups] = useState<GroupKey[]>(['auth', 'databases', 'storage']);
  const [validation, setValidation] = useState<Record<string, number> | null>(null);
  const [error, setError] = useState('');

  const list = useQuery({
    queryKey: ['migrations'],
    queryFn: async () => (await api.get('/migrations')).data.migrations as Migration[],
    refetchInterval: (q) =>
      (q.state.data ?? []).some((m) => m.status === 'running' || m.status === 'pending') ? 2000 : false,
  });

  const validate = useMutation({
    mutationFn: async () =>
      (await api.post('/migrations/validate', { source: source.id, groups, credentials: creds })).data,
    onSuccess: (d) => {
      setValidation(d.counts ?? {});
      setError('');
    },
    onError: (e: unknown) => setError(errText(e)),
  });

  const create = useMutation({
    mutationFn: async () =>
      (await api.post('/migrations', { source: source.id, groups, credentials: creds })).data,
    onSuccess: () => {
      setShowForm(false);
      setCreds({});
      setValidation(null);
      setError('');
      qc.invalidateQueries({ queryKey: ['migrations'] });
    },
    onError: (e: unknown) => setError(errText(e)),
  });

  const cancel = useMutation({
    mutationFn: async (id: string) => api.post(`/migrations/${id}/cancel`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['migrations'] }),
  });

  const migrations = list.data ?? [];

  return (
    <div className="mx-auto max-w-4xl px-6 py-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-zinc-100">Migrations</h1>
          <p className="mt-1 text-sm text-zinc-400">
            Import users, databases and storage from another platform into this project.
          </p>
        </div>
        <button
          onClick={() => setShowForm((v) => !v)}
          className="flex items-center gap-2 rounded-lg bg-[#6C47FF] px-3 py-2 text-sm font-medium text-white hover:bg-[#5a3ce0]"
        >
          <DownloadCloud size={16} /> Import data
        </button>
      </div>

      {showForm && (
        <div className="mb-8 rounded-xl border border-zinc-800 bg-[#16171B] p-5">
          <label className="mb-1 block text-xs font-medium text-zinc-400">Source</label>
          <select
            value={source.id}
            onChange={(e) => {
              const s = SOURCES.find((x) => x.id === e.target.value)!;
              setSource(s);
              setCreds({});
              setValidation(null);
            }}
            className="mb-4 w-full rounded-lg border border-zinc-700 bg-[#0B0B0F] px-3 py-2 text-sm text-zinc-100"
          >
            {SOURCES.map((s) => (
              <option key={s.id} value={s.id} disabled={!s.available}>
                {s.label}
              </option>
            ))}
          </select>

          {source.fields.map((f) =>
            f.multiline ? (
              <div key={f.key} className="mb-3">
                <label className="mb-1 block text-xs font-medium text-zinc-400">{f.label}</label>
                <textarea
                  rows={5}
                  autoComplete="off"
                  placeholder={f.placeholder}
                  value={creds[f.key] ?? ''}
                  onChange={(e) => setCreds((c) => ({ ...c, [f.key]: e.target.value }))}
                  className="w-full rounded-lg border border-zinc-700 bg-[#0B0B0F] px-3 py-2 font-mono text-xs text-zinc-100"
                />
              </div>
            ) : (
              <div key={f.key} className="mb-3">
                <label className="mb-1 block text-xs font-medium text-zinc-400">
                  {f.label}
                  {f.optional && <span className="ml-1 text-zinc-600">(optional)</span>}
                </label>
                <input
                  type={f.secret ? 'password' : 'text'}
                  autoComplete="off"
                  placeholder={f.placeholder}
                  value={creds[f.key] ?? ''}
                  onChange={(e) => setCreds((c) => ({ ...c, [f.key]: e.target.value }))}
                  className="w-full rounded-lg border border-zinc-700 bg-[#0B0B0F] px-3 py-2 text-sm text-zinc-100"
                />
              </div>
            ),
          )}

          <div className="my-4">
            <span className="mb-2 block text-xs font-medium text-zinc-400">What to import</span>
            <div className="flex flex-wrap gap-4">
              {ALL_GROUPS.map((g) => (
                <label key={g.key} className="flex items-center gap-2 text-sm text-zinc-200">
                  <input
                    type="checkbox"
                    checked={groups.includes(g.key)}
                    onChange={(e) =>
                      setGroups((gs) => (e.target.checked ? [...gs, g.key] : gs.filter((x) => x !== g.key)))
                    }
                  />
                  {g.label}
                  {validation && validation[g.key] != null && (
                    <span className="text-zinc-500">({validation[g.key]})</span>
                  )}
                </label>
              ))}
            </div>
          </div>

          {error && <p className="mb-3 text-sm text-red-400">{error}</p>}

          <div className="flex gap-2">
            <button
              onClick={() => validate.mutate()}
              disabled={validate.isPending || !source.available}
              className="rounded-lg border border-zinc-700 px-3 py-2 text-sm text-zinc-200 hover:bg-zinc-800 disabled:opacity-50"
            >
              {validate.isPending ? 'Checking…' : 'Validate'}
            </button>
            <button
              onClick={() => create.mutate()}
              disabled={create.isPending || !source.available || groups.length === 0}
              className="rounded-lg bg-[#6C47FF] px-3 py-2 text-sm font-medium text-white hover:bg-[#5a3ce0] disabled:opacity-50"
            >
              {create.isPending ? 'Starting…' : 'Start import'}
            </button>
          </div>
          <p className="mt-3 text-xs text-zinc-500">
            Imported accounts keep their passwords and re-hash on first sign-in. OAuth-only logins,
            functions and timestamps are not migrated.
          </p>
        </div>
      )}

      <div className="space-y-3">
        {migrations.length === 0 && !list.isLoading && (
          <p className="text-sm text-zinc-500">No migrations yet.</p>
        )}
        {migrations.map((m) => (
          <MigrationRow key={m.id} m={m} onCancel={() => cancel.mutate(m.id)} />
        ))}
      </div>
    </div>
  );
}

function MigrationRow({ m, onCancel }: { m: Migration; onCancel: () => void }) {
  const pct = useMemo(() => progressOf(m), [m]);
  const Icon =
    m.status === 'completed'
      ? CheckCircle2
      : m.status === 'failed'
        ? XCircle
        : m.status === 'cancelled'
          ? Ban
          : Loader2;
  return (
    <div className="rounded-xl border border-zinc-800 bg-[#16171B] p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Icon size={16} className={`${statusStyle[m.status]} ${m.status === 'running' ? 'animate-spin' : ''}`} />
          <span className="text-sm font-medium text-zinc-100 capitalize">{m.sourceType}</span>
          <span className={`text-xs ${statusStyle[m.status]}`}>{m.status}</span>
        </div>
        {(m.status === 'running' || m.status === 'pending') && (
          <button onClick={onCancel} className="text-xs text-zinc-400 hover:text-red-400">
            Cancel
          </button>
        )}
      </div>
      <div className="mt-3 h-1.5 w-full overflow-hidden rounded-full bg-zinc-800">
        <div className="h-full bg-[#6C47FF] transition-all" style={{ width: `${pct}%` }} />
      </div>
      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-zinc-500">
        {Object.entries(m.counts ?? {}).map(([g, c]) => (
          <span key={g}>
            {g}: {c.done}/{c.total}
            {c.error ? <span className="text-red-400"> · {c.error} failed</span> : null}
          </span>
        ))}
      </div>
      {m.error && <p className="mt-2 text-xs text-red-400">{m.error}</p>}
    </div>
  );
}

function errText(e: unknown): string {
  const anyE = e as { response?: { data?: { message?: string } }; message?: string };
  return anyE?.response?.data?.message ?? anyE?.message ?? 'Something went wrong';
}

export default MigrationsPage;
