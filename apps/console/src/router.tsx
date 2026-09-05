import { lazy, useEffect } from 'react';
import {
  createBrowserRouter,
  Navigate,
  Outlet,
  useLocation,
} from 'react-router-dom';
import { Loader2 } from 'lucide-react';
import { useAuthStore } from './stores/auth';
import { AiChatOverlay } from './features/ai/AiChatOverlay';
import { Shell } from './shell/Shell';
import { PlaceholderPage } from './components/placeholder-page';
import { LoginPage } from './features/login/LoginPage';
import { GitHubSetupPage } from '@/features/deploy-shared/GitHubSetup';
import { InvitePage } from './features/login/InvitePage';
import { ProjectsPage } from './features/projects/ProjectsPage';
import { AccountPage } from './features/account/AccountPage';
import { OnboardingPage } from './features/onboarding/OnboardingPage';
import { ExperimentsPage } from './features/experiments/ExperimentsPage';
import { extensionRoutes, extensionStandaloneRoutes } from '@/extensions';
import { AppError } from '@/components/app-error';
// Shell feature pages are lazy-loaded so heavy deps (Monaco in databases/
// functions, React Flow in workflows) only download when the route is visited.
const named = <M extends Record<string, unknown>, K extends keyof M>(
  loader: () => Promise<M>,
  key: K,
) => lazy(() => loader().then((m) => ({ default: m[key] as React.ComponentType })));

const OverviewPage = named(() => import('./features/overview/OverviewPage'), 'OverviewPage');
const GetStartedPage = named(() => import('./features/get-started/GetStartedPage'), 'GetStartedPage');
const AuthPage = named(() => import('./features/auth/AuthPage'), 'AuthPage');
const DatabasesPage = named(() => import('./features/databases/DatabasesPage'), 'DatabasesPage');
const StoragePage = named(() => import('./features/storage/StoragePage'), 'StoragePage');
const FunctionsPage = named(() => import('./features/functions/FunctionsPage'), 'FunctionsPage');
const MessagingPage = named(() => import('./features/messaging/MessagingPage'), 'MessagingPage');
const FlagsPage = named(() => import('./features/flags/FlagsPage'), 'FlagsPage');
const SettingsPage = named(() => import('./features/settings/SettingsPage'), 'SettingsPage');
const ApiKeyDetailPage = named(() => import('./features/settings/ApiKeyDetailPage'), 'ApiKeyDetailPage');
const WorkflowsPage = named(() => import('./features/workflows/WorkflowsPage'), 'WorkflowsPage');
const EndpointsPage = named(() => import('./features/endpoints/EndpointsPage'), 'EndpointsPage');
const MigrationsPage = named(() => import('./features/migrations/MigrationsPage'), 'MigrationsPage');
const SitesPage = named(() => import('./features/sites/SitesPage'), 'SitesPage');
const ContainersPage = named(() => import('./features/containers/ContainersPage'), 'ContainersPage');
const MobilePage = named(() => import('./features/mobile/MobilePage'), 'MobilePage');
const DesktopPage = named(() => import('./features/desktop/DesktopPage'), 'DesktopPage');
const PlatformsPage = named(() => import('./features/platforms/PlatformsPage'), 'PlatformsPage');
const VaultPage = named(() => import('./features/vault/VaultPage'), 'VaultPage');
const EnvironmentsPage = named(() => import('./features/environments/EnvironmentsPage'), 'EnvironmentsPage');
const RealtimePage = named(() => import('./features/realtime/RealtimePage'), 'RealtimePage');
const HealthPage = named(() => import('./features/health/HealthPage'), 'HealthPage');
const AnalyticsPage = named(() => import('./features/analytics/AnalyticsPage'), 'AnalyticsPage');

/* Router — ports console/lib/core/router/router.dart.
 * Guard: no token → /login; token on /login → /projects.
 * Standalone routes (no shell) + project-scoped routes under <Shell>. */

function RootBoot() {
  const init = useAuthStore((s) => s.init);
  const status = useAuthStore((s) => s.status);

  useEffect(() => {
    void init();
  }, [init]);

  if (status === 'loading') {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <Loader2 className="h-6 w-6 animate-spin text-text-muted" />
      </div>
    );
  }
  return (
    <>
      <Outlet />
      {/* Global Applad AI overlay — self-gates on token + config + route. */}
      <AiChatOverlay />
    </>
  );
}

function RequireAuth() {
  const status = useAuthStore((s) => s.status);
  if (status === 'unauthenticated') return <Navigate to="/login" replace />;
  return <Outlet />;
}

function RedirectIfAuthed({ children }: { children: React.ReactNode }) {
  const status = useAuthStore((s) => s.status);
  if (status === 'authenticated') return <Navigate to="/projects" replace />;
  return <>{children}</>;
}

function RootRedirect() {
  const status = useAuthStore((s) => s.status);
  return <Navigate to={status === 'authenticated' ? '/projects' : '/login'} replace />;
}

