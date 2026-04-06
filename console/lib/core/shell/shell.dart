import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/auth_provider.dart';
import '../providers/project_provider.dart';

class AppShell extends ConsumerWidget {
  final Widget child;
  const AppShell({super.key, required this.child});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final currentPath = GoRouterState.of(context).uri.path;
    final projectId = ref.watch(currentProjectProvider);

    return Scaffold(
      body: Row(
        children: [
          NavigationRail(
            selectedIndex: _indexForPath(currentPath),
            onDestinationSelected: (i) =>
                context.go(_pathForIndex(i)),
            labelType: NavigationRailLabelType.all,
            leading: Padding(
              padding: const EdgeInsets.symmetric(vertical: 16),
              child: Column(
                children: [
                  const Icon(Icons.cloud, size: 32),
                  const SizedBox(height: 4),
                  Text(
                    'Applad',
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                  if (projectId != null) ...[
                    const SizedBox(height: 2),
                    Text(
                      projectId.length > 8
                          ? projectId.substring(0, 8)
                          : projectId,
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                ],
              ),
            ),
            destinations: const [
              NavigationRailDestination(
                icon: Icon(Icons.dashboard_outlined),
                selectedIcon: Icon(Icons.dashboard),
                label: Text('Overview'),
              ),
              NavigationRailDestination(
                icon: Icon(Icons.table_chart_outlined),
                selectedIcon: Icon(Icons.table_chart),
                label: Text('Databases'),
              ),
              NavigationRailDestination(
                icon: Icon(Icons.folder_outlined),
                selectedIcon: Icon(Icons.folder),
                label: Text('Storage'),
              ),
              NavigationRailDestination(
                icon: Icon(Icons.people_outlined),
                selectedIcon: Icon(Icons.people),
                label: Text('Auth'),
              ),
              NavigationRailDestination(
                icon: Icon(Icons.rocket_launch_outlined),
                selectedIcon: Icon(Icons.rocket_launch),
                label: Text('Deploy'),
              ),
              NavigationRailDestination(
                icon: Icon(Icons.mail_outlined),
                selectedIcon: Icon(Icons.mail),
                label: Text('Messaging'),
              ),
              NavigationRailDestination(
                icon: Icon(Icons.account_tree_outlined),
                selectedIcon: Icon(Icons.account_tree),
                label: Text('Workflows'),
              ),
              NavigationRailDestination(
                icon: Icon(Icons.settings_outlined),
                selectedIcon: Icon(Icons.settings),
                label: Text('Settings'),
              ),
            ],
          ),
          const VerticalDivider(thickness: 1, width: 1),
          Expanded(
            child: Column(
              children: [
                // Top bar with logout
                Container(
                  height: 48,
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.end,
                    children: [
                      Consumer(builder: (context, ref, _) {
                        final auth = ref.watch(consoleAuthProvider);
                        final user = auth.valueOrNull;
                        if (user == null) return const SizedBox.shrink();
                        return Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(user.email,
                                style: Theme.of(context).textTheme.bodySmall),
                            const SizedBox(width: 8),
                            IconButton(
                              icon: const Icon(Icons.logout, size: 20),
                              tooltip: 'Sign out',
                              onPressed: () {
                                ref.read(consoleAuthProvider.notifier).logout();
                                context.go('/login');
                              },
                            ),
                          ],
                        );
                      }),
                    ],
                  ),
                ),
                const Divider(height: 1),
                Expanded(child: child),
              ],
            ),
          ),
        ],
      ),
    );
  }

  int _indexForPath(String path) {
    const paths = [
      '/overview',
      '/databases',
      '/storage',
      '/auth',
      '/deploy',
      '/messaging',
      '/workflows',
      '/settings',
    ];
    final idx = paths.indexOf(path);
    return idx >= 0 ? idx : 0;
  }

  String _pathForIndex(int index) {
    const paths = [
      '/overview',
      '/databases',
      '/storage',
      '/auth',
      '/deploy',
      '/messaging',
      '/workflows',
      '/settings',
    ];
    return paths[index];
  }
}
