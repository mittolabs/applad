import {
  Box,
  Code2,
  Cpu,
  FileCode,
  Gem,
  Server,
  Settings2,
  Terminal,
  Zap,
  type LucideIcon,
} from 'lucide-react';

/* Ports the hardcoded `_runtimes` list from functions_page.dart. Keep the ids,
 * labels, and ordering identical — they are the API's runtime identifiers. */
export interface Runtime {
  id: string;
  label: string;
  icon: LucideIcon;
  /** Monaco language id for inline source editing. */
  language: string;
}

export const RUNTIMES: Runtime[] = [
  { id: 'node-18', label: 'Node.js 18', icon: Server, language: 'javascript' },
  { id: 'node-20', label: 'Node.js 20', icon: Server, language: 'javascript' },
  { id: 'node-22', label: 'Node.js 22', icon: Server, language: 'javascript' },
  { id: 'bun-1', label: 'Bun 1', icon: Zap, language: 'javascript' },
  { id: 'python-3.11', label: 'Python 3.11', icon: Code2, language: 'python' },
  { id: 'python-3.12', label: 'Python 3.12', icon: Code2, language: 'python' },
  { id: 'go-1.22', label: 'Go 1.22', icon: Cpu, language: 'go' },
  { id: 'dart-3', label: 'Dart 3', icon: Terminal, language: 'dart' },
  { id: 'rust-1', label: 'Rust 1', icon: Settings2, language: 'rust' },
  { id: 'ruby-3', label: 'Ruby 3', icon: Gem, language: 'ruby' },
  { id: 'php-8', label: 'PHP 8', icon: FileCode, language: 'php' },
  { id: 'custom', label: 'Custom', icon: Box, language: 'plaintext' },
];

export function runtimeById(id: string): Runtime {
  return RUNTIMES.find((r) => r.id === id) ?? RUNTIMES[RUNTIMES.length - 1];
}

export const RUNTIME_OPTIONS = RUNTIMES.map((r) => ({ value: r.id, label: r.label }));

/** Relative-time formatter — ports functions_page.dart `_relativeTime`. */
export function relativeTime(iso: string): string {
  if (!iso) return '—';
  const dt = new Date(iso);
  if (Number.isNaN(dt.getTime())) return '—';
  const diffMs = Date.now() - dt.getTime();
  const days = Math.floor(diffMs / 86_400_000);
  if (days > 365) return `${Math.floor(days / 365)}y ago`;
  if (days > 30) return `${Math.floor(days / 30)}mo ago`;
  if (days > 0) return `${days}d ago`;
  const hours = Math.floor(diffMs / 3_600_000);
  if (hours > 0) return `${hours}h ago`;
  const minutes = Math.floor(diffMs / 60_000);
  if (minutes > 0) return `${minutes}m ago`;
  return 'just now';
}

/** Short date (YYYY-MM-DD) for table cells. */
export function shortDate(v: unknown): string {
  const s = v == null ? '' : String(v);
  return s ? (s.split('T')[0] ?? s) : '—';
}
