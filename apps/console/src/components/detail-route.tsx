import { useQuery } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { api } from '@/api/client';
import { ErrorState } from '@/components/error-state';
import { type Row } from '@/components/data-table';

/*
 * Resolves a record from an id in the address.
 *
 * A detail view wants the whole record while the URL carries only an id, so
 * every routed detail otherwise reinvents the same fetch. Looking it up in the
 * already-loaded list is not enough: on a cold refresh the list has not loaded
 * yet, and the record may be on another page of it.
 */
export function DetailRoute({
  endpoint,
  id,
  queryKey,
  children,
}: {
  /** Resource path without the id, e.g. "/functions". */
  endpoint: string;
  id: string;
  /** Extra cache key parts, for resources scoped by a parent. */
  queryKey?: unknown[];
  children: (record: Row, refetch: () => void) => ReactNode;
}) {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['detail', endpoint, id, ...(queryKey ?? [])],
    queryFn: async () => (await api.get(`${endpoint}/${id}`)).data as Row,
  });

  if (isLoading) {
    return (
      <div className="p-6 text-[length:var(--text-body)] text-text-muted md:p-8">Loading...</div>
    );
  }

  if (error || !data) {
    return (
      <div className="p-6 md:p-8">
        <ErrorState error={error} onRetry={() => refetch()} />
      </div>
    );
  }

  return <>{children(data, () => refetch())}</>;
}