/** All project-scoped child routes under <Shell>, as [path, title] tuples. */
const shellSegments: [string, string][] = [
  ['overview', 'Overview'],
  ['get-started', 'Get started'],
  ['auth', 'Auth'],
  ['databases', 'Databases'],
  ['databases/:databaseId', 'Databases'],
  ['databases/:databaseId/:tableId', 'Databases'],
  ['functions', 'Functions'],
  ['functions/:functionId', 'Functions'],
  ['endpoints', 'Endpoints'],
  ['endpoints/:endpointId', 'Endpoints'],
  ['storage', 'Storage'],
  ['storage/:bucketId', 'Storage'],
  ['storage/:bucketId/:fileId', 'Storage'],
  ['messaging', 'Messaging'],
  ['realtime', 'Realtime'],
  ['workflows', 'Workflows'],
  ['workflows/:workflowId', 'Workflows'],
  ['flags', 'Feature Flags'],
  ['flags/:flagId', 'Feature Flags'],
  ['platforms', 'Platforms'],
  ['platforms/:platformId', 'Platforms'],
  ['sites', 'Sites'],
  ['sites/:siteId', 'Sites'],
  ['containers', 'Containers'],
  ['containers/:containerId', 'Containers'],
  ['mobile', 'Mobile'],
  ['mobile/:appId', 'Mobile'],
  ['desktop', 'Desktop'],
  ['desktop/:appId', 'Desktop'],
  ['analytics', 'Analytics'],
  ['events', 'Events'],
  ['funnels', 'Funnels'],
  ['uptime', 'Uptime'],
  ['crons', 'Crons'],
  ['health', 'Health'],
  ['settings', 'Settings'],
  ['settings/keys/:keyId', 'API Key'],
  ['migrations', 'Migrations'],
  ['vault', 'Vault'],
  ['environments', 'Environments'],
];

/** Ported feature pages, keyed by route segment. Missing segments fall back to
 * PlaceholderPage until their feature lands. */
const FEATURE_ELEMENTS: Record<string, React.ReactNode> = {
  overview: <OverviewPage />,
  'get-started': <GetStartedPage />,
  auth: <AuthPage />,
  databases: <DatabasesPage />,
  'databases/:databaseId': <DatabasesPage />,
  'databases/:databaseId/:tableId': <DatabasesPage />,
  storage: <StoragePage />,
  'storage/:bucketId': <StoragePage />,
  'storage/:bucketId/:fileId': <StoragePage />,
  functions: <FunctionsPage />,
  'functions/:functionId': <FunctionsPage />,
  endpoints: <EndpointsPage />,
  'endpoints/:endpointId': <EndpointsPage />,
  messaging: <MessagingPage />,
  flags: <FlagsPage />,
  'flags/:flagId': <FlagsPage />,
  settings: <SettingsPage />,
  'settings/keys/:keyId': <ApiKeyDetailPage />,
  migrations: <MigrationsPage />,
  workflows: <WorkflowsPage />,
  'workflows/:workflowId': <WorkflowsPage />,
  sites: <SitesPage />,
  'sites/:siteId': <SitesPage />,
  containers: <ContainersPage />,
  'containers/:containerId': <ContainersPage />,
  mobile: <MobilePage />,
  'mobile/:appId': <MobilePage />,
  desktop: <DesktopPage />,
  'desktop/:appId': <DesktopPage />,
  platforms: <PlatformsPage />,
  'platforms/:platformId': <PlatformsPage />,
  vault: <VaultPage />,
  environments: <EnvironmentsPage />,
  realtime: <RealtimePage />,
  health: <HealthPage />,
  // Analytics: all 5 segments render the one AnalyticsPage (self-selects by URL).
  analytics: <AnalyticsPage />,
  events: <AnalyticsPage />,
  funnels: <AnalyticsPage />,
  uptime: <AnalyticsPage />,
  crons: <AnalyticsPage />,
};

export const router = createBrowserRouter([
  {
    element: <RootBoot />,
    // Catches render errors anywhere below, so one bad page is a screen the user
    // can act on rather than a blank document with a minified stack.
    errorElement: <AppError />,
    children: [
      {
        path: '/login',
        element: (
          <RedirectIfAuthed>
            <LoginPage />
          </RedirectIfAuthed>
        ),
      },
      // Invite redemption stands apart from login: the token is the
      // credential, and it works on instances where signup is closed.
      { path: '/invite/:token', element: <InvitePage /> },
      // Fixed by the GitHub App's own configuration: GitHub returns everyone
      // here after an install, whichever project sent them.
      { path: '/git/setup', element: <GitHubSetupPage /> },
      { path: '/', element: <RootRedirect /> },
      {
        element: <RequireAuth />,
        children: [
          { path: '/onboarding', element: <OnboardingPage /> },
          { path: '/projects', element: <ProjectsPage /> },
          { path: '/org/:orgId/projects', element: <ProjectsPage /> },
          { path: '/account', element: <AccountPage /> },
          { path: '/experiments', element: <ExperimentsPage /> },
          // Authed pages a compiled-in module owns that are not project-scoped
          // (billing belongs to an organization, not a project). A default build
          // contributes none.
          ...extensionStandaloneRoutes().map(({ path, element: El }) => ({
            path,
            element: <El />,
          })),
          {
            path: '/project/:projectId',
            element: <Shell />,
            children: [
              { index: true, element: <Navigate to="overview" replace /> },
              ...shellSegments.map(([path, title]) => ({
                path,
                element: FEATURE_ELEMENTS[path] ?? <PlaceholderPage name={title} />,
              })),
              // Project-scoped module pages, rendered inside the shell.
              ...extensionRoutes().map(({ path, element: El }) => ({
                path,
                element: <El />,
              })),
            ],
          },
        ],
      },
      { path: '*', element: <NotFound /> },
    ],
  },
]);

function NotFound() {
  const location = useLocation();
  return (
    <div className="flex h-screen flex-col items-center justify-center gap-3 bg-background text-center">
      <div className="text-[length:var(--text-h2)] font-semibold text-text-primary">404</div>
      <div className="text-[length:var(--text-body)] text-text-muted">
        No route for {location.pathname}
      </div>
      <a
        href="/projects"
        className="mt-2 rounded-[var(--radius)] bg-[var(--color-accent)] px-4 py-2 text-[length:var(--text-body)] font-medium text-white"
      >
        Go to projects
      </a>
    </div>
  );
}
