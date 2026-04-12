import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/app_empty_state.dart';
import '../../core/widgets/app_error_state.dart';
import '../../core/widgets/rich_text_editor.dart';
import '../../core/widgets/status_chip.dart';

// ── Providers ─────────────────────────────────────────────────────────────────

final _typesProvider = FutureProvider<List<Map<String, dynamic>>>((ref) async {
  final pid = ref.watch(currentProjectProvider);
  if (pid == null) return [];
  final res = await ref.read(apiClientProvider).get('/content/types');
  return List<Map<String, dynamic>>.from(
      (res.data as Map)['types'] as List? ?? []);
});

// (typeId, statusFilter) → entries response
final _entriesProvider = FutureProvider.family<Map<String, dynamic>, (String, String)>(
  (ref, args) async {
    final (typeId, statusFilter) = args;
    final pid = ref.watch(currentProjectProvider);
    if (pid == null) return {'total': 0, 'entries': []};
    final q = statusFilter.isEmpty ? '' : '?status=$statusFilter';
    final res = await ref.read(apiClientProvider)
        .get('/content/types/$typeId/entries$q');
    return res.data as Map<String, dynamic>;
  },
);

// (typeId, entryId) → version list
final _versionsProvider = FutureProvider.family<List<Map<String, dynamic>>, (String, String)>(
  (ref, args) async {
    final (typeId, entryId) = args;
    final res = await ref.read(apiClientProvider)
        .get('/content/types/$typeId/entries/$entryId/versions');
    return List<Map<String, dynamic>>.from(
        (res.data as Map)['versions'] as List? ?? []);
  },
);

// ── Models ────────────────────────────────────────────────────────────────────

const _fieldTypes = [
  'text', 'richtext', 'number', 'boolean',
  'date', 'media', 'reference', 'slug', 'seo',
];

// ── Page ──────────────────────────────────────────────────────────────────────

class ContentPage extends ConsumerStatefulWidget {
  const ContentPage({super.key});

  @override
  ConsumerState<ContentPage> createState() => _ContentPageState();
}

class _ContentPageState extends ConsumerState<ContentPage> {
  Map<String, dynamic>? _selectedType;
  Map<String, dynamic>? _selectedEntry;
  bool _creatingEntry = false;

  void _selectType(Map<String, dynamic> t) =>
      setState(() { _selectedType = t; _selectedEntry = null; _creatingEntry = false; });

  void _back() => setState(() {
    if (_selectedEntry != null || _creatingEntry) {
      _selectedEntry = null;
      _creatingEntry = false;
    } else {
      _selectedType = null;
    }
  });

  bool get _inEntryEditor => _selectedEntry != null || _creatingEntry;

  @override
  Widget build(BuildContext context) {
    final cs   = consoleColors(context);
    final hPad = pageHPad(context);

    return Scaffold(
      backgroundColor: cs.background,
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (!_inEntryEditor) ...[
            Padding(
              padding: EdgeInsets.fromLTRB(hPad, 32, hPad, 20),
              child: _Header(
                selectedType: _selectedType,
                selectedEntry: _selectedEntry,
                creatingEntry: _creatingEntry,
                onBack: _selectedType != null ? _back : null,
                cs: cs,
              ),
            ),
          ],
          Expanded(
            child: Padding(
              padding: EdgeInsets.symmetric(horizontal: hPad),
              child: _body(cs),
            ),
          ),
        ],
      ),
    );
  }

  Widget _body(dynamic cs) {
    if (_selectedType == null) {
      return _TypesView(onSelectType: _selectType, cs: cs);
    }
    if (_selectedEntry != null || _creatingEntry) {
      return _EntryEditor(
        type: _selectedType!,
        entry: _selectedEntry,
        onBack: _back,
        onSaved: () { ref.invalidate(_typesProvider); ref.invalidate(_entriesProvider); _back(); },
        cs: cs,
      );
    }
    return _EntriesView(
      type: _selectedType!,
      onSelectEntry: (e) => setState(() => _selectedEntry = e),
      onNewEntry: () => setState(() => _creatingEntry = true),
      cs: cs,
    );
  }
}

// ── Header ────────────────────────────────────────────────────────────────────

class _Header extends StatelessWidget {
  final Map<String, dynamic>? selectedType;
  final Map<String, dynamic>? selectedEntry;
  final bool creatingEntry;
  final VoidCallback? onBack;
  final dynamic cs;

  const _Header({
    required this.selectedType,
    required this.selectedEntry,
    required this.creatingEntry,
    required this.onBack,
    required this.cs,
  });

  String get _title {
    if (selectedType == null) return 'Content';
    if (creatingEntry) return 'New entry · ${selectedType!['name']}';
    if (selectedEntry != null) {
      return '${selectedType!['name']} · '
          '${selectedEntry!['slug'] ?? selectedEntry![r'$id'] ?? 'Entry'}';
    }
    return selectedType!['name'] as String;
  }

  String? get _subtitle {
    if (selectedType == null) {
      return 'Structured content types with custom fields, versioning and localization';
    }
    if (!creatingEntry && selectedEntry == null) {
      return selectedType!['slug'] as String? ?? '';
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (onBack != null) ...[
          Padding(
            padding: const EdgeInsets.only(top: 4),
            child: GestureDetector(
              onTap: onBack,
              child: Icon(LucideIcons.arrowLeft, size: 16, color: cs.textSecondary),
            ),
          ),
          const SizedBox(width: 8),
        ],
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(_title,
                style: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            if (_subtitle != null && _subtitle!.isNotEmpty) ...[
              const SizedBox(height: 4),
              Text(_subtitle!,
                  style: TextStyle(color: cs.textSubtle, fontSize: 13)),
            ],
          ],
        ),
      ],
    );
  }
}

// ══════════════════════════════════════════════════════════════════════════════
// Types View
// ══════════════════════════════════════════════════════════════════════════════

class _TypesView extends ConsumerStatefulWidget {
  final ValueChanged<Map<String, dynamic>> onSelectType;
  final dynamic cs;

  const _TypesView({required this.onSelectType, required this.cs});

  @override
  ConsumerState<_TypesView> createState() => _TypesViewState();
}

class _TypesViewState extends ConsumerState<_TypesView> {
  final _search = TextEditingController();
  bool _isGrid   = true;
  String _filter = ''; // '' | 'versioned' | 'localized'

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final cs         = widget.cs;
    final typesAsync = ref.watch(_typesProvider);

