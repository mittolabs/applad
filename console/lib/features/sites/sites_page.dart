import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/deploy_create_entry.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/search_list.dart';

// --- Colors ------------------------------------------------------------------

const _bg = Color(0xFF0B0B0F);
const _surface = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _green = Color(0xFF10B981);
const _red = Color(0xFFEF4444);
const _orange = Color(0xFFF59E0B);
const _fieldFill = Color(0x0AFFFFFF);
const _fieldBorder = Color(0x1AFFFFFF);

// --- Framework metadata ------------------------------------------------------

class _Framework {
  final String id;
  final String label;
  final IconData icon;
  final String buildCommand;
  final String outputDir;
  final String installCommand;

  const _Framework({
    required this.id,
    required this.label,
    required this.icon,
    required this.buildCommand,
    required this.outputDir,
    required this.installCommand,
  });
}

const _frameworks = <_Framework>[
  _Framework(
    id: 'nextjs',
    label: 'Next.js',
    icon: LucideIcons.hexagon,
    buildCommand: 'npm run build',
    outputDir: '.next',
    installCommand: 'npm install',
  ),
  _Framework(
    id: 'sveltekit',
    label: 'SvelteKit',
    icon: LucideIcons.flame,
    buildCommand: 'npm run build',
    outputDir: 'build',
    installCommand: 'npm install',
  ),
  _Framework(
    id: 'nuxt',
    label: 'Nuxt',
    icon: LucideIcons.triangle,
    buildCommand: 'npm run build',
    outputDir: '.output/public',
    installCommand: 'npm install',
  ),
  _Framework(
    id: 'astro',
    label: 'Astro',
    icon: LucideIcons.rocket,
    buildCommand: 'npm run build',
    outputDir: 'dist',
    installCommand: 'npm install',
  ),
  _Framework(
    id: 'react',
    label: 'React',
    icon: LucideIcons.atom,
    buildCommand: 'npm run build',
    outputDir: 'build',
    installCommand: 'npm install',
  ),
  _Framework(
    id: 'vue',
    label: 'Vue',
    icon: LucideIcons.layers,
    buildCommand: 'npm run build',
    outputDir: 'dist',
    installCommand: 'npm install',
  ),
  _Framework(
    id: 'flutter_web',
    label: 'Flutter Web',
    icon: LucideIcons.smartphone,
    buildCommand: 'flutter build web --release',
    outputDir: 'build/web',
    installCommand: 'flutter pub get',
  ),
  _Framework(
    id: 'static',
    label: 'Static',
    icon: LucideIcons.fileCode,
    buildCommand: '',
    outputDir: 'public',
    installCommand: '',
  ),
];

_Framework _frameworkById(String id) =>
    _frameworks.firstWhere((f) => f.id == id, orElse: () => _frameworks.last);

// --- Providers ---------------------------------------------------------------

final _siteSearchProvider = StateProvider<String>((ref) => '');
final _sitePerPageProvider = StateProvider<int>((ref) => 12);
final _sitePageProvider = StateProvider<int>((ref) => 1);
final _selectedSiteProvider =
    StateProvider<Map<String, dynamic>?>((ref) => null);
final _listTabProvider = StateProvider<int>((ref) => 0);
final _detailTabProvider = StateProvider<int>((ref) => 0);

final sitesProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final search = ref.watch(_siteSearchProvider);
  final limit = ref.watch(_sitePerPageProvider);
  final page = ref.watch(_sitePageProvider);
  final offset = (page - 1) * limit;
  final params = <String, dynamic>{
    'limit': limit,
    'offset': offset,
    'type': 'web',
  };
  if (search.isNotEmpty) params['search'] = search;
  final res = await api.get('/deploy/targets', params: params);
  return res.data as Map<String, dynamic>;
});

final _siteDetailProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, id) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy/targets/$id');
  return res.data as Map<String, dynamic>;
});

final _siteDomainsProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, id) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy/targets/$id/domains');
  return res.data as Map<String, dynamic>;
});

final _siteReleasesProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, id) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy/releases', params: {'targetId': id});
  return res.data as Map<String, dynamic>;
});

final _siteStatsProvider = FutureProvider.family<Map<String, dynamic>,
    ({String id, String range})>((ref, args) async {
  final api = ref.read(apiClientProvider);
  final res = await api
      .get('/deploy/targets/${args.id}/stats', params: {'range': args.range});
  return res.data as Map<String, dynamic>;
});

final _siteLogsProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, id) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/deploy/targets/$id/logs');
  return res.data as Map<String, dynamic>;
});

// --- Page --------------------------------------------------------------------

class SitesPage extends ConsumerStatefulWidget {
  const SitesPage({super.key});

  @override
  ConsumerState<SitesPage> createState() => _SitesPageState();
}

class _SitesPageState extends ConsumerState<SitesPage> {
  final _searchCtrl = TextEditingController();

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  void _doSearch() {
    ref.read(_siteSearchProvider.notifier).state = _searchCtrl.text.trim();
    ref.read(_sitePageProvider.notifier).state = 1;
  }

  @override
  Widget build(BuildContext context) {
    final selected = ref.watch(_selectedSiteProvider);

    return Scaffold(
      backgroundColor: _bg,
      body: selected != null
          ? _SiteDetailView(
              site: selected,
              onBack: () {
                ref.read(_selectedSiteProvider.notifier).state = null;
                ref.read(_detailTabProvider.notifier).state = 0;
              },
            )
          : _buildSiteList(),
    );
  }

  // ── List view ──────────────────────────────────────────────────────────────

