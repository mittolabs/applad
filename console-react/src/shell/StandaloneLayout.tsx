import type { ReactNode } from 'react';
import { StandaloneNav } from './StandaloneNav';
import { ConsoleFooter } from '@/components/console-footer';

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
      {children}
      <ConsoleFooter />
    </div>
  );
}
