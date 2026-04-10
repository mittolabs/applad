import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/theme/console_colors.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';

// --- Colors ----------------------------------------------------------------

const _accent = Color(0xFF3472A4);

// --- Providers -------------------------------------------------------------

final _flagSearchProvider = StateProvider<String>((ref) => '');
final _flagPerPageProvider = StateProvider<int>((ref) => 12);
final _flagPageProvider = StateProvider<int>((ref) => 1);
final _selectedFlagProvider = StateProvider<Map<String, dynamic>?>((ref) => null);
final _detailTabProvider = StateProvider<int>((ref) => 0);

final flagsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final search = ref.watch(_flagSearchProvider);
  final limit = ref.watch(_flagPerPageProvider);
  final page = ref.watch(_flagPageProvider);
  final offset = (page - 1) * limit;
  final params = <String, dynamic>{'limit': limit, 'offset': offset};
  if (search.isNotEmpty) params['search'] = search;
  final res = await api.get('/flags', params: params);
  return res.data as Map<String, dynamic>;
});

final _flagRulesProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, flagId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/flags/$flagId/rules');
  return res.data as Map<String, dynamic>;
});

final _flagOverridesProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, flagId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/flags/$flagId/overrides');
  return res.data as Map<String, dynamic>;
});

final _flagStatsProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, flagId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/flags/$flagId/stats');
  return res.data as Map<String, dynamic>;
});

// --- Page ------------------------------------------------------------------

class FlagsPage extends ConsumerStatefulWidget {
  const FlagsPage({super.key});

  @override
  ConsumerState<FlagsPage> createState() => _FlagsPageState();
}

class _FlagsPageState extends ConsumerState<FlagsPage> {
  final _searchCtrl = TextEditingController();
  late ConsoleColors _cs;

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  void _doSearch() {
    ref.read(_flagSearchProvider.notifier).state = _searchCtrl.text.trim();
    ref.read(_flagPageProvider.notifier).state = 1;
  }

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
    final selected = ref.watch(_selectedFlagProvider);