  Widget _buildSiteList() {
    final sitesAsync = ref.watch(sitesProvider);
    final perPage = ref.watch(_sitePerPageProvider);
    final currentPage = ref.watch(_sitePageProvider);
    final listTab = ref.watch(_listTabProvider);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Title + subtitle
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 20, 24, 0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Sites',
                  style: Theme.of(context)
                      .textTheme
                      .headlineSmall
                      ?.copyWith(color: Colors.white)),
              const SizedBox(height: 4),
              const Text(
                'Deploy and manage web applications with automatic builds.',
                style: TextStyle(color: _dimText, fontSize: 13),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),
        // Tabs
        Padding(
          padding: const EdgeInsets.only(left: 24),
          child: PageTabs(
            tabs: const ['Sites', 'Usage'],
            selected: listTab,
            onChanged: (i) =>
                ref.read(_listTabProvider.notifier).state = i,
          ),
        ),
        const SizedBox(height: 16),
        // Tab body
        Expanded(
          child: listTab == 0
              ? _buildSitesTab(sitesAsync, perPage, currentPage)
              : _SiteListUsageTab(),
        ),
      ],
    );
  }

  Widget _buildSitesTab(
    AsyncValue<Map<String, dynamic>> sitesAsync,
    int perPage,
    int currentPage,
  ) {
    return Column(
      children: [
        // Search header
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24),
          child: SearchListHeader(
            searchController: _searchCtrl,
            total: sitesAsync.whenOrNull(
                    data: (d) => d['total'] as int? ?? 0) ??
                0,
            perPage: perPage,
            currentPage: currentPage,
            onPerPageChanged: (v) {
              ref.read(_sitePerPageProvider.notifier).state = v;
              ref.read(_sitePageProvider.notifier).state = 1;
            },
            onPrev: () =>
                ref.read(_sitePageProvider.notifier).update((s) => s - 1),
            onNext: () =>
                ref.read(_sitePageProvider.notifier).update((s) => s + 1),
            onSearch: _doSearch,
            trailing: FilledButton.icon(
              style: FilledButton.styleFrom(
                backgroundColor: _accent,
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
              ),
              onPressed: () => _showCreateSiteDialog(context, ref),
              icon: const Icon(LucideIcons.plus, size: 16),
              label: const Text('Create site'),
            ),
          ),
        ),
        const SizedBox(height: 8),
        Container(
            height: 1,
            margin: const EdgeInsets.symmetric(horizontal: 24),
            color: Colors.white.withOpacity(0.06)),
        const SizedBox(height: 16),
        // Grid
        Expanded(
          child: sitesAsync.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(LucideIcons.alertCircle,
                      size: 48, color: _subtleText),
                  const SizedBox(height: 16),
                  Text('Failed to load sites: $e',
                      style: const TextStyle(color: _dimText)),
                  const SizedBox(height: 8),
                  FilledButton(
                    onPressed: () => ref.invalidate(sitesProvider),
                    child: const Text('Retry'),
                  ),
                ],
              ),
            ),
            data: (data) {
              final sites =
                  List<Map<String, dynamic>>.from(data['targets'] ?? []);
              if (sites.isEmpty) return _buildEmptyState();
              return _SiteGrid(
                sites: sites,
                onSelect: (s) =>
                    ref.read(_selectedSiteProvider.notifier).state = s,
              );
            },
          ),
        ),
        // Footer
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24),
          child: SearchListFooter(
            total: sitesAsync.whenOrNull(
                    data: (d) => d['total'] as int? ?? 0) ??
                0,
            perPage: perPage,
            currentPage: currentPage,
            itemLabel: 'sites',
            onPrev: () =>
                ref.read(_sitePageProvider.notifier).update((s) => s - 1),
            onNext: () =>
                ref.read(_sitePageProvider.notifier).update((s) => s + 1),
            onPerPageChanged: (v) {
              ref.read(_sitePerPageProvider.notifier).state = v;
              ref.read(_sitePageProvider.notifier).state = 1;
            },
          ),
        ),
        const SizedBox(height: 12),
      ],
    );
  }

  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          // Framework icons row
          Row(
            mainAxisSize: MainAxisSize.min,
            children: _frameworks
                .take(6)
                .map((fw) => Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 8),
                      child: Container(
                        width: 48,
                        height: 48,
                        decoration: BoxDecoration(
                          color: _surface,
                          borderRadius: BorderRadius.circular(10),
                          border: Border.all(
                              color: Colors.white.withOpacity(0.06)),
                        ),
                        child: Icon(fw.icon, size: 22, color: _subtleText),
                      ),
                    ))
                .toList(),
          ),
          const SizedBox(height: 24),
          const Text('Create your first site',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 16,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 8),
          const Text(
            'Deploy a web application from a Git repository or upload.',
            style: TextStyle(color: _dimText, fontSize: 13),
          ),
          const SizedBox(height: 20),
          FilledButton.icon(
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            onPressed: () => _showCreateSiteDialog(context, ref),
            icon: const Icon(LucideIcons.plus, size: 16),
            label: const Text('Create site'),
          ),
        ],
      ),
    );
  }

  // ── Create site dialog (3-option entry + multi-step) ────────────────────────

  void _showCreateSiteDialog(BuildContext context, WidgetRef ref) async {
    final result = await showCreateEntryDialog(
      context: context,
      ref: ref,
      category: 'sites',
      title: 'Create Site',
      subtitle: 'Choose how to get started',
    );

    if (result == null || !context.mounted) return;

    String prefillName = '';
    String prefillRepo = '';
    String prefillBranch = 'main';
    String prefillFramework = 'nextjs';
    String entrySourceType = 'git';

    if (result.choice == CreateEntryChoice.template && result.templateConfig != null) {
      prefillName = result.templateConfig!['name'] ?? '';
      prefillFramework = result.templateConfig!['framework'] ?? 'nextjs';
      prefillRepo = result.templateConfig!['repository'] ?? '';
      entrySourceType = 'template';
    } else if (result.choice == CreateEntryChoice.repository && result.repoConfig != null) {
      prefillName = result.repoConfig!['name'] ?? '';
      prefillRepo = result.repoConfig!['cloneUrl'] ?? result.repoConfig!['url'] ?? '';
      prefillBranch = result.repoConfig!['defaultBranch'] ?? 'main';
      entrySourceType = 'git';
    } else {
      entrySourceType = 'upload';
    }

    _showCreateSiteForm(
      context: context,
      ref: ref,
      prefillName: prefillName,
      prefillRepo: prefillRepo,
      prefillBranch: prefillBranch,
      prefillFramework: prefillFramework,
      entrySourceType: entrySourceType,
      templateConfig: result.templateConfig,
      repoConfig: result.repoConfig,
    );
  }

  void _showCreateSiteForm({
    required BuildContext context,
    required WidgetRef ref,
    required String prefillName,
    required String prefillRepo,
    required String prefillBranch,
    required String prefillFramework,
    required String entrySourceType,
    Map<String, dynamic>? templateConfig,
    Map<String, dynamic>? repoConfig,
  }) {
    final nameCtrl = TextEditingController(text: prefillName);
    final repoCtrl = TextEditingController(text: prefillRepo);
    final branchCtrl = TextEditingController(text: prefillBranch);
    final buildCmdCtrl = TextEditingController();
    final outputDirCtrl = TextEditingController();
    final installCmdCtrl = TextEditingController();
    String selectedFramework = prefillFramework;
    String sourceType = entrySourceType == 'upload' ? 'upload' : 'git';
    int step = 0;
    bool creating = false;

    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) {
          final fw = _frameworkById(selectedFramework);
          if (step == 0) {
            // Pre-fill build config from framework when advancing
          }

          Widget stepContent;
          switch (step) {
            case 0:
              stepContent = Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  AppDialogField(
                    controller: nameCtrl,
                    label: 'Site name',
                    hint: 'my-awesome-site',
                    autofocus: true,
                  ),
                  const SizedBox(height: 16),
                  Text('Framework',
                      style: TextStyle(
                          color: Colors.white.withOpacity(0.5),
                          fontSize: 12,
                          fontWeight: FontWeight.w500)),
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: _frameworks
                        .map((fw) => _FrameworkCard(
                              framework: fw,
                              selected: fw.id == selectedFramework,
                              onTap: () => setDialogState(
                                  () => selectedFramework = fw.id),
                            ))
                        .toList(),
                  ),
                ],
              );
              break;
            case 1:
              stepContent = Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Source',
                      style: TextStyle(
                          color: Colors.white.withOpacity(0.5),
                          fontSize: 12,
                          fontWeight: FontWeight.w500)),
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      _SourceTypeChip(
                        label: 'Git repository',
                        icon: LucideIcons.gitBranch,
                        selected: sourceType == 'git',
                        onTap: () =>
                            setDialogState(() => sourceType = 'git'),
                      ),
                      const SizedBox(width: 8),
                      _SourceTypeChip(
                        label: 'Manual upload',
                        icon: LucideIcons.upload,
                        selected: sourceType == 'upload',
                        onTap: () =>
                            setDialogState(() => sourceType = 'upload'),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  if (sourceType == 'git') ...[
                    AppDialogField(
                      controller: repoCtrl,
                      label: 'Repository URL',
                      hint: 'https://github.com/user/repo',
                    ),
                    const SizedBox(height: 12),
                    AppDialogField(
                      controller: branchCtrl,
                      label: 'Branch',
                      hint: 'main',
                    ),
                  ] else ...[
                    Container(
                      width: double.infinity,
                      height: 120,
                      decoration: BoxDecoration(
                        color: _fieldFill,
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(
                            color: _fieldBorder, style: BorderStyle.solid),
                      ),
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Icon(LucideIcons.uploadCloud,
                              size: 32, color: _subtleText),
                          const SizedBox(height: 8),
                          const Text(
                            'Drag & drop your build output\nor click to browse',
                            textAlign: TextAlign.center,
                            style:
                                TextStyle(color: _dimText, fontSize: 12),
                          ),
                        ],
                      ),
                    ),
                  ],
                ],
              );
              break;
            case 2:
              // Auto-fill from framework if empty
              if (buildCmdCtrl.text.isEmpty) {
                buildCmdCtrl.text = fw.buildCommand;
              }
              if (outputDirCtrl.text.isEmpty) {
                outputDirCtrl.text = fw.outputDir;
              }
              if (installCmdCtrl.text.isEmpty) {
                installCmdCtrl.text = fw.installCommand;
              }
              stepContent = Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: _accent.withOpacity(0.08),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: _accent.withOpacity(0.15)),
                    ),
                    child: Row(
                      children: [
                        Icon(LucideIcons.info, size: 14, color: _accent),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            'Auto-detected from ${fw.label}. Edit if needed.',
                            style:
                                TextStyle(color: _accent, fontSize: 12),
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 16),
                  AppDialogField(
                    controller: installCmdCtrl,
                    label: 'Install command',
                    hint: 'npm install',
                  ),
                  const SizedBox(height: 12),
                  AppDialogField(
                    controller: buildCmdCtrl,
                    label: 'Build command',
                    hint: 'npm run build',
                  ),
                  const SizedBox(height: 12),
                  AppDialogField(
                    controller: outputDirCtrl,
                    label: 'Output directory',
                    hint: 'dist',
                  ),
                ],
              );
              break;
            default:
              stepContent = const SizedBox.shrink();
          }

          final stepLabels = ['Configuration', 'Source', 'Build'];

          return Center(
            child: Material(
              color: Colors.transparent,
              child: Container(
                width: 540,
                constraints: const BoxConstraints(maxHeight: 640),
                decoration: BoxDecoration(
                  color: _surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: Colors.white.withOpacity(0.08)),
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
                    // Header
                    Padding(
                      padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
                      child: Row(
                        children: [
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                const Text('Create site',
                                    style: TextStyle(
                                      color: Colors.white,
                                      fontSize: 16,
                                      fontWeight: FontWeight.w600,
                                    )),
                                const SizedBox(height: 4),
                                Text(
                                    'Step ${step + 1} of 3: ${stepLabels[step]}',
                                    style: TextStyle(
                                        color: Colors.white.withOpacity(0.45),
                                        fontSize: 13)),
                              ],
                            ),
                          ),
                          GestureDetector(
                            onTap: () => Navigator.of(ctx).pop(),
                            child: Icon(LucideIcons.x,
                                size: 16,
                                color: Colors.white.withOpacity(0.3)),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 12),
                    // Step indicator
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 20),
                      child: Row(
                        children: List.generate(3, (i) {
                          return Expanded(
                            child: Container(
                              height: 3,
                              margin: EdgeInsets.only(right: i < 2 ? 4 : 0),
                              decoration: BoxDecoration(
                                color: i <= step
                                    ? _accent
                                    : Colors.white.withOpacity(0.08),
                                borderRadius: BorderRadius.circular(2),
                              ),
                            ),
                          );
                        }),
                      ),
                    ),
                    const SizedBox(height: 16),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 20),
                      child: Container(
                          height: 1,
                          color: Colors.white.withOpacity(0.06)),
                    ),
                    const SizedBox(height: 16),
                    // Content
                    Flexible(
                      child: SingleChildScrollView(
                        padding: const EdgeInsets.symmetric(horizontal: 20),
                        child: stepContent,
                      ),
                    ),
                    const SizedBox(height: 16),
                    // Actions
                    Padding(
                      padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.end,
                        children: [
                          if (step > 0)
                            TextButton(
                              onPressed: () =>
                                  setDialogState(() => step--),
                              style: TextButton.styleFrom(
                                foregroundColor: Colors.white54,
                                padding: const EdgeInsets.symmetric(
                                    horizontal: 16, vertical: 10),
                              ),
                              child: const Text('Back',
                                  style: TextStyle(fontSize: 13)),
                            )
                          else
                            const AppDialogCancel(),
                          const SizedBox(width: 8),
                          if (step < 2)
                            AppDialogAction(
                              label: 'Next',
                              onTap: () => setDialogState(() => step++),
                            )
                          else
                            AppDialogAction(
                              label: 'Create',
                              loading: creating,
                              onTap: () async {
                                setDialogState(() => creating = true);
                                try {
                                  final api = ref.read(apiClientProvider);
                                  await api.post('/deploy/targets', data: {
                                    'name': nameCtrl.text,
                                    'type': 'web',
                                    'framework': selectedFramework,
                                    'source': sourceType,
                                    'repository': repoCtrl.text,
                                    'branch': branchCtrl.text,
                                    'buildCommand': buildCmdCtrl.text,
                                    'outputDirectory': outputDirCtrl.text,
                                    'installCommand': installCmdCtrl.text,
                                    if (templateConfig != null) 'templateId': templateConfig['\$id'],
                                  });
                                  if (ctx.mounted) Navigator.pop(ctx);
                                  ref.invalidate(sitesProvider);
                                } catch (e) {
                                  setDialogState(() => creating = false);
                                }
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

// --- Framework card (create dialog) ------------------------------------------

class _FrameworkCard extends StatefulWidget {
  final _Framework framework;
  final bool selected;
  final VoidCallback onTap;

  const _FrameworkCard({
    required this.framework,
    required this.selected,
    required this.onTap,
  });

  @override
  State<_FrameworkCard> createState() => _FrameworkCardState();
}

class _FrameworkCardState extends State<_FrameworkCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          width: 100,
          padding: const EdgeInsets.symmetric(vertical: 12),
          decoration: BoxDecoration(
            color: widget.selected
                ? _accent.withOpacity(0.1)
                : _hovered
                    ? Colors.white.withOpacity(0.03)
                    : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
              color: widget.selected
                  ? _accent.withOpacity(0.4)
                  : Colors.white.withOpacity(0.06),
            ),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(widget.framework.icon,
                  size: 24,
                  color: widget.selected ? _accent : _dimText),
              const SizedBox(height: 6),
              Text(
                widget.framework.label,
                style: TextStyle(
                  fontSize: 11,
                  color: widget.selected ? _accent : _dimText,
                  fontWeight:
                      widget.selected ? FontWeight.w500 : FontWeight.w400,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// --- Source type chip (create dialog) ----------------------------------------

class _SourceTypeChip extends StatefulWidget {
  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onTap;

  const _SourceTypeChip({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onTap,
  });

  @override
  State<_SourceTypeChip> createState() => _SourceTypeChipState();
}

class _SourceTypeChipState extends State<_SourceTypeChip> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          decoration: BoxDecoration(
            color: widget.selected
                ? _accent.withOpacity(0.1)
                : _hovered
                    ? Colors.white.withOpacity(0.03)
                    : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
              color: widget.selected
                  ? _accent.withOpacity(0.4)
                  : Colors.white.withOpacity(0.06),
            ),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(widget.icon,
                  size: 16,
                  color: widget.selected ? _accent : _dimText),
              const SizedBox(width: 8),
              Text(widget.label,
                  style: TextStyle(
                    fontSize: 13,
                    color: widget.selected ? _accent : _dimText,
                  )),
            ],
          ),
        ),
      ),
    );
  }
}

