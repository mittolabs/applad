import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';

final workflowsProvider =
    FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/workflows');
  return res.data as Map<String, dynamic>;
});

class WorkflowsPage extends ConsumerWidget {
  const WorkflowsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final workflowsAsync = ref.watch(workflowsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Workflows'),
        actions: [
          FilledButton.icon(
            onPressed: () => _showCreateDialog(context, ref),
            icon: const Icon(Icons.add),
            label: const Text('New Workflow'),
          ),
          const SizedBox(width: 16),
        ],
      ),
      body: workflowsAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.error_outline, size: 48),
              const SizedBox(height: 16),
              Text('Failed to load workflows: $e'),
              const SizedBox(height: 8),
              FilledButton(
                onPressed: () => ref.invalidate(workflowsProvider),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
        data: (data) {
          final workflows = List<Map<String, dynamic>>.from(
              data['workflows'] ?? []);
          if (workflows.isEmpty) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.account_tree_outlined,
                      size: 64, color: Colors.grey),
                  const SizedBox(height: 16),
                  const Text('No workflows yet'),
                  const SizedBox(height: 8),
                  FilledButton(
                    onPressed: () => _showCreateDialog(context, ref),
                    child: const Text('Create Workflow'),
                  ),
                ],
              ),
            );
          }
          return ListView.builder(
            padding: const EdgeInsets.all(16),
            itemCount: workflows.length,
            itemBuilder: (context, i) {
              final wf = workflows[i];
              return Card(
                child: ListTile(
                  leading: _iconForTrigger(wf['triggerType'] ?? 'manual'),
                  title: Text(wf['name'] ?? 'Unnamed'),
                  subtitle: Text(
                    '${wf['triggerType'] ?? 'manual'} trigger • '
                    '${(wf['nodes'] as List?)?.length ?? 0} nodes • '
                    '${wf['\$id'] ?? ''}',
                  ),
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      _StatusChip(status: wf['status'] ?? 'draft'),
                      const SizedBox(width: 8),
                      PopupMenuButton<String>(
                        onSelected: (action) =>
                            _onAction(context, ref, action, wf),
                        itemBuilder: (_) => [
                          const PopupMenuItem(
                              value: 'execute',
                              child: Text('Execute')),
                          const PopupMenuItem(
                              value: 'executions',
                              child: Text('View Executions')),
                          const PopupMenuDivider(),
                          const PopupMenuItem(
                              value: 'activate',
                              child: Text('Activate')),
                          const PopupMenuItem(
                              value: 'pause',
                              child: Text('Pause')),
                          const PopupMenuItem(
                              value: 'nodes',
                              child: Text('Edit Nodes')),
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

  Widget _iconForTrigger(String trigger) {
    switch (trigger) {
      case 'webhook':
        return const Icon(Icons.webhook);
      case 'cron':
        return const Icon(Icons.schedule);
      default:
        return const Icon(Icons.play_circle_outline);
    }
  }

  void _onAction(BuildContext context, WidgetRef ref, String action,
      Map<String, dynamic> wf) {
    final id = wf['\$id'] as String;
    switch (action) {
      case 'execute':
        _execute(context, ref, id);
      case 'executions':
        _showExecutions(context, ref, id);
      case 'activate':
        _updateStatus(ref, id, wf, 'active');
      case 'pause':
        _updateStatus(ref, id, wf, 'paused');
      case 'nodes':
        _showNodeEditor(context, ref, id, wf);
      case 'delete':
        _delete(ref, id);
    }
  }

  void _showCreateDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    final descCtrl = TextEditingController();
    String triggerType = 'manual';
    final cronCtrl = TextEditingController();

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => AlertDialog(
          title: const Text('Create Workflow'),
          content: SizedBox(
            width: 440,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: nameCtrl,
                  decoration:
                      const InputDecoration(labelText: 'Name'),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: descCtrl,
                  decoration:
                      const InputDecoration(labelText: 'Description'),
                  maxLines: 2,
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  value: triggerType,
                  decoration:
                      const InputDecoration(labelText: 'Trigger Type'),
                  items: const [
                    DropdownMenuItem(
                        value: 'manual', child: Text('Manual')),
                    DropdownMenuItem(
                        value: 'webhook', child: Text('Webhook')),
                    DropdownMenuItem(
                        value: 'cron', child: Text('Cron Schedule')),
                  ],
                  onChanged: (v) =>
                      setState(() => triggerType = v ?? 'manual'),
                ),
                if (triggerType == 'cron') ...[
                  const SizedBox(height: 12),
                  TextField(
                    controller: cronCtrl,
                    decoration: const InputDecoration(
                      labelText: 'Cron Expression',
                      hintText: '*/5 * * * *',
                    ),
                  ),
                ],
              ],
            ),
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(ctx),
                child: const Text('Cancel')),
            FilledButton(
              onPressed: () async {
                final api = ref.read(apiClientProvider);
                await api.post('/workflows', data: {
                  'name': nameCtrl.text,
                  'description': descCtrl.text,
                  'triggerType': triggerType,
                  if (triggerType == 'cron')
                    'triggerConfig': {'cron': cronCtrl.text},
                });
                if (ctx.mounted) Navigator.pop(ctx);
                ref.invalidate(workflowsProvider);
              },
              child: const Text('Create'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _execute(
      BuildContext context, WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.post('/workflows/$id/execute',
        data: {'triggerData': <String, dynamic>{}});
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Execution started')),
      );
    }
  }

  Future<void> _updateStatus(
      WidgetRef ref, String id, Map<String, dynamic> wf,
      String status) async {
    final api = ref.read(apiClientProvider);
    await api.put('/workflows/$id', data: {
      'name': wf['name'],
      'status': status,
      'triggerType': wf['triggerType'] ?? 'manual',
      'nodes': wf['nodes'] ?? [],
      'edges': wf['edges'] ?? [],
    });
    ref.invalidate(workflowsProvider);
  }

  Future<void> _delete(WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/workflows/$id');
    ref.invalidate(workflowsProvider);
  }

  void _showExecutions(
      BuildContext context, WidgetRef ref, String workflowId) {
    showDialog(
      context: context,
      builder: (ctx) => _ExecutionsDialog(
          workflowId: workflowId, ref: ref),
    );
  }

  void _showNodeEditor(BuildContext context, WidgetRef ref, String id,
      Map<String, dynamic> wf) {
    showDialog(
      context: context,
      builder: (ctx) => _NodeEditorDialog(
          workflowId: id, workflow: wf, ref: ref),
    );
  }
}

// ---------- Status chip ----------

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
      case 'paused':
        color = Colors.orange;
        icon = Icons.pause_circle;
      default:
        color = Colors.grey;
        icon = Icons.edit_note;
    }
    return Chip(
      avatar: Icon(icon, size: 16, color: color),
      label: Text(status),
      backgroundColor: color.withOpacity(0.1),
    );
  }
}

