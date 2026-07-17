import { ArrowLeft, Tag } from 'lucide-react';
import { IdText } from '@/components/id-text';
import { MessageStatusChip, formatDate, typeIcon, typeName } from './shared';

/* Ports _MessageDetail — read-only cards for a selected message. */

function Card({ children }: { children: React.ReactNode }) {
  return (
    <div className="w-full rounded-[var(--radius)] border border-border bg-surface p-5">
      {children}
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[length:var(--text-caption)] font-medium text-text-subtle">
        {label}
      </span>
      <div className="w-full rounded-[var(--radius-6)] border border-border bg-fill px-3 py-2.5 text-[length:var(--text-body)] text-text-secondary">
        {value}
      </div>
    </div>
  );
}

export interface MessageRow {
  id: string;
  type: string;
  subject: string;
  body: string;
  status: string;
  createdAt: string;
  recipients: string[];
}

export function toMessageRow(row: Record<string, unknown>): MessageRow {
  return {
    id: String(row.id ?? row.$id ?? ''),
    type: String(row.type ?? ''),
    subject: String(row.subject ?? ''),
    body: String(row.body ?? ''),
    status: String(row.status ?? 'processing'),
    createdAt: String(row.createdAt ?? row.$createdAt ?? ''),
    recipients: Array.isArray(row.recipients)
      ? (row.recipients as unknown[]).map((r) => String(r))
      : [],
  };
}

export function MessageDetail({
  msg,
  onBack,
}: {
  msg: MessageRow;
  onBack: () => void;
}) {
  const TypeIcon = typeIcon(msg.type);
  const label = typeName(msg.type);

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2 pb-6">
        <button
          type="button"
          onClick={onBack}
          className="text-text-muted transition-colors hover:text-text-primary"
          aria-label="Back"
        >
          <ArrowLeft size={18} />
        </button>
        <h2 className="text-[length:var(--text-title)] font-semibold text-text-primary">
          {msg.subject || label}
        </h2>
        <span className="ml-1 inline-flex items-center gap-1 rounded-[var(--radius-sm)] bg-fill px-2 py-[3px]">
          <Tag size={10} className="text-text-subtle" />
          <IdText id={msg.id} fontSize={11} />
        </span>
      </div>

      <div className="flex flex-1 flex-col gap-3 overflow-y-auto pb-6">
        <Card>
          <div className="flex items-center gap-2">
            <TypeIcon size={16} className="text-text-muted" />
            <span className="text-[length:var(--text-control)] font-medium text-text-primary">
              {label}
            </span>
            <span className="flex-1" />
            <span className="text-[length:var(--text-label)] text-text-subtle">
              Created: {formatDate(msg.createdAt)}
            </span>
            <MessageStatusChip status={msg.status} />
          </div>
        </Card>

        <Card>
          <div className="text-[length:var(--text-control)] font-medium text-text-primary">
            Message
          </div>
          <div className="my-3 border-t border-border" />
          <div className="flex flex-col gap-3">
            {msg.subject && <DetailRow label="Subject" value={msg.subject} />}
            <DetailRow
              label={msg.type === 'push' ? 'Body' : 'Message'}
              value={msg.body || '-'}
            />
          </div>
        </Card>

        <Card>
          <div className="text-[length:var(--text-control)] font-medium text-text-primary">
            Targets
          </div>
          <div className="my-3 border-t border-border" />
          {msg.recipients.length === 0 ? (
            <div className="py-1 text-[length:var(--text-body)] text-text-subtle">
              No targets selected
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {msg.recipients.map((r) => (
                <div key={r} className="flex items-center gap-2">
                  <TypeIcon size={13} className="text-text-muted" />
                  <span className="text-[length:var(--text-body)] text-text-secondary">{r}</span>
                </div>
              ))}
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}
