import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  ArrowUp,
  Code2,
  Database,
  HardDrive,
  Mail,
  Maximize2,
  MessageCircle,
  MessageSquare,
  Mic,
  Minimize2,
  Paperclip,
  Plus,
  Rocket,
  Smile,
  Users,
  Workflow,
  X,
  type LucideIcon,
} from 'lucide-react';
import { api } from '@/api/client';
import { useAuthStore } from '@/stores/auth';
import { cn } from '@/lib/utils';

/*
 * Applad AI — global floating chat overlay. Faithful port of
 * console/lib/features/ai/ai_chat.dart:
 *   • draggable mascot bubble (bottom-right)
 *   • compact Intercom Fin-style panel
 *   • full-screen expanded workspace (sidebar + quick actions + dot grid)
 * Mounted once at the router root. Shown only when logged in, off the login
 * page, and when the backend reports AI configured (`/ai/config`). Streams
 * tokens from POST /ai/stream (SSE: {delta} / {error} / [DONE]).
 */

// ── Palette (matches ai_chat.dart) ───────────────────────────────────────────
const C = {
  panelBg: '#1A1B1E',
  headerBg: '#1E1F23',
  msgBg: '#25262A',
  inputBg: '#25262A',
  codeBg: '#0D0E10',
  expandedBg: '#0F1011',
  sidebarBg: '#131416',
  accent: '#3472A4',
  accentDim: '#1D5A8A',
  divider: 'rgba(255,255,255,0.08)',
  textPri: '#FFFFFF',
  textSec: '#AAAAAA',
  textMuted: '#606068',
  codeText: '#7DD3FC',
};
const BUBBLE = 52;
const PANEL_W = 380;
const PANEL_H = 520;

interface Msg {
  role: 'user' | 'assistant';
  text: string;
  time: number;
}
interface Session {
  title: string;
  messages: Msg[];
}
interface QuickAction {
  icon: LucideIcon;
  label: string;
  prompt: string;
}

const QUICK_ACTIONS: QuickAction[] = [
  { icon: Database, label: 'Create a table', prompt: 'Help me create a new database table with proper columns and types.' },
  { icon: Code2, label: 'Write a function', prompt: 'Help me write and deploy a serverless function.' },
  { icon: Users, label: 'Set up OAuth', prompt: 'How do I configure OAuth (Google, GitHub, Apple) for my project?' },
  { icon: HardDrive, label: 'Create a bucket', prompt: 'How do I create a storage bucket and handle file uploads?' },
  { icon: Rocket, label: 'Deploy a container', prompt: 'Walk me through deploying a Docker container with Applad.' },
  { icon: Workflow, label: 'Build a workflow', prompt: 'Help me create an automated workflow using the DAG engine.' },
  { icon: Mail, label: 'Send emails', prompt: 'How do I send transactional emails from my project?' },
];

function routeLabel(path: string): string {
  if (path.includes('/databases')) return 'Databases page';
  if (path.includes('/functions')) return 'Functions page';
  if (path.includes('/storage')) return 'Storage page';
  if (path.includes('/auth')) return 'Auth page';
  if (/\/(deploy|sites|containers|mobile|desktop)/.test(path)) return 'Deploy page';
  if (path.includes('/messaging')) return 'Messaging page';
  if (path.includes('/workflows')) return 'Workflows page';
  if (path.includes('/settings')) return 'Project settings';
  if (path.includes('/overview')) return 'Project overview';
  return '';
}

function timeLabel(t: number): string {
  const s = Math.floor((Date.now() - t) / 1000);
  if (s < 60) return 'Just now';
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  return `${Math.floor(s / 3600)}h ago`;
}

// ── Root overlay ─────────────────────────────────────────────────────────────

