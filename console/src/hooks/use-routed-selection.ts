import { useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

/*
 * Selection that lives in the address.
 *
 * Holding "which row is open" in component state means the URL describes the
 * list while the screen shows a record: a refresh, a bookmark, a shared link
 * and the back button all lose the place. Putting it in the path costs one
 * route entry and makes the address say what is on screen.
 */

export interface RoutedSelection {
  /** The record currently open, or null on the list. */
  id: string | null;
  /** Open a record, or pass null to return to the list. */
  select: (id: string | null) => void;
  clear: () => void;
}

/**
 * Backs selection with a route parameter.
 *
 * `segment` is the feature's own path segment — "functions", "storage" — and
 * `param` the route parameter holding the id.
 */
export function useRoutedSelection(segment: string, param: string): RoutedSelection {
  const navigate = useNavigate();
  const params = useParams<Record<string, string>>();
  const projectId = params.projectId;
  const id = params[param] ?? null;

  const select = useCallback(
    (next: string | null) => {
      const base = `/project/${projectId}/${segment}`;
      navigate(next ? `${base}/${next}` : base);
    },
    [navigate, projectId, segment],
  );

  const clear = useCallback(() => select(null), [select]);

  return { id, select, clear };
}

/**
 * The same, one level deeper: a record inside a record, such as a file inside
 * a bucket or a table inside a database.
 */
export function useNestedSelection(
  segment: string,
  parentParam: string,
  childParam: string,
): {
  parentId: string | null;
  childId: string | null;
  select: (parent: string | null, child?: string | null) => void;
} {
  const navigate = useNavigate();
  const params = useParams<Record<string, string>>();
  const projectId = params.projectId;

  const select = useCallback(
    (parent: string | null, child?: string | null) => {
      const base = `/project/${projectId}/${segment}`;
      if (!parent) {
        navigate(base);
        return;
      }
      navigate(child ? `${base}/${parent}/${child}` : `${base}/${parent}`);
    },
    [navigate, projectId, segment],
  );

  return {
    parentId: params[parentParam] ?? null,
    childId: params[childParam] ?? null,
    select,
  };
}