// --- Site grid ---------------------------------------------------------------

class _SiteGrid extends StatelessWidget {
  final List<Map<String, dynamic>> sites;
  final ValueChanged<Map<String, dynamic>> onSelect;

  const _SiteGrid({required this.sites, required this.onSelect});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24),
      child: GridView.builder(
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 2,
          mainAxisSpacing: 12,
          crossAxisSpacing: 12,
          childAspectRatio: 2.2,
        ),
        itemCount: sites.length,
        itemBuilder: (context, index) =>
            _SiteCard(site: sites[index], onTap: () => onSelect(sites[index])),
      ),
    );
  }
}

// --- Site card ----------------------------------------------------------------

class _SiteCard extends StatefulWidget {
  final Map<String, dynamic> site;
  final VoidCallback onTap;

  const _SiteCard({required this.site, required this.onTap});

  @override
  State<_SiteCard> createState() => _SiteCardState();
}

class _SiteCardState extends State<_SiteCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final site = widget.site;
    final name = site['name'] ?? 'Untitled';
    final framework = site['framework'] ?? 'static';
    final fw = _frameworkById(framework);
    final status = site['status'] ?? 'active';
    final domain = site['domain'] ?? '';
    final updatedAt = site['\$updatedAt'] ?? site['updatedAt'] ?? '';

    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: _hovered
                ? Colors.white.withOpacity(0.03)
                : _surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
              color: _hovered
                  ? Colors.white.withOpacity(0.1)
                  : Colors.white.withOpacity(0.06),
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              // Top: name + status
              Row(
                children: [
                  Expanded(
                    child: Text(name,
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 14,
                          fontWeight: FontWeight.w500,
                        ),
                        overflow: TextOverflow.ellipsis),
                  ),
                  _StatusDot(status: status),
                ],
              ),
              const SizedBox(height: 8),
              // Framework badge
              _FrameworkBadge(framework: fw),
              const Spacer(),
              // Bottom: domain + last deployed
              Row(
                children: [
                  if (domain.isNotEmpty) ...[
                    Icon(LucideIcons.globe, size: 12, color: _subtleText),
                    const SizedBox(width: 4),
                    Expanded(
                      child: Text(domain,
                          style: const TextStyle(
                              color: _dimText, fontSize: 11),
                          overflow: TextOverflow.ellipsis),
                    ),
                  ] else
                    const Spacer(),
                  if (updatedAt.isNotEmpty)
                    Text(_timeAgo(updatedAt),
                        style: const TextStyle(
                            color: _subtleText, fontSize: 11)),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// --- Status dot --------------------------------------------------------------

class _StatusDot extends StatelessWidget {
  final String status;
  const _StatusDot({required this.status});

  @override
  Widget build(BuildContext context) {
    Color color;
    String label;
    switch (status) {
      case 'active':
        color = _green;
        label = 'Active';
      case 'building':
        color = _orange;
        label = 'Building';
      case 'failed':
        color = _red;
        label = 'Failed';
      default:
        color = _dimText;
        label = status;
    }
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 7,
          height: 7,
          decoration: BoxDecoration(color: color, shape: BoxShape.circle),
        ),
        const SizedBox(width: 5),
        Text(label, style: TextStyle(fontSize: 11, color: color)),
      ],
    );
  }
}

// --- Framework badge ---------------------------------------------------------

class _FrameworkBadge extends StatelessWidget {
  final _Framework framework;
  const _FrameworkBadge({required this.framework});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: _accent.withOpacity(0.08),
        borderRadius: BorderRadius.circular(5),
        border: Border.all(color: _accent.withOpacity(0.15)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(framework.icon, size: 11, color: _accent),
          const SizedBox(width: 4),
          Text(framework.label,
              style: const TextStyle(fontSize: 11, color: _accent)),
        ],
      ),
    );
  }
}

// --- List-level usage tab ----------------------------------------------------