export function AiChatOverlay() {
  const token = useAuthStore((s) => s.token);
  const location = useLocation();
  const onLogin = location.pathname === '/login';

  const { data: config } = useQuery({
    queryKey: ['ai-config', token],
    enabled: !!token,
    staleTime: 60_000,
    queryFn: async () => {
      const res = await api.get('/ai/config');
      return res.data as { configured?: boolean; model?: string };
    },
  });

  const [open, setOpen] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const [messages, setMessages] = useState<Msg[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(false);
  const [streaming, setStreaming] = useState(false);
  const [input, setInput] = useState('');
  const [pos, setPos] = useState<{ x: number; y: number } | null>(null);

  const scrollRef = useRef<HTMLDivElement>(null);
  const expScrollRef = useRef<HTMLDivElement>(null);

  // Initial bubble position (bottom-right).
  useEffect(() => {
    if (pos === null) {
      setPos({ x: window.innerWidth - BUBBLE - 20, y: window.innerHeight - BUBBLE - 28 });
    }
  }, [pos]);

  const scrollDown = useCallback(() => {
    requestAnimationFrame(() => {
      scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' });
      expScrollRef.current?.scrollTo({ top: expScrollRef.current.scrollHeight, behavior: 'smooth' });
    });
  }, []);

  useEffect(scrollDown, [messages, loading, scrollDown]);

  const send = useCallback(
    async (raw?: string) => {
      const text = (raw ?? input).trim();
      if (!text || loading || streaming) return;
      setInput('');
      const userMsg: Msg = { role: 'user', text, time: Date.now() };
      const history = [...messages, userMsg];
      setMessages(history);
      setLoading(true);
      setStreaming(true);
      scrollDown();

      try {
        const res = await fetch(`${api.defaults.baseURL}/ai/stream`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            messages: history.map((m) => ({ role: m.role, content: m.text })),
            context: routeLabel(location.pathname),
          }),
        });
        if (!res.body) throw new Error('no stream');
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let leftover = '';
        let gotFirst = false;

        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          const lines = (leftover + decoder.decode(value, { stream: true })).split('\n');
          leftover = lines.pop() ?? '';
          for (const line of lines) {
            if (!line.startsWith('data: ')) continue;
            const data = line.slice(6).trim();
            if (data === '[DONE]') continue;
            let ev: { delta?: string; error?: string };
            try {
              ev = JSON.parse(data);
            } catch {
              continue;
            }
            if (ev.error) {
              setLoading(false);
              gotFirst = true;
              setMessages((m) => [...m, { role: 'assistant', text: ev.error!, time: Date.now() }]);
              scrollDown();
            } else if (ev.delta != null) {
              setMessages((m) => {
                if (!gotFirst) {
                  return [...m, { role: 'assistant', text: ev.delta!, time: Date.now() }];
                }
                const copy = m.slice();
                const last = copy[copy.length - 1];
                copy[copy.length - 1] = { ...last, text: last.text + ev.delta! };
                return copy;
              });
              if (!gotFirst) {
                gotFirst = true;
                setLoading(false);
              }
              scrollDown();
            }
          }
        }
        if (!gotFirst) {
          setMessages((m) => [
            ...m,
            { role: 'assistant', text: 'Sorry, I ran into an issue. Please try again.', time: Date.now() },
          ]);
        }
      } catch {
        setMessages((m) => [
          ...m,
          { role: 'assistant', text: 'Sorry, I ran into an issue. Please try again.', time: Date.now() },
        ]);
      } finally {
        setLoading(false);
        setStreaming(false);
        scrollDown();
      }
    },
    [input, loading, streaming, messages, token, location.pathname, scrollDown],
  );

  const newChat = useCallback(() => {
    setMessages((cur) => {
      if (cur.length > 0) {
        const first = cur.find((m) => m.role === 'user')?.text ?? 'Chat';
        const title = first.length > 40 ? `${first.slice(0, 40)}…` : first;
        setSessions((s) => [{ title, messages: cur }, ...s]);
      }
      return [];
    });
  }, []);

  if (!token || onLogin || !config?.configured || !pos) return null;

  return (
    <>
      {expanded && (
        <ExpandedWorkspace
          messages={messages}
          sessions={sessions}
          loading={loading}
          streaming={streaming}
          model={config.model ?? ''}
          input={input}
          setInput={setInput}
          scrollRef={expScrollRef}
          onSend={send}
          onFill={(p) => setInput(p)}
          onCollapse={() => {
            setExpanded(false);
            setOpen(true);
          }}
          onClose={() => {
            setExpanded(false);
            setOpen(false);
          }}
          onNewChat={newChat}
          onLoadSession={(s) => setMessages(s.messages)}
        />
      )}

      {open && !expanded && (
        <CompactPanel
          pos={pos}
          messages={messages}
          loading={loading}
          streaming={streaming}
          input={input}
          setInput={setInput}
          scrollRef={scrollRef}
          onSend={send}
          onExpand={() => {
            setOpen(false);
            setExpanded(true);
          }}
          onClose={() => setOpen(false)}
        />
      )}

      {!expanded && (
        <DraggableBubble
          pos={pos}
          setPos={setPos}
          open={open}
          hasUnread={messages.length > 0 && !open}
          onToggle={() => setOpen((v) => !v)}
        />
      )}
    </>
  );
}

