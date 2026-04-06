import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';

class SettingsPage extends ConsumerWidget {
  const SettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final projectsAsync = ref.watch(projectsProvider);
    final currentProject = ref.watch(currentProjectProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings'),
        actions: [
          FilledButton.icon(
            onPressed: () => _showCreateProjectDialog(context, ref),
            icon: const Icon(Icons.add),
            label: const Text('New Project'),
          ),
          const SizedBox(width: 16),
        ],
      ),
      body: projectsAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('Error: $e'),
              const SizedBox(height: 8),
              FilledButton(
                onPressed: () => ref.invalidate(projectsProvider),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
        data: (projects) {
          if (projects.isEmpty) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.folder_open, size: 64),
                  const SizedBox(height: 16),
                  const Text('No projects yet'),
                  const SizedBox(height: 8),
                  FilledButton(
                    onPressed: () =>
                        _showCreateProjectDialog(context, ref),
                    child: const Text('Create Project'),
                  ),
                ],
              ),
            );
          }
          return Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Project list
              SizedBox(
                width: 300,
                child: ListView.builder(
                  padding: const EdgeInsets.all(16),
                  itemCount: projects.length,
                  itemBuilder: (context, i) {
                    final p = projects[i];
                    final id = p['\$id'] as String;
                    return Card(
                      child: ListTile(
                        leading: const Icon(Icons.folder),
                        title: Text(p['name'] ?? id),
                        subtitle: Text(id.length > 8 ? id.substring(0, 8) : id),
                        selected: currentProject == id,
                        onTap: () {
                          ref
                              .read(currentProjectProvider.notifier)
                              .state = id;
                          ref.read(apiClientProvider).setProject(id);
                        },
                        trailing: IconButton(
                          icon: const Icon(Icons.delete_outline),
                          onPressed: () =>
                              _deleteProject(context, ref, id),
                        ),
                      ),
                    );
                  },
                ),
              ),
              const VerticalDivider(width: 1),
              // Project detail / API keys
              if (currentProject != null)
                Expanded(
                    child: _ProjectDetail(projectId: currentProject)),
              if (currentProject == null)
                const Expanded(
                    child: Center(child: Text('Select a project'))),
            ],
          );
        },
      ),
    );
  }

  void _showCreateProjectDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    final descCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Create Project'),
        content: SizedBox(
          width: 400,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                  controller: nameCtrl,
                  decoration: const InputDecoration(labelText: 'Name')),
              const SizedBox(height: 8),
              TextField(
                  controller: descCtrl,
                  decoration:
                      const InputDecoration(labelText: 'Description')),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel')),
          FilledButton(
            onPressed: () async {
              await ref
                  .read(projectsProvider.notifier)
                  .create(nameCtrl.text, descCtrl.text);
              if (ctx.mounted) Navigator.pop(ctx);
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  Future<void> _deleteProject(
      BuildContext context, WidgetRef ref, String id) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Project'),
        content: const Text(
            'This will permanently delete the project and all its data.'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: const Text('Delete')),
        ],
      ),
    );
    if (confirmed == true) {
      await ref.read(projectsProvider.notifier).deleteProject(id);
      if (ref.read(currentProjectProvider) == id) {
        ref.read(currentProjectProvider.notifier).state = null;
      }
    }
  }
}

class _ProjectDetail extends ConsumerWidget {
  final String projectId;
  const _ProjectDetail({required this.projectId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final keysAsync = ref.watch(apiKeysProvider(projectId));

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Project: $projectId',
              style: Theme.of(context).textTheme.headlineSmall),
          const SizedBox(height: 24),
          Row(
            children: [
              Text('API Keys',
                  style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(width: 16),
              FilledButton.icon(
                onPressed: () =>
                    _showCreateKeyDialog(context, ref),
                icon: const Icon(Icons.add),
                label: const Text('Create Key'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          keysAsync.when(
            loading: () => const CircularProgressIndicator(),
            error: (e, _) => Text('Error: $e'),
            data: (keys) {
              if (keys.isEmpty) {
                return const Text('No API keys');
              }
              return Column(
                children: keys.map((k) {
                  return Card(
                    child: ListTile(
                      leading: const Icon(Icons.key),
                      title: Text(k['name'] ?? 'Unnamed'),
                      subtitle: Column(
                        crossAxisAlignment:
                            CrossAxisAlignment.start,
                        children: [
                          Text('ID: ${k['\$id']}'),
                          if (k['secret'] != null)
                            Row(
                              children: [
                                Expanded(
                                  child: SelectableText(
                                    k['secret'],
                                    style: const TextStyle(
                                        fontFamily: 'monospace'),
                                  ),
                                ),
                                IconButton(
                                  icon: const Icon(Icons.copy),
                                  onPressed: () {
                                    Clipboard.setData(ClipboardData(
                                        text: k['secret']));
                                    ScaffoldMessenger.of(context)
                                        .showSnackBar(
                                      const SnackBar(
                                          content: Text(
                                              'Copied to clipboard')),
                                    );
                                  },
                                ),
                              ],
                            ),
                        ],
                      ),
                      trailing: IconButton(
                        icon: const Icon(Icons.delete_outline),
                        onPressed: () => _deleteKey(ref, k['\$id']),
                      ),
                    ),
                  );
                }).toList(),
              );
            },
          ),
        ],
      ),
    );
  }

  void _showCreateKeyDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Create API Key'),
        content: TextField(
          controller: nameCtrl,
          decoration: const InputDecoration(labelText: 'Key Name'),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel')),
          FilledButton(
            onPressed: () async {
              final api = ref.read(apiClientProvider);
              await api.post('/projects/$projectId/keys', data: {
                'name': nameCtrl.text,
                'scopes': <String>[],
              });
              if (ctx.mounted) Navigator.pop(ctx);
              ref.invalidate(apiKeysProvider(projectId));
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  Future<void> _deleteKey(WidgetRef ref, String keyId) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/projects/$projectId/keys/$keyId');
    ref.invalidate(apiKeysProvider(projectId));
  }
}
