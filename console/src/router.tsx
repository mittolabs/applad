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
import { ProjectsPage } from './features/projects/ProjectsPage';
import { AccountPage } from './features/account/AccountPage';
import { OnboardingPage } from './features/onboarding/OnboardingPage';
import { ExperimentsPage } from './features/experiments/ExperimentsPage';
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
const SitesPage = named(() => import('./features/sites/SitesPage'), 'SitesPage');
const ContainersPage = named(() => import('./features/containers/ContainersPage'), 'ContainersPage');
const MobilePage = named(() => import('./features/mobile/MobilePage'), 'MobilePage');
const DesktopPage = named(() => import('./features/desktop/DesktopPage'), 'DesktopPage');
const PlatformsPage = named(() => import('./features/platforms/PlatformsPage'), 'PlatformsPage');
const VaultPage = named(() => import('./features/vault/VaultPage'), 'VaultPage');
const EnvironmentsPage = named(() => import('./features/environments/EnvironmentsPage'), 'EnvironmentsPage');
const RealtimePage = named(() => import('./features/realtime/RealtimePage'), 'RealtimePage');
const HealthPage = named(() => import('./features/health/HealthPage'), 'HealthPage');
const ObservePage = named(() => import('./features/observe/ObservePage'), 'ObservePage');

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
  ['functions', 'Functions'],
  ['storage', 'Storage'],
  ['messaging', 'Messaging'],
  ['realtime', 'Realtime'],
  ['workflows', 'Workflows'],
  ['flags', 'Feature Flags'],
  ['platforms', 'Platforms'],
  ['sites', 'Sites'],
  ['containers', 'Containers'],
  ['mobile', 'Mobile'],
  ['desktop', 'Desktop'],
  ['observe', 'Observe'],
  ['errors', 'Errors'],
  ['releases', 'Releases'],
  ['logs', 'Logs'],
  ['replays', 'Replays'],
  ['uptime', 'Uptime'],
  ['crons', 'Crons'],
  ['alerts', 'Alerts'],
  ['health', 'Health'],
  ['settings', 'Settings'],
  ['settings/keys/:keyId', 'API Key'],
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
  storage: <StoragePage />,
  functions: <FunctionsPage />,
  messaging: <MessagingPage />,
  flags: <FlagsPage />,
  settings: <SettingsPage />,
  'settings/keys/:keyId': <ApiKeyDetailPage />,
  workflows: <WorkflowsPage />,
  sites: <SitesPage />,
  containers: <ContainersPage />,
  mobile: <MobilePage />,
  desktop: <DesktopPage />,
  platforms: <PlatformsPage />,
  vault: <VaultPage />,
  environments: <EnvironmentsPage />,
  realtime: <RealtimePage />,
  health: <HealthPage />,
  // Observe: all 8 segments render the one ObservePage (self-selects by URL).
  observe: <ObservePage />,
  errors: <ObservePage />,
  releases: <ObservePage />,
  logs: <ObservePage />,
  replays: <ObservePage />,
  uptime: <ObservePage />,
  crons: <ObservePage />,
  alerts: <ObservePage />,
};

export const router = createBrowserRouter([
  {
    element: <RootBoot />,
    children: [
      {
        path: '/login',
        element: (
          <RedirectIfAuthed>
            <LoginPage />
          </RedirectIfAuthed>
        ),
      },
      { path: '/', element: <RootRedirect /> },
      {
        element: <RequireAuth />,
        children: [
          { path: '/onboarding', element: <OnboardingPage /> },
          { path: '/projects', element: <ProjectsPage /> },
          { path: '/org/:orgId/projects', element: <ProjectsPage /> },
          { path: '/account', element: <AccountPage /> },
          { path: '/experiments', element: <ExperimentsPage /> },
          {
            path: '/project/:projectId',
            element: <Shell />,
            children: [
              { index: true, element: <Navigate to="overview" replace /> },
              ...shellSegments.map(([path, title]) => ({
                path,
                element: FEATURE_ELEMENTS[path] ?? <PlaceholderPage name={title} />,
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