class _SiteListUsageTab extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Aggregate usage across all sites in this project.',
            style: TextStyle(color: _dimText, fontSize: 13),
          ),
          const SizedBox(height: 20),
          Row(
            children: [
              _StatCard(
                icon: LucideIcons.activity,
                label: 'Total requests',
                value: '--',
              ),
              const SizedBox(width: 12),
              _StatCard(
                icon: LucideIcons.hardDrive,
                label: 'Bandwidth',
                value: '--',
              ),
              const SizedBox(width: 12),
              _StatCard(
                icon: LucideIcons.timer,
                label: 'Build minutes',
                value: '--',
              ),
            ],
          ),
        ],
      ),
    );
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// SITE DETAIL VIEW
// ═══════════════════════════════════════════════════════════════════════════════

class _SiteDetailView extends ConsumerWidget {
  final Map<String, dynamic> site;
  final VoidCallback onBack;

  const _SiteDetailView({required this.site, required this.onBack});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final tab = ref.watch(_detailTabProvider);
    final siteId = site['\$id'] as String? ?? site['id'] as String? ?? '';
    final name = site['name'] ?? 'Untitled';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Header: back + name + ID badge
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
                      const Icon(LucideIcons.arrowLeft,
                          size: 16, color: _dimText),
                      const SizedBox(width: 8),
                      const Text('Sites',
                          style: TextStyle(color: _dimText, fontSize: 14)),
                    ],
                  ),
                ),
              ),
              const SizedBox(width: 12),
              const Icon(LucideIcons.chevronRight,
                  size: 14, color: _subtleText),
              const SizedBox(width: 12),
              Expanded(
                child: Text(name,
                    style: Theme.of(context)
                        .textTheme
                        .headlineSmall
                        ?.copyWith(color: Colors.white)),
              ),
              if (siteId.isNotEmpty)
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: _fieldFill,
                    borderRadius: BorderRadius.circular(5),
                    border: Border.all(color: Colors.white.withOpacity(0.06)),
                  ),
                  child: Text(siteId,
                      style: const TextStyle(
                          fontFamily: 'monospace',
                          fontSize: 11,
                          color: _dimText)),
                ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        // Tabs
        Padding(
          padding: const EdgeInsets.only(left: 24),
          child: PageTabs(
            tabs: const [
              'Overview',
              'Deployments',
              'Logs',
              'Domains',
              'Usage',
              'Settings'
            ],
            selected: tab,
            onChanged: (i) =>
                ref.read(_detailTabProvider.notifier).state = i,
          ),
        ),
        // Tab body
        Expanded(
          child: _detailBody(context, ref, tab, siteId),
        ),
      ],
    );
  }

  Widget _detailBody(
      BuildContext context, WidgetRef ref, int tab, String siteId) {
    switch (tab) {
      case 0:
        return _OverviewTab(site: site, siteId: siteId);
      case 1:
        return _DeploymentsTab(siteId: siteId);
      case 2:
        return _LogsTab(siteId: siteId);
      case 3:
        return _DomainsTab(siteId: siteId);
      case 4:
        return _UsageTab(siteId: siteId);
      case 5:
        return _SettingsTab(site: site, siteId: siteId);
      default:
        return const SizedBox.shrink();
    }
  }
}

// ── Overview tab ─────────────────────────────────────────────────────────────

class _OverviewTab extends ConsumerWidget {
  final Map<String, dynamic> site;
  final String siteId;

  const _OverviewTab({required this.site, required this.siteId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final domainsAsync = ref.watch(_siteDomainsProvider(siteId));
    final releasesAsync = ref.watch(_siteReleasesProvider(siteId));
    final framework = _frameworkById(site['framework'] ?? 'static');
    final source = site['source'] ?? 'git';
    final repository = site['repository'] ?? '';
    final updatedAt = site['\$updatedAt'] ?? site['updatedAt'] ?? '';

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Top section: preview + info
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Preview placeholder
              Container(
                width: 340,
                height: 200,
                decoration: BoxDecoration(
                  color: _surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: Colors.white.withOpacity(0.06)),
                ),
                child: const Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(LucideIcons.globe, size: 40, color: _subtleText),
                      SizedBox(height: 8),
                      Text('Site preview',
                          style: TextStyle(color: _subtleText, fontSize: 12)),
                    ],
                  ),
                ),
              ),
              const SizedBox(width: 24),
              // Info column
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Domains
                    _infoRow(LucideIcons.globe, 'Domain',
                        site['domain'] ?? 'No domain assigned'),
                    const SizedBox(height: 12),
                    // Deployed
                    _infoRow(LucideIcons.clock, 'Last deployed',
                        updatedAt.isNotEmpty ? _timeAgo(updatedAt) : 'Never'),
                    const SizedBox(height: 12),
                    // Source
                    _infoRow(
                      source == 'git'
                          ? LucideIcons.gitBranch
                          : LucideIcons.upload,
                      'Source',
                      repository.isNotEmpty
                          ? repository
                          : source == 'git'
                              ? 'Git repository'
                              : 'Manual upload',
                    ),
                    const SizedBox(height: 12),
                    // Framework
                    _infoRow(framework.icon, 'Framework', framework.label),
                    const SizedBox(height: 12),
                    // Build duration
                    _infoRow(LucideIcons.timer, 'Build duration',
                        _formatDuration(site['buildDuration'])),
                    const SizedBox(height: 12),
                    // Total size
                    _infoRow(LucideIcons.hardDrive, 'Total size',
                        _formatBytes(site['totalSize'])),
                    const SizedBox(height: 20),
                    // Action buttons
                    Row(
                      children: [
                        FilledButton.icon(
                          style: FilledButton.styleFrom(
                            backgroundColor: _accent,
                            shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(8)),
                            padding: const EdgeInsets.symmetric(
                                horizontal: 16, vertical: 10),
                          ),
                          onPressed: () {},
                          icon: const Icon(LucideIcons.externalLink, size: 14),
                          label: const Text('Visit',
                              style: TextStyle(fontSize: 13)),
                        ),
                        const SizedBox(width: 8),
                        OutlinedButton.icon(
                          style: OutlinedButton.styleFrom(
                            foregroundColor: Colors.white70,
                            side: BorderSide(
                                color: Colors.white.withOpacity(0.12)),
                            shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(8)),
                            padding: const EdgeInsets.symmetric(
                                horizontal: 16, vertical: 10),
                          ),
                          onPressed: releasesAsync.hasValue
                              ? () async {
                                  final list =
                                      List<Map<String, dynamic>>.from(
                                          releasesAsync.value!['releases'] ??
                                              []);
                                  final successes = list
                                      .where((r) => r['status'] == 'success')
                                      .toList();
                                  if (successes.isEmpty) {
                                    if (!context.mounted) return;
                                    ScaffoldMessenger.of(context).showSnackBar(
                                      const SnackBar(
                                          content: Text(
                                              'No successful deployment to roll back to')),
                                    );
                                    return;
                                  }
                                  final target = successes.first;
                                  final releaseId =
                                      (target['\$id'] ?? target['id'])
                                          as String;
                                  if (!context.mounted) return;
                                  final confirmed = await showDialog<bool>(
                                    context: context,
                                    builder: (ctx) => AlertDialog(
                                      backgroundColor: _surface,
                                      title: const Text('Instant rollback',
                                          style:
                                              TextStyle(color: Colors.white)),
                                      content: Text(
                                          'Roll back to deployment ${releaseId.length > 8 ? releaseId.substring(0, 8) : releaseId}…?',
                                          style: const TextStyle(
                                              color: _dimText)),
                                      actions: [
                                        TextButton(
                                          onPressed: () =>
                                              Navigator.pop(ctx, false),
                                          child: const Text('Cancel'),
                                        ),
                                        FilledButton(
                                          style: FilledButton.styleFrom(
                                              backgroundColor: _red),
                                          onPressed: () =>
                                              Navigator.pop(ctx, true),
                                          child: const Text('Rollback'),
                                        ),
                                      ],
                                    ),
                                  );
                                  if (confirmed != true || !context.mounted) {
                                    return;
                                  }
                                  try {
                                    final api = ref.read(apiClientProvider);
                                    await api.post(
                                        '/deploy/releases/$releaseId/rollback',
                                        data: {});
                                    ref.invalidate(
                                        _siteReleasesProvider(siteId));
                                    if (context.mounted) {
                                      ScaffoldMessenger.of(context)
                                          .showSnackBar(
                                        const SnackBar(
                                            content:
                                                Text('Rollback initiated')),
                                      );
                                    }
                                  } catch (e) {
                                    if (context.mounted) {
                                      ScaffoldMessenger.of(context)
                                          .showSnackBar(
                                        SnackBar(
                                            content: Text(
                                                'Rollback failed: $e')),
                                      );
                                    }
                                  }
                                }
                              : null,
                          icon: const Icon(LucideIcons.rotateCcw, size: 14),
                          label: const Text('Instant rollback',
                              style: TextStyle(fontSize: 13)),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 24),
          // Summary cards
          Row(
            children: [
              Expanded(
                child: _SummaryCard(
                  icon: LucideIcons.globe,
                  label: 'Domains',
                  value: domainsAsync.whenOrNull(
                          data: (d) =>
                              '${(d['domains'] as List?)?.length ?? 0}') ??
                      '--',
                  sublabel: 'View all domains',
                  onTap: () =>
                      ref.read(_detailTabProvider.notifier).state = 3,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: _SummaryCard(
                  icon: LucideIcons.rocket,
                  label: 'Deployments',
                  value: releasesAsync.whenOrNull(
                          data: (d) => '${d['total'] ?? 0}') ??
                      '--',
                  sublabel: 'View all deployments',
                  onTap: () =>
                      ref.read(_detailTabProvider.notifier).state = 1,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _infoRow(IconData icon, String label, String value) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icon, size: 14, color: _subtleText),
        const SizedBox(width: 8),
        SizedBox(
          width: 110,
          child: Text(label,
              style: const TextStyle(color: _dimText, fontSize: 13)),
        ),
        Expanded(
          child: Text(value,
              style: const TextStyle(color: Colors.white, fontSize: 13),
              overflow: TextOverflow.ellipsis),
        ),
      ],
    );
  }
}