    return typesAsync.when(
      loading: () => const Center(child: CircularProgressIndicator(strokeWidth: 2)),
      error:   (e, _) => AppErrorState(error: e, onRetry: () => ref.invalidate(_typesProvider)),
      data: (allTypes) {
        // search + filter
        var types = allTypes;
        final q = _search.text.trim().toLowerCase();
        if (q.isNotEmpty) {
          types = types.where((t) =>
            (t['name'] as String? ?? '').toLowerCase().contains(q) ||
            (t['slug'] as String? ?? '').toLowerCase().contains(q)).toList();
        }
        if (_filter == 'versioned') {
          types = types.where((t) => t['versioning'] == true).toList();
        } else if (_filter == 'localized') {
          types = types.where((t) => t['localization'] == true).toList();
        }

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ── Toolbar ──────────────────────────────────────────────────
            Row(children: [
              // Search
              SizedBox(
                width: 220,
                height: 34,
                child: TextField(
                  controller: _search,
                  onChanged: (_) => setState(() {}),
                  style: TextStyle(fontSize: 13, color: cs.textPrimary),
                  decoration: InputDecoration(
                    hintText: 'Search types…',
                    hintStyle: TextStyle(color: cs.textMuted, fontSize: 13),
                    prefixIcon: Icon(LucideIcons.search,
                        size: 14, color: cs.textMuted),
                    prefixIconConstraints:
                        const BoxConstraints(minWidth: 32, minHeight: 32),
                    filled: true,
                    fillColor: cs.fieldFill,
                    isDense: true,
                    contentPadding: const EdgeInsets.symmetric(
                        horizontal: 10, vertical: 8),
                    border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(color: cs.fieldBorder)),
                    enabledBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(color: cs.fieldBorder)),
                    focusedBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide:
                            const BorderSide(color: Color(0xFF3472A4))),
                  ),
                ),
              ),
              const SizedBox(width: 10),
              // Filter chips
              for (final chip in [
                ('', 'All'),
                ('versioned', 'Versioned'),
                ('localized', 'Localized'),
              ]) ...[
                _FilterChip(
                  label: chip.$2,
                  active: _filter == chip.$1,
                  cs: cs,
                  onTap: () => setState(() => _filter = chip.$1),
                ),
                const SizedBox(width: 6),
              ],
              const Spacer(),
              // Count
              Text(
                '${types.length} type${types.length == 1 ? '' : 's'}',
                style: TextStyle(fontSize: 12, color: cs.textMuted),
              ),
              const SizedBox(width: 12),
              // Grid / list toggle
              _ViewToggle(
                isGrid: _isGrid,
                cs: cs,
                onToggle: () => setState(() => _isGrid = !_isGrid),
              ),
              const SizedBox(width: 10),
              FilledButton.icon(
                onPressed: () => _showCreateTypeDialog(context, ref),
                icon: const Icon(LucideIcons.plus, size: 14),
                label: const Text('New type'),
                style: FilledButton.styleFrom(
                  backgroundColor: const Color(0xFF3472A4),
                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 9),
                  textStyle:
                      const TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
                ),
              ),
            ]),
            const SizedBox(height: 16),

            // ── Content ──────────────────────────────────────────────────
            if (types.isEmpty)
              Expanded(
                child: AppEmptyState(
                  icon: LucideIcons.fileText,
                  title: q.isNotEmpty || _filter.isNotEmpty
                      ? 'No types match'
                      : 'No content types',
                  subtitle: q.isNotEmpty || _filter.isNotEmpty
                      ? 'Try a different search or filter.'
                      : 'Define structured content types with custom fields, versioning, and localization.',
                  actionLabel: q.isEmpty && _filter.isEmpty ? 'Create type' : null,
                  onAction: q.isEmpty && _filter.isEmpty
                      ? () => _showCreateTypeDialog(context, ref)
                      : null,
                ),
              )
            else if (_isGrid)
              Expanded(
                child: GridView.builder(
                  gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
                    maxCrossAxisExtent: 260,
                    mainAxisExtent: 140,
                    crossAxisSpacing: 12,
                    mainAxisSpacing: 12,
                  ),
                  itemCount: types.length,
                  itemBuilder: (_, i) => _TypeCard(
                    type: types[i],
                    cs: cs,
                    onTap: () => widget.onSelectType(types[i]),
                    onEdit: () => _showEditTypeDialog(context, ref, types[i]),
                    onDelete: () => _deleteType(context, ref, types[i]),
                  ),
                ),
              )
            else
              Expanded(
                child: ListView.separated(
                  itemCount: types.length,
                  separatorBuilder: (_, __) =>
                      Divider(height: 1, color: cs.border),
                  itemBuilder: (_, i) => _TypeListRow(
                    type: types[i],
                    cs: cs,
                    onTap: () => widget.onSelectType(types[i]),
                    onEdit: () => _showEditTypeDialog(context, ref, types[i]),
                    onDelete: () => _deleteType(context, ref, types[i]),
                  ),
                ),
              ),
          ],
        );
      },
    );
  }

  Future<void> _showCreateTypeDialog(BuildContext context, WidgetRef ref) async {
    final nameCtrl = TextEditingController();
    final slugCtrl = TextEditingController();
    bool versioning = false;
    bool localization = false;
    final fields = <Map<String, dynamic>>[];

    await showDialog(
      context: context,
      barrierColor: Colors.black.withValues(alpha: 0.6),
      builder: (ctx) => _TypeFormDialog(
        title: 'New content type',
        nameCtrl: nameCtrl,
        slugCtrl: slugCtrl,
        fields: fields,
        versioning: versioning,
        localization: localization,
        onVersioningChanged: (v) => versioning = v,
        onLocalizationChanged: (v) => localization = v,
        onConfirm: () async {
          final api = ref.read(apiClientProvider);
          await api.post('/content/types', data: {
            'name': nameCtrl.text.trim(),
            'slug': slugCtrl.text.trim().isEmpty
                ? nameCtrl.text.trim().toLowerCase().replaceAll(' ', '-')
                : slugCtrl.text.trim(),
            'fields': fields,
            'versioning': versioning,
            'localization': localization,
          });
          ref.invalidate(_typesProvider);
          if (ctx.mounted) Navigator.pop(ctx);
        },
      ),
    );
  }

  Future<void> _showEditTypeDialog(
      BuildContext context, WidgetRef ref, Map<String, dynamic> type) async {
    final nameCtrl = TextEditingController(text: type['name'] as String? ?? '');
    final slugCtrl = TextEditingController(text: type['slug'] as String? ?? '');
    final rawFields = (type['fields'] as List?)?.cast<Map<String, dynamic>>() ?? [];
    final fields = rawFields.map((f) => Map<String, dynamic>.from(f)).toList();

    await showDialog(
      context: context,
      barrierColor: Colors.black.withValues(alpha: 0.6),
      builder: (ctx) => _TypeFormDialog(
        title: 'Edit ${type['name']}',
        nameCtrl: nameCtrl,
        slugCtrl: slugCtrl,
        fields: fields,
        versioning: type['versioning'] as bool? ?? false,
        localization: type['localization'] as bool? ?? false,
        onVersioningChanged: (_) {},
        onLocalizationChanged: (_) {},
        onConfirm: () async {
          final api = ref.read(apiClientProvider);
          await api.put('/content/types/${type[r'$id']}', data: {
            'name': nameCtrl.text.trim(),
            'fields': fields,
          });
          ref.invalidate(_typesProvider);
          if (ctx.mounted) Navigator.pop(ctx);
        },
      ),
    );
  }

  Future<void> _deleteType(
      BuildContext context, WidgetRef ref, Map<String, dynamic> type) async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete type',
      subtitle: 'Delete "${type['name']}" and all its entries',
      content: Text(
        'This will permanently delete the content type and every entry it contains. This cannot be undone.',
        style: TextStyle(color: widget.cs.textSecondary, fontSize: 13),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Delete',
          destructive: true,
          onTap: () => Navigator.of(context, rootNavigator: true).pop(true),
        ),
      ],
    );
    if (confirmed != true) return;
    await ref.read(apiClientProvider).delete('/content/types/${type[r'$id']}');
    ref.invalidate(_typesProvider);
  }
}

