import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { useRoutedSelection } from '@/hooks/use-routed-selection';
import { DetailRoute } from '@/components/detail-route';
import { Flag as FlagIcon } from 'lucide-react';
import { api } from '@/api/client';
import { DataTable, type DataTableColumn, type Row } from '@/components/data-table';
import { Switch } from '@/components/ui/switch';
import { FormDialog, TextField, TextAreaField, SelectField } from '@/components/form-dialog';
import { useResourceList } from '@/hooks/use-resource-list';
import { FlagDetail, defaultValueHint } from './FlagDetail';
import { type Flag, FLAG_TYPES, FlagTypeBadge } from './flags-shared';

const COLUMNS: DataTableColumn[] = [
  { key: '$id', label: 'ID', flex: 3, defaultVisible: false },
  { key: 'key', label: 'Key', flex: 3 },
  { key: 'name', label: 'Name', flex: 3 },
  { key: 'type', label: 'Type', flex: 2 },
  { key: 'enabled', label: 'Enabled', flex: 2 },
];

export function FlagsPage() {
  const { projectId } = useParams();
  // Which record is open belongs in the address.
  const selection = useRoutedSelection('flags', 'flagId');
  const [creating, setCreating] = useState(false);
  const list = useResourceList<Flag>({ endpoint: '/flags', itemsKey: 'flags', scope: [projectId] });

  const toggleFlag = async (row: Row) => {
    const next = row.enabled !== true;
    await api.patch(`/flags/${String(row.$id)}/toggle`, { enabled: next });
    list.refetch();
  };

  // The backend /flags list returns all flags (no server-side filter), so
  // apply the Type/Status filters client-side — mirrors the Flutter console.
  const rows = list.rows.filter((r) => {
    const f = list.filters;
    if (f.type && String((r as Flag).type ?? 'boolean') !== f.type) return false;
    if (f.enabled) {
      const status = (r as Flag).enabled === true ? 'enabled' : 'disabled';
      if (status !== f.enabled) return false;
    }
    return true;
  });

  if (selection.id) {
    return (
      <DetailRoute endpoint="/flags" id={selection.id}>
        {(flag, refetch) => (
          <FlagDetail
            flag={flag as Flag}
            onBack={selection.clear}
            onChange={() => {
              refetch();
              list.refetch();
            }}
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
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Feature Flags</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Toggle features at runtime without redeploying your app
        </p>
      </div>

      <DataTable
        columns={COLUMNS}
        rows={rows}
        getCellValue={(row, key) => {
          switch (key) {
            case 'type':
              return String(row.type ?? 'boolean');
            case 'enabled':
              return row.enabled === true ? 'enabled' : 'disabled';
            default:
              return String(row[key] ?? '');
          }
        }}
        cellRender={(row, key) => {
          if (key === 'type') return <FlagTypeBadge type={String(row.type ?? 'boolean')} />;
          if (key === 'enabled') {
            return (
              <div onClick={(e) => e.stopPropagation()} className="inline-flex">
                <Switch checked={row.enabled === true} onCheckedChange={() => void toggleFlag(row)} />
              </div>
            );
          }
          return undefined;
        }}
        rowIcon={() => FlagIcon}
        rowIconColor={(row) => (row.enabled === true ? '#22C55E' : '#6B7280')}
        onRowClick={(row) => selection.select(String(row['$id'] ?? ''))}
        onDeleteRow={async (row) => {
          await api.delete(`/flags/${String(row.$id)}`);
          list.refetch();
        }}
        createLabel="Create flag"
        onCreate={() => setCreating(true)}
        searchHint="Search by key or name"
        searchValue={list.search}
        onSearchChange={list.setSearch}
        onSearch={list.runSearch}
        total={list.total}
        perPage={list.perPage}
        page={list.page}
        onPerPageChange={list.setPerPage}
        onPrev={list.prevPage}
        onNext={list.nextPage}
        itemLabel="flags"
        filters={[
          {
            key: 'type',
            label: 'Type',
            options: FLAG_TYPES.map((t) => ({ value: t, label: t })),
          },
          {
            key: 'enabled',
            label: 'Status',
            options: [
              { value: 'enabled', label: 'Enabled' },
              { value: 'disabled', label: 'Disabled' },
            ],
          },
        ]}
        filterValues={list.filters}
        onFiltersChange={list.setFilters}
        emptyIcon={FlagIcon}
        emptyTitle="No feature flags"
        emptySubtitle="Create a flag to start managing feature rollouts."
        loading={list.isLoading}
        error={list.error}
        onRetry={list.refetch}
      />

      {creating && (
        <CreateFlagDialog onClose={() => setCreating(false)} onCreated={() => list.refetch()} />
      )}
    </div>
  );
}

function CreateFlagDialog({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [key, setKey] = useState('');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [type, setType] = useState('boolean');
  const [defaultValue, setDefaultValue] = useState('');
  const [saving, setSaving] = useState(false);

  const save = async () => {
    setSaving(true);
    try {
      await api.post('/flags', {
        key: key.trim(),
        name: name.trim(),
        description: description.trim(),
        type,
        defaultValue: defaultValue.trim(),
      });
      onCreated();
      onClose();
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormDialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="Create Flag"
      submitLabel="Create"
      loading={saving}
      submitDisabled={!key.trim()}
      onSubmit={save}
    >
      <TextField label="Key" value={key} onChange={(e) => setKey(e.target.value)} placeholder="e.g. enable_dark_mode" autoFocus />
      <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Dark Mode" />
      <TextAreaField
        label="Description"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        placeholder="Controls whether dark mode is available"
        rows={3}
      />
      <SelectField
        label="Type"
        value={type}
        onChange={setType}
        options={FLAG_TYPES.map((t) => ({ value: t, label: t }))}
      />
      <TextField
        label="Default Value"
        value={defaultValue}
        onChange={(e) => setDefaultValue(e.target.value)}
        placeholder={defaultValueHint(type)}
      />
    </FormDialog>
  );
}
