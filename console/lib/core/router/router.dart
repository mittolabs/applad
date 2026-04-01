import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../features/auth/auth_page.dart';
import '../../features/databases/databases_page.dart';
import '../../features/storage/storage_page.dart';
import '../../features/deploy/deploy_page.dart';
import '../../features/messaging/messaging_page.dart';
import '../../features/workflows/workflows_page.dart';
import '../../features/settings/settings_page.dart';

final routerProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    initialLocation: '/databases',
    routes: [
      GoRoute(path: '/login', builder: (_, __) => const AuthPage()),
      GoRoute(path: '/databases', builder: (_, __) => const DatabasesPage()),
      GoRoute(path: '/storage', builder: (_, __) => const StoragePage()),
      GoRoute(path: '/deploy', builder: (_, __) => const DeployPage()),
      GoRoute(path: '/messaging', builder: (_, __) => const MessagingPage()),
      GoRoute(path: '/workflows', builder: (_, __) => const WorkflowsPage()),
      GoRoute(path: '/settings', builder: (_, __) => const SettingsPage()),
    ],
  );
});
