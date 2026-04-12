import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/app_error_state.dart';
import '../../core/widgets/id_text.dart';
import '../../core/widgets/page_tabs.dart';

const _accent = Color(0xFF3472A4);
const _red = Color(0xFFEF4444);

// ── Type metadata ─────────────────────────────────────────────────────────────

class _PlatformType {
  final String id;
  final String label;
  final IconData icon;

  const _PlatformType(this.id, this.label, this.icon);
}

const _types = [
  _PlatformType('web', 'Web', LucideIcons.globe),
  _PlatformType('ios', 'iOS', LucideIcons.smartphone),
  _PlatformType('android', 'Android', LucideIcons.smartphone),
  _PlatformType('desktop', 'Desktop', LucideIcons.monitor),
  _PlatformType('server', 'Server', LucideIcons.server),
];

// ── Provider ──────────────────────────────────────────────────────────────────

final _platformsProvider =
    FutureProvider.family<List<Map<String, dynamic>>, String>(
        (ref, projectId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/projects/$projectId/platforms');
  final data = res.data as Map<String, dynamic>;
  return List<Map<String, dynamic>>.from(data['platforms'] ?? []);
});

// ── Page ──────────────────────────────────────────────────────────────────────

class PlatformsPage extends ConsumerStatefulWidget {
  const PlatformsPage({super.key});

  @override
  ConsumerState<PlatformsPage> createState() => _PlatformsPageState();
}

class _PlatformsPageState extends ConsumerState<PlatformsPage> {
  final _searchCtrl = TextEditingController();
  int _page = 1;
  int _perPage = 12;

  // Detail state — holds the full row when a platform is selected
  Map<String, dynamic>? _selectedPlatform;
  int _detailTab = 0;

  // Settings tab editing
  final _nameEditCtrl = TextEditingController();
  final _hostnameEditCtrl = TextEditingController();
  bool _settingsSaving = false;

  // Linked deploy target cache
  Future<Map<String, dynamic>?>? _linkedTargetFuture;
  String? _linkedTargetFutureId;

  @override
  void dispose() {
    _searchCtrl.dispose();
    _nameEditCtrl.dispose();
    _hostnameEditCtrl.dispose();
    super.dispose();
  }

  // ── Helpers ───────────────────────────────────────────────────────────────

  String _fmtDate(dynamic v) => v?.toString().split('T').first ?? '—';

  String _typeLabel(String type) => switch (type) {
        'web' => 'Web',
        'ios' => 'iOS',
        'android' => 'Android',
        'desktop' => 'Desktop',
        'server' => 'Server',
        _ => type,
      };

  IconData _typeIconFor(String type) => switch (type) {
        'web' => LucideIcons.globe,
        'ios' || 'android' => LucideIcons.smartphone,
        'desktop' => LucideIcons.monitor,
        'server' => LucideIcons.server,
        _ => LucideIcons.layers,
      };

  IconData _typeIcon(Map<String, dynamic> row) =>
      _typeIconFor(row['type'] as String? ?? '');

  String _identityLabel(String type) => switch (type) {
        'web' => 'Hostname',
        'ios' => 'Bundle ID',
        'android' => 'Package name',
        'desktop' => 'App identifier',
        'server' => 'Hostname / IP',
        _ => 'Identifier',
      };

  String _identityHint(String type) => switch (type) {
        'web' => 'myapp.com',
        'ios' || 'android' || 'desktop' => 'com.example.myapp',
        'server' => '192.168.1.1',
        _ => '',
      };

  String _identityValue(Map<String, dynamic> row) =>
      row['hostname'] as String? ?? '';

  Future<Map<String, dynamic>?> _fetchLinkedTarget(String targetId) {
    if (_linkedTargetFutureId == targetId && _linkedTargetFuture != null) {
      return _linkedTargetFuture!;
    }
    _linkedTargetFutureId = targetId;
    _linkedTargetFuture = ref
        .read(apiClientProvider)
        .get('/deploy/targets/$targetId')
        .then((r) => r.data as Map<String, dynamic>?)
        .catchError((_) => null);
    return _linkedTargetFuture!;
  }

  // ── Build ─────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    if (_selectedPlatform != null) return _detailView(context);

    final colors = consoleColors(context);
    final projectId = ref.watch(currentProjectProvider) ?? '';
    final dataAsync = ref.watch(_platformsProvider(projectId));

