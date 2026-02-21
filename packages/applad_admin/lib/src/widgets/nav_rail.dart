import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// The main side navigation rail for the admin app.
class AppNavRail extends StatelessWidget {
  const AppNavRail({super.key});

  static const _destinations = [
    _NavItem(
        icon: Icons.dashboard_outlined, label: 'Dashboard', path: '/dashboard'),
    _NavItem(
        icon: Icons.table_chart_outlined, label: 'Tables', path: '/tables'),
    _NavItem(icon: Icons.lock_outlined, label: 'Auth', path: '/auth'),
    _NavItem(icon: Icons.folder_outlined, label: 'Storage', path: '/storage'),
    _NavItem(
        icon: Icons.functions_outlined, label: 'Functions', path: '/functions'),
    _NavItem(
        icon: Icons.account_tree_outlined,
        label: 'Workflows',
        path: '/workflows'),
    _NavItem(
        icon: Icons.message_outlined, label: 'Messaging', path: '/messaging'),
    _NavItem(icon: Icons.flag_outlined, label: 'Flags', path: '/flags'),
    _NavItem(
        icon: Icons.analytics_outlined, label: 'Analytics', path: '/analytics'),
    _NavItem(icon: Icons.web_outlined, label: 'Hosting', path: '/hosting'),
    _NavItem(
        icon: Icons.rocket_launch_outlined,
        label: 'Deployments',
        path: '/deployments'),
    _NavItem(
        icon: Icons.monitor_heart_outlined,
        label: 'Observability',
        path: '/observability'),
    _NavItem(
        icon: Icons.auto_awesome_outlined,
        label: 'Instruct',
        path: '/instruct'),
    _NavItem(
        icon: Icons.settings_outlined, label: 'Settings', path: '/settings'),
  ];

  @override
  Widget build(BuildContext context) {
    final currentPath = GoRouterState.of(context).uri.path;
    final selectedIndex =
        _destinations.indexWhere((d) => currentPath.startsWith(d.path));

    return NavigationRail(
      extended: MediaQuery.sizeOf(context).width > 1200,
      destinations: _destinations
          .map((d) => NavigationRailDestination(
                icon: Icon(d.icon),
                label: Text(d.label),
              ))
          .toList(),
      selectedIndex: selectedIndex < 0 ? 0 : selectedIndex,
      onDestinationSelected: (index) {
        context.go(_destinations[index].path);
      },
      leading: Padding(
        padding: const EdgeInsets.symmetric(vertical: 16),
        child: Column(
          children: [
            Icon(
              Icons.bolt,
              color: Theme.of(context).colorScheme.primary,
              size: 32,
            ),
            const SizedBox(height: 4),
            Text(
              'Applad',
              style: Theme.of(context).textTheme.labelSmall?.copyWith(
                    color: Theme.of(context).colorScheme.primary,
                    fontWeight: FontWeight.bold,
                  ),
            ),
          ],
        ),
      ),
    );
  }
}

class _NavItem {
  const _NavItem({required this.icon, required this.label, required this.path});
  final IconData icon;
  final String label;
  final String path;
}
