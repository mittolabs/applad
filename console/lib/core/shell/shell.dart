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
import '../providers/get_started_provider.dart';

// ═════════════════════════════════════════════════════════════════════════════
// Constants
// ═════════════════════════════════════════════════════════════════════════════

const _railBg = Color(0xFF101014);
const _panelBg = Color(0xFF131317);
const _accent = Color(0xFF3472A4);
const _railWidth = 68.0;
const _panelWidth = 220.0;

bool _isLight(BuildContext context) =>
  Theme.of(context).brightness == Brightness.light;

Color _shellBg(BuildContext context) => Theme.of(context).scaffoldBackgroundColor;

Color _railSurface(BuildContext context) =>
  _isLight(context) ? const Color(0xFFF3F5F8) : _railBg;

Color _panelSurface(BuildContext context) =>
  _isLight(context) ? Colors.white : _panelBg;

Color _dividerColor(BuildContext context) => _isLight(context)
  ? Colors.black.withValues(alpha: 0.08)
  : Colors.white.withValues(alpha: 0.06);

Color _primaryTextColor(BuildContext context) =>
  _isLight(context) ? const Color(0xFF1A1A2E) : Colors.white;

Color _secondaryTextColor(BuildContext context) => _isLight(context)
  ? const Color(0xFF1A1A2E).withValues(alpha: 0.62)
  : Colors.white.withValues(alpha: 0.55);

Color _mutedTextColor(BuildContext context) => _isLight(context)
  ? Colors.black.withValues(alpha: 0.35)
  : Colors.white.withValues(alpha: 0.35);

Color _iconColor(BuildContext context) => _isLight(context)
  ? Colors.black.withValues(alpha: 0.55)
  : Colors.white.withValues(alpha: 0.45);

Color _hoverFillColor(BuildContext context) => _isLight(context)
  ? Colors.black.withValues(alpha: 0.04)
  : Colors.white.withValues(alpha: 0.04);

Color _activeFillColor(BuildContext context) => _isLight(context)
  ? _accent.withValues(alpha: 0.1)
  : Colors.white.withValues(alpha: 0.08);

Color _placeholderTextColor(BuildContext context) => _isLight(context)
  ? Colors.black.withValues(alpha: 0.22)
  : Colors.white.withValues(alpha: 0.22);

Color _inputFillColor(BuildContext context) => _isLight(context)
  ? Colors.black.withValues(alpha: 0.03)
  : Colors.white.withValues(alpha: 0.04);

Color _popupSurface(BuildContext context) => Theme.of(context).popupMenuTheme.color ??
  (_isLight(context) ? Colors.white : const Color(0xFF1A1A22));

bool _isMobile(BuildContext context) => MediaQuery.sizeOf(context).width < 650;

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
  final bool placeholder;

  // ignore: unused_element_parameter
  const _NavChild(this.label, this.route, this.icon, {this.placeholder = false});
}

