import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/api/client';
import { ErrorState } from '@/components/error-state';
import { Activity, Globe, HardDrive, Plus, Timer } from 'lucide-react';
import { useResourceList } from '@/hooks/use-resource-list';
import { useTabIndex } from '@/hooks/use-tab-param';
import { PageTabs } from '@/components/page-tabs';
import { StatusChip } from '@/components/status-chip';
import { type DataTableColumn, type Row } from '@/components/data-table';
import { DeployCreateEntry, type CreateEntryResult } from '@/components/deploy-create-entry';
import { TargetList } from '../deploy-shared/TargetList';
import { frameworkById } from '../deploy-shared/frameworks';
import { shortDate } from '../deploy-shared/format';
import { StatCard } from './SiteDetail';
import { CreateSiteDialog, type SitePrefill } from './CreateSiteDialog';
import { SiteDetail } from './SiteDetail';

const LIST_TABS = ['Sites', 'Usage'];

const COLUMNS: DataTableColumn[] = [
  { key: 'name', label: 'Name', flex: 4, sortable: true },
  { key: 'framework', label: 'Framework', flex: 2 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'updatedAt', label: 'Updated', flex: 2 },
];

export function SitesPage() {
  const { projectId, siteId } = useParams<{ projectId: string; siteId: string }>();
  const navigate = useNavigate();
  const [tab, setTab] = useTabIndex(LIST_TABS);

  /*
   * Which site is open lives in the address, not in component state. Holding
   * it in state meant a refresh, a shared link or the back button all landed
   * on the list with no way back to where somebody was.
   */
  const open = (row: Row | null) => {
    const id = row ? String(row['$id'] ?? row['id'] ?? '') : '';
    navigate(id ? `/project/${projectId}/sites/${id}` : `/project/${projectId}/sites`);
  };
  const [entryOpen, setEntryOpen] = useState(false);
  const [prefill, setPrefill] = useState<SitePrefill | null>(null);

  const list = useResourceList<Row>({
    endpoint: '/deploy/targets',
    itemsKey: 'targets',
    params: { type: 'web' },
    scope: [projectId],
  });

  const onEntryResult = (r: CreateEntryResult) => {
    if (r.choice === 'template' && r.templateConfig) {
      const t = r.templateConfig;
      setPrefill({
        name: String(t['name'] ?? ''),
        framework: String(t['framework'] ?? 'nextjs'),
        repository: String(t['repository'] ?? ''),
        branch: 'main',
        sourceType: 'git',
        templateId: t['$id'] as string | undefined,
      });
    } else if (r.choice === 'repository' && r.repoConfig) {
      const c = r.repoConfig;
      setPrefill({
        name: String(c['name'] ?? ''),
        framework: 'nextjs',
        repository: String(c['cloneUrl'] ?? c['url'] ?? ''),
        branch: String(c['defaultBranch'] ?? 'main'),
        sourceType: 'git',
      });
    } else {
      setPrefill({ name: '', framework: 'nextjs', repository: '', branch: 'main', sourceType: 'upload' });
    }
  };

  if (siteId) {
    // Fetched by id rather than looked up in the list: on a refresh the list
    // may not have loaded, and the site may not be on the current page.
    return (
      <SiteDetailRoute
        siteId={siteId}
        onBack={() => open(null)}
        onDeleted={() => {
          open(null);
          list.refetch();
        }}
      />
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Sites</h1>

      <PageTabs tabs={LIST_TABS} selected={tab} onChange={setTab} />

      {tab === 0 ? (
        <TargetList
          list={list}
          columns={COLUMNS}
          getCellValue={(row, key) => {
            switch (key) {
              case 'name':
                return String(row['name'] ?? '');
              case 'framework':
                return frameworkById(String(row['framework'] ?? 'static')).label;
              case 'status':
                return String(row['status'] ?? 'active');
              case 'updatedAt':
                return shortDate(row['updatedAt'] ?? row['$updatedAt']);
              default:
                return '';
            }
          }}
          cellRender={(row, key) =>
            key === 'status' ? <StatusChip label={String(row['status'] ?? 'active')} /> : undefined
          }
          rowIcon={() => Globe}
          onRowClick={open}
          onDeleted={() => list.refetch()}
          createLabel="Create site"
          onCreate={() => setEntryOpen(true)}
          itemLabel="sites"
          searchHint="Search sites…"
          emptyIcon={Plus}
          emptyTitle="No sites yet"
          emptySubtitle="Deploy a web application from a Git repository or upload."
        />
      ) : (
        <UsageTab />
      )}

      <DeployCreateEntry
        open={entryOpen}
        onOpenChange={setEntryOpen}
        category="sites"
        title="Create Site"
        subtitle="Choose how to get started"
        onResult={onEntryResult}
      />

      <CreateSiteDialog
        open={prefill !== null}
        onOpenChange={(o) => !o && setPrefill(null)}
        prefill={prefill}
        onCreated={() => {
          setPrefill(null);
          list.refetch();
        }}
      />
    </div>
  );
}

function UsageTab() {
  return (
    <div className="flex flex-col gap-5">
      <p className="text-[length:var(--text-body)] text-text-muted">
        Aggregate usage across all sites in this project.
      </p>
      <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
        <StatCard icon={Activity} label="Total requests" value="--" />
        <StatCard icon={HardDrive} label="Bandwidth" value="--" />
        <StatCard icon={Timer} label="Build minutes" value="--" />
      </div>
    </div>
  );
}

/*
 * Resolves a site from the address.
 *
 * SiteDetail wants the whole record, and the address carries only an id, so
 * it is fetched here. That is what makes a refresh, a bookmark or a shared
 * link land on the site somebody was actually looking at.
 */
function SiteDetailRoute({
  siteId,
  onBack,
  onDeleted,
}: {
  siteId: string;
  onBack: () => void;
  onDeleted: () => void;
}) {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['site', siteId],
    queryFn: async () => (await api.get(`/deploy/targets/${siteId}`)).data as Row,
  });

  if (isLoading) {
    return <div className="p-6 text-[length:var(--text-body)] text-text-muted md:p-8">Loading...</div>;
  }
  if (error || !data) {
    return (
      <div className="p-6 md:p-8">
        <ErrorState error={error} onRetry={() => refetch()} />
      </div>
    );
  }

  return <SiteDetail site={data} onChange={() => refetch()} onBack={onBack} onDeleted={onDeleted} />;
}
