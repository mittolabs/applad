import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';
import '../../core/widgets/app_dialog.dart';

final functionsProvider =
    FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/functions');
  return res.data as Map<String, dynamic>;
});

const _runtimes = [
  'node-18',
  'node-20',
  'node-22',
  'bun-1',
  'python-3.11',
  'python-3.12',
  'go-1.22',
  'dart-3',
  'rust-1',
  'ruby-3',
  'php-8',
  'custom',
];

class FunctionsPage extends ConsumerWidget {
  const FunctionsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final functionsAsync = ref.watch(functionsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Functions'),
        actions: [
          FilledButton.icon(
            onPressed: () => _showCreateDialog(context, ref),
            icon: const Icon(Icons.add),
            label: const Text('New Function'),
          ),
          const SizedBox(width: 16),
        ],
      ),
      body: functionsAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.error_outline, size: 48),
              const SizedBox(height: 16),
              Text('Failed to load functions: $e'),
              const SizedBox(height: 8),
              FilledButton(
                onPressed: () => ref.invalidate(functionsProvider),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
        data: (data) {
          final functions = List<Map<String, dynamic>>.from(
              data['functions'] ?? []);
          if (functions.isEmpty) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.functions_outlined,
                      size: 64, color: Colors.grey),
                  const SizedBox(height: 16),
                  const Text('No functions yet'),
                  const SizedBox(height: 8),
                  FilledButton(
                    onPressed: () => _showCreateDialog(context, ref),
                    child: const Text('Create Function'),
                  ),
                ],
              ),
            );
          }
          return ListView.builder(
            padding: const EdgeInsets.all(16),
            itemCount: functions.length,
            itemBuilder: (context, i) {
              final fn = functions[i];
              return Card(
                child: ListTile(
                  leading: _iconForRuntime(fn['runtime'] ?? 'custom'),
                  title: Text(fn['name'] ?? 'Unnamed'),
                  subtitle: Text(
                    '${fn['entrypoint'] ?? ''} • ${fn['\$id'] ?? ''}',
                  ),
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Chip(
                        label: Text(fn['runtime'] ?? 'custom'),
                        backgroundColor: Colors.deepPurple.withOpacity(0.1),
                      ),
                      const SizedBox(width: 8),
                      _StatusChip(status: fn['status'] ?? 'building'),
                      const SizedBox(width: 8),
                      PopupMenuButton<String>(
                        onSelected: (action) =>
                            _onAction(context, ref, action, fn),
                        itemBuilder: (_) => [
                          const PopupMenuItem(
                              value: 'execute',
                              child: Text('Execute')),
                          const PopupMenuItem(
                              value: 'executions',
                              child: Text('Executions')),
                          const PopupMenuDivider(),
                          const PopupMenuItem(
                              value: 'edit',
                              child: Text('Edit')),
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

  Widget _iconForRuntime(String runtime) {
    if (runtime.startsWith('node') || runtime.startsWith('bun')) {
      return const Icon(Icons.javascript);
    } else if (runtime.startsWith('python')) {
      return const Icon(Icons.data_object);
    } else if (runtime.startsWith('go')) {
      return const Icon(Icons.speed);
    } else if (runtime.startsWith('dart')) {
      return const Icon(Icons.flutter_dash);
    } else if (runtime.startsWith('rust')) {
      return const Icon(Icons.settings);
    } else if (runtime.startsWith('ruby')) {
      return const Icon(Icons.diamond);
    } else if (runtime.startsWith('php')) {
      return const Icon(Icons.php);
    }
    return const Icon(Icons.terminal);
  }

  void _onAction(BuildContext context, WidgetRef ref, String action,
      Map<String, dynamic> fn) {
    final id = fn['\$id'] as String;
    switch (action) {
      case 'execute':
        _execute(context, ref, id);
      case 'executions':
        _showExecutions(context, ref, id);
      case 'edit':
        _showEditDialog(context, ref, fn);
      case 'delete':
        _delete(ref, id);
    }
  }

  void _showCreateDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    final entrypointCtrl = TextEditingController();
    final timeoutCtrl = TextEditingController(text: '15');
    final sourceCtrl = TextEditingController();
    String selectedRuntime = 'node-20';

    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => Center(
          child: Material(
            color: Colors.transparent,
            child: Container(
              width: 500,
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
                          child: Text('Create Function',
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
                  Flexible(
                    child: SingleChildScrollView(
                      padding: const EdgeInsets.symmetric(horizontal: 20),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          AppDialogField(
                            controller: nameCtrl,
                            label: 'Name',
                            hint: 'Function name',
                            autofocus: true,
                          ),
                          const SizedBox(height: 12),
                          DropdownButtonFormField<String>(
                            value: selectedRuntime,
                            dropdownColor: const Color(0xFF16171B),
                            style: const TextStyle(
                                color: Colors.white, fontSize: 13),
                            decoration: InputDecoration(
                              labelText: 'Runtime',
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
                            items: _runtimes
                                .map((r) => DropdownMenuItem(
                                    value: r, child: Text(r)))
                                .toList(),
                            onChanged: (v) => setState(
                                () => selectedRuntime = v ?? 'node-20'),
                          ),
                          const SizedBox(height: 12),
                          AppDialogField(
                            controller: entrypointCtrl,
                            label: 'Entrypoint',
                            hint: 'index.js',
                          ),
                          const SizedBox(height: 12),
                          AppDialogField(
                            controller: timeoutCtrl,
                            label: 'Timeout (seconds)',
                            hint: '15',
                            keyboardType: TextInputType.number,
                          ),
                          const SizedBox(height: 12),
                          AppDialogField(
                            controller: sourceCtrl,
                            label: 'Source Code',
                            hint: '',
                            maxLines: 8,
                          ),
                        ],
                      ),
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
                            await api.post('/functions', data: {
                              'name': nameCtrl.text,
                              'runtime': selectedRuntime,
                              'entrypoint': entrypointCtrl.text,
                              'timeout':
                                  int.tryParse(timeoutCtrl.text) ?? 15,
                              'source': sourceCtrl.text,
                            });
                            if (ctx.mounted) Navigator.pop(ctx);
                            ref.invalidate(functionsProvider);
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

  void _showEditDialog(BuildContext context, WidgetRef ref,
      Map<String, dynamic> fn) {
    final id = fn['\$id'] as String;
    final nameCtrl = TextEditingController(text: fn['name'] ?? '');
    final entrypointCtrl =
        TextEditingController(text: fn['entrypoint'] ?? '');
    final timeoutCtrl =
        TextEditingController(text: '${fn['timeout'] ?? 15}');
    final sourceCtrl =
        TextEditingController(text: fn['source'] ?? '');
    String selectedRuntime = fn['runtime'] ?? 'node-20';

    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => Center(
          child: Material(
            color: Colors.transparent,
            child: Container(
              width: 500,
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
                          child: Text('Edit Function',
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
                  Flexible(
                    child: SingleChildScrollView(
                      padding: const EdgeInsets.symmetric(horizontal: 20),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          AppDialogField(
                            controller: nameCtrl,
                            label: 'Name',
                            hint: 'Function name',
                            autofocus: true,
                          ),
                          const SizedBox(height: 12),
                          DropdownButtonFormField<String>(
                            value: selectedRuntime,
                            dropdownColor: const Color(0xFF16171B),
                            style: const TextStyle(
                                color: Colors.white, fontSize: 13),
                            decoration: InputDecoration(
                              labelText: 'Runtime',
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
                            items: _runtimes
                                .map((r) => DropdownMenuItem(
                                    value: r, child: Text(r)))
                                .toList(),
                            onChanged: (v) => setState(
                                () => selectedRuntime = v ?? 'node-20'),
                          ),
                          const SizedBox(height: 12),
                          AppDialogField(
                            controller: entrypointCtrl,
                            label: 'Entrypoint',
                            hint: 'index.js',
                          ),
                          const SizedBox(height: 12),
                          AppDialogField(
                            controller: timeoutCtrl,
                            label: 'Timeout (seconds)',
                            hint: '15',
                            keyboardType: TextInputType.number,
                          ),
                          const SizedBox(height: 12),
                          AppDialogField(
                            controller: sourceCtrl,
                            label: 'Source Code',
                            hint: '',
                            maxLines: 8,
                          ),
                        ],
                      ),
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
                          label: 'Save',
                          onTap: () async {
                            final api = ref.read(apiClientProvider);
                            await api.put('/functions/$id', data: {
                              'name': nameCtrl.text,
                              'runtime': selectedRuntime,
                              'entrypoint': entrypointCtrl.text,
                              'timeout':
                                  int.tryParse(timeoutCtrl.text) ?? 15,
                              'source': sourceCtrl.text,
                            });
                            if (ctx.mounted) Navigator.pop(ctx);
                            ref.invalidate(functionsProvider);
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

  Future<void> _execute(
      BuildContext context, WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.post('/functions/$id/executions', data: <String, dynamic>{});
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Execution started')),
      );
    }
  }

  void _showExecutions(
      BuildContext context, WidgetRef ref, String functionId) {
    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => _ExecutionsDialog(
          functionId: functionId, ref: ref),
    );
  }

  Future<void> _delete(WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/functions/$id');
    ref.invalidate(functionsProvider);
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
      case 'building':
        color = Colors.orange;
        icon = Icons.build;
      case 'failed':
        color = Colors.red;
        icon = Icons.error;
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

// ---------- Executions dialog ----------

class _ExecutionsDialog extends StatelessWidget {
  final String functionId;
  final WidgetRef ref;
  const _ExecutionsDialog(
      {required this.functionId, required this.ref});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Material(
        color: Colors.transparent,
        child: Container(
          width: 600,
          constraints: const BoxConstraints(maxHeight: 520),
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
                      child: Text('Execution History',
                          style: TextStyle(
                            color: Colors.white,
                            fontSize: 16,
                            fontWeight: FontWeight.w600,
                          )),
                    ),
                    GestureDetector(
                      onTap: () => Navigator.of(context).pop(),
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
              Flexible(
                child: SizedBox(
                  height: 400,
                  child: FutureBuilder<dynamic>(
                    future: ref
                        .read(apiClientProvider)
                        .get('/functions/$functionId/executions')
                        .then((r) => r.data),
                    builder: (ctx, snap) {
                      if (snap.connectionState == ConnectionState.waiting) {
                        return const Center(child: CircularProgressIndicator());
                      }
                      if (snap.hasError) {
                        return Center(child: Text('Error: ${snap.error}',
                            style: TextStyle(color: Colors.white.withOpacity(0.7))));
                      }
                      final execs = List<Map<String, dynamic>>.from(
                          (snap.data as Map)['executions'] ?? []);
                      if (execs.isEmpty) {
                        return Center(child: Text('No executions yet',
                            style: TextStyle(color: Colors.white.withOpacity(0.4))));
                      }
                      return ListView.builder(
                        padding: const EdgeInsets.symmetric(horizontal: 20),
                        itemCount: execs.length,
                        itemBuilder: (ctx, i) {
                          final e = execs[i];
                          final status = e['status'] ?? 'pending';
                          final dur = e['duration'] ?? 0;
                          return ExpansionTile(
                            leading: _execIcon(status),
                            title: Text('${e['\$id'] ?? ''} — $status',
                                style: const TextStyle(color: Colors.white)),
                            subtitle: Text('${dur}ms',
                                style: TextStyle(
                                    color: Colors.white.withOpacity(0.4))),
                            children: [
                              if (e['output'] != null &&
                                  (e['output'] as String).isNotEmpty)
                                Padding(
                                  padding: const EdgeInsets.all(12),
                                  child: SelectableText(
                                    e['output'] as String,
                                    style: const TextStyle(
                                        fontFamily: 'monospace',
                                        fontSize: 13,
                                        color: Colors.white),
                                  ),
                                ),
                              if (e['error'] != null &&
                                  (e['error'] as String).isNotEmpty)
                                Padding(
                                  padding: const EdgeInsets.all(12),
                                  child: Text('Error: ${e['error']}',
                                      style: const TextStyle(color: Colors.red)),
                                ),
                            ],
                          );
                        },
                      );
                    },
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Padding(
                padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    const AppDialogCancel(label: 'Close'),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
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