// ── Summary card (overview) ─────────────────────────────────────────────────

class _SummaryCard extends StatefulWidget {
  final IconData icon;
  final String label;
  final String value;
  final String sublabel;
  final VoidCallback onTap;

  const _SummaryCard({
    required this.icon,
    required this.label,
    required this.value,
    required this.sublabel,
    required this.onTap,
  });

  @override
  State<_SummaryCard> createState() => _SummaryCardState();
}

class _SummaryCardState extends State<_SummaryCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: _hovered ? Colors.white.withOpacity(0.03) : _surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: Colors.white.withOpacity(0.06)),
          ),
          child: Row(
            children: [
              Icon(widget.icon, size: 20, color: _accent),
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(widget.label,
                      style: const TextStyle(
                          color: _dimText, fontSize: 12)),
                  const SizedBox(height: 2),
                  Text(widget.value,
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 20,
                          fontWeight: FontWeight.w600)),
                ],
              ),
              const Spacer(),
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(widget.sublabel,
                      style: const TextStyle(color: _accent, fontSize: 12)),
                  const SizedBox(width: 4),
                  const Icon(LucideIcons.arrowRight,
                      size: 12, color: _accent),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Deployments tab ─────────────────────────────────────────────────────────

class _DeploymentsTab extends ConsumerWidget {
  final String siteId;
  const _DeploymentsTab({required this.siteId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final releasesAsync = ref.watch(_siteReleasesProvider(siteId));

    return Column(
      children: [
        // Metrics row
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 12),
          child: releasesAsync.when(
            loading: () => const SizedBox.shrink(),
            error: (_, __) => const SizedBox.shrink(),
            data: (data) {
              final releases = List<Map<String, dynamic>>.from(
                  data['releases'] ?? []);
              final total = releases.length;
              final successful =
                  releases.where((r) => r['status'] == 'ready' || r['status'] == 'active').length;
              final failed =
                  releases.where((r) => r['status'] == 'failed').length;
              final totalDuration = releases.fold<int>(
                  0,
                  (sum, r) =>
                      sum + ((r['buildDuration'] as num?)?.toInt() ?? 0));
              final avgDuration =
                  total > 0 ? (totalDuration / total).round() : 0;
              final totalSize = releases.fold<int>(
                  0,
                  (sum, r) =>
                      sum + ((r['totalSize'] as num?)?.toInt() ?? 0));

              return Row(
                children: [
                  _MetricBadge(
                      label: 'Total builds', value: '$total'),
                  const SizedBox(width: 12),
                  _MetricBadge(
                      label: 'Total size', value: _formatBytes(totalSize)),
                  const SizedBox(width: 12),
                  _MetricBadge(
                      label: 'Total time',
                      value: _formatDuration(totalDuration)),
                  const SizedBox(width: 12),
                  _MetricBadge(
                      label: 'Avg time',
                      value: _formatDuration(avgDuration)),
                  const SizedBox(width: 12),
                  _MetricBadge(
                      label: 'Successful',
                      value: '$successful',
                      color: _green),
                  const SizedBox(width: 12),
                  _MetricBadge(
                      label: 'Failed', value: '$failed', color: _red),
                ],
              );
            },
          ),
        ),
        Container(height: 1, color: Colors.white.withOpacity(0.06)),
        // Header
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 12, 24, 8),
          child: Row(
            children: [
              const Spacer(),
              FilledButton.icon(
                style: FilledButton.styleFrom(
                  backgroundColor: _accent,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 12, vertical: 8),
                ),
                onPressed: () async {
                  final api = ref.read(apiClientProvider);
                  await api
                      .post('/deploy/pipelines/$siteId/trigger', data: {});
                  ref.invalidate(_siteReleasesProvider(siteId));
                },
                icon: const Icon(LucideIcons.play, size: 14),
                label: const Text('Create deployment',
                    style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
        ),
        // Table
        Expanded(
          child: releasesAsync.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(LucideIcons.alertCircle,
                      size: 48, color: _subtleText),
                  const SizedBox(height: 16),
                  Text('Failed to load deployments: $e',
                      style: const TextStyle(color: _dimText)),
                ],
              ),
            ),
            data: (data) {
              final releases = List<Map<String, dynamic>>.from(
                  data['releases'] ?? []);
              if (releases.isEmpty) {
                return const Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(LucideIcons.rocket, size: 48, color: _subtleText),
                      SizedBox(height: 16),
                      Text('No deployments yet',
                          style: TextStyle(color: _dimText)),
                    ],
                  ),
                );
              }
              return SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: SingleChildScrollView(
                  child: DataTable(
                    columnSpacing: 24,
                    headingRowColor: WidgetStateProperty.all(_surface),
                    dataRowColor: WidgetStateProperty.all(_bg),
                    columns: const [
                      DataColumn(
                          label: Text('Deployment ID',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Status',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Build duration',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Total size',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Source',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Updated',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                    ],
                    rows: releases.map((r) {
                      final id = r['\$id'] ?? r['id'] ?? '';
                      final status = r['status'] ?? 'pending';
                      final shortId = id.length > 8
                          ? '${id.substring(0, 8)}...'
                          : id;
                      return DataRow(cells: [
                        DataCell(Text(shortId,
                            style: const TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 12,
                                color: _accent))),
                        DataCell(_DeployStatusChip(status: status)),
                        DataCell(Text(
                            _formatDuration(r['buildDuration']),
                            style: const TextStyle(
                                color: Colors.white, fontSize: 13))),
                        DataCell(Text(_formatBytes(r['totalSize']),
                            style: const TextStyle(
                                color: Colors.white, fontSize: 13))),
                        DataCell(Text(r['source'] ?? 'git',
                            style: const TextStyle(
                                color: _dimText, fontSize: 13))),
                        DataCell(Text(
                            _formatTimestamp(
                                r['\$updatedAt'] ?? r['updatedAt']),
                            style: const TextStyle(
                                color: _dimText, fontSize: 13))),
                      ]);
                    }).toList(),
                  ),
                ),
              );
            },
          ),
        ),
      ],
    );
  }
}

// ── Deploy status chip ──────────────────────────────────────────────────────

class _DeployStatusChip extends StatelessWidget {
  final String status;
  const _DeployStatusChip({required this.status});

  @override
  Widget build(BuildContext context) {
    Color color;
    switch (status) {
      case 'active':
        color = _green;
      case 'ready':
        color = _green;
      case 'building':
        color = _orange;
      case 'failed':
        color = _red;
      default:
        color = _dimText;
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(5),
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
          const SizedBox(width: 5),
          Text(
            status[0].toUpperCase() + status.substring(1),
            style: TextStyle(fontSize: 12, color: color),
          ),
        ],
      ),
    );
  }
}

// ── Metric badge (deployments tab) ──────────────────────────────────────────

class _MetricBadge extends StatelessWidget {
  final String label;
  final String value;
  final Color? color;

