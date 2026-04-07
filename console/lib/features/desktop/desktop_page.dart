import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/deploy_create_entry.dart';
import '../../core/widgets/search_list.dart';
import '../../core/widgets/page_tabs.dart';

const _bg = Color(0xFF0B0B0F);
const _surface = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _border = Color(0x14FFFFFF);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _green = Color(0xFF10B981);
const _red = Color(0xFFEF4444);
const _orange = Color(0xFFF59E0B);
const _fieldFill = Color(0x0AFFFFFF);
const _fieldBorder = Color(0x1AFFFFFF);

final _desktopProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final pid = ref.watch(currentProjectProvider);
  if (pid == null) return {'targets': []};
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy/targets?type=desktop');
  return res.data as Map<String, dynamic>;
});

class DesktopPage extends ConsumerStatefulWidget {
  const DesktopPage({super.key});
  @override
  ConsumerState<DesktopPage> createState() => _DesktopPageState();
}

class _DesktopPageState extends ConsumerState<DesktopPage> {
  String? _selectedId;
  int _detailTab = 0;
  final _searchCtrl = TextEditingController();
  int _page = 1;
  int _perPage = 6;

  @override
  Widget build(BuildContext context) {
    if (_selectedId != null) return _detailView();

    final dataAsync = ref.watch(_desktopProvider);

    return Container(
      color: _bg,
      child: Column(children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(32, 28, 32, 0),
          child: Row(children: [
            const Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('Desktop Apps', style: TextStyle(color: Colors.white, fontSize: 24, fontWeight: FontWeight.w700)),
                SizedBox(height: 4),
                Text('Build and distribute macOS, Windows, and Linux desktop applications',
                    style: TextStyle(color: _dimText, fontSize: 14)),
              ]),
            ),
            FilledButton.icon(
              style: FilledButton.styleFrom(
                backgroundColor: _accent,
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
                padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
              ),
              icon: const Icon(LucideIcons.plus, size: 16),
              label: const Text('Create app', style: TextStyle(fontSize: 13)),
              onPressed: _create,
            ),
          ]),
        ),
        const SizedBox(height: 24),
        Expanded(
          child: dataAsync.when(
            loading: () => const Center(child: CircularProgressIndicator(color: _accent)),
            error: (e, _) => Center(child: Text('$e', style: const TextStyle(color: _dimText))),
            data: (data) {
              final targets = List<Map<String, dynamic>>.from(data['targets'] ?? []);
              if (targets.isEmpty) return _emptyState();
              return _list(targets);
            },
          ),
        ),
      ]),
    );
  }

  Widget _emptyState() => Center(
    child: Column(mainAxisSize: MainAxisSize.min, children: [
      Row(mainAxisSize: MainAxisSize.min, children: [
        Container(width: 56, height: 56, decoration: BoxDecoration(color: _accent.withOpacity(0.1), borderRadius: BorderRadius.circular(12)),
          child: const Icon(LucideIcons.monitor, size: 24, color: _accent)),
        const SizedBox(width: 16),
        Container(width: 56, height: 56, decoration: BoxDecoration(color: _green.withOpacity(0.1), borderRadius: BorderRadius.circular(12)),
          child: const Icon(LucideIcons.laptop, size: 24, color: _green)),
      ]),
      const SizedBox(height: 24),
      const Text('No desktop apps yet', style: TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.w600)),
      const SizedBox(height: 8),
      const Text('Build for macOS, Windows, and Linux from source', style: TextStyle(color: _dimText, fontSize: 14)),
      const SizedBox(height: 24),
      FilledButton.icon(
        style: FilledButton.styleFrom(backgroundColor: _accent, shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8))),
        icon: const Icon(LucideIcons.plus, size: 16),
        label: const Text('Create app'),
        onPressed: _create,
      ),
    ]),
  );

  Widget _list(List<Map<String, dynamic>> targets) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 32),
      child: Column(children: [
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
            itemCount: targets.length,
            separatorBuilder: (_, __) => const SizedBox(height: 8),
            itemBuilder: (ctx, i) {
              final t = targets[i];
              final platform = (t['platform'] ?? 'cross-platform') as String;
              final icon = _platformIcon(platform);
              final badgeColor = _platformColor(platform);
              return GestureDetector(
                onTap: () => setState(() { _selectedId = t['\$id'] as String; _detailTab = 0; }),
                child: MouseRegion(
                  cursor: SystemMouseCursors.click,
                  child: Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: _surface, borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: _border),
                    ),
                    child: Row(children: [
                      Container(
                        width: 40, height: 40,
                        decoration: BoxDecoration(color: _accent.withOpacity(0.1), borderRadius: BorderRadius.circular(8)),
                        child: Icon(icon, size: 20, color: _accent),
                      ),
                      const SizedBox(width: 14),
                      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text(t['name'] ?? '', style: const TextStyle(color: Colors.white, fontSize: 14, fontWeight: FontWeight.w500)),
                        Text(t['framework'] ?? 'No framework', style: const TextStyle(color: _subtleText, fontSize: 12)),
                      ])),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(color: badgeColor.withOpacity(0.1), borderRadius: BorderRadius.circular(4)),
                        child: Text(_platformLabel(platform), style: TextStyle(color: badgeColor, fontSize: 11)),
                      ),
                      const SizedBox(width: 10),
                      _buildStatusBadge(t['lastBuildStatus'] as String?),
                    ]),
                  ),
                ),
              );
            },
          ),
        ),
      ]),
    );
  }

  Widget _buildStatusBadge(String? status) {
    if (status == null || status.isEmpty) {
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
        decoration: BoxDecoration(color: _subtleText.withOpacity(0.1), borderRadius: BorderRadius.circular(4)),
        child: const Text('No builds', style: TextStyle(color: _subtleText, fontSize: 11)),
      );
    }
    final color = status == 'completed' ? _green : status == 'failed' ? _red : _orange;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(color: color.withOpacity(0.1), borderRadius: BorderRadius.circular(4)),
      child: Text(status, style: TextStyle(color: color, fontSize: 11)),
    );
  }

  IconData _platformIcon(String platform) {
    switch (platform) {
      case 'macos': return LucideIcons.apple;
      case 'windows': return LucideIcons.monitor;
      case 'linux': return LucideIcons.terminal;
      default: return LucideIcons.laptop;
    }
  }

  Color _platformColor(String platform) {
    switch (platform) {
      case 'macos': return const Color(0xFF9CA3AF);
      case 'windows': return const Color(0xFF3B82F6);
      case 'linux': return const Color(0xFFF59E0B);
      default: return _green;
    }
  }

  String _platformLabel(String platform) {
    switch (platform) {
      case 'macos': return 'macOS';
      case 'windows': return 'Windows';
      case 'linux': return 'Linux';
      default: return 'Cross-platform';
    }
  }

  // ── Detail view ─────────────────────────────────────────────────────────────

  Widget _detailView() {
    return FutureBuilder<dynamic>(
      future: ref.read(apiClientProvider).get('/deploy/targets/$_selectedId').then((r) => r.data),
      builder: (ctx, snap) {
        if (!snap.hasData) return const Center(child: CircularProgressIndicator(color: _accent));
        final target = snap.data as Map<String, dynamic>;

        return Container(
          color: _bg,
          child: Column(children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(32, 20, 32, 16),
              child: Row(children: [
                IconButton(icon: const Icon(LucideIcons.arrowLeft, size: 18, color: _dimText), onPressed: () => setState(() => _selectedId = null)),
                const SizedBox(width: 8),
                Expanded(child: Text(target['name'] ?? '', style: const TextStyle(color: Colors.white, fontSize: 20, fontWeight: FontWeight.w600))),
              ]),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 32),
              child: PageTabs(
                tabs: const ['Overview', 'Builds', 'Signing', 'Distribution', 'Settings'],
                selected: _detailTab,
                onChanged: (i) => setState(() => _detailTab = i),
              ),
            ),
            const SizedBox(height: 16),
            Expanded(child: _buildDetailTab(target)),
          ]),
        );
      },
    );
  }

  Widget _buildDetailTab(Map<String, dynamic> target) {
    switch (_detailTab) {
      case 0: return _overviewTab(target);
      case 1: return _buildsTab();
      case 2: return _signingTab(target);
      case 3: return _distributionTab(target);
      case 4: return _settingsTab(target);
      default: return const SizedBox.shrink();
    }
  }

  // ── Overview tab ────────────────────────────────────────────────────────────

  Widget _overviewTab(Map<String, dynamic> t) {
    final platform = (t['platform'] ?? 'cross-platform') as String;
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          _infoCard('Platform', _platformLabel(platform)),
          const SizedBox(width: 16),
          _infoCard('Framework', t['framework'] ?? '—'),
          const SizedBox(width: 16),
          _infoCard('Build Type', t['buildType'] ?? 'release'),
        ]),
        const SizedBox(height: 16),
        Row(children: [
          _infoCard('Source', t['source'] ?? 'manual'),
          const SizedBox(width: 16),
          _infoCard('Repository', t['repository'] ?? '—'),
          const SizedBox(width: 16),
          _infoCard('Branch', t['branch'] ?? 'main'),
        ]),
      ]),
    );
  }

  // ── Builds tab ──────────────────────────────────────────────────────────────

  Widget _buildsTab() {
    return FutureBuilder<dynamic>(
      future: ref.read(apiClientProvider).get('/deploy/releases?targetId=$_selectedId').then((r) => r.data),
      builder: (ctx, snap) {
        if (!snap.hasData) return const Center(child: CircularProgressIndicator(color: _accent));
        final releases = List<Map<String, dynamic>>.from((snap.data as Map)['releases'] ?? []);
        if (releases.isEmpty) return const Center(child: Text('No builds yet', style: TextStyle(color: _dimText)));
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
              decoration: BoxDecoration(color: _surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _border)),
              child: Row(children: [
                Container(width: 8, height: 8, decoration: BoxDecoration(color: sc, shape: BoxShape.circle)),
                const SizedBox(width: 12),
                Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text(r['\$id'] ?? '', style: const TextStyle(color: Colors.white, fontSize: 12, fontFamily: 'monospace')),
                  Text('${r['triggerType'] ?? 'manual'} • ${r['durationMs'] ?? 0}ms', style: const TextStyle(color: _subtleText, fontSize: 11)),
                ])),
                Text(status, style: TextStyle(color: sc, fontSize: 12)),
              ]),
            );
          },
        );
      },
    );
  }

  // ── Signing tab ─────────────────────────────────────────────────────────────

  Widget _signingTab(Map<String, dynamic> t) {
    final platform = (t['platform'] ?? 'cross-platform') as String;
    return SingleChildScrollView(
      padding: const EdgeInsets.all(32),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Text('Code Signing', style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.w600)),
        const SizedBox(height: 24),
        // macOS signing
        if (platform == 'macos' || platform == 'cross-platform') ...[
          _signingSection(
            title: 'macOS Certificate',
            description: 'Upload your Developer ID certificate (.p12) for macOS code signing',
            buttonLabel: 'Upload .p12 certificate',
          ),
          const SizedBox(height: 16),
          _configSection(
            title: 'Apple Developer Team ID',
            description: 'Required for notarization and distribution outside the Mac App Store',
            fields: [
              _configField('Team ID', t['appleTeamId'] ?? ''),
            ],
          ),
          const SizedBox(height: 16),
          _configSection(
            title: 'Notarization',
            description: 'Apple notarization ensures your app is checked for malicious content',
            fields: [
              _configField('Apple ID', t['appleId'] ?? ''),
              _configField('App-specific password', t['notarizationPassword'] != null ? '********' : ''),
            ],
          ),
          const SizedBox(height: 24),
        ],
        // Windows signing
        if (platform == 'windows' || platform == 'cross-platform') ...[
          _signingSection(
            title: 'Windows Code Signing Certificate',
            description: 'Upload your code signing certificate (.pfx) for Windows executables',
            buttonLabel: 'Upload .pfx certificate',
          ),
          const SizedBox(height: 24),
        ],
        // Linux signing
        if (platform == 'linux' || platform == 'cross-platform') ...[
          _configSection(
            title: 'Linux GPG Signing',
            description: 'Configure GPG key for signing Linux packages',
            fields: [
              _configField('GPG Key ID', t['gpgKeyId'] ?? ''),
              _configField('GPG Key Server', t['gpgKeyServer'] ?? 'hkps://keys.openpgp.org'),
            ],
          ),
        ],
      ]),
    );
  }

  Widget _signingSection({required String title, required String description, required String buttonLabel}) {
    return Container(
      width: double.infinity, padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(color: _surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _border)),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title, style: const TextStyle(color: Colors.white, fontSize: 14)),
        const SizedBox(height: 4),
        Text(description, style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 13)),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          style: OutlinedButton.styleFrom(foregroundColor: Colors.white70, side: BorderSide(color: Colors.white.withOpacity(0.12))),
          icon: const Icon(LucideIcons.upload, size: 14),
          label: Text(buttonLabel, style: const TextStyle(fontSize: 12)),
          onPressed: () {},
        ),
      ]),
    );
  }

  Widget _configSection({required String title, required String description, required List<Widget> fields}) {
    return Container(
      width: double.infinity, padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(color: _surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _border)),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title, style: const TextStyle(color: Colors.white, fontSize: 14)),
        const SizedBox(height: 4),
        Text(description, style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 13)),
        const SizedBox(height: 16),
        ...fields,
      ]),
    );
  }

  Widget _configField(String label, String value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Row(children: [
        SizedBox(width: 180, child: Text(label, style: const TextStyle(color: _subtleText, fontSize: 13))),
        Expanded(child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(color: _fieldFill, borderRadius: BorderRadius.circular(6), border: Border.all(color: _fieldBorder)),
          child: Text(value.isEmpty ? '—' : value, style: TextStyle(color: value.isEmpty ? _subtleText : Colors.white, fontSize: 13)),
        )),
      ]),
    );
  }

  // ── Distribution tab ────────────────────────────────────────────────────────

  Widget _distributionTab(Map<String, dynamic> t) {
    final platform = (t['platform'] ?? 'cross-platform') as String;
    return SingleChildScrollView(
      padding: const EdgeInsets.all(32),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Text('Distribution', style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.w600)),
        const SizedBox(height: 24),
        // macOS distribution
        if (platform == 'macos' || platform == 'cross-platform') ...[
          _distributionSection(
            icon: LucideIcons.apple,
            title: 'macOS Distribution',
            items: [
              _distributionItem('DMG Installer', 'Create a .dmg disk image for drag-and-drop install', t['dmgEnabled'] == true),
              _distributionItem('PKG Installer', 'Create a .pkg installer package', t['pkgEnabled'] == true),
              _distributionItem('Homebrew Cask', 'Publish to a Homebrew tap for `brew install` support', t['homebrewEnabled'] == true),
            ],
          ),
          const SizedBox(height: 16),
        ],
        // Windows distribution
        if (platform == 'windows' || platform == 'cross-platform') ...[
          _distributionSection(
            icon: LucideIcons.monitor,
            title: 'Windows Distribution',
            items: [
              _distributionItem('MSIX Package', 'Create MSIX package for modern Windows deployment', t['msixEnabled'] == true),
              _distributionItem('NSIS Installer', 'Create a traditional .exe installer with NSIS', t['nsisEnabled'] == true),
              _distributionItem('Microsoft Store', 'Publish to the Microsoft Store', t['msStoreEnabled'] == true),
            ],
          ),
          const SizedBox(height: 16),
        ],
        // Linux distribution
        if (platform == 'linux' || platform == 'cross-platform') ...[
          _distributionSection(
            icon: LucideIcons.terminal,
            title: 'Linux Distribution',
            items: [
              _distributionItem('DEB Package', 'Create .deb package for Debian/Ubuntu', t['debEnabled'] == true),
              _distributionItem('RPM Package', 'Create .rpm package for Fedora/RHEL', t['rpmEnabled'] == true),
              _distributionItem('AppImage', 'Create portable AppImage binary', t['appImageEnabled'] == true),
              _distributionItem('Flatpak', 'Create Flatpak package for Flathub', t['flatpakEnabled'] == true),
              _distributionItem('Snap', 'Create Snap package for the Snap Store', t['snapEnabled'] == true),
            ],
          ),
        ],
      ]),
    );
  }

  Widget _distributionSection({required IconData icon, required String title, required List<Widget> items}) {
    return Container(
      width: double.infinity, padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(color: _surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _border)),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Icon(icon, size: 16, color: _accent),
          const SizedBox(width: 8),
          Text(title, style: const TextStyle(color: Colors.white, fontSize: 14, fontWeight: FontWeight.w600)),
        ]),
        const SizedBox(height: 16),
        ...items,
      ]),
    );
  }

  Widget _distributionItem(String title, String description, bool enabled) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Row(children: [
        Container(
          width: 32, height: 32,
          decoration: BoxDecoration(
            color: enabled ? _green.withOpacity(0.1) : _subtleText.withOpacity(0.05),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Icon(
            enabled ? LucideIcons.check : LucideIcons.minus,
            size: 14,
            color: enabled ? _green : _subtleText,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(title, style: const TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w500)),
          Text(description, style: TextStyle(color: Colors.white.withOpacity(0.35), fontSize: 11)),
        ])),
        GestureDetector(
          onTap: () {},
          child: MouseRegion(
            cursor: SystemMouseCursors.click,
            child: Text(
              enabled ? 'Configured' : 'Configure',
              style: TextStyle(color: enabled ? _green : _accent, fontSize: 12, fontWeight: FontWeight.w500),
            ),
          ),
        ),
      ]),
    );
  }

  // ── Settings tab ────────────────────────────────────────────────────────────

  Widget _settingsTab(Map<String, dynamic> t) {
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Text('App Settings', style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.w600)),
        const SizedBox(height: 24),
        _settingRow('Name', t['name'] ?? ''),
        _settingRow('Platform', _platformLabel((t['platform'] ?? 'cross-platform') as String)),
        _settingRow('Framework', t['framework'] ?? '—'),
        _settingRow('Source', t['source'] ?? 'manual'),
        _settingRow('Repository', t['repository'] ?? '—'),
        _settingRow('Branch', t['branch'] ?? 'main'),
        const SizedBox(height: 32),
        Container(
          width: double.infinity, padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(color: _surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _red.withOpacity(0.3))),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Danger zone', style: TextStyle(color: _red, fontSize: 14, fontWeight: FontWeight.w600)),
            const SizedBox(height: 8),
            Text('Delete this app and all its builds.', style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 13)),
            const SizedBox(height: 12),
            OutlinedButton(
              style: OutlinedButton.styleFrom(foregroundColor: _red, side: const BorderSide(color: _red)),
              onPressed: () async {
                await ref.read(apiClientProvider).delete('/deploy/targets/$_selectedId');
                ref.invalidate(_desktopProvider);
                setState(() => _selectedId = null);
              },
              child: const Text('Delete app'),
            ),
          ]),
        ),
      ]),
    );
  }

  Widget _infoCard(String label, String value) => Expanded(
    child: Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(color: _surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: _border)),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(label, style: const TextStyle(color: _subtleText, fontSize: 12)),
        const SizedBox(height: 6),
        Text(value, style: const TextStyle(color: Colors.white, fontSize: 14, fontWeight: FontWeight.w500)),
      ]),
    ),
  );

  Widget _settingRow(String label, String value) => Padding(
    padding: const EdgeInsets.only(bottom: 16),
    child: Row(children: [
      SizedBox(width: 140, child: Text(label, style: const TextStyle(color: _subtleText, fontSize: 13))),
      Expanded(child: Text(value, style: const TextStyle(color: Colors.white, fontSize: 13))),
    ]),
  );

  // ── Create flow ─────────────────────────────────────────────────────────────

  void _create() async {
    final result = await showCreateEntryDialog(
      context: context,
      ref: ref,
      category: 'desktop',
      title: 'Create Desktop App',
      subtitle: 'Choose how to get started',
    );

    if (result == null || !mounted) return;

    // Pre-fill from template or repo
    String prefillName = '';
    String prefillFramework = 'flutter';
    String prefillRepo = '';
    String prefillBranch = 'main';
    String sourceType = 'manual';

    if (result.choice == CreateEntryChoice.template && result.templateConfig != null) {
      prefillName = result.templateConfig!['name'] ?? '';
      prefillFramework = result.templateConfig!['framework'] ?? 'flutter';
      sourceType = 'template';
    } else if (result.choice == CreateEntryChoice.repository && result.repoConfig != null) {
      prefillName = result.repoConfig!['name'] ?? '';
      prefillRepo = result.repoConfig!['cloneUrl'] ?? result.repoConfig!['url'] ?? '';
      prefillBranch = result.repoConfig!['defaultBranch'] ?? 'main';
      sourceType = 'git';
    } else {
      sourceType = 'upload';
    }

    _showCreateForm(
      prefillName: prefillName,
      prefillFramework: prefillFramework,
      prefillRepo: prefillRepo,
      prefillBranch: prefillBranch,
      sourceType: sourceType,
      templateConfig: result.templateConfig,
    );
  }

  void _showCreateForm({
    required String prefillName,
    required String prefillFramework,
    required String prefillRepo,
    required String prefillBranch,
    required String sourceType,
    Map<String, dynamic>? templateConfig,
  }) {
    final nameCtrl = TextEditingController(text: prefillName);
    String platform = 'cross-platform';
    String framework = prefillFramework;

    showAppDialog(
      context: context,
      title: 'Create Desktop App',
      subtitle: 'Configure your desktop application',
      width: 500,
      content: StatefulBuilder(builder: (ctx, setD) => Column(mainAxisSize: MainAxisSize.min, children: [
        AppDialogField(controller: nameCtrl, label: 'App name', hint: 'my-desktop-app', autofocus: true),
        const SizedBox(height: 16),
        Text('Platform', style: TextStyle(color: Colors.white.withOpacity(0.5), fontSize: 12, fontWeight: FontWeight.w500)),
        const SizedBox(height: 8),
        Wrap(spacing: 8, runSpacing: 8, children: [
          _platformChip('macOS', 'macos', LucideIcons.apple, platform, (v) => setD(() => platform = v)),
          _platformChip('Windows', 'windows', LucideIcons.monitor, platform, (v) => setD(() => platform = v)),
          _platformChip('Linux', 'linux', LucideIcons.terminal, platform, (v) => setD(() => platform = v)),
          _platformChip('Cross-platform', 'cross-platform', LucideIcons.laptop, platform, (v) => setD(() => platform = v)),
        ]),
        const SizedBox(height: 16),
        Text('Framework', style: TextStyle(color: Colors.white.withOpacity(0.5), fontSize: 12, fontWeight: FontWeight.w500)),
        const SizedBox(height: 8),
        Wrap(spacing: 8, runSpacing: 8, children: [
          _frameworkChip('Flutter', 'flutter', framework, (v) => setD(() => framework = v)),
          _frameworkChip('Electron', 'electron', framework, (v) => setD(() => framework = v)),
          _frameworkChip('Tauri', 'tauri', framework, (v) => setD(() => framework = v)),
        ]),
      ])),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(label: 'Create', onTap: () async {
          await ref.read(apiClientProvider).post('/deploy/targets', data: {
            'name': nameCtrl.text.trim(),
            'type': 'desktop',
            'platform': platform,
            'framework': framework,
            'source': sourceType,
            'repository': prefillRepo,
            'branch': prefillBranch,
            if (templateConfig != null) 'templateId': templateConfig['\$id'],
          });
          ref.invalidate(_desktopProvider);
          if (ctx.mounted) Navigator.pop(ctx);
        }),
      ],
    );
  }

  Widget _platformChip(String label, String value, IconData icon, String current, ValueChanged<String> onTap) {
    final active = value == current;
    return GestureDetector(
      onTap: () => onTap(value),
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          width: 110,
          padding: const EdgeInsets.symmetric(vertical: 12),
          decoration: BoxDecoration(
            color: active ? _accent.withOpacity(0.15) : _surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: active ? _accent : _border),
          ),
          child: Column(children: [
            Icon(icon, size: 18, color: active ? _accent : _dimText),
            const SizedBox(height: 5),
            Text(label, style: TextStyle(color: active ? Colors.white : _dimText, fontSize: 12, fontWeight: active ? FontWeight.w600 : FontWeight.w400)),
          ]),
        ),
      ),
    );
  }

  Widget _frameworkChip(String label, String value, String current, ValueChanged<String> onTap) {
    final active = value == current;
    return GestureDetector(
      onTap: () => onTap(value),
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
            color: active ? _accent.withOpacity(0.15) : _surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: active ? _accent : _border),
          ),
          child: Text(label, style: TextStyle(color: active ? Colors.white : _dimText, fontSize: 13, fontWeight: active ? FontWeight.w600 : FontWeight.w400)),
        ),
      ),
    );
  }
}
