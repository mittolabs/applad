import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
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
import '../../core/widgets/status_chip.dart';

const _accent = Color(0xFF3472A4);
const _green = Color(0xFF10B981);
const _red = Color(0xFFEF4444);
const _orange = Color(0xFFF59E0B);

// ── Type metadata ─────────────────────────────────────────────────────────────

class _PlatformType {
  final String id;
  final String label;
  final IconData icon;

  const _PlatformType(this.id, this.label, this.icon);
}

const _types = [
  _PlatformType('web', 'Web', LucideIcons.globe),
  _PlatformType('mobile', 'Mobile', LucideIcons.smartphone),
  _PlatformType('desktop', 'Desktop', LucideIcons.monitor),
  _PlatformType('container', 'Container', LucideIcons.box),
];

// ── Provider ──────────────────────────────────────────────────────────────────

final _platformsProvider =
    FutureProvider.family<List<Map<String, dynamic>>, String>(
        (ref, projectId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy/targets');
  final data = res.data as Map<String, dynamic>;
  return List<Map<String, dynamic>>.from(data['targets'] ?? []);
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

  // Detail state
  String? _selectedId;
  String? _selectedType;
  int _detailTab = 0;

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  String _fmtDate(dynamic v) => v?.toString().split('T').first ?? '—';

  String _typeLabel(String type) => switch (type) {
        'web' => 'Web',
        'mobile' => 'Mobile',
        'desktop' => 'Desktop',
        'container' => 'Container',
        _ => type,
      };

  IconData _typeIcon(Map<String, dynamic> row) {
    final type = row['type'] as String? ?? '';
    return switch (type) {
      'web' => LucideIcons.globe,
      'mobile' => LucideIcons.smartphone,
      'desktop' => LucideIcons.monitor,
      'container' => LucideIcons.box,
      _ => LucideIcons.layers,
    };
  }

  List<String> _tabsForType(String type) => switch (type) {
        'web' => ['Overview', 'Builds', 'Domains', 'Settings'],
        'mobile' => ['Overview', 'Builds', 'Signing', 'Settings'],
        'desktop' => ['Overview', 'Builds', 'Signing', 'Distribution', 'Settings'],
        'container' => ['Overview', 'Logs', 'Settings'],
        _ => ['Overview', 'Settings'],
      };

  @override
  Widget build(BuildContext context) {
    if (_selectedId != null) return _detailView(context);

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
                'Manage the platforms where your application is deployed.',
                style: TextStyle(color: colors.textSecondary, fontSize: 13)),
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
                        return name.contains(query) || type.contains(query);
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
                    searchHint: 'Search by name or type…',
                    columns: const [
                      AppTableColumn(key: 'name', label: 'Name', flex: 4),
                      AppTableColumn(key: 'type', label: 'Type', flex: 2),
                      AppTableColumn(key: 'status', label: 'Status', flex: 2),
                      AppTableColumn(key: 'updated', label: 'Updated', flex: 2),
                    ],
                    rows: paged,
                    getCellValue: (row, key) => switch (key) {
                      'name' => row['name'] as String? ?? 'Unnamed',
                      'type' => _typeLabel(row['type'] as String? ?? ''),
                      'status' => row['status'] as String? ?? 'unknown',
                      'updated' =>
                        _fmtDate(row[r'$updatedAt'] ?? row['updatedAt']),
                      _ => '',
                    },
                    cellBuilder: (row, key) {
                      if (key == 'status') {
                        final s = row['status'] as String? ?? 'unknown';
                        return StatusChip.fromStatus(s);
                      }
                      return null;
                    },
                    getRowIcon: _typeIcon,
                    onRowTap: (row) {
                      final id = row[r'$id'] as String? ??
                          row['id'] as String? ?? '';
                      final type = row['type'] as String? ?? '';
                      setState(() {
                        _selectedId = id;
                        _selectedType = type;
                        _detailTab = 0;
                      });
                    },
                    onDeleteRow: (row) async {
                      final id = row[r'$id'] as String? ??
                          row['id'] as String? ?? '';
                      await ref
                          .read(apiClientProvider)
                          .delete('/deploy/targets/$id');
                      ref.invalidate(_platformsProvider(projectId));
                    },
                    filters: const [
                      AppTableFilter(
                        key: 'type',
                        label: 'Type',
                        options: ['web', 'mobile', 'desktop', 'container'],
                      ),
                      AppTableFilter(
                        key: 'status',
                        label: 'Status',
                        options: ['active', 'inactive', 'failed', 'unknown'],
                      ),
                    ],
                    onFiltersChanged: (active) {
                      // Client-side filtering — search already handles it
                    },
                    gridCardBuilder: (row) => _GridCard(
                      row: row,
                      typeIcon: _typeIcon(row),
                      typeLabel: _typeLabel(row['type'] as String? ?? ''),
                      fmtDate: _fmtDate,
                    ),
                    persistKey: 'platforms_view',
                    total: total,
                    perPage: _perPage,
                    currentPage: _page,
                    onPrev: () => setState(() => _page--),
                    onNext: () => setState(() => _page++),
                    onPerPageChanged: (pp) =>
                        setState(() { _perPage = pp; _page = 1; }),
                    itemLabel: 'platforms',
                    emptyIcon: LucideIcons.plus,
                    emptyTitle: 'No platforms yet',
                    emptySubtitle:
                        'Add a platform to start deploying your app.',
                    createLabel: 'Add platform',
                    onCreateTap: () => _showAddDialog(context, projectId),
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
    return FutureBuilder<dynamic>(
      future: ref
          .read(apiClientProvider)
          .get('/deploy/targets/$_selectedId')
          .then((r) => r.data),
      builder: (ctx, snap) {
        final colors = consoleColors(ctx);
        if (!snap.hasData) {
          return Scaffold(
            backgroundColor: colors.background,
            body: const Center(child: CircularProgressIndicator()),
          );
        }
        final target = snap.data as Map<String, dynamic>;
        final type = _selectedType ?? (target['type'] as String? ?? 'web');
        final tabs = _tabsForType(type);

        return Scaffold(
          backgroundColor: colors.background,
          body: Column(children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(32, 20, 32, 16),
              child: Row(children: [
                IconButton(
                  icon: Icon(LucideIcons.arrowLeft,
                      size: 18, color: colors.textSecondary),
                  onPressed: () => setState(() {
                    _selectedId = null;
                    _selectedType = null;
                    _detailTab = 0;
                  }),
                ),
                const SizedBox(width: 8),
                Icon(_typeIcon({'type': type}),
                    size: 16, color: colors.textSubtle),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    target['name'] as String? ?? '',
                    style: TextStyle(
                        color: colors.textPrimary,
                        fontSize: 20,
                        fontWeight: FontWeight.w600),
                  ),
                ),
                StatusChip.fromStatus(
                    target['status'] as String? ?? 'unknown'),
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
            Expanded(child: _buildDetailTab(ctx, target, type, tabs)),
          ]),
        );
      },
    );
  }

  Widget _buildDetailTab(
      BuildContext ctx, Map<String, dynamic> target, String type, List<String> tabs) {
    final tabName = tabs[_detailTab.clamp(0, tabs.length - 1)];
    return switch (tabName) {
      'Overview' => _overviewTab(ctx, target, type),
      'Builds' => _buildsTab(ctx),
      'Domains' => _domainsTab(ctx),
      'Signing' => _signingTab(ctx, target, type),
      'Distribution' => _distributionTab(ctx, target),
      'Logs' => _logsTab(ctx),
      'Settings' => _settingsTab(ctx, target, type),
      _ => const SizedBox.shrink(),
    };
  }

  // ── Overview tab ─────────────────────────────────────────────────────────────

  Widget _overviewTab(
      BuildContext ctx, Map<String, dynamic> t, String type) {
    final cs = consoleColors(ctx);
    final rows = <List<Widget>>[];

    switch (type) {
      case 'web':
        rows.add([
          _infoCard(ctx, 'Framework', t['framework'] ?? '—'),
          _infoCard(ctx, 'Source', t['source'] ?? 'manual'),
          _infoCard(ctx, 'Branch', t['branch'] ?? '—'),
        ]);
        rows.add([
          _infoCard(ctx, 'Build command', t['buildCommand'] ?? '—'),
          _infoCard(ctx, 'Output dir', t['outputDir'] ?? '—'),
          _infoCard(ctx, 'Repository', t['repository'] ?? '—'),
        ]);
      case 'mobile':
        rows.add([
          _infoCard(ctx, 'Platform',
              t['buildType'] == 'ipa' ? 'iOS' : 'Android'),
          _infoCard(ctx, 'Build type', t['buildType'] ?? 'apk'),
          _infoCard(ctx, 'Runtime', t['runtime'] ?? '—'),
        ]);
      case 'desktop':
        rows.add([
          _infoCard(ctx, 'Platform', _desktopPlatformLabel(t['platform'] as String? ?? '')),
          _infoCard(ctx, 'Framework', t['framework'] ?? '—'),
          _infoCard(ctx, 'Build type', t['buildType'] ?? 'release'),
        ]);
        rows.add([
          _infoCard(ctx, 'Source', t['source'] ?? 'manual'),
          _infoCard(ctx, 'Repository', t['repository'] ?? '—'),
          _infoCard(ctx, 'Branch', t['branch'] ?? 'main'),
        ]);
      case 'container':
        rows.add([
          _infoCard(ctx, 'Image', t['image'] ?? '—'),
          _infoCard(ctx, 'Registry', t['registry'] ?? 'docker.io'),
          _infoCard(ctx, 'Tag', t['tag'] ?? 'latest'),
        ]);
        rows.add([
          _infoCard(ctx, 'Port', t['port']?.toString() ?? '—'),
          _infoCard(ctx, 'Restart policy', t['restartPolicy'] ?? 'always'),
          _infoCard(ctx, 'Memory limit', t['memoryLimit'] ?? '—'),
        ]);
    }

    return SingleChildScrollView(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final row in rows) ...[
            Row(children: [
              for (int i = 0; i < row.length; i++) ...[
                row[i],
                if (i < row.length - 1) const SizedBox(width: 16),
              ],
            ]),
            const SizedBox(height: 16),
          ],
          if (type == 'web') ...[
            Text('Domains',
                style: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 16,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: cs.surface,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: cs.border),
              ),
              child: Text(
                'Open the Domains tab to manage custom domains and SSL.',
                style: TextStyle(color: cs.textSecondary, fontSize: 13),
              ),
            ),
          ],
        ],
      ),
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
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(label,
              style: TextStyle(color: cs.textSubtle, fontSize: 12)),
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

  // ── Builds tab ────────────────────────────────────────────────────────────────

  Widget _buildsTab(BuildContext ctx) {
    return FutureBuilder<dynamic>(
      future: ref
          .read(apiClientProvider)
          .get('/deploy/releases', params: {'targetId': _selectedId})
          .then((r) => r.data),
      builder: (ctx, snap) {
        final cs = consoleColors(ctx);
        if (!snap.hasData) {
          return const Center(child: CircularProgressIndicator());
        }
        final releases = List<Map<String, dynamic>>.from(
            (snap.data as Map)['releases'] ?? []);
        if (releases.isEmpty) {
          return Center(
            child: Text('No builds yet',
                style: TextStyle(color: cs.textSecondary, fontSize: 14)),
          );
        }
        return ListView.separated(
          padding: const EdgeInsets.all(32),
          itemCount: releases.length,
          separatorBuilder: (_, __) => const SizedBox(height: 8),
          itemBuilder: (_, i) {
            final r = releases[i];
            final status = r['status'] as String? ?? 'pending';
            final sc = status == 'completed'
                ? _green
                : status == 'failed'
                    ? _red
                    : _orange;
            return Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(
                color: cs.surface,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: cs.border),
              ),
              child: Row(children: [
                Container(
                    width: 8,
                    height: 8,
                    decoration:
                        BoxDecoration(color: sc, shape: BoxShape.circle)),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        IdText(id: r[r'$id'] ?? '', fontSize: 12),
                        Text(
                            '${r['triggerType'] ?? 'manual'} · ${r['durationMs'] ?? 0}ms',
                            style: TextStyle(
                                color: cs.textSubtle, fontSize: 11)),
                      ]),
                ),
                Text(status,
                    style: TextStyle(color: sc, fontSize: 12)),
              ]),
            );
          },
        );
      },
    );
  }

  // ── Domains tab (web only) ────────────────────────────────────────────────────

  Widget _domainsTab(BuildContext ctx) {
    return FutureBuilder<dynamic>(
      future: ref
          .read(apiClientProvider)
          .get('/deploy/targets/$_selectedId/domains')
          .then((r) => r.data),
      builder: (ctx, snap) {
        final cs = consoleColors(ctx);
        if (!snap.hasData) {
          return const Center(child: CircularProgressIndicator());
        }
        final domains = List<Map<String, dynamic>>.from(
            (snap.data as Map)['domains'] ?? []);
        return Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(children: [
                Text('Custom domains',
                    style: TextStyle(
                        color: cs.textPrimary,
                        fontSize: 16,
                        fontWeight: FontWeight.w600)),
                const Spacer(),
                OutlinedButton.icon(
                  style: OutlinedButton.styleFrom(
                    foregroundColor: cs.textSecondary,
                    side: BorderSide(color: cs.border),
                  ),
                  icon: const Icon(LucideIcons.plus, size: 14),
                  label:
                      const Text('Add domain', style: TextStyle(fontSize: 12)),
                  onPressed: () {},
                ),
              ]),
              const SizedBox(height: 16),
              if (domains.isEmpty)
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(20),
                  decoration: BoxDecoration(
                    color: cs.surface,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: cs.border),
                  ),
                  child: Text(
                    'No custom domains configured. Your site is accessible via the default subdomain.',
                    style: TextStyle(color: cs.textSecondary, fontSize: 13),
                  ),
                )
              else
                ...domains.map((d) {
                  final verified = d['verified'] == true;
                  return Container(
                    margin: const EdgeInsets.only(bottom: 8),
                    padding: const EdgeInsets.all(14),
                    decoration: BoxDecoration(
                      color: cs.surface,
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: cs.border),
                    ),
                    child: Row(children: [
                      Icon(LucideIcons.globe,
                          size: 14, color: cs.textSubtle),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Text(d['domain'] as String? ?? '',
                            style: TextStyle(
                                color: cs.textPrimary, fontSize: 13)),
                      ),
                      StatusChip.fromStatus(verified ? 'verified' : 'pending'),
                    ]),
                  );
                }),
            ],
          ),
        );
      },
    );
  }

  // ── Signing tab (mobile + desktop) ────────────────────────────────────────────

  Widget _signingTab(
      BuildContext ctx, Map<String, dynamic> t, String type) {
    final cs = consoleColors(ctx);
    final isMobile = type == 'mobile';
    final platform = t['platform'] as String? ?? 'cross-platform';

    return SingleChildScrollView(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Code Signing',
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 24),
          if (isMobile) ...[
            _signingSection(ctx,
                title: 'Android Keystore',
                description:
                    'Upload your keystore file for signed APK/AAB builds',
                buttonLabel: 'Upload keystore'),
            const SizedBox(height: 16),
            _signingSection(ctx,
                title: 'iOS Provisioning',
                description:
                    'Upload provisioning profile + P12 certificate for IPA builds',
                buttonLabel: 'Upload profile'),
          ] else ...[
            if (platform == 'macos' || platform == 'cross-platform') ...[
              _signingSection(ctx,
                  title: 'macOS Certificate',
                  description:
                      'Upload your Developer ID certificate (.p12) for macOS code signing',
                  buttonLabel: 'Upload .p12 certificate'),
              const SizedBox(height: 16),
              _configSection(ctx,
                  title: 'Apple Developer Team ID',
                  description:
                      'Required for notarization and distribution outside the Mac App Store',
                  fields: [
                    _configField(ctx, 'Team ID', t['appleTeamId'] ?? ''),
                  ]),
              const SizedBox(height: 16),
            ],
            if (platform == 'windows' || platform == 'cross-platform') ...[
              _signingSection(ctx,
                  title: 'Windows Code Signing Certificate',
                  description:
                      'Upload your code signing certificate (.pfx) for Windows executables',
                  buttonLabel: 'Upload .pfx certificate'),
              const SizedBox(height: 16),
            ],
            if (platform == 'linux' || platform == 'cross-platform') ...[
              _configSection(ctx,
                  title: 'Linux GPG Signing',
                  description:
                      'Configure GPG key for signing Linux packages',
                  fields: [
                    _configField(ctx, 'GPG Key ID', t['gpgKeyId'] ?? ''),
                  ]),
            ],
          ],
        ],
      ),
    );
  }

  Widget _signingSection(BuildContext ctx,
      {required String title,
      required String description,
      required String buttonLabel}) {
    final cs = consoleColors(ctx);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title,
            style: TextStyle(color: cs.textPrimary, fontSize: 14)),
        const SizedBox(height: 4),
        Text(description,
            style: TextStyle(color: cs.textSubtle, fontSize: 13)),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          style: OutlinedButton.styleFrom(
              foregroundColor: cs.textSecondary,
              side: BorderSide(color: cs.border)),
          icon: const Icon(LucideIcons.upload, size: 14),
          label: Text(buttonLabel, style: const TextStyle(fontSize: 12)),
          onPressed: () {},
        ),
      ]),
    );
  }

  Widget _configSection(BuildContext ctx,
      {required String title,
      required String description,
      required List<Widget> fields}) {
    final cs = consoleColors(ctx);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title,
            style: TextStyle(color: cs.textPrimary, fontSize: 14)),
        const SizedBox(height: 4),
        Text(description,
            style: TextStyle(color: cs.textSubtle, fontSize: 13)),
        const SizedBox(height: 16),
        ...fields,
      ]),
    );
  }

  Widget _configField(BuildContext ctx, String label, String value) {
    final cs = consoleColors(ctx);
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Row(children: [
        SizedBox(
            width: 180,
            child: Text(label,
                style: TextStyle(color: cs.textSubtle, fontSize: 13))),
        Expanded(
          child: Container(
            padding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
                color: cs.fieldFill,
                borderRadius: BorderRadius.circular(6),
                border: Border.all(color: cs.fieldBorder)),
            child: Text(
                value.isEmpty ? '—' : value,
                style: TextStyle(
                    color: value.isEmpty ? cs.textSubtle : cs.textPrimary,
                    fontSize: 13)),
          ),
        ),
      ]),
    );
  }

  // ── Distribution tab (desktop only) ──────────────────────────────────────────

  Widget _distributionTab(BuildContext ctx, Map<String, dynamic> t) {
    final cs = consoleColors(ctx);
    final platform = t['platform'] as String? ?? 'cross-platform';

    return SingleChildScrollView(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Distribution',
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 24),
          if (platform == 'macos' || platform == 'cross-platform') ...[
            _distributionSection(ctx,
                icon: LucideIcons.apple,
                title: 'macOS Distribution',
                items: [
                  _distItem(ctx, 'DMG Installer',
                      'Create a .dmg disk image for drag-and-drop install',
                      t['dmgEnabled'] == true),
                  _distItem(ctx, 'PKG Installer',
                      'Create a .pkg installer package',
                      t['pkgEnabled'] == true),
                  _distItem(ctx, 'Homebrew Cask',
                      'Publish to a Homebrew tap for `brew install` support',
                      t['homebrewEnabled'] == true),
                ]),
            const SizedBox(height: 16),
          ],
          if (platform == 'windows' || platform == 'cross-platform') ...[
            _distributionSection(ctx,
                icon: LucideIcons.monitor,
                title: 'Windows Distribution',
                items: [
                  _distItem(ctx, 'MSIX Package',
                      'Create MSIX package for modern Windows deployment',
                      t['msixEnabled'] == true),
                  _distItem(ctx, 'NSIS Installer',
                      'Create a traditional .exe installer with NSIS',
                      t['nsisEnabled'] == true),
                  _distItem(ctx, 'Microsoft Store',
                      'Publish to the Microsoft Store',
                      t['msStoreEnabled'] == true),
                ]),
            const SizedBox(height: 16),
          ],
          if (platform == 'linux' || platform == 'cross-platform') ...[
            _distributionSection(ctx,
                icon: LucideIcons.terminal,
                title: 'Linux Distribution',
                items: [
                  _distItem(ctx, 'DEB Package',
                      'Create .deb package for Debian/Ubuntu',
                      t['debEnabled'] == true),
                  _distItem(ctx, 'RPM Package',
                      'Create .rpm package for Fedora/RHEL',
                      t['rpmEnabled'] == true),
                  _distItem(ctx, 'AppImage',
                      'Create portable AppImage binary',
                      t['appImageEnabled'] == true),
                  _distItem(ctx, 'Snap',
                      'Create Snap package for the Snap Store',
                      t['snapEnabled'] == true),
                ]),
          ],
        ],
      ),
    );
  }

  Widget _distributionSection(BuildContext ctx,
      {required IconData icon,
      required String title,
      required List<Widget> items}) {
    final cs = consoleColors(ctx);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: cs.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Icon(icon, size: 16, color: _accent),
          const SizedBox(width: 8),
          Text(title,
              style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 14,
                  fontWeight: FontWeight.w600)),
        ]),
        const SizedBox(height: 16),
        ...items,
      ]),
    );
  }

  Widget _distItem(
      BuildContext ctx, String title, String description, bool enabled) {
    final cs = consoleColors(ctx);
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Row(children: [
        Container(
          width: 32,
          height: 32,
          decoration: BoxDecoration(
            color: enabled
                ? _green.withValues(alpha: 0.1)
                : cs.textSubtle.withValues(alpha: 0.05),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Icon(
            enabled ? LucideIcons.check : LucideIcons.minus,
            size: 14,
            color: enabled ? _green : cs.textSubtle,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(title,
                style: TextStyle(
                    color: cs.textPrimary,
                    fontSize: 13,
                    fontWeight: FontWeight.w500)),
            Text(description,
                style: TextStyle(color: cs.textSubtle, fontSize: 11)),
          ]),
        ),
        Text(
          enabled ? 'Configured' : 'Configure',
          style: TextStyle(
              color: enabled ? _green : _accent,
              fontSize: 12,
              fontWeight: FontWeight.w500),
        ),
      ]),
    );
  }

  // ── Logs tab (container only) ─────────────────────────────────────────────────

  Widget _logsTab(BuildContext ctx) {
    return FutureBuilder<dynamic>(
      future: ref
          .read(apiClientProvider)
          .get('/deploy/targets/$_selectedId/logs')
          .then((r) => r.data),
      builder: (ctx, snap) {
        final cs = consoleColors(ctx);
        if (!snap.hasData) {
          return const Center(child: CircularProgressIndicator());
        }
        final lines = List<String>.from(
            (snap.data as Map?)?['logs'] as List? ?? []);
        if (lines.isEmpty) {
          return Center(
            child: Text('No logs yet',
                style: TextStyle(color: cs.textSecondary, fontSize: 14)),
          );
        }
        return Container(
          margin: const EdgeInsets.all(32),
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: const Color(0xFF0D0D12),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: cs.border),
          ),
          child: ListView.builder(
            itemCount: lines.length,
            itemBuilder: (_, i) => Text(
              lines[i],
              style: const TextStyle(
                  color: Color(0xFFD4D4D8),
                  fontSize: 12,
                  fontFamily: 'monospace'),
            ),
          ),
        );
      },
    );
  }

  // ── Settings tab ─────────────────────────────────────────────────────────────

  Widget _settingsTab(
      BuildContext ctx, Map<String, dynamic> t, String type) {
    final cs = consoleColors(ctx);
    final projectId = ref.watch(currentProjectProvider) ?? '';
    final fields = <Widget>[];

    fields.add(_settingRow(ctx, 'Name', t['name'] ?? ''));
    fields.add(_settingRow(ctx, 'Type', _typeLabel(type)));

    switch (type) {
      case 'web':
        fields.add(_settingRow(ctx, 'Framework', t['framework'] ?? '—'));
        fields.add(_settingRow(ctx, 'Repository', t['repository'] ?? '—'));
        fields.add(_settingRow(ctx, 'Branch', t['branch'] ?? 'main'));
      case 'mobile':
        fields.add(_settingRow(
            ctx,
            'Platform',
            t['buildType'] == 'ipa' ? 'iOS' : 'Android'));
        fields.add(_settingRow(ctx, 'Build type', t['buildType'] ?? 'apk'));
      case 'desktop':
        fields.add(_settingRow(ctx, 'Platform',
            _desktopPlatformLabel(t['platform'] as String? ?? '')));
        fields.add(_settingRow(ctx, 'Framework', t['framework'] ?? '—'));
        fields.add(_settingRow(ctx, 'Repository', t['repository'] ?? '—'));
      case 'container':
        fields.add(_settingRow(ctx, 'Image', t['image'] ?? '—'));
        fields.add(_settingRow(ctx, 'Registry', t['registry'] ?? 'docker.io'));
        fields.add(_settingRow(ctx, 'Tag', t['tag'] ?? 'latest'));
    }

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
          ...fields,
          const SizedBox(height: 32),
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: cs.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: _red.withValues(alpha: 0.3)),
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
                  Text('Delete this platform and all its builds.',
                      style:
                          TextStyle(color: cs.textSubtle, fontSize: 13)),
                  const SizedBox(height: 12),
                  OutlinedButton(
                    style: OutlinedButton.styleFrom(
                        foregroundColor: _red,
                        side: const BorderSide(color: _red)),
                    onPressed: () async {
                      final confirmed = await showAppDialog<bool>(
                        context: context,
                        title: 'Delete platform',
                        content: Text(
                          'Delete this platform and all its builds. This action cannot be undone.',
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
                      if (confirmed != true) return;
                      await ref
                          .read(apiClientProvider)
                          .delete('/deploy/targets/$_selectedId');
                      ref.invalidate(_platformsProvider(projectId));
                      setState(() {
                        _selectedId = null;
                        _selectedType = null;
                        _detailTab = 0;
                      });
                    },
                    child: const Text('Delete platform'),
                  ),
                ]),
          ),
        ],
      ),
    );
  }

  Widget _settingRow(BuildContext ctx, String label, String value) {
    final cs = consoleColors(ctx);
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Row(children: [
        SizedBox(
            width: 140,
            child: Text(label,
                style: TextStyle(color: cs.textSubtle, fontSize: 13))),
        Expanded(
            child: Text(value,
                style:
                    TextStyle(color: cs.textPrimary, fontSize: 13))),
      ]),
    );
  }

  String _desktopPlatformLabel(String platform) => switch (platform) {
        'macos' => 'macOS',
        'windows' => 'Windows',
        'linux' => 'Linux',
        _ => 'Cross-platform',
      };

  // ── Add platform dialog ───────────────────────────────────────────────────────

  void _showAddDialog(BuildContext context, String projectId) {
    final colors = consoleColors(context);
    final nameCtrl = TextEditingController();
    String selectedType = 'web';

    showAppDialog(
      context: context,
      title: 'Add platform',
      subtitle: 'Choose where you want to deploy',
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
                      onTap: () =>
                          setDialogState(() => selectedType = t.id),
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 8),
                        decoration: BoxDecoration(
                          color: sel
                              ? _accent.withValues(alpha: 0.15)
                              : colors.fieldFill,
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(
                              color:
                                  sel ? _accent : colors.fieldBorder),
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
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (nameCtrl.text.trim().isEmpty) return;
            final nav = Navigator.of(context, rootNavigator: true);
            final api = ref.read(apiClientProvider);
            await api.post('/deploy/targets', data: {
              'type': selectedType,
              'name': nameCtrl.text.trim(),
            });
            ref.invalidate(_platformsProvider(projectId));
            nav.pop();
          },
        ),
      ],
    );
  }
}

// ── Grid card ─────────────────────────────────────────────────────────────────

class _GridCard extends StatelessWidget {
  final Map<String, dynamic> row;
  final IconData typeIcon;
  final String typeLabel;
  final String Function(dynamic) fmtDate;

  const _GridCard({
    required this.row,
    required this.typeIcon,
    required this.typeLabel,
    required this.fmtDate,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final name = row['name'] as String? ?? 'Unnamed';
    final status = row['status'] as String? ?? 'unknown';
    final updated = fmtDate(row[r'$updatedAt'] ?? row['updatedAt']);

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
            StatusChip.fromStatus(status),
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
          Text(typeLabel,
              style:
                  TextStyle(color: cs.textSecondary, fontSize: 12)),
          const Spacer(),
          Text('Updated $updated',
              style: TextStyle(color: cs.textSubtle, fontSize: 11)),
        ],
      ),
    );
  }
}
