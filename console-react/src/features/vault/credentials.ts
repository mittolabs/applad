import {
  Database,
  Fingerprint,
  Key,
  Lock,
  ShieldCheck,
  Terminal,
  Webhook,
  type LucideIcon,
} from 'lucide-react';

/* Shared credential metadata — ports the type/icon/color tables from
 * vault_page.dart so the list, badges, form and detail views stay in sync. */

export const CRED_TYPES = [
  'generic',
  'api_key',
  'database',
  'ssh',
  'webhook',
  'tls',
  'oauth2',
] as const;

export type CredType = (typeof CRED_TYPES)[number];

export const CRED_TYPE_OPTIONS: { value: CredType; label: string }[] = CRED_TYPES.map(
  (t) => ({ value: t, label: t.replace(/_/g, ' ') }),
);

export function typeIcon(type: string): LucideIcon {
  switch (type) {
    case 'api_key':
      return Key;
    case 'database':
      return Database;
    case 'ssh':
      return Terminal;
    case 'webhook':
      return Webhook;
    case 'tls':
      return Lock;
    case 'oauth2':
      return Fingerprint;
    default:
      return ShieldCheck;
  }
}

export function typeColor(type: string): string {
  switch (type) {
    case 'api_key':
      return '#6C47FF';
    case 'database':
      return '#0EA5E9';
    case 'ssh':
      return '#10B981';
    case 'webhook':
      return '#F59E0B';
    case 'tls':
      return '#EF4444';
    case 'oauth2':
      return '#8B5CF6';
    default:
      return '#6B7280';
  }
}

export function actionColor(action: string): string {
  switch (action) {
    case 'create':
      return '#10B981';
    case 'read':
      return '#3472A4';
    case 'update':
      return '#F59E0B';
    case 'delete':
      return '#EF4444';
    case 'rotate':
      return '#8B5CF6';
    default:
      return '#6B7280';
  }
}
