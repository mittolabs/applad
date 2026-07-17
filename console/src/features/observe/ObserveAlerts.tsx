import { useMemo, useState } from 'react';
import { AlertTriangle, Bell } from 'lucide-react';
import { DataTable, type DataTableColumn } from '@/components/data-table';
import { FormDialog, FormField, SelectField, TextField } from '@/components/form-dialog';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import {
  OB_ACCENT,
  OB_ORANGE,
  OB_RED,
  ObSectionTitle,
  asRows,
  cap,
  obTimeAgo,
  tint,
  useObserveResource,
} from './observe-shared';

/* ObserveAlerts — ports observe_alerts.dart. Alert-rule list with a firing
 * incidents banner + a create-rule dialog. */

const COLUMNS: DataTableColumn[] = [
  { key: 'severity', label: 'Severity', flex: 2 },
  { key: 'name', label: 'Name', flex: 3 },
  { key: 'condition', label: 'Condition', flex: 4 },
  { key: 'time_window', label: 'Window', flex: 2, sortable: false },
  { key: 'channel', label: 'Channel', flex: 2, sortable: false },
  { key: 'enabled', label: 'Enabled', flex: 1 },
  { key: 'lastFired', label: 'Last fired', flex: 2, sortable: false },
];

function severityColor(s: string): string {
  return s === 'critical' ? OB_RED : s === 'warning' ? OB_ORANGE : OB_ACCENT;
}

function conditionStr(row: Record<string, unknown>): string {
  const metric = String(row.metric ?? '');
  const op = String(row.operator ?? 'gt');
  const thresh = row.threshold;
  const opStr = op === 'gt' ? '>' : op === 'lt' ? '<' : op === 'gte' ? '≥' : op === 'lte' ? '≤' : op;
  return `${metric} ${opStr} ${thresh}`;
}

