import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/deploy_create_entry.dart';
import '../../core/widgets/id_text.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_error_state.dart';
import '../../core/widgets/page_tabs.dart';

const _accent = Color(0xFF3472A4);
const _green = Color(0xFF10B981);
const _red = Color(0xFFEF4444);

final _mobileProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final pid = ref.watch(currentProjectProvider);
  if (pid == null) return {'targets': []};
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy/targets?type=mobile');
  return res.data as Map<String, dynamic>;
});

class MobilePage extends ConsumerStatefulWidget {
  const MobilePage({super.key});
  @override
  ConsumerState<MobilePage> createState() => _MobilePageState();
}

class _MobilePageState extends ConsumerState<MobilePage> {
  String? _selectedId;
  int _detailTab = 0;
  final _searchCtrl = TextEditingController();
  int _page = 1;
  int _perPage = 12;

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  String _fmtDate(dynamic v) => v?.toString().split('T').first ?? '—';

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    if (_selectedId != null) return _detailView();
    final dataAsync = ref.watch(_mobileProvider);

    return Scaffold(
      backgroundColor: colors.background,
      body: Padding(
        padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Text('Mobile Apps', style: TextStyle(color: colors.textPrimary, fontSize: 22, fontWeight: FontWeight.w600)),
            const SizedBox(height: 24),
            Expanded(
              child: dataAsync.when(
                loading: () => const Center(child: CircularProgressIndicator(color: _accent)),
                error: (e, _) => AppErrorState(error: e, onRetry: () => ref.invalidate(_mobileProvider)),
                data: (data) {
                  final targets = List<Map<String, dynamic>>.from(data['targets'] ?? []);
                  final search = _searchCtrl.text.toLowerCase();
                  final filtered = search.isEmpty ? targets : targets.where((t) => (t['name'] as String? ?? '').toLowerCase().contains(search)).toList();
                  return AppDataTable(
                    columns: const [
                      AppTableColumn(key: 'name',      label: 'Name',    flex: 4),
                      AppTableColumn(key: 'buildType', label: 'Type',    flex: 2),
                      AppTableColumn(key: 'status',    label: 'Status',  flex: 2),
                      AppTableColumn(key: 'updatedAt', label: 'Updated', flex: 2),
                    ],
                    rows: filtered,
                    getCellValue: (row, key) => switch (key) {
                      'name'      => row['name'] as String? ?? '',
                      'buildType' => row['buildType'] as String? ?? '',
                      'status'    => row['status'] as String? ?? 'active',
                      'updatedAt' => _fmtDate(row['updatedAt'] ?? row[r'$updatedAt']),
                      _           => '',
                    },
                    getRowIcon: (row) => (row['buildType'] as String? ?? '') == 'ipa' ? LucideIcons.tablet : LucideIcons.smartphone,
                    onRowTap: (row) => setState(() => _selectedId = row[r'$id'] as String? ?? row['id'] as String?),
                    createLabel: 'Create app',
                    onCreateTap: _create,
                    total: filtered.length,
                    perPage: _perPage,
                    currentPage: _page,
                    onPrev: () => setState(() => _page--),
                    onNext: () => setState(() => _page++),
                    onPerPageChanged: (v) => setState(() { _perPage = v; _page = 1; }),
                    itemLabel: 'apps',
                    searchController: _searchCtrl,
                    onSearch: () => setState(() {}),
                    emptyIcon: LucideIcons.plus,
                    emptyTitle: 'No mobile apps yet',
                    emptySubtitle: 'Build Android APK/AAB and iOS IPA from source.',
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _detailView() {
    return FutureBuilder<dynamic>(
      future: ref.read(apiClientProvider).get('/deploy/targets/$_selectedId').then((r) => r.data),
      builder: (ctx, snap) {
        final colors = consoleColors(ctx);
        if (!snap.hasData) return const Center(child: CircularProgressIndicator(color: _accent));
        final target = snap.data as Map<String, dynamic>;
        return Container(
          color: colors.background,
          child: Column(children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(32, 20, 32, 16),
              child: Row(children: [
                IconButton(icon: Icon(LucideIcons.arrowLeft, size: 18, color: colors.textSecondary), onPressed: () => setState(() => _selectedId = null)),
                const SizedBox(width: 8),
                Expanded(child: Text(target['name'] ?? '', style: TextStyle(color: colors.textPrimary, fontSize: 20, fontWeight: FontWeight.w600))),
              ]),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 32),
              child: PageTabs(
                tabs: const ['Overview', 'Builds', 'Signing', 'Settings'],
                selected: _detailTab,
                onChanged: (i) => setState(() => _detailTab = i),
              ),
            ),
            const SizedBox(height: 16),
            Expanded(child: _detailTab == 0 ? _overviewTab(target) : _detailTab == 1 ? _buildsTab() : _detailTab == 2 ? _signingTab(target) : _settingsTab(target)),
          ]),
        );
      },
    );
  }

  Widget _overviewTab(Map<String, dynamic> t) {
    final colors = consoleColors(context);
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          _infoCard('Platform', t['buildType'] == 'ipa' ? 'iOS' : 'Android'),
          const SizedBox(width: 16),
          _infoCard('Build Type', t['buildType'] ?? 'apk'),
          const SizedBox(width: 16),
          _infoCard('Runtime', t['runtime'] ?? '—'),
        ]),
        const SizedBox(height: 24),
        Text('Distribution', style: TextStyle(color: colors.textPrimary, fontSize: 16, fontWeight: FontWeight.w600)),
        const SizedBox(height: 12),
        Container(
          width: double.infinity, padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(color: colors.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: colors.border)),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text('Store publishing and distribution can be configured in Settings.',
                style: TextStyle(color: colors.textSecondary, fontSize: 13)),
          ]),
        ),
      ]),
    );
  }

  Widget _infoCard(String label, String value) {
    final colors = consoleColors(context);
    return Expanded(
    child: Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(color: colors.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: colors.border)),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(label, style: TextStyle(color: colors.textSubtle, fontSize: 12)),
        const SizedBox(height: 6),
        Text(value, style: TextStyle(color: colors.textPrimary, fontSize: 14, fontWeight: FontWeight.w500)),
      ]),
    ),
  );
  }

  Widget _buildsTab() {
    return FutureBuilder<dynamic>(
      future: ref.read(apiClientProvider).get('/deploy/releases?targetId=$_selectedId').then((r) => r.data),
      builder: (ctx, snap) {
        final colors = consoleColors(ctx);
        if (!snap.hasData) return const Center(child: CircularProgressIndicator(color: _accent));
        final releases = List<Map<String, dynamic>>.from((snap.data as Map)['releases'] ?? []);
        if (releases.isEmpty) return Center(child: Text('No builds yet', style: TextStyle(color: colors.textSecondary)));
        return ListView.separated(
          padding: const EdgeInsets.all(32),
          itemCount: releases.length,
          separatorBuilder: (_, __) => const SizedBox(height: 8),
          itemBuilder: (ctx, i) {
            final r = releases[i];
            final status = r['status'] ?? 'pending';
            final sc = status == 'completed' ? _green : status == 'failed' ? _red : const Color(0xFFF59E0B);
            return Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(color: colors.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: colors.border)),
              child: Row(children: [
                Container(width: 8, height: 8, decoration: BoxDecoration(color: sc, shape: BoxShape.circle)),
                const SizedBox(width: 12),
                Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  IdText(id: r['\$id'] ?? '', fontSize: 12),
                  Text('${r['triggerType'] ?? 'manual'} • ${r['durationMs'] ?? 0}ms', style: TextStyle(color: colors.textSubtle, fontSize: 11)),
                ])),
                Text(status, style: TextStyle(color: sc, fontSize: 12)),
              ]),
            );
          },
        );
      },
    );
  }

  Widget _signingTab(Map<String, dynamic> t) {
    final colors = consoleColors(context);
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('Code Signing', style: TextStyle(color: colors.textPrimary, fontSize: 16, fontWeight: FontWeight.w600)),
        const SizedBox(height: 24),
        Container(
          width: double.infinity, padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(color: colors.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: colors.border)),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text('Android Keystore', style: TextStyle(color: colors.textPrimary, fontSize: 14)),
            const SizedBox(height: 4),
            Text('Upload your keystore file for signed APK/AAB builds', style: TextStyle(color: colors.textSecondary, fontSize: 13)),
            const SizedBox(height: 12),
            OutlinedButton.icon(
              style: OutlinedButton.styleFrom(foregroundColor: colors.textSecondary, side: BorderSide(color: colors.border)),
              icon: const Icon(LucideIcons.upload, size: 14),
              label: const Text('Upload keystore', style: TextStyle(fontSize: 12)),
              onPressed: () {},
            ),
          ]),
        ),
        const SizedBox(height: 16),
        Container(
          width: double.infinity, padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(color: colors.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: colors.border)),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text('iOS Provisioning', style: TextStyle(color: colors.textPrimary, fontSize: 14)),
            const SizedBox(height: 4),
            Text('Upload provisioning profile + P12 certificate for IPA builds', style: TextStyle(color: colors.textSecondary, fontSize: 13)),
            const SizedBox(height: 12),
            OutlinedButton.icon(
              style: OutlinedButton.styleFrom(foregroundColor: colors.textSecondary, side: BorderSide(color: colors.border)),
              icon: const Icon(LucideIcons.upload, size: 14),
              label: const Text('Upload profile', style: TextStyle(fontSize: 12)),
              onPressed: () {},
            ),
          ]),
        ),
      ]),
    );
  }

  Widget _settingsTab(Map<String, dynamic> t) {
    final colors = consoleColors(context);
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('App Settings', style: TextStyle(color: colors.textPrimary, fontSize: 16, fontWeight: FontWeight.w600)),
        const SizedBox(height: 24),
        _settingRow('Name', t['name'] ?? ''),
        _settingRow('Platform', t['buildType'] == 'ipa' ? 'iOS' : 'Android'),
        _settingRow('Build Type', t['buildType'] ?? 'apk'),
        const SizedBox(height: 32),
        Container(
          width: double.infinity, padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(color: colors.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _red.withValues(alpha: 0.3))),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Danger zone', style: TextStyle(color: _red, fontSize: 14, fontWeight: FontWeight.w600)),
            const SizedBox(height: 8),
            Text('Delete this app and all its builds.', style: TextStyle(color: colors.textSecondary, fontSize: 13)),
            const SizedBox(height: 12),
            OutlinedButton(
              style: OutlinedButton.styleFrom(foregroundColor: _red, side: const BorderSide(color: _red)),
              onPressed: () async {
                final confirmed = await showAppDialog<bool>(
                  context: context,
                  title: 'Delete app',
                  content: Text(
                    'Delete this app and all its builds. This action cannot be undone.',
                    style: TextStyle(color: colors.textSecondary),
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
                await ref.read(apiClientProvider).delete('/deploy/targets/$_selectedId');
                ref.invalidate(_mobileProvider);
                setState(() => _selectedId = null);
              },
              child: const Text('Delete app'),
            ),
          ]),
        ),
      ]),
    );
  }

  Widget _settingRow(String label, String value) {
    final colors = consoleColors(context);
    return Padding(
    padding: const EdgeInsets.only(bottom: 16),
    child: Row(children: [
      SizedBox(width: 140, child: Text(label, style: TextStyle(color: colors.textSubtle, fontSize: 13))),
      Expanded(child: Text(value, style: TextStyle(color: colors.textPrimary, fontSize: 13))),
    ]),
  );
  }

  void _create() async {
    final result = await showCreateEntryDialog(
      context: context,
      ref: ref,
      category: 'mobile',
      title: 'Create Mobile App',
      subtitle: 'Choose how to get started',
    );

    if (result == null || !mounted) return;

    String prefillName = '';
    String sourceType = 'manual';

    if (result.choice == CreateEntryChoice.template && result.templateConfig != null) {
      prefillName = result.templateConfig!['name'] ?? '';
      sourceType = 'template';
    } else if (result.choice == CreateEntryChoice.repository && result.repoConfig != null) {
      prefillName = result.repoConfig!['name'] ?? '';
      sourceType = 'git';
    } else {
      sourceType = 'upload';
    }

    final nameCtrl = TextEditingController(text: prefillName);
    String platform = 'android';
    showAppDialog(
      context: context,
      title: 'Create Mobile App',
      subtitle: 'Build for Android or iOS',
      content: StatefulBuilder(builder: (ctx, setD) => Column(mainAxisSize: MainAxisSize.min, children: [
        AppDialogField(controller: nameCtrl, label: 'App name', hint: 'my-app', autofocus: true),
        const SizedBox(height: 16),
        Row(children: [
          _platformChip('Android', 'android', LucideIcons.smartphone, platform, (v) => setD(() => platform = v)),
          const SizedBox(width: 8),
          _platformChip('iOS', 'ios', LucideIcons.tablet, platform, (v) => setD(() => platform = v)),
        ]),
      ])),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(label: 'Create', onTap: () async {
          await ref.read(apiClientProvider).post('/deploy/targets', data: {
            'name': nameCtrl.text.trim(),
            'type': 'mobile',
            'buildType': platform == 'ios' ? 'ipa' : 'apk',
            'source': sourceType,
            if (result.templateConfig != null) 'templateId': result.templateConfig!['\$id'],
            if (result.repoConfig != null) 'repository': result.repoConfig!['cloneUrl'] ?? result.repoConfig!['url'] ?? '',
            if (result.repoConfig != null) 'branch': result.repoConfig!['defaultBranch'] ?? 'main',
          });
          ref.invalidate(_mobileProvider);
          if (mounted) Navigator.of(context, rootNavigator: true).pop();
        }),
      ],
    );
  }

  Widget _platformChip(String label, String value, IconData icon, String current, ValueChanged<String> onTap) {
    final colors = consoleColors(context);
    final active = value == current;
    return Expanded(
      child: GestureDetector(
        onTap: () => onTap(value),
        child: Container(
          padding: const EdgeInsets.symmetric(vertical: 14),
          decoration: BoxDecoration(
            color: active ? _accent.withValues(alpha: 0.15) : colors.surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: active ? _accent : colors.border),
          ),
          child: Column(children: [
            Icon(icon, size: 20, color: active ? _accent : colors.textSecondary),
            const SizedBox(height: 6),
            Text(label, style: TextStyle(color: active ? colors.textPrimary : colors.textSecondary, fontSize: 13, fontWeight: active ? FontWeight.w600 : FontWeight.w400)),
          ]),
        ),
      ),
    );
  }
}
