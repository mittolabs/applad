import { useEffect, useState } from 'react';

/**
 * Tracks a CSS media query, re-rendering when it flips. SSR-safe (defaults to
 * false until mounted). Used by the shell to swap between the desktop rail and
 * the mobile bottom-nav layout, mirroring shell.dart `_isMobile` (width < 650).
 */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia(query).matches : false,
  );

  useEffect(() => {
    const mql = window.matchMedia(query);
    const onChange = () => setMatches(mql.matches);
    onChange();
    mql.addEventListener('change', onChange);
    return () => mql.removeEventListener('change', onChange);
  }, [query]);

  return matches;
}

/** Shell breakpoints — mirror the Flutter thresholds exactly. */
export const useIsMobile = () => useMediaQuery('(max-width: 649px)');
/** Below this the top nav collapses its buttons into an overflow menu. */
export const useIsNavCompact = () => useMediaQuery('(max-width: 779px)');