    return Scaffold(
      backgroundColor: _cs.background,
      body: selected != null
          ? _FlagDetailView(
              flag: selected,
              onBack: () =>
                  ref.read(_selectedFlagProvider.notifier).state = null,
            )
          : _buildFlagList(),
    );
  }

  Widget _buildFlagList() {
    final flagsAsync = ref.watch(flagsProvider);
    final perPage = ref.watch(_flagPerPageProvider);
    final currentPage = ref.watch(_flagPageProvider);

    return Padding(
      padding: EdgeInsets.symmetric(
        horizontal: MediaQuery.of(context).size.width > 1400 ? 80 : 40,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SizedBox(height: 32),
          Text(
            'Feature Flags',
            style: TextStyle(
              color: _cs.textPrimary,
              fontSize: 22,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 24),
          Expanded(
            child: flagsAsync.when(
              loading: () =>
                  const Center(child: CircularProgressIndicator(strokeWidth: 2)),
              error: (e, _) => Center(
                child: Text('Error: $e',
                    style: const TextStyle(color: Color(0xFFEF4444))),
              ),
              data: (data) {
                final flags =
                    List<Map<String, dynamic>>.from(data['flags'] ?? []);
                final total = data['total'] as int? ?? flags.length;
                return AppDataTable(
                  columns: const [
                    AppTableColumn(
                        key: r'$id', label: 'ID', flex: 3, defaultVisible: false),
                    AppTableColumn(key: 'key',     label: 'Key',     flex: 3),
                    AppTableColumn(key: 'name',    label: 'Name',    flex: 3),
                    AppTableColumn(
                        key: 'type', label: 'Type', flex: 2, sortable: false),
                    AppTableColumn(
                        key: 'enabled', label: 'Enabled', flex: 2, sortable: false),
                  ],
                  rows: flags,
                  getCellValue: (row, key) => switch (key) {
                    r'$id'    => row[r'$id'] as String? ?? '',
                    'key'     => row['key'] as String? ?? '',
                    'name'    => row['name'] as String? ?? '',
                    'type'    => row['type'] as String? ?? 'boolean',
                    'enabled' => (row['enabled'] == true) ? 'enabled' : 'disabled',
                    _         => '',
                  },
                  cellBuilder: (row, key) {
                    if (key == 'enabled') {
                      return Switch(
                        value: row['enabled'] == true,
                        onChanged: (_) => _toggleFlag(ref, row),
                        activeColor: _accent,
                      );
                    }
                    if (key == 'type') {
                      return _TypeBadge(
                          type: row['type'] as String? ?? 'boolean');
                    }
                    return null;
                  },
                  getRowIcon: (_) => LucideIcons.flag,
                  getRowIconColor: (row) => row['enabled'] == true
                      ? const Color(0xFF22C55E)
                      : const Color(0xFF6B7280),
                  onRowTap: (row) =>
                      ref.read(_selectedFlagProvider.notifier).state = row,
                  onDeleteRow: (row) => _deleteFlag(context, ref, row),
                  createLabel: 'Create flag',
                  onCreateTap: () => _showCreateFlagDialog(context, ref),
                  total: total,
                  perPage: perPage,
                  currentPage: currentPage,
                  onPrev: () =>
                      ref.read(_flagPageProvider.notifier).update((s) => s - 1),
                  onNext: () =>
                      ref.read(_flagPageProvider.notifier).update((s) => s + 1),
                  onPerPageChanged: (v) {
                    ref.read(_flagPerPageProvider.notifier).state = v;
                    ref.read(_flagPageProvider.notifier).state = 1;
                  },
                  itemLabel: 'flags',
                  searchController: _searchCtrl,
                  onSearch: _doSearch,
                  searchHint: 'Search by key or name',
                  emptyIcon: LucideIcons.flag,
                  emptyTitle: 'No feature flags',
                  emptySubtitle:
                      'Create a flag to start managing feature rollouts.',
                  filters: const [
                    AppTableFilter(
                        key: 'type',
                        label: 'Type',
                        options: ['boolean', 'string', 'number', 'json']),
                    AppTableFilter(
                        key: 'enabled',
                        label: 'Status',
                        options: ['enabled', 'disabled']),
                  ],
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _toggleFlag(WidgetRef ref, Map<String, dynamic> flag) async {
    final api = ref.read(apiClientProvider);
    await api.patch('/flags/${flag['\$id']}/toggle');
    ref.invalidate(flagsProvider);
  }

  void _showCreateFlagDialog(BuildContext context, WidgetRef ref) {
    final keyCtrl = TextEditingController();
    final nameCtrl = TextEditingController();
    final descCtrl = TextEditingController();
    final defaultValCtrl = TextEditingController();
    String selectedType = 'boolean';

    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) {
          final cs = consoleColors(ctx);
          return Center(
            child: Material(
              color: Colors.transparent,
              child: Container(
                width: 480,
                constraints: const BoxConstraints(maxHeight: 600),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: cs.border),
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
                          Expanded(
                            child: Text('Create Flag',
                                style: TextStyle(
                                  color: cs.textPrimary,
                                  fontSize: 16,
                                  fontWeight: FontWeight.w600,
                                )),
                          ),
                          GestureDetector(
                            onTap: () => Navigator.of(ctx).pop(),
                            child: Icon(LucideIcons.x,
                                size: 16,
                                color: cs.textSubtle),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 16),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 20),
                      child: Container(
                          height: 1, color: cs.border),
                    ),
                    const SizedBox(height: 16),
                    Flexible(
                      child: SingleChildScrollView(
                        padding: const EdgeInsets.symmetric(horizontal: 20),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            AppDialogField(
                              controller: keyCtrl,
                              label: 'Key',
                              hint: 'e.g. enable_dark_mode',
                              autofocus: true,
                            ),
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: nameCtrl,
                              label: 'Name',
                              hint: 'Dark Mode',
                            ),
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: descCtrl,
                              label: 'Description',
                              hint: 'Controls whether dark mode is available',
                              maxLines: 3,
                            ),
                            const SizedBox(height: 12),
                            _buildDropdown(
                              cs: cs,
                              label: 'Type',
                              value: selectedType,
                              items: const [
                                'boolean',
                                'string',
                                'number',
                                'json'
                              ],
                              onChanged: (v) =>
                                  setDialogState(() => selectedType = v!),
                            ),
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: defaultValCtrl,
                              label: 'Default Value',
                              hint: selectedType == 'boolean'
                                  ? 'true or false'
                                  : selectedType == 'number'
                                      ? '0'
                                      : selectedType == 'json'
                                          ? '{}'
                                          : 'value',
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
                              await api.post('/flags', data: {
                                'key': keyCtrl.text,
                                'name': nameCtrl.text,
                                'description': descCtrl.text,
                                'type': selectedType,
                                'defaultValue': defaultValCtrl.text,
                              });
                              if (ctx.mounted) Navigator.pop(ctx);
                              ref.invalidate(flagsProvider);
                            },
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  void _showEditFlagDialog(
      BuildContext context, WidgetRef ref, Map<String, dynamic> flag) {
    final keyCtrl = TextEditingController(text: flag['key'] ?? '');
    final nameCtrl = TextEditingController(text: flag['name'] ?? '');
    final descCtrl = TextEditingController(text: flag['description'] ?? '');
    final defaultValCtrl =
        TextEditingController(text: '${flag['defaultValue'] ?? ''}');
    String selectedType = flag['type'] ?? 'boolean';

    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) {
          final cs = consoleColors(ctx);
          return Center(
            child: Material(
              color: Colors.transparent,
              child: Container(
                width: 480,
                constraints: const BoxConstraints(maxHeight: 600),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: cs.border),
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
                          Expanded(
                            child: Text('Edit Flag',
                                style: TextStyle(
                                  color: cs.textPrimary,
                                  fontSize: 16,
                                  fontWeight: FontWeight.w600,
                                )),
                          ),
                          GestureDetector(
                            onTap: () => Navigator.of(ctx).pop(),
                            child: Icon(LucideIcons.x,
                                size: 16,
                                color: cs.textSubtle),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 16),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 20),
                      child: Container(
                          height: 1, color: cs.border),
                    ),
                    const SizedBox(height: 16),
                    Flexible(
                      child: SingleChildScrollView(
                        padding: const EdgeInsets.symmetric(horizontal: 20),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            AppDialogField(
                              controller: keyCtrl,
                              label: 'Key',
                              hint: 'e.g. enable_dark_mode',
                              autofocus: true,
                            ),
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: nameCtrl,
                              label: 'Name',
                              hint: 'Dark Mode',
                            ),
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: descCtrl,
                              label: 'Description',
                              hint: 'Controls whether dark mode is available',
                              maxLines: 3,
                            ),
                            const SizedBox(height: 12),
                            _buildDropdown(
                              cs: cs,
                              label: 'Type',
                              value: selectedType,
                              items: const [
                                'boolean',
                                'string',
                                'number',
                                'json'
                              ],
                              onChanged: (v) =>
                                  setDialogState(() => selectedType = v!),
                            ),
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: defaultValCtrl,
                              label: 'Default Value',
                              hint: selectedType == 'boolean'
                                  ? 'true or false'
                                  : selectedType == 'number'
                                      ? '0'
                                      : selectedType == 'json'
                                          ? '{}'
                                          : 'value',
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
                              await api.put('/flags/${flag['\$id']}', data: {
                                'key': keyCtrl.text,
                                'name': nameCtrl.text,
                                'description': descCtrl.text,
                                'type': selectedType,
                                'defaultValue': defaultValCtrl.text,
                              });
                              if (ctx.mounted) Navigator.pop(ctx);
                              ref.invalidate(flagsProvider);
                            },
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  Future<void> _deleteFlag(
      BuildContext context, WidgetRef ref, Map<String, dynamic> flag) async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete Flag',
      content: Text(
        'Are you sure you want to delete "${flag['name'] ?? flag['key']}"? '
        'This action cannot be undone.',
        style: TextStyle(color: _cs.textSecondary),
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
      final api = ref.read(apiClientProvider);
      await api.delete('/flags/${flag['\$id']}');
      ref.invalidate(flagsProvider);
    }
  }
}

// --- Shared dropdown builder -----------------------------------------------

Widget _buildDropdown({
  required ConsoleColors cs,
  required String label,
  required String value,
  required List<String> items,
  required ValueChanged<String?> onChanged,
}) {
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Text(label,
          style: TextStyle(
              color: cs.textMuted,
              fontSize: 12,
              fontWeight: FontWeight.w500)),
      const SizedBox(height: 6),
      DropdownButtonFormField<String>(
        value: value,
        dropdownColor: cs.surface,
        style: TextStyle(color: cs.textPrimary, fontSize: 13),
        decoration: InputDecoration(
          filled: true,
          fillColor: cs.fieldFill,
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: cs.fieldBorder),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: cs.fieldBorder),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: _accent),
          ),
          contentPadding:
              const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        ),
        items: items
            .map((t) => DropdownMenuItem(value: t, child: Text(t)))
            .toList(),
        onChanged: onChanged,
      ),
    ],
  );
}

