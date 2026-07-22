import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Container as ContainerIcon, Trash2 } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/empty-state';
import { ErrorState } from '@/components/error-state';
import { ConfirmDialog } from '@/components/form-dialog';
import { toast } from '@/components/toast';
import type { Row } from '@/components/data-table';
import { TargetDetailScaffold } from '../deploy-shared/TargetDetailScaffold';
import { DeploymentsPanel } from '../deploy-shared/DeploymentsPanel';
import { asNumber, rowId } from '../deploy-shared/format';

const DETAIL_TABS = ['Overview', 'Images', 'Releases', 'Settings'];

export function ContainerDetail({
  container,
  onBack,
  onDeleted,
}: {
  container: Row;
  onBack: () => void;
  onDeleted: () => void;
}) {
  const [tab, setTab] = useState(0);
  const id = rowId(container);
  const name = String(container['name'] ?? '');

  return (
    <TargetDetailScaffold
      backLabel="Containers"
      onBack={onBack}
      name={name}
      id={id}
      tabs={DETAIL_TABS}
      tab={tab}
      onTab={setTab}
    >
      {tab === 0 && <OverviewTab t={container} />}
      {tab === 1 && <ImagesTab id={id} />}
      {tab === 2 && <DeploymentsPanel targetId={id} showMetrics={false} showTrigger={false} />}
      {tab === 3 && <SettingsTab t={container} id={id} onDeleted={onDeleted} />}
    </TargetDetailScaffold>
  );
}

function OverviewTab({ t }: { t: Row }) {
  const cards: [string, string][] = [
    ['Registry', String(t['registryUrl'] ?? '—')],
    ['Dockerfile', String(t['dockerfile'] ?? 'Dockerfile')],
    ['Tag Strategy', String(t['tagStrategy'] ?? 'latest')],
    ['Runtime', String(t['runtime'] ?? '—')],
    ['Type', String(t['type'] ?? 'container')],
  ];
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {cards.map(([label, value]) => (
        <div key={label} className="rounded-[var(--radius)] border border-border bg-surface p-4">
          <div className="text-[length:var(--text-label)] text-text-subtle">{label}</div>
          <div className="mt-1.5 text-[length:var(--text-control)] font-medium text-text-primary">{value}</div>
        </div>
      ))}
    </div>
  );
}

function ImagesTab({ id }: { id: string }) {
  const images = useQuery({
    queryKey: ['container-images', id],
    queryFn: async () => (await api.get(`/deploy/targets/${id}/images`)).data as Record<string, unknown>,
  });
  const rows = (images.data?.['images'] as Row[] | undefined) ?? [];

  if (images.isLoading) {
    return <div className="py-10 text-center text-[length:var(--text-body)] text-text-muted">Loading…</div>;
  }
  if (images.error) return <ErrorState error={images.error} onRetry={() => images.refetch()} />;
  if (rows.length === 0) return <EmptyState icon={ContainerIcon} title="No images pushed yet" />;

  return (
    <div className="flex flex-col gap-2">
      {rows.map((img) => (
        <ImageRow
          key={rowId(img) || `${String(img['repository'])}:${String(img['tag'])}`}
          id={id}
          img={img}
          onRefetch={() => images.refetch()}
        />
      ))}
    </div>
  );
}

function ImageRow({ id, img, onRefetch }: { id: string; img: Row; onRefetch: () => void }) {
  const imageId = rowId(img);
  const ref = `${String(img['repository'] ?? '')}:${String(img['tag'] ?? 'latest')}`;
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const del = async () => {
    setDeleting(true);
    try {
      await api.delete(`/deploy/targets/${id}/images/${imageId}`);
      setConfirmDelete(false);
      onRefetch();
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="flex items-center gap-3 rounded-[var(--radius)] border border-border bg-surface px-3.5 py-3">
      <ContainerIcon size={16} className="text-[var(--color-accent)]" />
      <span className="flex-1 truncate font-[family-name:var(--font-mono)] text-[length:var(--text-body)] text-text-primary">
        {ref}
      </span>
      <span className="text-[length:var(--text-caption)] text-text-subtle">{String(img['platform'] ?? '')}</span>
      <span className="text-[length:var(--text-caption)] text-text-subtle">
        {(asNumber(img['sizeBytes']) / 1_048_576).toFixed(1)} MB
      </span>
      <button
        type="button"
        onClick={() => setConfirmDelete(true)}
        className="rounded-[var(--radius-6)] p-1.5 text-text-muted transition-colors hover:bg-fill hover:text-[var(--color-danger)]"
        aria-label={`Delete ${ref}`}
      >
        <Trash2 size={14} />
      </button>
      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Delete image"
        message={`Delete ${ref} from the registry? This cannot be undone.`}
        loading={deleting}
        onConfirm={del}
      />
    </div>
  );
}

function SettingsTab({ t, id, onDeleted }: { t: Row; id: string; onDeleted: () => void }) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const rows: [string, string][] = [
    ['Name', String(t['name'] ?? '')],
    ['Registry URL', String(t['registryUrl'] ?? '—')],
    ['Dockerfile', String(t['dockerfile'] ?? 'Dockerfile')],
    ['Tag Strategy', String(t['tagStrategy'] ?? 'latest')],
  ];

  const del = async () => {
    setDeleting(true);
    try {
      await api.delete(`/deploy/targets/${id}`);
      onDeleted();
    } catch (e) {
      toast.error(friendlyError(e));
      setDeleting(false);
    }
  };

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div className="text-[length:var(--text-title)] font-semibold text-text-primary">Container Settings</div>

      <div className="flex flex-col gap-4">
        {rows.map(([label, value]) => (
          <div key={label} className="flex gap-3">
            <span className="w-36 shrink-0 text-[length:var(--text-body)] text-text-subtle">{label}</span>
            <span className="min-w-0 flex-1 truncate text-[length:var(--text-body)] text-text-primary">{value}</span>
          </div>
        ))}
      </div>

      <div className="rounded-[var(--radius)] border border-[color-mix(in_srgb,var(--color-danger)_30%,transparent)] bg-surface p-5">
        <div className="text-[length:var(--text-control)] font-semibold text-[var(--color-danger)]">Danger zone</div>
        <div className="mt-2 text-[length:var(--text-body)] text-text-subtle">
          Delete this container target and all its images.
        </div>
        <Button variant="outline" className="mt-3" onClick={() => setConfirmDelete(true)}>
          Delete container
        </Button>
      </div>

      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Delete container"
        message="Delete this container target and all its images. This action cannot be undone."
        loading={deleting}
        onConfirm={del}
      />
    </div>
  );
}
