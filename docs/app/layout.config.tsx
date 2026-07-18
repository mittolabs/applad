import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <>
        <img src="/applad-mascot-head.png" alt="" width={22} height={22} style={{ borderRadius: 9999 }} />
        Applad Docs
      </>
    ),
    url: 'http://applad.io.localhost',
  },
  links: [
    { text: 'Console', url: 'http://applad.dev.localhost', active: 'none' },
    { text: 'GitHub', url: 'https://github.com/mittolabs/applad', active: 'none' },
  ],
};
