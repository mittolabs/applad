import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';

final databasesProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/databases');
  return res.data as Map<String, dynamic>;
});

final selectedDbProvider = StateProvider<String?>((ref) => null);

final collectionsProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, dbId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/databases/$dbId/collections');
  return res.data as Map<String, dynamic>;
});

final selectedCollProvider = StateProvider<String?>((ref) => null);

final documentsProvider = FutureProvider.family<Map<String, dynamic>,
    ({String dbId, String collId})>((ref, params) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get(
      '/databases/${params.dbId}/collections/${params.collId}/documents',
      params: {'limit': 50});
  return res.data as Map<String, dynamic>;
});

class DatabasesPage extends ConsumerWidget {
  const DatabasesPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final dbAsync = ref.watch(databasesProvider);
    final selectedDb = ref.watch(selectedDbProvider);
    final selectedColl = ref.watch(selectedCollProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Databases'),
        actions: [
          FilledButton.icon(
            onPressed: () => _showCreateDbDialog(context, ref),
            icon: const Icon(Icons.add),
            label: const Text('Create Database'),
          ),
          const SizedBox(width: 16),
        ],
      ),
      body: Row(
        children: [
          // Database list panel
          SizedBox(
            width: 240,
            child: dbAsync.when(
              loading: () =>
                  const Center(child: CircularProgressIndicator()),
              error: (e, _) => Center(child: Text('Error: $e')),
              data: (data) {
                final dbs = List<Map<String, dynamic>>.from(
                    data['databases'] ?? []);
                if (dbs.isEmpty) {
                  return const Center(child: Text('No databases'));
                }
                return ListView.builder(
                  itemCount: dbs.length,
                  itemBuilder: (context, i) {
                    final db = dbs[i];
                    final id = db['\$id'] as String;
                    return ListTile(
                      leading: const Icon(Icons.storage),
                      title: Text(db['name'] ?? id),
                      selected: selectedDb == id,
                      onTap: () {
                        ref.read(selectedDbProvider.notifier).state = id;
                        ref.read(selectedCollProvider.notifier).state =
                            null;
                      },
                      trailing: IconButton(
                        icon: const Icon(Icons.delete_outline, size: 18),
                        onPressed: () => _deleteDb(ref, id),
                      ),
                    );
                  },
                );
              },
            ),
          ),
          const VerticalDivider(width: 1),
          // Collections panel
          if (selectedDb != null)
            SizedBox(
              width: 240,
              child: _CollectionsPanel(dbId: selectedDb),
            ),
          if (selectedDb != null) const VerticalDivider(width: 1),
          // Documents panel
          if (selectedDb != null && selectedColl != null)
            Expanded(
              child: _DocumentsPanel(
                  dbId: selectedDb, collId: selectedColl),
            ),
          if (selectedDb == null)
            const Expanded(
              child: Center(child: Text('Select a database')),
            ),
          if (selectedDb != null && selectedColl == null)
            const Expanded(
              child: Center(child: Text('Select a collection')),
            ),
        ],
      ),
    );
  }

  void _showCreateDbDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Create Database'),
        content: TextField(
          controller: nameCtrl,
          decoration: const InputDecoration(labelText: 'Name'),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel')),
          FilledButton(
            onPressed: () async {
              final api = ref.read(apiClientProvider);
              await api.post('/databases', data: {
                'databaseId': 'unique()',
                'name': nameCtrl.text,
              });
              if (ctx.mounted) Navigator.pop(ctx);
              ref.invalidate(databasesProvider);
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  Future<void> _deleteDb(WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/databases/$id');
    ref.read(selectedDbProvider.notifier).state = null;
    ref.invalidate(databasesProvider);
  }
}

class _CollectionsPanel extends ConsumerWidget {
  final String dbId;
  const _CollectionsPanel({required this.dbId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final collAsync = ref.watch(collectionsProvider(dbId));
    final selectedColl = ref.watch(selectedCollProvider);

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(8),
          child: Row(
            children: [
              const Text('Collections'),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.add),
                onPressed: () =>
                    _showCreateCollDialog(context, ref),
              ),
            ],
          ),
        ),
        Expanded(
          child: collAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(child: Text('Error: $e')),
            data: (data) {
              final colls = List<Map<String, dynamic>>.from(
                  data['collections'] ?? []);
              if (colls.isEmpty) {
                return const Center(child: Text('No collections'));
              }
              return ListView.builder(
                itemCount: colls.length,
                itemBuilder: (context, i) {
                  final c = colls[i];
                  final id = c['\$id'] as String;
                  return ListTile(
                    leading: const Icon(Icons.list_alt),
                    title: Text(c['name'] ?? id),
                    selected: selectedColl == id,
                    onTap: () => ref
                        .read(selectedCollProvider.notifier)
                        .state = id,
                    trailing: IconButton(
                      icon:
                          const Icon(Icons.delete_outline, size: 18),
                      onPressed: () => _deleteColl(ref, id),
                    ),
                  );
                },
              );
            },
          ),
        ),
      ],
    );
  }

  void _showCreateCollDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Create Collection'),
        content: TextField(
          controller: nameCtrl,
          decoration: const InputDecoration(labelText: 'Name'),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel')),
          FilledButton(
            onPressed: () async {
              final api = ref.read(apiClientProvider);
              await api.post('/databases/$dbId/collections', data: {
                'collectionId': 'unique()',
                'name': nameCtrl.text,
                'permissions': <String>[],
              });
              if (ctx.mounted) Navigator.pop(ctx);
              ref.invalidate(collectionsProvider(dbId));
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  Future<void> _deleteColl(WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/databases/$dbId/collections/$id');
    ref.read(selectedCollProvider.notifier).state = null;
    ref.invalidate(collectionsProvider(dbId));
  }
}

class _DocumentsPanel extends ConsumerWidget {
  final String dbId;
  final String collId;
  const _DocumentsPanel(
      {required this.dbId, required this.collId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final docsAsync = ref.watch(
        documentsProvider((dbId: dbId, collId: collId)));

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Text('Documents',
                  style: Theme.of(context).textTheme.titleMedium),
              const Spacer(),
              FilledButton.icon(
                onPressed: () =>
                    _showCreateDocDialog(context, ref),
                icon: const Icon(Icons.add),
                label: const Text('Add Document'),
              ),
            ],
          ),
        ),
        Expanded(
          child: docsAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(child: Text('Error: $e')),
            data: (data) {
              final docs = List<Map<String, dynamic>>.from(
                  data['documents'] ?? []);
              if (docs.isEmpty) {
                return const Center(child: Text('No documents'));
              }

              // Extract all unique data keys across all documents
              final dataKeys = <String>{};
              for (final doc in docs) {
                for (final key in doc.keys) {
                  if (!key.startsWith('\$')) {
                    dataKeys.add(key);
                  }
                }
              }
              final columns = dataKeys.toList()..sort();

              return SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: SingleChildScrollView(
                  child: DataTable(
                    columnSpacing: 24,
                    headingRowColor: WidgetStateProperty.all(
                      Theme.of(context)
                          .colorScheme
                          .surfaceContainerHighest,
                    ),
                    columns: [
                      const DataColumn(
                          label: Text('\$id',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold))),
                      ...columns.map((col) => DataColumn(
                          label: Text(col,
                              style: const TextStyle(
                                  fontWeight: FontWeight.bold)))),
                      const DataColumn(
                          label: Text('\$createdAt',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold))),
                      const DataColumn(label: Text('')),
                    ],
                    rows: docs.map((doc) {
                      return DataRow(
                        cells: [
                          DataCell(SelectableText(
                            (doc['\$id'] ?? '').toString(),
                            style: const TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 12),
                          )),
                          ...columns.map((col) {
                            final val = doc[col];
                            return DataCell(
                              SelectableText(
                                val?.toString() ?? '',
                                style: const TextStyle(fontSize: 13),
                              ),
                            );
                          }),
                          DataCell(Text(
                            _formatTimestamp(
                                doc['\$createdAt']?.toString() ?? ''),
                            style: const TextStyle(fontSize: 12),
                          )),
                          DataCell(
                            IconButton(
                              icon: const Icon(Icons.delete_outline,
                                  size: 18),
                              onPressed: () =>
                                  _deleteDoc(ref, doc['\$id']),
                            ),
                          ),
                        ],
                      );
                    }).toList(),
                  ),
                ),
              );
            },
          ),
        ),
        // Total count footer
        docsAsync.whenOrNull(
              data: (data) => Padding(
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                child: Text(
                  '${data['total'] ?? 0} documents',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ),
            ) ??
            const SizedBox.shrink(),
      ],
    );
  }

  String _formatTimestamp(String ts) {
    if (ts.isEmpty) return '';
    try {
      final dt = DateTime.parse(ts);
      return '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')} '
          '${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    } catch (_) {
      return ts;
    }
  }

  void _showCreateDocDialog(BuildContext context, WidgetRef ref) {
    final dataCtrl = TextEditingController(text: '{}');
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Create Document'),
        content: SizedBox(
          width: 400,
          child: TextField(
            controller: dataCtrl,
            maxLines: 8,
            decoration: const InputDecoration(
              labelText: 'Data (JSON)',
              border: OutlineInputBorder(),
            ),
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel')),
          FilledButton(
            onPressed: () async {
              final api = ref.read(apiClientProvider);
              await api.post(
                '/databases/$dbId/collections/$collId/documents',
                data: {
                  'documentId': 'unique()',
                  'data': dataCtrl.text,
                  'permissions': <String>[],
                },
              );
              if (ctx.mounted) Navigator.pop(ctx);
              ref.invalidate(
                  documentsProvider((dbId: dbId, collId: collId)));
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  Future<void> _deleteDoc(WidgetRef ref, String docId) async {
    final api = ref.read(apiClientProvider);
    await api.delete(
        '/databases/$dbId/collections/$collId/documents/$docId');
    ref.invalidate(
        documentsProvider((dbId: dbId, collId: collId)));
  }
}
