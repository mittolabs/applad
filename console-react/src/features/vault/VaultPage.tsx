import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { Plus, RefreshCw, ShieldCheck, ShieldOff } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { useResourceList } from '@/hooks/use-resource-list';
import { toast } from '@/components/toast';
import { Button } from '@/components/ui/button';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { ConfirmDialog } from '@/components/form-dialog';
import { CRED_TYPE_OPTIONS, typeColor, typeIcon } from './credentials';
import { ExpiryChip, TypeBadge } from './CredentialBadges';
import { CredentialFormDialog } from './CredentialFormDialog';
import { CredentialDetailDialog } from './CredentialDetailDialog';

const COLUMNS: DataTableColumn[] = [
  { key: '$id', label: 'ID', flex: 3, defaultVisible: false },
  { key: 'name', label: 'Name', flex: 3, sortable: true },
  { key: 'type', label: 'Type', flex: 2 },
  { key: 'description', label: 'Description', flex: 3 },
  { key: 'expiresAt', label: 'Expires', flex: 2 },
];

export function VaultPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const list = useResourceList<Row>({
    endpoint: '/credentials',
    itemsKey: 'credentials',
    scope: [projectId],
    defaultPerPage: 25,
  });

  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<Row | null>(null);
  const [detail, setDetail] = useState<Row | null>(null);
  const [rotateOpen, setRotateOpen] = useState(false);
  const [rotating, setRotating] = useState(false);

  const openCreate = () => {
    setEditing(null);
    setCreating(true);
  };

  const openEdit = (cred: Row) => {
    setCreating(false);
    setEditing(cred);
  };

  const rotate = async () => {
    setRotating(true);
    try {
      const res = await api.post('/credentials/rotate');
      const n = Number((res.data as Record<string, unknown>)['rotated'] ?? 0);
      toast.success(`Rotated ${n} credential${n === 1 ? '' : 's'}`);
      setRotateOpen(false);
      list.refetch();
    } catch (e) {
      toast.error(`Rotation failed: ${friendlyError(e)}`);
    } finally {
      setRotating(false);
    }
  };

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex items-center gap-2.5">
        <ShieldCheck size={20} style={{ color: 'var(--color-accent)' }} />
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Vault</h1>
      </div>

      <DataTable
        columns={COLUMNS}
        rows={list.rows}
        getCellValue={(row, key) => {
          switch (key) {
            case '$id':
              return String(row['$id'] ?? '');
            case 'name':
              return String(row['name'] ?? '');
            case 'type':
              return String(row['type'] ?? 'generic');
            case 'description':
              return String(row['description'] ?? '');
            case 'expiresAt':
              return String(row['expiresAt'] ?? '');
            default:
              return '';
          }
        }}
        cellRender={(row, key) => {
          if (key === 'type') return <TypeBadge type={String(row['type'] ?? 'generic')} />;
          if (key === 'expiresAt') {
            const exp = String(row['expiresAt'] ?? '');
            return exp ? <ExpiryChip expiresAt={exp} /> : null;
          }
          return undefined;
        }}
        rowIcon={(row) => typeIcon(String(row['type'] ?? 'generic'))}
        rowIconColor={(row) => typeColor(String(row['type'] ?? 'generic'))}
        onRowClick={(row) => setDetail(row)}
        onDeleteRow={async (row) => {
          try {
            await api.delete(`/credentials/${String(row['$id'])}`);
            toast.success('Credential deleted');
            list.refetch();
          } catch (e) {
            toast.error(friendlyError(e));
          }
        }}
        createWidget={
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => setRotateOpen(true)}>
              <RefreshCw size={14} />
              Rotate keys
            </Button>
            <Button size="sm" onClick={openCreate}>
              <Plus size={14} />
              Add credential
            </Button>
          </div>
        }
        searchHint="Search by name or type"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="credentials"
        filters={[{ key: 'type', label: 'Type', options: CRED_TYPE_OPTIONS }]}
        filterValues={list.filters}
        onFiltersChange={list.setFilters}
        emptyIcon={ShieldOff}
        emptyTitle="No credentials yet"
        emptySubtitle="Store API keys, database passwords, SSH keys and more."
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      <CredentialFormDialog
        open={creating || editing !== null}
        onOpenChange={(o) => {
          if (!o) {
            setCreating(false);
            setEditing(null);
          }
        }}
        existing={editing}
        onSaved={list.refetch}
      />

      <CredentialDetailDialog
        cred={detail}
        open={detail !== null}
        onOpenChange={(o) => !o && setDetail(null)}
        onEdit={openEdit}
      />

      <ConfirmDialog
        open={rotateOpen}
        onOpenChange={setRotateOpen}
        title="Rotate encryption keys"
        message="All credentials will be re-encrypted with the current CREDENTIALS_ENCRYPTION_KEY. This is safe and non-destructive. Continue?"
        confirmLabel="Rotate"
        destructive={false}
        loading={rotating}
        onConfirm={rotate}
      />
    </div>
  );
}
