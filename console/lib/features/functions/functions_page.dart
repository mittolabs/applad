import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/id_text.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/search_list.dart';

// ── Colors ────────────────────────────────────────────────────────────────────

const _accent = Color(0xFF3472A4);
const _green = Color(0xFF10B981);
const _red = Color(0xFFEF4444);
const _orange = Color(0xFFF59E0B);

// ── Runtime metadata ──────────────────────────────────────────────────────────

class _Runtime {
  final String id;
  final String label;
  final IconData icon;
  const _Runtime({required this.id, required this.label, required this.icon});
}

const _runtimes = <_Runtime>[
  _Runtime(id: 'node-18',     label: 'Node.js 18',  icon: LucideIcons.server),
  _Runtime(id: 'node-20',     label: 'Node.js 20',  icon: LucideIcons.server),
  _Runtime(id: 'node-22',     label: 'Node.js 22',  icon: LucideIcons.server),
  _Runtime(id: 'bun-1',       label: 'Bun 1',       icon: LucideIcons.zap),
  _Runtime(id: 'python-3.11', label: 'Python 3.11', icon: LucideIcons.code2),
  _Runtime(id: 'python-3.12', label: 'Python 3.12', icon: LucideIcons.code2),
  _Runtime(id: 'go-1.22',     label: 'Go 1.22',     icon: LucideIcons.cpu),
  _Runtime(id: 'dart-3',      label: 'Dart 3',      icon: LucideIcons.terminal),
  _Runtime(id: 'rust-1',      label: 'Rust 1',      icon: LucideIcons.settings2),
  _Runtime(id: 'ruby-3',      label: 'Ruby 3',      icon: LucideIcons.gem),
  _Runtime(id: 'php-8',       label: 'PHP 8',       icon: LucideIcons.fileCode),
  _Runtime(id: 'custom',      label: 'Custom',      icon: LucideIcons.box),
];

_Runtime _runtimeById(String id) =>
    _runtimes.firstWhere((r) => r.id == id, orElse: () => _runtimes.last);

// ── Providers ─────────────────────────────────────────────────────────────────

final _funcSearchProvider   = StateProvider<String>((ref) => '');
final _funcPerPageProvider  = StateProvider<int>((ref) => 12);
final _funcPageProvider     = StateProvider<int>((ref) => 1);
final _funcListTabProvider  = StateProvider<int>((ref) => 0);
final _funcDetailTabProvider = StateProvider<int>((ref) => 0);
final _selectedFuncProvider = StateProvider<Map<String, dynamic>?>((ref) => null);

final functionsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final projectId = ref.watch(currentProjectProvider);
  if (projectId == null) return {'functions': [], 'total': 0};
  api.setProject(projectId);
  final search = ref.watch(_funcSearchProvider);
  final limit  = ref.watch(_funcPerPageProvider);
  final page   = ref.watch(_funcPageProvider);
  final offset = (page - 1) * limit;
  final params = <String, dynamic>{'limit': limit, 'offset': offset};
  if (search.isNotEmpty) params['search'] = search;
  final res = await api.get('/functions', params: params);
  return res.data as Map<String, dynamic>;
});

final _funcExecutionsProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, fnId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/functions/$fnId/executions',
      params: {'limit': 25});
  return res.data as Map<String, dynamic>;
});

// ── Page ──────────────────────────────────────────────────────────────────────

class FunctionsPage extends ConsumerStatefulWidget {
  const FunctionsPage({super.key});

  @override
  ConsumerState<FunctionsPage> createState() => _FunctionsPageState();
}

class _FunctionsPageState extends ConsumerState<FunctionsPage> {
  final _searchCtrl = TextEditingController();

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  void _doSearch() {
    ref.read(_funcSearchProvider.notifier).state = _searchCtrl.text.trim();
    ref.read(_funcPageProvider.notifier).state = 1;
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final selected = ref.watch(_selectedFuncProvider);

    return Scaffold(
      backgroundColor: colors.background,
      body: selected != null
          ? _FuncDetailView(
              fn: selected,
              onBack: () {
                ref.read(_selectedFuncProvider.notifier).state = null;
                ref.read(_funcDetailTabProvider.notifier).state = 0;
              },
            )
          : _buildList(),
    );
  }

  // ── List ────────────────────────────────────────────────────────────────────