  const _MetricBadge({
    required this.label,
    required this.value,
    this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: _surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.white.withOpacity(0.06)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label,
              style: const TextStyle(color: _subtleText, fontSize: 10)),
          const SizedBox(height: 2),
          Text(value,
              style: TextStyle(
                  color: color ?? Colors.white,
                  fontSize: 14,
                  fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }
}

// ── Logs tab ────────────────────────────────────────────────────────────────

class _LogsTab extends ConsumerWidget {
  final String siteId;
  const _LogsTab({required this.siteId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final logsAsync = ref.watch(_siteLogsProvider(siteId));

    return Column(
      children: [
        // Header
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 8),
          child: Row(
            children: [
              const Icon(LucideIcons.scrollText, size: 16, color: _dimText),
              const SizedBox(width: 8),
              const Text('Access Logs',
                  style: TextStyle(
                      color: Colors.white,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
              const Spacer(),
              OutlinedButton.icon(
                style: OutlinedButton.styleFrom(
                  foregroundColor: Colors.white70,
                  side: BorderSide(color: Colors.white.withOpacity(0.12)),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 12, vertical: 8),
                ),
                onPressed: () => ref.invalidate(_siteLogsProvider(siteId)),
                icon: const Icon(LucideIcons.refreshCw, size: 14),
                label:
                    const Text('Refresh', style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
        ),
        Container(height: 1, color: Colors.white.withOpacity(0.06)),
        Expanded(
          child: logsAsync.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(LucideIcons.alertCircle,
                      size: 48, color: _subtleText),
                  const SizedBox(height: 16),
                  Text('Failed to load logs: $e',
                      style: const TextStyle(color: _dimText)),
                ],
              ),
            ),
            data: (data) {
              final logs =
                  List<Map<String, dynamic>>.from(data['logs'] ?? []);
              if (logs.isEmpty) {
                return const Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(LucideIcons.scrollText,
                          size: 48, color: _subtleText),
                      SizedBox(height: 16),
                      Text('No logs yet',
                          style: TextStyle(color: _dimText)),
                    ],
                  ),
                );
              }
              return SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: SingleChildScrollView(
                  child: DataTable(
                    columnSpacing: 24,
                    headingRowColor: WidgetStateProperty.all(_surface),
                    dataRowColor: WidgetStateProperty.all(_bg),
                    columns: const [
                      DataColumn(
                          label: Text('Log ID',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Path',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Method',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Status',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Duration',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Created',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                    ],
                    rows: logs.map((log) {
                      final id = log['\$id'] ?? log['id'] ?? '';
                      final shortId = id.length > 8
                          ? '${id.substring(0, 8)}...'
                          : id;
                      final statusCode =
                          (log['statusCode'] as num?)?.toInt() ?? 0;
                      return DataRow(cells: [
                        DataCell(Text(shortId,
                            style: const TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 12,
                                color: _dimText))),
                        DataCell(Text(log['path'] ?? '/',
                            style: const TextStyle(
                                color: Colors.white, fontSize: 13))),
                        DataCell(_MethodBadge(
                            method: log['method'] ?? 'GET')),
                        DataCell(_HttpStatusBadge(code: statusCode)),
                        DataCell(Text(
                            '${log['duration'] ?? 0}ms',
                            style: const TextStyle(
                                color: _dimText, fontSize: 13))),
                        DataCell(Text(
                            _formatTimestamp(
                                log['\$createdAt'] ?? log['createdAt']),
                            style: const TextStyle(
                                color: _dimText, fontSize: 13))),
                      ]);
                    }).toList(),
                  ),
                ),
              );
            },
          ),
        ),
      ],
    );
  }
}

// ── Method badge ────────────────────────────────────────────────────────────

class _MethodBadge extends StatelessWidget {
  final String method;
  const _MethodBadge({required this.method});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: _fieldFill,
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(method,
          style: const TextStyle(
              fontFamily: 'monospace',
              fontSize: 11,
              fontWeight: FontWeight.w600,
              color: _dimText)),
    );
  }
}

// ── HTTP status badge ───────────────────────────────────────────────────────

class _HttpStatusBadge extends StatelessWidget {
  final int code;
  const _HttpStatusBadge({required this.code});

  @override
  Widget build(BuildContext context) {
    Color color;
    if (code >= 200 && code < 300) {
      color = _green;
    } else if (code >= 400 && code < 500) {
      color = _orange;
    } else if (code >= 500) {
      color = _red;
    } else {
      color = _dimText;
    }
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withOpacity(0.25)),
      ),
      child: Text('$code',
          style: TextStyle(
              fontFamily: 'monospace',
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: color)),
    );
  }
}

// ── Domains tab ─────────────────────────────────────────────────────────────

