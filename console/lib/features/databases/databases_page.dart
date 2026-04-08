import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/search_list.dart';

// --- Constants ---------------------------------------------------------------

const _bgColor = Color(0xFF0B0B0F);
const _cardColor = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _border = Color(0x0FFFFFFF);
const _red = Color(0xFFEF4444);
const _green = Color(0xFF10B981);

// --- Providers ---------------------------------------------------------------

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

final _tablesProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, dbId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/databases/$dbId/tables');
  return res.data as Map<String, dynamic>;
});

final _columnsProvider = FutureProvider.family<List<Map<String, dynamic>>,
    ({String dbId, String tableId})>((ref, p) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/databases/${p.dbId}/tables/${p.tableId}/columns');
  final data = res.data as Map<String, dynamic>;
  return List<Map<String, dynamic>>.from(data['columns'] ?? []);
});

final _indexesProvider = FutureProvider.family<List<Map<String, dynamic>>,
    ({String dbId, String tableId})>((ref, p) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/databases/${p.dbId}/tables/${p.tableId}/indexes');
  final data = res.data as Map<String, dynamic>;
  return List<Map<String, dynamic>>.from(data['indexes'] ?? []);
});

final _relationshipsProvider = FutureProvider.family<
    List<Map<String, dynamic>>,
    ({String dbId, String tableId})>((ref, p) async {
  final api = ref.read(apiClientProvider);
  final res =
      await api.get('/databases/${p.dbId}/tables/${p.tableId}/relationships');
  final data = res.data as Map<String, dynamic>;
  return List<Map<String, dynamic>>.from(data['relationships'] ?? []);
});

final _rowsProvider = FutureProvider.family<Map<String, dynamic>,
    ({String dbId, String tableId})>((ref, p) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get(
      '/databases/${p.dbId}/tables/${p.tableId}/rows',
      params: {'limit': 100});
  return res.data as Map<String, dynamic>;
});

// --- Page (3-level navigation like Storage) ----------------------------------

class DatabasesPage extends ConsumerStatefulWidget {
  const DatabasesPage({super.key});

  @override
  ConsumerState<DatabasesPage> createState() => _DatabasesPageState();
}

class _DatabasesPageState extends ConsumerState<DatabasesPage> {
  final _searchCtrl = TextEditingController();
  String? _selectedDbId;
  String? _selectedTableId;

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_selectedTableId != null && _selectedDbId != null) {
      return _TableDetailView(
        dbId: _selectedDbId!,
        tableId: _selectedTableId!,
        onBack: () => setState(() => _selectedTableId = null),
      );
    }
    if (_selectedDbId != null) {
      return _DatabaseDetailView(
        dbId: _selectedDbId!,
        onBack: () => setState(() => _selectedDbId = null),
        onTableSelect: (id) => setState(() => _selectedTableId = id),
      );
    }
    return _buildDatabaseList();
  }

  // ===========================================================================
  // Level 1: Database List
  // ===========================================================================

  Widget _buildDatabaseList() {
    final dbAsync = ref.watch(databasesProvider);
    final perPage = ref.watch(_dbPerPageProvider);
    final currentPage = ref.watch(_dbPageProvider);
    final total =
        dbAsync.whenOrNull(data: (d) => d['total'] as int? ?? 0) ?? 0;

    return Scaffold(
      backgroundColor: _bgColor,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: MediaQuery.of(context).size.width > 1400 ? 80 : 40,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            const Text('Databases',
                style: TextStyle(
                    color: Colors.white,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 24),
            PageTabs(
              tabs: const ['Databases', 'Usage'],
              selected: 0,
              onChanged: (_) {},
            ),
            const SizedBox(height: 20),
            Row(
              children: [
                _SearchBox(
                  controller: _searchCtrl,
                  hint: 'Search by name or ID',
                  onChanged: (v) {
                    ref.read(_dbSearchProvider.notifier).state = v;
                    ref.read(_dbPageProvider.notifier).state = 1;
                  },
                ),
                const Spacer(),
                _AccentButton(
                  label: 'Create database',
                  onTap: _showCreateDbDialog,
                ),
              ],
            ),
            const SizedBox(height: 16),
            Expanded(
              child: dbAsync.when(
                loading: () =>
                    const Center(child: CircularProgressIndicator()),
                error: (e, _) => Center(
                    child: Text('Error: $e',
                        style: const TextStyle(color: _red))),
                data: (data) {
                  final dbs = List<Map<String, dynamic>>.from(
                      data['databases'] ?? []);
                  if (dbs.isEmpty) {
                    return _EmptyState(
                      icon: LucideIcons.database,
                      title: 'No databases',
                      subtitle: 'Create a database to get started',
                      actionLabel: 'Create database',
                      onAction: _showCreateDbDialog,
                    );
                  }
                  return _DataTable(
                    headers: const ['Database ID', 'Name', 'Created'],
                    flexes: const [3, 3, 2],
                    rows: dbs,
                    rowBuilder: (db) {
                      final id = db['\$id'] as String? ?? '';
                      final name = db['name'] as String? ?? '';
                      final created = _fmtDate(db['createdAt'] ?? db['\$createdAt']);
                      return [
                        Row(children: [
                          Icon(LucideIcons.database, size: 14, color: _accent),
                          const SizedBox(width: 8),
                          Expanded(
                              child: Text(id,
                                  style: const TextStyle(
                                      color: Colors.white,
                                      fontSize: 13,
                                      fontFamily: 'monospace'),
                                  overflow: TextOverflow.ellipsis)),
                        ]),
                        Text(name,
                            style: const TextStyle(
                                color: Colors.white, fontSize: 13)),
                        Text(created,
                            style: const TextStyle(
                                color: _dimText, fontSize: 12)),
                      ];
                    },
                    onRowTap: (db) =>
                        setState(() => _selectedDbId = db['\$id'] as String),
                    onRowDelete: (db) => _deleteDb(db['\$id'] as String),
                  );
                },
              ),
            ),
            SearchListFooter(
              total: total,
              perPage: perPage,
              currentPage: currentPage,
              onPrev: () =>
                  ref.read(_dbPageProvider.notifier).update((s) => s - 1),
              onNext: () =>
                  ref.read(_dbPageProvider.notifier).update((s) => s + 1),
              onPerPageChanged: (v) {
                ref.read(_dbPerPageProvider.notifier).state = v;
                ref.read(_dbPageProvider.notifier).state = 1;
              },
              itemLabel: 'Databases',
            ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }

  void _showCreateDbDialog() {
    final nameCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create database',
      content: AppDialogField(
        controller: nameCtrl,
        label: 'Name',
        hint: 'Enter database name',
        autofocus: true,
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (nameCtrl.text.trim().isEmpty) return;
            await ref.read(apiClientProvider).post('/databases',
                data: {'name': nameCtrl.text.trim()});
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
            ref.invalidate(databasesProvider);
          },
        ),
      ],
    );
  }

  Future<void> _deleteDb(String id) async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete database',
      content: Text(
          'All tables and data in this database will be permanently deleted.',
          style: TextStyle(color: Colors.white.withOpacity(0.6))),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Delete',
          destructive: true,
          onTap: () => Navigator.of(context, rootNavigator: true).pop(true),
        ),
      ],
    );
    if (confirmed == true) {
      await ref.read(apiClientProvider).delete('/databases/$id');
      ref.invalidate(databasesProvider);
    }
  }
}

