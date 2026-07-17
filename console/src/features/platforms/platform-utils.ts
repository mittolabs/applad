import { Globe, Layers, Monitor, Server, Smartphone, type LucideIcon } from 'lucide-react';
import type { Row } from '@/components/data-table';

/* Type metadata — ports the `_types` list + switch helpers in platforms_page.dart. */

export interface PlatformType {
  id: string;
  label: string;
  icon: LucideIcon;
}

export const PLATFORM_TYPES: PlatformType[] = [
  { id: 'web', label: 'Web', icon: Globe },
  { id: 'ios', label: 'iOS', icon: Smartphone },
  { id: 'android', label: 'Android', icon: Smartphone },
  { id: 'desktop', label: 'Desktop', icon: Monitor },
  { id: 'server', label: 'Server', icon: Server },
];

export function typeLabel(type: string): string {
  return PLATFORM_TYPES.find((t) => t.id === type)?.label ?? type;
}

export function typeIconFor(type: string): LucideIcon {
  switch (type) {
    case 'web':
      return Globe;
    case 'ios':
    case 'android':
      return Smartphone;
    case 'desktop':
      return Monitor;
    case 'server':
      return Server;
    default:
      return Layers;
  }
}

/** iOS badge uses the secondary purple accent; everything else uses the primary accent. */
export function typeBadgeColor(type: string): string {
  return type === 'ios' ? 'var(--color-accent-2)' : 'var(--color-accent)';
}

export function identityLabel(type: string): string {
  switch (type) {
    case 'web':
      return 'Hostname';
    case 'ios':
      return 'Bundle ID';
    case 'android':
      return 'Package name';
    case 'desktop':
      return 'App identifier';
    case 'server':
      return 'Hostname / IP';
    default:
      return 'Identifier';
  }
}

export function identityHint(type: string): string {
  switch (type) {
    case 'web':
      return 'myapp.com';
    case 'ios':
    case 'android':
    case 'desktop':
      return 'com.example.myapp';
    case 'server':
      return '192.168.1.1';
    default:
      return '';
  }
}

export function identityValue(row: Row): string {
  return String(row['hostname'] ?? '');
}

export function platformId(row: Row): string {
  return String(row['$id'] ?? row['id'] ?? '');
}

export function fmtDate(v: unknown): string {
  if (v == null || v === '') return '—';
  return String(v).split('T')[0];
}