export function ObserveAlerts({ projectId }: { projectId?: string }) {
  const query = useObserveResource('/observe/alerts', projectId);
  const [search, setSearch] = useState('');
  const [filters, setFilters] = useState<Record<string, string | null>>({});
  const [creating, setCreating] = useState(false);

  const allRules = asRows(query.data?.rules);
  const incidents = asRows(query.data?.incidents);

  const rows = useMemo(() => {
    const q = search.trim().toLowerCase();
    return allRules.filter((r) => {
      if (
        q &&
        !String(r.name ?? '').toLowerCase().includes(q) &&
        !String(r.metric ?? '').toLowerCase().includes(q)
      )
        return false;
      if (filters.severity && String(r.severity ?? 'warning') !== filters.severity) return false;
      return true;
    });
  }, [allRules, search, filters]);

  const toggle = async (id: string) => {
    try {
      await api.patch(`/observe/alerts/${id}/toggle`);
      await query.refetch();
    } catch (e) {
      toast.error(friendlyError(e));
    }
  };

  return (
    <div className="flex flex-col gap-3">
      {incidents.length > 0 && (
        <div className="px-6 md:px-8">
          <div className="flex items-center gap-2">
            <ObSectionTitle title="Firing Now" />
            <span
              className="rounded-full px-[7px] py-0.5 text-[length:var(--text-caption)] font-bold text-white"
              style={{ backgroundColor: OB_RED }}
            >
              {incidents.length}
            </span>
          </div>
          <div className="mt-2 flex flex-col gap-1.5">
            {incidents.map((inc, i) => (
              <IncidentCard key={i} incident={inc} />
            ))}
          </div>
        </div>
      )}

      <div className="px-6 md:px-8">
        <DataTable
          columns={COLUMNS}
          rows={rows}
          getCellValue={(row, key) => {
            switch (key) {
              case 'severity':
                return String(row.severity ?? 'warning');
              case 'name':
                return String(row.name ?? '');
              case 'condition':
                return conditionStr(row);
              case 'time_window':
                return String(row.time_window ?? row.window ?? '');
              case 'channel':
                return String(row.channel ?? '');
              case 'enabled':
                return String(row.enabled !== false);
              case 'lastFired':
                return obTimeAgo(row.lastFired);
              default:
                return '';
            }
          }}
          cellRender={(row, key) => {
            if (key === 'severity') {
              const s = String(row.severity ?? 'warning');
              const c = severityColor(s);
              return (
                <span className="inline-flex items-center gap-1.5" style={{ color: c }}>
                  <span className="h-2 w-2 rounded-full" style={{ backgroundColor: c }} />
                  <span className="text-[length:var(--text-label)] font-medium">{cap(s)}</span>
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
          rowIcon={() => Bell}
          rowIconColor={(row) => severityColor(String(row.severity ?? 'warning'))}
          onDeleteRow={async (row) => {
            try {
              await api.delete(`/observe/alerts/${row.$id}`);
              await query.refetch();
            } catch (e) {
              toast.error(friendlyError(e));
            }
          }}
          createLabel="Create rule"
          onCreate={() => setCreating(true)}
          filters={[
            {
              key: 'severity',
              label: 'Severity',
              options: ['info', 'warning', 'critical'].map((v) => ({ value: v, label: cap(v) })),
            },
          ]}
          filterValues={filters}
          onFiltersChange={setFilters}
          searchHint="Search rules…"
          searchValue={search}
          onSearchChange={setSearch}
          itemLabel="rules"
          emptyIcon={Bell}
          emptyTitle="No alert rules"
          emptySubtitle="Create rules to get notified when metrics exceed thresholds"
          loading={query.isLoading}
          error={query.error}
          onRetry={query.refetch}
        />
      </div>

      <CreateRuleDialog open={creating} onOpenChange={setCreating} onCreated={() => query.refetch()} />
    </div>
  );
}

function IncidentCard({ incident }: { incident: Record<string, unknown> }) {
  const severity = String(incident.severity ?? 'warning');
  const name = String(incident.ruleName ?? 'Alert');
  const value = incident.value;
  const firedAt = obTimeAgo(incident.firedAt);
  const sc = severityColor(severity);
  return (
    <div
      className="flex items-center gap-3 rounded-[var(--radius)] border p-3.5"
      style={{ backgroundColor: tint(sc, 8), borderColor: tint(sc, 30) }}
    >
      <AlertTriangle size={16} style={{ color: sc }} />
      <div className="flex flex-1 flex-col">
        <span className="text-[length:var(--text-body)] font-semibold text-text-primary">{name}</span>
        <span className="text-[length:var(--text-caption)] text-text-secondary">
          Value: {String(value)} &nbsp;•&nbsp; Fired {firedAt}
        </span>
      </div>
      <span
        className="rounded-[var(--radius-sm)] px-2 py-[3px] text-[length:var(--text-caption)] font-bold"
        style={{ color: sc, backgroundColor: tint(sc, 15) }}
      >
        {severity.toUpperCase()}
      </span>
    </div>
  );
}

function CreateRuleDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState('');
  const [metric, setMetric] = useState('error_rate');
  const [operator, setOperator] = useState('gt');
  const [threshold, setThreshold] = useState('5');
  const [win, setWin] = useState('5m');
  const [severity, setSeverity] = useState('warning');
  const [channel, setChannel] = useState('email');
  const [saving, setSaving] = useState(false);

  const reset = () => {
    setName('');
    setMetric('error_rate');
    setOperator('gt');
    setThreshold('5');
    setWin('5m');
    setSeverity('warning');
    setChannel('email');
  };

  const submit = async () => {
    setSaving(true);
    try {
      await api.post('/observe/alerts', {
        name: name.trim(),
        metric,
        operator,
        threshold: parseFloat(threshold) || 5.0,
        window: win,
        severity,
        channel,
        enabled: true,
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
      title="Create alert rule"
      submitLabel="Create"
      loading={saving}
      submitDisabled={!name.trim()}
      onSubmit={submit}
    >
      <TextField label="Rule name" placeholder="High error rate" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      <SelectField
        label="Metric"
        value={metric}
        onChange={setMetric}
        options={[
          { value: 'error_rate', label: 'Error rate (%)' },
          { value: 'p95_latency', label: 'P95 latency (ms)' },
          { value: 'p99_latency', label: 'P99 latency (ms)' },
          { value: 'request_rate', label: 'Request rate (req/s)' },
          { value: 'uptime', label: 'Uptime (%)' },
          { value: 'log_errors', label: 'Log errors / min' },
          { value: 'lcp', label: 'LCP (ms)' },
          { value: 'fid', label: 'FID (ms)' },
          { value: 'cls', label: 'CLS score' },
          { value: 'apdex', label: 'Apdex score' },
        ]}
      />
      <div className="flex gap-3">
        <div className="flex-1">
          <SelectField
            label="Condition"
            value={operator}
            onChange={setOperator}
            options={[
              { value: 'gt', label: 'Above' },
              { value: 'lt', label: 'Below' },
              { value: 'gte', label: 'At or above' },
              { value: 'lte', label: 'At or below' },
            ]}
          />
        </div>
        <div className="flex-1">
          <FormField label="Threshold">
            <Input
              type="number"
              placeholder="5"
              value={threshold}
              onChange={(e) => setThreshold(e.target.value)}
            />
          </FormField>
        </div>
      </div>
      <SelectField
        label="Time window"
        value={win}
        onChange={setWin}
        options={[
          { value: '1m', label: '1 minute' },
          { value: '5m', label: '5 minutes' },
          { value: '15m', label: '15 minutes' },
          { value: '30m', label: '30 minutes' },
          { value: '1h', label: '1 hour' },
          { value: '24h', label: '24 hours' },
        ]}
      />
      <SelectField
        label="Severity"
        value={severity}
        onChange={setSeverity}
        options={['info', 'warning', 'critical'].map((v) => ({ value: v, label: cap(v) }))}
      />
      <SelectField
        label="Notify via"
        value={channel}
        onChange={setChannel}
        options={[
          { value: 'email', label: 'Email' },
          { value: 'slack', label: 'Slack' },
          { value: 'webhook', label: 'Webhook' },
          { value: 'pagerduty', label: 'PagerDuty' },
          { value: 'opsgenie', label: 'Opsgenie' },
        ]}
      />
    </FormDialog>
  );
}
