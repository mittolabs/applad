import { useMemo, useState } from 'react';
import { Check, Copy } from 'lucide-react';
import { cn } from '@/lib/utils';

/* Ports code_block.dart — selectable monospace code with a lightweight
 * highlighter (comments / strings / numbers / keywords) using the same
 * token palette, plus a copy button and optional language badge. */

const KEYWORDS =
  /\b(const|let|var|function|return|if|else|for|while|import|from|export|class|extends|new|await|async|final|void|int|String|bool|true|false|null|def|end|do|then|package|func|type|struct|interface|public|private|static)\b/g;

type Token = { text: string; kind: 'plain' | 'comment' | 'string' | 'number' | 'keyword' };

function tokenizeLine(line: string): Token[] {
  // Comments first (rest of line).
  const commentMatch = line.match(/(\/\/|#).*/);
  if (commentMatch && commentMatch.index !== undefined) {
    const before = line.slice(0, commentMatch.index);
    return [
      ...tokenizeLine(before),
      { text: commentMatch[0], kind: 'comment' },
    ];
  }
  const tokens: Token[] = [];
  const stringRe = /(["'`])(?:\\.|(?!\1).)*\1/g;
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = stringRe.exec(line))) {
    if (m.index > last) tokens.push(...splitWords(line.slice(last, m.index)));
    tokens.push({ text: m[0], kind: 'string' });
    last = m.index + m[0].length;
  }
  if (last < line.length) tokens.push(...splitWords(line.slice(last)));
  return tokens;
}

function splitWords(text: string): Token[] {
  const out: Token[] = [];
  let last = 0;
  const re = new RegExp(`${KEYWORDS.source}|\\b\\d+(?:\\.\\d+)?\\b`, 'g');
  let m: RegExpExecArray | null;
  while ((m = re.exec(text))) {
    if (m.index > last) out.push({ text: text.slice(last, m.index), kind: 'plain' });
    out.push({ text: m[0], kind: /^\d/.test(m[0]) ? 'number' : 'keyword' });
    last = m.index + m[0].length;
  }
  if (last < text.length) out.push({ text: text.slice(last), kind: 'plain' });
  return out;
}

const COLORS: Record<Token['kind'], string> = {
  plain: 'var(--code-plain)',
  comment: 'var(--code-comment)',
  string: 'var(--code-string)',
  number: 'var(--code-number)',
  keyword: 'var(--code-keyword)',
};

export function CodeBlock({
  code,
  language,
  fontSize = 11.5,
  className,
}: {
  code: string;
  language?: string;
  fontSize?: number;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);
  const lines = useMemo(() => code.replace(/\n$/, '').split('\n'), [code]);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard unavailable */
    }
  };

  return (
    <div
      className={cn(
        'group relative overflow-hidden rounded-[var(--radius)] border border-border bg-surface-alt',
        className,
      )}
    >
      {language && (
        <div className="flex items-center justify-between border-b border-border px-3 py-1.5">
          <span className="text-[length:var(--text-2xs)] uppercase tracking-wide text-text-subtle">
            {language}
          </span>
        </div>
      )}
      <button
        type="button"
        onClick={copy}
        className="absolute right-2 top-2 z-10 rounded-[var(--radius-6)] border border-border bg-surface p-1.5 text-text-muted opacity-0 transition-opacity hover:text-text-primary group-hover:opacity-100"
        aria-label="Copy code"
      >
        {copied ? <Check size={13} className="text-status-success" /> : <Copy size={13} />}
      </button>
      <pre
        className="overflow-x-auto px-3 py-2.5 font-[family-name:var(--font-mono)] leading-relaxed"
        style={{ fontSize }}
      >
        <code>
          {lines.map((line, i) => (
            <div key={i}>
              {line === '' ? (
                ' '
              ) : (
                tokenizeLine(line).map((t, j) => (
                  <span key={j} style={{ color: COLORS[t.kind] }}>
                    {t.text}
                  </span>
                ))
              )}
            </div>
          ))}
        </code>
      </pre>
    </div>
  );
}