  Widget _buildList() {
    final colors = consoleColors(context);
    final functionsAsync = ref.watch(functionsProvider);
    final perPage      = ref.watch(_funcPerPageProvider);
    final currentPage  = ref.watch(_funcPageProvider);
    final listTab      = ref.watch(_funcListTabProvider);

    return Padding(
      padding: EdgeInsets.symmetric(
        horizontal: MediaQuery.of(context).size.width > 1400 ? 80.0 : 40.0,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SizedBox(height: 32),
          Text('Functions',
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 22,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 24),
          PageTabs(
            tabs: const ['Functions', 'Usage'],
            selected: listTab,
            onChanged: (i) =>
                ref.read(_funcListTabProvider.notifier).state = i,
          ),
          const SizedBox(height: 20),
          Expanded(
            child: listTab == 0
                ? _buildFunctionsTab(functionsAsync, perPage, currentPage)
                : const _FuncUsageTab(),
          ),
        ],
      ),
    );
  }

  Widget _buildFunctionsTab(
    AsyncValue<Map<String, dynamic>> functionsAsync,
    int perPage,
    int currentPage,
  ) {
    final colors = consoleColors(context);
    return Column(
      children: [
        SearchListHeader(
          searchController: _searchCtrl,
          total: functionsAsync.whenOrNull(
                  data: (d) => d['total'] as int? ?? 0) ??
              0,
          perPage: perPage,
          currentPage: currentPage,
          onPerPageChanged: (v) {
            ref.read(_funcPerPageProvider.notifier).state = v;
            ref.read(_funcPageProvider.notifier).state = 1;
          },
          onPrev: () =>
              ref.read(_funcPageProvider.notifier).update((s) => s - 1),
          onNext: () =>
              ref.read(_funcPageProvider.notifier).update((s) => s + 1),
          onSearch: _doSearch,
          trailing: FilledButton.icon(
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            onPressed: () => _showCreateDialog(context, ref),
            icon: const Icon(LucideIcons.plus, size: 16),
            label: const Text('Create function'),
          ),
        ),
        const SizedBox(height: 12),
        Expanded(
          child: functionsAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(LucideIcons.alertCircle,
                      size: 48, color: colors.textSubtle),
                  const SizedBox(height: 16),
                  Text('Failed to load functions: $e',
                      style: TextStyle(color: colors.textSecondary)),
                  const SizedBox(height: 8),
                  FilledButton(
                    onPressed: () => ref.invalidate(functionsProvider),
                    child: const Text('Retry'),
                  ),
                ],
              ),
            ),
            data: (data) {
              final fns = List<Map<String, dynamic>>.from(
                  data['functions'] ?? []);
              if (fns.isEmpty) {
                return _buildEmptyState();
              }
              return ListView.separated(
                padding: EdgeInsets.zero,
                itemCount: fns.length,
                separatorBuilder: (_, __) =>
                    const SizedBox(height: 8),
                itemBuilder: (context, i) => _FuncCard(
                  fn: fns[i],
                  onTap: () =>
                      ref.read(_selectedFuncProvider.notifier).state =
                          fns[i],
                  onDelete: () => _delete(ref, fns[i]['\$id'] as String),
                ),
              );
            },
          ),
        ),
        SearchListFooter(
          total: functionsAsync.whenOrNull(
                  data: (d) => d['total'] as int? ?? 0) ??
              0,
          perPage: perPage,
          currentPage: currentPage,
          itemLabel: 'functions',
          onPrev: () =>
              ref.read(_funcPageProvider.notifier).update((s) => s - 1),
          onNext: () =>
              ref.read(_funcPageProvider.notifier).update((s) => s + 1),
          onPerPageChanged: (v) {
            ref.read(_funcPerPageProvider.notifier).state = v;
            ref.read(_funcPageProvider.notifier).state = 1;
          },
        ),
        const SizedBox(height: 12),
      ],
    );
  }

  Widget _buildEmptyState() {
    final colors = consoleColors(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            mainAxisSize: MainAxisSize.min,
            children: _runtimes
                .take(6)
                .map((r) => Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 6),
                      child: Container(
                        width: 44,
                        height: 44,
                        decoration: BoxDecoration(
                          color: colors.surface,
                          borderRadius: BorderRadius.circular(10),
                          border: Border.all(color: colors.border),
                        ),
                        child: Icon(r.icon, size: 20, color: colors.textSubtle),
                      ),
                    ))
                .toList(),
          ),
          const SizedBox(height: 24),
          Text('Create your first function',
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 16,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 8),
          Text(
            'Write backend logic that runs on demand in any language.',
            style: TextStyle(color: colors.textSecondary, fontSize: 13),
          ),
          const SizedBox(height: 20),
          FilledButton.icon(
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            onPressed: () => _showCreateDialog(context, ref),
            icon: const Icon(LucideIcons.plus, size: 16),
            label: const Text('Create function'),
          ),
        ],
      ),
    );
  }

  // ── Create dialog ────────────────────────────────────────────────────────────

  void _showCreateDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl       = TextEditingController();
    final entryCtrl      = TextEditingController(text: 'index.js');
    final timeoutCtrl    = TextEditingController(text: '15');
    final sourceCtrl     = TextEditingController();
    String runtime       = 'node-20';

    showAppDialog(
      context: context,
      title: 'Create function',
      subtitle: 'Deploy serverless backend logic',
      width: 480,
      content: StatefulBuilder(
        builder: (ctx, setState) => Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AppDialogField(
                controller: nameCtrl,
                label: 'Name',
                hint: 'my-function',
                autofocus: true),
            const SizedBox(height: 12),
            _RuntimePicker(
              value: runtime,
              onChanged: (v) => setState(() => runtime = v),
            ),
            const SizedBox(height: 12),
            AppDialogField(
                controller: entryCtrl,
                label: 'Entrypoint',
                hint: 'index.js'),
            const SizedBox(height: 12),
            AppDialogField(
                controller: timeoutCtrl,
                label: 'Timeout (seconds)',
                hint: '15',
                keyboardType: TextInputType.number),
            const SizedBox(height: 12),
            AppDialogField(
                controller: sourceCtrl,
                label: 'Source code',
                hint: '// your code here',
                maxLines: 6),
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            final api = ref.read(apiClientProvider);
            await api.post('/functions', data: {
              'name':       nameCtrl.text.trim(),
              'runtime':    runtime,
              'entrypoint': entryCtrl.text.trim(),
              'timeout':    int.tryParse(timeoutCtrl.text) ?? 15,
              'source':     sourceCtrl.text,
            });
            if (context.mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
            ref.invalidate(functionsProvider);
          },
        ),
      ],
    );
  }

  Future<void> _delete(WidgetRef ref, String id) async {
    final colors = consoleColors(context);
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete function',
      content: Text(
        'This will permanently delete the function and all its executions.',
        style: TextStyle(color: colors.textSecondary),
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
    if (confirmed == true) {
      await ref.read(apiClientProvider).delete('/functions/$id');
      ref.invalidate(functionsProvider);
    }
  }
}