class _TypeCard extends StatefulWidget {
  final Map<String, dynamic> type;
  final dynamic cs;
  final VoidCallback onTap;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  const _TypeCard({
    required this.type,
    required this.cs,
    required this.onTap,
    required this.onEdit,
    required this.onDelete,
  });

  @override
  State<_TypeCard> createState() => _TypeCardState();
}

class _TypeCardState extends State<_TypeCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final type = widget.type;
    final cs = widget.cs;
    final fieldCount = (type['fields'] as List?)?.length ?? 0;
    final versioning = type['versioning'] as bool? ?? false;
    final localization = type['localization'] as bool? ?? false;

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: _hovered ? cs.fillHover : cs.surface,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(
              color: _hovered
                  ? const Color(0xFF3472A4).withValues(alpha: 0.35)
                  : cs.border,
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Icon + actions row
              Row(
                children: [
                  Container(
                    width: 34,
                    height: 34,
                    decoration: BoxDecoration(
                      color: cs.fill,
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Icon(LucideIcons.fileText, size: 16, color: cs.textSubtle),
                  ),
                  const Spacer(),
                  _IconBtn(LucideIcons.pencil, cs, widget.onEdit),
                  const SizedBox(width: 2),
                  _IconBtn(LucideIcons.trash2, cs, widget.onDelete),
                ],
              ),
              const SizedBox(height: 12),
              // Name
              Text(
                type['name'] as String? ?? '',
                style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 2),
              Text(
                type['slug'] as String? ?? '',
                style: TextStyle(
                  color: cs.textMuted,
                  fontSize: 11,
                  fontFamily: 'monospace',
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              const Spacer(),
              // Badges row
              Wrap(
                spacing: 5,
                children: [
                  _Badge('$fieldCount field${fieldCount == 1 ? '' : 's'}', cs),
                  if (versioning) _Badge('versioned', cs),
                  if (localization) _Badge('i18n', cs),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ══════════════════════════════════════════════════════════════════════════════
// Entries View
// ══════════════════════════════════════════════════════════════════════════════

class _EntriesView extends ConsumerStatefulWidget {
  final Map<String, dynamic> type;
  final ValueChanged<Map<String, dynamic>> onSelectEntry;
  final VoidCallback onNewEntry;
  final dynamic cs;

  const _EntriesView({
    required this.type,
    required this.onSelectEntry,
    required this.onNewEntry,
    required this.cs,
  });

  @override
  ConsumerState<_EntriesView> createState() => _EntriesViewState();
}

class _EntriesViewState extends ConsumerState<_EntriesView> {
  String _statusFilter = '';

  String get _typeId => widget.type[r'$id'] as String;

  void _setFilter(String s) => setState(() => _statusFilter = s);

  @override
  Widget build(BuildContext context) {
    final cs = widget.cs;
    final dataAsync = ref.watch(_entriesProvider((_typeId, _statusFilter)));

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Toolbar
        Row(
          children: [
            for (final label in ['All', 'Draft', 'Published', 'Archived']) ...[
              _FilterChip(
                label: label,
                active: _statusFilter == (label == 'All' ? '' : label.toLowerCase()),
                cs: cs,
                onTap: () => _setFilter(label == 'All' ? '' : label.toLowerCase()),
              ),
              const SizedBox(width: 6),
            ],
            const Spacer(),
            FilledButton.icon(
              onPressed: widget.onNewEntry,
              icon: const Icon(LucideIcons.plus, size: 14),
              label: const Text('New entry'),
              style: FilledButton.styleFrom(
                backgroundColor: const Color(0xFF3472A4),
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 9),
                textStyle: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
              ),
            ),
          ],
        ),
        const SizedBox(height: 16),
        Expanded(
          child: dataAsync.when(
            loading: () => const Center(child: CircularProgressIndicator(strokeWidth: 2)),
            error: (e, _) => AppErrorState(error: e, onRetry: () => ref.invalidate(_entriesProvider((_typeId, _statusFilter)))),
            data: (data) {
              final entries = List<Map<String, dynamic>>.from(
                  (data['entries'] as List?) ?? []);
              if (entries.isEmpty) {
                return AppEmptyState(
                  icon: LucideIcons.fileEdit,
                  title: 'No entries',
                  subtitle: _statusFilter.isEmpty
                      ? 'Create your first entry for this content type.'
                      : 'No $_statusFilter entries found.',
                  actionLabel: _statusFilter.isEmpty ? 'New entry' : null,
                  onAction: _statusFilter.isEmpty ? widget.onNewEntry : null,
                );
              }
              return Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Table header
                  _TableHeader(type: widget.type, cs: cs),
                  const SizedBox(height: 4),
                  Expanded(
                    child: ListView.separated(
                      itemCount: entries.length,
                      separatorBuilder: (_, __) =>
                          Divider(height: 1, color: cs.border),
                      itemBuilder: (_, i) => _EntryRow(
                        entry: entries[i],
                        type: widget.type,
                        cs: cs,
                        onTap: () => widget.onSelectEntry(entries[i]),
                        onPublish: () => _publish(entries[i]),
                        onUnpublish: () => _unpublish(entries[i]),
                        onDelete: () => _delete(context, entries[i]),
                      ),
                    ),
                  ),
                ],
              );
            },
          ),
        ),
      ],
    );
  }

  Future<void> _publish(Map<String, dynamic> entry) async {
    await ref.read(apiClientProvider).patch(
        '/content/types/${widget.type[r'$id']}/entries/${entry[r'$id']}/publish');
    ref.invalidate(_entriesProvider((_typeId, _statusFilter)));
  }

  Future<void> _unpublish(Map<String, dynamic> entry) async {
    await ref.read(apiClientProvider).patch(
        '/content/types/${widget.type[r'$id']}/entries/${entry[r'$id']}/unpublish');
    ref.invalidate(_entriesProvider((_typeId, _statusFilter)));
  }

  Future<void> _delete(BuildContext context, Map<String, dynamic> entry) async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete entry',
      content: Text(
        'Delete this entry? This cannot be undone.',
        style: TextStyle(color: widget.cs.textSecondary, fontSize: 13),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Delete',
          destructive: true,
          onTap: () => Navigator.of(context, rootNavigator: true).pop(true),
        ),
      ],
    );
    if (confirmed != true) return;
    await ref.read(apiClientProvider).delete(
        '/content/types/${widget.type[r'$id']}/entries/${entry[r'$id']}');
    ref.invalidate(_entriesProvider((_typeId, _statusFilter)));
  }
}