// --- Type badge ------------------------------------------------------------

class _TypeBadge extends StatelessWidget {
  final String type;
  const _TypeBadge({required this.type});

  @override
  Widget build(BuildContext context) {
    IconData icon;
    switch (type) {
      case 'boolean':
        icon = LucideIcons.toggleLeft;
      case 'string':
        icon = LucideIcons.type;
      case 'number':
        icon = LucideIcons.hash;
      case 'json':
        icon = LucideIcons.braces;
      default:
        icon = LucideIcons.helpCircle;
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: _accent.withOpacity(0.1),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: _accent.withOpacity(0.2)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 12, color: _accent),
          const SizedBox(width: 4),
          Text(type,
              style: const TextStyle(fontSize: 12, color: _accent)),
        ],
      ),
    );
  }
}

// --- Status chip -----------------------------------------------------------

class _StatusChip extends StatelessWidget {
  final bool enabled;
  const _StatusChip({required this.enabled});

  @override
  Widget build(BuildContext context) {
    final color = enabled ? const Color(0xFF22C55E) : const Color(0xFFEF4444);
    final label = enabled ? 'Enabled' : 'Disabled';
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: color.withOpacity(0.3)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
          const SizedBox(width: 6),
          Text(label, style: TextStyle(fontSize: 12, color: color)),
        ],
      ),
    );
  }
}