// ── Function card ─────────────────────────────────────────────────────────────

class _FuncCard extends StatefulWidget {
  final Map<String, dynamic> fn;
  final VoidCallback onTap;
  final VoidCallback onDelete;

  const _FuncCard(
      {required this.fn, required this.onTap, required this.onDelete});

  @override
  State<_FuncCard> createState() => _FuncCardState();
}

class _FuncCardState extends State<_FuncCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final fn      = widget.fn;
    final name    = fn['name'] as String? ?? 'Unnamed';
    final runtime = fn['runtime'] as String? ?? 'custom';
    final status  = fn['status'] as String? ?? 'inactive';
    final rt      = _runtimeById(runtime);
    final createdAt = fn['\$createdAt'] as String? ?? '';

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit:  (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: _hovered ? colors.fillHover : colors.surface,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(
              color: _hovered ? colors.fieldBorder : colors.border,
            ),
          ),
          child: Row(
            children: [
              // Runtime icon
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: _accent.withValues(alpha: 0.10),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(rt.icon, size: 18, color: _accent),
              ),
              const SizedBox(width: 14),

              // Name + runtime
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(name,
                      style: TextStyle(
                        color: colors.textPrimary,
                            fontSize: 14,
                            fontWeight: FontWeight.w500)),
                    const SizedBox(height: 2),
                    Text(rt.label,
                      style: TextStyle(
                        color: colors.textSecondary, fontSize: 12)),
                  ],
                ),
              ),

              // Created
              Text(_relativeTime(createdAt),
                    style: TextStyle(color: colors.textSubtle, fontSize: 12)),
              const SizedBox(width: 16),

              // Status
              _StatusBadge(status: status),
              const SizedBox(width: 12),

              // Delete
              if (_hovered)
                IconButton(
                  icon: Icon(LucideIcons.trash2,
                      size: 14, color: colors.textSubtle),
                  onPressed: widget.onDelete,
                  padding: EdgeInsets.zero,
                  constraints: const BoxConstraints(),
                )
              else
                Icon(LucideIcons.chevronRight,
                    size: 14, color: colors.textSubtle),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Status badge ──────────────────────────────────────────────────────────────

class _StatusBadge extends StatelessWidget {
  final String status;
  const _StatusBadge({required this.status});

