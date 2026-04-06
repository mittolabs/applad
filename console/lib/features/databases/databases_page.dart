import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/search_list.dart';

// --- Providers -----------------------------------------------------------

final _dbSearchProvider = StateProvider<String>((ref) => '');
final _dbPerPageProvider = StateProvider<int>((ref) => 12);
final _dbPageProvider = StateProvider<int>((ref) => 1);

final databasesProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final search = ref.watch(_dbSearchProvider);
  final limit = ref.watch(_dbPerPageProvider);
  final page = ref.watch(_dbPageProvider);
  final offset = (page - 1) * limit;
  final params = <String, dynamic>{'limit': limit, 'offset': offset};
  if (search.isNotEmpty) params['search'] = search;
  final res = await api.get('/databases', params: params);
  return res.data as Map<String, dynamic>;
});

final selectedDbProvider = StateProvider<String?>((ref) => null);

final tablesProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, dbId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/databases/$dbId/tables');
  return res.data as Map<String, dynamic>;
});

final selectedTableProvider = StateProvider<String?>((ref) => null);

final rowsProvider = FutureProvider.family<Map<String, dynamic>,
    ({String dbId, String tableId})>((ref, params) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get(
      '/databases/${params.dbId}/tables/${params.tableId}/rows',
      params: {'limit': 50});
  return res.data as Map<String, dynamic>;
});

// --- Page ----------------------------------------------------------------

class DatabasesPage extends ConsumerStatefulWidget {
  const DatabasesPage({super.key});

  @override
  ConsumerState<DatabasesPage> createState() => _DatabasesPageState();
}

class _DatabasesPageState extends ConsumerState<DatabasesPage> {
  final _searchCtrl = TextEditingController();

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  void _doSearch() {
    ref.read(_dbSearchProvider.notifier).state = _searchCtrl.text.trim();
    ref.read(_dbPageProvider.notifier).state = 1;
  }

  @override
  Widget build(BuildContext context) {
    final dbAsync = ref.watch(databasesProvider);
    final selectedDb = ref.watch(selectedDbProvider);
    final selectedTable = ref.watch(selectedTableProvider);
    final perPage = ref.watch(_dbPerPageProvider);
    final currentPage = ref.watch(_dbPageProvider);
    final total =
        dbAsync.whenOrNull(data: (d) => d['total'] as int? ?? 0) ?? 0;

    return Scaffold(
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(24, 20, 24, 0),
            child: Text('Databases',
                style: Theme.of(context)
                    .textTheme
                    .headlineSmall
                    ?.copyWith(color: Colors.white)),
          ),
          const SizedBox(height: 8),
          const Divider(height: 1, color: Color(0xFF2A2B30)),
          Expanded(
            child: Row(
              children: [
                // Database list panel with search + pagination
                SizedBox(
                  width: 280,
                  child: Column(
                    children: [
                      // Mini search header for databases panel
                      _PanelSearchHeader(
                        searchController: _searchCtrl,
                        onSearch: _doSearch,
                        onAdd: () => _showCreateDbDialog(context, ref),
                      ),
                      Expanded(
                        child: dbAsync.when(
                          loading: () => const Center(
                              child: CircularProgressIndicator()),
                          error: (e, _) =>
                              Center(child: Text('Error: $e')),
                          data: (data) {
                            final dbs =
                                List<Map<String, dynamic>>.from(
                                    data['databases'] ?? []);
                            if (dbs.isEmpty) {
                              return const Center(
                                  child: Text('No databases'));
                            }
                            return ListView.builder(
                              itemCount: dbs.length,
                              itemBuilder: (context, i) {
                                final db = dbs[i];
                                final id = db['\$id'] as String;
                                return ListTile(
                                  leading:
                                      const Icon(Icons.storage),
                                  title: Text(db['name'] ?? id),
                                  selected: selectedDb == id,
                                  onTap: () {
                                    ref
                                        .read(selectedDbProvider
                                            .notifier)
                                        .state = id;
                                    ref
                                        .read(selectedTableProvider
                                            .notifier)
                                        .state = null;
                                  },
                                  trailing: IconButton(
                                    icon: const Icon(
                                        Icons.delete_outline,
                                        size: 18),
                                    onPressed: () =>
                                        _deleteDb(ref, id),
                                  ),
                                );
                              },
                            );
                          },
                        ),
                      ),
                      // Pagination footer for databases panel
                      SearchListFooter(
                        total: total,
                        perPage: perPage,
                        currentPage: currentPage,
                        onPrev: () => ref
                            .read(_dbPageProvider.notifier)
                            .update((s) => s - 1),
                        onNext: () => ref
                            .read(_dbPageProvider.notifier)
                            .update((s) => s + 1),
                        onPerPageChanged: (v) {
                          ref.read(_dbPerPageProvider.notifier).state =
                              v;
                          ref.read(_dbPageProvider.notifier).state = 1;
                        },
                      ),
                    ],
                  ),
                ),
                const VerticalDivider(width: 1),
                // Tables panel
                if (selectedDb != null)
                  SizedBox(
                    width: 240,
                    child: _TablesPanel(dbId: selectedDb),
                  ),
                if (selectedDb != null)
                  const VerticalDivider(width: 1),
                // Rows panel
                if (selectedDb != null && selectedTable != null)
                  Expanded(
                    child: _RowsPanel(
                        dbId: selectedDb, tableId: selectedTable),
                  ),
                if (selectedDb == null)
                  const Expanded(
                    child:
                        Center(child: Text('Select a database')),
                  ),
                if (selectedDb != null && selectedTable == null)
                  const Expanded(
                    child: Center(child: Text('Select a table')),
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  void _showCreateDbDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create Database',
      content: AppDialogField(
        controller: nameCtrl,
        label: 'Name',
        hint: 'Database name',
        autofocus: true,
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            final api = ref.read(apiClientProvider);
            await api.post('/databases', data: {
              'databaseId': 'unique()',
              'name': nameCtrl.text,
            });
            if (context.mounted) Navigator.pop(context);
            ref.invalidate(databasesProvider);
          },
        ),
      ],
    );
  }

  Future<void> _deleteDb(WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/databases/$id');
    ref.read(selectedDbProvider.notifier).state = null;
    ref.invalidate(databasesProvider);
  }
}

// --- Panel search header (compact, for side panels) ----------------------

class _PanelSearchHeader extends StatelessWidget {
  final TextEditingController searchController;
  final VoidCallback onSearch;
  final VoidCallback onAdd;