// --- Flag detail view ------------------------------------------------------

class _FlagDetailView extends ConsumerWidget {
  final Map<String, dynamic> flag;
  final VoidCallback onBack;

  const _FlagDetailView({required this.flag, required this.onBack});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cs = consoleColors(context);
    final tab = ref.watch(_detailTabProvider);
    final flagId = flag['\$id'] as String;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Header with back button
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 20, 24, 0),
          child: Row(
            children: [
              GestureDetector(
                onTap: onBack,
                child: MouseRegion(
                  cursor: SystemMouseCursors.click,
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(LucideIcons.arrowLeft,
                          size: 16, color: cs.textMuted),
                      const SizedBox(width: 8),
                      Text('Feature Flags',
                          style: TextStyle(
                              color: cs.textMuted,
                              fontSize: 14)),
                    ],
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Icon(LucideIcons.chevronRight,
                  size: 14, color: cs.textSubtle),
              const SizedBox(width: 12),
              Expanded(
                child: Text(flag['name'] ?? flag['key'] ?? '',
                    style: Theme.of(context)
                        .textTheme
                        .headlineSmall
                        ?.copyWith(color: cs.textPrimary)),
              ),
              _StatusChip(enabled: flag['enabled'] == true),
            ],
          ),
        ),
        const SizedBox(height: 4),
        // Key + type info
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24),
          child: Row(
            children: [
              Icon(LucideIcons.key, size: 12, color: cs.textSubtle),
              const SizedBox(width: 6),
              Text(flag['key'] ?? '',
                  style: TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                      color: cs.textMuted)),
              const SizedBox(width: 16),
              _TypeBadge(type: flag['type'] ?? 'boolean'),
            ],
          ),
        ),
        const SizedBox(height: 12),
        // Tabs
        Padding(
          padding: const EdgeInsets.only(left: 24),
          child: PageTabs(
            tabs: const ['Settings', 'Rules', 'Overrides', 'Stats'],
            selected: tab,
            onChanged: (i) =>
                ref.read(_detailTabProvider.notifier).state = i,
          ),
        ),
        // Tab body
        Expanded(
          child: _detailTabBody(context, ref, tab, flagId),
        ),
      ],
    );
  }

  Widget _detailTabBody(
      BuildContext context, WidgetRef ref, int tab, String flagId) {
    switch (tab) {
      case 0:
        return _SettingsTab(flag: flag);
      case 1:
        return _RulesTab(flagId: flagId);
      case 2:
        return _OverridesTab(flagId: flagId);
      case 3:
        return _StatsTab(flagId: flagId);
      default:
        return const SizedBox.shrink();
    }
  }
}

// --- Settings tab ----------------------------------------------------------

class _SettingsTab extends StatelessWidget {
  final Map<String, dynamic> flag;
  const _SettingsTab({required this.flag});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: cs.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: cs.border),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _settingRow(cs, 'Key', flag['key'] ?? '', mono: true),
            Divider(color: cs.border, height: 24),
            _settingRow(cs, 'Name', flag['name'] ?? ''),
            Divider(color: cs.border, height: 24),
            _settingRow(cs, 'Description', flag['description'] ?? 'No description'),
            Divider(color: cs.border, height: 24),
            _settingRow(cs, 'Type', flag['type'] ?? 'boolean'),
            Divider(color: cs.border, height: 24),
            _settingRow(cs, 'Default Value', '${flag['defaultValue'] ?? 'N/A'}',
                mono: true),
            Divider(color: cs.border, height: 24),
            _settingRow(
                cs, 'Status', flag['enabled'] == true ? 'Enabled' : 'Disabled'),
            Divider(color: cs.border, height: 24),
            _settingRow(cs, 'Created', _formatTimestamp(flag['\$createdAt'])),
            Divider(color: cs.border, height: 24),
            _settingRow(cs, 'Updated', _formatTimestamp(flag['\$updatedAt'])),
          ],
        ),
      ),
    );
  }

  Widget _settingRow(ConsoleColors cs, String label, String value, {bool mono = false}) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 140,
          child: Text(label,
              style: TextStyle(
                  color: cs.textMuted, fontSize: 13, fontWeight: FontWeight.w500)),
        ),
        Expanded(
          child: SelectableText(
            value,
            style: TextStyle(
              color: cs.textPrimary,
              fontSize: 13,
              fontFamily: mono ? 'monospace' : null,
            ),
          ),
        ),
      ],
    );
  }

  String _formatTimestamp(dynamic ts) {
    if (ts == null) return 'N/A';
    final str = ts.toString();
    if (str.isEmpty) return 'N/A';
    try {
      final dt = DateTime.parse(str);
      return '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')} '
          '${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
    } catch (_) {
      return str;
    }
  }
}