  @override
  Widget build(BuildContext context) {
    final (label, bg, fg) = switch (status) {
      'active'   => ('Active',   const Color(0xFF064E3B), _green),
      'building' => ('Building', const Color(0xFF451A03), _orange),
      'failed'   => ('Failed',   const Color(0xFF450A0A), _red),
      _          => ('Inactive', const Color(0xFF1F2937), consoleColors(context).textSecondary),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
          color: bg, borderRadius: BorderRadius.circular(4)),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        Container(
            width: 5,
            height: 5,
            decoration: BoxDecoration(color: fg, shape: BoxShape.circle)),
        const SizedBox(width: 5),
        Text(label,
            style: TextStyle(
                color: fg, fontSize: 11, fontWeight: FontWeight.w500)),
      ]),
    );
  }
}

// ── Detail view ───────────────────────────────────────────────────────────────

class _FuncDetailView extends ConsumerStatefulWidget {
  final Map<String, dynamic> fn;
  final VoidCallback onBack;

  const _FuncDetailView({required this.fn, required this.onBack});

  @override
  ConsumerState<_FuncDetailView> createState() => _FuncDetailViewState();
}

class _FuncDetailViewState extends ConsumerState<_FuncDetailView> {
  late Map<String, dynamic> _fn;

  @override
  void initState() {
    super.initState();
    _fn = widget.fn;
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final tab = ref.watch(_funcDetailTabProvider);
    final id  = _fn['\$id'] as String;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Header
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 20, 24, 0),
          child: Row(
            children: [
              GestureDetector(
                onTap: widget.onBack,
                child: MouseRegion(
                  cursor: SystemMouseCursors.click,
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(LucideIcons.chevronLeft,
                        size: 16, color: colors.textSecondary),
                      const SizedBox(width: 4),
                      Text('Functions',
                        style: TextStyle(
                          color: colors.textSecondary, fontSize: 13)),
                    ],
                  ),
                ),
              ),
              const SizedBox(width: 8),
                  Text('/', style: TextStyle(color: colors.textSubtle, fontSize: 13)),
              const SizedBox(width: 8),
              Text(_fn['name'] as String? ?? 'Function',
                    style: TextStyle(
                      color: colors.textPrimary,
                      fontSize: 13,
                      fontWeight: FontWeight.w500)),
              const Spacer(),
              FilledButton.icon(
                style: FilledButton.styleFrom(
                  backgroundColor: _accent,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 14, vertical: 8),
                ),
                onPressed: () => _execute(context, id),
                icon: const Icon(LucideIcons.play, size: 14),
                label: const Text('Execute',
                    style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),
        Padding(
          padding: const EdgeInsets.only(left: 24),
          child: PageTabs(
            tabs: const ['Executions', 'Variables', 'Settings'],
            selected: tab,
            onChanged: (i) =>
                ref.read(_funcDetailTabProvider.notifier).state = i,
          ),
        ),
        const SizedBox(height: 16),
        Expanded(
          child: switch (tab) {
            0 => _ExecutionsTab(fnId: id),
            1 => _VariablesTab(fn: _fn, onUpdated: (updated) => setState(() => _fn = updated)),
            _ => _FuncSettingsTab(fn: _fn, onUpdated: (updated) => setState(() => _fn = updated)),
          },
        ),
      ],
    );
  }

  Future<void> _execute(BuildContext context, String id) async {
    try {
      await ref
          .read(apiClientProvider)
          .post('/functions/$id/executions', data: <String, dynamic>{});
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Execution triggered')),
        );
      }
      ref.invalidate(_funcExecutionsProvider(id));
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed: $e'),
              backgroundColor: _red),
        );
      }
    }
  }
}

// ── Executions tab ────────────────────────────────────────────────────────────

class _ExecutionsTab extends ConsumerWidget {
  final String fnId;
  const _ExecutionsTab({required this.fnId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final execsAsync = ref.watch(_funcExecutionsProvider(fnId));

    return execsAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
        child: Text('Failed to load executions: $e',
        style: TextStyle(color: colors.textSecondary)),
      ),
      data: (data) {
        final execs = List<Map<String, dynamic>>.from(
            data['executions'] ?? []);

        if (execs.isEmpty) {
          return Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(LucideIcons.activity,
                    size: 36, color: colors.textSubtle),
                const SizedBox(height: 12),
                Text('No executions yet',
                    style: TextStyle(
                        color: colors.textPrimary,
                        fontSize: 15,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 4),
                Text('Trigger your first execution above.',
                    style: TextStyle(color: colors.textSecondary, fontSize: 13)),
              ],
            ),
          );
        }

