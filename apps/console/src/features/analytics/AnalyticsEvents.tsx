import { useMemo, useState } from 'react';
import { BarChart3 } from 'lucide-react';
import { DataTable, type DataTableColumn } from '@/components/data-table';
import {
  ACCENT,
  asRows,
  timeAgo,
  useAnalyticsResource,
} from './analytics-shared';

/* AnalyticsEvents — the raw event stream. Filtering is client-side over the
 * page the API returned, which is honest about what is on screen: the count in
 * the footer is the number of rows fetched, not a total the server promised. */

const COLUMNS: DataTableColumn[] = [
  { key: 'event', label: 'Event', flex: 3 },
  { key: 'userId', label: 'User', flex: 2, sortable: false },
  { key: 'deviceType', label: 'Device', flex: 2 },
  { key: 'country', label: 'Country', flex: 2 },
  { key: 'url', label: 'URL', flex: 3, sortable: false, defaultVisible: false },
  { key: 'when', label: 'When', flex: 2, sortable: false },
];

export function AnalyticsEvents({ projectId }: { projectId?: string }) {
  const query = useAnalyticsResource('/analytics/events', projectId, { limit: 200 });
  const [search, setSearch] = useState('');
  const [filters, setFilters] = useState<Record<string, string | null>>({});

  const allEvents = asRows(query.data?.events);

  const deviceOptions = useMemo(() => {
    const seen = new Set<string>();
    for (const e of allEvents) {
      const d = String(e.deviceType ?? '');
      if (d) seen.add(d);
    }
    return [...seen].sort().map((v) => ({ value: v, label: v }));
  }, [allEvents]);

  const rows = useMemo(() => {
    const q = search.trim().toLowerCase();
    return allEvents.filter((e) => {
      if (
        q &&
        !String(e.event ?? '').toLowerCase().includes(q) &&
        !String(e.userId ?? '').toLowerCase().includes(q)
      )
        return false;
      if (filters.deviceType && String(e.deviceType ?? '') !== filters.deviceType) return false;
      return true;
    });
  }, [allEvents, search, filters]);

  return (
    <div className="px-6 md:px-8">
      <DataTable
        columns={COLUMNS}
        rows={rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'event':
              return String(row.event ?? '');
            case 'userId':
              return String(row.userId ?? '—');
            case 'deviceType':
              return String(row.deviceType ?? '—');
            case 'country':
              return String(row.country ?? '—');
            case 'url':
              return String(row.url ?? '—');
            case 'when':
              return timeAgo(row.$createdAt);
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'event') {
            return (
              <span
                className="font-[family-name:var(--font-mono)] text-[length:var(--text-label)] font-medium"
                style={{ color: ACCENT }}
              >
                {String(row.event ?? '')}
              </span>
            );
          }
          return undefined;
        }}
        rowIcon={() => BarChart3}
        rowIconColor={() => ACCENT}
        filters={
          deviceOptions.length > 0
            ? [{ key: 'deviceType', label: 'Device', options: deviceOptions }]
            : undefined
        }
        filterValues={filters}
        onFiltersChange={setFilters}
        searchHint="Search events…"
        searchValue={search}
        onSearchChange={setSearch}
        itemLabel="events"
        emptyIcon={BarChart3}
        emptyTitle="No events yet"
        emptySubtitle="Call analytics.trackEvent() from your app and the events land here."
        loading={query.isLoading}
        error={query.error}
        onRetry={query.refetch}
      />
    </div>
  );
}