// --- Rules tab -------------------------------------------------------------

class _RulesTab extends ConsumerWidget {
  final String flagId;
  const _RulesTab({required this.flagId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cs = consoleColors(context);
    final rulesAsync = ref.watch(_flagRulesProvider(flagId));

    return Column(
      children: [
        // Header
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 8),
          child: Row(
            children: [
              Icon(LucideIcons.gitBranch, size: 16, color: cs.textMuted),
              const SizedBox(width: 8),
              Text('Targeting Rules',
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
              const Spacer(),
              FilledButton.icon(
                style: FilledButton.styleFrom(
                  backgroundColor: _accent,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding:
                      const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                ),
                onPressed: () => _showAddRuleDialog(context, ref),
                icon: const Icon(LucideIcons.plus, size: 14),
                label: const Text('Add Rule', style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
        ),
        Divider(height: 1, color: cs.border),
        // Rules list
        Expanded(
          child: rulesAsync.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text('Failed to load rules: $e',
                      style: TextStyle(color: cs.textMuted)),
                  const SizedBox(height: 8),
                  FilledButton(
                    onPressed: () =>
                        ref.invalidate(_flagRulesProvider(flagId)),
                    child: const Text('Retry'),
                  ),
                ],
              ),
            ),
            data: (data) {
              final rules =
                  List<Map<String, dynamic>>.from(data['rules'] ?? []);
              if (rules.isEmpty) {
                return Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(LucideIcons.gitBranch, size: 48, color: cs.textSubtle),
                      const SizedBox(height: 16),
                      Text('No rules defined',
                          style: TextStyle(color: cs.textMuted)),
                      const SizedBox(height: 4),
                      Text('All users will receive the default value.',
                          style: TextStyle(color: cs.textSubtle, fontSize: 12)),
                    ],
                  ),
                );
              }
              return ListView.builder(
                padding: const EdgeInsets.all(16),
                itemCount: rules.length,
                itemBuilder: (context, i) =>
                    _RuleCard(flagId: flagId, rule: rules[i]),
              );
            },
          ),
        ),
      ],
    );
  }

  void _showAddRuleDialog(BuildContext context, WidgetRef ref) {
    final valueCtrl = TextEditingController();
    final conditionsCtrl = TextEditingController();
    String selectedType = 'percentage';
    double rollout = 100;

    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) {
          final cs = consoleColors(ctx);
          return Center(
            child: Material(
              color: Colors.transparent,
              child: Container(
                width: 480,
                constraints: const BoxConstraints(maxHeight: 560),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: cs.border),
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
                          Expanded(
                            child: Text('Add Rule',
                                style: TextStyle(
                                  color: cs.textPrimary,
                                  fontSize: 16,
                                  fontWeight: FontWeight.w600,
                                )),
                          ),
                          GestureDetector(
                            onTap: () => Navigator.of(ctx).pop(),
                            child: Icon(LucideIcons.x,
                                size: 16,
                                color: cs.textSubtle),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 16),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 20),
                      child: Container(
                          height: 1, color: cs.border),
                    ),
                    const SizedBox(height: 16),
                    Flexible(
                      child: SingleChildScrollView(
                        padding: const EdgeInsets.symmetric(horizontal: 20),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            _buildDropdown(
                              cs: cs,
                              label: 'Rule Type',
                              value: selectedType,
                              items: const [
                                'percentage',
                                'attribute',
                                'segment',
                                'schedule'
                              ],
                              onChanged: (v) =>
                                  setDialogState(() => selectedType = v!),
                            ),
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: conditionsCtrl,
                              label: 'Conditions (JSON)',
                              hint:
                                  '{"attribute": "country", "operator": "eq", "value": "US"}',
                              maxLines: 3,
                            ),
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: valueCtrl,
                              label: 'Value',
                              hint: 'Value when rule matches',
                            ),
                            const SizedBox(height: 16),
                            Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text('Rollout Percentage',
                                    style: TextStyle(
                                        color: cs.textMuted,
                                        fontSize: 12,
                                        fontWeight: FontWeight.w500)),
                                const SizedBox(height: 8),
                                Row(
                                  children: [
                                    Expanded(
                                      child: SliderTheme(
                                        data: SliderThemeData(
                                          activeTrackColor: _accent,
                                          inactiveTrackColor:
                                              _accent.withOpacity(0.2),
                                          thumbColor: _accent,
                                          overlayColor:
                                              _accent.withOpacity(0.12),
                                        ),
                                        child: Slider(
                                          value: rollout,
                                          min: 0,
                                          max: 100,
                                          divisions: 100,
                                          onChanged: (v) =>
                                              setDialogState(() => rollout = v),
                                        ),
                                      ),
                                    ),
                                    const SizedBox(width: 12),
                                    SizedBox(
                                      width: 48,
                                      child: Text(
                                        '${rollout.round()}%',
                                        style: TextStyle(
                                            color: cs.textPrimary,
                                            fontSize: 13,
                                            fontWeight: FontWeight.w500),
                                      ),
                                    ),
                                  ],
                                ),
                              ],
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
                            label: 'Add Rule',
                            onTap: () async {
                              final api = ref.read(apiClientProvider);
                              await api.post('/flags/$flagId/rules', data: {
                                'type': selectedType,
                                'conditions': conditionsCtrl.text,
                                'value': valueCtrl.text,
                                'rolloutPercentage': rollout.round(),
                              });
                              if (ctx.mounted) Navigator.pop(ctx);
                              ref.invalidate(_flagRulesProvider(flagId));
                            },
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}