List<_NavGroup> _buildGroups() => [
      const _NavGroup('overview', 'Overview', LucideIcons.barChart3, []),
      const _NavGroup('build', 'Build', LucideIcons.box, [
        _NavChild('Auth', 'auth', LucideIcons.users),
        _NavChild('Databases', 'databases', LucideIcons.database),
        _NavChild('Functions', 'functions', LucideIcons.zap),
        _NavChild('Storage', 'storage', LucideIcons.folderClosed),
        _NavChild('Messaging', 'messaging', LucideIcons.messageSquare),
        _NavChild('Content', 'content', LucideIcons.fileText),
        _NavChild('Realtime', 'realtime', LucideIcons.radio),
        _NavChild('Workflows', 'workflows', LucideIcons.gitBranch),
        _NavChild('Feature Flags', 'flags', LucideIcons.toggleRight),
      ]),
      const _NavGroup('platforms', 'Platforms', LucideIcons.layers, [
        _NavChild('Sites', 'sites', LucideIcons.globe),
        _NavChild('Containers', 'containers', LucideIcons.box),
        _NavChild('Mobile', 'mobile', LucideIcons.smartphone),
        _NavChild('Desktop', 'desktop', LucideIcons.monitor),
      ]),
      const _NavGroup('observe', 'Observe', LucideIcons.activity, [
        _NavChild('Overview',    'observe',     LucideIcons.layoutDashboard),
        _NavChild('Errors',      'errors',      LucideIcons.alertTriangle),
        _NavChild('Releases',    'releases',    LucideIcons.tag),
        _NavChild('Logs',        'logs',        LucideIcons.terminal),
        _NavChild('Replays',     'replays',     LucideIcons.video),
        _NavChild('Uptime',      'uptime',      LucideIcons.heartPulse),
        _NavChild('Crons',       'crons',       LucideIcons.clock),
        _NavChild('Alerts',      'alerts',      LucideIcons.bell),
      ]),
      const _NavGroup('settings', 'Settings', LucideIcons.settings, [
        _NavChild('General', 'settings', LucideIcons.settings),
        _NavChild('Team', 'settings', LucideIcons.users),
        _NavChild('Vault', 'vault', LucideIcons.shieldCheck),
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
    // Set the project header immediately so providers that fire during this
    // build frame already have the correct header on their requests.
    ref.read(apiClientProvider).setProject(projectId);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      ref.read(currentProjectProvider.notifier).state = projectId;
    });
  }

  void _openSearch(BuildContext context, String? projectId) {
    final projects = ref.read(projectsProvider).valueOrNull ?? [];
    final orgs = ref.read(orgsProvider).valueOrNull ?? [];
    showDialog(
      context: context,
      barrierColor: Colors.black.withValues(alpha: 0.65),
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

  void _showGroupSheet(BuildContext context, _NavGroup group, String? projectId, String currentPath) {
    showModalBottomSheet(
      context: context,
      backgroundColor: _panelSurface(context),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => _GroupBottomSheet(
        group: group,
        projectId: projectId ?? '',
        currentPath: currentPath,
        onNavigate: (path) {
          Navigator.pop(ctx);
          context.go(path);
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
      'functions': 'build',
      'messaging': 'build',
      'content': 'build',
      'workflows': 'build',
      'flags': 'build',
      'realtime': 'build',
      'platforms': 'platforms',

      'sites': 'platforms',
      'containers': 'platforms',
      'mobile': 'platforms',
      'desktop': 'platforms',
      'vault': 'settings',
      'settings': 'settings',
      'observe':     'observe',
      'errors':      'observe',
      'logs':        'observe',
      'releases':    'observe',
      'replays':     'observe',
      'uptime':      'observe',
      'crons':       'observe',
      'alerts':      'observe',
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
    final groups = _buildGroups();
    final activeGroup = _activeGroup(currentPath, projectId ?? '');

    final isMobile = _isMobile(context);

    void handleGroupTap(String id) {
      const directRoutes = {'overview': 'overview', 'platforms': 'platforms', 'settings': 'settings'};
      if (isMobile) {
        if (directRoutes.containsKey(id)) {
          if (projectId != null) context.go('/project/$projectId/${directRoutes[id]}');
          return;
        }
        final group = groups.firstWhere((g) => g.id == id, orElse: () => groups.first);
        if (group.children.isEmpty) {
          if (projectId != null) context.go('/project/$projectId/${group.id}');
          return;
        }
        _showGroupSheet(context, group, projectId, currentPath);
        return;
      }
      // Desktop behaviour
      if (directRoutes.containsKey(id)) {
        setState(() => _expandedGroup = null);
        if (projectId != null) context.go('/project/$projectId/${directRoutes[id]}');
        return;
      }
      setState(() {
        if (id == 'ai') { _aiChatOpen = !_aiChatOpen; return; }
        if (_expandedGroup == id) {
          _expandedGroup = null;
        } else {
          _expandedGroup = id;
          if (projectId != null) {
            final group = groups.firstWhere((g) => g.id == id, orElse: () => groups.first);
            final firstChild = group.children.cast<_NavChild?>().firstWhere(
                (c) => c != null && !c.placeholder, orElse: () => null);
            if (firstChild != null) context.go('/project/$projectId/${firstChild.route}');
          }
        }
      });
    }

    // ── Mobile layout ──────────────────────────────────────────────────────────
    if (isMobile) {
      return Shortcuts(
        shortcuts: <LogicalKeySet, Intent>{
          LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.keyK): const _OpenSearchIntent(),
          LogicalKeySet(LogicalKeyboardKey.control, LogicalKeyboardKey.keyK): const _OpenSearchIntent(),
        },
        child: Actions(
          actions: <Type, Action<Intent>>{
            _OpenSearchIntent: CallbackAction<_OpenSearchIntent>(
              onInvoke: (_) { _openSearch(context, projectId); return null; },
            ),
          },
          child: Focus(
            autofocus: true,
            child: Scaffold(
              backgroundColor: _shellBg(context),
              bottomNavigationBar: _BottomNav(
                groups: groups,
                activeGroup: activeGroup,
                projectId: projectId,
                onGroupTap: handleGroupTap,
              ),
              body: Column(
                children: [
                  _TopNavBar(
                    projectId: projectId,
                    currentPath: currentPath,
                    onSearchTap: () => _openSearch(context, projectId),
                    compact: true,
                  ),
                  Expanded(child: widget.child),
                ],
              ),
            ),
          ),
        ),
      );
    }

    // ── Desktop layout ─────────────────────────────────────────────────────────
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
            backgroundColor: _shellBg(context),
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
                        onGroupTap: handleGroupTap,
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
                          onClose: () => setState(() => _expandedGroup = null),
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
                          onClose: () => setState(() => _aiChatOpen = false),
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
        color: _railSurface(context),
        border: Border(right: BorderSide(color: _dividerColor(context))),
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
              customIcon: ClipOval(
                child: Image.asset(
                  'assets/applad-mascot-head.png',
                  width: 22,
                  height: 22,
                  fit: BoxFit.cover,
                ),
              ),
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
  final Widget? customIcon;

  const _RailIcon({
    required this.icon,
    required this.tooltip,
    required this.isActive,
    required this.isExpanded,
    required this.onTap,
    this.accentColor,
    this.customIcon,
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
    final activeFill = _activeFillColor(context);
    final hoverFill = _hoverFillColor(context);
    final activeIconColor = _isLight(context) ? accentColor : Colors.white;
    final hoverIconColor = _isLight(context)
      ? _primaryTextColor(context).withValues(alpha: 0.72)
      : Colors.white.withValues(alpha: 0.65);
    final idleIconColor = _isLight(context)
      ? _primaryTextColor(context).withValues(alpha: 0.4)
      : Colors.white.withValues(alpha: 0.28);

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
                          ? activeFill
                          : _hovered
                              ? hoverFill
                              : Colors.transparent,
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: widget.customIcon != null
                        ? widget.customIcon!
                        : Icon(
                            widget.icon,
                            size: 18,
                            color: active
                                ? activeIconColor
                                : _hovered
                                    ? hoverIconColor
                                    : idleIconColor,
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
    // Permanently dismissed via localStorage → hide the ring entirely.
    final done = ref.watch(getStartedDoneProvider(projectId));
    if (done) return const SizedBox.shrink();

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
                        decoration: const BoxDecoration(
                          color: _accent,
                          borderRadius: BorderRadius.only(
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
                            color: Colors.white.withValues(alpha: 0.08),
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
                                ? _primaryTextColor(context)
                                : _secondaryTextColor(context),
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
// Bottom Navigation (mobile, <650px)
// ═════════════════════════════════════════════════════════════════════════════

class _BottomNav extends StatelessWidget {
  final List<_NavGroup> groups;
  final String activeGroup;
  final String? projectId;
  final ValueChanged<String> onGroupTap;

  const _BottomNav({
    required this.groups,
    required this.activeGroup,
    required this.projectId,
    required this.onGroupTap,
  });

  @override
  Widget build(BuildContext context) {
    // Keep only groups meaningful for bottom nav (non-experimental visible ones)
    // Pin-bottom groups (settings) go last; rest in order; cap at 5
    final regular = groups.where((g) => !g.pinBottom).toList();
    final pinned = groups.where((g) => g.pinBottom).toList();
    final all = [...regular, ...pinned];
    final visible = all.length > 5 ? all.sublist(0, 4) + [all.last] : all;

    final bottomPad = MediaQuery.paddingOf(context).bottom;

    return Container(
      decoration: BoxDecoration(
        color: _railSurface(context),
        border: Border(top: BorderSide(color: _dividerColor(context))),
      ),
      padding: EdgeInsets.only(bottom: bottomPad),
      height: 56 + bottomPad,
      child: Row(
        children: visible
            .map((g) => _BottomNavItem(
                  icon: g.icon,
                  label: g.label,
                  isActive: activeGroup == g.id,
                  onTap: () => onGroupTap(g.id),
                ))
            .toList(),
      ),
    );
  }
}

class _BottomNavItem extends StatelessWidget {
  final IconData icon;
  final String label;
  final bool isActive;
  final VoidCallback onTap;

  const _BottomNavItem({
    required this.icon,
    required this.label,
    required this.isActive,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final active = isActive;
    final iconColor = active
        ? (_isLight(context) ? _accent : Colors.white)
        : (_isLight(context)
            ? Colors.black.withValues(alpha: 0.4)
            : Colors.white.withValues(alpha: 0.35));
    final labelColor = active
        ? (_isLight(context) ? _accent : Colors.white)
        : (_isLight(context)
            ? Colors.black.withValues(alpha: 0.4)
            : Colors.white.withValues(alpha: 0.35));

    return Expanded(
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, size: 20, color: iconColor),
            const SizedBox(height: 3),
            Text(
              label,
              style: TextStyle(
                fontSize: 10,
                color: labelColor,
                fontWeight: active ? FontWeight.w600 : FontWeight.w400,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ],
        ),
      ),
    );
  }
}

// ── Group bottom sheet (mobile) ───────────────────────────────────────────────

class _GroupBottomSheet extends StatelessWidget {
  final _NavGroup group;
  final String projectId;
  final String currentPath;
  final ValueChanged<String> onNavigate;

  const _GroupBottomSheet({
    required this.group,
    required this.projectId,
    required this.currentPath,
    required this.onNavigate,
  });

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Handle
          Center(
            child: Container(
              margin: const EdgeInsets.only(top: 12, bottom: 8),
              width: 36,
              height: 4,
              decoration: BoxDecoration(
                color: _dividerColor(context),
                borderRadius: BorderRadius.circular(2),
              ),
            ),
          ),
          // Title
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 4, 20, 12),
            child: Text(
              group.label,
              style: TextStyle(
                color: _primaryTextColor(context),
                fontSize: 15,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
          // Items
          ...group.children.map((child) {
            final path = '/project/$projectId/${child.route}';
            final active = currentPath.startsWith(path);
            return ListTile(
              dense: true,
              contentPadding: const EdgeInsets.symmetric(horizontal: 20, vertical: 0),
              leading: Icon(
                child.icon,
                size: 18,
                color: child.placeholder
                    ? _mutedTextColor(context)
                    : active
                        ? (_isLight(context) ? _accent : Colors.white)
                        : _secondaryTextColor(context),
              ),
              title: Text(
                child.label,
                style: TextStyle(
                  color: child.placeholder
                      ? _mutedTextColor(context)
                      : active
                          ? _primaryTextColor(context)
                          : _secondaryTextColor(context),
                  fontSize: 14,
                  fontWeight: active ? FontWeight.w500 : FontWeight.w400,
                ),
              ),
              trailing: child.placeholder
                  ? Container(
                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(
                        color: _hoverFillColor(context),
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text('Soon',
                          style: TextStyle(color: _mutedTextColor(context), fontSize: 10)),
                    )
                  : null,
              onTap: child.placeholder ? null : () => onNavigate(path),
              tileColor: active ? _activeFillColor(context) : null,
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            );
          }),
          const SizedBox(height: 12),
        ],
      ),
    );
  }
}

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
        color: _panelSurface(context),
        border: Border(right: BorderSide(color: _dividerColor(context))),
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
                  style: TextStyle(
                    color: _primaryTextColor(context),
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const Spacer(),
                GestureDetector(
                  onTap: onClose,
                  child: Icon(LucideIcons.panelLeftClose,
                      size: 16,
                      color: _mutedTextColor(context)),
                ),
              ],
            ),
          ),
          Container(
            height: 1,
            margin: const EdgeInsets.symmetric(horizontal: 12),
            color: _dividerColor(context),
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
    final activeFill = _activeFillColor(context);
    final hoverFill = _hoverFillColor(context);
    final placeholderIconColor = _isLight(context)
      ? _primaryTextColor(context).withValues(alpha: 0.25)
      : Colors.white.withValues(alpha: 0.15);
    final activeIconColor = _isLight(context)
      ? _accent
      : Colors.white.withValues(alpha: 0.9);
    final idleIconColor = _isLight(context)
      ? _primaryTextColor(context).withValues(alpha: 0.45)
      : Colors.white.withValues(alpha: 0.4);
    final placeholderTextColor = _isLight(context)
      ? _primaryTextColor(context).withValues(alpha: 0.32)
      : Colors.white.withValues(alpha: 0.2);
    final activeTextColor = _isLight(context)
      ? _primaryTextColor(context)
      : Colors.white.withValues(alpha: 0.9);
    final idleTextColor = _secondaryTextColor(context);

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
                ? activeFill
                : _hovered && !widget.placeholder
                    ? hoverFill
                    : Colors.transparent,
            borderRadius: BorderRadius.circular(7),
          ),
          child: Row(
            children: [
              Icon(
                widget.icon,
                size: 15,
                color: widget.placeholder
                    ? placeholderIconColor
                    : widget.active
                        ? activeIconColor
                        : idleIconColor,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  widget.label,
                  style: TextStyle(
                    color: widget.placeholder
                        ? placeholderTextColor
                        : widget.active
                            ? activeTextColor
                            : idleTextColor,
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
                    color: _hoverFillColor(context),
                    borderRadius: BorderRadius.circular(3),
                  ),
                  child: Text('Soon',
                      style: TextStyle(
                          color: placeholderTextColor,
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
        color: _panelSurface(context),
        border: Border(left: BorderSide(color: _dividerColor(context))),
      ),
      child: Column(
        children: [
          // Header
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 8, 12),
            child: Row(
              children: [
                ClipOval(
                  child: Image.asset(
                    'assets/applad-mascot-head.png',
                    width: 22,
                    height: 22,
                    fit: BoxFit.cover,
                  ),
                ),
                const SizedBox(width: 8),
                Text('AI Assistant',
                  style: TextStyle(
                    color: _primaryTextColor(context),
                        fontSize: 14,
                        fontWeight: FontWeight.w600)),
                const Spacer(),
                GestureDetector(
                  onTap: widget.onClose,
                  child: Icon(LucideIcons.x,
                      size: 16,
                      color: _mutedTextColor(context)),
                ),
              ],
            ),
          ),
          Container(height: 1, color: _dividerColor(context)),

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
                                  color: _mutedTextColor(context)),
                            ),
                            const SizedBox(width: 8),
                            Text('Thinking...',
                                style: TextStyle(
                                  color: _mutedTextColor(context),
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
                                    ? _accent.withValues(alpha: 0.15)
                                    : const Color(0xFF8B5CF6)
                                        .withValues(alpha: 0.15),
                                borderRadius: BorderRadius.circular(6),
                              ),
                              child: isUser
                                  ? const Icon(LucideIcons.user, size: 12, color: _accent)
                                  : ClipOval(
                                      child: Image.asset(
                                        'assets/applad-mascot-head.png',
                                        width: 24,
                                        height: 24,
                                        fit: BoxFit.cover,
                                      ),
                                    ),
                            ),
                            const SizedBox(width: 10),
                            Expanded(
                              child: Text(
                                msg['text'] ?? '',
                                style: TextStyle(
                                  color: (isUser
                                          ? _primaryTextColor(context)
                                          : _secondaryTextColor(context))
                                      .withValues(alpha: isUser ? 0.92 : 0.88),
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
                  top: BorderSide(color: _dividerColor(context))),
            ),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _msgCtrl,
                    style: TextStyle(
                        color: _primaryTextColor(context), fontSize: 13),
                    decoration: InputDecoration(
                      hintText: 'Ask anything...',
                      hintStyle: TextStyle(
                          color: _placeholderTextColor(context),
                          fontSize: 13),
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(
                            color: _dividerColor(context)),
                      ),
                      enabledBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(
                            color: _dividerColor(context)),
                      ),
                      focusedBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide:
                            const BorderSide(color: Color(0xFF8B5CF6)),
                      ),
                      filled: true,
                      fillColor: _inputFillColor(context),
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
            ClipRRect(
              borderRadius: BorderRadius.circular(14),
              child: Image.asset(
                'assets/applad-mascot-head.png',
                width: 56,
                height: 56,
                fit: BoxFit.cover,
              ),
            ),
            const SizedBox(height: 20),
            Text('AI Assistant',
              style: TextStyle(
                color: _primaryTextColor(context),
                    fontSize: 16,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 8),
            Text(
              'Create apps, manage databases, deploy services, and automate workflows — all through conversation.',
              textAlign: TextAlign.center,
              style: TextStyle(
                color: _secondaryTextColor(context),
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
          color: _hoverFillColor(context),
          borderRadius: BorderRadius.circular(20),
          border: Border.all(color: _dividerColor(context)),
        ),
        child: Text(text,
            style: TextStyle(
            color: _secondaryTextColor(context),
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
  final bool compact;

  const _TopNavBar({
    required this.projectId,
    required this.currentPath,
    required this.onSearchTap,
    this.compact = false,
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
        color: _shellBg(context),
        border: Border(bottom: BorderSide(color: _dividerColor(context))),
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
          _sep(context),
          const SizedBox(width: 10),

          _OrgNavButton(orgName: orgName, orgs: orgs, ref: ref),

          if (projectId != null) ...[
            const SizedBox(width: 10),
            _sep(context),
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

          LayoutBuilder(
            builder: (ctx, constraints) {
              // constraints.maxWidth is the remaining space after the Spacer,
              // but since we're inside a Row we need the full nav width instead.
              final navWidth = MediaQuery.of(ctx).size.width;
              // Collapse Feedback + Support below 780 px
              if (navWidth < 780) {
                return _NavOverflowMenu();
              }
              return const Row(mainAxisSize: MainAxisSize.min, children: [
                FeedbackButton(),
                SizedBox(width: 2),
                SupportButton(),
                SizedBox(width: 2),
              ]);
            },
          ),
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
                    color: _iconColor(context)),
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

  Widget _sep(BuildContext context) => Text(
        '/',
        style: TextStyle(
          color: _isLight(context)
              ? Colors.black.withValues(alpha: 0.18)
              : Colors.white.withValues(alpha: 0.18),
          fontSize: 18,
          fontWeight: FontWeight.w300,
        ),
      );

  String _short(String id) => id.length > 8 ? id.substring(0, 8) : id;

}

// ── Overflow menu (collapses Feedback + Support on narrow screens) ─────────────

class _NavOverflowMenu extends StatefulWidget {
  @override
  State<_NavOverflowMenu> createState() => _NavOverflowMenuState();
}

class _NavOverflowMenuState extends State<_NavOverflowMenu> {
  final _link = LayerLink();
  OverlayEntry? _overlay;

  void _toggle(BuildContext context) {
    if (_overlay != null) {
      _close();
      return;
    }

    final box = context.findRenderObject() as RenderBox;
    final offset = box.localToGlobal(Offset.zero);
    final size = box.size;

    _overlay = OverlayEntry(
      builder: (_) => Stack(
        children: [
          Positioned.fill(
            child: GestureDetector(
              behavior: HitTestBehavior.opaque,
              onTap: _close,
            ),
          ),
          Positioned(
            top: offset.dy + size.height + 4,
            right: MediaQuery.of(context).size.width - offset.dx - size.width,
            child: CompositedTransformFollower(
              link: _link,
              showWhenUnlinked: false,
              offset: Offset(0, size.height + 4),
              child: _OverflowMenuPopup(onClose: _close),
            ),
          ),
        ],
      ),
    );
    Overlay.of(context).insert(_overlay!);
    setState(() {});
  }

  void _close() {
    _overlay?.remove();
    _overlay = null;
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return CompositedTransformTarget(
      link: _link,
      child: Tooltip(
        message: 'More',
        child: InkWell(
          onTap: () => _toggle(context),
          borderRadius: BorderRadius.circular(8),
          child: SizedBox(
            width: 34,
            height: 34,
            child: Icon(
              LucideIcons.moreHorizontal,
              size: 17,
              color: _iconColor(context),
            ),
          ),
        ),
      ),
    );
  }
}

class _OverflowMenuPopup extends StatelessWidget {
  final VoidCallback onClose;
  const _OverflowMenuPopup({required this.onClose});

  @override
  Widget build(BuildContext context) {
    final bg = _isLight(context) ? Colors.white : const Color(0xFF1C1D22);
    final border = _isLight(context)
        ? const Color(0xFFE4E4E7)
        : const Color(0xFF2A2B31);

    return Material(
      color: Colors.transparent,
      child: Container(
        width: 160,
        decoration: BoxDecoration(
          color: bg,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: border),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.18),
              blurRadius: 16,
              offset: const Offset(0, 4),
            ),
          ],
        ),
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            _OverflowMenuItem(
              icon: LucideIcons.messageSquare,
              label: 'Feedback',
              onTap: () {
                onClose();
                showFeedbackPanel(context);
              },
            ),
            _OverflowMenuItem(
              icon: LucideIcons.lifeBuoy,
              label: 'Support',
              onTap: () {
                onClose();
                showSupportPanel(context);
              },
            ),
          ],
        ),
      ),
    );
  }
}

class _OverflowMenuItem extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _OverflowMenuItem({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final textColor = _isLight(context)
        ? const Color(0xFF3F3F46)
        : const Color(0xFFA1A1AA);

    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
        child: Row(children: [
          Icon(icon, size: 14, color: textColor),
          const SizedBox(width: 10),
          Text(label,
              style: TextStyle(
                  color: textColor,
                  fontSize: 13,
                  fontWeight: FontWeight.w400)),
        ]),
      ),
    );
  }
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
    final popupSurface = _popupSurface(context);
    final activeText = _primaryTextColor(context);
    final inactiveText = _secondaryTextColor(context);

    return PopupMenuButton<String>(
      offset: const Offset(0, 36),
      color: popupSurface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: BorderSide(color: _dividerColor(context)),
      ),
      onSelected: (id) => context.go('/project/$id/${_section()}'),
      itemBuilder: (_) => projects
          .map((p) => PopupMenuItem<String>(
                value: p['\$id'] as String,
                child: Text(
                  p['name'] ?? 'Unnamed',
                  style: TextStyle(
                    color: p['\$id'] == projectId
                        ? activeText
                        : inactiveText,
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
              style: TextStyle(
                  color: _primaryTextColor(context),
                  fontSize: 13,
                  fontWeight: FontWeight.w500)),
          const SizedBox(width: 4),
          Icon(LucideIcons.chevronDown,
              size: 14, color: _mutedTextColor(context)),
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
    final popupSurface = _popupSurface(context);

    return PopupMenuButton<String>(
      offset: const Offset(0, 36),
      color: popupSurface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: BorderSide(color: _dividerColor(context)),
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
                style: TextStyle(color: _secondaryTextColor(context), fontSize: 13)),
          ),
      ],
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(orgName,
              style: TextStyle(
                  color: _primaryTextColor(context),
                  fontSize: 13,
                  fontWeight: FontWeight.w500)),
          const SizedBox(width: 4),
          Icon(LucideIcons.chevronDown,
              size: 14, color: _mutedTextColor(context)),
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

    final projectId = ref.read(currentProjectProvider);
    return PopupMenuButton<String>(
      offset: const Offset(0, 36),
      color: _popupSurface(context),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: BorderSide(color: _dividerColor(context)),
      ),
      onSelected: (value) {
        if (value == '__manage__') {
          if (projectId != null) context.go('/project/$projectId/environments');
        } else {
          ref.read(currentEnvironmentProvider.notifier).state = value;
        }
      },
      itemBuilder: (_) => [
        ...envs.map((e) {
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
                color: currentEnvId == id ? _primaryTextColor(context) : _secondaryTextColor(context), fontSize: 13,
                fontWeight: currentEnvId == id ? FontWeight.w600 : FontWeight.w400)),
            ]),
          );
        }),
        const PopupMenuDivider(),
        PopupMenuItem<String>(
          value: '__manage__',
          child: Row(children: [
            Icon(LucideIcons.settings2, size: 13, color: _secondaryTextColor(context)),
            const SizedBox(width: 8),
            Text('Manage environments', style: TextStyle(color: _secondaryTextColor(context), fontSize: 13)),
          ]),
        ),
      ],
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: envColor.withValues(alpha: 0.1),
          borderRadius: BorderRadius.circular(6),
          border: Border.all(color: envColor.withValues(alpha: 0.25)),
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
