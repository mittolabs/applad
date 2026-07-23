import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import { useOrgStore } from '../stores/org';

export type NoticeRegion = 'app.top' | 'page.top' | 'project.top';

export interface NoticeAction {
  label: string;
  href: string;
}

export interface Theme {
  background?: string;
  gradientTo?: string;
  gradientAngle?: number;
  image?: string;
  effect?: 'snow' | 'confetti' | 'shimmer' | 'pulse' | 'twinkle';
  textColor?: string;
  accentColor?: string;
  height?: 'compact' | 'normal' | 'tall';
  align?: 'left' | 'center';
  icon?: string;
}

export interface Notice {
  id: string;
  level: 'info' | 'warn' | 'critical';
  title: string;
  body?: string;
  action?: NoticeAction;
  dismissible: boolean;
  scope?: 'org' | 'project';
  region: NoticeRegion;
  /** Presentation from a vocabulary the server validated. Never markup. */
  theme?: Theme;
}

export interface Limit {
  limit: number;
  used: number;
  scope: 'org' | 'project';
}

export interface Entitlements {
  features: Record<string, boolean>;
  limits: Record<string, Limit>;
  notices: Notice[];
}

const UNLIMITED: Entitlements = { features: {}, limits: {}, notices: [] };

/**
 * What this subject may use, and anything to tell them about it.
 *
 * On a default install the server answers unlimited with no notices, so every
 * gate below is open and no banner renders. A request that fails is treated the
 * same way: this drives what we OFFER, and the server is what actually enforces,
 * so being generous here can only ever cost a wasted request, never a breach.
 */
export function useEntitlements() {
  // Scoped to the current organization: limits like project count are counted
  // against the org, and a notice is usually addressed to one.
  const orgId = useOrgStore((s) => s.currentOrgId);
  const { data } = useQuery({
    queryKey: ['entitlements', orgId],
    queryFn: async () =>
      (await api.get<Entitlements>('/entitlements', { params: orgId ? { org: orgId } : undefined }))
        .data,
    staleTime: 60_000,
    retry: 1,
  });
  return data ?? UNLIMITED;
}

export interface Gate {
  allowed: boolean;
  reason?: string;
  action?: NoticeAction;
  /** Present when the denial came from a metered allowance. */
  limit?: Limit;
}

const ALLOWED: Gate = { allowed: true };

/**
 * Whether to offer an action, and what to say when we do not.
 *
 * The capability key must exist in the server's registry. This is a HINT that
 * mirrors the server: the server enforces the same key and answers 402, so a
 * drift between the two shows up as a button that fails rather than as a hole.
 */
export function useGate(key: string): Gate {
  const ent = useEntitlements();

  const limit = ent.limits[key];
  if (limit && limit.limit >= 0 && limit.used >= limit.limit) {
    return {
      allowed: false,
      reason: `Limit reached (${limit.used}/${limit.limit})`,
      limit,
    };
  }
  if (ent.features[key] === false) {
    return { allowed: false, reason: 'Not available on this plan' };
  }
  return ALLOWED;
}

const LEVEL_RANK: Record<Notice['level'], number> = { critical: 3, warn: 2, info: 1 };

/**
 * The ONE notice to show in a region.
 *
 * Stacking banners pushes the product down the page and buries the message that
 * matters, so a region shows a single notice: the most severe, and among equals
 * the most recent (the server orders newest first). Core decides this rather
 * than trusting whatever supplies notices to send only one, because the console
 * is what has to stay usable.
 */
export function useNotices(region: NoticeRegion): Notice[] {
  const ent = useEntitlements();
  const inRegion = ent.notices.filter((n) => n.region === region);
  if (inRegion.length <= 1) return inRegion;

  const winner = inRegion.reduce((best, n) =>
    (LEVEL_RANK[n.level] ?? 0) > (LEVEL_RANK[best.level] ?? 0) ? n : best,
  );
  return [winner];
}

/**
 * Dismiss a notice for the signed-in user, permanently.
 *
 * Recorded server-side rather than in this tab, so a refresh or another device
 * does not resurrect something the user already cleared. It is per user: the
 * banner was shown to everyone in the organization, and each clears their own.
 */
export function useDismissNotice() {
  const qc = useQueryClient();
  const { mutate } = useMutation({
    mutationFn: async (id: string) => {
      await api.post(`/entitlements/notices/${encodeURIComponent(id)}/dismiss`);
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['entitlements'] });
    },
  });
  return mutate;
}