// =============================================================================
// Level 2: Database Detail (Tables, Usage, Settings)
// =============================================================================

class _DatabaseDetailView extends ConsumerStatefulWidget {
  final String dbId;
  final VoidCallback onBack;
  final ValueChanged<String> onTableSelect;

  const _DatabaseDetailView({
    required this.dbId,
    required this.onBack,
    required this.onTableSelect,
  });

  @override
  ConsumerState<_DatabaseDetailView> createState() =>
      _DatabaseDetailViewState();
}

class _DatabaseDetailViewState extends ConsumerState<_DatabaseDetailView> {
  int _tabIndex = 0;

  @override
  Widget build(BuildContext context) {
    final tablesAsync = ref.watch(_tablesProvider(widget.dbId));
    // Try to get database name from the tables response or just use ID
    final dbName = widget.dbId;

    return Scaffold(
      backgroundColor: _bgColor,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: MediaQuery.of(context).size.width > 1400 ? 80 : 40,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            _BackHeader(
              title: dbName,
              subtitle: widget.dbId,
              icon: LucideIcons.database,
              onBack: widget.onBack,
            ),
            const SizedBox(height: 24),
            PageTabs(
              tabs: const ['Tables', 'Usage', 'Settings'],
              selected: _tabIndex,
              onChanged: (i) => setState(() => _tabIndex = i),
            ),
            const SizedBox(height: 20),
            Expanded(
              child: _tabIndex == 0
                  ? _buildTablesTab(tablesAsync)
                  : _tabIndex == 1
                      ? const _PlaceholderTab(label: 'Usage')
                      : _buildSettingsTab(),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTablesTab(AsyncValue<Map<String, dynamic>> tablesAsync) {
    return Column(
      children: [
        Row(
          children: [
            _SearchBox(
                controller: TextEditingController(),
                hint: 'Search by name or ID'),
            const Spacer(),
            _AccentButton(
              label: 'Create table',
              onTap: _showCreateTableDialog,
            ),
          ],
        ),
        const SizedBox(height: 16),
        Expanded(
          child: tablesAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
                child: Text('Error: $e',
                    style: const TextStyle(color: _red))),
            data: (data) {
              final tables = List<Map<String, dynamic>>.from(
                  data['tables'] ?? []);
              if (tables.isEmpty) {
                return _EmptyState(
                  icon: LucideIcons.table2,
                  title: 'No tables yet',
                  subtitle:
                      'Create, organize, and query structured data with Tables.',
                  actionLabel: 'Create table',
                  onAction: _showCreateTableDialog,
                );
              }
              return _DataTable(
                headers: const ['Table ID', 'Name', 'Created'],
                flexes: const [3, 3, 2],
                rows: tables,
                rowBuilder: (t) {
                  final id = t['\$id'] as String? ?? '';
                  final name = t['name'] as String? ?? '';
                  final created =
                      _fmtDate(t['createdAt'] ?? t['\$createdAt']);
                  return [
                    Row(children: [
                      Icon(LucideIcons.table2,
                          size: 14, color: _accent),
                      const SizedBox(width: 8),
                      Expanded(
                          child: Text(id,
                              style: const TextStyle(
                                  color: Colors.white,
                                  fontSize: 13,
                                  fontFamily: 'monospace'),
                              overflow: TextOverflow.ellipsis)),
                    ]),
                    Text(name,
                        style: const TextStyle(
                            color: Colors.white, fontSize: 13)),
                    Text(created,
                        style: const TextStyle(
                            color: _dimText, fontSize: 12)),
                  ];
                },
                onRowTap: (t) =>
                    widget.onTableSelect(t['\$id'] as String),
                onRowDelete: (t) async {
                  await ref.read(apiClientProvider).delete(
                      '/databases/${widget.dbId}/tables/${t['\$id']}');
                  ref.invalidate(_tablesProvider(widget.dbId));
                },
              );
            },
          ),
        ),
      ],
    );
  }

  Widget _buildSettingsTab() {
    return SingleChildScrollView(
      child: Column(
        children: [
          _SettingsCard(
            title: 'Database settings',
            children: [
              _InfoRow(label: 'Database ID', value: widget.dbId),
            ],
          ),
          const SizedBox(height: 16),
          _DangerCard(
            title: 'Delete database',
            description:
                'All tables and data will be permanently deleted.',
            onDelete: () async {
              await ref
                  .read(apiClientProvider)
                  .delete('/databases/${widget.dbId}');
              ref.invalidate(databasesProvider);
              widget.onBack();
            },
          ),
        ],
      ),
    );
  }

  void _showCreateTableDialog() {
    final nameCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create table',
      content: AppDialogField(
        controller: nameCtrl,
        label: 'Name',
        hint: 'Enter table name',
        autofocus: true,
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (nameCtrl.text.trim().isEmpty) return;
            await ref.read(apiClientProvider).post(
                '/databases/${widget.dbId}/tables',
                data: {'name': nameCtrl.text.trim()});
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
            ref.invalidate(_tablesProvider(widget.dbId));
          },
        ),
      ],
    );
  }
}

// =============================================================================
// Level 3: Table Detail (Rows, Columns, Indexes, Relationships, Settings)
// =============================================================================

class _TableDetailView extends ConsumerStatefulWidget {
  final String dbId;
  final String tableId;
  final VoidCallback onBack;

  const _TableDetailView({
    required this.dbId,
    required this.tableId,
    required this.onBack,
  });

  @override
  ConsumerState<_TableDetailView> createState() =>
      _TableDetailViewState();
}

class _TableDetailViewState extends ConsumerState<_TableDetailView> {
  int _tabIndex = 0;
  bool _showCreateColumn = false;

  ({String dbId, String tableId}) get _key =>
      (dbId: widget.dbId, tableId: widget.tableId);

