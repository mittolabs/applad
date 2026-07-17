import { useState } from 'react';
import { Check, Copy } from 'lucide-react';
import { cn } from '@/lib/utils';

/* Ports id_text.dart — monospace truncated ID; click text to expand,
 * click copy icon → clipboard with 2s check-state feedback. */
export function IdText({
  id,
  previewLength = 12,
  fontSize = 13,
  className,
}: {
  id: string;
  previewLength?: number;
  fontSize?: number;
  className?: string;
}) {
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);

  const display =
    expanded || id.length <= previewLength ? id : `${id.slice(0, previewLength)}…`;

  const copy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(id);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard unavailable */
    }
  };

  return (
    <span className={cn('inline-flex items-center gap-1.5', className)}>
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          setExpanded((v) => !v);
        }}
        className="font-[family-name:var(--font-mono)] text-text-secondary transition-colors hover:text-text-primary"
        style={{ fontSize }}
        title={id}
      >
        {display}
      </button>
      <button
        type="button"
        onClick={copy}
        className="text-text-subtle transition-colors hover:text-text-primary"
        aria-label="Copy ID"
      >
        {copied ? (
          <Check size={12} className="text-status-success" />
        ) : (
          <Copy size={12} />
        )}
      </button>
    </span>
  );
}