class _DomainsTab extends ConsumerWidget {
  final String siteId;
  const _DomainsTab({required this.siteId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final domainsAsync = ref.watch(_siteDomainsProvider(siteId));

    return Column(
      children: [
        // Header
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 16, 24, 8),
          child: Row(
            children: [
              const Icon(LucideIcons.globe, size: 16, color: _dimText),
              const SizedBox(width: 8),
              const Text('Domains',
                  style: TextStyle(
                      color: Colors.white,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
              const Spacer(),
              FilledButton.icon(
                style: FilledButton.styleFrom(
                  backgroundColor: _accent,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 12, vertical: 8),
                ),
                onPressed: () =>
                    _showAddDomainDialog(context, ref, siteId),
                icon: const Icon(LucideIcons.plus, size: 14),
                label: const Text('Add domain',
                    style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
        ),
        Container(height: 1, color: Colors.white.withOpacity(0.06)),
        Expanded(
          child: domainsAsync.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(LucideIcons.alertCircle,
                      size: 48, color: _subtleText),
                  const SizedBox(height: 16),
                  Text('Failed to load domains: $e',
                      style: const TextStyle(color: _dimText)),
                ],
              ),
            ),
            data: (data) {
              final domains =
                  List<Map<String, dynamic>>.from(data['domains'] ?? []);
              if (domains.isEmpty) {
                return Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(LucideIcons.globe,
                          size: 48, color: _subtleText),
                      const SizedBox(height: 16),
                      const Text('No custom domains',
                          style: TextStyle(color: _dimText)),
                      const SizedBox(height: 12),
                      FilledButton.icon(
                        style: FilledButton.styleFrom(
                          backgroundColor: _accent,
                          shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(8)),
                        ),
                        onPressed: () =>
                            _showAddDomainDialog(context, ref, siteId),
                        icon: const Icon(LucideIcons.plus, size: 16),
                        label: const Text('Add domain'),
                      ),
                    ],
                  ),
                );
              }
              return SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: SingleChildScrollView(
                  child: DataTable(
                    columnSpacing: 24,
                    headingRowColor: WidgetStateProperty.all(_surface),
                    dataRowColor: WidgetStateProperty.all(_bg),
                    columns: const [
                      DataColumn(
                          label: Text('Domain',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                      DataColumn(
                          label: Text('Target',
                              style: TextStyle(
                                  fontWeight: FontWeight.bold,
                                  color: _dimText))),
                    ],
                    rows: domains.map((d) {
                      final target = d['targetType'] ?? 'active_deployment';
                      String targetLabel;
                      IconData targetIcon;
                      switch (target) {
                        case 'active_deployment':
                          targetLabel = 'Active deployment';
                          targetIcon = LucideIcons.rocket;
                        case 'git_branch':
                          targetLabel =
                              'Branch: ${d['targetValue'] ?? 'main'}';
                          targetIcon = LucideIcons.gitBranch;
                        case 'redirect':
                          targetLabel =
                              'Redirect: ${d['targetValue'] ?? ''}';
                          targetIcon = LucideIcons.externalLink;
                        default:
                          targetLabel = target;
                          targetIcon = LucideIcons.link;
                      }
                      return DataRow(cells: [
                        DataCell(Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            const Icon(LucideIcons.globe,
                                size: 14, color: _accent),
                            const SizedBox(width: 8),
                            Text(d['domain'] ?? '',
                                style: const TextStyle(
                                    color: Colors.white,
                                    fontSize: 13)),
                          ],
                        )),
                        DataCell(Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Icon(targetIcon,
                                size: 14, color: _subtleText),
                            const SizedBox(width: 8),
                            Text(targetLabel,
                                style: const TextStyle(
                                    color: _dimText, fontSize: 13)),
                          ],
                        )),
                      ]);
                    }).toList(),
                  ),
                ),
              );
            },
          ),
        ),
      ],
    );
  }

  void _showAddDomainDialog(
      BuildContext context, WidgetRef ref, String siteId) {
    final domainCtrl = TextEditingController();
    final targetValueCtrl = TextEditingController();
    String targetType = 'active_deployment';
    bool adding = false;

    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.6),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) => Center(
          child: Material(
            color: Colors.transparent,
            child: Container(
              width: 440,
              constraints: const BoxConstraints(maxHeight: 500),
              decoration: BoxDecoration(
                color: _surface,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.white.withOpacity(0.08)),
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
                  // Header
                  Padding(
                    padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
                    child: Row(
                      children: [
                        const Expanded(
                          child: Text('Add domain',
                              style: TextStyle(
                                color: Colors.white,
                                fontSize: 16,
                                fontWeight: FontWeight.w600,
                              )),
                        ),
                        GestureDetector(
                          onTap: () => Navigator.of(ctx).pop(),
                          child: Icon(LucideIcons.x,
                              size: 16,
                              color: Colors.white.withOpacity(0.3)),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 16),
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 20),
                    child: Container(
                        height: 1,
                        color: Colors.white.withOpacity(0.06)),
                  ),
                  const SizedBox(height: 16),
                  Flexible(
                    child: SingleChildScrollView(
                      padding: const EdgeInsets.symmetric(horizontal: 20),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          AppDialogField(
                            controller: domainCtrl,
                            label: 'Domain',
                            hint: 'example.com',
                            autofocus: true,
                          ),
                          const SizedBox(height: 16),
                          Text('Target type',
                              style: TextStyle(
                                  color: Colors.white.withOpacity(0.5),
                                  fontSize: 12,
                                  fontWeight: FontWeight.w500)),
                          const SizedBox(height: 8),
                          Wrap(
                            spacing: 8,
                            runSpacing: 8,
                            children: [
                              _TargetTypeChip(
                                label: 'Active deployment',
                                icon: LucideIcons.rocket,
                                selected:
                                    targetType == 'active_deployment',
                                onTap: () => setDialogState(() =>
                                    targetType = 'active_deployment'),
                              ),
                              _TargetTypeChip(
                                label: 'Git branch',
                                icon: LucideIcons.gitBranch,
                                selected: targetType == 'git_branch',
                                onTap: () => setDialogState(
                                    () => targetType = 'git_branch'),
                              ),
                              _TargetTypeChip(
                                label: 'Redirect',
                                icon: LucideIcons.externalLink,
                                selected: targetType == 'redirect',
                                onTap: () => setDialogState(
                                    () => targetType = 'redirect'),
                              ),
                            ],
                          ),
                          if (targetType != 'active_deployment') ...[
                            const SizedBox(height: 12),
                            AppDialogField(
                              controller: targetValueCtrl,
                              label: targetType == 'git_branch'
                                  ? 'Branch name'
                                  : 'Redirect URL',
                              hint: targetType == 'git_branch'
                                  ? 'main'
                                  : 'https://example.com',
                            ),
                          ],
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
                          label: 'Add',
                          loading: adding,
                          onTap: () async {
                            setDialogState(() => adding = true);
                            try {
                              final api = ref.read(apiClientProvider);
                              await api.post(
                                  '/deploy/targets/$siteId/domains',
                                  data: {
                                    'domain': domainCtrl.text,
                                    'targetType': targetType,
                                    'targetValue': targetValueCtrl.text,
                                  });
                              if (ctx.mounted) Navigator.pop(ctx);
                              ref.invalidate(
                                  _siteDomainsProvider(siteId));
                            } catch (e) {
                              setDialogState(() => adding = false);
                            }
                          },
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// ── Target type chip (add domain dialog) ────────────────────────────────────

class _TargetTypeChip extends StatefulWidget {
  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onTap;

  const _TargetTypeChip({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onTap,
  });

  @override
  State<_TargetTypeChip> createState() => _TargetTypeChipState();
}

class _TargetTypeChipState extends State<_TargetTypeChip> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: widget.selected
                ? _accent.withOpacity(0.1)
                : _hovered
                    ? Colors.white.withOpacity(0.03)
                    : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
              color: widget.selected
                  ? _accent.withOpacity(0.4)
                  : Colors.white.withOpacity(0.06),
            ),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(widget.icon,
                  size: 14,
                  color: widget.selected ? _accent : _dimText),
              const SizedBox(width: 6),
              Text(widget.label,
                  style: TextStyle(
                    fontSize: 12,
                    color: widget.selected ? _accent : _dimText,
                  )),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Usage tab ───────────────────────────────────────────────────────────────

class _UsageTab extends ConsumerStatefulWidget {
  final String siteId;
  const _UsageTab({required this.siteId});

  @override
  ConsumerState<_UsageTab> createState() => _UsageTabState();
}

class _UsageTabState extends ConsumerState<_UsageTab> {
  String _range = '30d';

  @override
  Widget build(BuildContext context) {
    final statsAsync = ref.watch(
        _siteStatsProvider((id: widget.siteId, range: _range)));

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Time range selector
          Row(
            children: [
              const Text('Usage',
                  style: TextStyle(
                      color: Colors.white,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
              const Spacer(),
              _TimeRangeSelector(
                value: _range,
                onChanged: (v) => setState(() => _range = v),
              ),
            ],
          ),
          const SizedBox(height: 20),
          // Stat cards
          statsAsync.when(
            loading: () => Row(
              children: [
                Expanded(
                    child: _StatCard(
                        icon: LucideIcons.activity,
                        label: 'Requests',
                        value: '--')),
                const SizedBox(width: 12),
                Expanded(
                    child: _StatCard(
                        icon: LucideIcons.hardDrive,
                        label: 'Bandwidth',
                        value: '--')),
                const SizedBox(width: 12),
                Expanded(
                    child: _StatCard(
                        icon: LucideIcons.timer,
                        label: 'Build minutes',
                        value: '--')),
              ],
            ),
            error: (e, _) => Center(
              child: Text('Failed to load stats: $e',
                  style: const TextStyle(color: _dimText)),
            ),
            data: (data) {
              return Row(
                children: [
                  Expanded(
                      child: _StatCard(
                          icon: LucideIcons.activity,
                          label: 'Requests',
                          value: _formatNumber(data['requests']))),
                  const SizedBox(width: 12),
                  Expanded(
                      child: _StatCard(
                          icon: LucideIcons.hardDrive,
                          label: 'Bandwidth',
                          value: _formatBytes(data['bandwidth']))),
                  const SizedBox(width: 12),
                  Expanded(
                      child: _StatCard(
                          icon: LucideIcons.timer,
                          label: 'Build minutes',
                          value: _formatDuration(data['buildMinutes']))),
                ],
              );
            },
          ),
        ],
      ),
    );
  }
}

// ── Time range selector ─────────────────────────────────────────────────────

class _TimeRangeSelector extends StatelessWidget {
  final String value;
  final ValueChanged<String> onChanged;

  const _TimeRangeSelector({required this.value, required this.onChanged});

  @override
  Widget build(BuildContext context) {
    const ranges = ['24h', '7d', '30d'];
    return Container(
      decoration: BoxDecoration(
        color: _surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.white.withOpacity(0.06)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: ranges.map((r) {
          final selected = r == value;
          return GestureDetector(
            onTap: () => onChanged(r),
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                decoration: BoxDecoration(
                  color:
                      selected ? _accent.withOpacity(0.15) : Colors.transparent,
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Text(r,
                    style: TextStyle(
                      fontSize: 12,
                      color: selected ? _accent : _dimText,
                      fontWeight:
                          selected ? FontWeight.w500 : FontWeight.w400,
                    )),
              ),
            ),
          );
        }).toList(),
      ),
    );
  }
}

// ── Stat card ───────────────────────────────────────────────────────────────

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
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: _surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.white.withOpacity(0.06)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18, color: _accent),
          const SizedBox(height: 10),
          Text(label,
              style: const TextStyle(color: _dimText, fontSize: 12)),
          const SizedBox(height: 4),
          Text(value,
              style: const TextStyle(
                  color: Colors.white,
                  fontSize: 22,
                  fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }
}

// ── Settings tab ────────────────────────────────────────────────────────────

class _SettingsTab extends ConsumerStatefulWidget {
  final Map<String, dynamic> site;
  final String siteId;

  const _SettingsTab({required this.site, required this.siteId});

  @override
  ConsumerState<_SettingsTab> createState() => _SettingsTabState();
}

class _SettingsTabState extends ConsumerState<_SettingsTab> {
  late final TextEditingController _nameCtrl;
  late final TextEditingController _repoCtrl;
  late final TextEditingController _buildCmdCtrl;
  late final TextEditingController _outputDirCtrl;
  late final TextEditingController _installCmdCtrl;
  final List<MapEntry<TextEditingController, TextEditingController>> _envVars =
      [];
  bool _saving = false;
  bool _deleting = false;

  @override
  void initState() {
    super.initState();
    _nameCtrl = TextEditingController(text: widget.site['name'] ?? '');
    _repoCtrl = TextEditingController(text: widget.site['repository'] ?? '');
    _buildCmdCtrl =
        TextEditingController(text: widget.site['buildCommand'] ?? '');
    _outputDirCtrl =
        TextEditingController(text: widget.site['outputDirectory'] ?? '');
    _installCmdCtrl =
        TextEditingController(text: widget.site['installCommand'] ?? '');
    // Load existing env vars
    final envMap =
        widget.site['environmentVariables'] as Map<String, dynamic>? ?? {};
    for (final e in envMap.entries) {
      _envVars.add(MapEntry(
        TextEditingController(text: e.key),
        TextEditingController(text: '${e.value}'),
      ));
    }
  }

  @override
  void dispose() {
    _nameCtrl.dispose();
    _repoCtrl.dispose();
    _buildCmdCtrl.dispose();
    _outputDirCtrl.dispose();
    _installCmdCtrl.dispose();
    for (final pair in _envVars) {
      pair.key.dispose();
      pair.value.dispose();
    }
    super.dispose();
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final api = ref.read(apiClientProvider);
      final envMap = <String, String>{};
      for (final pair in _envVars) {
        if (pair.key.text.isNotEmpty) {
          envMap[pair.key.text] = pair.value.text;
        }
      }
      await api.put('/deploy/targets/${widget.siteId}', data: {
        'name': _nameCtrl.text,
        'repository': _repoCtrl.text,
        'buildCommand': _buildCmdCtrl.text,
        'outputDirectory': _outputDirCtrl.text,
        'installCommand': _installCmdCtrl.text,
        'environmentVariables': envMap,
      });
      ref.invalidate(sitesProvider);
      ref.invalidate(_siteDetailProvider(widget.siteId));
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _delete() async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete site',
      content: Text(
        'Are you sure you want to delete "${widget.site['name']}"? '
        'This will remove all deployments, domains, and data. This action cannot be undone.',
        style: TextStyle(color: Colors.white.withOpacity(0.7)),
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
      setState(() => _deleting = true);
      try {
        final api = ref.read(apiClientProvider);
        await api.delete('/deploy/targets/${widget.siteId}');
        ref.invalidate(sitesProvider);
        ref.read(_selectedSiteProvider.notifier).state = null;
      } finally {
        if (mounted) setState(() => _deleting = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final framework = _frameworkById(widget.site['framework'] ?? 'static');

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // ── General ──────────────────────────────────────────────────────
          _SectionHeader(title: 'General'),
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: _surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.white.withOpacity(0.06)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                AppDialogField(
                  controller: _nameCtrl,
                  label: 'Site name',
                  hint: 'my-site',
                ),
                const SizedBox(height: 12),
                _settingRow('Framework', framework.label),
                const SizedBox(height: 12),
                AppDialogField(
                  controller: _repoCtrl,
                  label: 'Git repository',
                  hint: 'https://github.com/user/repo',
                ),
              ],
            ),
          ),

          const SizedBox(height: 24),

          // ── Build configuration ──────────────────────────────────────────
          _SectionHeader(title: 'Build configuration'),
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: _surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.white.withOpacity(0.06)),
            ),
            child: Column(
              children: [
                AppDialogField(
                  controller: _installCmdCtrl,
                  label: 'Install command',
                  hint: 'npm install',
                ),
                const SizedBox(height: 12),
                AppDialogField(
                  controller: _buildCmdCtrl,
                  label: 'Build command',
                  hint: 'npm run build',
                ),
                const SizedBox(height: 12),
                AppDialogField(
                  controller: _outputDirCtrl,
                  label: 'Output directory',
                  hint: 'dist',
                ),
              ],
            ),
          ),

          const SizedBox(height: 24),

          // ── Environment variables ────────────────────────────────────────
          _SectionHeader(title: 'Environment variables'),
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: _surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.white.withOpacity(0.06)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                ..._envVars.asMap().entries.map((entry) {
                  final idx = entry.key;
                  final pair = entry.value;
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 8),
                    child: Row(
                      children: [
                        Expanded(
                          child: TextField(
                            controller: pair.key,
                            style: const TextStyle(
                                color: Colors.white,
                                fontSize: 13,
                                fontFamily: 'monospace'),
                            decoration: InputDecoration(
                              hintText: 'KEY',
                              hintStyle: TextStyle(
                                  color: Colors.white.withOpacity(0.22),
                                  fontSize: 13),
                              filled: true,
                              fillColor: _fieldFill,
                              isDense: true,
                              contentPadding: const EdgeInsets.symmetric(
                                  horizontal: 10, vertical: 8),
                              border: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(6),
                                borderSide:
                                    const BorderSide(color: _fieldBorder),
                              ),
                              enabledBorder: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(6),
                                borderSide:
                                    const BorderSide(color: _fieldBorder),
                              ),
                              focusedBorder: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(6),
                                borderSide:
                                    const BorderSide(color: _accent),
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        Expanded(
                          child: TextField(
                            controller: pair.value,
                            style: const TextStyle(
                                color: Colors.white,
                                fontSize: 13,
                                fontFamily: 'monospace'),
                            decoration: InputDecoration(
                              hintText: 'VALUE',
                              hintStyle: TextStyle(
                                  color: Colors.white.withOpacity(0.22),
                                  fontSize: 13),
                              filled: true,
                              fillColor: _fieldFill,
                              isDense: true,
                              contentPadding: const EdgeInsets.symmetric(
                                  horizontal: 10, vertical: 8),
                              border: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(6),
                                borderSide:
                                    const BorderSide(color: _fieldBorder),
                              ),
                              enabledBorder: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(6),
                                borderSide:
                                    const BorderSide(color: _fieldBorder),
                              ),
                              focusedBorder: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(6),
                                borderSide:
                                    const BorderSide(color: _accent),
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        IconButton(
                          icon: const Icon(LucideIcons.trash2,
                              size: 14, color: _dimText),
                          onPressed: () => setState(() {
                            _envVars[idx].key.dispose();
                            _envVars[idx].value.dispose();
                            _envVars.removeAt(idx);
                          }),
                          tooltip: 'Remove',
                          padding: EdgeInsets.zero,
                          constraints: const BoxConstraints(
                              minWidth: 28, minHeight: 28),
                        ),
                      ],
                    ),
                  );
                }),
                const SizedBox(height: 4),
                GestureDetector(
                  onTap: () => setState(() {
                    _envVars.add(MapEntry(
                      TextEditingController(),
                      TextEditingController(),
                    ));
                  }),
                  child: MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(LucideIcons.plus, size: 14, color: _accent),
                        const SizedBox(width: 6),
                        Text('Add variable',
                            style: TextStyle(
                                color: _accent, fontSize: 13)),
                      ],
                    ),
                  ),
                ),
              ],
            ),
          ),

          const SizedBox(height: 16),

          // Save button
          Align(
            alignment: Alignment.centerRight,
            child: AppDialogAction(
              label: 'Save changes',
              loading: _saving,
              onTap: _save,
            ),
          ),

          const SizedBox(height: 32),

          // ── Danger zone ──────────────────────────────────────────────────
          _SectionHeader(title: 'Danger zone', danger: true),
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: _surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: _red.withOpacity(0.2)),
            ),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('Delete this site',
                          style: TextStyle(
                              color: Colors.white,
                              fontSize: 14,
                              fontWeight: FontWeight.w500)),
                      const SizedBox(height: 4),
                      Text(
                        'Once deleted, all deployments, domains, and logs will be permanently removed.',
                        style: TextStyle(
                            color: Colors.white.withOpacity(0.5),
                            fontSize: 12),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 16),
                AppDialogAction(
                  label: _deleting ? 'Deleting...' : 'Delete site',
                  destructive: true,
                  loading: _deleting,
                  onTap: _delete,
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),
        ],
      ),
    );
  }

  Widget _settingRow(String label, String value) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 120,
          child: Text(label,
              style: TextStyle(
                  color: Colors.white.withOpacity(0.5),
                  fontSize: 12,
                  fontWeight: FontWeight.w500)),
        ),
        Text(value,
            style: const TextStyle(color: Colors.white, fontSize: 13)),
      ],
    );
  }
}

