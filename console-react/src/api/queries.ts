import { useQuery } from '@tanstack/react-query';
import { api } from './client';
import { useOrgStore } from '@/stores/org';
import { useAuthStore } from '@/stores/auth';

/* Shared query hooks for org/project/environment context — used by the shell
 * top nav and the projects page. Mirror the Riverpod FutureProviders. */

export interface Org {
  $id: string;
  name: string;
  [k: string]: unknown;
}
export interface Project {
  $id: string;
  name: string;
  [k: string]: unknown;
}
export interface Environment {
  $id: string;
  name: string;
  isDefault?: boolean;
  [k: string]: unknown;
}

export function useOrgs(enabled = true) {
  // Wait for the console user (which sets the X-Console-User-* headers) so the
  // backend returns this user's orgs, not an empty list (mirrors the Dart
  // orgsProvider that awaited consoleAuthProvider).
  const userId = useAuthStore((s) => s.user?.id);
  return useQuery({
    queryKey: ['organizations', userId],
    enabled: enabled && !!userId,
    queryFn: async () => {
      const res = await api.get('/organizations');
      return ((res.data as { organizations?: Org[] }).organizations ?? []) as Org[];
    },
  });
}

export function useProjects() {
  const orgId = useOrgStore((s) => s.currentOrgId);
  return useQuery({
    queryKey: ['projects', orgId],
    queryFn: async () => {
      const res = await api.get('/projects', {
        params: orgId ? { orgId } : undefined,
      });
      return ((res.data as { projects?: Project[] }).projects ?? []) as Project[];
    },
  });
}

export function useProject(projectId: string | undefined) {
  return useQuery({
    queryKey: ['project', projectId],
    enabled: !!projectId,
    queryFn: async () => {
      const res = await api.get(`/projects/${projectId}`);
      return res.data as Project;
    },
  });
}

export function useEnvironments(enabled = true) {
  return useQuery({
    queryKey: ['environments'],
    enabled,
    queryFn: async () => {
      const res = await api.get('/deploy/environments');
      return ((res.data as { environments?: Environment[] }).environments ?? []) as Environment[];
    },
  });
}
