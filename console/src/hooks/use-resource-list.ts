import { useCallback, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api } from '@/api/client';

/*
 * useResourceList — the anti-rewrite lever. Encapsulates search + perPage +
 * page (URL-synced via ?page) + a TanStack Query fetch, matching the Flutter
 * per-page-provider pattern (params {limit, offset, search?}; response
 * {<itemsKey>: [...], total: N}). Every list page is a thin config over this.
 */
export interface UseResourceListOptions {
  /** API path relative to /v1, e.g. "/databases". */
  endpoint: string;
  /** Key in the response holding the array, e.g. "databases". */
  itemsKey: string;
  /** react-query key parts beyond endpoint (e.g. the current project id). */
  scope?: unknown[];
  /** Extra static query params merged into every request. */
  params?: Record<string, string | number | undefined>;
  defaultPerPage?: number;
  enabled?: boolean;
}

export interface UseResourceListResult<T> {
  rows: T[];
  total: number;
  page: number;
  perPage: number;
  search: string;
  filters: Record<string, string | null>;
  isLoading: boolean;
  isFetching: boolean;
  error: unknown;
  setSearch: (v: string) => void;
  runSearch: () => void;
  setPerPage: (n: number) => void;
  nextPage: () => void;
  prevPage: () => void;
  setFilters: (values: Record<string, string | null>) => void;
  refetch: () => void;
}

export function useResourceList<T = Record<string, unknown>>(
  opts: UseResourceListOptions,
): UseResourceListResult<T> {
  const { endpoint, itemsKey, scope = [], params = {}, defaultPerPage = 12, enabled = true } = opts;

  const [urlParams, setUrlParams] = useSearchParams();
  const page = Math.max(1, Number(urlParams.get('page') ?? '1'));
  const [perPage, setPerPageState] = useState(defaultPerPage);
  const [searchInput, setSearchInput] = useState('');
  const [activeSearch, setActiveSearch] = useState('');
  const [filters, setFiltersState] = useState<Record<string, string | null>>({});

  const setPage = useCallback(
    (p: number) => {
      const next = new URLSearchParams(urlParams);
      if (p <= 1) next.delete('page');
      else next.set('page', String(p));
      setUrlParams(next, { replace: true });
    },
    [urlParams, setUrlParams],
  );

  const offset = (page - 1) * perPage;
  const activeFilters = useMemo(
    () => Object.fromEntries(Object.entries(filters).filter(([, v]) => v)),
    [filters],
  );

  const query = useQuery({
    queryKey: [endpoint, ...scope, { offset, perPage, activeSearch, activeFilters, params }],
    enabled,
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const res = await api.get(endpoint, {
        params: {
          limit: perPage,
          offset,
          ...(activeSearch ? { search: activeSearch } : {}),
          ...activeFilters,
          ...params,
        },
      });
      return res.data as Record<string, unknown>;
    },
  });

  const data = query.data;
  const rows = (data?.[itemsKey] as T[] | undefined) ?? [];
  const total = Number(data?.['total'] ?? rows.length);

  return {
    rows,
    total,
    page,
    perPage,
    search: searchInput,
    filters,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    error: query.error,
    setSearch: setSearchInput,
    runSearch: () => {
      setActiveSearch(searchInput);
      setPage(1);
    },
    setPerPage: (n) => {
      setPerPageState(n);
      setPage(1);
    },
    nextPage: () => setPage(page + 1),
    prevPage: () => setPage(Math.max(1, page - 1)),
    setFilters: (values) => {
      setFiltersState(values);
      setPage(1);
    },
    refetch: () => void query.refetch(),
  };
}
