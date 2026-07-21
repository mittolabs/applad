import { useEffect, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Github, Loader2, TriangleAlert } from 'lucide-react';
import { api, friendlyError, setProject } from '@/api/client';

/*
 * Where GitHub sends somebody after they install the Applad app.
 *
 * The app is installed on a GitHub *account*, not on a project, so GitHub
 * knows nothing about which project asked and returns only an installation id
 * and the state we sent. The project is picked back up from where the flow
 * left it, and the server checks that state against the project it recorded —
 * so a link somebody was tricked into following cannot attach their
 * installation to another project.
 */

const PENDING_KEY = 'applad_git_install';

/** Remembers which project began an install, before leaving for GitHub. */
export function rememberInstall(projectId: string, returnTo: string) {
  localStorage.setItem(PENDING_KEY, JSON.stringify({ projectId, returnTo }));
}

export function GitHubSetupPage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const [error, setError] = useState('');
  // React runs effects twice in development; without this the second pass
  // posts an installation whose one-time state the first already consumed,
  // and the page reports a failure that did not happen.
  const done = useRef(false);

  useEffect(() => {
    if (done.current) return;
    done.current = true;

    const installationId = params.get('installation_id') ?? '';
    const state = params.get('state') ?? '';

    let pending: { projectId?: string; returnTo?: string } = {};
    try {
      pending = JSON.parse(localStorage.getItem(PENDING_KEY) ?? '{}');
    } catch {
      pending = {};
    }

    if (!installationId || !pending.projectId) {
      setError(
        'This install could not be matched to a project. Start again from the project you want to connect.',
      );
      return;
    }

    setProject(pending.projectId);
    api
      .post('/deploy/git/github/installations', { installationId, state })
      .then(() => {
        localStorage.removeItem(PENDING_KEY);
        navigate(pending.returnTo || `/project/${pending.projectId}/sites`, { replace: true });
      })
      .catch((e) => setError(friendlyError(e)));
  }, [params, navigate]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <div className="flex w-full max-w-[420px] flex-col items-center gap-4 rounded-[var(--radius-10)] border border-border bg-surface p-8 text-center">
        {error ? (
          <>
            <TriangleAlert size={22} style={{ color: '#F87171' }} />
            <div className="text-[length:var(--text-control)] font-medium text-text-primary">
              Could not finish connecting
            </div>
            <p className="text-[length:var(--text-body)] text-text-secondary">{error}</p>
            <button
              onClick={() => navigate('/projects', { replace: true })}
              className="mt-2 rounded-[var(--radius)] border border-field-border bg-fill px-3 py-2 text-[length:var(--text-label)] text-text-primary transition-colors hover:bg-fill-hover"
            >
              Back to projects
            </button>
          </>
        ) : (
          <>
            <Github size={22} className="text-text-secondary" />
            <div className="flex items-center gap-2 text-[length:var(--text-body)] text-text-secondary">
              <Loader2 size={14} className="animate-spin" />
              Connecting your GitHub account…
            </div>
          </>
        )}
      </div>
    </div>
  );
}
