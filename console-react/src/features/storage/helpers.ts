import {
  File as FileIcon,
  FileText,
  Image as ImageIcon,
  type LucideIcon,
  Video,
} from 'lucide-react';
import { api } from '@/api/client';

const MONTHS = [
  '',
  'Jan',
  'Feb',
  'Mar',
  'Apr',
  'May',
  'Jun',
  'Jul',
  'Aug',
  'Sep',
  'Oct',
  'Nov',
  'Dec',
];

/** "Jan 5, 2025" — ports storage_page.dart _fmtDate. */
export function fmtDate(raw: unknown): string {
  if (raw == null) return '—';
  const dt = new Date(String(raw));
  if (Number.isNaN(dt.getTime())) return '—';
  return `${MONTHS[dt.getMonth() + 1]} ${dt.getDate()}, ${dt.getFullYear()}`;
}

/** "Today" / "Yesterday" / "N days ago" / date — ports _FileRow._timeAgo. */
export function timeAgo(raw: unknown): string {
  if (raw == null) return '—';
  const dt = new Date(String(raw));
  if (Number.isNaN(dt.getTime())) return '—';
  const days = Math.floor((Date.now() - dt.getTime()) / 86_400_000);
  if (days <= 0) return 'Today';
  if (days === 1) return 'Yesterday';
  if (days < 30) return `${days} days ago`;
  return `${MONTHS[dt.getMonth() + 1]} ${dt.getDate()}, ${dt.getFullYear()}`;
}

/** Short size for tables — ports _FileRow._formatSize. */
export function formatSize(bytes: unknown): string {
  const b = typeof bytes === 'number' ? bytes : Number(bytes ?? 0);
  if (!Number.isFinite(b) || b < 1024) return `${Number.isFinite(b) ? b : 0} B`;
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(0)} KB`;
  return `${(b / (1024 * 1024)).toFixed(1)} MB`;
}

/** Precise size for the file detail — ports _FileDetailView._fmtBytes. */
export function fmtBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const i = Math.min(3, Math.floor(Math.log(bytes) / Math.log(1024)));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

/** Icon + tint for a file row based on its MIME type — ports _FileRow._mimeIcon. */
export function mimeIcon(mime: string): { icon: LucideIcon; color: string } {
  if (mime.startsWith('image/')) return { icon: ImageIcon, color: '#10B981' };
  if (mime.startsWith('video/')) return { icon: Video, color: '#7C3AED' };
  if (mime.includes('pdf')) return { icon: FileText, color: '#EF4444' };
  return { icon: FileIcon, color: 'var(--color-accent)' };
}

/**
 * Direct file view/download URL: `{origin}/v1/storage/buckets/{id}/files/{id}/view`.
 * Ports the Dart `${baseUrl}/storage/...` link. Uses the axios baseURL — absolute
 * when VITE_API_URL is set, otherwise the relative "/v1" resolved against origin.
 */
export function fileViewUrl(bucketId: string, fileId: string): string {
  const base = api.defaults.baseURL ?? '/v1';
  const origin = base.startsWith('http') ? base : `${window.location.origin}${base}`;
  return `${origin}/storage/buckets/${bucketId}/files/${fileId}/view`;
}