        return Column(
          children: [
            // Table header
            Container(
              color: colors.surfaceAlt,
              padding: const EdgeInsets.symmetric(
                  horizontal: 24, vertical: 10),
              child: Row(children: [
                Expanded(
                    flex: 3,
                    child: Text('Execution ID',
                        style: TextStyle(
                    color: colors.textSecondary,
                            fontSize: 11,
                            fontWeight: FontWeight.w600))),
                Expanded(
                    flex: 2,
                    child: Text('Status',
                        style: TextStyle(
                    color: colors.textSecondary,
                            fontSize: 11,
                            fontWeight: FontWeight.w600))),
                Expanded(
                    flex: 2,
                    child: Text('Duration',
                        style: TextStyle(
                    color: colors.textSecondary,
                            fontSize: 11,
                            fontWeight: FontWeight.w600))),
                Expanded(
                    flex: 3,
                    child: Text('Triggered',
                        style: TextStyle(
                    color: colors.textSecondary,
                            fontSize: 11,
                            fontWeight: FontWeight.w600))),
              ]),
            ),
            Divider(height: 1, color: colors.border),
            Expanded(
              child: ListView.separated(
                itemCount: execs.length,
                separatorBuilder: (_, __) =>
                Divider(height: 1, color: colors.border),
                itemBuilder: (context, i) =>
                    _ExecutionRow(exec: execs[i]),
              ),
            ),
          ],
        );
      },
    );
  }
}

class _ExecutionRow extends StatefulWidget {
  final Map<String, dynamic> exec;
  const _ExecutionRow({required this.exec});

  @override
  State<_ExecutionRow> createState() => _ExecutionRowState();
}

class _ExecutionRowState extends State<_ExecutionRow> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final e      = widget.exec;
    final id     = e['\$id'] as String? ?? '';
    final status = e['status'] as String? ?? 'pending';
    final dur    = e['duration'] as num? ?? 0;
    final ts     = e['\$createdAt'] as String? ?? '';
    final output = e['output'] as String? ?? '';
    final error  = e['errors'] as String? ?? '';

    return Column(
      children: [
        InkWell(
          onTap: () => setState(() => _expanded = !_expanded),
          child: Padding(
            padding: const EdgeInsets.symmetric(
                horizontal: 24, vertical: 12),
            child: Row(children: [
              Expanded(
                flex: 3,
                child: IdText(id: id, fontSize: 12),
              ),
              Expanded(flex: 2, child: _StatusBadge(status: status)),
              Expanded(
                flex: 2,
                child: Text('${dur.toStringAsFixed(0)} ms',
                    style: TextStyle(
                        color: colors.textSecondary, fontSize: 12)),
              ),
              Expanded(
                flex: 3,
                child: Text(_relativeTime(ts),
                    style: TextStyle(
                        color: colors.textSubtle, fontSize: 12)),
              ),
              Icon(
                _expanded
                    ? LucideIcons.chevronUp
                    : LucideIcons.chevronDown,
                size: 14,
                color: colors.textSubtle,
              ),
            ]),
          ),
        ),
        if (_expanded && (output.isNotEmpty || error.isNotEmpty))
          Container(
            width: double.infinity,
            margin: const EdgeInsets.fromLTRB(24, 0, 24, 8),
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: colors.surfaceAlt,
              borderRadius: BorderRadius.circular(6),
              border: Border.all(color: colors.border),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (output.isNotEmpty)
                  SelectableText(output,
                      style: TextStyle(
                          color: colors.textPrimary,
                          fontSize: 12,
                          fontFamily: 'monospace')),
                if (error.isNotEmpty)
                  SelectableText(error,
                      style: const TextStyle(
                          color: _red,
                          fontSize: 12,
                          fontFamily: 'monospace')),
              ],
            ),
          ),
      ],
    );
  }
}

// ── Variables tab ─────────────────────────────────────────────────────────────

class _VariablesTab extends ConsumerStatefulWidget {
  final Map<String, dynamic> fn;
  final ValueChanged<Map<String, dynamic>> onUpdated;

  const _VariablesTab({required this.fn, required this.onUpdated});

  @override
  ConsumerState<_VariablesTab> createState() => _VariablesTabState();
}

class _VariablesTabState extends ConsumerState<_VariablesTab> {
  late Map<String, String> _vars;

