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
import '../../features/account/account_page.dart';
import '../../features/projects/projects_page.dart';

Page<void> _noTransition(GoRouterState state, Widget child) {
  return NoTransitionPage(key: state.pageKey, child: child);
}

Page<void> _fade(GoRouterState state, Widget child) {
  return CustomTransitionPage(
    key: state.pageKey,
    child: child,
    transitionDuration: const Duration(milliseconds: 150),
    transitionsBuilder: (_, animation, __, child) =>
        FadeTransition(opacity: animation, child: child),
  );
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
        pageBuilder: (_, state) => _fade(state, const LoginPage()),
      ),
      GoRoute(
        path: '/onboarding',
        pageBuilder: (_, state) => _fade(state, const OnboardingPage()),
      ),
      GoRoute(
        path: '/projects',
        pageBuilder: (_, state) => _fade(state, const ProjectsPage()),
      ),
      GoRoute(
        path: '/account',
        pageBuilder: (_, state) => _fade(state, const AccountPage()),
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
            path: '/project/:projectId/settings',
            pageBuilder: (_, state) =>
                _noTransition(state, const SettingsPage()),
          ),
        ],
      ),
    ],
  );
});