// ---------- Executions dialog ----------

class _ExecutionsDialog extends StatelessWidget {
  final String workflowId;
  final WidgetRef ref;
  const _ExecutionsDialog(
      {required this.workflowId, required this.ref});

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Execution History'),
      content: SizedBox(
        width: 600,
        height: 400,
        child: FutureBuilder<dynamic>(
          future: ref
              .read(apiClientProvider)
              .get('/workflows/$workflowId/executions')
              .then((r) => r.data),
          builder: (ctx, snap) {
            if (snap.connectionState == ConnectionState.waiting) {
              return const Center(child: CircularProgressIndicator());
            }
            if (snap.hasError) {
              return Center(child: Text('Error: ${snap.error}'));
            }
            final execs = List<Map<String, dynamic>>.from(
                (snap.data as Map)['executions'] ?? []);
            if (execs.isEmpty) {
              return const Center(child: Text('No executions yet'));
            }
            return ListView.builder(
              itemCount: execs.length,
              itemBuilder: (ctx, i) {
                final e = execs[i];
                final status = e['status'] ?? 'pending';
                final dur = e['durationMs'] ?? 0;
                return ExpansionTile(
                  leading: _execIcon(status),
                  title: Text('${e['\$id'] ?? ''} — $status'),
                  subtitle: Text('${dur}ms'),
                  children: [
                    if (e['error'] != null &&
                        (e['error'] as String).isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.all(12),
                        child: Text('Error: ${e['error']}',
                            style: const TextStyle(color: Colors.red)),
                      ),
                    ...List<Map<String, dynamic>>.from(e['logs'] ?? [])
                        .map((log) => ListTile(
                              dense: true,
                              leading: Icon(
                                log['status'] == 'completed'
                                    ? Icons.check
                                    : log['status'] == 'skipped'
                                        ? Icons.skip_next
                                        : Icons.close,
                                size: 18,
                                color: log['status'] == 'completed'
                                    ? Colors.green
                                    : log['status'] == 'skipped'
                                        ? Colors.grey
                                        : Colors.red,
                              ),
                              title: Text(
                                  '${log['label']} (${log['nodeType']})'),
                              subtitle: Text('${log['durationMs']}ms'),
                              trailing: log['error'] != null &&
                                      (log['error'] as String)
                                          .isNotEmpty
                                  ? Tooltip(
                                      message: log['error'] as String,
                                      child: const Icon(Icons.warning,
                                          size: 16, color: Colors.red),
                                    )
                                  : null,
                            )),
                  ],
                );
              },
            );
          },
        ),
      ),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Close')),
      ],
    );
  }

  Widget _execIcon(String status) {
    switch (status) {
      case 'completed':
        return const Icon(Icons.check_circle, color: Colors.green);
      case 'running':
        return const Icon(Icons.sync, color: Colors.blue);
      case 'failed':
        return const Icon(Icons.error, color: Colors.red);
      default:
        return const Icon(Icons.schedule, color: Colors.grey);
    }
  }
}