  @override
  void initState() {
    super.initState();
    final raw = widget.fn['envVars'] as Map<String, dynamic>? ?? {};
    _vars = raw.map((k, v) => MapEntry(k, v.toString()));
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 4, 24, 16),
          child: Row(
            children: [
              Text('${_vars.length} variable${_vars.length == 1 ? '' : 's'}',
                  style: TextStyle(color: colors.textSecondary, fontSize: 13)),
              const Spacer(),
              OutlinedButton.icon(
                style: OutlinedButton.styleFrom(
                  foregroundColor: colors.textSecondary,
                  side: BorderSide(color: colors.border),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 14, vertical: 8),
                ),
                onPressed: () => _showAddVar(context),
                icon: const Icon(LucideIcons.plus, size: 14),
                label: const Text('Add variable',
                    style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
        ),
        if (_vars.isEmpty)
          Expanded(
            child: Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(LucideIcons.variable,
                    size: 32, color: colors.textSubtle),
                  const SizedBox(height: 12),
                  Text('No variables yet',
                    style: TextStyle(
                      color: colors.textPrimary,
                          fontSize: 15,
                          fontWeight: FontWeight.w500)),
                  const SizedBox(height: 4),
                  Text(
                    'Add environment variables for your function to use at runtime.',
                  style: TextStyle(color: colors.textSecondary, fontSize: 13),
                    textAlign: TextAlign.center,
                  ),
                ],
              ),
            ),
          )
        else
          Expanded(
            child: ListView(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              children: _vars.entries
                  .map((e) => _VarRow(
                        varKey: e.key,
                        varValue: e.value,
                        onDelete: () => _deleteVar(e.key),
                      ))
                  .toList(),
            ),
          ),
      ],
    );
  }

  void _showAddVar(BuildContext context) {
    final keyCtrl = TextEditingController();
    final valCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Add variable',
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          AppDialogField(
              controller: keyCtrl,
              label: 'Key',
              hint: 'API_KEY',
              autofocus: true),
          const SizedBox(height: 12),
          AppDialogField(
              controller: valCtrl,
              label: 'Value',
              hint: 'your-secret-value'),
        ],
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Add',
          onTap: () async {
            final key = keyCtrl.text.trim();
            final val = valCtrl.text;
            if (key.isEmpty) return;
            await _saveVars({..._vars, key: val});
            if (context.mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
          },
        ),
      ],
    );
  }

  Future<void> _deleteVar(String key) async {
    final updated = Map<String, String>.from(_vars)..remove(key);
    await _saveVars(updated);
  }

  Future<void> _saveVars(Map<String, String> newVars) async {
    final id = widget.fn['\$id'] as String;
    await ref.read(apiClientProvider).put('/functions/$id', data: {
      ...widget.fn,
      'envVars': newVars,
    });
    setState(() => _vars = newVars);
    widget.onUpdated({...widget.fn, 'envVars': newVars});
  }
}

class _VarRow extends StatelessWidget {
  final String varKey;
  final String varValue;
  final VoidCallback onDelete;

  const _VarRow(
      {required this.varKey,
      required this.varValue,
      required this.onDelete});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colors.border),
      ),
      child: Row(children: [
        Expanded(
          flex: 2,
          child: Text(varKey,
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 13,
                  fontFamily: 'monospace',
                  fontWeight: FontWeight.w500)),
        ),
        Expanded(
          flex: 3,
          child: Text(
            varValue.length > 32
                ? '${varValue.substring(0, 32)}…'
                : varValue,
            style: TextStyle(
              color: colors.textSecondary, fontSize: 13, fontFamily: 'monospace'),
          ),
        ),
        IconButton(
            icon: Icon(LucideIcons.trash2,
              size: 14, color: colors.textSubtle),
          onPressed: onDelete,
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(),
        ),
      ]),
    );
  }
}

// ── Settings tab ──────────────────────────────────────────────────────────────

class _FuncSettingsTab extends ConsumerStatefulWidget {
  final Map<String, dynamic> fn;
  final ValueChanged<Map<String, dynamic>> onUpdated;

  const _FuncSettingsTab({required this.fn, required this.onUpdated});

  @override
  ConsumerState<_FuncSettingsTab> createState() => _FuncSettingsTabState();
}