  String get _basePath =>
      '/databases/${widget.dbId}/tables/${widget.tableId}';

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: _bgColor,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: MediaQuery.of(context).size.width > 1400 ? 80 : 40,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            _BackHeader(
              title: widget.tableId,
              subtitle: widget.tableId,
              icon: LucideIcons.table2,
              onBack: widget.onBack,
            ),
            const SizedBox(height: 24),
            PageTabs(
              tabs: const [
                'Rows',
                'Columns',
                'Indexes',
                'Relationships',
                'Settings',
              ],
              selected: _tabIndex,
              onChanged: (i) => setState(() {
                _tabIndex = i;
                _showCreateColumn = false;
              }),
            ),
            const SizedBox(height: 20),
            Expanded(
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(child: _buildCurrentTab()),
                  if (_showCreateColumn)
                    _CreateColumnPanel(
                      basePath: _basePath,
                      onClose: () =>
                          setState(() => _showCreateColumn = false),
                      onCreated: () {
                        ref.invalidate(_columnsProvider(_key));
                        setState(() => _showCreateColumn = false);
                      },
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildCurrentTab() {
    switch (_tabIndex) {
      case 0:
        return _buildRowsTab();
      case 1:
        return _buildColumnsTab();
      case 2:
        return _buildIndexesTab();
      case 3:
        return _buildRelationshipsTab();
      case 4:
        return _buildTableSettingsTab();
      default:
        return const SizedBox();
    }
  }

  // --- Rows Tab ---
  Widget _buildRowsTab() {
    final rowsAsync = ref.watch(_rowsProvider(_key));
    final columnsAsync = ref.watch(_columnsProvider(_key));
    final columns = columnsAsync.valueOrNull ?? [];

    return Column(
      children: [
        Row(
          children: [
            _SearchBox(
                controller: TextEditingController(),
                hint: 'Search rows'),
            const Spacer(),
            _GhostButton(
              label: 'Import CSV',
              icon: LucideIcons.upload,
              onTap: () {},
            ),
            const SizedBox(width: 8),
            _AccentButton(
              label: 'Create row',
              onTap: () => _showCreateRowDialog(columns),
            ),
          ],
        ),
        const SizedBox(height: 16),
        Expanded(
          child: rowsAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
                child: Text('Error: $e',
                    style: const TextStyle(color: _red))),
            data: (data) {
              final rows = List<Map<String, dynamic>>.from(
                  data['documents'] ?? data['rows'] ?? []);
              if (rows.isEmpty && columns.isEmpty) {
                return _EmptyState(
                  icon: LucideIcons.columns,
                  title: 'You have no columns yet',
                  subtitle: 'Create columns to define your data schema.',
                  actionLabel: 'Create column',
                  onAction: () => setState(() {
                    _tabIndex = 1;
                    _showCreateColumn = true;
                  }),
                );
              }
              if (rows.isEmpty) {
                return _EmptyState(
                  icon: LucideIcons.alignJustify,
                  title: 'No rows yet',
                  subtitle: 'Create a row or import data from CSV.',
                  actionLabel: 'Create row',
                  onAction: () => _showCreateRowDialog(columns),
                );
              }
              // Build dynamic headers from columns
              final colKeys = columns
                  .map((c) => c['key'] as String? ?? '')
                  .where((k) => k.isNotEmpty)
                  .toList();
              final displayKeys = ['\$id', ...colKeys];

              return _RowsGrid(
                displayKeys: displayKeys,
                columns: columns,
                rows: rows,
                onDelete: (rowId) async {
                  await ref
                      .read(apiClientProvider)
                      .delete('$_basePath/rows/$rowId');
                  ref.invalidate(_rowsProvider(_key));
                },
              );
            },
          ),
        ),
      ],
    );
  }