  const _PanelSearchHeader({
    required this.searchController,
    required this.onSearch,
    required this.onAdd,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(8),
      color: const Color(0xFF16171B),
      child: Row(
        children: [
          Expanded(
            child: SizedBox(
              height: 32,
              child: TextField(
                controller: searchController,
                onSubmitted: (_) => onSearch(),
                style:
                    const TextStyle(fontSize: 12, color: Colors.white),
                decoration: InputDecoration(
                  hintText: 'Search...',
                  hintStyle: const TextStyle(
                      color: Color(0x40FFFFFF), fontSize: 12),
                  prefixIcon: const Icon(Icons.search,
                      size: 16, color: Color(0x40FFFFFF)),
                  filled: true,
                  fillColor: const Color(0x0AFFFFFF),
                  contentPadding: const EdgeInsets.symmetric(
                      vertical: 0, horizontal: 8),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(6),
                    borderSide: BorderSide.none,
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(width: 4),
          SizedBox(
            width: 32,
            height: 32,
            child: IconButton(
              padding: EdgeInsets.zero,
              iconSize: 18,
              icon: const Icon(Icons.add, color: Colors.white),
              onPressed: onAdd,
            ),
          ),
        ],
      ),
    );
  }
}

// --- Tables panel --------------------------------------------------------

class _TablesPanel extends ConsumerWidget {
  final String dbId;
  const _TablesPanel({required this.dbId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final tableAsync = ref.watch(tablesProvider(dbId));
    final selectedTable = ref.watch(selectedTableProvider);

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(8),
          child: Row(
            children: [
              const Text('Tables'),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.add),
                onPressed: () =>
                    _showCreateTableDialog(context, ref),
              ),
            ],
          ),
        ),
        Expanded(
          child: tableAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(child: Text('Error: $e')),
            data: (data) {
              final tables = List<Map<String, dynamic>>.from(
                  data['tables'] ?? []);
              if (tables.isEmpty) {
                return const Center(child: Text('No tables'));
              }
              return ListView.builder(
                itemCount: tables.length,
                itemBuilder: (context, i) {
                  final t = tables[i];
                  final id = t['\$id'] as String;
                  return ListTile(
                    leading: const Icon(Icons.list_alt),
                    title: Text(t['name'] ?? id),
                    selected: selectedTable == id,
                    onTap: () => ref
                        .read(selectedTableProvider.notifier)
                        .state = id,
                    trailing: IconButton(
                      icon:
                          const Icon(Icons.delete_outline, size: 18),
                      onPressed: () => _deleteTable(ref, id),
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

  void _showCreateTableDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create Table',
      content: AppDialogField(
        controller: nameCtrl,
        label: 'Name',
        hint: 'Table name',
        autofocus: true,
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            final api = ref.read(apiClientProvider);
            await api.post('/databases/$dbId/tables', data: {
              'tableId': 'unique()',
              'name': nameCtrl.text,
              'permissions': <String>[],
            });
            if (context.mounted) Navigator.pop(context);
            ref.invalidate(tablesProvider(dbId));
          },
        ),
      ],
    );
  }

  Future<void> _deleteTable(WidgetRef ref, String id) async {
    final api = ref.read(apiClientProvider);
    await api.delete('/databases/$dbId/tables/$id');
    ref.read(selectedTableProvider.notifier).state = null;
    ref.invalidate(tablesProvider(dbId));
  }
}

// --- Rows panel ----------------------------------------------------------

class _RowsPanel extends ConsumerWidget {
  final String dbId;
  final String tableId;
  const _RowsPanel(
      {required this.dbId, required this.tableId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final rowsAsync = ref.watch(
        rowsProvider((dbId: dbId, tableId: tableId)));

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Text('Rows',
                  style: Theme.of(context).textTheme.titleMedium),
              const Spacer(),
              FilledButton.icon(
                onPressed: () =>
                    _showCreateRowDialog(context, ref),
                icon: const Icon(Icons.add),
                label: const Text('Add Row'),
              ),
            ],
          ),
        ),
        Expanded(
          child: rowsAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(child: Text('Error: $e')),
            data: (data) {
              final rows = List<Map<String, dynamic>>.from(
                  data['rows'] ?? []);
              if (rows.isEmpty) {
                return const Center(child: Text('No rows'));
              }

              final dataKeys = <String>{};
              for (final row in rows) {
                for (final key in row.keys) {
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
                    rows: rows.map((row) {
                      return DataRow(
                        cells: [
                          DataCell(SelectableText(
                            (row['\$id'] ?? '').toString(),
                            style: const TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 12),
                          )),
                          ...columns.map((col) {
                            final val = row[col];
                            return DataCell(
                              SelectableText(
                                val?.toString() ?? '',
                                style: const TextStyle(fontSize: 13),
                              ),
                            );
                          }),
                          DataCell(Text(
                            _formatTimestamp(
                                row['\$createdAt']?.toString() ?? ''),
                            style: const TextStyle(fontSize: 12),
                          )),
                          DataCell(
                            IconButton(
                              icon: const Icon(Icons.delete_outline,
                                  size: 18),
                              onPressed: () =>
                                  _deleteRow(ref, row['\$id']),
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
        rowsAsync.whenOrNull(
              data: (data) => Padding(
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                child: Text(
                  '${data['total'] ?? 0} rows',
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

  void _showCreateRowDialog(BuildContext context, WidgetRef ref) {
    final dataCtrl = TextEditingController(text: '{}');
    showAppDialog(
      context: context,
      title: 'Create Row',
      content: AppDialogField(
        controller: dataCtrl,
        label: 'Data (JSON)',
        hint: '{}',
        maxLines: 8,
        autofocus: true,
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            final api = ref.read(apiClientProvider);
            await api.post(
              '/databases/$dbId/tables/$tableId/rows',
              data: {
                'rowId': 'unique()',
                'data': dataCtrl.text,
                'permissions': <String>[],
              },
            );
            if (context.mounted) Navigator.pop(context);
            ref.invalidate(
                rowsProvider((dbId: dbId, tableId: tableId)));
          },
        ),
      ],
    );
  }

  Future<void> _deleteRow(WidgetRef ref, String rowId) async {
    final api = ref.read(apiClientProvider);
    await api.delete(
        '/databases/$dbId/tables/$tableId/rows/$rowId');
    ref.invalidate(
        rowsProvider((dbId: dbId, tableId: tableId)));
  }
}