class _FuncSettingsTabState extends ConsumerState<_FuncSettingsTab> {
  late TextEditingController _nameCtrl;
  late TextEditingController _entryCtrl;
  late TextEditingController _timeoutCtrl;
  late TextEditingController _sourceCtrl;
  late TextEditingController _cronCtrl;
  late String _runtime;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    final fn = widget.fn;
    _nameCtrl    = TextEditingController(text: fn['name'] ?? '');
    _entryCtrl   = TextEditingController(text: fn['entrypoint'] ?? '');
    _timeoutCtrl = TextEditingController(text: '${fn['timeout'] ?? 15}');
    _sourceCtrl  = TextEditingController(text: fn['source'] ?? '');
    _cronCtrl    = TextEditingController(text: fn['cron'] ?? '');
    _runtime     = fn['runtime'] ?? 'node-20';
  }

  @override
  void dispose() {
    _nameCtrl.dispose();
    _entryCtrl.dispose();
    _timeoutCtrl.dispose();
    _sourceCtrl.dispose();
    _cronCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(24, 4, 24, 24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _sectionCard(
            title: 'General',
            description: 'Basic function configuration.',
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _field('Name', _nameCtrl, 'my-function'),
                const SizedBox(height: 12),
                _label('Runtime'),
                const SizedBox(height: 6),
                _RuntimePicker(
                  value: _runtime,
                  onChanged: (v) => setState(() => _runtime = v),
                ),
                const SizedBox(height: 12),
                _field('Entrypoint', _entryCtrl, 'index.js'),
                const SizedBox(height: 12),
                _field('Timeout (seconds)', _timeoutCtrl, '15',
                    keyboard: TextInputType.number),
                const SizedBox(height: 12),
                _field('Schedule (cron)', _cronCtrl, '0 * * * *'),
                const SizedBox(height: 4),
                Text(
                  'Standard 5-field cron expression (minute hour day month weekday). Leave empty for manual execution only.',
                  style: TextStyle(
                      color: consoleColors(context).textSubtle, fontSize: 11),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          _sectionCard(
            title: 'Source code',
            description: 'Inline source for simple functions.',
            child: TextField(
              controller: _sourceCtrl,
              maxLines: null,
              minLines: 8,
                style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 13,
                  fontFamily: 'monospace'),
              decoration: InputDecoration(
                hintText: '// your function code',
                hintStyle: TextStyle(color: colors.textSubtle),
                filled: true,
                fillColor: colors.surfaceAlt,
                isDense: true,
                contentPadding: const EdgeInsets.all(12),
                border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                  borderSide: BorderSide(color: colors.fieldBorder)),
                enabledBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                  borderSide: BorderSide(color: colors.fieldBorder)),
                focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: const BorderSide(color: _accent)),
              ),
            ),
          ),
          const SizedBox(height: 20),
          SizedBox(
            height: 36,
            child: FilledButton(
              style: FilledButton.styleFrom(
                backgroundColor: _accent,
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
                padding: const EdgeInsets.symmetric(horizontal: 20),
              ),
              onPressed: _saving ? null : _save,
              child: _saving
                  ? SizedBox(
                      width: 14,
                      height: 14,
                      child: CircularProgressIndicator(
                      strokeWidth: 2, color: colors.textPrimary))
                  : const Text('Save changes',
                      style: TextStyle(fontSize: 13)),
            ),
          ),

          // Danger zone
          const SizedBox(height: 32),
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: _red.withValues(alpha: 0.05),
              borderRadius: BorderRadius.circular(8),
              border:
                  Border.all(color: _red.withValues(alpha: 0.2)),
            ),
            child: Row(children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text('Delete function',
                        style: TextStyle(
                            color: _red,
                            fontSize: 14,
                            fontWeight: FontWeight.w600)),
                    const SizedBox(height: 2),
                    Text(
                      'Permanently removes the function and all execution history.',
                      style: TextStyle(color: consoleColors(context).textSubtle, fontSize: 12),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 16),
              OutlinedButton(
                style: OutlinedButton.styleFrom(
                  foregroundColor: _red,
                  side: BorderSide(color: _red.withValues(alpha: 0.5)),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 14, vertical: 8),
                ),
                onPressed: () => _deleteFunction(context),
                child: const Text('Delete', style: TextStyle(fontSize: 13)),
              ),
            ]),
          ),
        ],
      ),
    );
  }

  Widget _sectionCard(
      {required String title,
      required String description,
      required Widget child}) {
    final colors = consoleColors(context);
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title,
              style: TextStyle(
                color: colors.textPrimary,
                  fontSize: 14,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 2),
          Text(description,
              style: TextStyle(color: colors.textSubtle, fontSize: 12)),
          const SizedBox(height: 16),
          child,
        ],
      ),
    );
  }

  Widget _label(String text) => Text(text,
      style: TextStyle(
            color: consoleColors(context).textSecondary,
          fontSize: 12,
          fontWeight: FontWeight.w500));

  Widget _field(String label, TextEditingController ctrl, String hint,
      {TextInputType? keyboard}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _label(label),
        const SizedBox(height: 6),
        TextField(
          controller: ctrl,
          keyboardType: keyboard,
          style: TextStyle(color: consoleColors(context).textPrimary, fontSize: 13),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: TextStyle(
            color: consoleColors(context).textSubtle,
                fontSize: 13),
            filled: true,
          fillColor: consoleColors(context).fieldFill,
            isDense: true,
            contentPadding: const EdgeInsets.symmetric(
                horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: consoleColors(context).fieldBorder)),
            enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: consoleColors(context).fieldBorder)),
            focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: _accent)),
          ),
        ),
      ],
    );
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final id = widget.fn['\$id'] as String;
      final res = await ref.read(apiClientProvider).put('/functions/$id', data: {
        'name':       _nameCtrl.text.trim(),
        'runtime':    _runtime,
        'entrypoint': _entryCtrl.text.trim(),
        'timeout':    int.tryParse(_timeoutCtrl.text) ?? 15,
        'source':     _sourceCtrl.text,
        'cron':       _cronCtrl.text.trim(),
      });
      final updated = res.data as Map<String, dynamic>;
      widget.onUpdated(updated);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Function updated')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
              content: Text('Failed: $e'),
              backgroundColor: _red),
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _deleteFunction(BuildContext context) async {
    final colors = consoleColors(context);
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete function',
      content: Text(
        'This will permanently delete the function and all its executions.',
        style: TextStyle(color: colors.textSecondary),
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
    if (confirmed == true) {
      final id = widget.fn['\$id'] as String;
      await ref.read(apiClientProvider).delete('/functions/$id');
      ref.invalidate(functionsProvider);
      // Navigate back to list
      ref.read(_selectedFuncProvider.notifier).state = null;
    }
  }
}

