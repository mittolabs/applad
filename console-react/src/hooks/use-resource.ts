import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/api/client';

/*
 * useResource — standard create/update/delete mutations for a REST collection,
 * with automatic invalidation of the matching list query key. Keeps every
 * feature's CRUD wiring to a few lines instead of bespoke mutation code.
 */
export function useResource(endpoint: string, invalidateKey?: unknown[]) {
  const qc = useQueryClient();
  const invalidate = () =>
    qc.invalidateQueries({ queryKey: invalidateKey ?? [endpoint] });

  const create = useMutation({
    mutationFn: (data: unknown) => api.post(endpoint, data).then((r) => r.data),
    onSuccess: invalidate,
  });

  const update = useMutation({
    mutationFn: ({ id, data, method = 'put' }: { id: string; data: unknown; method?: 'put' | 'patch' }) =>
      (method === 'patch'
        ? api.patch(`${endpoint}/${id}`, data)
        : api.put(`${endpoint}/${id}`, data)
      ).then((r) => r.data),
    onSuccess: invalidate,
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.delete(`${endpoint}/${id}`).then((r) => r.data),
    onSuccess: invalidate,
  });

  return { create, update, remove, invalidate };
}