// ---------- Node editor dialog ----------

const _nodeTypes = [
  'http_request',
  'send_email',
  'set_variable',
  'code',
  'if_condition',
  'delay',
];

class _NodeEditorDialog extends StatefulWidget {
  final String workflowId;
  final Map<String, dynamic> workflow;
  final WidgetRef ref;

  const _NodeEditorDialog({
    required this.workflowId,
    required this.workflow,
    required this.ref,
  });

  @override
  State<_NodeEditorDialog> createState() => _NodeEditorDialogState();
}

class _NodeEditorDialogState extends State<_NodeEditorDialog> {
  late List<Map<String, dynamic>> nodes;
  late List<Map<String, dynamic>> edges;

  @override
  void initState() {
    super.initState();
    nodes = List<Map<String, dynamic>>.from(
        widget.workflow['nodes'] ?? []);
    edges = List<Map<String, dynamic>>.from(
        widget.workflow['edges'] ?? []);
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Edit Nodes'),
      content: SizedBox(
        width: 560,
        height: 480,
        child: Column(
          children: [
            Row(
              children: [
                FilledButton.icon(
                  onPressed: _addNode,
                  icon: const Icon(Icons.add),
                  label: const Text('Add Node'),
                ),
                const Spacer(),
                Text('${nodes.length} nodes, ${edges.length} edges'),
              ],
            ),
            const SizedBox(height: 12),
            Expanded(
              child: nodes.isEmpty
                  ? const Center(child: Text('No nodes. Add one above.'))
                  : ReorderableListView.builder(
                      itemCount: nodes.length,
                      onReorder: _reorder,
                      itemBuilder: (ctx, i) {
                        final node = nodes[i];
                        return Card(
                          key: ValueKey(node['id']),
                          child: ExpansionTile(
                            leading: _nodeIcon(
                                node['type'] ?? 'http_request'),
                            title: Text(node['label'] ?? node['type']),
                            subtitle: Text(node['type'] ?? ''),
                            trailing: IconButton(
                              icon: const Icon(Icons.delete_outline,
                                  size: 20),
                              onPressed: () => setState(
                                  () => nodes.removeAt(i)),
                            ),
                            children: [
                              _nodeConfigForm(i, node),
                            ],
                          ),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel')),
        FilledButton(
          onPressed: _save,
          child: const Text('Save'),
        ),
      ],
    );
  }

  void _addNode() {
    setState(() {
      final id = 'node_${DateTime.now().millisecondsSinceEpoch}';
      nodes.add({
        'id': id,
        'type': 'http_request',
        'label': 'New Node',
        'config': <String, dynamic>{},
      });
      // Auto-chain: connect to previous node
      if (nodes.length > 1) {
        edges.add({
          'id': 'edge_${DateTime.now().millisecondsSinceEpoch}',
          'source': nodes[nodes.length - 2]['id'],
          'target': id,
        });
      }
    });
  }

  void _reorder(int oldIndex, int newIndex) {
    setState(() {
      if (newIndex > oldIndex) newIndex--;
      final item = nodes.removeAt(oldIndex);
      nodes.insert(newIndex, item);
      // Rebuild sequential edges
      _rebuildEdges();
    });
  }

  void _rebuildEdges() {
    edges.clear();
    for (int i = 0; i < nodes.length - 1; i++) {
      edges.add({
        'id': 'edge_${i}',
        'source': nodes[i]['id'],
        'target': nodes[i + 1]['id'],
      });
    }
  }

  Widget _nodeConfigForm(int index, Map<String, dynamic> node) {
    final config =
        Map<String, dynamic>.from(node['config'] ?? {});
    final type = node['type'] as String? ?? 'http_request';

    return Padding(
      padding: const EdgeInsets.all(12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Label + type
          Row(
            children: [
              Expanded(
                child: TextField(
                  decoration:
                      const InputDecoration(labelText: 'Label'),
                  controller:
                      TextEditingController(text: node['label'] ?? ''),
                  onChanged: (v) =>
                      setState(() => nodes[index]['label'] = v),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: DropdownButtonFormField<String>(
                  value: type,
                  decoration:
                      const InputDecoration(labelText: 'Type'),
                  items: _nodeTypes
                      .map((t) =>
                          DropdownMenuItem(value: t, child: Text(t)))
                      .toList(),
                  onChanged: (v) => setState(() {
                    nodes[index]['type'] = v ?? 'http_request';
                    nodes[index]['config'] = <String, dynamic>{};
                  }),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          // Type-specific config
          ..._configFieldsForType(index, type, config),
        ],
      ),
    );
  }

  List<Widget> _configFieldsForType(
      int index, String type, Map<String, dynamic> config) {
    switch (type) {
      case 'http_request':
        return [
          _configField(index, config, 'url', 'URL',
              hint: 'https://api.example.com/data'),
          _configField(index, config, 'method', 'Method',
              hint: 'GET'),
          _configField(index, config, 'body', 'Body (JSON)',
              maxLines: 3),
        ];
      case 'send_email':
        return [
          _configField(index, config, 'smtpHost', 'SMTP Host'),
          _configField(index, config, 'smtpPort', 'SMTP Port',
              hint: '587'),
          _configField(index, config, 'from', 'From'),
          _configField(index, config, 'to', 'To'),
          _configField(index, config, 'subject', 'Subject'),
          _configField(index, config, 'body', 'Body (HTML)',
              maxLines: 3),
        ];
      case 'set_variable':
        return [
          _configField(index, config, 'key', 'Key'),
          _configField(index, config, 'value', 'Value'),
        ];
      case 'code':
        return [
          _configField(
              index, config, 'expression', 'Expression (Go template)',
              maxLines: 4,
              hint: '{{.trigger.name}} is {{.trigger.status}}'),
        ];
      case 'if_condition':
        return [
          _configField(index, config, 'field', 'Field',
              hint: 'trigger.status'),
          _configField(index, config, 'operator', 'Operator',
              hint: 'eq, neq, contains, empty'),
          _configField(index, config, 'value', 'Value'),
        ];
      case 'delay':
        return [
          _configField(index, config, 'durationMs', 'Duration (ms)',
              hint: '1000'),
        ];
      default:
        return [];
    }
  }

  Widget _configField(int nodeIndex, Map<String, dynamic> config,
      String key, String label,
      {String? hint, int maxLines = 1}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: TextField(
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          isDense: true,
        ),
        maxLines: maxLines,
        controller:
            TextEditingController(text: '${config[key] ?? ''}'),
        onChanged: (v) => setState(() {
          final cfg =
              Map<String, dynamic>.from(nodes[nodeIndex]['config'] ?? {});
          cfg[key] = v;
          nodes[nodeIndex]['config'] = cfg;
        }),
      ),
    );
  }

  Widget _nodeIcon(String type) {
    switch (type) {
      case 'http_request':
        return const Icon(Icons.http);
      case 'send_email':
        return const Icon(Icons.email);
      case 'set_variable':
        return const Icon(Icons.data_object);
      case 'code':
        return const Icon(Icons.code);
      case 'if_condition':
        return const Icon(Icons.call_split);
      case 'delay':
        return const Icon(Icons.timer);
      default:
        return const Icon(Icons.extension);
    }
  }

  Future<void> _save() async {
    final api = widget.ref.read(apiClientProvider);
    await api.put('/workflows/${widget.workflowId}', data: {
      'name': widget.workflow['name'],
      'description': widget.workflow['description'] ?? '',
      'status': widget.workflow['status'] ?? 'draft',
      'triggerType': widget.workflow['triggerType'] ?? 'manual',
      'triggerConfig': widget.workflow['triggerConfig'] ?? {},
      'nodes': nodes,
      'edges': edges,
    });
    if (mounted) Navigator.pop(context);
    widget.ref.invalidate(workflowsProvider);
  }
}