class _TableHeader extends StatelessWidget {
  final Map<String, dynamic> type;
  final dynamic cs;
  const _TableHeader({required this.type, required this.cs});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 8),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: cs.border)),
      ),
      child: Row(
        children: [
          Expanded(flex: 4, child: _th('Slug', cs)),
          Expanded(flex: 2, child: _th('Status', cs)),
          Expanded(flex: 2, child: _th('Locale', cs)),
          Expanded(flex: 1, child: _th('Ver.', cs)),
          Expanded(flex: 3, child: _th('Updated', cs)),
          const SizedBox(width: 80),
        ],
      ),
    );
  }

  Widget _th(String label, dynamic cs) => Text(
        label,
        style: TextStyle(fontSize: 11, fontWeight: FontWeight.w600, color: cs.textMuted),
      );
}

class _EntryRow extends StatefulWidget {
  final Map<String, dynamic> entry;
  final Map<String, dynamic> type;
  final dynamic cs;
  final VoidCallback onTap;
  final VoidCallback onPublish;
  final VoidCallback onUnpublish;
  final VoidCallback onDelete;

  const _EntryRow({
    required this.entry,
    required this.type,
    required this.cs,
    required this.onTap,
    required this.onPublish,
    required this.onUnpublish,
    required this.onDelete,
  });

  @override
  State<_EntryRow> createState() => _EntryRowState();
}