// --- Rule card --------------------------------------------------------------

class _RuleCard extends ConsumerWidget {
  final String flagId;
  final Map<String, dynamic> rule;
  const _RuleCard({required this.flagId, required this.rule});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cs = consoleColors(context);
    final ruleType = rule['type'] ?? 'percentage';
    final enabled = rule['enabled'] != false;
    final rollout = (rule['rolloutPercentage'] ?? 100).toDouble();

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Top row: type badge, toggle, delete
          Row(
            children: [
              _RuleTypeBadge(type: ruleType),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  rule['conditions']?.toString() ?? 'No conditions',
                  style: TextStyle(
                      color: cs.textMuted, fontSize: 12, fontFamily: 'monospace'),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              Switch(
                value: enabled,
                activeColor: _accent,
                onChanged: (_) {
                  // Toggle would require a PATCH endpoint; placeholder
                },
              ),
              IconButton(
                icon: Icon(LucideIcons.trash2,
                    size: 16, color: cs.textMuted),
                tooltip: 'Delete rule',
                onPressed: () async {
                  final api = ref.read(apiClientProvider);
                  await api.delete('/flags/$flagId/rules/${rule['\$id']}');
                  ref.invalidate(_flagRulesProvider(flagId));
                },
              ),
            ],
          ),
          const SizedBox(height: 8),
          // Value
          Row(
            children: [
              Icon(LucideIcons.arrowRight, size: 12, color: cs.textSubtle),
              const SizedBox(width: 6),
              Text('Value: ',
                  style: TextStyle(color: cs.textMuted, fontSize: 12)),
              Text('${rule['value'] ?? 'N/A'}',
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 12,
                      fontFamily: 'monospace')),
            ],
          ),
          const SizedBox(height: 8),
          // Rollout bar
          Row(
            children: [
              Icon(LucideIcons.users, size: 12, color: cs.textSubtle),
              const SizedBox(width: 6),
              Text('Rollout: ${rollout.round()}%',
                  style: TextStyle(color: cs.textMuted, fontSize: 12)),
              const SizedBox(width: 12),
              Expanded(
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(3),
                  child: LinearProgressIndicator(
                    value: rollout / 100,
                    backgroundColor: _accent.withOpacity(0.1),
                    valueColor:
                        const AlwaysStoppedAnimation<Color>(_accent),
                    minHeight: 6,
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

// --- Rule type badge -------------------------------------------------------

class _RuleTypeBadge extends StatelessWidget {
  final String type;
  const _RuleTypeBadge({required this.type});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    Color badgeColor;
    IconData icon;
    switch (type) {
      case 'percentage':
        badgeColor = const Color(0xFF8B5CF6);
        icon = LucideIcons.percent;
      case 'attribute':
        badgeColor = const Color(0xFF0EA5E9);
        icon = LucideIcons.tag;
      case 'segment':
        badgeColor = const Color(0xFFF59E0B);
        icon = LucideIcons.users;
      case 'schedule':
        badgeColor = const Color(0xFF22C55E);
        icon = LucideIcons.clock;
      default:
        badgeColor = cs.textMuted;
        icon = LucideIcons.helpCircle;
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: badgeColor.withOpacity(0.1),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: badgeColor.withOpacity(0.3)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 12, color: badgeColor),
          const SizedBox(width: 4),
          Text(type,
              style: TextStyle(fontSize: 11, color: badgeColor)),
        ],
      ),
    );
  }
}

// --- Overrides tab ---------------------------------------------------------

