import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/widgets/app_dialog.dart';

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
    showAppDialog(
      context: context,
      title: 'Create Project',
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          AppDialogField(
            controller: nameCtrl,
            label: 'Name',
            hint: 'Project name',
            autofocus: true,
          ),
          const SizedBox(height: 12),
          AppDialogField(
            controller: descCtrl,
            label: 'Description',
            hint: 'Optional description',
          ),
        ],
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            await ref
                .read(projectsProvider.notifier)
                .create(nameCtrl.text, descCtrl.text);
            if (context.mounted) Navigator.pop(context);
          },
        ),
      ],
    );
  }

  Future<void> _deleteProject(
      BuildContext context, WidgetRef ref, String id) async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete Project',
      content: Text(
        'This will permanently delete the project and all its data.',
        style: TextStyle(color: Colors.white.withOpacity(0.7)),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Delete',
          destructive: true,
          onTap: () => Navigator.of(context).pop(true),
        ),
      ],
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
    showAppDialog(
      context: context,
      title: 'Create API Key',
      content: AppDialogField(
        controller: nameCtrl,
        label: 'Key Name',
        hint: 'e.g. Production key',
        autofocus: true,
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            final api = ref.read(apiClientProvider);
            await api.post('/projects/$projectId/keys', data: {
              'name': nameCtrl.text,
              'scopes': <String>[],
            });
            if (context.mounted) Navigator.pop(context);
            ref.invalidate(apiKeysProvider(projectId));
          },
        ),
      ],
    );
  }

  Future<void> _deleteKey(WidgetRef ref, String keyId) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/projects/$projectId/keys/$keyId');
    ref.invalidate(apiKeysProvider(projectId));
  }
}
