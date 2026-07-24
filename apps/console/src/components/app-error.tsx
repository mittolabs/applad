import { useEffect } from 'react';
import { useNavigate, useRouteError } from 'react-router-dom';
import { AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { reportClientError } from '@/extensions';

/*
 * What the user sees when a page fails to render.
 *
 * Without this, React unmounts the whole tree and the browser is left showing a
 * blank document with a minified stack — which tells the user nothing, loses
 * their navigation, and looks like the product broke rather than one page. This
 * keeps the failure to a screen they can act on, and hands the details to
 * whoever is registered to record them (nobody, on a default build).
 *
 * The raw message stays available but folded away: useful when someone reports
 * it, noise otherwise.
 */
export function AppError() {
  const error = useRouteError();
  const navigate = useNavigate();

  const err = error instanceof Error ? error : new Error(String(error));

  useEffect(() => {
    reportClientError({
      message: err.message,
      stack: err.stack,
      path: typeof window !== 'undefined' ? window.location.pathname : '',
      userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
    });
    // Also to the browser console, for whoever has devtools open.
    console.error('console: render error', err);
  }, [err]);

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-background px-6 text-center">
      <div className="flex h-11 w-11 items-center justify-center rounded-full bg-[var(--color-danger)]/10">
        <AlertTriangle size={20} className="text-[var(--color-danger)]" />
      </div>
      <div>
        <h1 className="text-[length:var(--text-title)] font-semibold text-text-primary">
          Something went wrong on this page
        </h1>
        <p className="mt-1 max-w-md text-[length:var(--text-body)] text-text-muted">
          The rest of the console is fine. Try again, or head back to your projects. We have been
          told about this.
        </p>
      </div>
      <div className="flex gap-2">
        <Button onClick={() => window.location.reload()}>Try again</Button>
        <Button variant="secondary" onClick={() => navigate('/projects')}>
          Go to projects
        </Button>
      </div>
      <details className="mt-2 max-w-xl text-left">
        <summary className="cursor-pointer text-[length:var(--text-caption)] text-text-subtle">
          Technical details
        </summary>
        <pre className="mt-2 max-h-56 overflow-auto rounded-[var(--radius-6)] border border-border bg-surface p-3 text-[length:var(--text-2xs)] text-text-muted">
          {err.message}
          {err.stack ? `\n\n${err.stack}` : ''}
        </pre>
      </details>
    </div>
  );
}
