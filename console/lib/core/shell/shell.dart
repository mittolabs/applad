import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../providers/project_provider.dart';
import '../providers/org_provider.dart';
import '../api/client.dart';
import '../widgets/search_modal.dart';
import '../widgets/app_dialog.dart';
import '../widgets/navbar_popovers.dart';
import '../widgets/console_footer.dart';
import '../providers/experiments_provider.dart';
import '../providers/environment_provider.dart';

// ═════════════════════════════════════════════════════════════════════════════
// Constants
// ═════════════════════════════════════════════════════════════════════════════

const _bg = Color(0xFF0B0B0F);
const _railBg = Color(0xFF101014);
const _panelBg = Color(0xFF131317);
const _accent = Color(0xFF3472A4);
const _railWidth = 68.0;
const _panelWidth = 220.0;

// ═════════════════════════════════════════════════════════════════════════════
// Navigation group model
// ═════════════════════════════════════════════════════════════════════════════

class _NavGroup {
  final String id;
  final String label;
  final IconData icon;
  final List<_NavChild> children;
  final bool pinBottom;

  const _NavGroup(this.id, this.label, this.icon, this.children,
      {this.pinBottom = false});
}

class _NavChild {
  final String label;
  final String route; // relative to /project/{id}/
  final IconData icon;
  final bool placeholder; // not yet implemented

  const _NavChild(this.label, this.route, this.icon,
      {this.placeholder = false});
}

List<_NavGroup> _buildGroups() => [
      _NavGroup('overview', 'Overview', LucideIcons.barChart3, []),
      _NavGroup('specify', 'Specify', LucideIcons.fileText, [
        _NavChild('Feature specs', 'specify', LucideIcons.fileText,
            placeholder: true),
        _NavChild('User stories', 'specify/stories', LucideIcons.bookOpen,
            placeholder: true),
        _NavChild(
            'API contracts', 'specify/api', LucideIcons.fileCode,
            placeholder: true),
      ]),
      _NavGroup('design', 'Design', LucideIcons.figma, [
        _NavChild('Components', 'design', LucideIcons.component,
            placeholder: true),
        _NavChild('Pages', 'design/pages', LucideIcons.layout,
            placeholder: true),
        _NavChild('Assets', 'design/assets', LucideIcons.image,
            placeholder: true),
        _NavChild('Prototypes', 'design/prototypes', LucideIcons.play,
            placeholder: true),
      ]),
      _NavGroup('build', 'Build', LucideIcons.box, [
        _NavChild('Auth', 'auth', LucideIcons.users),
        _NavChild('Databases', 'databases', LucideIcons.database),
        _NavChild('Storage', 'storage', LucideIcons.folderClosed),
        _NavChild('Messaging', 'messaging', LucideIcons.messageSquare),
        _NavChild('Workflows', 'workflows', LucideIcons.gitBranch),
      ]),
      _NavGroup('test', 'Test', LucideIcons.flaskConical, [
        _NavChild('Test recorder', 'test', LucideIcons.video,
            placeholder: true),
        _NavChild('Test suites', 'test/suites', LucideIcons.listChecks,
            placeholder: true),
        _NavChild('Device lab', 'test/devices', LucideIcons.smartphone,
            placeholder: true),
        _NavChild('Bug capture', 'test/bugs', LucideIcons.bug,
            placeholder: true),
        _NavChild('Coverage', 'test/coverage', LucideIcons.pieChart,
            placeholder: true),
      ]),
      _NavGroup('deploy', 'Deploy', LucideIcons.rocket, [
        _NavChild('Functions', 'functions', LucideIcons.zap),
        _NavChild('Sites', 'sites', LucideIcons.globe),
        _NavChild('Mobile', 'mobile', LucideIcons.smartphone),
        _NavChild('Desktop', 'desktop', LucideIcons.monitor),
        _NavChild('Containers', 'containers', LucideIcons.box),
        _NavChild('Environments', 'environments', LucideIcons.layers),
        _NavChild('Feature Flags', 'flags', LucideIcons.toggleRight),
      ]),
      _NavGroup('observe', 'Observe', LucideIcons.activity, [
        _NavChild('Analytics', 'analytics', LucideIcons.barChart3,
            placeholder: true),
        _NavChild('Logs', 'logs', LucideIcons.terminal, placeholder: true),
        _NavChild('Health', 'health', LucideIcons.heartPulse,
            placeholder: true),
        _NavChild('Errors', 'errors', LucideIcons.alertTriangle,
            placeholder: true),
      ]),
_NavGroup('settings', 'Settings', LucideIcons.settings, [
        _NavChild('General', 'settings', LucideIcons.settings),
        _NavChild('Platforms', 'settings', LucideIcons.smartphone),
        _NavChild('Team', 'settings', LucideIcons.users),
        _NavChild('Experiments', 'experiments', LucideIcons.flaskConical),
      ], pinBottom: true),
    ];

