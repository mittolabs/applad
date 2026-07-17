import { useState } from 'react';
import { useParams } from 'react-router-dom';
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
  const { projectId } = useParams<{ projectId: string }>();
  const [tab, setTab] = useTabIndex(LIST_TABS);
  const [selected, setSelected] = useState<Row | null>(null);
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

  if (selected) {
    return (
      <SiteDetail
        site={selected}
        onChange={setSelected}
        onBack={() => setSelected(null)}
        onDeleted={() => {
          setSelected(null);
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
          onRowClick={setSelected}
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
