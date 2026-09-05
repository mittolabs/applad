import { useLocation, useParams } from 'react-router-dom';
import { AnalyticsOverview } from './AnalyticsOverview';
import { AnalyticsEvents } from './AnalyticsEvents';
import { AnalyticsFunnels } from './AnalyticsFunnels';
import { AnalyticsUptime } from './AnalyticsUptime';
import { AnalyticsCrons } from './AnalyticsCrons';

/* AnalyticsPage — the router points all of these project-scoped segments at
 * this one component: analytics (Overview), events, funnels, uptime, crons.
 * The active sub-view is derived from the last path segment.
 *
 * This replaced Observe. Errors, logs, releases, replays and alerts left with
 * it: diagnostics is Bugslad's product. What stayed is what Applad measures
 * about itself — request latency, uptime and cron check-ins — alongside the
 * product analytics the platform was already collecting but never showed. */

const SECTIONS: Record<string, { title: string; subtitle: string }> = {
  analytics: {
    title: 'Overview',
    subtitle: 'Events, active users, request latency and uptime for your project',
  },
  events: { title: 'Events', subtitle: 'The raw event stream your app is sending' },
  funnels: { title: 'Funnels', subtitle: 'Step-by-step conversion, and where people drop out' },
  uptime: { title: 'Uptime', subtitle: 'HTTP monitors for the services this project depends on' },
  crons: { title: 'Crons', subtitle: 'Scheduled jobs, and the runs that never checked in' },
};

export function AnalyticsPage() {
  const { projectId } = useParams();
  const { pathname } = useLocation();
  const seg = pathname.split('/').filter(Boolean).pop() ?? 'analytics';
  const section = SECTIONS[seg] ?? SECTIONS.analytics;

  return (
    <div className="flex min-h-0 flex-col">
      <div className="flex flex-col gap-1 px-6 pb-5 pt-8 md:px-8">
        <h1 className="text-[length:var(--text-h1)] font-semibold text-text-primary">
          {section.title}
        </h1>
        <p className="text-[length:var(--text-body)] text-text-secondary">{section.subtitle}</p>
      </div>

      <div className="min-h-0 flex-1">
        {seg === 'events' ? (
          <AnalyticsEvents projectId={projectId} />
        ) : seg === 'funnels' ? (
          <AnalyticsFunnels projectId={projectId} />
        ) : seg === 'uptime' ? (
          <AnalyticsUptime projectId={projectId} />
        ) : seg === 'crons' ? (
          <AnalyticsCrons projectId={projectId} />
        ) : (
          <AnalyticsOverview projectId={projectId} />
        )}
      </div>
    </div>
  );
}