// ═════════════════════════════════════════════════════════════════════════════
// Search intent
// ═════════════════════════════════════════════════════════════════════════════

class _OpenSearchIntent extends Intent {
  const _OpenSearchIntent();
}

// ═════════════════════════════════════════════════════════════════════════════
// Shell
// ═════════════════════════════════════════════════════════════════════════════

class AppShell extends ConsumerStatefulWidget {
  final Widget child;
  const AppShell({super.key, required this.child});

  @override
  ConsumerState<AppShell> createState() => _AppShellState();
}

class _AppShellState extends ConsumerState<AppShell> {
  String? _syncedProjectId;
  String? _expandedGroup; // null = panel collapsed
  bool _aiChatOpen = false;

  void _syncProject(String projectId) {
    if (_syncedProjectId == projectId) return;
    _syncedProjectId = projectId;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      ref.read(currentProjectProvider.notifier).state = projectId;
      ref.read(apiClientProvider).setProject(projectId);
    });
  }

  void _openSearch(BuildContext context, String? projectId) {
    final projects = ref.read(projectsProvider).valueOrNull ?? [];
    final orgs = ref.read(orgsProvider).valueOrNull ?? [];
    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.65),
      builder: (ctx) => SearchModal(
        projects: projects,
        orgs: orgs,
        projectId: projectId,
        onNavigate: (path) {
          Navigator.of(ctx).pop();
          context.go(path);
        },
        onCreateProject: () {
          Navigator.of(ctx).pop();
          context.go('/projects');
        },
        onCreateOrg: () {
          Navigator.of(ctx).pop();
          _showCreateOrgDialog(context);
        },
      ),
    );
  }

  void _showCreateOrgDialog(BuildContext context) {
    final nameCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create organization',
      subtitle: 'Organize your projects into teams',
      content: AppDialogField(
          controller: nameCtrl,
          label: 'Organization name',
          hint: 'My org',
          autofocus: true),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            try {
              final api = ref.read(apiClientProvider);
              await api.post('/organizations', data: {'name': nameCtrl.text});
              ref.invalidate(orgsProvider);
            } catch (_) {}
            if (context.mounted) Navigator.of(context, rootNavigator: true).pop();
          },
        ),
      ],
    );
  }

  /// Determine which group is active based on the current path.
  String _activeGroup(String currentPath, String projectId) {
    final base = '/project/$projectId/';
    if (!currentPath.startsWith(base)) return 'overview';
    final segment = currentPath.substring(base.length).split('/').first;
    const routeToGroup = {
      'overview': 'overview',
      'auth': 'build',
      'databases': 'build',
      'storage': 'build',
      'functions': 'deploy',
      'messaging': 'build',
      'workflows': 'build',
      'deploy': 'deploy',
      'sites': 'deploy',
      'containers': 'deploy',
      'mobile': 'deploy',
      'desktop': 'deploy',
      'flags': 'deploy',
      'environments': 'deploy',
      'settings': 'settings',
      'specify': 'specify',
      'design': 'design',
      'test': 'test',
      'analytics': 'observe',
      'logs': 'observe',
      'health': 'observe',
    };
    return routeToGroup[segment] ?? 'overview';
  }

  @override
  Widget build(BuildContext context) {
    final routerState = GoRouterState.of(context);
    final projectId = routerState.pathParameters['projectId'];
    final currentPath = routerState.uri.path;

    if (projectId != null) _syncProject(projectId);

    final experiments = ref.watch(experimentsProvider);
    final allGroups = _buildGroups();
    // Filter groups based on experiments
    final experimentalGroups = {'specify', 'design', 'test', 'observe'};
    final groups = allGroups.where((g) {
      if (!experimentalGroups.contains(g.id)) return true;
      final map = experiments.toMap();
      return map[g.id] == true;
    }).toList();
    final activeGroup = _activeGroup(currentPath, projectId ?? '');

    return Shortcuts(
      shortcuts: <LogicalKeySet, Intent>{
        LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.keyK):
            const _OpenSearchIntent(),
        LogicalKeySet(LogicalKeyboardKey.control, LogicalKeyboardKey.keyK):
            const _OpenSearchIntent(),
      },
      child: Actions(
        actions: <Type, Action<Intent>>{
          _OpenSearchIntent: CallbackAction<_OpenSearchIntent>(
            onInvoke: (_) {
              _openSearch(context, projectId);
              return null;
            },
          ),
        },
        child: Focus(
          autofocus: true,
          child: Scaffold(
            backgroundColor: _bg,
            body: Column(
              children: [
                _TopNavBar(
                  projectId: projectId,
                  currentPath: currentPath,
                  onSearchTap: () => _openSearch(context, projectId),
                ),
                Expanded(
                  child: Row(
                    children: [
                      // Icon rail
                      _IconRail(
                        groups: groups,
                        activeGroup: activeGroup,
                        expandedGroup: _expandedGroup,
                        onGroupTap: (id) {
                          // Direct-navigate groups (tabs are on the page)
                          const directGroups = {
                            'overview': 'overview',
                            'settings': 'settings',
                          };
                          if (directGroups.containsKey(id)) {
                            setState(() => _expandedGroup = null);
                            if (projectId != null) {
                              context.go(
                                  '/project/$projectId/${directGroups[id]}');
                            }
                            return;
                          }
                          setState(() {
                            if (id == 'ai') {
                              _aiChatOpen = !_aiChatOpen;
                              return;
                            }
                            if (_expandedGroup == id) {
                              _expandedGroup = null;
                            } else {
                              _expandedGroup = id;
                              // Auto-navigate to first non-placeholder child
                              if (projectId != null) {
                                final group = groups.firstWhere(
                                    (g) => g.id == id,
                                    orElse: () => groups.first);
                                final firstChild =
                                    group.children.cast<_NavChild?>().firstWhere(
                                        (c) => c != null && !c.placeholder,
                                        orElse: () => null);
                                if (firstChild != null) {
                                  context.go(
                                      '/project/$projectId/${firstChild.route}');
                                }
                              }
                            }
                          });
                        },
                        aiChatOpen: _aiChatOpen,
                        showAiButton: experiments.aiChat,
                        projectId: projectId,
                        currentPath: currentPath,
                      ),
                      // Expanded detail panel
                      if (_expandedGroup != null)
                        _DetailPanel(
                          group: groups.firstWhere(
                              (g) => g.id == _expandedGroup,
                              orElse: () => groups.first),
                          projectId: projectId ?? '',
                          currentPath: currentPath,
                          onClose: () =>
                              setState(() => _expandedGroup = null),
                        ),
                      // Main content
                      Expanded(
                        child: Column(
                          children: [
                            Expanded(child: widget.child),
                            const ConsoleFooter(),
                          ],
                        ),
                      ),
                      // AI chat panel (right overlay)
                      if (_aiChatOpen && experiments.aiChat)
                        _AIChatPanel(
                          onClose: () =>
                              setState(() => _aiChatOpen = false),
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

// ═════════════════════════════════════════════════════════════════════════════
// Icon Rail (left, 56px)
// ═════════════════════════════════════════════════════════════════════════════

class _IconRail extends StatelessWidget {
  final List<_NavGroup> groups;
  final String activeGroup;
  final String? expandedGroup;
  final ValueChanged<String> onGroupTap;
  final bool aiChatOpen;
  final bool showAiButton;
  final String? projectId;
  final String currentPath;

  const _IconRail({
    required this.groups,
    required this.activeGroup,
    required this.expandedGroup,
    required this.onGroupTap,
    required this.aiChatOpen,
    this.showAiButton = false,
    this.projectId,
    this.currentPath = '',
  });

  @override
  Widget build(BuildContext context) {
    final topGroups = groups.where((g) => !g.pinBottom).toList();
    final bottomGroups = groups.where((g) => g.pinBottom).toList();
    final isOnGetStarted = projectId != null &&
        currentPath.endsWith('/get-started');

    return Container(
      width: _railWidth,
      decoration: BoxDecoration(
        color: _railBg,
        border: Border(
            right: BorderSide(color: Colors.white.withOpacity(0.06))),
      ),
      child: Column(
        children: [
          const SizedBox(height: 8),
          // Get started progress ring
          if (projectId != null)
            _GetStartedRailItem(
              projectId: projectId!,
              isActive: isOnGetStarted,
              onTap: () => context.go('/project/$projectId/get-started'),
            ),
          // Top groups
          ...topGroups
              .map((g) => _RailIcon(
                    icon: g.icon,
                    tooltip: g.label,
                    isActive: !isOnGetStarted &&
                        expandedGroup == null &&
                        activeGroup == g.id,
                    isExpanded: expandedGroup == g.id,
                    onTap: () => onGroupTap(g.id),
                  )),

          const Spacer(),

          // AI button (only when experiment enabled)
          if (showAiButton)
            _RailIcon(
              icon: LucideIcons.sparkles,
              tooltip: 'AI Assistant',
              isActive: aiChatOpen,
              isExpanded: false,
              onTap: () => onGroupTap('ai'),
              accentColor: const Color(0xFF8B5CF6),
            ),

          const SizedBox(height: 4),

          // Bottom groups (settings)
          ...bottomGroups
              .map((g) => _RailIcon(
                    icon: g.icon,
                    tooltip: g.label,
                    isActive: expandedGroup == null && activeGroup == g.id,
                    isExpanded: expandedGroup == g.id,
                    onTap: () => onGroupTap(g.id),
                  )),
          const SizedBox(height: 10),
        ],
      ),
    );
  }
}

class _RailIcon extends StatefulWidget {
  final IconData icon;
  final String tooltip;
  final bool isActive;
  final bool isExpanded;
  final VoidCallback onTap;
  final Color? accentColor;

  const _RailIcon({
    required this.icon,
    required this.tooltip,
    required this.isActive,
    required this.isExpanded,
    required this.onTap,
    this.accentColor,
  });

  @override
  State<_RailIcon> createState() => _RailIconState();
}

class _RailIconState extends State<_RailIcon> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final accentColor = widget.accentColor ?? _accent;
    final active = widget.isActive || widget.isExpanded;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Tooltip(
        message: widget.tooltip,
        preferBelow: false,
        waitDuration: const Duration(milliseconds: 400),
        child: MouseRegion(
          onEnter: (_) => setState(() => _hovered = true),
          onExit: (_) => setState(() => _hovered = false),
          cursor: SystemMouseCursors.click,
          child: GestureDetector(
            onTap: widget.onTap,
            child: SizedBox(
              width: _railWidth,
              height: 44,
              child: Stack(
                alignment: Alignment.center,
                children: [
                  // Left accent bar — only when active
                  if (active)
                    Positioned(
                      left: 0,
                      top: 6,
                      bottom: 6,
                      child: Container(
                        width: 3,
                        decoration: BoxDecoration(
                          color: accentColor,
                          borderRadius: const BorderRadius.only(
                            topRight: Radius.circular(3),
                            bottomRight: Radius.circular(3),
                          ),
                        ),
                      ),
                    ),
                  // Icon box
                  Container(
                    width: 38,
                    height: 38,
                    decoration: BoxDecoration(
                      color: active
                          ? Colors.white.withOpacity(0.08)
                          : _hovered
                              ? Colors.white.withOpacity(0.04)
                              : Colors.transparent,
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Icon(
                      widget.icon,
                      size: 18,
                      color: active
                          ? Colors.white
                          : _hovered
                              ? Colors.white.withOpacity(0.65)
                              : Colors.white.withOpacity(0.28),
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

// ═════════════════════════════════════════════════════════════════════════════
// Get Started rail item (progress ring)
// ═════════════════════════════════════════════════════════════════════════════

class _GetStartedRailItem extends ConsumerWidget {
  final String projectId;
  final bool isActive;
  final VoidCallback onTap;

  const _GetStartedRailItem({
    required this.projectId,
    required this.isActive,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Reuse the stats provider from overview to compute completion
    final statsAsync = ref.watch(_getStartedProgressProvider(projectId));
    final progress = statsAsync.valueOrNull ?? 0.0;

    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
      child: Tooltip(
        message: 'Get started',
        preferBelow: false,
        waitDuration: const Duration(milliseconds: 400),
        child: MouseRegion(
          cursor: SystemMouseCursors.click,
          child: GestureDetector(
            onTap: onTap,
            child: SizedBox(
              width: _railWidth,
              height: 44,
              child: Stack(
                alignment: Alignment.center,
                children: [
                  if (isActive)
                    Positioned(
                      left: 0,
                      top: 6,
                      bottom: 6,
                      child: Container(
                        width: 3,
                        decoration: BoxDecoration(
                          color: _accent,
                          borderRadius: const BorderRadius.only(
                            topRight: Radius.circular(3),
                            bottomRight: Radius.circular(3),
                          ),
                        ),
                      ),
                    ),
                  // Progress ring
                  SizedBox(
                    width: 32,
                    height: 32,
                    child: Stack(
                      alignment: Alignment.center,
                      children: [
                        // Background ring
                        SizedBox(
                          width: 32,
                          height: 32,
                          child: CircularProgressIndicator(
                            value: 1.0,
                            strokeWidth: 2.5,
                            color: Colors.white.withOpacity(0.08),
                          ),
                        ),
                        // Progress ring
                        SizedBox(
                          width: 32,
                          height: 32,
                          child: CircularProgressIndicator(
                            value: progress,
                            strokeWidth: 2.5,
                            color: progress >= 1.0
                                ? const Color(0xFF10B981)
                                : _accent,
                            strokeCap: StrokeCap.round,
                          ),
                        ),
                        // Percentage text
                        Text(
                          '${(progress * 100).round()}',
                          style: TextStyle(
                            color: isActive
                                ? Colors.white
                                : Colors.white.withOpacity(0.5),
                            fontSize: 9,
                            fontWeight: FontWeight.w600,
                          ),
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

/// Computes get-started progress (0.0 to 1.0) from resource counts.
final _getStartedProgressProvider =
    FutureProvider.family<double, String>((ref, projectId) async {
  final api = ref.read(apiClientProvider);
  int done = 1; // "Create project" is always done
  const total = 3; // Create project, Connect platform, Build your app

  Future<int> count(String path, String key) async {
    try {
      final res = await api.get(path);
      final data = res.data;
      if (data is Map && data[key] is List) return (data[key] as List).length;
      if (data is Map && data['total'] is int) return data['total'] as int;
    } catch (_) {}
    return 0;
  }

  final results = await Future.wait([
    count('/projects/$projectId/platforms', 'platforms'),
    count('/databases', 'databases'),
    count('/storage/buckets', 'buckets'),
    count('/functions', 'functions'),
    count('/deploy/targets', 'targets'),
    count('/workflows', 'workflows'),
  ]);

  // Connect platform
  if (results[0] > 0) done++;
  // Build your app (any resource)
  if (results.skip(1).any((c) => c > 0)) done++;

  return done / total;
});

// ═════════════════════════════════════════════════════════════════════════════
// Detail Panel (expandable, 220px)
// ═════════════════════════════════════════════════════════════════════════════

class _DetailPanel extends StatelessWidget {
  final _NavGroup group;
  final String projectId;
  final String currentPath;
  final VoidCallback onClose;

  const _DetailPanel({
    required this.group,
    required this.projectId,
    required this.currentPath,
    required this.onClose,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: _panelWidth,
      decoration: BoxDecoration(
        color: _panelBg,
        border: Border(
            right: BorderSide(color: Colors.white.withOpacity(0.06))),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Panel header
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 8, 8),
            child: Row(
              children: [
                Text(
                  group.label,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const Spacer(),
                GestureDetector(
                  onTap: onClose,
                  child: Icon(LucideIcons.panelLeftClose,
                      size: 16,
                      color: Colors.white.withOpacity(0.3)),
                ),
              ],
            ),
          ),
          Container(
            height: 1,
            margin: const EdgeInsets.symmetric(horizontal: 12),
            color: Colors.white.withOpacity(0.06),
          ),
          const SizedBox(height: 6),
          // Children
          ...group.children.map((child) {
            final path = '/project/$projectId/${child.route}';
            final active = currentPath.startsWith(path);

            return _PanelItem(
              icon: child.icon,
              label: child.label,
              active: active,
              placeholder: child.placeholder,
              onTap: () {
                if (!child.placeholder) {
                  context.go(path);
                }
              },
            );
          }),
        ],
      ),
    );
  }
}

class _PanelItem extends StatefulWidget {
  final IconData icon;
  final String label;
  final bool active;
  final bool placeholder;
  final VoidCallback onTap;

  const _PanelItem({
    required this.icon,
    required this.label,
    required this.active,
    required this.placeholder,
    required this.onTap,
  });

  @override
  State<_PanelItem> createState() => _PanelItemState();
}

class _PanelItemState extends State<_PanelItem> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: widget.placeholder
          ? SystemMouseCursors.basic
          : SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          width: double.infinity,
          margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 1),
          padding:
              const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
          decoration: BoxDecoration(
            color: widget.active
                ? Colors.white.withOpacity(0.08)
                : _hovered && !widget.placeholder
                    ? Colors.white.withOpacity(0.04)
                    : Colors.transparent,
            borderRadius: BorderRadius.circular(7),
          ),
          child: Row(
            children: [
              Icon(
                widget.icon,
                size: 15,
                color: widget.placeholder
                    ? Colors.white.withOpacity(0.15)
                    : widget.active
                        ? Colors.white.withOpacity(0.9)
                        : Colors.white.withOpacity(0.4),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  widget.label,
                  style: TextStyle(
                    color: widget.placeholder
                        ? Colors.white.withOpacity(0.2)
                        : widget.active
                            ? Colors.white.withOpacity(0.9)
                            : Colors.white.withOpacity(0.55),
                    fontSize: 13,
                    fontWeight:
                        widget.active ? FontWeight.w500 : FontWeight.w400,
                  ),
                ),
              ),
              if (widget.placeholder)
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 5, vertical: 1),
                  decoration: BoxDecoration(
                    color: Colors.white.withOpacity(0.06),
                    borderRadius: BorderRadius.circular(3),
                  ),
                  child: Text('Soon',
                      style: TextStyle(
                          color: Colors.white.withOpacity(0.2),
                          fontSize: 9)),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

// ═════════════════════════════════════════════════════════════════════════════
// AI Chat Panel (right side overlay)
// ═════════════════════════════════════════════════════════════════════════════

class _AIChatPanel extends StatefulWidget {
  final VoidCallback onClose;
  const _AIChatPanel({required this.onClose});

  @override
  State<_AIChatPanel> createState() => _AIChatPanelState();
}

class _AIChatPanelState extends State<_AIChatPanel> {
  final _msgCtrl = TextEditingController();
  final _messages = <Map<String, String>>[];
  bool _loading = false;

  @override
  void dispose() {
    _msgCtrl.dispose();
    super.dispose();
  }

  void _send() {
    final text = _msgCtrl.text.trim();
    if (text.isEmpty) return;
    setState(() {
      _messages.add({'role': 'user', 'text': text});
      _msgCtrl.clear();
      _loading = true;
    });
    // Simulate AI response
    Future.delayed(const Duration(milliseconds: 800), () {
      if (mounted) {
        setState(() {
          _messages.add({
            'role': 'assistant',
            'text':
                'I can help you with that. This AI chat will be connected to Applad\'s APIs to create resources, trigger workflows, and deploy — all from this conversation.'
          });
          _loading = false;
        });
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 360,
      decoration: BoxDecoration(
        color: _panelBg,
        border: Border(
            left: BorderSide(color: Colors.white.withOpacity(0.06))),
      ),
      child: Column(
        children: [
          // Header
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 8, 12),
            child: Row(
              children: [
                Icon(LucideIcons.sparkles,
                    size: 18, color: const Color(0xFF8B5CF6)),
                const SizedBox(width: 8),
                const Text('AI Assistant',
                    style: TextStyle(
                        color: Colors.white,
                        fontSize: 14,
                        fontWeight: FontWeight.w600)),
                const Spacer(),
                GestureDetector(
                  onTap: widget.onClose,
                  child: Icon(LucideIcons.x,
                      size: 16,
                      color: Colors.white.withOpacity(0.3)),
                ),
              ],
            ),
          ),
          Container(height: 1, color: Colors.white.withOpacity(0.06)),

          // Messages
          Expanded(
            child: _messages.isEmpty
                ? _emptyChat()
                : ListView.builder(
                    padding: const EdgeInsets.all(16),
                    itemCount: _messages.length + (_loading ? 1 : 0),
                    itemBuilder: (ctx, i) {
                      if (i >= _messages.length) {
                        return Padding(
                          padding: const EdgeInsets.only(top: 12),
                          child: Row(children: [
                            SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                  color: Colors.white.withOpacity(0.3)),
                            ),
                            const SizedBox(width: 8),
                            Text('Thinking...',
                                style: TextStyle(
                                    color: Colors.white.withOpacity(0.3),
                                    fontSize: 12)),
                          ]),
                        );
                      }
                      final msg = _messages[i];
                      final isUser = msg['role'] == 'user';
                      return Padding(
                        padding: const EdgeInsets.only(bottom: 12),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Container(
                              width: 24,
                              height: 24,
                              decoration: BoxDecoration(
                                color: isUser
                                    ? _accent.withOpacity(0.15)
                                    : const Color(0xFF8B5CF6)
                                        .withOpacity(0.15),
                                borderRadius: BorderRadius.circular(6),
                              ),
                              child: Icon(
                                isUser
                                    ? LucideIcons.user
                                    : LucideIcons.sparkles,
                                size: 12,
                                color: isUser
                                    ? _accent
                                    : const Color(0xFF8B5CF6),
                              ),
                            ),
                            const SizedBox(width: 10),
                            Expanded(
                              child: Text(
                                msg['text'] ?? '',
                                style: TextStyle(
                                  color: Colors.white
                                      .withOpacity(isUser ? 0.9 : 0.7),
                                  fontSize: 13,
                                  height: 1.5,
                                ),
                              ),
                            ),
                          ],
                        ),
                      );
                    },
                  ),
          ),

          // Input
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              border: Border(
                  top: BorderSide(
                      color: Colors.white.withOpacity(0.06))),
            ),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _msgCtrl,
                    style: const TextStyle(
                        color: Colors.white, fontSize: 13),
                    decoration: InputDecoration(
                      hintText: 'Ask anything...',
                      hintStyle: TextStyle(
                          color: Colors.white.withOpacity(0.22),
                          fontSize: 13),
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(
                            color: Colors.white.withOpacity(0.1)),
                      ),
                      enabledBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(
                            color: Colors.white.withOpacity(0.1)),
                      ),
                      focusedBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide:
                            const BorderSide(color: Color(0xFF8B5CF6)),
                      ),
                      filled: true,
                      fillColor: Colors.white.withOpacity(0.04),
                      contentPadding: const EdgeInsets.symmetric(
                          horizontal: 12, vertical: 10),
                    ),
                    onSubmitted: (_) => _send(),
                  ),
                ),
                const SizedBox(width: 8),
                GestureDetector(
                  onTap: _send,
                  child: Container(
                    width: 36,
                    height: 36,
                    decoration: BoxDecoration(
                      color: const Color(0xFF8B5CF6),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: const Icon(LucideIcons.arrowUp,
                        size: 16, color: Colors.white),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _emptyChat() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 56,
              height: 56,
              decoration: BoxDecoration(
                color: const Color(0xFF8B5CF6).withOpacity(0.1),
                borderRadius: BorderRadius.circular(14),
              ),
              child: const Icon(LucideIcons.sparkles,
                  size: 24, color: Color(0xFF8B5CF6)),
            ),
            const SizedBox(height: 20),
            const Text('AI Assistant',
                style: TextStyle(
                    color: Colors.white,
                    fontSize: 16,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 8),
            Text(
              'Create apps, manage databases, deploy services, and automate workflows — all through conversation.',
              textAlign: TextAlign.center,
              style: TextStyle(
                  color: Colors.white.withOpacity(0.4),
                  fontSize: 13,
                  height: 1.5),
            ),
            const SizedBox(height: 24),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              alignment: WrapAlignment.center,
              children: [
                _suggestion('Create a users table'),
                _suggestion('Deploy to production'),
                _suggestion('Build a login flow'),
                _suggestion('Set up a webhook'),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _suggestion(String text) {
    return GestureDetector(
      onTap: () {
        _msgCtrl.text = text;
        _send();
      },
      child: Container(
        padding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: Colors.white.withOpacity(0.04),
          borderRadius: BorderRadius.circular(20),
          border:
              Border.all(color: Colors.white.withOpacity(0.08)),
        ),
        child: Text(text,
            style: TextStyle(
                color: Colors.white.withOpacity(0.5),
                fontSize: 12)),
      ),
    );
  }
}

// ═════════════════════════════════════════════════════════════════════════════
// Top navigation bar (unchanged)
// ═════════════════════════════════════════════════════════════════════════════

class _TopNavBar extends ConsumerWidget {
  final String? projectId;
  final String currentPath;
  final VoidCallback onSearchTap;

  const _TopNavBar({
    required this.projectId,
    required this.currentPath,
    required this.onSearchTap,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final orgs = ref.watch(orgsProvider).valueOrNull ?? [];
    final currentOrgId = ref.watch(currentOrgProvider);
    final projects = ref.watch(projectsProvider).valueOrNull ?? [];

    String orgName = orgs.isNotEmpty ? orgs.first['name'] as String : '';
    if (currentOrgId != null) {
      final org = orgs.where((o) => o['\$id'] == currentOrgId).firstOrNull;
      if (org != null) orgName = org['name'] as String;
    } else if (orgs.isNotEmpty) {
      orgName = orgs.first['name'] as String;
    }

    String? projectName;
    if (projectId != null) {
      final proj =
          projects.where((p) => p['\$id'] == projectId).firstOrNull;
      projectName = proj?['name'] as String?;
    }

    return Container(
      height: 52,
      decoration: BoxDecoration(
        color: _bg,
        border: Border(
            bottom: BorderSide(color: Colors.white.withOpacity(0.06))),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          GestureDetector(
            onTap: () => context.go('/projects'),
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: ClipRRect(
                borderRadius: BorderRadius.circular(6),
                child: Image.asset(
                  'assets/icon.png',
                  width: 42,
                  height: 42,
                  fit: BoxFit.cover,
                ),
              ),
            ),
          ),

          const SizedBox(width: 10),
          _sep(),
          const SizedBox(width: 10),

          _OrgNavButton(orgName: orgName, orgs: orgs, ref: ref),

          if (projectId != null) ...[
            const SizedBox(width: 10),
            _sep(),
            const SizedBox(width: 10),
            _ProjectNavButton(
              projectId: projectId!,
              projectName: projectName ?? _short(projectId!),
              projects: projects,
              currentPath: currentPath,
            ),
            const SizedBox(width: 10),
            // Environment badge
            _EnvironmentBadge(),
          ],

          const Spacer(),

          const FeedbackButton(),
          const SizedBox(width: 2),
          const SupportButton(),
          const SizedBox(width: 2),
          const ThemeToggleButton(),
          const SizedBox(width: 4),

          Tooltip(
            message: '⌘K',
            child: InkWell(
              onTap: onSearchTap,
              borderRadius: BorderRadius.circular(8),
              child: SizedBox(
                width: 34,
                height: 34,
                child: Icon(LucideIcons.search,
                    size: 17,
                    color: Colors.white.withOpacity(0.45)),
              ),
            ),
          ),

          const SizedBox(width: 8),

          const UserMenuButton(),

          const SizedBox(width: 4),
        ],
      ),
    );
  }

  Widget _sep() => Text(
        '/',
        style: TextStyle(
          color: Colors.white.withOpacity(0.18),
          fontSize: 18,
          fontWeight: FontWeight.w300,
        ),
      );

  String _short(String id) => id.length > 8 ? id.substring(0, 8) : id;

}

// ═════════════════════════════════════════════════════════════════════════════
// Shared nav widgets (unchanged)
// ═════════════════════════════════════════════════════════════════════════════

class _ProjectNavButton extends StatelessWidget {
  final String projectId;
  final String projectName;
  final List<Map<String, dynamic>> projects;
  final String currentPath;

  const _ProjectNavButton({
    required this.projectId,
    required this.projectName,
    required this.projects,
    required this.currentPath,
  });

  String _section() {
    final prefix = '/project/$projectId/';
    if (currentPath.startsWith(prefix)) {
      return currentPath.substring(prefix.length).split('/').first;
    }
    return 'overview';
  }

  @override
  Widget build(BuildContext context) {
    return PopupMenuButton<String>(
      offset: const Offset(0, 36),
      color: const Color(0xFF1A1A22),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: BorderSide(color: Colors.white.withOpacity(0.08)),
      ),
      onSelected: (id) => context.go('/project/$id/${_section()}'),
      itemBuilder: (_) => projects
          .map((p) => PopupMenuItem<String>(
                value: p['\$id'] as String,
                child: Text(
                  p['name'] ?? 'Unnamed',
                  style: TextStyle(
                    color: p['\$id'] == projectId
                        ? Colors.white
                        : Colors.white70,
                    fontSize: 13,
                    fontWeight: p['\$id'] == projectId
                        ? FontWeight.w600
                        : FontWeight.w400,
                  ),
                ),
              ))
          .toList(),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(projectName,
              style: const TextStyle(
                  color: Colors.white,
                  fontSize: 13,
                  fontWeight: FontWeight.w500)),
          const SizedBox(width: 4),
          Icon(LucideIcons.chevronDown,
              size: 14, color: Colors.white.withOpacity(0.35)),
        ],
      ),
    );
  }
}

class _OrgNavButton extends StatelessWidget {
  final String orgName;
  final List<Map<String, dynamic>> orgs;
  final WidgetRef ref;

  const _OrgNavButton({
    required this.orgName,
    required this.orgs,
    required this.ref,
  });

  @override
  Widget build(BuildContext context) {
    return PopupMenuButton<String>(
      offset: const Offset(0, 36),
      color: const Color(0xFF1A1A22),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: BorderSide(color: Colors.white.withOpacity(0.08)),
      ),
      onSelected: (value) {
        if (value != '__create__') {
          ref.read(currentOrgProvider.notifier).state = value;
          context.go('/org/$value/projects');
        }
      },
      itemBuilder: (_) => [
        for (final org in orgs)
          PopupMenuItem<String>(
            value: org['\$id'] as String,
            child: Text(org['name'] ?? 'Unnamed',
                style: const TextStyle(color: Colors.white70, fontSize: 13)),
          ),
      ],
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(orgName,
              style: const TextStyle(
                  color: Colors.white,
                  fontSize: 13,
                  fontWeight: FontWeight.w500)),
          const SizedBox(width: 4),
          Icon(LucideIcons.chevronDown,
              size: 14, color: Colors.white.withOpacity(0.35)),
        ],
      ),
    );
  }
}

// ═════════════════════════════════════════════════════════════════════════════
// Environment badge (navbar)
// ═════════════════════════════════════════════════════════════════════════════

class _EnvironmentBadge extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final envsAsync = ref.watch(environmentsProvider);
    final currentEnvId = ref.watch(currentEnvironmentProvider);
    final envs = envsAsync.valueOrNull ?? [];

    if (envs.isEmpty) return const SizedBox.shrink();

    String envName = 'production';
    Color envColor = const Color(0xFF10B981);
    if (currentEnvId != null) {
      final env = envs.where((e) => e['\$id'] == currentEnvId).firstOrNull;
      if (env != null) {
        envName = env['slug'] as String? ?? env['name'] as String? ?? 'production';
      }
    }
    if (envName == 'staging') envColor = const Color(0xFFF59E0B);
    if (envName == 'development') envColor = const Color(0xFF8B5CF6);

    return PopupMenuButton<String>(
      offset: const Offset(0, 36),
      color: const Color(0xFF1A1A22),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: BorderSide(color: Colors.white.withOpacity(0.08)),
      ),
      onSelected: (id) => ref.read(currentEnvironmentProvider.notifier).state = id,
      itemBuilder: (_) => envs.map((e) {
        final id = e['\$id'] as String;
        final name = e['slug'] as String? ?? e['name'] as String? ?? '';
        Color c = const Color(0xFF10B981);
        if (name == 'staging') c = const Color(0xFFF59E0B);
        if (name == 'development') c = const Color(0xFF8B5CF6);
        return PopupMenuItem<String>(
          value: id,
          child: Row(children: [
            Container(width: 8, height: 8, decoration: BoxDecoration(color: c, shape: BoxShape.circle)),
            const SizedBox(width: 8),
            Text(name, style: TextStyle(
              color: currentEnvId == id ? Colors.white : Colors.white70, fontSize: 13,
              fontWeight: currentEnvId == id ? FontWeight.w600 : FontWeight.w400)),
          ]),
        );
      }).toList(),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: envColor.withOpacity(0.1),
          borderRadius: BorderRadius.circular(6),
          border: Border.all(color: envColor.withOpacity(0.25)),
        ),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          Container(width: 6, height: 6, decoration: BoxDecoration(color: envColor, shape: BoxShape.circle)),
          const SizedBox(width: 6),
          Text(envName, style: TextStyle(color: envColor, fontSize: 11, fontWeight: FontWeight.w500)),
          const SizedBox(width: 4),
          Icon(LucideIcons.chevronDown, size: 10, color: envColor),
        ]),
      ),
    );
  }
}