// ── Section header ──────────────────────────────────────────────────────────

class _SectionHeader extends StatelessWidget {
  final String title;
  final bool danger;

  const _SectionHeader({required this.title, this.danger = false});

  @override
  Widget build(BuildContext context) {
    return Text(title,
        style: TextStyle(
          color: danger ? _red : Colors.white,
          fontSize: 14,
          fontWeight: FontWeight.w600,
        ));
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// UTILITY HELPERS
// ═══════════════════════════════════════════════════════════════════════════════

String _timeAgo(dynamic ts) {
  if (ts == null) return '';
  try {
    final dt = DateTime.parse(ts.toString());
    final diff = DateTime.now().difference(dt);
    if (diff.inDays > 30) return '${(diff.inDays / 30).floor()}mo ago';
    if (diff.inDays > 0) return '${diff.inDays}d ago';
    if (diff.inHours > 0) return '${diff.inHours}h ago';
    if (diff.inMinutes > 0) return '${diff.inMinutes}m ago';
    return 'just now';
  } catch (_) {
    return '';
  }
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

String _formatDuration(dynamic seconds) {
  if (seconds == null) return '--';
  final s = (seconds is num) ? seconds.toInt() : 0;
  if (s <= 0) return '--';
  if (s < 60) return '${s}s';
  final m = s ~/ 60;
  final r = s % 60;
  return r > 0 ? '${m}m ${r}s' : '${m}m';
}

String _formatBytes(dynamic bytes) {
  if (bytes == null) return '--';
  final b = (bytes is num) ? bytes.toDouble() : 0.0;
  if (b <= 0) return '--';
  if (b < 1024) return '${b.toInt()} B';
  if (b < 1024 * 1024) return '${(b / 1024).toStringAsFixed(1)} KB';
  if (b < 1024 * 1024 * 1024) {
    return '${(b / (1024 * 1024)).toStringAsFixed(1)} MB';
  }
  return '${(b / (1024 * 1024 * 1024)).toStringAsFixed(2)} GB';
}

String _formatNumber(dynamic n) {
  if (n == null) return '--';
  final num v = (n is num) ? n : 0;
  if (v >= 1000000) return '${(v / 1000000).toStringAsFixed(1)}M';
  if (v >= 1000) return '${(v / 1000).toStringAsFixed(1)}K';
  return '${v.toInt()}';
}
