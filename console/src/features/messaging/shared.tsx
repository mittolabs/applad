import { Bell, Mail, MessageSquare, type LucideIcon } from 'lucide-react';
import { StatusChip, type StatusVariant } from '@/components/status-chip';

/* Shared helpers for the messaging feature — message type → icon/label and a
 * status chip whose variant mapping matches _StatusBadge in messaging_page.dart. */

export type MsgType = 'email' | 'sms' | 'push';

export function typeIcon(type: string): LucideIcon {
  if (type === 'email') return Mail;
  if (type === 'sms') return MessageSquare;
  return Bell;
}

export function typeName(type: string): string {
  if (type === 'email') return 'Email';
  if (type === 'sms') return 'SMS';
  return 'Push';
}

const STATUS_VARIANT: Record<string, StatusVariant> = {
  sent: 'success',
  failed: 'danger',
  draft: 'neutral',
  processing: 'info',
};

export function MessageStatusChip({ status }: { status: string }) {
  const variant = STATUS_VARIANT[status] ?? 'info';
  return <StatusChip label={status || 'processing'} variant={variant} />;
}

export function formatDate(iso: string): string {
  if (!iso) return '';
  const dt = new Date(iso);
  if (Number.isNaN(dt.getTime())) return iso;
  const months = [
    'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
    'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
  ];
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${dt.getDate()} ${months[dt.getMonth()]} ${dt.getFullYear()}, ${pad(dt.getHours())}:${pad(dt.getMinutes())}`;
}
