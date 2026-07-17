import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/id_text.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/app_error_state.dart';

// --- Constants ---------------------------------------------------------------

const _accent = Color(0xFF3472A4);
const _red = Color(0xFFEF4444);

// --- Providers ---------------------------------------------------------------

final _deploySearchProvider = StateProvider<String>((ref) => '');
final _deployPerPageProvider = StateProvider<int>((ref) => 12);
final _deployPageProvider = StateProvider<int>((ref) => 1);

final deploymentsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final search = ref.watch(_deploySearchProvider);
  final limit = ref.watch(_deployPerPageProvider);
  final page = ref.watch(_deployPageProvider);
  final offset = (page - 1) * limit;
  final params = <String, dynamic>{'limit': limit, 'offset': offset};
  if (search.isNotEmpty) params['search'] = search;
  final res = await api.get('/deploy', params: params);
  return res.data as Map<String, dynamic>;
});

final _deployDetailProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, id) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy/$id');
  return res.data as Map<String, dynamic>;
});

// --- Page --------------------------------------------------------------------

class DeployPage extends ConsumerStatefulWidget {
  const DeployPage({super.key});

  @override
  ConsumerState<DeployPage> createState() => _DeployPageState();
}

class _DeployPageState extends ConsumerState<DeployPage> {
  final _searchCtrl = TextEditingController();
  String? _selectedDeployId;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final urlPage = pageFromQuery(context);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (ref.read(_deployPageProvider) != urlPage) {
        ref.read(_deployPageProvider.notifier).state = urlPage;
      }
    });
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  void _doSearch() {
    ref.read(_deploySearchProvider.notifier).state = _searchCtrl.text.trim();
    ref.read(_deployPageProvider.notifier).state = 1;
  }

  @override
  Widget build(BuildContext context) {
    if (_selectedDeployId != null) {
      return _DeployDetailView(
        deployId: _selectedDeployId!,
        onBack: () => setState(() => _selectedDeployId = null),
      );
    }
    return _buildDeployList();
  }

  Widget _buildDeployList() {
    final cs = consoleColors(context);
    final deploymentsAsync = ref.watch(deploymentsProvider);
    final perPage = ref.watch(_deployPerPageProvider);
    final currentPage = ref.watch(_deployPageProvider);
    final total =
        deploymentsAsync.whenOrNull(data: (d) => d['total'] as int? ?? 0) ?? 0;
    final deployments = deploymentsAsync.valueOrNull == null
        ? <Map<String, dynamic>>[]
        : List<Map<String, dynamic>>.from(
            deploymentsAsync.valueOrNull!['deployments'] ?? []);

    return Scaffold(
      backgroundColor: cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Text('Deploy',
                style: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 24),
            PageTabs(
              tabs: const ['Deployments', 'Activity'],
              selected: 0,
              onChanged: (_) {},
            ),
            const SizedBox(height: 20),
            if (deploymentsAsync.isLoading)
              const Expanded(
                  child: Center(child: CircularProgressIndicator()))
            else if (deploymentsAsync.hasError)
              Expanded(child: AppErrorState(error: deploymentsAsync.error!))
            else
              Expanded(
                child: AppDataTable(
                  columns: const [
                    AppTableColumn(key: r'$id',     label: 'Deployment ID', flex: 3),
                    AppTableColumn(key: 'name',      label: 'Name',         flex: 3),
                    AppTableColumn(key: 'type',      label: 'Type',         flex: 2),
                    AppTableColumn(key: 'status',    label: 'Status',       flex: 2, sortable: false),
                    AppTableColumn(key: 'createdAt', label: 'Created',      flex: 2),
                  ],
                  rows: deployments,
                  getCellValue: (row, key) => switch (key) {
                    r'$id'      => row[r'$id'] as String? ?? '',
                    'name'      => row['name'] as String? ?? '',
                    'type'      => row['type'] as String? ?? 'web',
                    'status'    => row['status'] as String? ?? 'pending',
                    'createdAt' => _deployFmtDate(
                        row['createdAt'] ?? row[r'$createdAt']),
                    _           => '',
                  },
                  getRowIcon: (row) =>
                      _iconForType(row['type'] as String? ?? 'web'),
                  cellBuilder: (row, key) {
                    if (key == 'status') {
                      return _StatusBadge(
                          status: row['status'] as String? ?? 'pending');
                    }
                    return null; // fallback to default rendering
                  },
                  onRowTap: (row) =>
                      setState(() => _selectedDeployId = row[r'$id'] as String?),
                  onDeleteRow: (row) =>
                      _deleteDeployment(row[r'$id'] as String),
                  createLabel: 'Create deployment',
                  onCreateTap: _showCreateDialog,
                  filters: const [
                    AppTableFilter(
                      key: 'type',
                      label: 'Type',
                      options: ['web', 'function', 'container'],
                    ),
                    AppTableFilter(
                      key: 'status',
                      label: 'Status',
                      options: [
                        'pending', 'building', 'deploying', 'active', 'failed'
                      ],
                    ),
                  ],
                  total: total,
                  perPage: perPage,
                  currentPage: currentPage,
                  onPrev: () {
                    final p = currentPage - 1;
                    ref.read(_deployPageProvider.notifier).state = p;
                    context.go(withQuery(context, {'page': '$p'}));
                  },
                  onNext: () {
                    final p = currentPage + 1;
                    ref.read(_deployPageProvider.notifier).state = p;
                    context.go(withQuery(context, {'page': '$p'}));
                  },
                  onPerPageChanged: (v) {
                    ref.read(_deployPerPageProvider.notifier).state = v;
                    ref.read(_deployPageProvider.notifier).state = 1;
                  },
                  itemLabel: 'Deployments',
                  searchController: _searchCtrl,
                  onSearch: _doSearch,
                  emptyIcon: LucideIcons.rocket,
                  emptyTitle: 'No deployments',
                  emptySubtitle: 'Create a deployment to get started',
                  gridCardBuilder: (row) => _DeployGridCard(
                    deployment: row,
                    onTap: () => setState(
                        () => _selectedDeployId = row[r'$id'] as String?),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  void _showCreateDialog() {
    showAppDialog(
      context: context,
      title: 'Create deployment',
      subtitle: 'Deploy your application to the platform',
      content: _CreateDeployContent(
        onSubmit: (name, type) async {
          final api = ref.read(apiClientProvider);
          await api.post('/deploy', data: {
            'name': name,
            'type': type,
            'config': <String, dynamic>{},
          });
          if (mounted) Navigator.of(context, rootNavigator: true).pop();
          ref.invalidate(deploymentsProvider);
        },
      ),
    );
  }

  Future<void> _deleteDeployment(String id) async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete deployment',
      content: Text(
          'This deployment will be permanently deleted. This action is irreversible.',
          style: TextStyle(color: consoleColors(context).textSecondary)),
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
      await ref.read(apiClientProvider).delete('/deploy/$id');
      ref.invalidate(deploymentsProvider);
    }
  }
}

// =============================================================================
// Create dialog content
// =============================================================================

class _CreateDeployContent extends ConsumerStatefulWidget {
  final Future<void> Function(String name, String type) onSubmit;

  const _CreateDeployContent({required this.onSubmit});

  @override
  ConsumerState<_CreateDeployContent> createState() =>
      _CreateDeployContentState();
}

class _CreateDeployContentState extends ConsumerState<_CreateDeployContent> {
  final _nameCtrl = TextEditingController();
  String _type = 'web';
  bool _loading = false;

  @override
  void dispose() {
    _nameCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        AppDialogField(
          controller: _nameCtrl,
          label: 'Name',
          hint: 'e.g. my-app',
          autofocus: true,
        ),
        const SizedBox(height: 16),
        Text('Type',
            style: TextStyle(
                color: cs.textSecondary,
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        DropdownButtonFormField<String>(
          initialValue: _type,
          dropdownColor: cs.popupSurface,
          style: TextStyle(color: cs.textPrimary, fontSize: 13),
          decoration: InputDecoration(
            filled: true,
            fillColor: cs.fieldFill,
            isDense: true,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
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
          ),
          items: const [
            DropdownMenuItem(value: 'web', child: Text('Web')),
            DropdownMenuItem(value: 'function', child: Text('Function')),
            DropdownMenuItem(value: 'container', child: Text('Container')),
          ],
          onChanged: (v) => setState(() => _type = v ?? 'web'),
        ),
        const SizedBox(height: 4),
        Row(
          mainAxisAlignment: MainAxisAlignment.end,
          children: [
            const AppDialogCancel(),
            AppDialogAction(
              label: 'Create',
              loading: _loading,
              onTap: () async {
                if (_nameCtrl.text.trim().isEmpty) return;
                setState(() => _loading = true);
                await widget.onSubmit(_nameCtrl.text.trim(), _type);
                if (mounted) setState(() => _loading = false);
              },
            ),
          ],
        ),
      ],
    );
  }
}

// =============================================================================
// Shared helpers
// =============================================================================

IconData _iconForType(String type) => switch (type) {
      'web'       => LucideIcons.globe,
      'function'  => LucideIcons.zap,
      'container' => LucideIcons.box,
      _           => LucideIcons.rocket,
    };

String _deployFmtDate(dynamic raw) {
  if (raw == null) return '—';
  try {
    final dt = raw is DateTime ? raw : DateTime.parse(raw.toString());
    const m = ['', 'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
                'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    return '${m[dt.month.clamp(1, 12)]} ${dt.day}, ${dt.year}';
  } catch (_) {
    return '—';
  }
}

// =============================================================================
// Deploy grid card (for grid view)
// =============================================================================

class _DeployGridCard extends StatefulWidget {
  final Map<String, dynamic> deployment;
  final VoidCallback onTap;

  const _DeployGridCard({required this.deployment, required this.onTap});

  @override
  State<_DeployGridCard> createState() => _DeployGridCardState();
}

class _DeployGridCardState extends State<_DeployGridCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final id = widget.deployment[r'$id'] as String? ?? '';
    final name = widget.deployment['name'] as String? ?? '';
    final type = widget.deployment['type'] as String? ?? 'web';
    final status = widget.deployment['status'] as String? ?? 'pending';

    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: _hovered ? cs.fillHover : cs.surface,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(
                color: _hovered ? _accent.withValues(alpha: 0.35) : cs.border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Icon(_iconForType(type), size: 18, color: _accent),
                  const Spacer(),
                  _StatusBadge(status: status),
                ],
              ),
              const SizedBox(height: 10),
              Text(name,
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 14,
                      fontWeight: FontWeight.w500),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis),
              const Spacer(),
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
                decoration: BoxDecoration(
                  color: cs.fill,
                  borderRadius: BorderRadius.circular(5),
                  border: Border.all(color: cs.border),
                ),
                child: IdText(id: id, fontSize: 11),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// =============================================================================
// Status Badge
// =============================================================================

class _StatusBadge extends StatelessWidget {
  final String status;
  const _StatusBadge({required this.status});

  @override
  Widget build(BuildContext context) {
    final (color, label) = switch (status) {
      'active' => (const Color(0xFF22C55E), 'Active'),
      'building' => (const Color(0xFFF97316), 'Building'),
      'deploying' => (const Color(0xFF3B82F6), 'Deploying'),
      'failed' => (_red, 'Failed'),
      _ => (const Color(0xFF6B7280), 'Pending'),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withValues(alpha: 0.3)),
      ),
      child: Text(label,
          style: TextStyle(
              color: color, fontSize: 11, fontWeight: FontWeight.w500)),
    );
  }
}

