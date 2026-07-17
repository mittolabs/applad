import { useMemo, useState } from 'react';
import { HeartPulse } from 'lucide-react';
import { DataTable, type DataTableColumn } from '@/components/data-table';
import { FormDialog, SelectField, TextField } from '@/components/form-dialog';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import {
  OB_GREEN,
  OB_ORANGE,
  OB_RED,
  OB_SLATE,
  asRows,
  cap,
  num,
  obTimeAgo,
  useObserveResource,
} from './observe-shared';

/* ObserveUptime — ports observe_uptime.dart. HTTP monitor list + add dialog. */

const COLUMNS: DataTableColumn[] = [
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'name', label: 'Name', flex: 3 },
  { key: 'url', label: 'URL', flex: 4 },
  { key: 'uptime', label: 'Uptime', flex: 2, sortable: false },
  { key: 'latency', label: 'Latency', flex: 2, sortable: false },
  { key: 'checked', label: 'Checked', flex: 2, sortable: false },
];

function statusColor(s: string): string {
  return s === 'up' ? OB_GREEN : s === 'down' ? OB_RED : s === 'degraded' ? OB_ORANGE : OB_SLATE;
}

export function ObserveUptime({ projectId }: { projectId?: string }) {
  const query = useObserveResource('/observe/uptime', projectId);
  const [search, setSearch] = useState('');
  const [filters, setFilters] = useState<Record<string, string | null>>({});
  const [adding, setAdding] = useState(false);

  const allMonitors = asRows(query.data?.monitors);

  const rows = useMemo(() => {
    const q = search.trim().toLowerCase();
    return allMonitors.filter((m) => {
      if (
        q &&
        !String(m.name ?? '').toLowerCase().includes(q) &&
        !String(m.url ?? '').toLowerCase().includes(q)
      )
        return false;
      if (filters.status && String(m.status ?? 'up') !== filters.status) return false;
      return true;
    });
  }, [allMonitors, search, filters]);

  return (
    <div className="px-6 md:px-8">
      <DataTable
        columns={COLUMNS}
        rows={rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'status':
              return String(row.status ?? 'up');
            case 'name':
              return String(row.name ?? '');
            case 'url':
              return String(row.url ?? '');
            case 'uptime':
              return `${num(row.uptimePct, 100).toFixed(2)}%`;
            case 'latency':
              return `${num(row.latencyMs)}ms`;
            case 'checked':
              return obTimeAgo(row.lastChecked);
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'status') {
            const s = String(row.status ?? 'up');
            const c = statusColor(s);
            return (
              <span className="inline-flex items-center gap-1.5" style={{ color: c }}>
                <span className="h-2 w-2 rounded-full" style={{ backgroundColor: c }} />
                <span className="text-[length:var(--text-label)] font-medium">{cap(s)}</span>
              </span>
            );
          }
          if (key === 'uptime') {
            const pct = num(row.uptimePct, 100);
            const c = pct >= 99.9 ? OB_GREEN : pct >= 99.0 ? OB_ORANGE : OB_RED;
            return (
              <span className="text-[length:var(--text-body)] font-semibold" style={{ color: c }}>
                {pct.toFixed(2)}%
              </span>
            );
          }
          return undefined;
        }}
        rowIcon={() => HeartPulse}
        rowIconColor={(row) => statusColor(String(row.status ?? 'up'))}
        onDeleteRow={async (row) => {
          try {
            await api.delete(`/observe/uptime/${row.$id}`);
            await query.refetch();
          } catch (e) {
            toast.error(friendlyError(e));
          }
        }}
        createLabel="Add monitor"
        onCreate={() => setAdding(true)}
        filters={[
          {
            key: 'status',
            label: 'Status',
            options: ['up', 'down', 'degraded', 'paused'].map((v) => ({ value: v, label: cap(v) })),
          },
        ]}
        filterValues={filters}
        onFiltersChange={setFilters}
        searchHint="Search monitors…"
        searchValue={search}
        onSearchChange={setSearch}
        itemLabel="monitors"
        emptyIcon={HeartPulse}
        emptyTitle="No uptime monitors"
        emptySubtitle="Track availability of your services and endpoints"
        loading={query.isLoading}
        error={query.error}
        onRetry={query.refetch}
      />

      <AddMonitorDialog open={adding} onOpenChange={setAdding} onCreated={() => query.refetch()} />
    </div>
  );
}

function AddMonitorDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [checkType, setCheckType] = useState('http');
  const [intervalSecs, setIntervalSecs] = useState('60');
  const [saving, setSaving] = useState(false);

  const reset = () => {
    setName('');
    setUrl('');
    setCheckType('http');
    setIntervalSecs('60');
  };

  const submit = async () => {
    setSaving(true);
    try {
      await api.post('/observe/uptime', {
        name: name.trim(),
        url: url.trim(),
        checkType,
        intervalSecs: parseInt(intervalSecs, 10) || 60,
      });
      onOpenChange(false);
      reset();
      onCreated();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open={open}
      onOpenChange={(o) => {
        onOpenChange(o);
        if (!o) reset();
      }}
      title="Add uptime monitor"
      submitLabel="Add"
      loading={saving}
      submitDisabled={!name.trim() || !url.trim()}
      onSubmit={submit}
    >
      <TextField label="Name" placeholder="API health" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      <TextField
        label="URL or host"
        placeholder="https://api.example.com/health"
        value={url}
        onChange={(e) => setUrl(e.target.value)}
      />
      <SelectField
        label="Check type"
        value={checkType}
        onChange={setCheckType}
        options={[
          { value: 'http', label: 'HTTP(S)' },
          { value: 'tcp', label: 'TCP port' },
          { value: 'ping', label: 'ICMP ping' },
          { value: 'keyword', label: 'Keyword match' },
        ]}
      />
      <SelectField
        label="Check interval"
        value={intervalSecs}
        onChange={setIntervalSecs}
        options={[
          { value: '30', label: '30 seconds' },
          { value: '60', label: '1 minute' },
          { value: '300', label: '5 minutes' },
          { value: '600', label: '10 minutes' },
          { value: '1800', label: '30 minutes' },
        ]}
      />
    </FormDialog>
  );
}