// ── Draggable bubble ─────────────────────────────────────────────────────────

function DraggableBubble({
  pos,
  setPos,
  open,
  hasUnread,
  onToggle,
}: {
  pos: { x: number; y: number };
  setPos: (p: { x: number; y: number }) => void;
  open: boolean;
  hasUnread: boolean;
  onToggle: () => void;
}) {
  const drag = useRef<{ moved: boolean; sx: number; sy: number; px: number; py: number } | null>(null);

  const onDown = (e: React.PointerEvent) => {
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
    drag.current = { moved: false, sx: e.clientX, sy: e.clientY, px: pos.x, py: pos.y };
  };
  const onMove = (e: React.PointerEvent) => {
    const d = drag.current;
    if (!d) return;
    const dx = e.clientX - d.sx;
    const dy = e.clientY - d.sy;
    if (Math.abs(dx) + Math.abs(dy) > 4) d.moved = true;
    setPos({
      x: Math.min(Math.max(0, d.px + dx), window.innerWidth - BUBBLE),
      y: Math.min(Math.max(0, d.py + dy), window.innerHeight - BUBBLE),
    });
  };
  const onUp = (e: React.PointerEvent) => {
    const d = drag.current;
    drag.current = null;
    (e.target as HTMLElement).releasePointerCapture(e.pointerId);
    if (d && !d.moved) onToggle();
  };

  return (
    <div
      onPointerDown={onDown}
      onPointerMove={onMove}
      onPointerUp={onUp}
      className="fixed z-[60] flex cursor-pointer items-center justify-center rounded-full bg-white ai-bubble-pulse"
      style={{
        left: pos.x,
        top: pos.y,
        width: BUBBLE,
        height: BUBBLE,
        touchAction: 'none',
      }}
    >
      {open ? (
        <X size={22} color="#444444" />
      ) : (
        <img
          src="/applad-mascot-head.png"
          alt="Applad AI"
          className="rounded-full object-cover"
          style={{ width: BUBBLE - 8, height: BUBBLE - 8 }}
          draggable={false}
        />
      )}
      {hasUnread && (
        <span
          className="absolute rounded-full"
          style={{ top: 8, right: 8, width: 9, height: 9, background: '#10B981' }}
        />
      )}
    </div>
  );
}

// ── Compact panel ────────────────────────────────────────────────────────────

