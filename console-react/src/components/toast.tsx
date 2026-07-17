import { useEffect } from 'react';
import { create } from 'zustand';
import { AlertTriangle, Check, Info, X } from 'lucide-react';
import { cn } from '@/lib/utils';

/* Minimal toast system (replaces Flutter SnackBars). Call `toast.success(...)`,
 * `toast.error(...)`, `toast.info(...)` from anywhere. Mount <Toaster/> once. */

type ToastKind = 'success' | 'error' | 'info';
interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}

interface ToastState {
  toasts: Toast[];
  push: (kind: ToastKind, message: string) => void;
  dismiss: (id: number) => void;
}

let seq = 0;
const useToastStore = create<ToastState>((set) => ({
  toasts: [],
  push: (kind, message) => {
    const id = ++seq;
    set((s) => ({ toasts: [...s.toasts, { id, kind, message }] }));
    setTimeout(() => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })), 4000);
  },
  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}));

export const toast = {
  success: (m: string) => useToastStore.getState().push('success', m),
  error: (m: string) => useToastStore.getState().push('error', m),
  info: (m: string) => useToastStore.getState().push('info', m),
};

const ICON = { success: Check, error: AlertTriangle, info: Info };
const COLOR = {
  success: 'var(--status-success)',
  error: 'var(--color-danger)',
  info: 'var(--status-info)',
};

export function Toaster() {
  const toasts = useToastStore((s) => s.toasts);
  const dismiss = useToastStore((s) => s.dismiss);
  return (
    <div className="pointer-events-none fixed bottom-4 right-4 z-[100] flex flex-col gap-2">
      {toasts.map((t) => {
        const Icon = ICON[t.kind];
        return (
          <ToastItem key={t.id} onDone={() => dismiss(t.id)}>
            <div className="pointer-events-auto flex min-w-[260px] max-w-sm items-center gap-2.5 rounded-[var(--radius)] border border-border bg-[var(--popup-surface)] px-3 py-2.5 shadow-[0_8px_32px_var(--shadow)]">
              <Icon size={15} style={{ color: COLOR[t.kind] }} />
              <span className="flex-1 text-[length:var(--text-body)] text-text-primary">
                {t.message}
              </span>
              <button
                onClick={() => dismiss(t.id)}
                className="text-text-subtle hover:text-text-primary"
                aria-label="Dismiss"
              >
                <X size={14} />
              </button>
            </div>
          </ToastItem>
        );
      })}
    </div>
  );
}

function ToastItem({ children, onDone }: { children: React.ReactNode; onDone: () => void }) {
  useEffect(() => {
    return () => {};
  }, [onDone]);
  return <div className={cn('animate-in fade-in-0 slide-in-from-bottom-2')}>{children}</div>;
}
