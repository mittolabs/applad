import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';

/* Ports url_utils.dart tabFromQuery + withQuery for tabs.
 * Syncs the active tab to ?tab=<name>. Returns [tab, setTab].
 *
 * `key` names the query parameter, so a detail view nested inside a list can
 * keep its own tab without the two fighting over ?tab. */
export function useTabParam(
  tabs: string[],
  defaultTab?: string,
  key = 'tab',
): [string, (tab: string) => void] {
  const [params, setParams] = useSearchParams();
  const fallback = defaultTab ?? tabs[0];
  const current = params.get(key);
  const tab = current && tabs.includes(current) ? current : fallback;

  const setTab = useCallback(
    (next: string) => {
      const p = new URLSearchParams(params);
      if (next === fallback) p.delete(key);
      else p.set(key, next);
      p.delete('page'); // reset pagination when switching tabs
      setParams(p, { replace: true });
    },
    [params, setParams, fallback, key],
  );

  return [tab, setTab];
}

/** Index-based variant for <PageTabs> which is index-driven. */
export function useTabIndex(
  tabs: string[],
  defaultTab?: string,
  key = 'tab',
): [number, (index: number) => void] {
  const [tab, setTab] = useTabParam(tabs, defaultTab, key);
  const index = Math.max(0, tabs.indexOf(tab));
  return [index, (i: number) => setTab(tabs[i])];
}