    return Scaffold(
      backgroundColor: colors.background,
      body: Padding(
        padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Text('Platforms',
                style: TextStyle(
                    color: colors.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            Text(
                'Register your applications to enable API access and optionally link deployments.',
                style:
                    TextStyle(color: colors.textSecondary, fontSize: 13)),
            const SizedBox(height: 24),
            dataAsync.when(
              loading: () => const Expanded(
                  child: Center(child: CircularProgressIndicator())),
              error: (e, _) => Expanded(
                  child: AppErrorState(
                      error: e,
                      onRetry: () =>
                          ref.invalidate(_platformsProvider(projectId)))),
              data: (platforms) {
                final query = _searchCtrl.text.toLowerCase();
                final filtered = query.isEmpty
                    ? platforms
                    : platforms.where((p) {
                        final name =
                            (p['name'] as String? ?? '').toLowerCase();
                        final type =
                            (p['type'] as String? ?? '').toLowerCase();
                        final identity =
                            _identityValue(p).toLowerCase();
                        return name.contains(query) ||
                            type.contains(query) ||
                            identity.contains(query);
                      }).toList();

                final total = filtered.length;
                final start = (_page - 1) * _perPage;
                final end = (start + _perPage).clamp(0, total);
                final paged = start < total
                    ? filtered.sublist(start, end)
                    : <Map<String, dynamic>>[];

                return Expanded(
                  child: AppDataTable(
                    searchController: _searchCtrl,
                    onSearch: () => setState(() => _page = 1),
                    searchHint: 'Search by name, type, or identifier…',
                    columns: const [
                      AppTableColumn(key: 'name', label: 'Name', flex: 4),
                      AppTableColumn(key: 'type', label: 'Type', flex: 2),
                      AppTableColumn(
                          key: 'identity', label: 'Identifier', flex: 4),
                      AppTableColumn(
                          key: 'created', label: 'Created', flex: 2),
                    ],
                    rows: paged,
                    getCellValue: (row, key) => switch (key) {
                      'name' => row['name'] as String? ?? 'Unnamed',
                      'type' =>
                        _typeLabel(row['type'] as String? ?? ''),
                      'identity' => _identityValue(row),
                      'created' => _fmtDate(
                          row[r'$createdAt'] ?? row['createdAt']),
                      _ => '',
                    },
                    cellBuilder: (row, key) {
                      if (key == 'type') {
                        final type = row['type'] as String? ?? '';
                        return Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 6, vertical: 2),
                          decoration: BoxDecoration(
                            color: _accent.withValues(alpha: 0.12),
                            borderRadius: BorderRadius.circular(4),
                          ),
                          child: Text(
                            _typeLabel(type),
                            style: const TextStyle(
                                color: _accent,
                                fontSize: 11,
                                fontWeight: FontWeight.w500),
                          ),
                        );
                      }
                      if (key == 'identity') {
                        final val = _identityValue(row);
                        final cs = consoleColors(context);
                        if (val.isEmpty) {
                          return Text('—',
                              style: TextStyle(
                                  color: cs.textSubtle, fontSize: 13));
                        }
                        return Text(val,
                            style: TextStyle(
                                color: cs.textSecondary,
                                fontSize: 12,
                                fontFamily: 'monospace'));
                      }
                      return null;
                    },
                    getRowIcon: _typeIcon,
                    onRowTap: (row) {
                      final targetId =
                          row['deployTargetId'] as String? ?? '';
                      setState(() {
                        _selectedPlatform = row;
                        _detailTab = 0;
                        _nameEditCtrl.text =
                            row['name'] as String? ?? '';
                        _hostnameEditCtrl.text = _identityValue(row);
                        _settingsSaving = false;
                        _linkedTargetFutureId = null;
                        _linkedTargetFuture = targetId.isNotEmpty
                            ? _fetchLinkedTarget(targetId)
                            : null;
                      });
                    },
                    onDeleteRow: (row) async {
                      final id = row[r'$id'] as String? ??
                          row['id'] as String? ??
                          '';
                      await ref
                          .read(apiClientProvider)
                          .delete('/projects/$projectId/platforms/$id');
                      ref.invalidate(_platformsProvider(projectId));
                    },
                    filters: [
                      AppTableFilter(
                        key: 'type',
                        label: 'Type',
                        options: _types.map((t) => t.id).toList(),
                      ),
                    ],
                    onFiltersChanged: (_) {},
                    gridCardBuilder: (row) => _GridCard(
                      row: row,
                      typeIcon: _typeIcon(row),
                      typeLabel:
                          _typeLabel(row['type'] as String? ?? ''),
                      identityLabel: _identityLabel(
                          row['type'] as String? ?? ''),
                      identityValue: _identityValue(row),
                    ),
                    persistKey: 'platforms_view',
                    total: total,
                    perPage: _perPage,
                    currentPage: _page,
                    onPrev: () => setState(() => _page--),
                    onNext: () => setState(() => _page++),
                    onPerPageChanged: (pp) =>
                        setState(() {
                          _perPage = pp;
                          _page = 1;
                        }),
                    itemLabel: 'platforms',
                    emptyIcon: LucideIcons.layers,
                    emptyTitle: 'No platforms registered',
                    emptySubtitle:
                        'Register your web, mobile, desktop, or server platforms to enable API access.',
                    createLabel: 'Add platform',
                    onCreateTap: () =>
                        _showAddDialog(context, projectId),
                  ),
                );
              },
            ),
          ],
        ),
      ),
    );
  }

  // ── Detail view ──────────────────────────────────────────────────────────────

  Widget _detailView(BuildContext context) {
    final t = _selectedPlatform!;
    final cs = consoleColors(context);
    final type = t['type'] as String? ?? 'web';
    const tabs = ['Overview', 'Settings'];

    return Scaffold(
      backgroundColor: cs.background,
      body: Column(children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(32, 20, 32, 16),
          child: Row(children: [
            IconButton(
              icon: Icon(LucideIcons.arrowLeft,
                  size: 18, color: cs.textSecondary),
              onPressed: () => setState(() {
                _selectedPlatform = null;
                _detailTab = 0;
                _linkedTargetFuture = null;
                _linkedTargetFutureId = null;
              }),
            ),
            const SizedBox(width: 8),
            Icon(_typeIconFor(type), size: 16, color: cs.textSubtle),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    t['name'] as String? ?? '',
                    style: TextStyle(
                        color: cs.textPrimary,
                        fontSize: 20,
                        fontWeight: FontWeight.w600),
                  ),
                  if (_identityValue(t).isNotEmpty)
                    Text(
                      _identityValue(t),
                      style: TextStyle(
                          color: cs.textSecondary,
                          fontSize: 12,
                          fontFamily: 'monospace'),
                    ),
                ],
              ),
            ),
          ]),
        ),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 32),
          child: PageTabs(
            tabs: tabs,
            selected: _detailTab,
            onChanged: (i) => setState(() => _detailTab = i),
          ),
        ),
        const SizedBox(height: 16),
        Expanded(
            child: _buildDetailTab(context, t, type, tabs)),
      ]),
    );
  }

  Widget _buildDetailTab(BuildContext ctx, Map<String, dynamic> t,
      String type, List<String> tabs) {
    final tabName = tabs[_detailTab.clamp(0, tabs.length - 1)];
    return switch (tabName) {
      'Overview' => _overviewTab(ctx, t, type),
      'Settings' => _settingsTab(ctx, t, type),
      _ => const SizedBox.shrink(),
    };
  }

  // ── Overview tab ─────────────────────────────────────────────────────────────

  Widget _overviewTab(
      BuildContext ctx, Map<String, dynamic> t, String type) {
    final cs = consoleColors(ctx);
    final id = t[r'$id'] as String? ?? t['id'] as String? ?? '';
    final identity = _identityValue(t);
    final created = _fmtDate(t[r'$createdAt'] ?? t['createdAt']);
    final projectId = ref.watch(currentProjectProvider) ?? '';

    return SingleChildScrollView(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            _infoCard(ctx, 'Type', _typeLabel(type)),
            const SizedBox(width: 16),
            _infoCard(
                ctx,
                _identityLabel(type),
                identity.isEmpty ? '—' : identity),
            const SizedBox(width: 16),
            _infoCard(ctx, 'Registered', created),
          ]),
          const SizedBox(height: 16),
          // Platform ID — for SDK initialisation
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: cs.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: cs.border),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Platform ID',
                    style: TextStyle(
                        color: cs.textSubtle, fontSize: 12)),
                const SizedBox(height: 8),
                IdText(id: id, fontSize: 13),
                const SizedBox(height: 6),
                Text(
                  'Use this ID when initialising the SDK to identify and restrict API access to this platform.',
                  style: TextStyle(color: cs.textSubtle, fontSize: 11),
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),
          // Connected deployment section
          _deploymentSection(ctx, t, projectId),
        ],
      ),
    );
  }

  Widget _deploymentSection(
      BuildContext ctx, Map<String, dynamic> t, String projectId) {
    final cs = consoleColors(ctx);
    final platformId =
        t[r'$id'] as String? ?? t['id'] as String? ?? '';
    final linkedTargetId = t['deployTargetId'] as String? ?? '';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(children: [
          Text('Deployment',
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w600)),
          const Spacer(),
          if (linkedTargetId.isNotEmpty)
            TextButton.icon(
              style: TextButton.styleFrom(
                  foregroundColor: cs.textSecondary,
                  textStyle: const TextStyle(fontSize: 12)),
              icon: const Icon(LucideIcons.externalLink, size: 12),
              label: const Text('View in Deploy'),
              onPressed: () =>
                  ctx.go('/project/$projectId/deploy'),
            ),
        ]),
        const SizedBox(height: 10),
        if (linkedTargetId.isEmpty)
          Container(
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
                Row(children: [
                  Icon(LucideIcons.rocket,
                      size: 14, color: cs.textSubtle),
                  const SizedBox(width: 8),
                  Text('No deployment connected',
                      style: TextStyle(
                          color: cs.textPrimary,
                          fontSize: 13,
                          fontWeight: FontWeight.w500)),
                ]),
                const SizedBox(height: 6),
                Text(
                  'Connect a deploy target to enable automatic builds and track deployment status directly from this platform.',
                  style: TextStyle(
                      color: cs.textSecondary, fontSize: 12),
                ),
                const SizedBox(height: 14),
                OutlinedButton.icon(
                  style: OutlinedButton.styleFrom(
                    foregroundColor: _accent,
                    side: const BorderSide(color: _accent),
                    padding: const EdgeInsets.symmetric(
                        horizontal: 14, vertical: 8),
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(6)),
                  ),
                  icon: const Icon(LucideIcons.link2, size: 13),
                  label: const Text('Connect deployment',
                      style: TextStyle(fontSize: 12)),
                  onPressed: () => _showConnectDeploymentDialog(
                      ctx, projectId, platformId),
                ),
              ],
            ),
          )
        else
          FutureBuilder<Map<String, dynamic>?>(
            future: _fetchLinkedTarget(linkedTargetId),
            builder: (ctx, snap) {
              if (!snap.hasData) {
                return Container(
                  height: 80,
                  decoration: BoxDecoration(
                    color: cs.surface,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: cs.border),
                  ),
                  child: const Center(
                      child: CircularProgressIndicator()),
                );
              }
              final target = snap.data;
              if (target == null) {
                return _deploymentErrorCard(ctx, projectId,
                    platformId, linkedTargetId);
              }
              final tName =
                  target['name'] as String? ?? linkedTargetId;
              final tType = target['type'] as String? ?? '';
              return Container(
                width: double.infinity,
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: cs.border),
                ),
                child: Row(children: [
                  Container(
                    width: 36,
                    height: 36,
                    decoration: BoxDecoration(
                      color: _accent.withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: const Icon(LucideIcons.rocket,
                        size: 16, color: _accent),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment:
                          CrossAxisAlignment.start,
                      children: [
                        Text(tName,
                            style: TextStyle(
                                color: cs.textPrimary,
                                fontSize: 13,
                                fontWeight: FontWeight.w500)),
                        Text(tType,
                            style: TextStyle(
                                color: cs.textSecondary,
                                fontSize: 11)),
                      ],
                    ),
                  ),
                  IconButton(
                    tooltip: 'Disconnect',
                    icon: Icon(LucideIcons.unlink,
                        size: 14, color: cs.textSubtle),
                    onPressed: () => _disconnectDeployment(
                        ctx, projectId, platformId),
                  ),
                ]),
              );
            },
          ),
      ],
    );
  }

  Widget _deploymentErrorCard(BuildContext ctx, String projectId,
      String platformId, String targetId) {
    final cs = consoleColors(ctx);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Row(children: [
        Icon(LucideIcons.alertTriangle,
            size: 14, color: cs.textSubtle),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            'Deployment target not found. It may have been deleted.',
            style:
                TextStyle(color: cs.textSecondary, fontSize: 12),
          ),
        ),
        TextButton(
          style: TextButton.styleFrom(
              foregroundColor: _red,
              textStyle: const TextStyle(fontSize: 12)),
          onPressed: () => _disconnectDeployment(
              ctx, projectId, platformId),
          child: const Text('Remove'),
        ),
      ]),
    );
  }

  Future<void> _disconnectDeployment(BuildContext ctx,
      String projectId, String platformId) async {
    final api = ref.read(apiClientProvider);
    await api.patch('/projects/$projectId/platforms/$platformId',
        data: {'deployTargetId': ''});
    if (!mounted) return;
    setState(() {
      _selectedPlatform = {
        ..._selectedPlatform!,
        'deployTargetId': '',
      };
      _linkedTargetFuture = null;
      _linkedTargetFutureId = null;
    });
    ref.invalidate(_platformsProvider(projectId));
  }

  void _showConnectDeploymentDialog(
      BuildContext context, String projectId, String platformId) {
    showAppDialog(
      context: context,
      title: 'Connect deployment',
      subtitle: 'Link a deploy target to enable automatic builds',
      content: _ConnectDeploymentContent(
        projectId: projectId,
        onSelect: (targetId) async {
          final nav =
              Navigator.of(context, rootNavigator: true);
          final api = ref.read(apiClientProvider);
          await api.patch(
              '/projects/$projectId/platforms/$platformId',
              data: {'deployTargetId': targetId});
          final future = _fetchLinkedTarget(targetId);
          if (!mounted) return;
          setState(() {
            _selectedPlatform = {
              ..._selectedPlatform!,
              'deployTargetId': targetId,
            };
            _linkedTargetFuture = future;
          });
          ref.invalidate(_platformsProvider(projectId));
          nav.pop();
        },
      ),
      actions: const [AppDialogCancel()],
    );
  }

  Widget _infoCard(BuildContext ctx, String label, String value) {
    final cs = consoleColors(ctx);
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: cs.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: cs.border),
        ),
        child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(label,
                  style:
                      TextStyle(color: cs.textSubtle, fontSize: 12)),
              const SizedBox(height: 6),
              Text(value,
                  style: TextStyle(
                      color: cs.textPrimary,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
            ]),
      ),
    );
  }

  // ── Settings tab ─────────────────────────────────────────────────────────────

  Widget _settingsTab(
      BuildContext ctx, Map<String, dynamic> t, String type) {
    final cs = consoleColors(ctx);
    final projectId = ref.watch(currentProjectProvider) ?? '';
    final id =
        t[r'$id'] as String? ?? t['id'] as String? ?? '';

    return SingleChildScrollView(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Settings',
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 24),
          AppDialogField(
            controller: _nameEditCtrl,
            label: 'Name',
            hint: 'My app',
          ),
          const SizedBox(height: 16),
          AppDialogField(
            controller: _hostnameEditCtrl,
            label: _identityLabel(type),
            hint: _identityHint(type),
          ),
          const SizedBox(height: 24),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              padding: const EdgeInsets.symmetric(
                  horizontal: 20, vertical: 12),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            onPressed: _settingsSaving
                ? null
                : () async {
                    setState(() => _settingsSaving = true);
                    final api = ref.read(apiClientProvider);
                    try {
                      await api.patch(
                          '/projects/$projectId/platforms/$id',
                          data: {
                            'name': _nameEditCtrl.text.trim(),
                            'hostname': _hostnameEditCtrl.text.trim(),
                          });
                      if (!mounted) return;
                      setState(() {
                        _selectedPlatform = {
                          ..._selectedPlatform!,
                          'name': _nameEditCtrl.text.trim(),
                          'hostname': _hostnameEditCtrl.text.trim(),
                        };
                        _settingsSaving = false;
                      });
                      ref.invalidate(_platformsProvider(projectId));
                    } catch (_) {
                      if (mounted) setState(() => _settingsSaving = false);
                    }
                  },
            child: _settingsSaving
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: Colors.white))
                : const Text('Save changes',
                    style: TextStyle(fontSize: 13)),
          ),
          const SizedBox(height: 48),
          // Danger zone
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: cs.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(
                  color: _red.withValues(alpha: 0.3)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('Danger zone',
                    style: TextStyle(
                        color: _red,
                        fontSize: 14,
                        fontWeight: FontWeight.w600)),
                const SizedBox(height: 8),
                Text(
                  'Remove this platform. API access from this platform will be revoked.',
                  style: TextStyle(
                      color: cs.textSubtle, fontSize: 13),
                ),
                const SizedBox(height: 12),
                OutlinedButton(
                  style: OutlinedButton.styleFrom(
                      foregroundColor: _red,
                      side: const BorderSide(color: _red)),
                  onPressed: () async {
                    final confirmed =
                        await showAppDialog<bool>(
                      context: context,
                      title: 'Remove platform',
                      content: Text(
                        'This platform will be removed and API access revoked. This cannot be undone.',
                        style: TextStyle(
                            color: cs.textSecondary),
                      ),
                      actions: [
                        const AppDialogCancel(),
                        AppDialogAction(
                          label: 'Remove',
                          destructive: true,
                          onTap: () => Navigator.of(
                            context,
                            rootNavigator: true,
                          ).pop(true),
                        ),
                      ],
                    );
                    if (confirmed != true) return;
                    await ref
                        .read(apiClientProvider)
                        .delete(
                            '/projects/$projectId/platforms/$id');
                    ref.invalidate(
                        _platformsProvider(projectId));
                    if (mounted) {
                      setState(() {
                        _selectedPlatform = null;
                        _detailTab = 0;
                      });
                    }
                  },
                  child: const Text('Remove platform'),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  // ── Add platform dialog ───────────────────────────────────────────────────────

  void _showAddDialog(BuildContext context, String projectId) {
    final colors = consoleColors(context);
    final nameCtrl = TextEditingController();
    final hostnameCtrl = TextEditingController();
    String selectedType = 'web';

    showAppDialog(
      context: context,
      title: 'Add platform',
      subtitle: 'Register an application platform',
      content: StatefulBuilder(
        builder: (ctx, setDialogState) => Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Type',
                    style: TextStyle(
                        color: colors.textSecondary,
                        fontSize: 12,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: _types.map((t) {
                    final sel = selectedType == t.id;
                    return GestureDetector(
                      onTap: () => setDialogState(() {
                        selectedType = t.id;
                        hostnameCtrl.clear();
                      }),
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 8),
                        decoration: BoxDecoration(
                          color: sel
                              ? _accent.withValues(alpha: 0.15)
                              : colors.fieldFill,
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(
                              color: sel
                                  ? _accent
                                  : colors.fieldBorder),
                        ),
                        child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(t.icon,
                                  size: 14,
                                  color: sel
                                      ? _accent
                                      : colors.textSecondary),
                              const SizedBox(width: 6),
                              Text(t.label,
                                  style: TextStyle(
                                      color: sel
                                          ? colors.textPrimary
                                          : colors.textSecondary,
                                      fontSize: 12)),
                            ]),
                      ),
                    );
                  }).toList(),
                ),
              ],
            ),
            const SizedBox(height: 16),
            AppDialogField(
                controller: nameCtrl,
                label: 'Name',
                hint: 'My app',
                autofocus: true),
            const SizedBox(height: 16),
            AppDialogField(
              controller: hostnameCtrl,
              label: _identityLabel(selectedType),
              hint: _identityHint(selectedType),
            ),
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (nameCtrl.text.trim().isEmpty) return;
            final nav =
                Navigator.of(context, rootNavigator: true);
            final api = ref.read(apiClientProvider);
            await api.post('/projects/$projectId/platforms',
                data: {
                  'type': selectedType,
                  'name': nameCtrl.text.trim(),
                  'hostname': hostnameCtrl.text.trim(),
                });
            ref.invalidate(_platformsProvider(projectId));
            nav.pop();
          },
        ),
      ],
    );
  }
}