class _EntryRowState extends State<_EntryRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final e = widget.entry;
    final cs = widget.cs;
    final status = e['status'] as String? ?? 'draft';
    final slug = e['slug'] as String? ?? e[r'$id'] as String? ?? '—';
    final locale = e['locale'] as String? ?? 'en';
    final version = e['version']?.toString() ?? '1';
    final updated = (e[r'$updatedAt'] ?? e['updatedAt'] ?? '')
        .toString()
        .split('T')
        .first;

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          color: _hovered ? cs.fill : Colors.transparent,
          padding: const EdgeInsets.symmetric(vertical: 10),
          child: Row(
            children: [
              Expanded(
                flex: 4,
                child: Text(slug,
                    style: TextStyle(
                        fontSize: 13,
                        fontFamily: 'monospace',
                        color: cs.textPrimary)),
              ),
              Expanded(
                flex: 2,
                child: StatusChip.fromStatus(status),
              ),
              Expanded(
                flex: 2,
                child: Text(locale,
                    style: TextStyle(fontSize: 12, color: cs.textSecondary)),
              ),
              Expanded(
                flex: 1,
                child: Text('v$version',
                    style: TextStyle(fontSize: 12, color: cs.textMuted)),
              ),
              Expanded(
                flex: 3,
                child: Text(updated,
                    style: TextStyle(fontSize: 12, color: cs.textSecondary)),
              ),
              SizedBox(
                width: 80,
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    if (status == 'draft')
                      _IconBtn(LucideIcons.send, cs, widget.onPublish,
                          tooltip: 'Publish'),
                    if (status == 'published')
                      _IconBtn(LucideIcons.eyeOff, cs, widget.onUnpublish,
                          tooltip: 'Unpublish'),
                    const SizedBox(width: 4),
                    _IconBtn(LucideIcons.trash2, cs, widget.onDelete),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ══════════════════════════════════════════════════════════════════════════════
// Entry Editor
// ══════════════════════════════════════════════════════════════════════════════

class _EntryEditor extends ConsumerStatefulWidget {
  final Map<String, dynamic> type;
  final Map<String, dynamic>? entry;
  final VoidCallback onSaved;
  final VoidCallback onBack;
  final dynamic cs;

  const _EntryEditor({
    required this.type,
    required this.entry,
    required this.onSaved,
    required this.onBack,
    required this.cs,
  });

  @override
  ConsumerState<_EntryEditor> createState() => _EntryEditorState();
}

class _EntryEditorState extends ConsumerState<_EntryEditor> {
  final _localeCtrl = TextEditingController(text: 'en');
  final _fieldCtrls = <String, TextEditingController>{};
  final _boolValues = <String, bool>{};
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    final fields = _fields();
    for (final f in fields) {
      final key = f['key'] as String;
      final type = f['type'] as String? ?? 'text';
      if (type == 'boolean') {
        _boolValues[key] = false;
      } else {
        _fieldCtrls[key] = TextEditingController();
      }
    }
    if (widget.entry != null) {
      _localeCtrl.text = widget.entry!['locale'] as String? ?? 'en';
      final data = widget.entry!['data'] as Map<String, dynamic>? ?? {};
      for (final f in fields) {
        final key = f['key'] as String;
        final type = f['type'] as String? ?? 'text';
        if (type == 'boolean') {
          _boolValues[key] = data[key] as bool? ?? false;
        } else {
          _fieldCtrls[key]?.text = data[key]?.toString() ?? '';
        }
      }
    }
  }

  // Derives a URL-safe slug from the first text/richtext/slug field value.
  String _deriveSlug() {
    final fields = _fields();
    final firstText = fields.firstWhere(
      (f) => ['text', 'richtext', 'slug'].contains(f['type'] ?? 'text'),
      orElse: () => {},
    );
    if (firstText.isEmpty) return '';
    final raw = _fieldCtrls[firstText['key']]?.text.trim() ?? '';
    return raw
        .toLowerCase()
        .replaceAll(RegExp(r'[^a-z0-9\s-]'), '')
        .replaceAll(RegExp(r'\s+'), '-')
        .replaceAll(RegExp(r'-+'), '-');
  }

  @override
  void dispose() {
    _localeCtrl.dispose();
    for (final c in _fieldCtrls.values) {
      c.dispose();
    }
    super.dispose();
  }

  List<Map<String, dynamic>> _fields() =>
      (widget.type['fields'] as List?)?.cast<Map<String, dynamic>>() ?? [];

  Map<String, dynamic> _collectData() {
    final out = <String, dynamic>{};
    for (final f in _fields()) {
      final key = f['key'] as String;
      final type = f['type'] as String? ?? 'text';
      if (type == 'boolean') {
        out[key] = _boolValues[key] ?? false;
      } else if (type == 'number') {
        out[key] = num.tryParse(_fieldCtrls[key]?.text ?? '') ?? 0;
      } else {
        out[key] = _fieldCtrls[key]?.text ?? '';
      }
    }
    return out;
  }

  Future<void> _save({bool publish = false}) async {
    setState(() { _saving = true; _error = null; });
    try {
      final api = ref.read(apiClientProvider);
      final data = _collectData();
      final typeId = widget.type[r'$id'] as String;

      if (widget.entry == null) {
        final res = await api.post('/content/types/$typeId/entries', data: {
          'slug': _deriveSlug(),
          'locale': _localeCtrl.text.trim().isEmpty ? 'en' : _localeCtrl.text.trim(),
          'data': data,
        });
        if (publish) {
          final entryId = (res.data as Map)[r'$id'] as String;
          await api.patch('/content/types/$typeId/entries/$entryId/publish');
        }
      } else {
        final entryId = widget.entry![r'$id'] as String;
        await api.put('/content/types/$typeId/entries/$entryId', data: {'data': data});
        if (publish) {
          await api.patch('/content/types/$typeId/entries/$entryId/publish');
        }
      }
      ref.invalidate(_entriesProvider);
      widget.onSaved();
    } catch (e) {
      setState(() { _error = e.toString(); _saving = false; });
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = widget.cs;
    final fields = _fields();
    final isNew = widget.entry == null;
    final status = widget.entry?['status'] as String? ?? 'draft';
    final typeName = widget.type['name'] as String? ?? 'Entry';
    final entryLabel = isNew
        ? 'New entry'
        : (widget.entry!['slug'] as String? ?? widget.entry![r'$id'] as String? ?? 'Entry');

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // ── Integrated header: back + title + actions ──────────────────
        Row(
          children: [
            GestureDetector(
              onTap: widget.onBack,
              child: Icon(LucideIcons.arrowLeft, size: 16, color: cs.textSecondary),
            ),
            const SizedBox(width: 10),
            Text(
              '$typeName · $entryLabel',
              style: TextStyle(
                color: cs.textPrimary,
                fontSize: 22,
                fontWeight: FontWeight.w600,
              ),
            ),
            const Spacer(),
            if (_error != null) ...[
              Text(_error!,
                  style: const TextStyle(color: Colors.redAccent, fontSize: 12)),
              const SizedBox(width: 12),
            ],
            OutlinedButton(
              onPressed: _saving ? null : () => _save(),
              style: OutlinedButton.styleFrom(
                side: BorderSide(color: cs.border),
                foregroundColor: cs.textSecondary,
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 9),
                textStyle: const TextStyle(fontSize: 13),
              ),
              child: Text(_saving ? 'Saving…' : 'Save draft'),
            ),
            const SizedBox(width: 8),
            FilledButton(
              onPressed: _saving ? null : () => _save(publish: true),
              style: FilledButton.styleFrom(
                backgroundColor: const Color(0xFF10B981),
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 9),
                textStyle: const TextStyle(
                    fontSize: 13, fontWeight: FontWeight.w500),
              ),
              child: Text(
                  status == 'published' ? 'Update & publish' : 'Publish'),
            ),
          ],
        ),

        const SizedBox(height: 24),

        // ── Body: form + sidebar ───────────────────────────────────────
        Expanded(
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Main form
              Expanded(
                flex: 3,
                child: ListView(
                  children: [
                    if (widget.type['localization'] == true) ...[
                      _FieldLabel('Locale', cs),
                      const SizedBox(height: 6),
                      _textField(_localeCtrl, 'en', cs),
                      const SizedBox(height: 20),
                    ],
                    for (final f in fields) ...[
                      _FieldLabel(
                        f['label'] as String? ?? f['key'] as String,
                        cs,
                        required: f['required'] as bool? ?? false,
                      ),
                      const SizedBox(height: 6),
                      _buildFieldInput(f, cs),
                      const SizedBox(height: 20),
                    ],
                    const SizedBox(height: 32),
                  ],
                ),
              ),

              const SizedBox(width: 28),

              // Sidebar
              SizedBox(
                width: 210,
                child: ListView(
                  children: [
                    if (!isNew) ...[
                      _SidebarSection(
                        title: 'Status',
                        cs: cs,
                        child: StatusChip.fromStatus(status),
                      ),
                      const SizedBox(height: 16),
                      _SidebarSection(
                        title: 'Version',
                        cs: cs,
                        child: Text(
                          'v${widget.entry!['version'] ?? 1}',
                          style: TextStyle(
                              fontSize: 13, color: cs.textSecondary),
                        ),
                      ),
                      const SizedBox(height: 16),
                      _VersionHistory(
                        typeId: widget.type[r'$id'] as String,
                        entryId: widget.entry![r'$id'] as String,
                        cs: cs,
                      ),
                    ] else ...[
                      Container(
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          color: cs.surface,
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(color: cs.border),
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(children: [
                              Icon(LucideIcons.info,
                                  size: 13, color: cs.textSubtle),
                              const SizedBox(width: 6),
                              Text('Draft',
                                  style: TextStyle(
                                      fontSize: 12,
                                      fontWeight: FontWeight.w600,
                                      color: cs.textSecondary)),
                            ]),
                            const SizedBox(height: 6),
                            Text(
                              'Saved as a draft. Use Publish to make it live.',
                              style: TextStyle(
                                  fontSize: 12,
                                  color: cs.textMuted,
                                  height: 1.5),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildFieldInput(Map<String, dynamic> field, dynamic cs) {
    final key = field['key'] as String;
    final type = field['type'] as String? ?? 'text';

    switch (type) {
      case 'boolean':
        return Row(
          children: [
            Switch(
              value: _boolValues[key] ?? false,
              onChanged: (v) => setState(() => _boolValues[key] = v),
              activeThumbColor: const Color(0xFF3472A4),
            ),
            const SizedBox(width: 8),
            Text(
              (_boolValues[key] ?? false) ? 'True' : 'False',
              style: TextStyle(fontSize: 13, color: cs.textSecondary),
            ),
          ],
        );
      case 'richtext':
        return RichTextEditor(
          controller: _fieldCtrls[key]!,
          hintText: 'Write in Markdown…\n\nTip: **bold**, *italic*, `code`, ## headings',
          minLines: 14,
        );
      case 'number':
        return _textField(_fieldCtrls[key]!, '0', cs,
            inputType: TextInputType.number);
      case 'date':
        return _textField(_fieldCtrls[key]!, 'YYYY-MM-DD', cs);
      case 'seo':
        // SEO renders as a labelled block with three sub-fields
        final ctrl = _fieldCtrls[key]!;
        return Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: cs.fill,
            borderRadius: BorderRadius.circular(7),
            border: Border.all(color: cs.border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('SEO meta (stored as JSON in this field)',
                  style:
                      TextStyle(fontSize: 11, color: cs.textMuted, height: 1.4)),
              const SizedBox(height: 8),
              _textField(ctrl, '{"title":"","description":"","image":""}', cs,
                  maxLines: 3),
            ],
          ),
        );
      default: // text, media, reference, slug
        return _textField(_fieldCtrls[key]!, '', cs);
    }
  }

  Widget _textField(
    TextEditingController ctrl,
    String hint,
    dynamic cs, {
    int? maxLines,
    TextInputType? inputType,
  }) {
    return TextField(
      controller: ctrl,
      maxLines: maxLines ?? 1,
      keyboardType: inputType,
      style: TextStyle(fontSize: 13, color: cs.textPrimary),
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: TextStyle(color: cs.textMuted, fontSize: 13),
        filled: true,
        fillColor: cs.fill,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(7),
          borderSide: BorderSide(color: cs.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(7),
          borderSide: BorderSide(color: cs.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(7),
          borderSide: const BorderSide(color: Color(0xFF3472A4)),
        ),
      ),
    );
  }
}

class _SidebarSection extends StatelessWidget {
  final String title;
  final Widget child;
  final dynamic cs;
  const _SidebarSection(
      {required this.title, required this.child, required this.cs});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(title,
            style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: cs.textMuted,
                letterSpacing: 0.5)),
        const SizedBox(height: 6),
        child,
      ],
    );
  }
}

class _VersionHistory extends ConsumerWidget {
  final String typeId;
  final String entryId;
  final dynamic cs;

  const _VersionHistory({
    required this.typeId,
    required this.entryId,
    required this.cs,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final versionsAsync = ref.watch(_versionsProvider((typeId, entryId)));

    return _SidebarSection(
      title: 'Version history',
      cs: cs,
      child: versionsAsync.when(
        loading: () => const SizedBox(
            height: 20, child: Center(child: CircularProgressIndicator(strokeWidth: 1.5))),
        error: (_, __) => const SizedBox.shrink(),
        data: (versions) => Column(
          children: versions.take(5).map((v) {
            final date = (v[r'$createdAt'] ?? v['createdAt'] ?? '')
                .toString()
                .split('T')
                .first;
            return Padding(
              padding: const EdgeInsets.only(bottom: 6),
              child: Row(
                children: [
                  Text('v${v['version']}',
                      style: TextStyle(
                          fontSize: 11,
                          fontFamily: 'monospace',
                          color: cs.textSecondary)),
                  const Spacer(),
                  Text(date,
                      style: TextStyle(fontSize: 11, color: cs.textMuted)),
                ],
              ),
            );
          }).toList(),
        ),
      ),
    );
  }
}

// ══════════════════════════════════════════════════════════════════════════════
// Type Form Dialog (create / edit)
// ══════════════════════════════════════════════════════════════════════════════

class _TypeFormDialog extends StatefulWidget {
  final String title;
  final TextEditingController nameCtrl;
  final TextEditingController slugCtrl;
  final List<Map<String, dynamic>> fields;
  final bool versioning;
  final bool localization;
  final ValueChanged<bool> onVersioningChanged;
  final ValueChanged<bool> onLocalizationChanged;
  final Future<void> Function() onConfirm;

  const _TypeFormDialog({
    required this.title,
    required this.nameCtrl,
    required this.slugCtrl,
    required this.fields,
    required this.versioning,
    required this.localization,
    required this.onVersioningChanged,
    required this.onLocalizationChanged,
    required this.onConfirm,
  });

  @override
  State<_TypeFormDialog> createState() => _TypeFormDialogState();
}

class _TypeFormDialogState extends State<_TypeFormDialog> {
  late bool _versioning;
  late bool _localization;
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _versioning = widget.versioning;
    _localization = widget.localization;
  }

  void _addField() {
    setState(() {
      widget.fields.add({'key': '', 'label': '', 'type': 'text', 'required': false});
    });
  }

  void _removeField(int index) => setState(() => widget.fields.removeAt(index));

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);

    return Dialog(
      backgroundColor: cs.surface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: cs.border),
      ),
      child: SizedBox(
        width: 560,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 20, 20, 0),
              child: Row(
                children: [
                  Text(widget.title,
                      style: TextStyle(
                          color: cs.textPrimary,
                          fontSize: 15,
                          fontWeight: FontWeight.w600)),
                  const Spacer(),
                  GestureDetector(
                    onTap: () => Navigator.pop(context),
                    child: Icon(LucideIcons.x, size: 16, color: cs.textMuted),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            Divider(height: 1, color: cs.border),

            // Scrollable body
            ConstrainedBox(
              constraints: const BoxConstraints(maxHeight: 480),
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Name
                    _FieldLabel('Name', cs),
                    const SizedBox(height: 6),
                    _dialogField(widget.nameCtrl, 'Blog posts', cs),
                    const SizedBox(height: 16),
                    // Slug
                    _FieldLabel('Slug (API identifier)', cs),
                    const SizedBox(height: 6),
                    _dialogField(widget.slugCtrl, 'blog-posts', cs,
                        hint2: 'Auto-generated from name if left blank'),
                    const SizedBox(height: 20),
                    // Options
                    Row(
                      children: [
                        _ToggleRow('Versioning', _versioning, cs, (v) {
                          setState(() => _versioning = v);
                          widget.onVersioningChanged(v);
                        }),
                        const SizedBox(width: 24),
                        _ToggleRow('Localization', _localization, cs, (v) {
                          setState(() => _localization = v);
                          widget.onLocalizationChanged(v);
                        }),
                      ],
                    ),
                    const SizedBox(height: 24),
                    // Fields
                    Row(
                      children: [
                        Text('Fields',
                            style: TextStyle(
                                color: cs.textPrimary,
                                fontSize: 13,
                                fontWeight: FontWeight.w600)),
                        const Spacer(),
                        GestureDetector(
                          onTap: _addField,
                          child: const Row(children: [
                            Icon(LucideIcons.plus, size: 13,
                                color: Color(0xFF3472A4)),
                            SizedBox(width: 4),
                            Text('Add field',
                                style: TextStyle(
                                    fontSize: 12,
                                    color: Color(0xFF3472A4),
                                    fontWeight: FontWeight.w500)),
                          ]),
                        ),
                      ],
                    ),
                    const SizedBox(height: 10),
                    if (widget.fields.isEmpty)
                      Text('No fields yet. Add a field to define the schema.',
                          style: TextStyle(fontSize: 12, color: cs.textMuted)),
                    for (int i = 0; i < widget.fields.length; i++)
                      _FieldRow(
                        field: widget.fields[i],
                        cs: cs,
                        onRemove: () => _removeField(i),
                        onChanged: (updated) =>
                            setState(() => widget.fields[i] = updated),
                      ),
                  ],
                ),
              ),
            ),

            Divider(height: 1, color: cs.border),
            // Footer
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 14, 24, 18),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (_error != null) ...[
                    Text(_error!, style: const TextStyle(color: Colors.redAccent, fontSize: 12)),
                    const SizedBox(height: 10),
                  ],
                  Row(
                children: [
                  const Spacer(),
                  OutlinedButton(
                    onPressed: () => Navigator.pop(context),
                    style: OutlinedButton.styleFrom(
                      foregroundColor: cs.textSecondary,
                      side: BorderSide(color: cs.border),
                      padding: const EdgeInsets.symmetric(
                          horizontal: 16, vertical: 9),
                      textStyle: const TextStyle(fontSize: 13),
                    ),
                    child: const Text('Cancel'),
                  ),
                  const SizedBox(width: 8),
                  FilledButton(
                    onPressed: _saving ? null : () async {
                      setState(() { _saving = true; _error = null; });
                      try {
                        await widget.onConfirm();
                      } catch (e) {
                        if (mounted) setState(() { _saving = false; _error = e.toString(); });
                      }
                    },
                    style: FilledButton.styleFrom(
                      backgroundColor: const Color(0xFF3472A4),
                      padding: const EdgeInsets.symmetric(
                          horizontal: 16, vertical: 9),
                      textStyle: const TextStyle(
                          fontSize: 13, fontWeight: FontWeight.w500),
                    ),
                    child: Text(_saving ? 'Saving…' : 'Save'),
                  ),
                ],
              ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _dialogField(
    TextEditingController ctrl,
    String hint,
    dynamic cs, {
    String? hint2,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextField(
          controller: ctrl,
          style: TextStyle(fontSize: 13, color: cs.textPrimary),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: TextStyle(color: cs.textMuted, fontSize: 13),
            filled: true,
            fillColor: cs.fill,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(7),
              borderSide: BorderSide(color: cs.border),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(7),
              borderSide: BorderSide(color: cs.border),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(7),
              borderSide: const BorderSide(color: Color(0xFF3472A4)),
            ),
          ),
        ),
        if (hint2 != null) ...[
          const SizedBox(height: 4),
          Text(hint2,
              style: TextStyle(fontSize: 11, color: cs.textMuted)),
        ],
      ],
    );
  }
}