function CompactPanel({
  pos,
  messages,
  loading,
  streaming,
  input,
  setInput,
  scrollRef,
  onSend,
  onExpand,
  onClose,
}: {
  pos: { x: number; y: number };
  messages: Msg[];
  loading: boolean;
  streaming: boolean;
  input: string;
  setInput: (v: string) => void;
  scrollRef: React.RefObject<HTMLDivElement>;
  onSend: () => void;
  onExpand: () => void;
  onClose: () => void;
}) {
  const availH = window.innerHeight - 72;
  const panelH = Math.min(PANEL_H, availH);
  let x = pos.x - PANEL_W - 12;
  if (x < 12) x = pos.x + BUBBLE + 12;
  x = Math.min(Math.max(12, x), Math.max(12, window.innerWidth - PANEL_W - 12));
  const y = Math.max(60, window.innerHeight - panelH - 12);

  return (
    <div
      className="fixed z-[60] flex flex-col overflow-hidden"
      style={{
        left: x,
        top: y,
        width: PANEL_W,
        height: panelH,
        background: C.panelBg,
        borderRadius: 16,
        border: `1px solid ${C.divider}`,
        boxShadow: '0 16px 48px rgba(0,0,0,0.55)',
      }}
    >
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-3.5" style={{ background: C.headerBg }}>
        <img src="/applad-mascot-head.png" alt="" className="h-[34px] w-[34px] rounded-full object-cover" />
        <div className="min-w-0 flex-1">
          <div style={{ color: C.textPri }} className="text-[14px] font-semibold leading-tight">
            Applad AI
          </div>
          <div style={{ color: C.textSec }} className="text-[11.5px] leading-tight">
            Ask anything about your project
          </div>
        </div>
        <IconBtn icon={Maximize2} onClick={onExpand} />
        <IconBtn icon={X} onClick={onClose} />
      </div>
      <div style={{ height: 1, background: C.divider }} />

      {/* Messages */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-4 pb-3 pt-4">
        <WelcomeBubble compact />
        {messages.map((m, i) => (
          <MsgBubble key={i} msg={m} compact />
        ))}
        {loading && <ThinkingBubble compact />}
      </div>

      <div style={{ height: 1, background: C.divider }} />
      <FinInput value={input} setValue={setInput} onSend={onSend} loading={loading || streaming} />
      {/* Footer */}
      <div
        className="flex items-center justify-center gap-1.5 py-2"
        style={{ background: C.headerBg }}
      >
        <MessageCircle size={11} color={C.textMuted} />
        <span style={{ color: C.textMuted }} className="text-[11px]">
          Chat with Applad
        </span>
      </div>
    </div>
  );
}

// ── Expanded workspace ───────────────────────────────────────────────────────

function ExpandedWorkspace({
  messages,
  sessions,
  loading,
  streaming,
  model,
  input,
  setInput,
  scrollRef,
  onSend,
  onFill,
  onCollapse,
  onClose,
  onNewChat,
  onLoadSession,
}: {
  messages: Msg[];
  sessions: Session[];
  loading: boolean;
  streaming: boolean;
  model: string;
  input: string;
  setInput: (v: string) => void;
  scrollRef: React.RefObject<HTMLDivElement>;
  onSend: () => void;
  onFill: (p: string) => void;
  onCollapse: () => void;
  onClose: () => void;
  onNewChat: () => void;
  onLoadSession: (s: Session) => void;
}) {
  return (
    <div className="fixed inset-0 z-[70] flex" style={{ background: C.expandedBg }}>
      {/* Sidebar */}
      <div className="flex w-[210px] flex-col" style={{ background: C.sidebarBg }}>
        <div className="flex items-center gap-2.5 px-4 pb-3 pt-4">
          <img src="/applad-mascot-head.png" alt="" className="h-[26px] w-[26px] rounded-full object-cover" />
          <span style={{ color: C.textPri }} className="flex-1 text-[13px] font-semibold">
            Applad AI
          </span>
          <IconBtn icon={Plus} onClick={onNewChat} />
        </div>
        <div style={{ height: 1, background: C.divider }} />
        <div className="flex-1 overflow-y-auto px-1 pt-2">
          {sessions.length === 0 ? (
            <p style={{ color: C.textMuted }} className="px-3 pt-2 text-[12px] leading-relaxed">
              Previous conversations will appear here.
            </p>
          ) : (
            sessions.map((s, i) => (
              <button
                key={i}
                onClick={() => onLoadSession(s)}
                className="mb-px flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left hover:bg-white/[0.06]"
              >
                <MessageSquare size={13} color={C.textMuted} className="shrink-0" />
                <span style={{ color: C.textSec }} className="truncate text-[12.5px]">
                  {s.title}
                </span>
              </button>
            ))
          )}
        </div>
      </div>
      <div style={{ width: 1, background: C.divider }} />

      {/* Main */}
      <div className="relative flex flex-1 flex-col ai-dotgrid">
        <div className="flex items-center justify-end gap-0.5 px-3 pb-1.5 pt-2.5">
          <IconBtn icon={Minimize2} onClick={onCollapse} />
          <IconBtn icon={X} onClick={onClose} />
        </div>

        <div ref={scrollRef} className="flex-1 overflow-y-auto pt-4">
          {messages.map((m, i) =>
            m.role === 'user' ? (
              <div key={i} className="flex justify-end pb-5 pl-[60px] pr-8">
                <div
                  className="rounded-xl px-4 py-2.5"
                  style={{ background: 'rgba(255,255,255,0.06)', border: `1px solid ${C.divider}` }}
                >
                  <span style={{ color: C.textPri }} className="whitespace-pre-wrap text-[14px] leading-relaxed">
                    {m.text}
                  </span>
                </div>
              </div>
            ) : (
              <div key={i} className="flex gap-3 pb-5 pl-8 pr-[60px]">
                <img src="/applad-mascot-head.png" alt="" className="mt-px h-[22px] w-[22px] shrink-0 rounded-full object-cover" />
                <div className="min-w-0 flex-1">
                  <AssistantBubble text={m.text} compact={false} />
                </div>
              </div>
            ),
          )}
          {loading && (
            <div className="flex items-center gap-3 pb-5 pl-8 pr-[60px]">
              <img src="/applad-mascot-head.png" alt="" className="h-[22px] w-[22px] rounded-full object-cover" />
              <Dots />
            </div>
          )}
        </div>

        {/* Bottom input */}
        <div className="px-6 pb-7" style={{ background: C.expandedBg }}>
          <div className="mx-auto w-full max-w-[580px]">
            {messages.length === 0 && (
              <div className="mb-3 flex gap-2 overflow-x-auto pb-1">
                {QUICK_ACTIONS.slice(0, 4).map((a) => (
                  <button
                    key={a.label}
                    onClick={() => onFill(a.prompt)}
                    className="ai-chip flex shrink-0 items-center gap-2 rounded-full px-4 py-2.5 text-[13px]"
                  >
                    <a.icon size={14} />
                    {a.label}
                  </button>
                ))}
              </div>
            )}
            <BigInput
              value={input}
              setValue={setInput}
              onSend={onSend}
              loading={loading || streaming}
              model={model}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Message rendering ────────────────────────────────────────────────────────

function WelcomeBubble({ compact }: { compact: boolean }) {
  return (
    <div className={compact ? 'mb-3' : 'mb-4'}>
      <div
        className="px-3 py-2.5"
        style={{
          background: C.msgBg,
          border: `1px solid rgba(255,255,255,0.06)`,
          borderRadius: '4px 16px 16px 16px',
        }}
      >
        <span style={{ color: C.textPri }} className="whitespace-pre-wrap text-[13px] leading-relaxed">
          {'Hi there 👋\n\nI’m your Applad AI assistant. Ask me anything about your project — databases, functions, auth, storage, deployments, workflows, and more.'}
        </span>
      </div>
      <div style={{ color: C.textMuted }} className="mt-1.5 text-[11px]">
        Applad AI • AI Assistant • Just now
      </div>
    </div>
  );
}

function MsgBubble({ msg, compact }: { msg: Msg; compact: boolean }) {
  const isUser = msg.role === 'user';
  return (
    <div className={cn(compact ? 'mb-3' : 'mb-4', 'flex flex-col', isUser ? 'items-end' : 'items-start')}>
      {isUser ? (
        <div
          className="px-3 py-2"
          style={{
            background: `color-mix(in srgb, ${C.accent} 22%, transparent)`,
            border: `1px solid color-mix(in srgb, ${C.accent} 30%, transparent)`,
            borderRadius: '16px 4px 16px 16px',
          }}
        >
          <span style={{ color: C.textPri }} className="whitespace-pre-wrap text-[13px] leading-relaxed">
            {msg.text}
          </span>
        </div>
      ) : (
        <AssistantBubble text={msg.text} compact={compact} />
      )}
      <div style={{ color: C.textMuted }} className="mt-1.5 text-[11px]">
        {isUser ? `You • ${timeLabel(msg.time)}` : `Applad AI • AI Assistant • ${timeLabel(msg.time)}`}
      </div>
    </div>
  );
}

function AssistantBubble({ text, compact }: { text: string; compact: boolean }) {
  const parts = useMemo(() => splitCodeBlocks(text), [text]);
  return (
    <div
      className={compact ? 'px-3 py-2' : 'px-4 py-3'}
      style={{
        background: C.msgBg,
        border: `1px solid rgba(255,255,255,0.06)`,
        borderRadius: '4px 16px 16px 16px',
      }}
    >
      {parts.map((p, i) =>
        p.isCode ? (
          <pre
            key={i}
            className="my-1.5 overflow-x-auto rounded-lg p-3"
            style={{ background: C.codeBg, border: `1px solid ${C.divider}` }}
          >
            <code style={{ color: C.codeText }} className="font-[family-name:var(--font-mono)] text-[13px] leading-relaxed">
              {p.text}
            </code>
          </pre>
        ) : (
          <span
            key={i}
            style={{ color: C.textPri }}
            className={cn('block whitespace-pre-wrap leading-relaxed', compact ? 'text-[13px]' : 'text-[14px]')}
          >
            {p.text}
          </span>
        ),
      )}
    </div>
  );
}

function splitCodeBlocks(raw: string): { text: string; isCode: boolean }[] {
  const parts: { text: string; isCode: boolean }[] = [];
  const re = /```[\w]*\n?([\s\S]*?)```/g;
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(raw))) {
    if (m.index > last) parts.push({ text: raw.slice(last, m.index).trim(), isCode: false });
    parts.push({ text: (m[1] ?? '').trim(), isCode: true });
    last = m.index + m[0].length;
  }
  if (last < raw.length) {
    const tail = raw.slice(last).trim();
    if (tail) parts.push({ text: tail, isCode: false });
  }
  if (parts.length === 0) parts.push({ text: raw, isCode: false });
  return parts.filter((p) => p.text.length > 0 || p.isCode);
}

function ThinkingBubble({ compact }: { compact: boolean }) {
  return (
    <div className={compact ? 'mb-3' : 'mb-4'}>
      <div
        className="inline-flex px-4 py-3.5"
        style={{
          background: C.msgBg,
          border: `1px solid rgba(255,255,255,0.06)`,
          borderRadius: '4px 16px 16px 16px',
        }}
      >
        <Dots />
      </div>
      <div style={{ color: C.textMuted }} className="mt-1.5 text-[11px]">
        Applad AI • Thinking...
      </div>
    </div>
  );
}

function Dots() {
  return (
    <span className="flex items-center gap-1.5">
      {[0, 1, 2].map((i) => (
        <span
          key={i}
          className="ai-dot inline-block rounded-full"
          style={{ width: 7, height: 7, background: C.accent, animationDelay: `${i * 0.16}s` }}
        />
      ))}
    </span>
  );
}

// ── Inputs ───────────────────────────────────────────────────────────────────

function FinInput({
  value,
  setValue,
  onSend,
  loading,
}: {
  value: string;
  setValue: (v: string) => void;
  onSend: () => void;
  loading: boolean;
}) {
  return (
    <div className="px-3 py-2">
      <div
        className="flex flex-col"
        style={{ background: C.inputBg, borderRadius: 14, border: `1px solid rgba(255,255,255,0.1)` }}
      >
        <textarea
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              onSend();
            }
          }}
          placeholder="Ask a question..."
          rows={1}
          className="resize-none bg-transparent px-3.5 pb-0.5 pt-2.5 text-[13.5px] leading-relaxed outline-none placeholder:text-[color:var(--ai-muted)]"
          style={{ color: C.textPri, ['--ai-muted' as string]: C.textMuted }}
        />
        <div className="flex items-center gap-0.5 px-2.5 pb-2.5 pt-0.5">
          <ToolbarIcon icon={Paperclip} />
          <ToolbarIcon icon={Smile} />
          <ToolbarIcon icon={Mic} />
          <div className="flex-1" />
          <SendBtn onSend={onSend} loading={loading} />
        </div>
      </div>
    </div>
  );
}