class _OverridesTab extends ConsumerWidget {
  final String flagId;
  const _OverridesTab({required this.flagId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cs = consoleColors(context);
    final overridesAsync = ref.watch(_flagOverridesProvider(flagId));

    return Column(
      children: [
        // Header
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 8),
          child: Row(
            children: [
              Icon(LucideIcons.userCog, size: 16, color: cs.textMuted),
              const SizedBox(width: 8),
              Text('Target Overrides',
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
              const Spacer(),
              FilledButton.icon(
                style: FilledButton.styleFrom(
                  backgroundColor: _accent,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding:
                      const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                ),
                onPressed: () => _showAddOverrideDialog(context, ref),
                icon: const Icon(LucideIcons.plus, size: 14),
                label:
                    const Text('Add Override', style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
        ),
        Divider(height: 1, color: cs.border),
        // Overrides list
        Expanded(
          child: overridesAsync.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text('Failed to load overrides: $e',
                      style: TextStyle(color: cs.textMuted)),
                  const SizedBox(height: 8),
                  FilledButton(
                    onPressed: () =>
                        ref.invalidate(_flagOverridesProvider(flagId)),
                    child: const Text('Retry'),
                  ),
                ],
              ),
            ),
            data: (data) {
              final overrides =
                  List<Map<String, dynamic>>.from(data['overrides'] ?? []);
              if (overrides.isEmpty) {
                return Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(LucideIcons.userCog, size: 48, color: cs.textSubtle),
                      const SizedBox(height: 16),
                      Text('No overrides set',
                          style: TextStyle(color: cs.textMuted)),
                      const SizedBox(height: 4),
                      Text(
                          'Override flag values for specific users or teams.',
                          style:
                              TextStyle(color: cs.textSubtle, fontSize: 12)),
                    ],
                  ),
                );
              }
              return ListView.builder(
                padding: const EdgeInsets.all(16),
                itemCount: overrides.length,
                itemBuilder: (context, i) {
                  final o = overrides[i];
                  final targetType = o['targetType'] ?? 'user';
                  final isUser = targetType == 'user';
                  return Container(
                    margin: const EdgeInsets.only(bottom: 8),
                    padding: const EdgeInsets.symmetric(
                        horizontal: 16, vertical: 12),
                    decoration: BoxDecoration(
                      color: cs.surface,
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: cs.border),
                    ),
                    child: Row(
                      children: [
                        Icon(
                          isUser ? LucideIcons.user : LucideIcons.users,
                          size: 16,
                          color: cs.textMuted,
                        ),
                        const SizedBox(width: 10),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 6, vertical: 2),
                          decoration: BoxDecoration(
                            color: (isUser
                                    ? const Color(0xFF0EA5E9)
                                    : const Color(0xFFF59E0B))
                                .withOpacity(0.1),
                            borderRadius: BorderRadius.circular(4),
                          ),
                          child: Text(
                            targetType,
                            style: TextStyle(
                              fontSize: 11,
                              color: isUser
                                  ? const Color(0xFF0EA5E9)
                                  : const Color(0xFFF59E0B),
                            ),
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Text(
                            o['targetId'] ?? '',
                            style: TextStyle(
                              fontFamily: 'monospace',
                              fontSize: 12,
                              color: cs.textPrimary,
                            ),
                          ),
                        ),
                        Icon(LucideIcons.arrowRight,
                            size: 14, color: cs.textSubtle),
                        const SizedBox(width: 12),
                        Text(
                          '${o['value'] ?? 'N/A'}',
                          style: const TextStyle(
                            fontFamily: 'monospace',
                            fontSize: 12,
                            color: _accent,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                        const SizedBox(width: 12),
                        IconButton(
                          icon: Icon(LucideIcons.trash2,
                              size: 16, color: cs.textMuted),
                          tooltip: 'Remove override',
                          onPressed: () async {
                            final api = ref.read(apiClientProvider);
                            await api.delete(
                                '/flags/$flagId/overrides/$targetType/${o['targetId']}');
                            ref.invalidate(
                                _flagOverridesProvider(flagId));
                          },
                        ),
                      ],
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

  void _showAddOverrideDialog(BuildContext context, WidgetRef ref) {
    final targetIdCtrl = TextEditingController();
    final valueCtrl = TextEditingController();
    String targetType = 'user';

    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) {
          final cs = consoleColors(ctx);
          return Center(
            child: Material(
              color: Colors.transparent,
              child: Container(
                width: 440,
                constraints: const BoxConstraints(maxHeight: 420),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: cs.border),
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
                          Expanded(
                            child: Text('Add Override',
                                style: TextStyle(
                                  color: cs.textPrimary,
                                  fontSize: 16,
                                  fontWeight: FontWeight.w600,
                                )),
                          ),
                          GestureDetector(
                            onTap: () => Navigator.of(ctx).pop(),
                            child: Icon(LucideIcons.x,
                                size: 16,
                                color: cs.textSubtle),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 16),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 20),
                      child: Container(
                          height: 1, color: cs.border),
                    ),
                    const SizedBox(height: 16),
                    Flexible(
                      child: SingleChildScrollView(
                        padding: const EdgeInsets.symmetric(horizontal: 20),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            _buildDropdown(
                              cs: cs,
                              label: 'Target Type',
                              value: targetType,
                              items: const ['user', 'team'],
                              onChanged: (v) =>
                                  setDialogState(() => targetType = v!),
                            ),
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: targetIdCtrl,
                              label: 'Target ID',
                              hint: targetType == 'user'
                                  ? 'User ID'
                                  : 'Team ID',
                              autofocus: true,
                            ),
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: valueCtrl,
                              label: 'Value',
                              hint: 'Override value for this target',
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
                            label: 'Add Override',
                            onTap: () async {
                              final api = ref.read(apiClientProvider);
                              await api.post('/flags/$flagId/overrides',
                                  data: {
                                    'targetType': targetType,
                                    'targetId': targetIdCtrl.text,
                                    'value': valueCtrl.text,
                                  });
                              if (ctx.mounted) Navigator.pop(ctx);
                              ref.invalidate(
                                  _flagOverridesProvider(flagId));
                            },
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}

// --- Stats tab -------------------------------------------------------------

class _StatsTab extends ConsumerWidget {
  final String flagId;
  const _StatsTab({required this.flagId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cs = consoleColors(context);
    final statsAsync = ref.watch(_flagStatsProvider(flagId));

    return statsAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('Failed to load stats: $e',
                style: TextStyle(color: cs.textMuted)),
            const SizedBox(height: 8),
            FilledButton(
              onPressed: () =>
                  ref.invalidate(_flagStatsProvider(flagId)),
              child: const Text('Retry'),
            ),
          ],
        ),
      ),
      data: (data) {
        final totalEvals = data['totalEvaluations'] ?? 0;
        final uniqueUsers = data['uniqueUsers'] ?? 0;
        final distribution =
            Map<String, dynamic>.from(data['valueDistribution'] ?? {});

        return SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Stat cards row
              Row(
                children: [
                  Expanded(
                    child: _StatCard(
                      icon: LucideIcons.barChart3,
                      label: 'Total Evaluations',
                      value: _formatNumber(totalEvals),
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: _StatCard(
                      icon: LucideIcons.users,
                      label: 'Unique Users',
                      value: _formatNumber(uniqueUsers),
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: _StatCard(
                      icon: LucideIcons.pieChart,
                      label: 'Distinct Values',
                      value: '${distribution.length}',
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 24),
              // Value distribution
              Container(
                padding: const EdgeInsets.all(20),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: cs.border),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Icon(LucideIcons.pieChart,
                            size: 16, color: cs.textMuted),
                        const SizedBox(width: 8),
                        Text('Value Distribution',
                            style: TextStyle(
                                color: cs.textPrimary,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                      ],
                    ),
                    const SizedBox(height: 16),
                    if (distribution.isEmpty)
                      Center(
                        child: Padding(
                          padding: const EdgeInsets.all(24),
                          child: Text('No evaluation data yet',
                              style: TextStyle(color: cs.textSubtle)),
                        ),
                      )
                    else
                      ...distribution.entries.map((entry) {
                        final count = (entry.value is int)
                            ? entry.value as int
                            : int.tryParse(entry.value.toString()) ?? 0;
                        final total = totalEvals is int && totalEvals > 0
                            ? totalEvals as int
                            : 1;
                        final pct = (count / total * 100);
                        return Padding(
                          padding: const EdgeInsets.only(bottom: 12),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Row(
                                children: [
                                  Text(entry.key,
                                      style: TextStyle(
                                          color: cs.textPrimary,
                                          fontSize: 13,
                                          fontFamily: 'monospace')),
                                  const Spacer(),
                                  Text(
                                    '$count (${pct.toStringAsFixed(1)}%)',
                                    style: TextStyle(
                                        color: cs.textMuted, fontSize: 12),
                                  ),
                                ],
                              ),
                              const SizedBox(height: 6),
                              ClipRRect(
                                borderRadius: BorderRadius.circular(3),
                                child: LinearProgressIndicator(
                                  value: pct / 100,
                                  backgroundColor:
                                      _accent.withOpacity(0.1),
                                  valueColor:
                                      const AlwaysStoppedAnimation<Color>(
                                          _accent),
                                  minHeight: 8,
                                ),
                              ),
                            ],
                          ),
                        );
                      }),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  String _formatNumber(dynamic n) {
    if (n == null) return '0';
    final num = int.tryParse(n.toString()) ?? 0;
    if (num >= 1000000) return '${(num / 1000000).toStringAsFixed(1)}M';
    if (num >= 1000) return '${(num / 1000).toStringAsFixed(1)}K';
    return '$num';
  }
}

// --- Stat card -------------------------------------------------------------

class _StatCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;

  const _StatCard({
    required this.icon,
    required this.label,
    required this.value,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 16, color: _accent),
              const SizedBox(width: 8),
              Text(label,
                  style: TextStyle(color: cs.textMuted, fontSize: 12)),
            ],
          ),
          const SizedBox(height: 12),
          Text(value,
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 28,
                  fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }
}
