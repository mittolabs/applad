import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { PageTabs } from '@/components/page-tabs';
import { useTabIndex } from '@/hooks/use-tab-param';
import { MessagesTab } from './MessagesTab';
import { TopicsTab } from './TopicsTab';
import { TemplatesTab } from './TemplatesTab';
import { ProvidersTab } from './ProvidersTab';
import { CreateMessageView } from './CreateMessageView';
import { MessageDetail, type MessageRow } from './MessageDetail';
import type { MsgType } from './shared';

const TABS = ['Messages', 'Topics', 'Templates', 'Providers'];

export function MessagingPage() {
  const { projectId } = useParams();
  const [tab, setTab] = useTabIndex(TABS);
  const qc = useQueryClient();

  // Full-page sub-views (create / detail) take over the whole content area —
  // no page header or tabs — matching the Flutter messaging page.
  const [creating, setCreating] = useState<MsgType | null>(null);
  const [selected, setSelected] = useState<MessageRow | null>(null);

  if (creating) {
    return (
      <div className="flex flex-col p-6 md:p-8">
        <CreateMessageView
          type={creating}
          onBack={() => setCreating(null)}
          onDone={() => {
            setCreating(null);
            qc.invalidateQueries({ queryKey: ['/messaging/messages'] });
          }}
        />
      </div>
    );
  }

  if (selected) {
    return (
      <div className="flex flex-col p-6 md:p-8">
        <MessageDetail msg={selected} onBack={() => setSelected(null)} />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div className="flex flex-col gap-1">
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
          Messaging
        </h1>
        <p className="text-[length:var(--text-body)] text-text-secondary">
          Send email, SMS and push notifications to your users
        </p>
      </div>

      <PageTabs tabs={TABS} selected={tab} onChange={setTab} />

      <div>
        {tab === 0 && (
          <MessagesTab projectId={projectId} onCreate={setCreating} onSelect={setSelected} />
        )}
        {tab === 1 && <TopicsTab projectId={projectId} />}
        {tab === 2 && <TemplatesTab projectId={projectId} />}
        {tab === 3 && <ProvidersTab />}
      </div>
    </div>
  );
}