  // --- Columns Tab ---
  Widget _buildColumnsTab() {
    final columnsAsync = ref.watch(_columnsProvider(_key));

    return Column(
      children: [
        Row(
          children: [
            const Spacer(),
            _AccentButton(
              label: 'Create column',
              onTap: () => setState(() => _showCreateColumn = true),
            ),
          ],
        ),
        const SizedBox(height: 16),
        Expanded(
          child: columnsAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(child: Text('Error: $e')),
            data: (columns) {
              if (columns.isEmpty) {
                return _EmptyState(
                  icon: LucideIcons.columns,
                  title: 'No columns yet',
                  subtitle: 'Define your data schema by creating columns.',
                  actionLabel: 'Create column',
                  onAction: () =>
                      setState(() => _showCreateColumn = true),
                );
              }
              return _DataTable(
                headers: const [
                  'Column name',
                  'Type',
                  'Required',
                  'Default value'
                ],
                flexes: const [3, 2, 1, 2],
                rows: columns,
                rowBuilder: (col) {
                  final key = col['key'] as String? ?? '';
                  final type = col['type'] as String? ?? '';
                  final required = col['required'] == true;
                  final def = col['default']?.toString() ?? '-';
                  return [
                    Row(children: [
                      _ColumnTypeIcon(type: type),
                      const SizedBox(width: 8),
                      Text(key,
                          style: const TextStyle(
                              color: Colors.white, fontSize: 13)),
                      if (required) ...[
                        const SizedBox(width: 6),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 5, vertical: 1),
                          decoration: BoxDecoration(
                            color: _accent.withOpacity(0.15),
                            borderRadius: BorderRadius.circular(3),
                          ),
                          child: const Text('required',
                              style: TextStyle(
                                  color: _accent,
                                  fontSize: 10,
                                  fontWeight: FontWeight.w500)),
                        ),
                      ],
                    ]),
                    Text(type,
                        style: const TextStyle(
                            color: _dimText, fontSize: 13)),
                    Icon(
                      required
                          ? LucideIcons.checkSquare
                          : LucideIcons.square,
                      size: 14,
                      color: required ? _accent : _subtleText,
                    ),
                    Text(def,
                        style: const TextStyle(
                            color: _dimText, fontSize: 13)),
                  ];
                },
                onRowDelete: (col) async {
                  await ref.read(apiClientProvider).delete(
                      '$_basePath/columns/${col['key']}');
                  ref.invalidate(_columnsProvider(_key));
                },
              );
            },
          ),
        ),
      ],
    );
  }

  // --- Indexes Tab ---
  Widget _buildIndexesTab() {
    final indexesAsync = ref.watch(_indexesProvider(_key));

    return Column(
      children: [
        Row(
          children: [
            const Spacer(),
            _AccentButton(
              label: 'Create index',
              onTap: _showCreateIndexDialog,
            ),
          ],
        ),
        const SizedBox(height: 16),
        Expanded(
          child: indexesAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(child: Text('Error: $e')),
            data: (indexes) {
              if (indexes.isEmpty) {
                return _EmptyState(
                  icon: LucideIcons.listOrdered,
                  title: 'No indexes',
                  subtitle: 'Create indexes to optimize query performance.',
                  actionLabel: 'Create index',
                  onAction: _showCreateIndexDialog,
                );
              }
              return _DataTable(
                headers: const ['Key', 'Type', 'Columns', 'Orders'],
                flexes: const [2, 2, 3, 2],
                rows: indexes,
                rowBuilder: (idx) {
                  return [
                    Text(idx['key'] as String? ?? '',
                        style: const TextStyle(
                            color: Colors.white, fontSize: 13)),
                    Text(idx['type'] as String? ?? '',
                        style: const TextStyle(
                            color: _dimText, fontSize: 13)),
                    Text(
                        (idx['attributes'] as List?)?.join(', ') ??
                            '',
                        style: const TextStyle(
                            color: _dimText, fontSize: 12)),
                    Text(
                        (idx['orders'] as List?)?.join(', ') ?? '',
                        style: const TextStyle(
                            color: _dimText, fontSize: 12)),
                  ];
                },
                onRowDelete: (idx) async {
                  await ref.read(apiClientProvider).delete(
                      '$_basePath/indexes/${idx['key']}');
                  ref.invalidate(_indexesProvider(_key));
                },
              );
            },
          ),
        ),
      ],
    );
  }

  // --- Relationships Tab ---
  Widget _buildRelationshipsTab() {
    final relsAsync = ref.watch(_relationshipsProvider(_key));

    return Column(
      children: [
        Row(
          children: [
            const Spacer(),
            _AccentButton(
              label: 'Create relationship',
              onTap: _showCreateRelDialog,
            ),
          ],
        ),
        const SizedBox(height: 16),
        Expanded(
          child: relsAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(child: Text('Error: $e')),
            data: (rels) {
              if (rels.isEmpty) {
                return _EmptyState(
                  icon: LucideIcons.link,
                  title: 'No relationships',
                  subtitle:
                      'Define relationships between tables.',
                  actionLabel: 'Create relationship',
                  onAction: _showCreateRelDialog,
                );
              }
              return _DataTable(
                headers: const [
                  'Key',
                  'Type',
                  'Related Table',
                  'On Delete'
                ],
                flexes: const [2, 2, 2, 2],
                rows: rels,
                rowBuilder: (r) {
                  return [
                    Text(r['key'] as String? ?? '',
                        style: const TextStyle(
                            color: Colors.white, fontSize: 13)),
                    Text(r['type'] as String? ?? '',
                        style: const TextStyle(
                            color: _dimText, fontSize: 13)),
                    Text(r['relatedTableId'] as String? ?? '',
                        style: const TextStyle(
                            color: _dimText, fontSize: 13)),
                    Text(r['onDelete'] as String? ?? 'setNull',
                        style: const TextStyle(
                            color: _dimText, fontSize: 13)),
                  ];
                },
                onRowDelete: (r) async {
                  await ref.read(apiClientProvider).delete(
                      '$_basePath/relationships/${r['\$id'] ?? r['key']}');
                  ref.invalidate(_relationshipsProvider(_key));
                },
              );
            },
          ),
        ),
      ],
    );
  }

  // --- Settings Tab ---
  Widget _buildTableSettingsTab() {
    return SingleChildScrollView(
      child: Column(
        children: [
          // 1. Status + enabled toggle
          _SettingsSectionCard(
            children: [
              Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(widget.tableId,
                            style: const TextStyle(
                                color: Colors.white,
                                fontSize: 15,
                                fontWeight: FontWeight.w600)),
                        const SizedBox(height: 4),
                        Text('Table ID: ${widget.tableId}',
                            style: const TextStyle(
                                color: _subtleText, fontSize: 12)),
                      ],
                    ),
                  ),
                  Row(
                    children: [
                      Switch(
                        value: true,
                        onChanged: (_) {},
                        activeColor: _accent,
                      ),
                      const SizedBox(width: 4),
                      const Text('Enabled',
                          style: TextStyle(
                              color: Colors.white, fontSize: 13)),
                    ],
                  ),
                ],
              ),
            ],
            onUpdate: () {},
          ),

          // 2. Name
          _SettingsSectionCard(
            title: 'Name',
            children: [
              _SettingsTextField(
                initialValue: widget.tableId,
                onSaved: (v) async {
                  await ref.read(apiClientProvider).put(_basePath,
                      data: {'name': v});
                  ref.invalidate(_tablesProvider(widget.dbId));
                },
              ),
            ],
          ),

          // 3. Display name
          _SettingsSectionCard(
            title: 'Display name',
            subtitle:
                'Select up to 3 string columns to display as row names in the console. These help identify rows in places like relationships.',
            children: [
              _InfoRow(label: 'Row ID', value: '\$id'),
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: TextButton.icon(
                  onPressed: () {},
                  icon: const Icon(LucideIcons.plus, size: 14),
                  label: const Text('Add column',
                      style: TextStyle(fontSize: 12)),
                  style:
                      TextButton.styleFrom(foregroundColor: _accent),
                ),
              ),
            ],
            onUpdate: () {},
          ),

          // 4. Permissions
          _SettingsSectionCard(
            title: 'Permissions',
            subtitle:
                'Choose who can access your tables and rows.',
            children: [
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: Colors.white.withOpacity(0.02),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _border),
                ),
                child: Column(
                  children: [
                    Icon(LucideIcons.plus,
                        size: 20,
                        color: Colors.white.withOpacity(0.3)),
                    const SizedBox(height: 8),
                    Text('Add a role to get started',
                        style: TextStyle(
                            color: Colors.white.withOpacity(0.4),
                            fontSize: 13)),
                  ],
                ),
              ),
            ],
            onUpdate: () {},
          ),

          // 5. Row security
          _SettingsSectionCard(
            title: 'Row security',
            children: [
              Row(
                children: [
                  Switch(
                    value: false,
                    onChanged: (_) {},
                    activeColor: _accent,
                  ),
                  const SizedBox(width: 8),
                  const Text('Row security',
                      style: TextStyle(
                          color: Colors.white, fontSize: 13)),
                ],
              ),
              const SizedBox(height: 8),
              Text(
                'When row security is enabled, users will be able to access rows for which they have been granted either row or table permissions.\n\n'
                'If row security is disabled, users can access rows only if they have table permissions. Row permissions will be ignored.',
                style: TextStyle(
                    color: Colors.white.withOpacity(0.35),
                    fontSize: 12,
                    height: 1.5),
              ),
            ],
            onUpdate: () {},
          ),

          // 6. Delete table
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(20),
            margin: const EdgeInsets.only(bottom: 40),
            decoration: BoxDecoration(
              color: _cardColor,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: _red.withOpacity(0.3)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text('Delete table',
                              style: TextStyle(
                                  color: _red,
                                  fontSize: 14,
                                  fontWeight: FontWeight.w500)),
                          const SizedBox(height: 4),
                          Text(
                            'The table will be permanently deleted, including all the rows within it. This action is irreversible.',
                            style: TextStyle(
                                color: Colors.white.withOpacity(0.4),
                                fontSize: 13),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 16),
                    Container(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: Colors.white.withOpacity(0.03),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(color: _border),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(widget.tableId,
                              style: const TextStyle(
                                  color: Colors.white,
                                  fontSize: 13,
                                  fontWeight: FontWeight.w500)),
                        ],
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Align(
                  alignment: Alignment.centerRight,
                  child: OutlinedButton(
                    style: OutlinedButton.styleFrom(
                      foregroundColor: _red,
                      side: const BorderSide(color: _red),
                      padding: const EdgeInsets.symmetric(
                          horizontal: 20, vertical: 10),
                      shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(8)),
                    ),
                    onPressed: () async {
                      await ref
                          .read(apiClientProvider)
                          .delete(_basePath);
                      ref.invalidate(_tablesProvider(widget.dbId));
                      widget.onBack();
                    },
                    child: const Text('Delete',
                        style: TextStyle(fontSize: 13)),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  // --- Dialogs ---

  void _showCreateRowDialog(List<Map<String, dynamic>> columns) {
    final controllers = <String, TextEditingController>{};
    for (final col in columns) {
      final key = col['key'] as String? ?? '';
      if (key.isNotEmpty) controllers[key] = TextEditingController();
    }

    showAppDialog(
      context: context,
      title: 'Create row',
      width: 520,
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (controllers.isEmpty)
            AppDialogField(
              controller: TextEditingController(),
              label: 'Data (JSON)',
              hint: '{"key": "value"}',
              maxLines: 5,
            )
          else
            ...controllers.entries.map((e) => Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: AppDialogField(
                    controller: e.value,
                    label: e.key,
                    hint: 'Enter ${e.key}',
                  ),
                )),
        ],
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            Map<String, dynamic> data;
            if (controllers.isEmpty) {
              data = {};
            } else {
              data = {};
              for (final e in controllers.entries) {
                if (e.value.text.trim().isNotEmpty) {
                  data[e.key] = e.value.text.trim();
                }
              }
            }
            await ref
                .read(apiClientProvider)
                .post('$_basePath/rows', data: {'data': data});
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
            ref.invalidate(_rowsProvider(_key));
          },
        ),
      ],
    );
  }

  void _showCreateIndexDialog() {
    final keyCtrl = TextEditingController();
    final columnsCtrl = TextEditingController();
    String indexType = 'key';

    showAppDialog(
      context: context,
      title: 'Create index',
      content: StatefulBuilder(
        builder: (ctx, setDialogState) => Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AppDialogField(
                controller: keyCtrl,
                label: 'Key',
                hint: 'index_name',
                autofocus: true),
            const SizedBox(height: 12),
            AppDialogField(
                controller: columnsCtrl,
                label: 'Columns (comma separated)',
                hint: 'column1, column2'),
            const SizedBox(height: 12),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Type',
                    style: TextStyle(
                        color: Colors.white.withOpacity(0.5),
                        fontSize: 12,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  children: ['key', 'unique', 'fulltext'].map((t) {
                    final sel = indexType == t;
                    return GestureDetector(
                      onTap: () => setDialogState(() => indexType = t),
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 6),
                        decoration: BoxDecoration(
                          color: sel
                              ? _accent.withOpacity(0.15)
                              : const Color(0x0AFFFFFF),
                          borderRadius: BorderRadius.circular(6),
                          border: Border.all(
                              color: sel
                                  ? _accent
                                  : Colors.white.withOpacity(0.1)),
                        ),
                        child: Text(t,
                            style: TextStyle(
                                color: sel ? Colors.white : _dimText,
                                fontSize: 12)),
                      ),
                    );
                  }).toList(),
                ),
              ],
            ),
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (keyCtrl.text.trim().isEmpty) return;
            final cols = columnsCtrl.text
                .split(',')
                .map((s) => s.trim())
                .where((s) => s.isNotEmpty)
                .toList();
            await ref.read(apiClientProvider).post(
                '$_basePath/indexes',
                data: {
                  'key': keyCtrl.text.trim(),
                  'type': indexType,
                  'attributes': cols,
                  'orders': cols.map((_) => 'ASC').toList(),
                });
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
            ref.invalidate(_indexesProvider(_key));
          },
        ),
      ],
    );
  }

  void _showCreateRelDialog() {
    final keyCtrl = TextEditingController();
    final relTableCtrl = TextEditingController();
    String relType = 'oneToMany';

    showAppDialog(
      context: context,
      title: 'Create relationship',
      content: StatefulBuilder(
        builder: (ctx, setDialogState) => Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AppDialogField(
                controller: keyCtrl,
                label: 'Key',
                hint: 'relationship_name',
                autofocus: true),
            const SizedBox(height: 12),
            AppDialogField(
                controller: relTableCtrl,
                label: 'Related table ID',
                hint: 'table_id'),
            const SizedBox(height: 12),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Type',
                    style: TextStyle(
                        color: Colors.white.withOpacity(0.5),
                        fontSize: 12,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    'oneToOne',
                    'oneToMany',
                    'manyToOne',
                    'manyToMany'
                  ].map((t) {
                    final sel = relType == t;
                    return GestureDetector(
                      onTap: () => setDialogState(() => relType = t),
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 6),
                        decoration: BoxDecoration(
                          color: sel
                              ? _accent.withOpacity(0.15)
                              : const Color(0x0AFFFFFF),
                          borderRadius: BorderRadius.circular(6),
                          border: Border.all(
                              color: sel
                                  ? _accent
                                  : Colors.white.withOpacity(0.1)),
                        ),
                        child: Text(t,
                            style: TextStyle(
                                color: sel ? Colors.white : _dimText,
                                fontSize: 12)),
                      ),
                    );
                  }).toList(),
                ),
              ],
            ),
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (keyCtrl.text.trim().isEmpty ||
                relTableCtrl.text.trim().isEmpty) return;
            await ref.read(apiClientProvider).post(
                '$_basePath/columns/relationship',
                data: {
                  'key': keyCtrl.text.trim(),
                  'relatedTableId': relTableCtrl.text.trim(),
                  'type': relType,
                });
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
            ref.invalidate(_relationshipsProvider(_key));
          },
        ),
      ],
    );
  }
}

