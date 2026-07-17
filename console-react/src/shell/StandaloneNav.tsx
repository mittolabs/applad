import { TopNav } from './TopNav';

/* Shell-less pages (projects, account, experiments) use the SAME top nav as the
 * project shell — just without a project switcher. `showOrg=false` hides the org
 * switcher (account/experiments). */
export function StandaloneNav({ showOrg = true }: { showOrg?: boolean }) {
  return <TopNav showOrg={showOrg} />;
}
