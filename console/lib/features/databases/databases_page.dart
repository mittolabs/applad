import 'dart:convert';
// ignore: deprecated_member_use, avoid_web_libraries_in_flutter
import 'dart:html' as html;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_monaco/flutter_monaco.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/id_text.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/search_list.dart';
import '../../core/theme/console_colors.dart';
import '../../core/widgets/app_error_state.dart';

// --- Constants ---------------------------------------------------------------

const _accent = Color(0xFF3472A4);
const _red = Color(0xFFEF4444);
const _green = Color(0xFF10B981);

String _formatIndexType(String value) {
  switch (value) {
    case 'unique':
      return 'Unique';
    case 'fulltext':
      return 'GIN full-text';
    case 'btree':
    case 'key':
    default:
      return 'B-tree';
  }
}

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

final _tableDetailsProvider = FutureProvider.family<Map<String, dynamic>,
    ({String dbId, String tableId})>((ref, p) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/databases/${p.dbId}/tables/${p.tableId}');
  return res.data as Map<String, dynamic>;
});

final _tablePermissionsProvider = FutureProvider.family<
    List<Map<String, dynamic>>,
    ({String dbId, String tableId})>((ref, p) async {
  final api = ref.read(apiClientProvider);
  final res = await api
      .get('/databases/${p.dbId}/tables/${p.tableId}/permissions');
  final data = res.data as Map<String, dynamic>;
  return List<Map<String, dynamic>>.from(data['permissions'] ?? []);
});

// --- Page (3-level navigation like Storage) ----------------------------------

class DatabasesPage extends ConsumerStatefulWidget {
  const DatabasesPage({super.key});

  @override
  ConsumerState<DatabasesPage> createState() => _DatabasesPageState();
}

class _DatabasesPageState extends ConsumerState<DatabasesPage> {
  late ConsoleColors _cs;
  final _searchCtrl = TextEditingController();
  int _topTab = 0;
  String? _selectedDbId;
  String? _selectedDbName;
  String? _selectedTableId;
  String? _selectedTableName;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final urlPage = pageFromQuery(context);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (ref.read(_dbPageProvider) != urlPage) {
        ref.read(_dbPageProvider.notifier).state = urlPage;
      }
    });
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
    if (_selectedTableId != null && _selectedDbId != null) {
      return _TableDetailView(
        dbId: _selectedDbId!,
        dbName: _selectedDbName ?? _selectedDbId!,
        tableId: _selectedTableId!,
        tableName: _selectedTableName ?? _selectedTableId!,
        onBack: () => setState(() => _selectedTableId = null),
      );
    }
    if (_selectedDbId != null) {
      return _DatabaseDetailView(
        dbId: _selectedDbId!,
        dbName: _selectedDbName ?? _selectedDbId!,
        onBack: () => setState(() => _selectedDbId = null),
        onTableSelect: (id, name) => setState(() {
          _selectedTableId = id;
          _selectedTableName = name;
        }),
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
      backgroundColor: _cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Text('Databases',
                style: TextStyle(
                    color: _cs.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            Text('Create and query structured data with real-time sync and row-level security',
                style: TextStyle(color: _cs.textSecondary, fontSize: 13)),
            const SizedBox(height: 20),
            PageTabs(
              tabs: const ['Databases', 'Usage'],
              selected: _topTab,
              onChanged: (i) => setState(() => _topTab = i),
            ),
            const SizedBox(height: 20),
            if (_topTab == 1) ...[
              Expanded(child: _buildUsageTab()),
            ] else ...[
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
                error: (e, _) => AppErrorState(error: e),
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
                          const Icon(LucideIcons.database, size: 14, color: _accent),
                          const SizedBox(width: 8),
                          Expanded(child: IdText(id: id)),
                        ]),
                        Text(name,
                            style: TextStyle(
                                color: _cs.textPrimary, fontSize: 13)),
                        Text(created,
                            style: TextStyle(
                                color: _cs.textMuted, fontSize: 12)),
                      ];
                    },
                    onRowTap: (db) => setState(() {
                      _selectedDbId = db['\$id'] as String;
                      _selectedDbName = db['name'] as String?;
                    }),
                    onRowDelete: (db) => _deleteDb(db['\$id'] as String),
                  );
                },
              ),
            ),
            SearchListFooter(
              total: total,
              perPage: perPage,
              currentPage: currentPage,
              onPrev: () {
                final p = currentPage - 1;
                ref.read(_dbPageProvider.notifier).state = p;
                context.go(withQuery(context, {'page': '$p'}));
              },
              onNext: () {
                final p = currentPage + 1;
                ref.read(_dbPageProvider.notifier).state = p;
                context.go(withQuery(context, {'page': '$p'}));
              },
              onPerPageChanged: (v) {
                ref.read(_dbPerPageProvider.notifier).state = v;
                ref.read(_dbPageProvider.notifier).state = 1;
              },
              itemLabel: 'Databases',
            ),
            const SizedBox(height: 8),
            ],  // end else
          ],
        ),
      ),
    );
  }

  Widget _buildUsageTab() {
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Usage',
              style: TextStyle(color: _cs.textPrimary, fontSize: 16, fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
          Text('Database activity for the past 30 days.',
              style: TextStyle(color: _cs.textSecondary, fontSize: 13)),
          const SizedBox(height: 24),
          Row(children: [
            _dbStatCard('Total databases', '—', LucideIcons.database),
            const SizedBox(width: 12),
            _dbStatCard('Total tables', '—', LucideIcons.table2),
            const SizedBox(width: 12),
            _dbStatCard('Total rows', '—', LucideIcons.rows),
          ]),
          const SizedBox(height: 24),
          Container(
            height: 200,
            decoration: BoxDecoration(
              color: _cs.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: _cs.border),
            ),
            child: Center(
              child: Text('Usage charts coming soon',
                  style: TextStyle(color: _cs.textSubtle, fontSize: 13)),
            ),
          ),
        ],
      ),
    );
  }

  Widget _dbStatCard(String label, String value, IconData icon) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: _cs.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: _cs.border),
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Icon(icon, size: 16, color: _cs.textSecondary),
          const SizedBox(height: 12),
          Text(value,
              style: TextStyle(color: _cs.textPrimary, fontSize: 24, fontWeight: FontWeight.w700)),
          const SizedBox(height: 4),
          Text(label, style: TextStyle(color: _cs.textSecondary, fontSize: 12)),
        ]),
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
          style: TextStyle(color: _cs.textSecondary)),
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
  final String dbName;
  final VoidCallback onBack;
  final void Function(String id, String name) onTableSelect;

  const _DatabaseDetailView({
    required this.dbId,
    required this.dbName,
    required this.onBack,
    required this.onTableSelect,
  });

  @override
  ConsumerState<_DatabaseDetailView> createState() =>
      _DatabaseDetailViewState();
}

class _DatabaseDetailViewState extends ConsumerState<_DatabaseDetailView> {
  late ConsoleColors _cs;
  int _tabIndex = 0;
  final _sqlCtrl = TextEditingController();
  MonacoController? _sqlEditorController;
  final _sqlFocusNode = FocusNode();
  final List<String> _sqlHistory = [];
  Map<String, dynamic>? _sqlResult;
  String? _sqlError;
  bool _sqlRunning = false;
  bool _sqlWriteAllowed = false;
  bool _sqlSchemaLoading = false;
  String? _sqlSchemaError;
  List<_SQLSchemaTable> _sqlSchema = const [];
  List<_SQLSuggestion> _sqlSuggestions = const [];

  @override
  void initState() {
    super.initState();
    _sqlCtrl.addListener(_updateSQLSuggestions);
    _loadSQLSchema();
  }

