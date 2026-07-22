import axios from 'axios';
import {
  AlertTriangle,
  Lock,
  RefreshCw,
  ServerCrash,
  WifiOff,
} from 'lucide-react';
import { Button } from './ui/button';
import { friendlyError } from '@/api/client';
import { cn } from '@/lib/utils';

/* Ports app_error_state.dart — classifies the error into an icon + message
 * and offers a retry. friendlyError() lives in the api layer (shared). */
function iconFor(error: unknown) {
  if (axios.isAxiosError(error)) {
    const status = error.response?.status;
    if (error.code === 'ERR_NETWORK') return { Icon: WifiOff, color: 'var(--status-warning)' };
    if (error.code === 'ECONNABORTED') return { Icon: WifiOff, color: 'var(--status-warning)' };
    if (status === 401 || status === 403) return { Icon: Lock, color: 'var(--status-danger)' };
    if (status && status >= 500) return { Icon: ServerCrash, color: 'var(--status-danger)' };
  }
  return { Icon: AlertTriangle, color: 'var(--status-warning)' };
}

export function ErrorState({
  error,
  onRetry,
  className,
}: {
  error: unknown;
  onRetry?: () => void;
  className?: string;
}) {
  const { Icon, color } = iconFor(error);
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center px-6 py-16 text-center',
        className,
      )}
    >
      <div
        className="mb-4 flex h-12 w-12 items-center justify-center rounded-[var(--radius-12)] border border-border"
        style={{ color, backgroundColor: `color-mix(in srgb, ${color} 12%, transparent)` }}
      >
        <Icon size={22} />
      </div>
      <div className="text-[length:var(--text-subhead)] font-semibold text-text-primary">
        Something went wrong
      </div>
      <div className="mt-1 max-w-sm text-[length:var(--text-body)] text-text-muted">
        {friendlyError(error)}
      </div>
      {onRetry && (
        <Button variant="secondary" className="mt-5" onClick={onRetry}>
          <RefreshCw size={14} />
          Try again
        </Button>
      )}
    </div>
  );
}
