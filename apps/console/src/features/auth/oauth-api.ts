import { api } from '@/api/client';

/* Client for the per-project OAuth provider configuration API
 * (projects handler: /projects/{projectId}/auth/oauth). The GET never returns
 * client secrets — only enabled + clientId — so the console cannot display a
 * stored secret and an empty secret on save preserves the one already stored. */

export interface OAuthProviderConfig {
  $id?: string;
  provider: string;
  enabled: boolean;
  clientId: string;
  /** Auxiliary non-secret fields (e.g. Microsoft tenantId, Apple keyId/teamId).
   * Returned so the console can prefill them; secret material is never here. */
  extra?: Record<string, string>;
}

export async function listOAuthProviders(projectId: string): Promise<OAuthProviderConfig[]> {
  const res = await api.get(`/projects/${projectId}/auth/oauth`);
  return (res.data?.providers ?? []) as OAuthProviderConfig[];
}

export interface SetOAuthProviderBody {
  clientId: string;
  /** Empty preserves the stored secret; the GET never returns it. */
  clientSecret?: string;
  /** Auxiliary non-secret fields; omitted/undefined preserves the stored ones. */
  extra?: Record<string, string>;
  enabled: boolean;
}

export async function setOAuthProvider(
  projectId: string,
  provider: string,
  body: SetOAuthProviderBody,
): Promise<void> {
  await api.put(`/projects/${projectId}/auth/oauth/${provider}`, body);
}

export async function deleteOAuthProvider(projectId: string, provider: string): Promise<void> {
  await api.delete(`/projects/${projectId}/auth/oauth/${provider}`);
}