class _FieldRow extends StatefulWidget {
  final Map<String, dynamic> field;
  final dynamic cs;
  final VoidCallback onRemove;
  final ValueChanged<Map<String, dynamic>> onChanged;

  const _FieldRow({
    required this.field,
    required this.cs,
    required this.onRemove,
    required this.onChanged,
  });

  @override
  State<_FieldRow> createState() => _FieldRowState();
}

class _FieldRowState extends State<_FieldRow> {
  late final _keyCtrl = TextEditingController(
      text: widget.field['key'] as String? ?? '');
  late final _labelCtrl = TextEditingController(
      text: widget.field['label'] as String? ?? '');
  late String _type;
  late bool _required;

  @override
  void initState() {
    super.initState();
    _type = widget.field['type'] as String? ?? 'text';
    _required = widget.field['required'] as bool? ?? false;
  }

  @override
  void dispose() {
    _keyCtrl.dispose();
    _labelCtrl.dispose();
    super.dispose();
  }

  void _emit() {
    widget.onChanged({
      'key': _keyCtrl.text,
      'label': _labelCtrl.text,
      'type': _type,
      'required': _required,
    });
  }

  @override
  Widget build(BuildContext context) {
    final cs = widget.cs;
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: cs.fill,
        borderRadius: BorderRadius.circular(7),
        border: Border.all(color: cs.border),
      ),
      child: Row(
        children: [
          // Key
          Expanded(
            flex: 2,
            child: _miniField(_keyCtrl, 'key', cs, onChanged: (_) => _emit()),
          ),
          const SizedBox(width: 8),
          // Label
          Expanded(
            flex: 2,
            child: _miniField(_labelCtrl, 'Label', cs, onChanged: (_) => _emit()),
          ),
          const SizedBox(width: 8),
          // Type dropdown
          DropdownButton<String>(
            value: _type,
            isDense: true,
            underline: const SizedBox.shrink(),
            dropdownColor: cs.surface,
            style: TextStyle(fontSize: 12, color: cs.textSecondary),
            items: _fieldTypes
                .map((t) => DropdownMenuItem(
                      value: t,
                      child: Text(t,
                          style: TextStyle(
                              fontSize: 12, color: cs.textSecondary)),
                    ))
                .toList(),
            onChanged: (v) {
              setState(() => _type = v!);
              _emit();
            },
          ),
          const SizedBox(width: 8),
          // Required toggle
          Tooltip(
            message: 'Required',
            child: GestureDetector(
              onTap: () {
                setState(() => _required = !_required);
                _emit();
              },
              child: Icon(
                _required ? LucideIcons.asterisk : LucideIcons.minus,
                size: 13,
                color: _required
                    ? const Color(0xFF3472A4)
                    : cs.textMuted,
              ),
            ),
          ),
          const SizedBox(width: 8),
          GestureDetector(
            onTap: widget.onRemove,
            child: Icon(LucideIcons.x, size: 13, color: cs.textMuted),
          ),
        ],
      ),
    );
  }

  Widget _miniField(
    TextEditingController ctrl,
    String hint,
    dynamic cs, {
    ValueChanged<String>? onChanged,
  }) {
    return TextField(
      controller: ctrl,
      onChanged: onChanged,
      style: TextStyle(fontSize: 12, color: cs.textPrimary),
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: TextStyle(color: cs.textMuted, fontSize: 12),
        isDense: true,
        filled: true,
        fillColor: cs.surface,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 8, vertical: 7),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(5),
          borderSide: BorderSide(color: cs.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(5),
          borderSide: BorderSide(color: cs.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(5),
          borderSide: const BorderSide(color: Color(0xFF3472A4)),
        ),
      ),
    );
  }
}

