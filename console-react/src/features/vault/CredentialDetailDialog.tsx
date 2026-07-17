import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Copy, Eye, Lock, Pencil, Unlock } from 'lucide-react';
import { api, friendlyError } from '@/api/client';
import { toast } from '@/components/toast';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { PageTabs } from '@/components/page-tabs';
import { ErrorState } from '@/components/error-state';
import { typeIcon } from './credentials';
import { ActionBadge, ExpiryChip, TypeBadge } from './CredentialBadges';
import type { Row } from '@/components/data-table';

/* View a credential's metadata + access log — ports _CredentialDetailModal. */
export function CredentialDetailDialog({
  cred,
  open,
  onOpenChange,
  onEdit,
}: {
  cred: Row | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onEdit: (cred: Row) => void;
}) {
  const [tab, setTab] = useState(0);

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) setTab(0);
        onOpenChange(o);
      }}
    >
      <DialogContent width={520}>
        <DialogHeader>
          <DialogTitle>{String(cred?.['name'] ?? 'Credential')}</DialogTitle>
        </DialogHeader>
        <DialogBody>
          <PageTabs tabs={['Details', 'Access log']} selected={tab} onChange={setTab} />
          <div className="mt-4 min-h-[280px]">
            {cred &&
              (tab === 0 ? (
                <DetailsTab cred={cred} />
              ) : (
                <AccessLogTab credId={String(cred['$id'])} open={open} />
              ))}
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          {cred && (
            <Button
              onClick={() => {
                onOpenChange(false);
                onEdit(cred);
              }}
            >
              <Pencil size={14} />
              Edit
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function MetaRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-3 pb-2.5">
      <span className="w-28 shrink-0 text-[length:var(--text-label)] text-text-muted">{label}</span>
      <div className="flex-1 text-[length:var(--text-body)] text-text-primary">{children}</div>
    </div>
  );
}

function DetailsTab({ cred }: { cred: Row }) {
  const [revealed, setRevealed] = useState<string | null>(null);
  const [revealing, setRevealing] = useState(false);

  const type = String(cred['type'] ?? 'generic');
  const protectedFlag = cred['protected'] === true;
  const keyVersion = Number(cred['keyVersion'] ?? 0);
  const expiresAt = cred['expiresAt'] as string | undefined;
  const createdAt = cred['$createdAt'] as string | undefined;
  const TypeIcon = typeIcon(type);

  const reveal = async () => {
    setRevealing(true);
    try {
      const res = await api.get(`/credentials/${String(cred['$id'])}`);
      setRevealed(String((res.data as Record<string, unknown>)['data'] ?? ''));
    } catch (e) {
      toast.error(friendlyError(e));
    } finally {
      setRevealing(false);
    }
  };

  const copy = async () => {
    if (revealed === null) return;
    try {
      await navigator.clipboard.writeText(revealed);
      toast.success('Copied to clipboard');
    } catch {
      /* clipboard unavailable */
    }
  };

  return (
    <div className="flex flex-col">
      <MetaRow label="Type">
        <span className="inline-flex items-center gap-2">
          <TypeIcon size={13} style={{ color: 'var(--text-muted)' }} />
          <TypeBadge type={type} />
        </span>
      </MetaRow>
      <MetaRow label="Protected">
        <span className="inline-flex items-center gap-1.5">
          {protectedFlag ? (
            <Lock size={13} style={{ color: '#F59E0B' }} />
          ) : (
            <Unlock size={13} style={{ color: 'var(--text-muted)' }} />
          )}
          {protectedFlag ? 'Yes (API key required)' : 'No'}
        </span>
      </MetaRow>
      <MetaRow label="Key version">v{keyVersion}</MetaRow>
      {expiresAt && (
        <MetaRow label="Expires">
          <ExpiryChip expiresAt={expiresAt} />
        </MetaRow>
      )}
      {createdAt && <MetaRow label="Created">{createdAt.slice(0, 10)}</MetaRow>}

      <div className="mt-3">
        {revealed !== null ? (
          <div className="flex flex-col gap-1.5">
            <span className="text-[length:var(--text-label)] text-text-muted">Secret value</span>
            <div className="flex items-center gap-2 rounded-[var(--radius)] border border-border bg-surface p-3">
              <span className="flex-1 select-all break-all font-[family-name:var(--font-mono)] text-[length:var(--text-caption)] text-text-primary">
                {revealed}
              </span>
              <button
                type="button"
                onClick={copy}
                className="text-text-muted transition-colors hover:text-text-primary"
                aria-label="Copy secret"
              >
                <Copy size={14} />
              </button>
            </div>
          </div>
        ) : (
          <Button variant="outline" size="sm" loading={revealing} onClick={reveal}>
            <Eye size={14} />
            Reveal secret value
          </Button>
        )}
      </div>
    </div>
  );
}

function AccessLogTab({ credId, open }: { credId: string; open: boolean }) {
  const query = useQuery({
    queryKey: ['credential-accesses', credId],
    enabled: open,
    queryFn: async () => {
      const res = await api.get(`/credentials/${credId}/accesses`, { params: { limit: 50 } });
      return res.data as Record<string, unknown>;
    },
  });

  if (query.error) return <ErrorState error={query.error} onRetry={() => query.refetch()} />;
  if (query.isLoading) {
    return (
      <div className="py-10 text-center text-[length:var(--text-body)] text-text-muted">
        Loading…
      </div>
    );
  }

  const accesses = (query.data?.['accesses'] as Row[] | undefined) ?? [];
  if (accesses.length === 0) {
    return (
      <div className="py-10 text-center text-[length:var(--text-body)] text-text-muted">
        No access events yet
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      {accesses.map((a, i) => {
        const at = String(a['accessedAt'] ?? '');
        return (
          <div
            key={i}
            className="flex items-center gap-2.5 border-b border-border/40 py-2 last:border-0"
          >
            <ActionBadge action={String(a['action'] ?? '')} />
            <span className="flex-1 text-[length:var(--text-caption)] text-text-primary">
              {String(a['ip'] ?? '') || 'unknown'}
            </span>
            <span className="text-[length:var(--text-caption)] text-text-muted">
              {at.length >= 16 ? at.slice(0, 16).replace('T', ' ') : at}
            </span>
          </div>
        );
      })}
    </div>
  );
}