  @override
  void dispose() {
    _sqlEditorController?.dispose();
    _sqlCtrl.removeListener(_updateSQLSuggestions);
    _sqlCtrl.dispose();
    _sqlFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
    final tablesAsync = ref.watch(_tablesProvider(widget.dbId));
    final tables =
        List<Map<String, dynamic>>.from(tablesAsync.value?['tables'] ?? const []);

    return Scaffold(
      backgroundColor: _cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            _BackHeader(
              title: widget.dbName,
              subtitle: widget.dbId,
              icon: LucideIcons.database,
              onBack: widget.onBack,
            ),
            const SizedBox(height: 24),
            PageTabs(
              tabs: const ['Tables', 'SQL', 'Usage', 'Settings'],
              selected: _tabIndex,
              onChanged: (i) => setState(() => _tabIndex = i),
            ),
            const SizedBox(height: 20),
            Expanded(
              child: _tabIndex == 0
                  ? _buildTablesTab(tablesAsync)
                  : _tabIndex == 1
                      ? _buildSQLTab(tables)
                      : _tabIndex == 2
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
            error: (e, _) => AppErrorState(error: e),
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
                      const Icon(LucideIcons.table2,
                          size: 14, color: _accent),
                      const SizedBox(width: 8),
                      Expanded(child: IdText(id: id)),
                    ]),
                    Text(name,
                        style: TextStyle(
                            color: _cs.textPrimary, fontSize: 13)),
                    Text(created,
                        style: TextStyle(
                            color: _cs.textMuted, fontSize: 12)),
                  ];
                },
                onRowTap: (t) => widget.onTableSelect(
                  t['\$id'] as String,
                  t['name'] as String? ?? t['\$id'] as String,
                ),
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
              _InfoRow(label: 'Database ID', value: widget.dbId, isId: true),
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

  Widget _buildSQLTab(List<Map<String, dynamic>> tables) {
    final result = _sqlResult;
    final rows = List<Map<String, dynamic>>.from(result?['rows'] ?? const []);
    final columns = List<String>.from(result?['columns'] ?? const []);
    final tableIdByName = <String, String>{
      for (final table in tables)
        (table['name']?.toString() ?? ''): (table['\$id']?.toString() ?? ''),
    };

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          flex: 5,
          child: Column(
            children: [
              Container(
                padding: const EdgeInsets.all(18),
                decoration: BoxDecoration(
                  color: _cs.surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _cs.border),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Text(
                          'SQL editor',
                          style: TextStyle(
                            color: _cs.textPrimary,
                            fontSize: 15,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const Spacer(),
                        _GhostButton(
                          label: 'Copy',
                          icon: LucideIcons.copy,
                          onTap: () async {
                            await Clipboard.setData(
                              ClipboardData(text: _sqlCtrl.text),
                            );
                            if (mounted) {
                              ScaffoldMessenger.of(context).showSnackBar(
                                const SnackBar(content: Text('SQL copied')),
                              );
                            }
                          },
                        ),
                        const SizedBox(width: 8),
                        _AccentButton(
                          label: _sqlRunning ? 'Running...' : 'Run query',
                          onTap: _sqlRunning ? null : () {
                            _runSQL();
                          },
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'DDL is blocked. Queries run read-only by default with project and user context applied.',
                      style: TextStyle(
                        color: _cs.textMuted,
                        fontSize: 12,
                      ),
                    ),
                    const SizedBox(height: 16),
                    Container(
                      height: 320,
                      decoration: BoxDecoration(
                        color: _cs.fill,
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(color: _cs.border),
                      ),
                      child: Shortcuts(
                        shortcuts: const <ShortcutActivator, Intent>{
                          SingleActivator(
                            LogicalKeyboardKey.enter,
                            meta: true,
                          ): ActivateIntent(),
                          SingleActivator(
                            LogicalKeyboardKey.enter,
                            control: true,
                          ): ActivateIntent(),
                        },
                        child: Actions(
                          actions: <Type, Action<Intent>>{
                            ActivateIntent: CallbackAction<ActivateIntent>(
                              onInvoke: (_) {
                                if (!_sqlRunning) {
                                  _runSQL();
                                }
                                return null;
                              },
                            ),
                          },
                          child: ClipRRect(
                            borderRadius: BorderRadius.circular(8),
                            child: MonacoEditor(
                              key: const ValueKey('database-sql-editor'),
                              initialValue: _sqlCtrl.text,
                              readyTimeout: const Duration(seconds: 60),
                              options: EditorOptions(
                                language: MonacoLanguage.sql,
                                theme: consoleIsLight(context)
                                    ? MonacoTheme.vs
                                    : MonacoTheme.vsDark,
                                fontSize: 13,
                                fontFamily: 'Menlo, Monaco, Consolas, monospace',
                                lineHeight: 1.45,
                                wordWrap: true,
                                minimap: false,
                                automaticLayout: true,
                                quickSuggestions: true,
                                parameterHints: true,
                              ),
                              backgroundColor: _cs.background,
                              autofocus: true,
                              contentDebounce: const Duration(milliseconds: 75),
                              onReady: (controller) async {
                                _sqlEditorController = controller;
                                if (_sqlCtrl.text.isNotEmpty) {
                                  await controller.setValue(_sqlCtrl.text);
                                }
                              },
                              onFocus: () {
                                if (_sqlSuggestions.isNotEmpty && mounted) {
                                  setState(() => _sqlSuggestions = const []);
                                }
                              },
                              onContentChanged: (value) {
                                if (_sqlCtrl.text != value) {
                                  _sqlCtrl.text = value;
                                }
                              },
                              loadingBuilder: (_) => const Center(
                                child: CircularProgressIndicator(),
                              ),
                              errorBuilder: (_, error, __) => Center(
                                child: Padding(
                                  padding: const EdgeInsets.all(16),
                                  child: Text(
                                    'Editor failed to load: $error',
                                    style: const TextStyle(
                                      color: _red,
                                      fontSize: 12,
                                    ),
                                  ),
                                ),
                              ),
                            ),
                          ),
                        ),
                      ),
                    ),
                    const SizedBox(height: 14),
                    Row(
                      children: [
                        Switch(
                          value: _sqlWriteAllowed,
                          onChanged: _sqlRunning
                              ? null
                              : (v) {
                                  if (!v) {
                                    setState(() => _sqlWriteAllowed = false);
                                    return;
                                  }
                                  _confirmWriteMode();
                                },
                          activeThumbColor: _accent,
                        ),
                        const SizedBox(width: 8),
                        Text(
                          'Allow writes',
                          style: TextStyle(color: _cs.textPrimary, fontSize: 13),
                        ),
                        const SizedBox(width: 12),
                        Text(
                          _sqlWriteAllowed
                              ? 'INSERT, UPDATE and DELETE are allowed.'
                              : 'Transaction is forced read-only.',
                          style: TextStyle(
                            color: _sqlWriteAllowed ? const Color(0xFFFFD166) : _cs.textMuted,
                            fontSize: 12,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'Monaco powers the editor now. Use the schema browser on the right to insert tables and columns, then run with Cmd+Enter or Ctrl+Enter.',
                      style: TextStyle(
                        color: _cs.textSubtle,
                        fontSize: 11,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 16),
              Expanded(
                child: Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(18),
                  decoration: BoxDecoration(
                    color: _cs.surface,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: _cs.border),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Text(
                            'Results',
                            style: TextStyle(
                              color: _cs.textPrimary,
                              fontSize: 15,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                          const Spacer(),
                          if (result != null) ...[
                            if (columns.isNotEmpty) ...[
                              _GhostButton(
                                label: 'JSON',
                                icon: LucideIcons.fileJson,
                                onTap: () => _downloadSQLResults('json'),
                              ),
                              const SizedBox(width: 8),
                              _GhostButton(
                                label: 'CSV',
                                icon: LucideIcons.fileSpreadsheet,
                                onTap: () => _downloadSQLResults('csv'),
                              ),
                              const SizedBox(width: 8),
                            ],
                            _MetricPill(
                              label: '${result['rowCount'] ?? 0} rows',
                            ),
                            const SizedBox(width: 8),
                            _MetricPill(
                              label: '${result['executionMs'] ?? 0} ms',
                            ),
                          ],
                        ],
                      ),
                      const SizedBox(height: 14),
                      Expanded(
                        child: _sqlError != null
                            ? _SQLErrorView(message: _sqlError!)
                            : result == null
                                ? _EmptyState(
                                    icon: LucideIcons.database,
                                    title: 'No results yet',
                                    subtitle: 'Run a statement to inspect rows or affected counts.',
                                    actionLabel: 'Run query',
                                    onAction: () {
                                      _runSQL();
                                    },
                                  )
                                : rows.isEmpty
                                    ? _SQLSummaryView(result: result)
                                    : _SQLResultsGrid(columns: columns, rows: rows),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: 16),
        SizedBox(
          width: 300,
          child: Column(
            children: [
              Expanded(
                child: _SQLSchemaBrowser(
                  schema: _sqlSchema,
                  loading: _sqlSchemaLoading,
                  error: _sqlSchemaError,
                  tableIdByName: tableIdByName,
                  onTableSelected: widget.onTableSelect,
                  onUseSuggestion: _insertSQLIdentifier,
                ),
              ),
              const SizedBox(height: 16),
              SizedBox(
                height: 240,
                child: Container(
                  padding: const EdgeInsets.all(18),
                  decoration: BoxDecoration(
                    color: _cs.surface,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: _cs.border),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Recent queries',
                        style: TextStyle(
                          color: _cs.textPrimary,
                          fontSize: 15,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 12),
                      Expanded(
                        child: _sqlHistory.isEmpty
                            ? Text(
                                'Recent statements from this session will appear here.',
                                style: TextStyle(
                                  color: _cs.textSubtle,
                                  fontSize: 12,
                                  height: 1.5,
                                ),
                              )
                            : ListView.separated(
                                itemCount: _sqlHistory.length,
                                separatorBuilder: (_, __) => const SizedBox(height: 8),
                                itemBuilder: (_, index) {
                                  final statement = _sqlHistory[index];
                                  return InkWell(
                                    onTap: () => _setSQLText(statement),
                                    borderRadius: BorderRadius.circular(8),
                                    child: Container(
                                      padding: const EdgeInsets.all(12),
                                      decoration: BoxDecoration(
                                        color: _cs.fill,
                                        borderRadius: BorderRadius.circular(8),
                                        border: Border.all(color: _cs.border),
                                      ),
                                      child: Text(
                                        statement,
                                        maxLines: 4,
                                        overflow: TextOverflow.ellipsis,
                                        style: TextStyle(
                                          color: _cs.textPrimary,
                                          fontSize: 12,
                                          fontFamily: 'monospace',
                                          height: 1.4,
                                        ),
                                      ),
                                    ),
                                  );
                                },
                              ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Future<void> _confirmWriteMode() async {
    await showAppDialog(
      context: context,
      title: 'Enable write mode',
      subtitle:
          'Write mode allows INSERT, UPDATE, and DELETE statements in this database. Schema changes are still blocked.',
      content: Text(
        'Enable write mode only when you intend to mutate data. The SQL editor still blocks DDL statements and keeps the current project and user context applied.',
        style: TextStyle(color: _cs.textSecondary, fontSize: 13, height: 1.5),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Enable writes',
          destructive: true,
          onTap: () {
            setState(() => _sqlWriteAllowed = true);
            Navigator.of(context, rootNavigator: true).pop();
          },
        ),
      ],
    );
  }

  Future<void> _loadSQLSchema() async {
    setState(() {
      _sqlSchemaLoading = true;
      _sqlSchemaError = null;
    });

    try {
      final response = await ref.read(apiClientProvider).post(
        '/databases/${widget.dbId}/sql',
        data: {
          'statement': '''
select
  table_name,
  column_name,
  data_type,
  is_nullable,
  udt_name
from information_schema.columns
where table_schema = current_schema()
order by table_name, ordinal_position
''',
          'writeAllowed': false,
        },
      );
      final payload = Map<String, dynamic>.from(response.data as Map);
      final rows = List<Map<String, dynamic>>.from(payload['rows'] ?? const []);
      final grouped = <String, List<_SQLSchemaColumn>>{};
      for (final row in rows) {
        final tableName = row['table_name']?.toString() ?? '';
        final columnName = row['column_name']?.toString() ?? '';
        if (tableName.isEmpty || columnName.isEmpty) {
          continue;
        }
        grouped.putIfAbsent(tableName, () => []);
        grouped[tableName]!.add(
          _SQLSchemaColumn(
            name: columnName,
            type: row['data_type']?.toString() ?? row['udt_name']?.toString() ?? 'text',
            required: (row['is_nullable']?.toString() ?? '').toUpperCase() == 'NO',
          ),
        );
      }
      final schema = grouped.entries
          .map((entry) => _SQLSchemaTable(name: entry.key, columns: entry.value))
          .toList()
        ..sort((left, right) => left.name.compareTo(right.name));

      if (!mounted) {
        return;
      }
      setState(() {
        _sqlSchema = schema;
      });
      _updateSQLSuggestions();
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _sqlSchemaError = error.toString();
      });
    } finally {
      if (mounted) {
        setState(() => _sqlSchemaLoading = false);
      }
    }
  }

  void _updateSQLSuggestions() {
    final value = _sqlCtrl.value;
    final cursor = value.selection.baseOffset;
    if (cursor < 0 || !_sqlFocusNode.hasFocus) {
      if (_sqlSuggestions.isNotEmpty) {
        setState(() => _sqlSuggestions = const []);
      }
      return;
    }

    final prefix = value.text.substring(0, cursor);
    final dotted = RegExp(r'([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)?$').firstMatch(prefix);
    final nextSuggestions = <_SQLSuggestion>[];

    if (dotted != null) {
      final tableName = dotted.group(1)?.toLowerCase() ?? '';
      final filter = dotted.group(2)?.toLowerCase() ?? '';
      final table = _sqlSchema.where((item) => item.name.toLowerCase() == tableName).firstOrNull;
      if (table != null) {
        for (final column in table.columns) {
          if (filter.isNotEmpty && !column.name.toLowerCase().startsWith(filter)) {
            continue;
          }
          nextSuggestions.add(
            _SQLSuggestion(
              label: '${table.name}.${column.name}',
              detail: column.type,
              insertText: column.name,
              replacementPrefix: filter,
            ),
          );
        }
      }
    } else {
      final token = RegExp(r'([A-Za-z_][A-Za-z0-9_]*)$').firstMatch(prefix)?.group(1) ?? '';
      if (token.isNotEmpty) {
        final lowerToken = token.toLowerCase();
        for (final table in _sqlSchema) {
          if (table.name.toLowerCase().startsWith(lowerToken)) {
            nextSuggestions.add(
              _SQLSuggestion(
                label: table.name,
                detail: 'table',
                insertText: table.name,
                replacementPrefix: token,
              ),
            );
          }
          for (final column in table.columns) {
            if (!column.name.toLowerCase().startsWith(lowerToken)) {
              continue;
            }
            nextSuggestions.add(
              _SQLSuggestion(
                label: column.name,
                detail: '${column.type} • ${table.name}',
                insertText: column.name,
                replacementPrefix: token,
              ),
            );
          }
        }
      }
    }

    nextSuggestions.sort((left, right) => left.label.compareTo(right.label));
    final limited = nextSuggestions.take(8).toList(growable: false);
    if (!_sameSuggestions(limited, _sqlSuggestions)) {
      setState(() => _sqlSuggestions = limited);
    }
  }

  bool _sameSuggestions(List<_SQLSuggestion> left, List<_SQLSuggestion> right) {
    if (left.length != right.length) {
      return false;
    }
    for (var index = 0; index < left.length; index++) {
      if (left[index].label != right[index].label ||
          left[index].detail != right[index].detail ||
          left[index].insertText != right[index].insertText) {
        return false;
      }
    }
    return true;
  }

  void _insertSQLIdentifier(String identifier) {
    final value = _sqlCtrl.value;
    final insertionPoint = value.text.length;
    final prefix = insertionPoint > 0 && !value.text[insertionPoint - 1].contains(RegExp(r'\s'))
        ? ' '
        : '';
    const suffix = '';
    final insertText = '$prefix$identifier$suffix';
    final nextText = value.text.replaceRange(insertionPoint, insertionPoint, insertText);
    _setSQLText(nextText);
  }

  void _setSQLText(String value) {
    _sqlCtrl.value = TextEditingValue(
      text: value,
      selection: TextSelection.collapsed(offset: value.length),
      composing: TextRange.empty,
    );
    _sqlEditorController?.setValue(value);
  }

  Future<void> _downloadSQLResults(String format) async {
    final result = _sqlResult;
    if (result == null) {
      return;
    }

    final rows = List<Map<String, dynamic>>.from(result['rows'] ?? const []);
    final columns = List<String>.from(result['columns'] ?? const []);
    if (rows.isEmpty || columns.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('There are no tabular results to export.')),
      );
      return;
    }

    final timestamp = DateTime.now().toIso8601String().replaceAll(':', '-');
    final safeDbId = widget.dbId.replaceAll(RegExp(r'[^a-zA-Z0-9_-]'), '_');
    late final String content;
    late final String filename;
    late final String mimeType;

    if (format == 'csv') {
      content = _buildCSVExport(columns, rows);
      filename = '${safeDbId}_query_$timestamp.csv';
      mimeType = 'text/csv;charset=utf-8';
    } else {
      content = const JsonEncoder.withIndent('  ').convert(rows);
      filename = '${safeDbId}_query_$timestamp.json';
      mimeType = 'application/json;charset=utf-8';
    }

    final blob = html.Blob([content], mimeType);
    final url = html.Url.createObjectUrlFromBlob(blob);
    final anchor = html.AnchorElement(href: url)
      ..download = filename
      ..style.display = 'none';

    html.document.body?.append(anchor);
    anchor.click();
    anchor.remove();
    html.Url.revokeObjectUrl(url);

    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Downloaded $filename')),
      );
    }
  }

  Future<void> _runSQL() async {
    final statement = _sqlCtrl.text.trim();
    if (statement.isEmpty) {
      setState(() => _sqlError = 'SQL statement is required.');
      return;
    }

    setState(() {
      _sqlRunning = true;
      _sqlError = null;
    });

    try {
      final response = await ref.read(apiClientProvider).post(
        '/databases/${widget.dbId}/sql',
        data: {
          'statement': statement,
          'writeAllowed': _sqlWriteAllowed,
        },
      );
      final payload = Map<String, dynamic>.from(response.data as Map);
      setState(() {
        _sqlResult = payload;
        _sqlHistory.remove(statement);
        _sqlHistory.insert(0, statement);
        if (_sqlHistory.length > 12) {
          _sqlHistory.removeRange(12, _sqlHistory.length);
        }
        _sqlSuggestions = const [];
      });
    } catch (e) {
      setState(() {
        _sqlResult = null;
        _sqlError = e.toString();
      });
    } finally {
      if (mounted) {
        setState(() => _sqlRunning = false);
      }
    }
  }
}

class _SQLSchemaBrowser extends StatelessWidget {
  final List<_SQLSchemaTable> schema;
  final bool loading;
  final String? error;
  final Map<String, String> tableIdByName;
  final void Function(String id, String name) onTableSelected;
  final ValueChanged<String> onUseSuggestion;

  const _SQLSchemaBrowser({
    required this.schema,
    required this.loading,
    required this.error,
    required this.tableIdByName,
    required this.onTableSelected,
    required this.onUseSuggestion,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Schema browser',
            style: TextStyle(
              color: cs.textPrimary,
              fontSize: 15,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Inspect tables and columns while writing queries.',
            style: TextStyle(
              color: cs.textMuted,
              fontSize: 12,
              height: 1.5,
            ),
          ),
          const SizedBox(height: 12),
          Expanded(child: _buildSchemaBody(context)),
        ],
      ),
    );
  }

  Widget _buildSchemaBody(BuildContext context) {
    if (loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (error != null) {
      return Text(
        'Error: $error',
        style: const TextStyle(color: _red, fontSize: 12),
      );
    }
    final cs = consoleColors(context);
    if (schema.isEmpty) {
      return Text(
        'No schema metadata available yet.',
        style: TextStyle(
          color: cs.textSubtle,
          fontSize: 12,
          height: 1.5,
        ),
      );
    }

    return ListView.separated(
      itemCount: schema.length,
      separatorBuilder: (_, __) => const SizedBox(height: 10),
      itemBuilder: (context, index) {
        final table = schema[index];
        final tableId = tableIdByName[table.name] ?? '';
        return Container(
          decoration: BoxDecoration(
            color: cs.fill,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: cs.border),
          ),
          child: Theme(
            data: Theme.of(context).copyWith(dividerColor: Colors.transparent),
            child: ExpansionTile(
              tilePadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
              childrenPadding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
              iconColor: cs.textSecondary,
              collapsedIconColor: cs.textMuted,
              title: Text(
                table.name,
                style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                ),
              ),
              subtitle: Text(
                'information_schema',
                style: TextStyle(
                  color: cs.textMuted,
                  fontSize: 11,
                  fontFamily: 'monospace',
                ),
              ),
              children: [
                Align(
                  alignment: Alignment.centerLeft,
                  child: Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: [
                      TextButton.icon(
                        onPressed: () => onUseSuggestion(table.name),
                        icon: const Icon(LucideIcons.copyPlus, size: 14),
                        label: const Text('Insert table'),
                        style: TextButton.styleFrom(
                          foregroundColor: cs.textSecondary,
                          padding: const EdgeInsets.symmetric(horizontal: 0),
                        ),
                      ),
                      TextButton.icon(
                        onPressed: tableId.isEmpty ? null : () => onTableSelected(tableId, table.name),
                        icon: const Icon(LucideIcons.arrowUpRight, size: 14),
                        label: const Text('Open table'),
                        style: TextButton.styleFrom(
                          foregroundColor: cs.textSecondary,
                          padding: const EdgeInsets.symmetric(horizontal: 0),
                        ),
                      ),
                    ],
                  ),
                ),
                Column(
                  children: table.columns.map((column) {
                    return Container(
                      margin: const EdgeInsets.only(top: 8),
                      padding: const EdgeInsets.all(10),
                      decoration: BoxDecoration(
                        color: cs.surfaceAlt,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Icon(LucideIcons.table2, size: 14, color: _accent),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Row(
                                  children: [
                                    Expanded(
                                      child: Text(
                                        column.name,
                                        style: TextStyle(
                                          color: cs.textPrimary,
                                          fontSize: 12,
                                          fontFamily: 'monospace',
                                        ),
                                      ),
                                    ),
                                    GestureDetector(
                                      onTap: () => onUseSuggestion(column.name),
                                      child: Icon(
                                        LucideIcons.plus,
                                        size: 14,
                                        color: cs.textSecondary,
                                      ),
                                    ),
                                  ],
                                ),
                                const SizedBox(height: 4),
                                Wrap(
                                  spacing: 6,
                                  runSpacing: 6,
                                  children: [
                                    _MetricPill(label: column.type),
                                    if (column.required)
                                      const _MetricPill(label: 'required'),
                                  ],
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    );
                  }).toList(),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _SQLSchemaTable {
  final String name;
  final List<_SQLSchemaColumn> columns;

  const _SQLSchemaTable({required this.name, required this.columns});
}

class _SQLSchemaColumn {
  final String name;
  final String type;
  final bool required;

  const _SQLSchemaColumn({
    required this.name,
    required this.type,
    required this.required,
  });
}

class _SQLSuggestion {
  final String label;
  final String detail;
  final String insertText;
  final String replacementPrefix;

  const _SQLSuggestion({
    required this.label,
    required this.detail,
    required this.insertText,
    required this.replacementPrefix,
  });
}

// =============================================================================
// Level 3: Table Detail (Rows, Columns, Indexes, Relationships, Settings)
// =============================================================================

class _TableDetailView extends ConsumerStatefulWidget {
  final String dbId;
  final String dbName;
  final String tableId;
  final String tableName;
  final VoidCallback onBack;

  const _TableDetailView({
    required this.dbId,
    required this.dbName,
    required this.tableId,
    required this.tableName,
    required this.onBack,
  });

  @override
  ConsumerState<_TableDetailView> createState() =>
      _TableDetailViewState();
}

class _TableDetailViewState extends ConsumerState<_TableDetailView> {
  late ConsoleColors _cs;
  int _tabIndex = 0;
  bool _showCreateColumn = false;
  // Optimistic state for settings toggles (null = use API value)
  bool? _tableEnabled;
  bool? _tableRowSecurity;

  ({String dbId, String tableId}) get _key =>
      (dbId: widget.dbId, tableId: widget.tableId);

  String get _basePath =>
      '/databases/${widget.dbId}/tables/${widget.tableId}';

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
    return Scaffold(
      backgroundColor: _cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            _BackHeader(
              title: widget.tableName,
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
            error: (e, _) => AppErrorState(error: e),
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
                onEdit: (row) =>
                    _showEditRowDialog(row, columns),
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
            error: (e, _) => AppErrorState(error: e),
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
                  'Default value',
                  'Permissions',
                ],
                flexes: const [3, 2, 1, 2, 2],
                rows: columns,
                rowBuilder: (col) {
                  final key = col['key'] as String? ?? '';
                  final type = col['type'] as String? ?? '';
                  final required = col['required'] == true;
                  final def = col['default']?.toString() ?? '-';
                  final perms = List<String>.from(
                      col['\$permissions'] as List? ?? ['read', 'write']);
                  return [
                    Row(children: [
                      _ColumnTypeIcon(type: type),
                      const SizedBox(width: 8),
                      Text(key,
                          style: TextStyle(
                              color: _cs.textPrimary, fontSize: 13)),
                      if (required) ...[
                        const SizedBox(width: 6),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 5, vertical: 1),
                          decoration: BoxDecoration(
                            color: _accent.withValues(alpha: 0.15),
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
                        style: TextStyle(
                            color: _cs.textMuted, fontSize: 13)),
                    Icon(
                      required
                          ? LucideIcons.checkSquare
                          : LucideIcons.square,
                      size: 14,
                      color: required ? _accent : _cs.textSubtle,
                    ),
                    Text(def,
                        style: TextStyle(
                            color: _cs.textMuted, fontSize: 13)),
                    Wrap(
                      spacing: 4,
                      children: [
                        if (perms.contains('read'))
                          const _PermChip(label: 'read', color: _green),
                        if (perms.contains('write'))
                          const _PermChip(label: 'write', color: _accent),
                        if (!perms.contains('read') && !perms.contains('write'))
                          const _PermChip(label: 'none', color: _red),
                      ],
                    ),
                  ];
                },
                onRowTap: (col) => _showColumnPermDialog(col),
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
            error: (e, _) => AppErrorState(error: e),
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
                        style: TextStyle(
                            color: _cs.textPrimary, fontSize: 13)),
                  Text(_formatIndexType(idx['type'] as String? ?? ''),
                        style: TextStyle(
                            color: _cs.textMuted, fontSize: 13)),
                    Text(
                    (idx['columns'] as List?)?.join(', ') ??
                      (idx['attributes'] as List?)?.join(', ') ??
                            '',
                        style: TextStyle(
                            color: _cs.textMuted, fontSize: 12)),
                    Text(
                        (idx['orders'] as List?)?.join(', ') ?? '',
                        style: TextStyle(
                            color: _cs.textMuted, fontSize: 12)),
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
            error: (e, _) => AppErrorState(error: e),
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
                        style: TextStyle(
                            color: _cs.textPrimary, fontSize: 13)),
                    Text(r['type'] as String? ?? '',
                        style: TextStyle(
                            color: _cs.textMuted, fontSize: 13)),
                    Text(r['relatedTableId'] as String? ?? '',
                        style: TextStyle(
                            color: _cs.textMuted, fontSize: 13)),
                    Text(r['onDelete'] as String? ?? 'setNull',
                        style: TextStyle(
                            color: _cs.textMuted, fontSize: 13)),
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
    final tableAsync = ref.watch(_tableDetailsProvider(_key));
    final permsAsync = ref.watch(_tablePermissionsProvider(_key));

    return tableAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e),
      data: (table) {
        final apiEnabled = table['enabled'] as bool? ?? true;
        final apiRowSecurity = table['rowSecurity'] as bool? ?? false;
        final tableName = table['name'] as String? ?? widget.tableId;
        final enabledVal = _tableEnabled ?? apiEnabled;
        final rowSecurityVal = _tableRowSecurity ?? apiRowSecurity;

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
                            Text(tableName,
                                style: TextStyle(
                                    color: _cs.textPrimary,
                                    fontSize: 15,
                                    fontWeight: FontWeight.w600)),
                            const SizedBox(height: 4),
                            Text('Table ID: ${widget.tableId}',
                                style: TextStyle(
                                    color: _cs.textSubtle, fontSize: 12)),
                          ],
                        ),
                      ),
                      Row(
                        children: [
                          Switch(
                            value: enabledVal,
                            onChanged: (v) async {
                              setState(() => _tableEnabled = v);
                              await ref
                                  .read(apiClientProvider)
                                  .put(_basePath, data: {'enabled': v});
                              ref.invalidate(_tableDetailsProvider(_key));
                            },
                            activeThumbColor: _accent,
                          ),
                          const SizedBox(width: 4),
                          Text('Enabled',
                              style: TextStyle(
                                  color: _cs.textPrimary, fontSize: 13)),
                        ],
                      ),
                    ],
                  ),
                ],
              ),

              // 2. Name
              _SettingsSectionCard(
                title: 'Name',
                children: [
                  _SettingsTextField(
                    initialValue: tableName,
                    onSaved: (v) async {
                      await ref.read(apiClientProvider).put(_basePath,
                          data: {'name': v});
                      ref.invalidate(_tablesProvider(widget.dbId));
                      ref.invalidate(_tableDetailsProvider(_key));
                    },
                  ),
                ],
              ),

              // 3. Permissions
              _SettingsSectionCard(
                title: 'Permissions',
                subtitle: 'Choose who can access your tables and rows.',
                children: [
                  permsAsync.when(
                    loading: () => const Center(
                        child: CircularProgressIndicator()),
                    error: (e, _) => AppErrorState(error: e),
                    data: (perms) => _buildPermissionsPanel(perms),
                  ),
                ],
              ),

              // 4. Row security
              _SettingsSectionCard(
                title: 'Row security',
                children: [
                  Row(
                    children: [
                      Switch(
                        value: rowSecurityVal,
                        onChanged: (v) async {
                          setState(() => _tableRowSecurity = v);
                          await ref.read(apiClientProvider).put(
                              _basePath, data: {'rowSecurity': v});
                          ref.invalidate(_tableDetailsProvider(_key));
                        },
                        activeThumbColor: _accent,
                      ),
                      const SizedBox(width: 8),
                      Text('Row security',
                          style: TextStyle(
                              color: _cs.textPrimary, fontSize: 13)),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Text(
                    'When row security is enabled, users will be able to access rows for which they have been granted either row or table permissions.\n\n'
                    'If row security is disabled, users can access rows only if they have table permissions. Row permissions will be ignored.',
                    style: TextStyle(
                        color: _cs.textSubtle,
                        fontSize: 12,
                        height: 1.5),
                  ),
                ],
              ),

              // 5. Delete table
              _DangerCard(
                title: 'Delete table',
                description:
                    'The table will be permanently deleted, including all the rows within it. This action is irreversible.',
                onDelete: () async {
                  await ref.read(apiClientProvider).delete(_basePath);
                  ref.invalidate(_tablesProvider(widget.dbId));
                  widget.onBack();
                },
              ),
              const SizedBox(height: 40),
            ],
          ),
        );
      },
    );
  }

  Widget _buildPermissionsPanel(List<Map<String, dynamic>> perms) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (perms.isEmpty)
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: _cs.fill,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: _cs.border),
            ),
            child: Column(
              children: [
                Icon(LucideIcons.shieldOff,
                    size: 20, color: _cs.textSubtle),
                const SizedBox(height: 8),
                Text('No permissions set. All access is denied.',
                    style:
                        TextStyle(color: _cs.textMuted, fontSize: 13)),
              ],
            ),
          )
        else
          Container(
            decoration: BoxDecoration(
              border: Border.all(color: _cs.border),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Column(
              children: perms.asMap().entries.map((entry) {
                final i = entry.key;
                final p = entry.value;
                final role = p['role'] as String? ?? '';
                final action = p['action'] as String? ?? '';
                return Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 14, vertical: 10),
                  decoration: BoxDecoration(
                    border: Border(
                      bottom: i < perms.length - 1
                          ? BorderSide(color: _cs.border)
                          : BorderSide.none,
                    ),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(role,
                            style: TextStyle(
                                color: _cs.textPrimary, fontSize: 13)),
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(
                          color: _accent.withValues(alpha: 0.12),
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: Text(action,
                            style: const TextStyle(
                                color: _accent,
                                fontSize: 11,
                                fontWeight: FontWeight.w500)),
                      ),
                      const SizedBox(width: 8),
                      GestureDetector(
                        onTap: () async {
                          final updated = perms
                              .where((x) =>
                                  x['role'] != role ||
                                  x['action'] != action)
                              .toList();
                          await ref.read(apiClientProvider).post(
                            '$_basePath/permissions',
                            data: {
                              'permissions': updated
                                  .map((x) => {
                                        'role': x['role'],
                                        'action': x['action']
                                      })
                                  .toList()
                            },
                          );
                          ref.invalidate(_tablePermissionsProvider(_key));
                        },
                        child: Icon(LucideIcons.x,
                            size: 14, color: _cs.textSubtle),
                      ),
                    ],
                  ),
                );
              }).toList(),
            ),
          ),
        const SizedBox(height: 12),
        TextButton.icon(
          onPressed: () => _showAddPermissionDialog(perms),
          icon: const Icon(LucideIcons.plus, size: 14, color: _accent),
          label: const Text('Add permission',
              style: TextStyle(color: _accent, fontSize: 13)),
          style: TextButton.styleFrom(
            padding:
                const EdgeInsets.symmetric(horizontal: 0, vertical: 4),
          ),
        ),
      ],
    );
  }

  void _showColumnPermDialog(Map<String, dynamic> col) {
    final key = col['key'] as String? ?? '';
    final perms = List<String>.from(
        col['\$permissions'] as List? ?? ['read', 'write']);
    bool allowRead = perms.contains('read');
    bool allowWrite = perms.contains('write');

    showAppDialog<void>(
      context: context,
      title: 'Column permissions: $key',
      content: StatefulBuilder(
        builder: (ctx, setDialogState) {
          final cs = consoleColors(ctx);
          return Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Control whether this column value can be read or written via the API.',
                style: TextStyle(color: cs.textSecondary, fontSize: 13),
              ),
              const SizedBox(height: 16),
              _PermToggleRow(
                label: 'Allow read',
                subtitle: 'Column values are returned when fetching rows.',
                value: allowRead,
                onChanged: (v) => setDialogState(() => allowRead = v),
              ),
              const SizedBox(height: 8),
              _PermToggleRow(
                label: 'Allow write',
                subtitle: 'Column values can be set when creating or updating rows.',
                value: allowWrite,
                onChanged: (v) => setDialogState(() => allowWrite = v),
              ),
            ],
          );
        },
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Save',
          onTap: () async {
            final updated = <String>[
              if (allowRead) 'read',
              if (allowWrite) 'write',
            ];
            await ref.read(apiClientProvider).post(
              '$_basePath/columns/$key/permissions',
              data: {'permissions': updated},
            );
            ref.invalidate(_columnsProvider(_key));
            if (!mounted) return;
            Navigator.of(context, rootNavigator: true).pop();
          },
        ),
      ],
    );
  }

  void _showAddPermissionDialog(List<Map<String, dynamic>> existing) {
    final roleCtrl = TextEditingController();
    String action = 'read';

    showAppDialog(
      context: context,
      title: 'Add permission',
      width: 420,
      content: StatefulBuilder(
        builder: (ctx, setS) => Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AppDialogField(
              controller: roleCtrl,
              label: 'Role',
              hint: 'e.g. users, any, user:123',
              autofocus: true,
            ),
            const SizedBox(height: 12),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Action',
                    style: TextStyle(
                        color: consoleColors(ctx).textMuted,
                        fontSize: 12,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  runSpacing: 6,
                  children: ['read', 'create', 'update', 'delete']
                      .map((a) {
                    final sel = action == a;
                    return GestureDetector(
                      onTap: () => setS(() => action = a),
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 6),
                        decoration: BoxDecoration(
                          color: sel
                              ? _accent.withValues(alpha: 0.15)
                              : consoleColors(ctx).fill,
                          borderRadius: BorderRadius.circular(6),
                          border: Border.all(
                              color: sel
                                  ? _accent
                                  : consoleColors(ctx).border),
                        ),
                        child: Text(a,
                            style: TextStyle(
                                color: sel
                                    ? _cs.textPrimary
                                    : consoleColors(ctx).textMuted,
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
          label: 'Add',
          onTap: () async {
            final role = roleCtrl.text.trim();
            if (role.isEmpty) return;
            final updated = [
              ...existing.map(
                  (x) => {'role': x['role'], 'action': x['action']}),
              {'role': role, 'action': action},
            ];
            await ref.read(apiClientProvider).post(
              '$_basePath/permissions',
              data: {'permissions': updated},
            );
            if (mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
            ref.invalidate(_tablePermissionsProvider(_key));
          },
        ),
      ],
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

  void _showEditRowDialog(
      Map<String, dynamic> row, List<Map<String, dynamic>> columns) {
    final data = row['data'] as Map<String, dynamic>? ?? row;
    final rowId = row['\$id'] as String? ?? '';
    final controllers = <String, TextEditingController>{};
    for (final col in columns) {
      final key = col['key'] as String? ?? '';
      if (key.isNotEmpty) {
        controllers[key] =
            TextEditingController(text: data[key]?.toString() ?? '');
      }
    }

    showAppDialog(
      context: context,
      title: 'Edit row',
      width: 520,
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (controllers.isEmpty)
            AppDialogField(
              controller: TextEditingController(
                  text: const JsonEncoder.withIndent('  ')
                      .convert(data)),
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
          label: 'Save',
          onTap: () async {
            final updated = <String, dynamic>{};
            for (final e in controllers.entries) {
              updated[e.key] = e.value.text.trim();
            }
            await ref.read(apiClientProvider).patch(
                '$_basePath/rows/$rowId',
                data: {'data': updated});
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
    String indexType = 'btree';

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
                        color: _cs.textMuted,
                        fontSize: 12,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  children: const [
                    ('btree', 'B-tree'),
                    ('unique', 'Unique'),
                    ('fulltext', 'GIN full-text'),
                  ].map((option) {
                    final value = option.$1;
                    final label = option.$2;
                    final sel = indexType == value;
                    return GestureDetector(
                      onTap: () => setDialogState(() => indexType = value),
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 6),
                        decoration: BoxDecoration(
                          color: sel
                              ? _accent.withValues(alpha: 0.15)
                              : _cs.fill,
                          borderRadius: BorderRadius.circular(6),
                          border: Border.all(
                              color: sel
                                  ? _accent
                                  : _cs.border),
                        ),
                        child: Text(label,
                            style: TextStyle(
                                color: sel ? _cs.textPrimary : _cs.textMuted,
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
                  'columns': cols,
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
                        color: _cs.textMuted,
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
                              ? _accent.withValues(alpha: 0.15)
                              : _cs.fill,
                          borderRadius: BorderRadius.circular(6),
                          border: Border.all(
                              color: sel
                                  ? _accent
                                  : _cs.border),
                        ),
                        child: Text(t,
                            style: TextStyle(
                                color: sel ? _cs.textPrimary : _cs.textMuted,
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
                relTableCtrl.text.trim().isEmpty) { return; }
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
  late ConsoleColors _cs;
  final _keyCtrl = TextEditingController();
  final _sizeCtrl = TextEditingController(text: '256');
  final _defaultCtrl = TextEditingController();
  final _elementsCtrl = TextEditingController();
  // Validation fields
  final _patternCtrl = TextEditingController();
  final _validationMsgCtrl = TextEditingController();
  final _minCtrl = TextEditingController();
  final _maxCtrl = TextEditingController();
  final _minLenCtrl = TextEditingController();
  final _maxLenCtrl = TextEditingController();
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
    _patternCtrl.dispose();
    _validationMsgCtrl.dispose();
    _minCtrl.dispose();
    _maxCtrl.dispose();
    _minLenCtrl.dispose();
    _maxLenCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
    return Container(
      width: 320,
      margin: const EdgeInsets.only(left: 16),
      decoration: BoxDecoration(
        color: _cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 0),
            child: Row(
              children: [
                Text('Create column',
                    style: TextStyle(
                        color: _cs.textPrimary,
                        fontSize: 14,
                        fontWeight: FontWeight.w600)),
                const Spacer(),
                GestureDetector(
                  onTap: widget.onClose,
                  child: MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: Icon(LucideIcons.x,
                        size: 16,
                        color: _cs.textSubtle),
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
                  const _PanelLabel('Type'),
                  const SizedBox(height: 6),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 0),
                    decoration: BoxDecoration(
                      color: _cs.fieldFill,
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: _cs.fieldBorder),
                    ),
                    child: DropdownButtonHideUnderline(
                      child: DropdownButton<String>(
                        value: _type,
                        isExpanded: true,
                        isDense: true,
                        dropdownColor: _cs.popupSurface,
                        style: TextStyle(
                            color: _cs.textPrimary, fontSize: 13),
                        icon: Icon(LucideIcons.chevronDown,
                            size: 14, color: _cs.textMuted),
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
                  if (_type != 'boolean' && _type != 'datetime' &&
                      _type != 'point' && _type != 'relationship') ...[
                    const SizedBox(height: 16),
                    // ── Validation ────────────────────────────────────────
                    Text('Validation',
                        style: TextStyle(
                            color: _cs.textSecondary,
                            fontSize: 11,
                            fontWeight: FontWeight.w600,
                            letterSpacing: 0.6)),
                    const SizedBox(height: 10),
                    if (_type == 'string') ...[
                      _PanelField(
                          controller: _minLenCtrl,
                          label: 'Min length',
                          hint: 'e.g. 3',
                          keyboardType: TextInputType.number),
                      const SizedBox(height: 8),
                      _PanelField(
                          controller: _maxLenCtrl,
                          label: 'Max length',
                          hint: 'e.g. 255',
                          keyboardType: TextInputType.number),
                      const SizedBox(height: 8),
                    ],
                    if (_type == 'integer' || _type == 'float') ...[
                      _PanelField(
                          controller: _minCtrl,
                          label: 'Min value',
                          hint: 'e.g. 0',
                          keyboardType: const TextInputType.numberWithOptions(
                              decimal: true, signed: true)),
                      const SizedBox(height: 8),
                      _PanelField(
                          controller: _maxCtrl,
                          label: 'Max value',
                          hint: 'e.g. 100',
                          keyboardType: const TextInputType.numberWithOptions(
                              decimal: true, signed: true)),
                      const SizedBox(height: 8),
                    ],
                    if (_type == 'string' || _type == 'email') ...[
                      _PanelField(
                          controller: _patternCtrl,
                          label: 'Regex pattern',
                          hint: r'e.g. ^[a-zA-Z]+$'),
                      const SizedBox(height: 8),
                      _PanelField(
                          controller: _validationMsgCtrl,
                          label: 'Custom error message',
                          hint: 'e.g. Invalid format'),
                      const SizedBox(height: 8),
                    ],
                  ],
                ],
              ),
            ),
          ),
          // Actions
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              border: Border(
                  top: BorderSide(color: _cs.border)),
            ),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                  onPressed: widget.onClose,
                  style: TextButton.styleFrom(
                      foregroundColor: _cs.textMuted),
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
      // Build validation object if any rules are set
      final validation = <String, dynamic>{};
      if (_type == 'string') {
        final minLen = int.tryParse(_minLenCtrl.text.trim());
        final maxLen = int.tryParse(_maxLenCtrl.text.trim());
        if (minLen != null) validation['minLength'] = minLen;
        if (maxLen != null) validation['maxLength'] = maxLen;
      }
      if (_type == 'integer' || _type == 'float') {
        final minVal = double.tryParse(_minCtrl.text.trim());
        final maxVal = double.tryParse(_maxCtrl.text.trim());
        if (minVal != null) validation['min'] = minVal;
        if (maxVal != null) validation['max'] = maxVal;
      }
      final pattern = _patternCtrl.text.trim();
      if (pattern.isNotEmpty) validation['pattern'] = pattern;
      final msg = _validationMsgCtrl.text.trim();
      if (msg.isNotEmpty) validation['message'] = msg;
      if (validation.isNotEmpty) data['validation'] = validation;

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
  final void Function(Map<String, dynamic> row) onEdit;

  const _RowsGrid({
    required this.displayKeys,
    required this.columns,
    required this.rows,
    required this.onDelete,
    required this.onEdit,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
      children: [
        // Header
        Container(
          padding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            border: Border(
                bottom:
                    BorderSide(color: cs.border)),
          ),
          child: Row(
            children: [
              ...displayKeys.map((k) => Expanded(
                    child: Row(
                      children: [
                        if (k == '\$id')
                          Icon(LucideIcons.key,
                              size: 12, color: cs.textSubtle)
                        else
                          _ColumnTypeIcon(
                              type: _typeForKey(k)),
                        const SizedBox(width: 6),
                        Expanded(
                          child: Text(k,
                              style: TextStyle(
                                  color: cs.textMuted,
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
                          child: k == '\$id'
                              ? IdText(id: val, fontSize: 12)
                              : Text(val,
                                  style: TextStyle(
                                      color: cs.textPrimary,
                                      fontSize: 12,
                                      fontFamily: 'monospace'),
                                  overflow: TextOverflow.ellipsis),
                        ),
                      );
                    }),
                    SizedBox(
                      width: 40,
                      child: PopupMenuButton<String>(
                        color: cs.popupSurface,
                        shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(8)),
                        iconSize: 16,
                        icon: Icon(LucideIcons.moreHorizontal,
                            size: 14, color: cs.textSubtle),
                        onSelected: (v) async {
                          if (v == 'edit') onEdit(row);
                          if (v == 'delete') {
                            final cs2 = consoleColors(context);
                            final confirmed = await showAppDialog<bool>(
                              context: context,
                              title: 'Delete row',
                              content: Text(
                                'Are you sure? This action cannot be undone.',
                                style: TextStyle(color: cs2.textSecondary),
                              ),
                              actions: [
                                const AppDialogCancel(),
                                AppDialogAction(
                                  label: 'Delete',
                                  destructive: true,
                                  onTap: () => Navigator.of(
                                    context,
                                    rootNavigator: true,
                                  ).pop(true),
                                ),
                              ],
                            );
                            if (confirmed == true) await onDelete(rowId);
                          }
                        },
                        itemBuilder: (bCtx) => [
                          PopupMenuItem(
                            value: 'edit',
                            child: Text('Edit',
                                style: TextStyle(
                                    color:
                                        consoleColors(bCtx).textPrimary,
                                    fontSize: 13)),
                          ),
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

class _SQLResultsGrid extends StatelessWidget {
  final List<String> columns;
  final List<Map<String, dynamic>> rows;

  const _SQLResultsGrid({required this.columns, required this.rows});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final displayColumns = columns.isEmpty && rows.isNotEmpty
        ? rows.first.keys.toList()
        : columns;

    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: Container(
        decoration: BoxDecoration(
          border: Border.all(color: cs.border),
          borderRadius: BorderRadius.circular(8),
        ),
        child: SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: SizedBox(
            width: displayColumns.length * 220,
            child: Column(
              children: [
                Container(
                  color: cs.fill,
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  child: Row(
                    children: displayColumns
                        .map(
                          (column) => SizedBox(
                            width: 220,
                            child: Text(
                              column,
                              style: TextStyle(
                                color: cs.textMuted,
                                fontSize: 12,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                        )
                        .toList(),
                  ),
                ),
                Expanded(
                  child: ListView.builder(
                    itemCount: rows.length,
                    itemBuilder: (_, index) {
                      final row = rows[index];
                      return Container(
                        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                        decoration: BoxDecoration(
                          border: Border(
                            top: BorderSide(color: cs.border),
                          ),
                        ),
                        child: Row(
                          children: displayColumns
                              .map(
                                (column) => SizedBox(
                                  width: 220,
                                  child: Text(
                                    _formatSQLValue(row[column]),
                                    maxLines: 4,
                                    overflow: TextOverflow.ellipsis,
                                    style: TextStyle(
                                      color: cs.textPrimary,
                                      fontSize: 12,
                                      fontFamily: 'monospace',
                                      height: 1.4,
                                    ),
                                  ),
                                ),
                              )
                              .toList(),
                        ),
                      );
                    },
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _SQLSummaryView extends StatelessWidget {
  final Map<String, dynamic> result;

  const _SQLSummaryView({required this.result});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(LucideIcons.badgeCheck, size: 28, color: _green),
          const SizedBox(height: 12),
          Text(
            'Statement completed',
            style: TextStyle(
              color: cs.textPrimary,
              fontSize: 15,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            '${result['rowCount'] ?? 0} rows affected in ${result['executionMs'] ?? 0} ms.',
            style: TextStyle(
              color: cs.textMuted,
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }
}

class _SQLErrorView extends StatelessWidget {
  final String message;

  const _SQLErrorView({required this.message});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: _red.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _red.withValues(alpha: 0.25)),
      ),
      child: SelectableText(
        message,
        style: const TextStyle(
          color: _red,
          fontSize: 12,
          fontFamily: 'monospace',
          height: 1.45,
        ),
      ),
    );
  }
}

class _MetricPill extends StatelessWidget {
  final String label;

  const _MetricPill({required this.label});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: cs.badgeFill,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: cs.border),
      ),
      child: Text(
        label,
        style: TextStyle(color: cs.textPrimary, fontSize: 12),
      ),
    );
  }
}

String _formatSQLValue(Object? value) {
  if (value == null) {
    return 'NULL';
  }
  if (value is Map || value is List) {
    return const JsonEncoder.withIndent('  ').convert(value);
  }
  return value.toString();
}

String _buildCSVExport(
  List<String> columns,
  List<Map<String, dynamic>> rows,
) {
  String escapeCell(Object? value) {
    final normalized = value == null
        ? ''
        : value is Map || value is List
            ? json.encode(value)
            : value.toString();
    final escaped = normalized.replaceAll('"', '""');
    return '"$escaped"';
  }

  final buffer = StringBuffer()
    ..writeln(columns.map(escapeCell).join(','));

  for (final row in rows) {
    buffer.writeln(
      columns.map((column) => escapeCell(row[column])).join(','),
    );
  }

  return buffer.toString();
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
    final cs = consoleColors(context);
    return Row(
      children: [
        GestureDetector(
          onTap: onBack,
          child: MouseRegion(
            cursor: SystemMouseCursors.click,
            child: Icon(LucideIcons.arrowLeft,
                size: 20, color: cs.textMuted),
          ),
        ),
        const SizedBox(width: 12),
        Text(title,
            style: TextStyle(
                color: cs.textPrimary,
                fontSize: 22,
                fontWeight: FontWeight.w600)),
        const SizedBox(width: 10),
        Icon(icon, size: 13, color: cs.textSubtle),
        const SizedBox(width: 6),
        IdText(id: subtitle),
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
    final cs = consoleColors(context);
    return SizedBox(
      width: 280,
      child: TextField(
        controller: controller,
        onChanged: onChanged,
        style: TextStyle(fontSize: 13, color: cs.textPrimary),
        decoration: InputDecoration(
          hintText: hint,
          hintStyle: TextStyle(color: cs.textSubtle, fontSize: 13),
          prefixIcon: Padding(
            padding: const EdgeInsets.only(left: 10, right: 6),
            child: Icon(Icons.search, size: 16, color: cs.textSubtle),
          ),
          prefixIconConstraints:
              const BoxConstraints(minWidth: 32, minHeight: 0),
          filled: true,
          fillColor: cs.fieldFill,
          isDense: true,
          contentPadding:
              const EdgeInsets.symmetric(vertical: 10, horizontal: 12),
          border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide:
                  BorderSide(color: cs.fieldBorder)),
          enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide:
                  BorderSide(color: cs.fieldBorder)),
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
  final VoidCallback? onTap;

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
    final cs = consoleColors(context);
    return OutlinedButton.icon(
      style: OutlinedButton.styleFrom(
        foregroundColor: cs.textSecondary,
        side: BorderSide(color: cs.border),
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
    final cs = consoleColors(context);
    return Column(
      children: [
        Container(
          padding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            border: Border(
                bottom:
                    BorderSide(color: cs.border)),
          ),
          child: Row(
            children: [
              for (var i = 0; i < headers.length; i++)
                Expanded(
                  flex: flexes[i],
                  child: Text(headers[i],
                      style: TextStyle(
                          color: cs.textMuted,
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
                            onTap: () async {
                              final cs = consoleColors(context);
                              final confirmed = await showAppDialog<bool>(
                                context: context,
                                title: 'Delete item',
                                content: Text(
                                  'Are you sure? This action cannot be undone.',
                                  style: TextStyle(color: cs.textSecondary),
                                ),
                                actions: [
                                  const AppDialogCancel(),
                                  AppDialogAction(
                                    label: 'Delete',
                                    destructive: true,
                                    onTap: () => Navigator.of(
                                      context,
                                      rootNavigator: true,
                                    ).pop(true),
                                  ),
                                ],
                              );
                              if (confirmed == true) await onRowDelete!(row);
                            }),
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
  late ConsoleColors _cs;
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
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
            color: _hovered ? _cs.fillHover : null,
            border: Border(
                bottom:
                    BorderSide(color: _cs.border)),
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
  late ConsoleColors _cs;
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Icon(LucideIcons.trash2,
            size: 14,
            color:
                _hovered ? _red : _cs.textSubtle),
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
    final cs = consoleColors(context);
    return Icon(icon, size: 12, color: cs.textSubtle);
  }
}

class _SettingsCard extends StatelessWidget {
  final String title;
  final List<Widget> children;

  const _SettingsCard({required this.title, required this.children});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title,
              style: TextStyle(
                  color: cs.textPrimary,
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

  const _SettingsSectionCard({
    this.title,
    this.subtitle,
    required this.children,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (title != null) ...[
            Text(title!,
                style: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 15,
                    fontWeight: FontWeight.w600)),
            if (subtitle != null) ...[
              const SizedBox(height: 4),
              Text(subtitle!,
                  style: TextStyle(
                      color: cs.textMuted,
                      fontSize: 13)),
            ],
            const SizedBox(height: 16),
          ],
          ...children,
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
  late ConsoleColors _cs;
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
    _cs = consoleColors(context);
    return Row(
      children: [
        SizedBox(
          width: 80,
          child: Text('Name',
              style: TextStyle(
                  color: _cs.textMuted,
                  fontSize: 12,
                  fontWeight: FontWeight.w500)),
        ),
        Expanded(
          child: TextField(
            controller: _ctrl,
            style: TextStyle(color: _cs.textPrimary, fontSize: 13),
            decoration: InputDecoration(
              filled: true,
              fillColor: _cs.fieldFill,
              isDense: true,
              contentPadding: const EdgeInsets.symmetric(
                  horizontal: 12, vertical: 10),
              border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide:
                      BorderSide(color: _cs.fieldBorder)),
              enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide:
                      BorderSide(color: _cs.fieldBorder)),
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
  final Future<void> Function() onDelete;

  const _DangerCard({
    required this.title,
    required this.description,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _red.withValues(alpha: 0.3)),
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
                        color: cs.textMuted,
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
            onPressed: () async {
              final cs = consoleColors(context);
              final confirmed = await showAppDialog<bool>(
                context: context,
                title: 'Confirm delete',
                content: Text(
                  description,
                  style: TextStyle(color: cs.textSecondary),
                ),
                actions: [
                  const AppDialogCancel(),
                  AppDialogAction(
                    label: 'Delete',
                    destructive: true,
                    onTap: () =>
                        Navigator.of(context, rootNavigator: true).pop(true),
                  ),
                ],
              );
              if (confirmed == true) await onDelete();
            },
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
  final bool isId;
  const _InfoRow({required this.label, required this.value, this.isId = false});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          SizedBox(
            width: 120,
            child: Text(label,
                style: TextStyle(color: cs.textMuted, fontSize: 13)),
          ),
          Expanded(
            child: isId
                ? IdText(id: value)
                : SelectableText(value,
                    style: TextStyle(
                        color: cs.textSecondary,
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
    final cs = consoleColors(context);
    return Center(
      child: Text('$label — coming soon',
          style: TextStyle(color: cs.textMuted, fontSize: 14)),
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
    final cs = consoleColors(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: cs.fill,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(icon, size: 22, color: cs.textSubtle),
          ),
          const SizedBox(height: 16),
          Text(title,
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          Text(subtitle,
              style: TextStyle(color: cs.textMuted, fontSize: 13),
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
  final TextInputType? keyboardType;

  const _PanelField(
      {required this.controller, required this.label, required this.hint, this.keyboardType});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _PanelLabel(label),
        const SizedBox(height: 6),
        TextField(
          controller: controller,
          keyboardType: keyboardType,
          style: TextStyle(color: cs.textPrimary, fontSize: 13),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: TextStyle(
                color: cs.textSubtle, fontSize: 13),
            filled: true,
            fillColor: cs.fieldFill,
            isDense: true,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: BorderSide(color: cs.fieldBorder)),
            enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: BorderSide(color: cs.fieldBorder)),
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
    final cs = consoleColors(context);
    return Text(text,
        style: TextStyle(
            color: cs.textMuted,
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
    final cs = consoleColors(context);
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
                side: BorderSide(color: cs.border),
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
                      style: TextStyle(
                          color: cs.textPrimary,
                          fontSize: 13,
                          fontWeight: FontWeight.w500)),
                  Text(subtitle,
                      style: TextStyle(
                          color: cs.textSubtle,
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

class _PermChip extends StatelessWidget {
  final String label;
  final Color color;
  const _PermChip({required this.label, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withAlpha(30),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(label,
          style: TextStyle(
              color: color,
              fontSize: 10,
              fontWeight: FontWeight.w500)),
    );
  }
}

class _PermToggleRow extends StatelessWidget {
  final String label;
  final String subtitle;
  final bool value;
  final ValueChanged<bool> onChanged;

  const _PermToggleRow({
    required this.label,
    required this.subtitle,
    required this.value,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Row(
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(label,
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 13,
                      fontWeight: FontWeight.w500)),
              Text(subtitle,
                  style: TextStyle(
                      color: cs.textSubtle, fontSize: 11)),
            ],
          ),
        ),
        Switch(
          value: value,
          onChanged: onChanged,
          activeThumbColor: const Color(0xFF6C47FF),
        ),
      ],
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
