import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Database } from 'lucide-react';
import { api } from '@/api/client';
import { PageTabs } from '@/components/page-tabs';
import { Button } from '@/components/ui/button';
import { IdText } from '@/components/id-text';
import { ConfirmDialog } from '@/components/form-dialog';
import { BackHeader, type Json } from './shared';
import { TablesPanel } from './TablesPanel';
import { SqlConsole } from './SqlConsole';

const TABS = ['Tables', 'SQL', 'Usage', 'Settings'];

export function DatabaseDetail({
  dbId,
  dbName,
  onBack,
  onSelectTable,
}: {
  dbId: string;
  dbName: string;
  onBack: () => void;
  onSelectTable: (id: string, name: string) => void;
}) {
  const [tab, setTab] = useState(0);

  const tablesQuery = useQuery({
    queryKey: ['db-tables', dbId],
    queryFn: async () => {
      const res = await api.get(`/databases/${dbId}/tables`);
      return (res.data as { tables?: Json[] }).tables ?? [];
    },
  });

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <BackHeader title={dbName} subtitle={dbId} icon={Database} onBack={onBack} />
      <PageTabs tabs={TABS} selected={tab} onChange={setTab} />

      {tab === 0 && <TablesPanel dbId={dbId} onSelectTable={onSelectTable} />}
      {tab === 1 && (
        <SqlConsole dbId={dbId} tables={tablesQuery.data ?? []} onOpenTable={onSelectTable} />
      )}
      {tab === 2 && (
        <div className="flex h-48 items-center justify-center rounded-[var(--radius-10)] border border-border bg-surface text-[length:var(--text-body)] text-text-subtle">
          Usage coming soon
        </div>
      )}
      {tab === 3 && <DatabaseSettings dbId={dbId} onDeleted={onBack} />}
    </div>
  );
}

function DatabaseSettings({ dbId, onDeleted }: { dbId: string; onDeleted: () => void }) {
  const qc = useQueryClient();
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const del = async () => {
    setDeleting(true);
    try {
      await api.delete(`/databases/${dbId}`);
      qc.invalidateQueries({ queryKey: ['/databases'] });
      setConfirming(false);
      onDeleted();
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="flex max-w-2xl flex-col gap-4">
      <div className="rounded-[var(--radius-10)] border border-border bg-surface p-5">
        <div className="text-[length:var(--text-control)] font-medium text-text-primary">
          Database settings
        </div>
        <div className="mt-3 flex items-center gap-2 text-[length:var(--text-body)] text-text-secondary">
          <span className="text-text-muted">Database ID</span>
          <IdText id={dbId} />
        </div>
      </div>

      <div className="rounded-[var(--radius-10)] border border-[color-mix(in_srgb,var(--color-danger)_40%,var(--border))] bg-surface p-5">
        <div className="text-[length:var(--text-control)] font-medium text-[var(--status-danger)]">
          Delete database
        </div>
        <div className="mt-1 text-[length:var(--text-body)] text-text-muted">
          All tables and data will be permanently deleted.
        </div>
        <Button variant="destructive" className="mt-3" onClick={() => setConfirming(true)}>
          Delete database
        </Button>
      </div>

      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title="Delete database"
        message="All tables and data in this database will be permanently deleted."
        confirmLabel="Delete"
        loading={deleting}
        onConfirm={del}
      />
    </div>
  );
}
