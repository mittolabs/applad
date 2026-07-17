import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { PageTabs } from '@/components/page-tabs';
import { useTabIndex } from '@/hooks/use-tab-param';
import { DatabaseList, DatabaseUsageTab } from './DatabaseList';
import { DatabaseDetail } from './DatabaseDetail';
import { TableDetail } from './TableDetail';

/* Databases — 3-level: databases list → database detail (Tables/SQL/Usage/
 * Settings) → table detail (Rows/Columns/Indexes/Relationships/Settings).
 * Selection state lives here; each level renders its own padded container. */
export function DatabasesPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [db, setDb] = useState<{ id: string; name: string } | null>(null);
  const [table, setTable] = useState<{ id: string; name: string } | null>(null);
  const [tab, setTab] = useTabIndex(['Databases', 'Usage']);

  if (db && table) {
    return (
      <TableDetail
        dbId={db.id}
        tableId={table.id}
        tableName={table.name}
        onBack={() => setTable(null)}
      />
    );
  }

  if (db) {
    return (
      <DatabaseDetail
        dbId={db.id}
        dbName={db.name}
        onBack={() => setDb(null)}
        onSelectTable={(id, name) => setTable({ id, name })}
      />
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Databases</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Create and query structured data with real-time sync and row-level security
        </p>
      </div>
      <PageTabs tabs={['Databases', 'Usage']} selected={tab} onChange={setTab} />
      {tab === 0 && (
        <DatabaseList projectId={projectId} onSelectDb={(id, name) => setDb({ id, name })} />
      )}
      {tab === 1 && <DatabaseUsageTab />}
    </div>
  );
}