// ── Connect deployment dialog content ─────────────────────────────────────────

class _ConnectDeploymentContent extends ConsumerWidget {
  final String projectId;
  final Future<void> Function(String targetId) onSelect;

  const _ConnectDeploymentContent({
    required this.projectId,
    required this.onSelect,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cs = consoleColors(context);

    return FutureBuilder<dynamic>(
      future: ref
          .read(apiClientProvider)
          .get('/deploy/targets')
          .then((r) => r.data),
      builder: (ctx, snap) {
        if (!snap.hasData) {
          return const SizedBox(
            height: 80,
            child: Center(child: CircularProgressIndicator()),
          );
        }
        final targets = List<Map<String, dynamic>>.from(
            (snap.data as Map?)?['targets'] as List? ?? []);
        if (targets.isEmpty) {
          return Padding(
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Column(
              children: [
                Icon(LucideIcons.rocket,
                    size: 32, color: cs.textSubtle),
                const SizedBox(height: 12),
                Text('No deploy targets found',
                    style: TextStyle(
                        color: cs.textPrimary,
                        fontSize: 13,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 4),
                Text(
                  'Create a deploy target in the Deploy section first.',
                  style: TextStyle(
                      color: cs.textSecondary, fontSize: 12),
                  textAlign: TextAlign.center,
                ),
              ],
            ),
          );
        }
        return Column(
          mainAxisSize: MainAxisSize.min,
          children: targets.map((t) {
            final id =
                t[r'$id'] as String? ?? t['id'] as String? ?? '';
            final name = t['name'] as String? ?? id;
            final type = t['type'] as String? ?? '';
            return InkWell(
              borderRadius: BorderRadius.circular(8),
              onTap: () => onSelect(id),
              child: Container(
                margin: const EdgeInsets.only(bottom: 8),
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: cs.surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: cs.border),
                ),
                child: Row(children: [
                  Container(
                    width: 32,
                    height: 32,
                    decoration: BoxDecoration(
                      color: const Color(0xFF3472A4)
                          .withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: const Icon(LucideIcons.rocket,
                        size: 14, color: Color(0xFF3472A4)),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment:
                          CrossAxisAlignment.start,
                      children: [
                        Text(name,
                            style: TextStyle(
                                color: cs.textPrimary,
                                fontSize: 13,
                                fontWeight: FontWeight.w500)),
                        if (type.isNotEmpty)
                          Text(type,
                              style: TextStyle(
                                  color: cs.textSecondary,
                                  fontSize: 11)),
                      ],
                    ),
                  ),
                  Icon(LucideIcons.chevronRight,
                      size: 14, color: cs.textSubtle),
                ]),
              ),
            );
          }).toList(),
        );
      },
    );
  }
}

