import { useParams } from 'react-router-dom';
import { PageTabs } from '@/components/page-tabs';
import { useTabIndex } from '@/hooks/use-tab-param';
import { UsersTab } from './UsersTab';
import { TeamsTab } from './TeamsTab';
import { SecurityTab } from './SecurityTab';
import { TemplatesTab } from './TemplatesTab';
import { UsageTab } from './UsageTab';
import { SettingsTab } from './SettingsTab';

// URL ?tab= values (lowercase) mirror the Flutter source; labels are display-only.
const TAB_KEYS = ['users', 'teams', 'security', 'templates', 'usage', 'settings'];
const TAB_LABELS = ['Users', 'Teams', 'Security', 'Templates', 'Usage', 'Settings'];

export function AuthPage() {
  const { projectId } = useParams();
  const [tab, setTab] = useTabIndex(TAB_KEYS);
  const pid = projectId ?? '';

  return (
    <div className="flex flex-col gap-6 p-6 md:p-8">
      <div>
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">Auth</h1>
        <p className="mt-1 text-[length:var(--text-body)] text-text-secondary">
          Manage users, sessions, OAuth providers and access control
        </p>
      </div>

      <PageTabs tabs={TAB_LABELS} selected={tab} onChange={setTab} />

      <div>
        {tab === 0 && <UsersTab projectId={pid} />}
        {tab === 1 && <TeamsTab projectId={pid} />}
        {tab === 2 && <SecurityTab projectId={pid} />}
        {tab === 3 && <TemplatesTab projectId={pid} />}
        {tab === 4 && <UsageTab />}
        {tab === 5 && <SettingsTab projectId={pid} />}
      </div>
    </div>
  );
}