function BigInput({
  value,
  setValue,
  onSend,
  loading,
  model,
}: {
  value: string;
  setValue: (v: string) => void;
  onSend: () => void;
  loading: boolean;
  model: string;
}) {
  return (
    <div
      className="flex flex-col"
      style={{
        background: C.inputBg,
        borderRadius: 14,
        border: `1px solid rgba(255,255,255,0.1)`,
        boxShadow: '0 0 20px rgba(0,0,0,0.3)',
      }}
    >
      <textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            onSend();
          }
        }}
        placeholder="Ask anything about your project..."
        rows={1}
        className="max-h-40 resize-none bg-transparent px-4 pb-0.5 pt-3.5 text-[14.5px] leading-relaxed outline-none placeholder:text-[color:var(--ai-muted)]"
        style={{ color: C.textPri, ['--ai-muted' as string]: C.textMuted }}
      />
      <div className="flex items-center gap-0.5 px-3 pb-3 pt-1">
        <ToolbarIcon icon={Paperclip} />
        <ToolbarIcon icon={Smile} />
        <ToolbarIcon icon={Mic} />
        <div className="flex-1" />
        {model && (
          <span
            className="mr-2 flex items-center gap-1.5 rounded-full px-2.5 py-1"
            style={{ background: 'rgba(255,255,255,0.05)', border: `1px solid ${C.divider}` }}
          >
            <img src="/applad-mascot-head.png" alt="" className="h-[11px] w-[11px] rounded-full object-cover" />
            <span style={{ color: C.textMuted }} className="text-[11.5px]">
              {model}
            </span>
          </span>
        )}
        <SendBtn onSend={onSend} loading={loading} />
      </div>
    </div>
  );
}

