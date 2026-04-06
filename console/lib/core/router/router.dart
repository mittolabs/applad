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

final routerProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    initialLocation: '/login',
    redirect: (context, state) {
      final token = ref.read(consoleTokenProvider);
      final isLoginRoute = state.uri.path == '/login';

      // Not authenticated → force login (except if already on login page)
      if (token == null && !isLoginRoute) {
        return '/login';
      }

      // Authenticated and on login page → go to onboarding check
      if (token != null && isLoginRoute) {
        return '/onboarding';
      }

      return null;
    },
    routes: [
      GoRoute(
          path: '/login', builder: (_, __) => const LoginPage()),
      GoRoute(
          path: '/onboarding',
          builder: (_, __) => const OnboardingPage()),
      ShellRoute(
        builder: (_, __, child) => AppShell(child: child),
        routes: [
          GoRoute(
              path: '/overview',
              builder: (_, __) => const OverviewPage()),
          GoRoute(
              path: '/databases',
              builder: (_, __) => const DatabasesPage()),
          GoRoute(
              path: '/storage',
              builder: (_, __) => const StoragePage()),
          GoRoute(
              path: '/auth',
              builder: (_, __) => const AuthPage()),
          GoRoute(
              path: '/deploy',
              builder: (_, __) => const DeployPage()),
          GoRoute(
              path: '/functions',
              builder: (_, __) => const FunctionsPage()),
          GoRoute(
              path: '/messaging',
              builder: (_, __) => const MessagingPage()),
          GoRoute(
              path: '/workflows',
              builder: (_, __) => const WorkflowsPage()),
          GoRoute(
              path: '/settings',
              builder: (_, __) => const SettingsPage()),
        ],
      ),
    ],
  );
});
