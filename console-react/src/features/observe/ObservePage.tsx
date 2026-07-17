import { useLocation, useParams } from 'react-router-dom';
import { ObserveOverview } from './ObserveOverview';
import { ObserveErrors } from './ObserveErrors';
import { ObserveReleases } from './ObserveReleases';
import { ObserveLogs } from './ObserveLogs';
import { ObserveReplays } from './ObserveReplays';
import { ObserveUptime } from './ObserveUptime';
import { ObserveCrons } from './ObserveCrons';
import { ObserveAlerts } from './ObserveAlerts';

/* ObservePage — ports console/lib/features/observe/observe_page.dart.
 * The router points ALL of these project-scoped segments at this single
 * component: observe (Overview), errors, releases, logs, replays, uptime,
 * crons, alerts. The active sub-view is derived from the last path segment. */

const SECTIONS: Record<string, { title: string; subtitle: string }> = {
  observe: {
    title: 'Overview',
    subtitle: 'Errors, performance, releases, logs and uptime for your project',
  },
  errors: { title: 'Errors', subtitle: 'Track, triage and resolve errors in your project' },
  releases: { title: 'Releases', subtitle: 'Tag deployments and correlate them with errors' },
  logs: { title: 'Logs', subtitle: 'Search and tail structured logs from your project' },
  replays: { title: 'Replays', subtitle: 'Session replays for debugging user-reported issues' },
  uptime: { title: 'Uptime', subtitle: 'HTTP monitors with alerts for downtime' },
  crons: { title: 'Crons', subtitle: 'Monitor scheduled jobs and detect missed runs' },
  alerts: { title: 'Alerts', subtitle: 'Rules that notify you when metrics exceed thresholds' },
};

export function ObservePage() {
  const { projectId } = useParams();
  const { pathname } = useLocation();
  const seg = pathname.split('/').filter(Boolean).pop() ?? 'observe';
  const section = SECTIONS[seg] ?? SECTIONS.observe;

  return (
    <div className="flex min-h-0 flex-col">
      <div className="flex flex-col gap-1 px-6 pb-5 pt-8 md:px-8">
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
          {section.title}
        </h1>
        <p className="text-[length:var(--text-body)] text-text-secondary">{section.subtitle}</p>
      </div>

      <div className="min-h-0 flex-1">
        {seg === 'errors' ? (
          <ObserveErrors projectId={projectId} />
        ) : seg === 'releases' ? (
          <ObserveReleases projectId={projectId} />
        ) : seg === 'logs' ? (
          <ObserveLogs projectId={projectId} />
        ) : seg === 'replays' ? (
          <ObserveReplays projectId={projectId} />
        ) : seg === 'uptime' ? (
          <ObserveUptime projectId={projectId} />
        ) : seg === 'crons' ? (
          <ObserveCrons projectId={projectId} />
        ) : seg === 'alerts' ? (
          <ObserveAlerts projectId={projectId} />
        ) : (
          <ObserveOverview projectId={projectId} />
        )}
      </div>
    </div>
  );
}