// ── View toggle (grid / list) ─────────────────────────────────────────────────

class _ViewToggle extends StatelessWidget {
  final bool isGrid;
  final dynamic cs;
  final VoidCallback onToggle;
  const _ViewToggle({required this.isGrid, required this.cs, required this.onToggle});

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
          border: Border.all(color: cs.border),
          borderRadius: BorderRadius.circular(8)),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        _Btn(LucideIcons.layoutGrid, isGrid,  cs, onToggle),
        _Btn(LucideIcons.list,       !isGrid, cs, onToggle),
      ]),
    );
  }
}

class _Btn extends StatelessWidget {
  final IconData icon;
  final bool active;
  final dynamic cs;
  final VoidCallback onTap;
  const _Btn(this.icon, this.active, this.cs, this.onTap);

  @override
  Widget build(BuildContext context) => GestureDetector(
        onTap: onTap,
        child: Container(
          width: 30,
          height: 30,
          decoration: BoxDecoration(
              color: active ? cs.fillActive : Colors.transparent,
              borderRadius: BorderRadius.circular(7)),
          child: Icon(icon,
              size: 14,
              color: active ? const Color(0xFF3472A4) : cs.textMuted),
        ),
      );
}

// ── Type list row (list-view alternative to card) ─────────────────────────────