// =============================================================================
// Create Column Side Panel (like Appwrite)
// =============================================================================

class _CreateColumnPanel extends ConsumerStatefulWidget {
  final String basePath;
  final VoidCallback onClose;
  final VoidCallback onCreated;

  const _CreateColumnPanel({
    required this.basePath,
    required this.onClose,
    required this.onCreated,
  });

  @override
  ConsumerState<_CreateColumnPanel> createState() =>
      _CreateColumnPanelState();
}

class _CreateColumnPanelState extends ConsumerState<_CreateColumnPanel> {
  final _keyCtrl = TextEditingController();
  final _sizeCtrl = TextEditingController(text: '256');
  final _defaultCtrl = TextEditingController();
  final _elementsCtrl = TextEditingController();
  String _type = 'string';
  bool _required = false;
  bool _array = false;
  bool _creating = false;

  static const _types = [
    ('string', 'String', LucideIcons.type),
    ('integer', 'Integer', LucideIcons.hash),
    ('float', 'Float', LucideIcons.hash),
    ('boolean', 'Boolean', LucideIcons.toggleLeft),
    ('datetime', 'Datetime', LucideIcons.calendar),
    ('email', 'Email', LucideIcons.mail),
    ('url', 'URL', LucideIcons.link),
    ('enum', 'Enum', LucideIcons.list),
    ('point', 'Point', LucideIcons.mapPin),
    ('relationship', 'Relationship', LucideIcons.link2),
  ];