// =============================================================================
// Deploy Detail View
// =============================================================================

class _DeployDetailView extends ConsumerStatefulWidget {
  final String deployId;
  final VoidCallback onBack;

  const _DeployDetailView({
    required this.deployId,
    required this.onBack,
  });

  @override
  ConsumerState<_DeployDetailView> createState() => _DeployDetailViewState();
}

class _DeployDetailViewState extends ConsumerState<_DeployDetailView> {
  int _tabIndex = 0;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final deployAsync = ref.watch(_deployDetailProvider(widget.deployId));
    final deployName =
        deployAsync.valueOrNull?['name'] as String? ?? widget.deployId;

    return Scaffold(
      backgroundColor: cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            // Back + title
            Row(
              children: [
                GestureDetector(
                  onTap: widget.onBack,
                  child: MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: Icon(LucideIcons.arrowLeft,
                        size: 20, color: cs.textMuted),
                  ),
                ),
                const SizedBox(width: 12),
                Text(deployName,
                    style: TextStyle(
                        color: cs.textPrimary,
                        fontSize: 22,
                        fontWeight: FontWeight.w600)),
                const SizedBox(width: 12),
                Row(
                  children: [
                    Icon(LucideIcons.rocket,
                        size: 13, color: cs.textSubtle),
                    const SizedBox(width: 4),
                    IdText(id: widget.deployId),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 24),
            PageTabs(
              tabs: const ['Overview', 'Settings'],
              selected: _tabIndex,
              onChanged: (i) => setState(() => _tabIndex = i),
            ),
            const SizedBox(height: 20),
            Expanded(
              child: _tabIndex == 0
                  ? _buildOverviewTab(deployAsync)
                  : _buildSettingsTab(deployAsync),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildOverviewTab(AsyncValue<Map<String, dynamic>> deployAsync) {
    final cs = consoleColors(context);
    return deployAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
          child: Text('Error: $e', style: const TextStyle(color: _red))),
      data: (deploy) {
        final status = deploy['status'] as String? ?? 'pending';
        final type = deploy['type'] as String? ?? 'web';
        final created = deploy['createdAt'] ?? deploy['\$createdAt'] ?? '—';
        final updated = deploy['updatedAt'] ?? deploy['\$updatedAt'] ?? '—';

        return SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Status card
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(20),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: cs.border),
                ),
                child: Row(
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('Status',
                              style: TextStyle(
                                  color: cs.textMuted,
                                  fontSize: 12,
                                  fontWeight: FontWeight.w500)),
                          const SizedBox(height: 8),
                          _StatusBadge(status: status),
                        ],
                      ),
                    ),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('Type',
                              style: TextStyle(
                                  color: cs.textMuted,
                                  fontSize: 12,
                                  fontWeight: FontWeight.w500)),
                          const SizedBox(height: 8),
                          Text(type,
                              style: TextStyle(
                                  color: cs.textPrimary, fontSize: 13)),
                        ],
                      ),
                    ),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('Created',
                              style: TextStyle(
                                  color: cs.textMuted,
                                  fontSize: 12,
                                  fontWeight: FontWeight.w500)),
                          const SizedBox(height: 8),
                          Text(_fmt(created),
                              style: TextStyle(
                                  color: cs.textSecondary, fontSize: 13)),
                        ],
                      ),
                    ),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('Updated',
                              style: TextStyle(
                                  color: cs.textMuted,
                                  fontSize: 12,
                                  fontWeight: FontWeight.w500)),
                          const SizedBox(height: 8),
                          Text(_fmt(updated),
                              style: TextStyle(
                                  color: cs.textSecondary, fontSize: 13)),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildSettingsTab(AsyncValue<Map<String, dynamic>> deployAsync) {
    final cs = consoleColors(context);
    return deployAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
          child: Text('Error: $e', style: const TextStyle(color: _red))),
      data: (deploy) {
        final name = deploy['name'] as String? ?? '';
        final updated = deploy['updatedAt'] ?? deploy['\$updatedAt'] ?? '—';

        return SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Name section
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(20),
                margin: const EdgeInsets.only(bottom: 16),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: cs.border),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Name',
                        style: TextStyle(
                            color: cs.textPrimary,
                            fontSize: 14,
                            fontWeight: FontWeight.w500)),
                    const SizedBox(height: 4),
                    Text('Update the display name for this deployment.',
                        style:
                            TextStyle(color: cs.textSubtle, fontSize: 12)),
                    const SizedBox(height: 16),
                    _NameField(
                      initialValue: name,
                      onSave: (v) => _updateDeploy({'name': v}),
                    ),
                  ],
                ),
              ),

              // Delete section
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(20),
                margin: const EdgeInsets.only(bottom: 40),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _red.withValues(alpha: 0.3)),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              const Text('Delete deployment',
                                  style: TextStyle(
                                      color: _red,
                                      fontSize: 14,
                                      fontWeight: FontWeight.w500)),
                              const SizedBox(height: 4),
                              Text(
                                  'The deployment will be permanently deleted. This action is irreversible.',
                                  style: TextStyle(
                                      color: cs.textSecondary,
                                      fontSize: 13)),
                            ],
                          ),
                        ),
                        const SizedBox(width: 16),
                        Container(
                          padding: const EdgeInsets.all(12),
                          decoration: BoxDecoration(
                            color: cs.fill,
                            borderRadius: BorderRadius.circular(8),
                            border: Border.all(color: cs.border),
                          ),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(name,
                                  style: TextStyle(
                                      color: cs.textPrimary,
                                      fontSize: 13,
                                      fontWeight: FontWeight.w500)),
                              Text('Last updated: ${_fmt(updated)}',
                                  style: TextStyle(
                                      color: cs.textSubtle,
                                      fontSize: 11)),
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
                          final cs2 = consoleColors(context);
                          final confirmed = await showAppDialog<bool>(
                            context: context,
                            title: 'Delete deployment',
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
                          if (confirmed != true) return;
                          final api = ref.read(apiClientProvider);
                          await api.delete('/deploy/${widget.deployId}');
                          ref.invalidate(deploymentsProvider);
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
      },
    );
  }

  Future<void> _updateDeploy(Map<String, dynamic> data) async {
    try {
      final api = ref.read(apiClientProvider);
      await api.patch('/deploy/${widget.deployId}', data: data);
      ref.invalidate(_deployDetailProvider(widget.deployId));
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('Error: $e')));
      }
    }
  }

  String _fmt(dynamic raw) {
    if (raw == null) return '—';
    try {
      final dt = raw is DateTime ? raw : DateTime.parse(raw.toString());
      const names = [
        '', 'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
        'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'
      ];
      return '${names[dt.month.clamp(1, 12)]} ${dt.day}, ${dt.year}';
    } catch (_) {
      return '—';
    }
  }
}

