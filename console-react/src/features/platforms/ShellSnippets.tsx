import { useState } from 'react';
import { CodeBlock } from '@/components/code-block';
import { cn } from '@/lib/utils';

/* Tabbed shell snippets (Unix / CMD / PowerShell) — ports the tab row +
 * code block from the CLI deploy dialog in platforms_page.dart. */

export interface ShellSnippetSet {
  unix: string;
  cmd: string;
  powershell: string;
}

const TABS: { key: keyof ShellSnippetSet; label: string; language: string }[] = [
  { key: 'unix', label: 'Unix', language: 'bash' },
  { key: 'cmd', label: 'CMD', language: 'cmd' },
  { key: 'powershell', label: 'PowerShell', language: 'powershell' },
];

export function ShellSnippets({ snippets }: { snippets: ShellSnippetSet }) {
  const [tab, setTab] = useState<keyof ShellSnippetSet>('unix');
  const active = TABS.find((t) => t.key === tab) ?? TABS[0];

  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex items-center gap-0.5">
        {TABS.map((t) => {
          const selected = t.key === tab;
          return (
            <button
              key={t.key}
              type="button"
              onClick={() => setTab(t.key)}
              className={cn(
                'rounded-[var(--radius-6)] border px-3.5 py-1.5 text-[length:var(--text-label)] transition-colors',
                selected
                  ? 'border-[color-mix(in_srgb,var(--color-accent)_40%,transparent)] bg-[color-mix(in_srgb,var(--color-accent)_15%,transparent)] font-medium text-[var(--color-accent)]'
                  : 'border-border text-text-secondary hover:text-text-primary',
              )}
            >
              {t.label}
            </button>
          );
        })}
      </div>
      <CodeBlock code={snippets[tab]} language={active.language} />
    </div>
  );
}