class _TypeListRow extends StatefulWidget {
  final Map<String, dynamic> type;
  final dynamic cs;
  final VoidCallback onTap;
  final VoidCallback onEdit;
  final VoidCallback onDelete;
  const _TypeListRow({
    required this.type,
    required this.cs,
    required this.onTap,
    required this.onEdit,
    required this.onDelete,
  });

  @override
  State<_TypeListRow> createState() => _TypeListRowState();
}

class _TypeListRowState extends State<_TypeListRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final t           = widget.type;
    final cs          = widget.cs;
    final name        = t['name'] as String? ?? '';
    final slug        = t['slug'] as String? ?? '';
    final fieldCount  = (t['fields'] as List?)?.length ?? 0;
    final versioning  = t['versioning'] as bool? ?? false;
    final localization = t['localization'] as bool? ?? false;

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      cursor:  SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          color: _hovered ? cs.fill : Colors.transparent,
          padding: const EdgeInsets.symmetric(horizontal: 2, vertical: 10),
          child: Row(children: [
            Container(
              width: 32,
              height: 32,
              decoration: BoxDecoration(
                  color: cs.fill,
                  borderRadius: BorderRadius.circular(8)),
              child: Icon(LucideIcons.fileText, size: 14, color: cs.textSubtle),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(name,
                    style: TextStyle(
                        color: cs.textPrimary,
                        fontSize: 13,
                        fontWeight: FontWeight.w500)),
                Text(slug,
                    style: TextStyle(
                        color: cs.textMuted,
                        fontSize: 11,
                        fontFamily: 'monospace')),
              ]),
            ),
            _Badge('$fieldCount field${fieldCount == 1 ? '' : 's'}', cs),
            if (versioning) ...[
              const SizedBox(width: 6),
              _Badge('versioned', cs),
            ],
            if (localization) ...[
              const SizedBox(width: 6),
              _Badge('i18n', cs),
            ],
            const SizedBox(width: 16),
            if (_hovered) ...[
              _IconBtn(LucideIcons.pencil, cs, widget.onEdit),
              const SizedBox(width: 4),
              _IconBtn(LucideIcons.trash2, cs, widget.onDelete),
            ],
          ]),
        ),
      ),
    );
  }
}

// ── Shared small widgets ──────────────────────────────────────────────────────

class _Badge extends StatelessWidget {
  final String label;
  final dynamic cs;
  const _Badge(this.label, this.cs);

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
      decoration: BoxDecoration(
        color: cs.fill,
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: cs.border),
      ),
      child: Text(label,
          style: TextStyle(fontSize: 11, color: cs.textMuted)),
    );
  }
}

class _IconBtn extends StatelessWidget {
  final IconData icon;
  final dynamic cs;
  final VoidCallback onTap;
  final String? tooltip;
  const _IconBtn(this.icon, this.cs, this.onTap, {this.tooltip});

  @override
  Widget build(BuildContext context) {
    final btn = GestureDetector(
      onTap: onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Icon(icon, size: 14, color: cs.textMuted),
      ),
    );
    return tooltip != null ? Tooltip(message: tooltip!, child: btn) : btn;
  }
}

class _FilterChip extends StatelessWidget {
  final String label;
  final bool active;
  final dynamic cs;
  final VoidCallback onTap;
  const _FilterChip(
      {required this.label,
      required this.active,
      required this.cs,
      required this.onTap});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
          decoration: BoxDecoration(
            color: active ? const Color(0xFF3472A4).withValues(alpha: 0.12) : cs.fill,
            borderRadius: BorderRadius.circular(6),
            border: Border.all(
              color: active ? const Color(0xFF3472A4) : cs.border,
            ),
          ),
          child: Text(
            label,
            style: TextStyle(
              fontSize: 12,
              color: active ? const Color(0xFF3472A4) : cs.textSecondary,
              fontWeight: active ? FontWeight.w500 : FontWeight.w400,
            ),
          ),
        ),
      ),
    );
  }
}

class _FieldLabel extends StatelessWidget {
  final String label;
  final dynamic cs;
  final bool required;
  const _FieldLabel(this.label, this.cs, {this.required = false});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Text(label,
            style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w500,
                color: cs.textSecondary)),
        if (required) ...[
          const SizedBox(width: 3),
          const Text('*',
              style: TextStyle(fontSize: 11, color: Color(0xFFEF4444))),
        ],
      ],
    );
  }
}

class _ToggleRow extends StatelessWidget {
  final String label;
  final bool value;
  final dynamic cs;
  final ValueChanged<bool> onChanged;
  const _ToggleRow(this.label, this.value, this.cs, this.onChanged);

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Switch(
            value: value,
            onChanged: onChanged,
            activeThumbColor: const Color(0xFF3472A4)),
        const SizedBox(width: 6),
        Text(label,
            style: TextStyle(fontSize: 13, color: cs.textSecondary)),
      ],
    );
  }
}
