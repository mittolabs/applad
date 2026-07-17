import { useEffect, useRef, useState } from 'react';
import { Send, Sparkles, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

/* AI assistant side panel — ports ai/ai_chat.dart. Simulated responses
 * (faithful to the Flutter stub; the real assistant would call /ai). Gated by
 * the experiments.aiChat flag at the call site (shell). */

interface Message {
  role: 'user' | 'assistant';
  text: string;
}

const SUGGESTIONS = [
  'How do I create a database?',
  'Show me my recent deployments',
  'Explain feature flags',
];

const CANNED =
  "I'm the Applad assistant (preview). I can help you navigate the console, understand features, and draft config. Connect an AI provider to enable full answers.";

export function AiChatPanel({ onClose }: { onClose: () => void }) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [thinking, setThinking] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight });
  }, [messages, thinking]);

  const send = (text: string) => {
    const q = text.trim();
    if (!q) return;
    setMessages((m) => [...m, { role: 'user', text: q }]);
    setInput('');
    setThinking(true);
    // Simulated response (mirrors the Flutter placeholder behavior).
    setTimeout(() => {
      setMessages((m) => [...m, { role: 'assistant', text: CANNED }]);
      setThinking(false);
    }, 700);
  };

  return (
    <div className="flex w-[360px] shrink-0 flex-col border-l border-border bg-surface">
      <div className="flex h-[52px] items-center gap-2 border-b border-border px-4">
        <Sparkles size={16} className="text-[var(--color-accent)]" />
        <span className="text-[length:var(--text-control)] font-medium text-text-primary">
          Assistant
        </span>
        <button
          onClick={onClose}
          className="ml-auto rounded-[var(--radius-6)] p-1 text-text-muted hover:bg-fill hover:text-text-primary"
          aria-label="Close assistant"
        >
          <X size={16} />
        </button>
      </div>

      <div ref={scrollRef} className="flex-1 space-y-3 overflow-y-auto p-4">
        {messages.length === 0 && (
          <div className="flex flex-col items-center gap-4 py-8 text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-fill text-[var(--color-accent)]">
              <Sparkles size={22} />
            </div>
            <div className="text-[length:var(--text-body)] text-text-muted">
              Ask me anything about your project.
            </div>
            <div className="flex w-full flex-col gap-2">
              {SUGGESTIONS.map((s) => (
                <button
                  key={s}
                  onClick={() => send(s)}
                  className="rounded-[var(--radius)] border border-border bg-surface px-3 py-2 text-left text-[length:var(--text-body)] text-text-secondary hover:bg-fill"
                >
                  {s}
                </button>
              ))}
            </div>
          </div>
        )}
        {messages.map((m, i) => (
          <div
            key={i}
            className={cn('flex', m.role === 'user' ? 'justify-end' : 'justify-start')}
          >
            <div
              className={cn(
                'max-w-[85%] rounded-[var(--radius-10)] px-3 py-2 text-[length:var(--text-body)]',
                m.role === 'user'
                  ? 'bg-[var(--color-accent)] text-white'
                  : 'bg-surface text-text-primary',
              )}
            >
              {m.text}
            </div>
          </div>
        ))}
        {thinking && (
          <div className="flex justify-start">
            <div className="rounded-[var(--radius-10)] bg-surface px-3 py-2 text-[length:var(--text-body)] text-text-muted">
              Thinking…
            </div>
          </div>
        )}
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          send(input);
        }}
        className="flex items-center gap-2 border-t border-border p-3"
      >
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Message the assistant…"
          className="h-9 flex-1 rounded-[var(--radius)] border border-field-border bg-field-fill px-3 text-[length:var(--text-body)] text-text-primary placeholder:text-text-subtle focus:border-[var(--color-accent)] focus:outline-none"
        />
        <Button type="submit" size="icon" disabled={!input.trim()}>
          <Send size={15} />
        </Button>
      </form>
    </div>
  );
}
