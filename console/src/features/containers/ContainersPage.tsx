import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useRoutedSelection } from '@/hooks/use-routed-selection';
import { DetailRoute } from '@/components/detail-route';
import { Box } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { StatusChip } from '@/components/status-chip';
import { type DataTableColumn, type Row } from '@/components/data-table';
import { DeployCreateEntry, type CreateEntryResult } from '@/components/deploy-create-entry';
import { FormDialog, TextField } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import { TargetList } from '../deploy-shared/TargetList';
import { shortDate } from '../deploy-shared/format';
import { ContainerDetail } from './ContainerDetail';

const COLUMNS: DataTableColumn[] = [
  { key: 'name', label: 'Name', flex: 4, sortable: true },
  { key: 'registryUrl', label: 'Registry', flex: 3 },
  { key: 'tagStrategy', label: 'Tag', flex: 1 },
  { key: 'status', label: 'Status', flex: 2 },
  { key: 'updatedAt', label: 'Updated', flex: 2 },
];

interface ContainerPrefill {
  name: string;
  registryUrl: string;
  dockerfile: string;
  source: string;
  templateId?: string;
  repository?: string;
}

export function ContainersPage() {
  const { projectId } = useParams<{ projectId: string }>();
  // Which record is open belongs in the address.
  const selection = useRoutedSelection('containers', 'containerId');
  const [entryOpen, setEntryOpen] = useState(false);
  const [prefill, setPrefill] = useState<ContainerPrefill | null>(null);

  const list = useResourceList<Row>({
    endpoint: '/deploy/targets',
    itemsKey: 'targets',
    params: { type: 'container' },
    scope: [projectId],
  });

  const onEntryResult = (r: CreateEntryResult) => {
    if (r.choice === 'template' && r.templateConfig) {
      const t = r.templateConfig;
      setPrefill({
        name: String(t['name'] ?? ''),
        registryUrl: String(t['registryUrl'] ?? ''),
        dockerfile: String(t['dockerfile'] ?? 'Dockerfile'),
        source: 'template',
        templateId: t['$id'] as string | undefined,
      });
    } else if (r.choice === 'repository' && r.repoConfig) {
      const c = r.repoConfig;
      setPrefill({
        name: String(c['name'] ?? ''),
        registryUrl: '',
        dockerfile: 'Dockerfile',
        source: 'git',
        repository: String(c['cloneUrl'] ?? c['url'] ?? ''),
      });
    } else {
      setPrefill({ name: '', registryUrl: '', dockerfile: 'Dockerfile', source: 'upload' });
    }
  };

  if (selection.id) {
    return (
      <DetailRoute endpoint="/deploy/targets" id={selection.id}>
        {(container) => (
          <ContainerDetail
            container={container}
            onBack={selection.clear}
            onDeleted={() => {
              selection.clear();
              list.refetch();
            }}
          />
        )}
      </DetailRoute>
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Containers</h1>

      <TargetList
        list={list}
        columns={COLUMNS}
        getCellValue={(row, key) => {
          switch (key) {
            case 'name':
              return String(row['name'] ?? '');
            case 'registryUrl':
              return String(row['registryUrl'] ?? 'No registry configured');
            case 'tagStrategy':
              return String(row['tagStrategy'] ?? 'latest');
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
        rowIcon={() => Box}
        onRowClick={(row) => selection.select(String(row['$id'] ?? row['id'] ?? ''))}
        onDeleted={() => list.refetch()}
        createLabel="Create container"
        onCreate={() => setEntryOpen(true)}
        itemLabel="containers"
        searchHint="Search containers…"
        emptyIcon={Box}
        emptyTitle="No containers yet"
        emptySubtitle="Deploy your first Docker container."
      />

      <DeployCreateEntry
        open={entryOpen}
        onOpenChange={setEntryOpen}
        category="containers"
        title="Create Container"
        subtitle="Choose how to get started"
        onResult={onEntryResult}
      />

      <CreateContainerDialog
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

function CreateContainerDialog({
  open,
  onOpenChange,
  prefill,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  prefill: ContainerPrefill | null;
  onCreated: () => void;
}) {
  const [name, setName] = useState('');
  const [registryUrl, setRegistryUrl] = useState('');
  const [dockerfile, setDockerfile] = useState('Dockerfile');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open && prefill) {
      setName(prefill.name);
      setRegistryUrl(prefill.registryUrl);
      setDockerfile(prefill.dockerfile);
      setSaving(false);
    }
  }, [open, prefill]);

  const submit = async () => {
    if (!prefill) return;
    setSaving(true);
    try {
      await api.post('/deploy/targets', {
        name: name.trim(),
        type: 'container',
        registryUrl: registryUrl.trim(),
        dockerfile: dockerfile.trim(),
        tagStrategy: 'latest',
        source: prefill.source,
        ...(prefill.templateId ? { templateId: prefill.templateId } : {}),
        ...(prefill.repository ? { repository: prefill.repository } : {}),
      });
      onOpenChange(false);
      onCreated();
    } catch (e) {
      toast.error(friendlyError(e));
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Create Container"
      subtitle="Deploy a Docker container"
      submitLabel="Create"
      loading={saving}
      submitDisabled={!name.trim()}
      onSubmit={submit}
    >
      <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} placeholder="my-api" autoFocus />
      <TextField label="Registry URL" value={registryUrl} onChange={(e) => setRegistryUrl(e.target.value)} placeholder="ghcr.io/myorg/myimage" />
      <TextField label="Dockerfile" value={dockerfile} onChange={(e) => setDockerfile(e.target.value)} placeholder="Dockerfile" />
    </FormDialog>
  );
}
