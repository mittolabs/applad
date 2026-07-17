import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';

/* Ports url_utils.dart tabFromQuery + withQuery for tabs.
 * Syncs the active tab to ?tab=<name>. Returns [tab, setTab]. */
export function useTabParam(tabs: string[], defaultTab?: string): [string, (tab: string) => void] {
  const [params, setParams] = useSearchParams();
  const fallback = defaultTab ?? tabs[0];
  const current = params.get('tab');
  const tab = current && tabs.includes(current) ? current : fallback;

  const setTab = useCallback(
    (next: string) => {
      const p = new URLSearchParams(params);
      if (next === fallback) p.delete('tab');
      else p.set('tab', next);
      p.delete('page'); // reset pagination when switching tabs
      setParams(p, { replace: true });
    },
    [params, setParams, fallback],
  );

  return [tab, setTab];
}

/** Index-based variant for <PageTabs> which is index-driven. */
export function useTabIndex(
  tabs: string[],
  defaultTab?: string,
): [number, (index: number) => void] {
  const [tab, setTab] = useTabParam(tabs, defaultTab);
  const index = Math.max(0, tabs.indexOf(tab));
  return [index, (i: number) => setTab(tabs[i])];
}