  @override
  void dispose() {
    _keyCtrl.dispose();
    _sizeCtrl.dispose();
    _defaultCtrl.dispose();
    _elementsCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 320,
      margin: const EdgeInsets.only(left: 16),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 0),
            child: Row(
              children: [
                const Text('Create column',
                    style: TextStyle(
                        color: Colors.white,
                        fontSize: 14,
                        fontWeight: FontWeight.w600)),
                const Spacer(),
                GestureDetector(
                  onTap: widget.onClose,
                  child: MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: Icon(LucideIcons.x,
                        size: 16,
                        color: Colors.white.withOpacity(0.3)),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          Expanded(
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Key
                  _PanelField(controller: _keyCtrl, label: 'Key', hint: 'Enter key'),
                  const SizedBox(height: 12),
                  // Type
                  _PanelLabel('Type'),
                  const SizedBox(height: 6),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 12),
                    decoration: BoxDecoration(
                      color: const Color(0x0AFFFFFF),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: const Color(0x1AFFFFFF)),
                    ),
                    child: DropdownButtonHideUnderline(
                      child: DropdownButton<String>(
                        value: _type,
                        isExpanded: true,
                        dropdownColor: const Color(0xFF1E1F24),
                        style: const TextStyle(
                            color: Colors.white, fontSize: 13),
                        icon: const Icon(LucideIcons.chevronDown,
                            size: 14, color: _dimText),
                        items: _types
                            .map((t) => DropdownMenuItem(
                                  value: t.$1,
                                  child: Row(
                                    children: [
                                      Icon(t.$3,
                                          size: 14,
                                          color: _accent),
                                      const SizedBox(width: 8),
                                      Text(t.$2),
                                    ],
                                  ),
                                ))
                            .toList(),
                        onChanged: (v) {
                          if (v != null) setState(() => _type = v);
                        },
                      ),
                    ),
                  ),
                  const SizedBox(height: 12),
                  // Size (for string)
                  if (_type == 'string') ...[
                    _PanelField(
                        controller: _sizeCtrl,
                        label: 'Size',
                        hint: '256'),
                    const SizedBox(height: 12),
                  ],
                  // Elements (for enum)
                  if (_type == 'enum') ...[
                    _PanelField(
                        controller: _elementsCtrl,
                        label: 'Elements (comma separated)',
                        hint: 'value1, value2, value3'),
                    const SizedBox(height: 12),
                  ],
                  // Default
                  _PanelField(
                      controller: _defaultCtrl,
                      label: 'Default (optional)',
                      hint: 'Enter default value'),
                  const SizedBox(height: 16),
                  // Options
                  _PanelToggle(
                    label: 'Required',
                    subtitle: 'Indicate whether this column is required.',
                    value: _required,
                    onChanged: (v) => setState(() => _required = v),
                  ),
                  const SizedBox(height: 8),
                  _PanelToggle(
                    label: 'Array',
                    subtitle: 'Indicate whether this column is an array.',
                    value: _array,
                    onChanged: (v) => setState(() => _array = v),
                  ),
                ],
              ),
            ),
          ),
          // Actions
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              border: Border(
                  top: BorderSide(color: Colors.white.withOpacity(0.06))),
            ),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                  onPressed: widget.onClose,
                  style: TextButton.styleFrom(
                      foregroundColor: Colors.white54),
                  child:
                      const Text('Cancel', style: TextStyle(fontSize: 13)),
                ),
                const SizedBox(width: 8),
                FilledButton(
                  style: FilledButton.styleFrom(
                    backgroundColor: _accent,
                    padding: const EdgeInsets.symmetric(
                        horizontal: 20, vertical: 10),
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                  ),
                  onPressed: _creating ? null : _create,
                  child: _creating
                      ? const SizedBox(
                          width: 14,
                          height: 14,
                          child: CircularProgressIndicator(
                              strokeWidth: 2, color: Colors.white))
                      : const Text('Create',
                          style: TextStyle(fontSize: 13)),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _create() async {
    if (_keyCtrl.text.trim().isEmpty) return;
    setState(() => _creating = true);
    try {
      final api = ref.read(apiClientProvider);
      final data = <String, dynamic>{
        'key': _keyCtrl.text.trim(),
        'required': _required,
        'array': _array,
      };
      if (_defaultCtrl.text.trim().isNotEmpty) {
        data['default'] = _defaultCtrl.text.trim();
      }
      if (_type == 'string') {
        data['size'] = int.tryParse(_sizeCtrl.text) ?? 256;
      }
      if (_type == 'enum') {
        data['elements'] = _elementsCtrl.text
            .split(',')
            .map((s) => s.trim())
            .where((s) => s.isNotEmpty)
            .toList();
      }
      await api.post('${widget.basePath}/columns/$_type', data: data);
      widget.onCreated();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('Error: $e')));
      }
    }
    if (mounted) setState(() => _creating = false);
  }
}

// =============================================================================
// Rows Grid (spreadsheet-like)
// =============================================================================

class _RowsGrid extends StatelessWidget {
  final List<String> displayKeys;
  final List<Map<String, dynamic>> columns;
  final List<Map<String, dynamic>> rows;
  final Future<void> Function(String) onDelete;

  const _RowsGrid({
    required this.displayKeys,
    required this.columns,
    required this.rows,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        // Header
        Container(
          padding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            border: Border(
                bottom:
                    BorderSide(color: Colors.white.withOpacity(0.06))),
          ),
          child: Row(
            children: [
              ...displayKeys.map((k) => Expanded(
                    child: Row(
                      children: [
                        if (k == '\$id')
                          Icon(LucideIcons.key,
                              size: 12, color: _subtleText)
                        else
                          _ColumnTypeIcon(
                              type: _typeForKey(k)),
                        const SizedBox(width: 6),
                        Expanded(
                          child: Text(k,
                              style: const TextStyle(
                                  color: _dimText,
                                  fontSize: 12,
                                  fontWeight: FontWeight.w500),
                              overflow: TextOverflow.ellipsis),
                        ),
                      ],
                    ),
                  )),
              const SizedBox(width: 40),
            ],
          ),
        ),
        // Rows
        Expanded(
          child: ListView.builder(
            itemCount: rows.length,
            itemBuilder: (_, i) {
              final row = rows[i];
              final data =
                  row['data'] as Map<String, dynamic>? ?? row;
              final rowId = row['\$id'] as String? ?? '';

              return _HoverRow(
                child: Row(
                  children: [
                    ...displayKeys.map((k) {
                      String val;
                      if (k == '\$id') {
                        val = rowId;
                      } else {
                        val = data[k]?.toString() ?? '-';
                      }
                      return Expanded(
                        child: Padding(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 4),
                          child: Text(val,
                              style: const TextStyle(
                                  color: Colors.white,
                                  fontSize: 12,
                                  fontFamily: 'monospace'),
                              overflow: TextOverflow.ellipsis),
                        ),
                      );
                    }),
                    SizedBox(
                      width: 40,
                      child: PopupMenuButton<String>(
                        color: const Color(0xFF1A1A22),
                        shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(8)),
                        iconSize: 16,
                        icon: Icon(LucideIcons.moreHorizontal,
                            size: 14, color: _subtleText),
                        onSelected: (v) {
                          if (v == 'delete') onDelete(rowId);
                        },
                        itemBuilder: (_) => [
                          const PopupMenuItem(
                            value: 'delete',
                            child: Text('Delete',
                                style: TextStyle(
                                    color: _red, fontSize: 13)),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              );
            },
          ),
        ),
      ],
    );
  }

  String _typeForKey(String key) {
    for (final col in columns) {
      if (col['key'] == key) return col['type'] as String? ?? '';
    }
    return '';
  }
}

// =============================================================================
// Shared Widgets
// =============================================================================

class _BackHeader extends StatelessWidget {
  final String title;
  final String subtitle;
  final IconData icon;
  final VoidCallback onBack;

  const _BackHeader({
    required this.title,
    required this.subtitle,
    required this.icon,
    required this.onBack,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        GestureDetector(
          onTap: onBack,
          child: MouseRegion(
            cursor: SystemMouseCursors.click,
            child: Icon(LucideIcons.arrowLeft,
                size: 20, color: Colors.white.withOpacity(0.5)),
          ),
        ),
        const SizedBox(width: 12),
        Text(title,
            style: const TextStyle(
                color: Colors.white,
                fontSize: 22,
                fontWeight: FontWeight.w600)),
        const SizedBox(width: 12),
        Icon(icon, size: 13, color: Colors.white.withOpacity(0.3)),
        const SizedBox(width: 4),
        Text(
            subtitle.length > 16
                ? '${subtitle.substring(0, 16)}...'
                : subtitle,
            style: TextStyle(
                color: Colors.white.withOpacity(0.3),
                fontSize: 13,
                fontFamily: 'monospace')),
      ],
    );
  }
}

class _SearchBox extends StatelessWidget {
  final TextEditingController controller;
  final String hint;
  final ValueChanged<String>? onChanged;