// =============================================================================
// Name edit field
// =============================================================================

class _NameField extends StatefulWidget {
  final String initialValue;
  final Future<void> Function(String) onSave;

  const _NameField({required this.initialValue, required this.onSave});

  @override
  State<_NameField> createState() => _NameFieldState();
}

class _NameFieldState extends State<_NameField> {
  late final TextEditingController _ctrl;
  bool _loading = false;

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
    final cs = consoleColors(context);
    return Row(
      children: [
        Expanded(
          child: TextField(
            controller: _ctrl,
            style: TextStyle(color: cs.textPrimary, fontSize: 13),
            decoration: InputDecoration(
              filled: true,
              fillColor: cs.fieldFill,
              isDense: true,
              contentPadding:
                  const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
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
            ),
          ),
        ),
        const SizedBox(width: 12),
        FilledButton(
          style: FilledButton.styleFrom(
            backgroundColor: _accent,
            padding:
                const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8)),
          ),
          onPressed: _loading
              ? null
              : () async {
                  setState(() => _loading = true);
                  await widget.onSave(_ctrl.text.trim());
                  if (mounted) setState(() => _loading = false);
                },
          child: _loading
              ? const SizedBox(
                  width: 14,
                  height: 14,
                  child: CircularProgressIndicator(
                      strokeWidth: 2, color: Colors.white),
                )
              : const Text('Update', style: TextStyle(fontSize: 13)),
        ),
      ],
    );
  }
}

