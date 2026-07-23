import type { ReactNode } from 'react';
import { StandaloneNav } from './StandaloneNav';
import { ConsoleFooter } from '@/components/console-footer';
import { Notices } from '@/components/notices';

/* Layout for shell-less pages (projects, account, experiments): the shared
 * TopNav, the page content (which should be flex-1 to push the footer down),
 * and the SAME ConsoleFooter the project shell uses. Guarantees nav + footer
 * stay identical across every standalone page. */
export function StandaloneLayout({
  showOrg = true,
  children,
}: {
  showOrg?: boolean;
  children: ReactNode;
}) {
  return (
    <div className="flex min-h-screen flex-col bg-background">
      <StandaloneNav showOrg={showOrg} />
      {/* app.top is application-wide, so it appears here too, not only inside a
          project. project.top and page.top are out of scope on these pages. */}
      <Notices region="app.top" className="shrink-0 px-6 pt-4" />
      {children}
      <ConsoleFooter />
    </div>
  );
}