  const _SearchBox({
    required this.controller,
    required this.hint,
    this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 280,
      child: TextField(
        controller: controller,
        onChanged: onChanged,
        style: const TextStyle(fontSize: 13, color: Colors.white),
        decoration: InputDecoration(
          hintText: hint,
          hintStyle: const TextStyle(color: _subtleText, fontSize: 13),
          prefixIcon: const Padding(
            padding: EdgeInsets.only(left: 10, right: 6),
            child: Icon(Icons.search, size: 16, color: _subtleText),
          ),
          prefixIconConstraints:
              const BoxConstraints(minWidth: 32, minHeight: 0),
          filled: true,
          fillColor: const Color(0x0AFFFFFF),
          isDense: true,
          contentPadding:
              const EdgeInsets.symmetric(vertical: 10, horizontal: 12),
          border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide:
                  BorderSide(color: Colors.white.withOpacity(0.08))),
          enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide:
                  BorderSide(color: Colors.white.withOpacity(0.08))),
          focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: _accent)),
        ),
      ),
    );
  }
}

class _AccentButton extends StatelessWidget {
  final String label;
  final VoidCallback onTap;

  const _AccentButton({required this.label, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return FilledButton.icon(
      style: FilledButton.styleFrom(
        backgroundColor: _accent,
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        shape:
            RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
      icon: const Icon(LucideIcons.plus, size: 14),
      label: Text(label, style: const TextStyle(fontSize: 12)),
      onPressed: onTap,
    );
  }
}

class _GhostButton extends StatelessWidget {
  final String label;
  final IconData icon;
  final VoidCallback onTap;

  const _GhostButton(
      {required this.label, required this.icon, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return OutlinedButton.icon(
      style: OutlinedButton.styleFrom(
        foregroundColor: Colors.white70,
        side: BorderSide(color: Colors.white.withOpacity(0.1)),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        shape:
            RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
      icon: Icon(icon, size: 14),
      label: Text(label, style: const TextStyle(fontSize: 12)),
      onPressed: onTap,
    );
  }
}

class _DataTable extends StatelessWidget {
  final List<String> headers;
  final List<int> flexes;
  final List<Map<String, dynamic>> rows;
  final List<Widget> Function(Map<String, dynamic>) rowBuilder;
  final ValueChanged<Map<String, dynamic>>? onRowTap;
  final Future<void> Function(Map<String, dynamic>)? onRowDelete;

  const _DataTable({
    required this.headers,
    required this.flexes,
    required this.rows,
    required this.rowBuilder,
    this.onRowTap,
    this.onRowDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Container(
          padding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            border: Border(
                bottom:
                    BorderSide(color: Colors.white.withOpacity(0.06))),
          ),
          child: Row(
            children: [
              for (var i = 0; i < headers.length; i++)
                Expanded(
                  flex: flexes[i],
                  child: Text(headers[i],
                      style: const TextStyle(
                          color: _dimText,
                          fontSize: 12,
                          fontWeight: FontWeight.w500)),
                ),
              if (onRowDelete != null) const SizedBox(width: 40),
            ],
          ),
        ),
        Expanded(
          child: ListView.builder(
            itemCount: rows.length,
            itemBuilder: (_, i) {
              final row = rows[i];
              final cells = rowBuilder(row);
              return _HoverRow(
                onTap: onRowTap != null ? () => onRowTap!(row) : null,
                child: Row(
                  children: [
                    for (var j = 0; j < cells.length; j++)
                      Expanded(flex: flexes[j], child: cells[j]),
                    if (onRowDelete != null)
                      SizedBox(
                        width: 40,
                        child: _DeleteIcon(
                            onTap: () => onRowDelete!(row)),
                      ),
                  ],
                ),
              );
            },
          ),
        ),
      ],
    );
  }
}

class _HoverRow extends StatefulWidget {
  final Widget child;
  final VoidCallback? onTap;

  const _HoverRow({required this.child, this.onTap});

  @override
  State<_HoverRow> createState() => _HoverRowState();
}

class _HoverRowState extends State<_HoverRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: widget.onTap != null
          ? SystemMouseCursors.click
          : SystemMouseCursors.basic,
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: _hovered ? Colors.white.withOpacity(0.02) : null,
            border: Border(
                bottom:
                    BorderSide(color: Colors.white.withOpacity(0.04))),
          ),
          child: widget.child,
        ),
      ),
    );
  }
}

class _DeleteIcon extends StatefulWidget {
  final VoidCallback onTap;
  const _DeleteIcon({required this.onTap});

  @override
  State<_DeleteIcon> createState() => _DeleteIconState();
}

class _DeleteIconState extends State<_DeleteIcon> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Icon(LucideIcons.trash2,
            size: 14,
            color:
                _hovered ? _red : Colors.white.withOpacity(0.15)),
      ),
    );
  }
}

class _ColumnTypeIcon extends StatelessWidget {
  final String type;
  const _ColumnTypeIcon({required this.type});

  @override
  Widget build(BuildContext context) {
    IconData icon;
    switch (type) {
      case 'string':
        icon = LucideIcons.type;
      case 'integer':
      case 'float':
      case 'double':
        icon = LucideIcons.hash;
      case 'boolean':
        icon = LucideIcons.toggleLeft;
      case 'datetime':
        icon = LucideIcons.calendar;
      case 'email':
        icon = LucideIcons.mail;
      case 'url':
        icon = LucideIcons.link;
      case 'enum':
        icon = LucideIcons.list;
      case 'point':
        icon = LucideIcons.mapPin;
      case 'relationship':
        icon = LucideIcons.link2;
      default:
        icon = LucideIcons.circle;
    }
    return Icon(icon, size: 12, color: _subtleText);
  }
}

class _SettingsCard extends StatelessWidget {
  final String title;
  final List<Widget> children;

