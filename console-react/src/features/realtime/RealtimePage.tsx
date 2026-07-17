import { useParams } from 'react-router-dom';
import { PageTabs } from '@/components/page-tabs';
import { useTabIndex } from '@/hooks/use-tab-param';
import { OverviewTab } from './OverviewTab';
import { ChannelsTab } from './ChannelsTab';

const TABS = ['Overview', 'Channels'];

export function RealtimePage() {
  const { projectId } = useParams();
  const [tab, setTab] = useTabIndex(TABS);

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex flex-col gap-1">
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Realtime</h1>
        <p className="text-[length:var(--text-body)] text-text-secondary">
          Broadcast and subscribe to live events over WebSocket connections
        </p>
      </div>

      <PageTabs tabs={TABS} selected={tab} onChange={setTab} />

      <div>
        {tab === 0 && <OverviewTab projectId={projectId} />}
        {tab === 1 && <ChannelsTab projectId={projectId} />}
      </div>
    </div>
  );
}
