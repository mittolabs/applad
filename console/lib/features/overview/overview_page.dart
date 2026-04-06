import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';

final usageProvider =
    FutureProvider.family<Map<String, dynamic>?, String>((ref, projectId) async {
  final api = ref.read(apiClientProvider);
  try {
    final res = await api.get('/projects/$projectId/usage');
    return res.data as Map<String, dynamic>;
  } catch (_) {
    return null;
  }
});

class OverviewPage extends ConsumerWidget {
  const OverviewPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final projectId = ref.watch(currentProjectProvider);

    if (projectId == null) {
      return Scaffold(
        appBar: AppBar(title: const Text('Overview')),
        body: const Center(
          child: Text('Select a project in Settings to view usage analytics.'),
        ),
      );
    }

    final usageAsync = ref.watch(usageProvider(projectId));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Overview'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () => ref.invalidate(usageProvider(projectId)),
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: usageAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text('Error: $e')),
        data: (usage) {
          if (usage == null) {
            return const Center(child: Text('Failed to load usage data'));
          }
          return SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Project: $projectId',
                    style: Theme.of(context).textTheme.headlineSmall),
                const SizedBox(height: 24),

                // Primary stats grid
                _SectionTitle('Resources'),
                const SizedBox(height: 12),
                Wrap(
                  spacing: 16,
                  runSpacing: 16,
                  children: [
                    _StatCard(
                      icon: Icons.people,
                      label: 'Users',
                      value: '${usage['users'] ?? 0}',
                      color: Colors.blue,
                    ),
                    _StatCard(
                      icon: Icons.table_chart,
                      label: 'Databases',
                      value: '${usage['databases'] ?? 0}',
                      color: Colors.indigo,
                    ),
                    _StatCard(
                      icon: Icons.description,
                      label: 'Documents',
                      value: '${usage['documents'] ?? 0}',
                      color: Colors.purple,
                    ),
                    _StatCard(
                      icon: Icons.collections,
                      label: 'Collections',
                      value: '${usage['collections'] ?? 0}',
                      color: Colors.deepPurple,
                    ),
                    _StatCard(
                      icon: Icons.folder,
                      label: 'Buckets',
                      value: '${usage['buckets'] ?? 0}',
                      color: Colors.teal,
                    ),
                    _StatCard(
                      icon: Icons.insert_drive_file,
                      label: 'Files',
                      value: '${usage['files'] ?? 0}',
                      color: Colors.cyan,
                    ),
                    _StatCard(
                      icon: Icons.storage,
                      label: 'Storage',
                      value: _formatBytes(usage['storageBytes'] ?? 0),
                      color: Colors.green,
                    ),
                    _StatCard(
                      icon: Icons.groups,
                      label: 'Teams',
                      value: '${usage['teams'] ?? 0}',
                      color: Colors.orange,
                    ),
                  ],
                ),

                const SizedBox(height: 32),
                _SectionTitle('Automation'),
                const SizedBox(height: 12),
                Wrap(
                  spacing: 16,
                  runSpacing: 16,
                  children: [
                    _StatCard(
                      icon: Icons.account_tree,
                      label: 'Workflows',
                      value: '${usage['workflows'] ?? 0}',
                      color: Colors.amber,
                    ),
                    _StatCard(
                      icon: Icons.play_circle,
                      label: 'Executions',
                      value: '${usage['executions'] ?? 0}',
                      color: Colors.deepOrange,
                    ),
                    _StatCard(
                      icon: Icons.functions,
                      label: 'Functions',
                      value: '${usage['functions'] ?? 0}',
                      color: Colors.pink,
                    ),
                    _StatCard(
                      icon: Icons.rocket_launch,
                      label: 'Deployments',
                      value: '${usage['deployments'] ?? 0}',
                      color: Colors.red,
                    ),
                  ],
                ),

                const SizedBox(height: 32),
                _SectionTitle('Sessions'),
                const SizedBox(height: 12),
                _StatCard(
                  icon: Icons.login,
                  label: 'Active Sessions',
                  value: '${usage['sessions'] ?? 0}',
                  color: Colors.blueGrey,
                ),
              ],
            ),
          );
        },
      ),
    );
  }

  String _formatBytes(dynamic bytes) {
    final b = (bytes is int) ? bytes : 0;
    if (b < 1024) return '$b B';
    if (b < 1024 * 1024) return '${(b / 1024).toStringAsFixed(1)} KB';
    if (b < 1024 * 1024 * 1024) {
      return '${(b / (1024 * 1024)).toStringAsFixed(1)} MB';
    }
    return '${(b / (1024 * 1024 * 1024)).toStringAsFixed(1)} GB';
  }
}

class _SectionTitle extends StatelessWidget {
  final String title;
  const _SectionTitle(this.title);

  @override
  Widget build(BuildContext context) {
    return Text(title,
        style: Theme.of(context)
            .textTheme
            .titleMedium
            ?.copyWith(fontWeight: FontWeight.bold));
  }
}

class _StatCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final Color color;

  const _StatCard({
    required this.icon,
    required this.label,
    required this.value,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 180,
      child: Card(
        elevation: 2,
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(icon, color: color, size: 28),
              const SizedBox(height: 12),
              Text(
                value,
                style: Theme.of(context)
                    .textTheme
                    .headlineMedium
                    ?.copyWith(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 4),
              Text(label,
                  style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      color: Theme.of(context)
                          .colorScheme
                          .onSurface
                          .withOpacity(0.6))),
            ],
          ),
        ),
      ),
    );
  }
}
