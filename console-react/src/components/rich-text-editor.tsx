import { useRef, useState } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Bold, Heading, Italic, Link2, List } from 'lucide-react';
import { cn } from '@/lib/utils';

/* Ports rich_text_editor.dart — Markdown editor with Write/Preview tabs and a
 * toolbar (bold/italic/heading/list/link). Controlled via value/onChange. */
export function RichTextEditor({
  value,
  onChange,
  placeholder = 'Write markdown…',
  minRows = 12,
  className,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  minRows?: number;
  className?: string;
}) {
  const [tab, setTab] = useState<'write' | 'preview'>('write');
  const ref = useRef<HTMLTextAreaElement>(null);

  const surround = (before: string, after = before) => {
    const el = ref.current;
    if (!el) return;
    const { selectionStart: s, selectionEnd: e } = el;
    const sel = value.slice(s, e);
    onChange(value.slice(0, s) + before + sel + after + value.slice(e));
    requestAnimationFrame(() => {
      el.focus();
      el.selectionStart = s + before.length;
      el.selectionEnd = e + before.length;
    });
  };

  const linePrefix = (prefix: string) => {
    const el = ref.current;
    if (!el) return;
    const s = el.selectionStart;
    const lineStart = value.lastIndexOf('\n', s - 1) + 1;
    onChange(value.slice(0, lineStart) + prefix + value.slice(lineStart));
  };

  return (
    <div
      className={cn(
        'overflow-hidden rounded-[var(--radius)] border border-field-border bg-field-fill',
        className,
      )}
    >
      <div className="flex items-center gap-1 border-b border-border px-2 py-1.5">
        <ToolButton icon={Bold} onClick={() => surround('**')} label="Bold" />
        <ToolButton icon={Italic} onClick={() => surround('_')} label="Italic" />
        <ToolButton icon={Heading} onClick={() => linePrefix('## ')} label="Heading" />
        <ToolButton icon={List} onClick={() => linePrefix('- ')} label="List" />
        <ToolButton icon={Link2} onClick={() => surround('[', '](url)')} label="Link" />
        <div className="ml-auto flex items-center gap-0.5">
          {(['write', 'preview'] as const).map((t) => (
            <button
              key={t}
              type="button"
              onClick={() => setTab(t)}
              className={cn(
                'rounded-[var(--radius-6)] px-2 py-1 text-[length:var(--text-caption)] capitalize transition-colors',
                tab === t ? 'bg-fill text-text-primary' : 'text-text-muted hover:text-text-secondary',
              )}
            >
              {t}
            </button>
          ))}
        </div>
      </div>
      {tab === 'write' ? (
        <textarea
          ref={ref}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          rows={minRows}
          className="w-full resize-y bg-transparent px-3 py-2.5 font-[family-name:var(--font-mono)] text-[length:var(--text-body)] text-text-primary placeholder:text-text-subtle focus:outline-none"
        />
      ) : (
        <div className="prose-console px-3 py-2.5 text-[length:var(--text-body)] text-text-secondary">
          <Markdown remarkPlugins={[remarkGfm]}>{value || '_Nothing to preview_'}</Markdown>
        </div>
      )}
    </div>
  );
}

function ToolButton({
  icon: Icon,
  onClick,
  label,
}: {
  icon: typeof Bold;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-[var(--radius-6)] p-1.5 text-text-muted transition-colors hover:bg-fill hover:text-text-primary"
      aria-label={label}
      title={label}
    >
      <Icon size={14} />
    </button>
  );
}
