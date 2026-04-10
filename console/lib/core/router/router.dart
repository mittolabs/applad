import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../shell/shell.dart';
import '../providers/auth_provider.dart';
import '../../features/login/login_page.dart';
import '../../features/onboarding/onboarding_page.dart';
import '../../features/overview/overview_page.dart';
import '../../features/auth/auth_page.dart';
import '../../features/databases/databases_page.dart';
import '../../features/storage/storage_page.dart';
import '../../features/deploy/deploy_page.dart';
import '../../features/functions/functions_page.dart';
import '../../features/messaging/messaging_page.dart';
import '../../features/workflows/workflows_page.dart';
import '../../features/settings/settings_page.dart';
import '../../features/flags/flags_page.dart';
import '../../features/experiments/experiments_page.dart';
import '../../features/health/health_page.dart';
import '../../features/sites/sites_page.dart';
import '../../features/containers/containers_page.dart';
import '../../features/mobile/mobile_page.dart';
import '../../features/desktop/desktop_page.dart';
import '../../features/account/account_page.dart';
import '../../features/projects/projects_page.dart';
import '../../features/environments/environments_page.dart';
import '../../features/get_started/get_started_page.dart';

Page<void> _noTransition(GoRouterState state, Widget child) {
  return NoTransitionPage(key: state.pageKey, child: child);
}

final routerProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    initialLocation: '/login',
    redirect: (context, state) {
      final token = ref.read(consoleTokenProvider);
      final isLoginRoute = state.uri.path == '/login';

      if (token == null && !isLoginRoute) return '/login';
      if (token != null && isLoginRoute) return '/projects';
      return null;
    },
    routes: [
      // Full-page routes (no sidebar)
      GoRoute(
        path: '/login',
        pageBuilder: (_, state) => _noTransition(state, const LoginPage()),
      ),
      GoRoute(
        path: '/onboarding',
        pageBuilder: (_, state) => _noTransition(state, const OnboardingPage()),
      ),
      GoRoute(
        path: '/projects',
        redirect: (context, state) {
          // Legacy route — redirect to org-scoped route
          return null; // handled by ProjectsPage which reads org from provider
        },
        pageBuilder: (_, state) => _noTransition(state, const ProjectsPage()),
      ),
      GoRoute(
        path: '/org/:orgId/projects',
        pageBuilder: (_, state) => _noTransition(state, const ProjectsPage()),
      ),
      GoRoute(
        path: '/account',
        pageBuilder: (_, state) => _noTransition(state, const AccountPage()),
      ),

      // Bare project path — redirect to overview
      GoRoute(
        path: '/project/:projectId',
        redirect: (context, state) =>
            '/project/${state.pathParameters['projectId']}/overview',
      ),

      // Project-scoped routes (with sidebar shell)
      ShellRoute(
        builder: (_, __, child) => AppShell(child: child),
        routes: [
          GoRoute(
            path: '/project/:projectId/overview',
            pageBuilder: (_, state) =>
                _noTransition(state, const OverviewPage()),
          ),
          GoRoute(
            path: '/project/:projectId/databases',
            pageBuilder: (_, state) =>
                _noTransition(state, const DatabasesPage()),
          ),
          GoRoute(
            path: '/project/:projectId/storage',
            pageBuilder: (_, state) =>
                _noTransition(state, const StoragePage()),
          ),
          GoRoute(
            path: '/project/:projectId/auth',
            pageBuilder: (_, state) =>
                _noTransition(state, const AuthPage()),
          ),
          GoRoute(
            path: '/project/:projectId/deploy',
            pageBuilder: (_, state) =>
                _noTransition(state, const DeployPage()),
          ),
          GoRoute(
            path: '/project/:projectId/functions',
            pageBuilder: (_, state) =>
                _noTransition(state, const FunctionsPage()),
          ),
          GoRoute(
            path: '/project/:projectId/messaging',
            pageBuilder: (_, state) =>
                _noTransition(state, const MessagingPage()),
          ),
          GoRoute(
            path: '/project/:projectId/workflows',
            pageBuilder: (_, state) =>
                _noTransition(state, const WorkflowsPage()),
          ),
          GoRoute(
            path: '/project/:projectId/flags',
            pageBuilder: (_, state) =>
                _noTransition(state, const FlagsPage()),
          ),
          GoRoute(
            path: '/project/:projectId/experiments',
            pageBuilder: (_, state) =>
                _noTransition(state, const ExperimentsPage()),
          ),
          GoRoute(
            path: '/project/:projectId/health',
            pageBuilder: (_, state) =>
                _noTransition(state, const HealthPage()),
          ),
          GoRoute(
            path: '/project/:projectId/sites',
            pageBuilder: (_, state) =>
                _noTransition(state, const SitesPage()),
          ),
          GoRoute(
            path: '/project/:projectId/containers',
            pageBuilder: (_, state) =>
                _noTransition(state, const ContainersPage()),
          ),
          GoRoute(
            path: '/project/:projectId/mobile',
            pageBuilder: (_, state) =>
                _noTransition(state, const MobilePage()),
          ),
          GoRoute(
            path: '/project/:projectId/desktop',
            pageBuilder: (_, state) =>
                _noTransition(state, const DesktopPage()),
          ),
          GoRoute(
            path: '/project/:projectId/settings',
            pageBuilder: (_, state) =>
                _noTransition(state, const SettingsPage()),
          ),
          GoRoute(
            path: '/project/:projectId/environments',
            pageBuilder: (_, state) =>
                _noTransition(state, const EnvironmentsPage()),
          ),
          GoRoute(
            path: '/project/:projectId/get-started',
            pageBuilder: (_, state) =>
                _noTransition(state, const GetStartedPage()),
          ),
        ],
      ),
    ],
  );
});