  const _SettingsCard({required this.title, required this.children});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title,
              style: const TextStyle(
                  color: Colors.white,
                  fontSize: 15,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 16),
          ...children,
        ],
      ),
    );
  }
}

class _SettingsSectionCard extends StatelessWidget {
  final String? title;
  final String? subtitle;
  final List<Widget> children;
  final VoidCallback? onUpdate;

  const _SettingsSectionCard({
    this.title,
    this.subtitle,
    required this.children,
    this.onUpdate,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (title != null) ...[
            Text(title!,
                style: const TextStyle(
                    color: Colors.white,
                    fontSize: 15,
                    fontWeight: FontWeight.w600)),
            if (subtitle != null) ...[
              const SizedBox(height: 4),
              Text(subtitle!,
                  style: TextStyle(
                      color: Colors.white.withOpacity(0.4),
                      fontSize: 13)),
            ],
            const SizedBox(height: 16),
          ],
          ...children,
          if (onUpdate != null) ...[
            const SizedBox(height: 16),
            Align(
              alignment: Alignment.centerRight,
              child: FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: _accent,
                  padding: const EdgeInsets.symmetric(
                      horizontal: 20, vertical: 10),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                ),
                onPressed: onUpdate,
                child: const Text('Update',
                    style: TextStyle(fontSize: 13)),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _SettingsTextField extends StatefulWidget {
  final String initialValue;
  final ValueChanged<String> onSaved;

  const _SettingsTextField({
    required this.initialValue,
    required this.onSaved,
  });

  @override
  State<_SettingsTextField> createState() => _SettingsTextFieldState();
}

class _SettingsTextFieldState extends State<_SettingsTextField> {
  late final TextEditingController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: widget.initialValue);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        SizedBox(
          width: 80,
          child: Text('Name',
              style: TextStyle(
                  color: Colors.white.withOpacity(0.5),
                  fontSize: 12,
                  fontWeight: FontWeight.w500)),
        ),
        Expanded(
          child: TextField(
            controller: _ctrl,
            style: const TextStyle(color: Colors.white, fontSize: 13),
            decoration: InputDecoration(
              filled: true,
              fillColor: const Color(0x0AFFFFFF),
              isDense: true,
              contentPadding: const EdgeInsets.symmetric(
                  horizontal: 12, vertical: 10),
              border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide:
                      const BorderSide(color: Color(0x1AFFFFFF))),
              enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide:
                      const BorderSide(color: Color(0x1AFFFFFF))),
              focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: _accent)),
            ),
          ),
        ),
        const SizedBox(width: 12),
        FilledButton(
          style: FilledButton.styleFrom(
            backgroundColor: _accent,
            padding: const EdgeInsets.symmetric(
                horizontal: 20, vertical: 10),
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8)),
          ),
          onPressed: () => widget.onSaved(_ctrl.text.trim()),
          child: const Text('Update', style: TextStyle(fontSize: 13)),
        ),
      ],
    );
  }
}

class _DangerCard extends StatelessWidget {
  final String title;
  final String description;
  final VoidCallback onDelete;

  const _DangerCard({
    required this.title,
    required this.description,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _red.withOpacity(0.3)),
      ),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title,
                    style: const TextStyle(
                        color: _red,
                        fontSize: 14,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 4),
                Text(description,
                    style: TextStyle(
                        color: Colors.white.withOpacity(0.4),
                        fontSize: 13)),
              ],
            ),
          ),
          OutlinedButton(
            style: OutlinedButton.styleFrom(
              foregroundColor: _red,
              side: const BorderSide(color: _red),
              padding: const EdgeInsets.symmetric(
                  horizontal: 20, vertical: 10),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            onPressed: onDelete,
            child: const Text('Delete', style: TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  final String label;
  final String value;
  const _InfoRow({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          SizedBox(
            width: 120,
            child: Text(label,
                style: const TextStyle(color: _dimText, fontSize: 13)),
          ),
          Expanded(
            child: SelectableText(value,
                style: TextStyle(
                    color: Colors.white.withOpacity(0.8),
                    fontSize: 13,
                    fontFamily: 'monospace')),
          ),
        ],
      ),
    );
  }
}

class _PlaceholderTab extends StatelessWidget {
  final String label;
  const _PlaceholderTab({required this.label});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Text('$label — coming soon',
          style: const TextStyle(color: _dimText, fontSize: 14)),
    );
  }
}

class _EmptyState extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final String actionLabel;
  final VoidCallback onAction;

  const _EmptyState({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.actionLabel,
    required this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.04),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(icon, size: 22, color: _subtleText),
          ),
          const SizedBox(height: 16),
          Text(title,
              style: const TextStyle(
                  color: Colors.white,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          Text(subtitle,
              style: const TextStyle(color: _dimText, fontSize: 13),
              textAlign: TextAlign.center),
          const SizedBox(height: 16),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              padding:
                  const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            onPressed: onAction,
            child:
                Text(actionLabel, style: const TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }
}

class _PanelField extends StatelessWidget {
  final TextEditingController controller;
  final String label;
  final String hint;

  const _PanelField(
      {required this.controller, required this.label, required this.hint});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _PanelLabel(label),
        const SizedBox(height: 6),
        TextField(
          controller: controller,
          style: const TextStyle(color: Colors.white, fontSize: 13),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: TextStyle(
                color: Colors.white.withOpacity(0.22), fontSize: 13),
            filled: true,
            fillColor: const Color(0x0AFFFFFF),
            isDense: true,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: Color(0x1AFFFFFF))),
            enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: Color(0x1AFFFFFF))),
            focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: _accent)),
          ),
        ),
      ],
    );
  }
}

class _PanelLabel extends StatelessWidget {
  final String text;
  const _PanelLabel(this.text);

  @override
  Widget build(BuildContext context) {
    return Text(text,
        style: TextStyle(
            color: Colors.white.withOpacity(0.5),
            fontSize: 12,
            fontWeight: FontWeight.w500));
  }
}

class _PanelToggle extends StatelessWidget {
  final String label;
  final String subtitle;
  final bool value;
  final ValueChanged<bool> onChanged;

  const _PanelToggle({
    required this.label,
    required this.subtitle,
    required this.value,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () => onChanged(!value),
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SizedBox(
              width: 18,
              height: 18,
              child: Checkbox(
                value: value,
                onChanged: (v) => onChanged(v ?? false),
                activeColor: _accent,
                side: BorderSide(color: Colors.white.withOpacity(0.2)),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(3)),
              ),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(label,
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 13,
                          fontWeight: FontWeight.w500)),
                  Text(subtitle,
                      style: TextStyle(
                          color: Colors.white.withOpacity(0.35),
                          fontSize: 11)),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// --- Helpers ---

String _fmtDate(dynamic raw) {
  if (raw == null) return '—';
  try {
    final dt = raw is DateTime ? raw : DateTime.parse(raw.toString());
    const months = [
      '', 'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
      'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'
    ];
    return '${months[dt.month.clamp(1, 12)]} ${dt.day}, ${dt.year}';
  } catch (_) {
    return '—';
  }
}
