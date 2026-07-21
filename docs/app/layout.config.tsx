import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';

// Hosts differ between local dev (a `.localhost` suffix on every name) and
// production. These pages are server-rendered, so the values come from the
// environment rather than from window.location as they do on the site.
const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? 'https://applad.io';
const consoleUrl = process.env.NEXT_PUBLIC_CONSOLE_URL ?? 'https://console.applad.io';

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <>
        <img src="/applad-mascot-head.png" alt="" width={22} height={22} style={{ borderRadius: 9999 }} />
        Applad Docs
      </>
    ),
    url: siteUrl,
  },
  links: [
    { text: 'Console', url: consoleUrl, active: 'none' },
    { text: 'GitHub', url: 'https://github.com/mittolabs/applad', active: 'none' },
  ],
};
