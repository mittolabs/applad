import { Github, MessageCircle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { APP_VERSION } from '@/lib/version';

const YEAR = 2026; // Date.* is avoided; bump on release.

/* Ports console_footer.dart — 44px bottom bar: © year, GitHub/Discord icons,
 * version, Docs/Terms/Privacy links (open new tab). */
export function ConsoleFooter({ className }: { className?: string }) {
  return (
    <footer
      className={cn(
        'flex h-11 shrink-0 items-center gap-4 border-t border-border px-4 text-[length:var(--text-caption)] text-text-muted',
        className,
      )}
    >
      <span>© {YEAR} Mittolabs LTD. All rights reserved.</span>
      <a
        href="https://github.com/mittolabs/applad"
        target="_blank"
        rel="noreferrer"
        className="transition-colors hover:text-text-primary"
        aria-label="GitHub"
      >
        <Github size={14} />
      </a>
      <a
        href="https://discord.gg/applad"
        target="_blank"
        rel="noreferrer"
        className="transition-colors hover:text-text-primary"
        aria-label="Discord"
      >
        <MessageCircle size={14} />
      </a>
      <span className="text-text-subtle">{APP_VERSION}</span>
      <div className="ml-auto flex items-center gap-4">
        {[
          ['Docs', 'https://docs.applad.io'],
          ['Terms', 'https://applad.io/terms'],
          ['Privacy', 'https://applad.io/privacy'],
        ].map(([label, href]) => (
          <a
            key={label}
            href={href}
            target="_blank"
            rel="noreferrer"
            className="transition-colors hover:text-text-primary"
          >
            {label}
          </a>
        ))}
      </div>
    </footer>
  );
}
