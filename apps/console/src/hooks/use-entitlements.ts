import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';

export type NoticeRegion = 'app.top' | 'page.top' | 'project.top';

export interface NoticeAction {
  label: string;
  href: string;
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
  const { data } = useQuery({
    queryKey: ['entitlements'],
    queryFn: async () => (await api.get<Entitlements>('/entitlements')).data,
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

/** Notices for one region, newest-first as the server ordered them. */
export function useNotices(region: NoticeRegion): Notice[] {
  const ent = useEntitlements();
  return ent.notices.filter((n) => n.region === region);
}