// ── Grid card ─────────────────────────────────────────────────────────────────

class _GridCard extends StatelessWidget {
  final Map<String, dynamic> row;
  final IconData typeIcon;
  final String typeLabel;
  final String identityLabel;
  final String identityValue;

  const _GridCard({
    required this.row,
    required this.typeIcon,
    required this.typeLabel,
    required this.identityLabel,
    required this.identityValue,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final name = row['name'] as String? ?? 'Unnamed';

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            Container(
              width: 36,
              height: 36,
              decoration: BoxDecoration(
                color: _accent.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Icon(typeIcon, size: 18, color: _accent),
            ),
            const Spacer(),
            Container(
              padding: const EdgeInsets.symmetric(
                  horizontal: 6, vertical: 2),
              decoration: BoxDecoration(
                color: _accent.withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(4),
              ),
              child: Text(typeLabel,
                  style: const TextStyle(
                      color: _accent,
                      fontSize: 11,
                      fontWeight: FontWeight.w500)),
            ),
          ]),
          const SizedBox(height: 12),
          Text(name,
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 14,
                  fontWeight: FontWeight.w500),
              maxLines: 1,
              overflow: TextOverflow.ellipsis),
          const SizedBox(height: 4),
          identityValue.isNotEmpty
              ? Text(identityValue,
                  style: TextStyle(
                      color: cs.textSecondary,
                      fontSize: 11,
                      fontFamily: 'monospace'),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis)
              : Text('No identifier set',
                  style:
                      TextStyle(color: cs.textSubtle, fontSize: 11)),
          const Spacer(),
          Text(identityLabel,
              style:
                  TextStyle(color: cs.textSubtle, fontSize: 11)),
        ],
      ),
    );
  }
}