// ── Small controls ───────────────────────────────────────────────────────────

function IconBtn({ icon: Icon, onClick }: { icon: LucideIcon; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="rounded-lg p-1.5 text-[color:var(--ai-sec)] transition-colors hover:bg-white/[0.08] hover:text-white"
      style={{ ['--ai-sec' as string]: C.textSec }}
    >
      <Icon size={16} />
    </button>
  );
}

function ToolbarIcon({ icon: Icon }: { icon: LucideIcon }) {
  return (
    <button
      type="button"
      className="rounded-md p-1.5 text-[color:var(--ai-muted)] transition-colors hover:bg-white/[0.06] hover:text-[color:var(--ai-sec)]"
      style={{ ['--ai-muted' as string]: C.textMuted, ['--ai-sec' as string]: C.textSec }}
    >
      <Icon size={16} />
    </button>
  );
}

function SendBtn({ onSend, loading }: { onSend: () => void; loading: boolean }) {
  return (
    <button
      onClick={loading ? undefined : onSend}
      disabled={loading}
      className="flex h-8 w-8 items-center justify-center rounded-full transition-transform disabled:cursor-default"
      style={{
        background: loading ? 'rgba(255,255,255,0.06)' : `linear-gradient(135deg, ${C.accent}, ${C.accentDim})`,
      }}
    >
      {loading ? (
        <span
          className="ai-spin block rounded-full"
          style={{ width: 14, height: 14, border: `1.5px solid ${C.accent}99`, borderTopColor: 'transparent' }}
        />
      ) : (
        <ArrowUp size={15} color="#fff" />
      )}
    </button>
  );
}