// ── Usage tab ─────────────────────────────────────────────────────────────────

class _FuncUsageTab extends StatelessWidget {
  const _FuncUsageTab();

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
            Text('Usage',
              style: TextStyle(
                color: colors.textPrimary,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
            Text('Function execution metrics for the past 30 days.',
              style: TextStyle(color: colors.textSecondary, fontSize: 13)),
          const SizedBox(height: 24),
          Row(children: [
            _statCard(context, 'Total executions', '—', LucideIcons.activity),
            const SizedBox(width: 12),
            _statCard(context, 'Avg duration', '—', LucideIcons.timer),
            const SizedBox(width: 12),
            _statCard(context, 'Failure rate', '—', LucideIcons.alertTriangle),
          ]),
          const SizedBox(height: 24),
          Container(
            height: 180,
            decoration: BoxDecoration(
              color: colors.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: colors.border),
            ),
            child: Center(
              child: Text('Usage charts coming soon',
                  style: TextStyle(color: colors.textSubtle, fontSize: 13)),
            ),
          ),
        ],
      ),
    );
  }

  Widget _statCard(BuildContext context, String label, String value, IconData icon) {
    final colors = consoleColors(context);
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border),
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Icon(icon, size: 16, color: colors.textSecondary),
          const SizedBox(height: 12),
          Text(value,
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 24,
                  fontWeight: FontWeight.w700)),
          const SizedBox(height: 4),
          Text(label,
              style: TextStyle(color: colors.textSecondary, fontSize: 12)),
        ]),
      ),
    );
  }
}

// ── Runtime picker ────────────────────────────────────────────────────────────

class _RuntimePicker extends StatelessWidget {
  final String value;
  final ValueChanged<String> onChanged;

  const _RuntimePicker({required this.value, required this.onChanged});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return DropdownButtonFormField<String>(
      value: value,
      dropdownColor: colors.popupSurface,
      style: TextStyle(color: colors.textPrimary, fontSize: 13),
      decoration: InputDecoration(
        filled: true,
        fillColor: colors.fieldFill,
        isDense: true,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: colors.fieldBorder)),
        enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: colors.fieldBorder)),
        focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: const BorderSide(color: _accent)),
      ),
      items: _runtimes
          .map((r) => DropdownMenuItem(value: r.id, child: Text(r.label)))
          .toList(),
      onChanged: (v) => onChanged(v ?? value),
    );
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────────

String _relativeTime(String iso) {
  if (iso.isEmpty) return '—';
  try {
    final dt   = DateTime.parse(iso);
    final diff = DateTime.now().difference(dt);
    if (diff.inDays > 365) return '${(diff.inDays / 365).floor()}y ago';
    if (diff.inDays > 30)  return '${(diff.inDays / 30).floor()}mo ago';
    if (diff.inDays > 0)   return '${diff.inDays}d ago';
    if (diff.inHours > 0)  return '${diff.inHours}h ago';
    if (diff.inMinutes > 0) return '${diff.inMinutes}m ago';
    return 'just now';
  } catch (_) {
    return '—';
  }
}
