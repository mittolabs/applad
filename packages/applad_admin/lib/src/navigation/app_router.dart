import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../screens/dashboard/dashboard_screen.dart';
import '../screens/tables/tables_screen.dart';
import '../screens/auth/auth_screen.dart';
import '../screens/storage/storage_screen.dart';
import '../screens/functions/functions_screen.dart';
import '../screens/workflows/workflows_screen.dart';
import '../screens/messaging/messaging_screen.dart';
import '../screens/flags/flags_screen.dart';
import '../screens/analytics/analytics_screen.dart';
import '../screens/hosting/hosting_screen.dart';
import '../screens/deployments/deployments_screen.dart';
import '../screens/observability/observability_screen.dart';
import '../screens/instruct/instruct_screen.dart';
import '../screens/settings/settings_screen.dart';
import '../widgets/nav_rail.dart';

/// Application router configuration using go_router.
final class AppRouter {
  AppRouter._();

  static final router = GoRouter(
    initialLocation: '/dashboard',
    routes: [
      ShellRoute(
        builder: (context, state, child) => AppShell(child: child),
        routes: [
          GoRoute(
              path: '/dashboard', builder: (c, s) => const DashboardScreen()),
          GoRoute(path: '/tables', builder: (c, s) => const TablesScreen()),
          GoRoute(path: '/auth', builder: (c, s) => const AuthScreen()),
          GoRoute(path: '/storage', builder: (c, s) => const StorageScreen()),
          GoRoute(
              path: '/functions', builder: (c, s) => const FunctionsScreen()),
          GoRoute(
              path: '/workflows', builder: (c, s) => const WorkflowsScreen()),
          GoRoute(
              path: '/messaging', builder: (c, s) => const MessagingScreen()),
          GoRoute(path: '/flags', builder: (c, s) => const FlagsScreen()),
          GoRoute(
              path: '/analytics', builder: (c, s) => const AnalyticsScreen()),
          GoRoute(path: '/hosting', builder: (c, s) => const HostingScreen()),
          GoRoute(
              path: '/deployments',
              builder: (c, s) => const DeploymentsScreen()),
          GoRoute(
              path: '/observability',
              builder: (c, s) => const ObservabilityScreen()),
          GoRoute(path: '/instruct', builder: (c, s) => const InstructScreen()),
          GoRoute(path: '/settings', builder: (c, s) => const SettingsScreen()),
        ],
      ),
    ],
  );
}

/// The main app shell with navigation rail.
class AppShell extends StatelessWidget {
  const AppShell({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Row(
        children: [
          const AppNavRail(),
          const VerticalDivider(width: 1),
          Expanded(child: child),
        ],
      ),
    );
  }
}
