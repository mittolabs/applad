import { useMemo, useState } from 'react';
import { CheckCircle2, Clock, Loader, Timer, XCircle, type LucideIcon } from 'lucide-react';
import { DataTable, type DataTableColumn } from '@/components/data-table';
import { FormDialog, SelectField, TextField } from '@/components/form-dialog';
import { Switch } from '@/components/ui/switch';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import {
  ACCENT,
  GREEN,
  ORANGE,
  RED,
  SLATE,
  asRows,
  cap,
  timeAgo,
  useAnalyticsResource,
} from './analytics-shared';

/* AnalyticsCrons — cron monitor list + add dialog. A job checks in when it
 * finishes; one that stays silent past its grace period is marked missed. */

const COLUMNS: DataTableColumn[] = [
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'name', label: 'Name', flex: 3 },
  { key: 'schedule', label: 'Schedule', flex: 3 },
  { key: 'timezone', label: 'Timezone', flex: 2, sortable: false },
  { key: 'lastRun', label: 'Last run', flex: 2, sortable: false },
  { key: 'nextRun', label: 'Next run', flex: 2, sortable: false },
  { key: 'enabled', label: 'Enabled', flex: 1 },
];

function statusMeta(s: string): [string, LucideIcon] {
  switch (s) {
    case 'ok':
      return [GREEN, CheckCircle2];
    case 'missed':
      return [ORANGE, Clock];
    case 'failed':
      return [RED, XCircle];
    case 'running':
      return [ACCENT, Loader];
    default:
      return [SLATE, Timer];
  }
}

export function AnalyticsCrons({ projectId }: { projectId?: string }) {
  const query = useAnalyticsResource('/analytics/crons', projectId);
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
        !String(m.schedule ?? '').toLowerCase().includes(q)
      )
        return false;
      if (filters.status && String(m.status ?? 'waiting') !== filters.status) return false;
      return true;
    });
  }, [allMonitors, search, filters]);

  const toggle = async (id: string) => {
    try {
      await api.patch(`/analytics/crons/${id}/toggle`);
      await query.refetch();
    } catch (e) {
      toast.error(friendlyError(e));
    }
  };

  return (
    <div className="px-6 md:px-8">
      <DataTable
        columns={COLUMNS}
        rows={rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'status':
              return String(row.status ?? 'waiting');
            case 'name':
              return String(row.name ?? '');
            case 'schedule':
              return String(row.schedule ?? '');
            case 'timezone':
              return String(row.timezone ?? 'UTC');
            case 'lastRun':
              return timeAgo(row.lastRunAt);
            case 'nextRun':
              return timeAgo(row.nextRunAt);
            case 'enabled':
              return String(row.enabled !== false);
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'status') {
            const s = String(row.status ?? 'waiting');
            const [c, Icon] = statusMeta(s);
            return (
              <span className="inline-flex items-center gap-1.5" style={{ color: c }}>
                <Icon size={12} />
                <span className="text-[length:var(--text-label)] font-medium">{cap(s)}</span>
              </span>
            );
          }
          if (key === 'schedule') {
            return (
              <span
                className="font-[family-name:var(--font-mono)] text-[length:var(--text-label)] font-medium"
                style={{ color: ACCENT }}
              >
                {String(row.schedule ?? '')}
              </span>
            );
          }
          if (key === 'enabled') {
            return (
              <span onClick={(e) => e.stopPropagation()}>
                <Switch
                  checked={row.enabled !== false}
                  onCheckedChange={() => toggle(String(row.$id ?? ''))}
                />
              </span>
            );
          }
          return undefined;
        }}
        rowIcon={() => Clock}
        rowIconColor={(row) => statusMeta(String(row.status ?? 'waiting'))[0]}
        onDeleteRow={async (row) => {
          try {
            await api.delete(`/analytics/crons/${row.$id}`);
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
            options: ['ok', 'missed', 'failed', 'running', 'waiting'].map((v) => ({
              value: v,
              label: cap(v),
            })),
          },
        ]}
        filterValues={filters}
        onFiltersChange={setFilters}
        searchHint="Search monitors…"
        searchValue={search}
        onSearchChange={setSearch}
        itemLabel="monitors"
        emptyIcon={Clock}
        emptyTitle="No cron monitors"
        emptySubtitle="Get alerted when scheduled jobs miss their execution window"
        loading={query.isLoading}
        error={query.error}
        onRetry={query.refetch}
      />

      <AddCronDialog open={adding} onOpenChange={setAdding} onCreated={() => query.refetch()} />
    </div>
  );
}

function AddCronDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState('');
  const [schedule, setSchedule] = useState('0 * * * *');
  const [timezone, setTimezone] = useState('UTC');
  const [grace, setGrace] = useState('5');
  const [saving, setSaving] = useState(false);

  const reset = () => {
    setName('');
    setSchedule('0 * * * *');
    setTimezone('UTC');
    setGrace('5');
  };

  const submit = async () => {
    setSaving(true);
    try {
      await api.post('/analytics/crons', {
        name: name.trim(),
        schedule: schedule.trim(),
        timezone,
        gracePeriod: parseInt(grace, 10) || 5,
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
      title="Add cron monitor"
      submitLabel="Add"
      loading={saving}
      submitDisabled={!name.trim() || !schedule.trim()}
      onSubmit={submit}
    >
      <TextField label="Name" placeholder="Daily backup job" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      <TextField
        label="Schedule (cron expression)"
        placeholder="0 * * * *"
        hint='Use standard cron format — e.g. "0 9 * * 1-5" for weekdays at 9am'
        value={schedule}
        onChange={(e) => setSchedule(e.target.value)}
      />
      <SelectField
        label="Timezone"
        value={timezone}
        onChange={setTimezone}
        options={[
          'UTC',
          'America/New_York',
          'America/Los_Angeles',
          'Europe/London',
          'Europe/Berlin',
          'Asia/Tokyo',
          'Asia/Singapore',
          'Australia/Sydney',
        ].map((v) => ({ value: v, label: v }))}
      />
      <SelectField
        label="Grace period (minutes)"
        value={grace}
        onChange={setGrace}
        options={['1', '5', '10', '30', '60'].map((v) => ({ value: v, label: `${v} minutes` }))}
      />
    </FormDialog>
  );
}
