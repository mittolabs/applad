import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/id_text.dart';
import '../../core/widgets/deploy_create_entry.dart';
import '../../core/widgets/search_list.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/theme/console_colors.dart';

const _accent = Color(0xFF3472A4);
const _green = Color(0xFF10B981);
const _red = Color(0xFFEF4444);
const _orange = Color(0xFFF59E0B);

final _containersProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final pid = ref.watch(currentProjectProvider);
  if (pid == null) return {'targets': []};
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy/targets?type=container');
  return res.data as Map<String, dynamic>;
});

class ContainersPage extends ConsumerStatefulWidget {
  const ContainersPage({super.key});
  @override
  ConsumerState<ContainersPage> createState() => _ContainersPageState();
}

class _ContainersPageState extends ConsumerState<ContainersPage> {
  String? _selectedId;
  int _detailTab = 0;
  final _searchCtrl = TextEditingController();
  int _page = 1;
  int _perPage = 6;
  late ConsoleColors _cs;

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
    if (_selectedId != null) return _detailView();

    final dataAsync = ref.watch(_containersProvider);

    return Scaffold(
      backgroundColor: _cs.background,
      body: Padding(
        padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Row(
              children: [
                Expanded(
                  child: Text('Containers',
                      style: TextStyle(
                          color: _cs.textPrimary,
                          fontSize: 22,
                          fontWeight: FontWeight.w600)),
                ),
                FilledButton.icon(
                  style: FilledButton.styleFrom(
                    backgroundColor: _accent,
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                    padding: const EdgeInsets.symmetric(
                        horizontal: 20, vertical: 12),
                  ),
                  icon: const Icon(LucideIcons.plus, size: 16),
                  label: const Text('Create container',
                      style: TextStyle(fontSize: 13)),
                  onPressed: _create,
                ),
              ],
            ),
            const SizedBox(height: 24),
            Divider(height: 1, color: _cs.border),
            const SizedBox(height: 20),
            Expanded(
              child: dataAsync.when(
                loading: () =>
                    const Center(child: CircularProgressIndicator(color: _accent)),
                error: (e, _) =>
                    Center(child: Text('$e', style: TextStyle(color: _cs.textMuted))),
                data: (data) {
                  final targets = List<Map<String, dynamic>>.from(
                      data['targets'] ?? []);
                  if (targets.isEmpty) return _emptyState();
                  return _list(targets);
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _emptyState() => Center(
    child: Column(mainAxisSize: MainAxisSize.min, children: [
      Container(
        width: 72, height: 72,
        decoration: BoxDecoration(color: _accent.withValues(alpha: 0.1), shape: BoxShape.circle),
        child: const Icon(LucideIcons.box, size: 32, color: _accent),
      ),
      const SizedBox(height: 20),
      Text('No containers yet', style: TextStyle(color: _cs.textPrimary, fontSize: 18, fontWeight: FontWeight.w600)),
      const SizedBox(height: 8),
      Text('Deploy your first Docker container', style: TextStyle(color: _cs.textMuted, fontSize: 14)),
      const SizedBox(height: 24),
      FilledButton.icon(
        style: FilledButton.styleFrom(backgroundColor: _accent, shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8))),
        icon: const Icon(LucideIcons.plus, size: 16),
        label: const Text('Create container'),
        onPressed: _create,
      ),
    ]),
  );

  Widget _list(List<Map<String, dynamic>> targets) {
    return Column(
      children: [
        SearchListHeader(
          searchController: _searchCtrl,
          total: targets.length,
          perPage: _perPage,
          currentPage: _page,
          onPerPageChanged: (v) => setState(() { _perPage = v; _page = 1; }),
          onPrev: () => setState(() => _page--),
          onNext: () => setState(() => _page++),
          onSearch: () => setState(() => _page = 1),
          trailing: const SizedBox.shrink(),
        ),
        const SizedBox(height: 16),
        Expanded(
          child: ListView.separated(
            padding: EdgeInsets.zero,
            itemCount: targets.length,
            separatorBuilder: (_, __) => const SizedBox(height: 8),
            itemBuilder: (ctx, i) {
              final t = targets[i];
              return GestureDetector(
                onTap: () => setState(() => _selectedId = t['\$id'] as String),
                child: MouseRegion(
                  cursor: SystemMouseCursors.click,
                  child: Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: _cs.surface, borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: _cs.border),
                    ),
                    child: Row(children: [
                      Container(
                        width: 40, height: 40,
                        decoration: BoxDecoration(color: _accent.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(8)),
                        child: const Icon(LucideIcons.box, size: 20, color: _accent),
                      ),
                      const SizedBox(width: 14),
                      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text(t['name'] ?? '', style: TextStyle(color: _cs.textPrimary, fontSize: 14, fontWeight: FontWeight.w500)),
                        Text(t['registryUrl'] ?? 'No registry configured', style: TextStyle(color: _cs.textSubtle, fontSize: 12)),
                      ])),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(color: _green.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(4)),
                        child: Text(t['tagStrategy'] ?? 'latest', style: const TextStyle(color: _green, fontSize: 11)),
                      ),
                    ]),
                  ),
                ),
              );
            },
          ),
        ),
      ],
    );
  }

  Widget _detailView() {
    return FutureBuilder<dynamic>(
      future: ref.read(apiClientProvider).get('/deploy/targets/$_selectedId').then((r) => r.data),
      builder: (ctx, snap) {
        if (!snap.hasData) return const Center(child: CircularProgressIndicator(color: _accent));
        final target = snap.data as Map<String, dynamic>;

        return Container(
          color: _cs.background,
          child: Column(children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(32, 20, 32, 16),
              child: Row(children: [
                IconButton(icon: Icon(LucideIcons.arrowLeft, size: 18, color: _cs.textMuted), onPressed: () => setState(() => _selectedId = null)),
                const SizedBox(width: 8),
                Expanded(child: Text(target['name'] ?? '', style: TextStyle(color: _cs.textPrimary, fontSize: 20, fontWeight: FontWeight.w600))),
              ]),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 32),
              child: PageTabs(
                tabs: const ['Overview', 'Images', 'Releases', 'Settings'],
                selected: _detailTab,
                onChanged: (i) => setState(() => _detailTab = i),
              ),
            ),
            const SizedBox(height: 16),
            Expanded(child: _detailTab == 0 ? _overviewTab(target) : _detailTab == 1 ? _imagesTab() : _detailTab == 2 ? _releasesTab() : _settingsTab(target)),
          ]),
        );
      },
    );
  }

  Widget _overviewTab(Map<String, dynamic> t) {
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          _infoCard('Registry', t['registryUrl'] ?? '—'),
          const SizedBox(width: 16),
          _infoCard('Dockerfile', t['dockerfile'] ?? 'Dockerfile'),
          const SizedBox(width: 16),
          _infoCard('Tag Strategy', t['tagStrategy'] ?? 'latest'),
        ]),
        const SizedBox(height: 24),
        Row(children: [
          _infoCard('Runtime', t['runtime'] ?? '—'),
          const SizedBox(width: 16),
          _infoCard('Type', t['type'] ?? 'container'),
        ]),
      ]),
    );
  }

  Widget _infoCard(String label, String value) => Expanded(
    child: Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(color: _cs.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _cs.border)),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(label, style: TextStyle(color: _cs.textSubtle, fontSize: 12)),
        const SizedBox(height: 6),
        Text(value, style: TextStyle(color: _cs.textPrimary, fontSize: 14, fontWeight: FontWeight.w500)),
      ]),
    ),
  );

  Widget _imagesTab() {
    return FutureBuilder<dynamic>(
      future: ref.read(apiClientProvider).get('/deploy/targets/$_selectedId/images').then((r) => r.data),
      builder: (ctx, snap) {
        if (!snap.hasData) return const Center(child: CircularProgressIndicator(color: _accent));
        final images = List<Map<String, dynamic>>.from((snap.data as Map)['images'] ?? []);
        if (images.isEmpty) return Center(child: Text('No images pushed yet', style: TextStyle(color: _cs.textMuted)));
        return ListView.separated(
          padding: const EdgeInsets.all(32),
          itemCount: images.length,
          separatorBuilder: (_, __) => const SizedBox(height: 8),
          itemBuilder: (ctx, i) {
            final img = images[i];
            return Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(color: _cs.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _cs.border)),
              child: Row(children: [
                const Icon(LucideIcons.container, size: 16, color: _accent),
                const SizedBox(width: 12),
                Expanded(child: Text('${img['repository'] ?? ''}:${img['tag'] ?? 'latest'}', style: TextStyle(color: _cs.textPrimary, fontSize: 13, fontFamily: 'monospace'))),
                Text(img['platform'] ?? '', style: TextStyle(color: _cs.textSubtle, fontSize: 11)),
                const SizedBox(width: 12),
                Text('${((img['sizeBytes'] ?? 0) / 1048576).toStringAsFixed(1)} MB', style: TextStyle(color: _cs.textSubtle, fontSize: 11)),
              ]),
            );
          },
        );
      },
    );
  }

  Widget _releasesTab() {
    return FutureBuilder<dynamic>(
      future: ref.read(apiClientProvider).get('/deploy/releases?targetId=$_selectedId').then((r) => r.data),
      builder: (ctx, snap) {
        if (!snap.hasData) return const Center(child: CircularProgressIndicator(color: _accent));
        final releases = List<Map<String, dynamic>>.from((snap.data as Map)['releases'] ?? []);
        if (releases.isEmpty) return Center(child: Text('No releases yet', style: TextStyle(color: _cs.textMuted)));
        return ListView.separated(
          padding: const EdgeInsets.all(32),
          itemCount: releases.length,
          separatorBuilder: (_, __) => const SizedBox(height: 8),
          itemBuilder: (ctx, i) {
            final r = releases[i];
            final status = r['status'] ?? 'pending';
            final sc = status == 'completed' ? _green : status == 'failed' ? _red : _orange;
            return Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(color: _cs.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _cs.border)),
              child: Row(children: [
                Container(width: 8, height: 8, decoration: BoxDecoration(color: sc, shape: BoxShape.circle)),
                const SizedBox(width: 12),
                Expanded(child: IdText(id: r['\$id'] ?? '', fontSize: 12)),
                Text(status, style: TextStyle(color: sc, fontSize: 12)),
                const SizedBox(width: 16),
                Text('${r['durationMs'] ?? 0}ms', style: TextStyle(color: _cs.textSubtle, fontSize: 11)),
              ]),
            );
          },
        );
      },
    );
  }

  Widget _settingsTab(Map<String, dynamic> t) {
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('Container Settings', style: TextStyle(color: _cs.textPrimary, fontSize: 16, fontWeight: FontWeight.w600)),
        const SizedBox(height: 24),
        _settingRow('Name', t['name'] ?? ''),
        _settingRow('Registry URL', t['registryUrl'] ?? '—'),
        _settingRow('Dockerfile', t['dockerfile'] ?? 'Dockerfile'),
        _settingRow('Tag Strategy', t['tagStrategy'] ?? 'latest'),
        const SizedBox(height: 32),
        Container(
          width: double.infinity, padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(color: _cs.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _red.withValues(alpha: 0.3))),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Danger zone', style: TextStyle(color: _red, fontSize: 14, fontWeight: FontWeight.w600)),
            const SizedBox(height: 8),
            Text('Delete this container target and all its images.', style: TextStyle(color: _cs.textSubtle, fontSize: 13)),
            const SizedBox(height: 12),
            OutlinedButton(
              style: OutlinedButton.styleFrom(foregroundColor: _red, side: const BorderSide(color: _red)),
              onPressed: () async {
                final confirmed = await showAppDialog<bool>(
                  context: context,
                  title: 'Delete container',
                  content: Text(
                    'Delete this container target and all its images. This action cannot be undone.',
                    style: TextStyle(color: _cs.textSecondary),
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
                ref.invalidate(_containersProvider);
                setState(() => _selectedId = null);
              },
              child: const Text('Delete container'),
            ),
          ]),
        ),
      ]),
    );
  }

  Widget _settingRow(String label, String value) => Padding(
    padding: const EdgeInsets.only(bottom: 16),
    child: Row(children: [
      SizedBox(width: 140, child: Text(label, style: TextStyle(color: _cs.textSubtle, fontSize: 13))),
      Expanded(child: Text(value, style: TextStyle(color: _cs.textPrimary, fontSize: 13))),
    ]),
  );

  void _create() async {
    final result = await showCreateEntryDialog(
      context: context,
      ref: ref,
      category: 'containers',
      title: 'Create Container',
      subtitle: 'Choose how to get started',
    );

    if (result == null || !mounted) return;

    String prefillName = '';
    String prefillRegistry = '';
    String prefillDockerfile = 'Dockerfile';
    String sourceType = 'manual';

    if (result.choice == CreateEntryChoice.template && result.templateConfig != null) {
      prefillName = result.templateConfig!['name'] ?? '';
      prefillRegistry = result.templateConfig!['registryUrl'] ?? '';
      prefillDockerfile = result.templateConfig!['dockerfile'] ?? 'Dockerfile';
      sourceType = 'template';
    } else if (result.choice == CreateEntryChoice.repository && result.repoConfig != null) {
      prefillName = result.repoConfig!['name'] ?? '';
      sourceType = 'git';
    } else {
      sourceType = 'upload';
    }

    final nameCtrl = TextEditingController(text: prefillName);
    final registryCtrl = TextEditingController(text: prefillRegistry);
    final dockerfileCtrl = TextEditingController(text: prefillDockerfile);
    showAppDialog(
      context: context,
      title: 'Create Container',
      subtitle: 'Deploy a Docker container',
      content: Column(mainAxisSize: MainAxisSize.min, children: [
        AppDialogField(controller: nameCtrl, label: 'Name', hint: 'my-api', autofocus: true),
        const SizedBox(height: 16),
        AppDialogField(controller: registryCtrl, label: 'Registry URL', hint: 'ghcr.io/myorg/myimage'),
        const SizedBox(height: 16),
        AppDialogField(controller: dockerfileCtrl, label: 'Dockerfile', hint: 'Dockerfile'),
      ]),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(label: 'Create', onTap: () async {
          await ref.read(apiClientProvider).post('/deploy/targets', data: {
            'name': nameCtrl.text.trim(),
            'type': 'container',
            'registryUrl': registryCtrl.text.trim(),
            'dockerfile': dockerfileCtrl.text.trim(),
            'tagStrategy': 'latest',
            'source': sourceType,
            if (result.templateConfig != null) 'templateId': result.templateConfig!['\$id'],
            if (result.repoConfig != null) 'repository': result.repoConfig!['cloneUrl'] ?? result.repoConfig!['url'] ?? '',
          });
          ref.invalidate(_containersProvider);
          if (mounted) Navigator.of(context, rootNavigator: true).pop();
        }),
      ],
    );
  }
}
