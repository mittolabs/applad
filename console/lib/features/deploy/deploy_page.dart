import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';
import '../../core/widgets/app_dialog.dart';

final deploymentsProvider =
    FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy');
  return res.data as Map<String, dynamic>;
});

class DeployPage extends ConsumerWidget {
  const DeployPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final deploymentsAsync = ref.watch(deploymentsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Deploy'),
        actions: [
          FilledButton.icon(
            onPressed: () => _showCreateDialog(context, ref),
            icon: const Icon(Icons.add),
            label: const Text('New Deployment'),
          ),
          const SizedBox(width: 16),
        ],
      ),
      body: deploymentsAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.error_outline, size: 48),
              const SizedBox(height: 16),
              Text('Failed to load deployments: $e'),
              const SizedBox(height: 8),
              FilledButton(
                onPressed: () => ref.invalidate(deploymentsProvider),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
        data: (data) {
          final deployments = List<Map<String, dynamic>>.from(
              data['deployments'] ?? []);
          if (deployments.isEmpty) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.rocket_launch_outlined,
                      size: 64, color: Colors.grey),
                  const SizedBox(height: 16),
                  const Text('No deployments yet'),
                  const SizedBox(height: 8),
                  FilledButton(
                    onPressed: () => _showCreateDialog(context, ref),
                    child: const Text('Create Deployment'),
                  ),
                ],
              ),
            );
          }
          return ListView.builder(
            padding: const EdgeInsets.all(16),
            itemCount: deployments.length,
            itemBuilder: (context, i) {
              final d = deployments[i];
              return Card(
                child: ListTile(
                  leading: _iconForType(d['type'] ?? 'web'),
                  title: Text(d['name'] ?? 'Unnamed'),
                  subtitle: Text(
                      '${d['type'] ?? 'web'} • ${d['\$id'] ?? ''}'),
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      _StatusChip(status: d['status'] ?? 'pending'),
                      const SizedBox(width: 8),
                      PopupMenuButton<String>(
                        onSelected: (action) {
                          if (action == 'delete') {
                            _delete(ref, d['\$id']);
                          } else {
                            _updateStatus(ref, d['\$id'], action);
                          }
                        },
                        itemBuilder: (_) => [
                          const PopupMenuItem(
                              value: 'building',
                              child: Text('Start Build')),
                          const PopupMenuItem(
                              value: 'deploying',
                              child: Text('Deploy')),
                          const PopupMenuItem(
                              value: 'active',
                              child: Text('Mark Active')),
                          const PopupMenuItem(
                              value: 'failed',
                              child: Text('Mark Failed')),
                          const PopupMenuDivider(),
                          const PopupMenuItem(
                              value: 'delete',
                              child: Text('Delete',
                                  style: TextStyle(color: Colors.red))),
                        ],
                      ),
                    ],
                  ),
                ),
              );
            },
          );
        },
      ),
    );
  }

  Widget _iconForType(String type) {
    switch (type) {
      case 'web':
        return const Icon(Icons.web);
      case 'function':
        return const Icon(Icons.functions);
      case 'container':
        return const Icon(Icons.inventory_2);
      default:
        return const Icon(Icons.cloud_upload);
    }
  }

  void _showCreateDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    String selectedType = 'web';

    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => Center(
          child: Material(
            color: Colors.transparent,
            child: Container(
              width: 440,
              constraints: const BoxConstraints(maxHeight: 600),
              decoration: BoxDecoration(
                color: const Color(0xFF16171B),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.white.withOpacity(0.08)),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black.withOpacity(0.5),
                    blurRadius: 32,
                    offset: const Offset(0, 8),
                  ),
                ],
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Padding(
                    padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
                    child: Row(
                      children: [
                        const Expanded(
                          child: Text('Create Deployment',
                              style: TextStyle(
                                color: Colors.white,
                                fontSize: 16,
                                fontWeight: FontWeight.w600,
                              )),
                        ),
                        GestureDetector(
                          onTap: () => Navigator.of(ctx).pop(),
                          child: Icon(Icons.close,
                              size: 16, color: Colors.white.withOpacity(0.3)),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 16),
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 20),
                    child: Container(
                        height: 1, color: Colors.white.withOpacity(0.06)),
                  ),
                  const SizedBox(height: 16),
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 20),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        AppDialogField(
                          controller: nameCtrl,
                          label: 'Name',
                          hint: 'Deployment name',
                          autofocus: true,
                        ),
                        const SizedBox(height: 16),
                        DropdownButtonFormField<String>(
                          value: selectedType,
                          dropdownColor: const Color(0xFF16171B),
                          style: const TextStyle(
                              color: Colors.white, fontSize: 13),
                          decoration: InputDecoration(
                            labelText: 'Type',
                            labelStyle: TextStyle(
                                color: Colors.white.withOpacity(0.5),
                                fontSize: 12),
                            filled: true,
                            fillColor: Colors.white.withOpacity(0.04),
                            border: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(8),
                              borderSide: BorderSide(
                                  color: Colors.white.withOpacity(0.1)),
                            ),
                            enabledBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(8),
                              borderSide: BorderSide(
                                  color: Colors.white.withOpacity(0.1)),
                            ),
                            focusedBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(8),
                              borderSide: const BorderSide(
                                  color: Color(0xFF3472A4)),
                            ),
                          ),
                          items: const [
                            DropdownMenuItem(
                                value: 'web', child: Text('Web')),
                            DropdownMenuItem(
                                value: 'function',
                                child: Text('Function')),
                            DropdownMenuItem(
                                value: 'container',
                                child: Text('Container')),
                          ],
                          onChanged: (v) =>
                              setState(() => selectedType = v ?? 'web'),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 16),
                  Padding(
                    padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.end,
                      children: [
                        const AppDialogCancel(),
                        AppDialogAction(
                          label: 'Create',
                          onTap: () async {
                            final api = ref.read(apiClientProvider);
                            await api.post('/deploy', data: {
                              'name': nameCtrl.text,
                              'type': selectedType,
                              'config': <String, dynamic>{},
                            });
                            if (ctx.mounted) Navigator.pop(ctx);
                            ref.invalidate(deploymentsProvider);
                          },
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _updateStatus(
      WidgetRef ref, String id, String status) async {
    final api = ref.read(apiClientProvider);
    await api.patch('/deploy/$id', data: {'status': status});
    ref.invalidate(deploymentsProvider);
  }

  Future<void> _delete(WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/deploy/$id');
    ref.invalidate(deploymentsProvider);
  }
}

class _StatusChip extends StatelessWidget {
  final String status;
  const _StatusChip({required this.status});

  @override
  Widget build(BuildContext context) {
    Color color;
    IconData icon;
    switch (status) {
      case 'active':
        color = Colors.green;
        icon = Icons.check_circle;
        break;
      case 'building':
        color = Colors.orange;
        icon = Icons.build;
        break;
      case 'deploying':
        color = Colors.blue;
        icon = Icons.cloud_upload;
        break;
      case 'failed':
        color = Colors.red;
        icon = Icons.error;
        break;
      default:
        color = Colors.grey;
        icon = Icons.schedule;
    }
    return Chip(
      avatar: Icon(icon, size: 16, color: color),
      label: Text(status),
      backgroundColor: color.withOpacity(0.1),
    );
  }
}
