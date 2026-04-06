import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../providers/auth_provider.dart';
import '../providers/project_provider.dart';
import '../providers/org_provider.dart';
import '../api/client.dart';
import '../widgets/search_modal.dart';
import '../widgets/app_dialog.dart';
import '../widgets/navbar_popovers.dart';
import '../widgets/console_footer.dart';

// ── Search intent ─────────────────────────────────────────────────────────────

class _OpenSearchIntent extends Intent {
  const _OpenSearchIntent();
}

// ── Shell ─────────────────────────────────────────────────────────────────────

class AppShell extends ConsumerStatefulWidget {
  final Widget child;
  const AppShell({super.key, required this.child});

  @override
  ConsumerState<AppShell> createState() => _AppShellState();
}

class _AppShellState extends ConsumerState<AppShell> {
  String? _syncedProjectId;

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
      content: AppDialogField(controller: nameCtrl, label: 'Organization name', hint: 'My org', autofocus: true),
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
            if (context.mounted) Navigator.pop(context);
          },
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final routerState = GoRouterState.of(context);
    final projectId = routerState.pathParameters['projectId'];
    final currentPath = routerState.uri.path;

    if (projectId != null) _syncProject(projectId);

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
            backgroundColor: const Color(0xFF0B0B0F),
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
                      _Sidebar(
                          currentPath: currentPath,
                          projectId: projectId ?? ''),
                      Expanded(child: widget.child),
                    ],
                  ),
                ),
                const ConsoleFooter(),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

// ── Top navigation bar ────────────────────────────────────────────────────────

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

    String orgName = 'Personal';
    if (currentOrgId != null) {
      final org = orgs.where((o) => o['\$id'] == currentOrgId).firstOrNull;
      if (org != null) orgName = org['name'] ?? 'Organization';
    } else if (orgs.isNotEmpty) {
      orgName = orgs.first['name'] ?? 'Organization';
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
        color: const Color(0xFF0B0B0F),
        border: Border(
            bottom: BorderSide(color: Colors.white.withOpacity(0.06))),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          // App logo mark → /projects
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

          // Org breadcrumb
          _OrgNavButton(orgName: orgName, orgs: orgs, ref: ref),

          // Project breadcrumb (only when inside a project)
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
          ],

          const Spacer(),

          const FeedbackButton(),
          const SizedBox(width: 2),
          const SupportButton(),
          const SizedBox(width: 2),
          const ThemeToggleButton(),
          const SizedBox(width: 4),

          // Search
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

          // User avatar
          Consumer(builder: (context, ref, _) {
            final user = ref.watch(consoleAuthProvider).valueOrNull;
            if (user == null) return const SizedBox.shrink();
            final initials = _initials(user.name, user.email);
            return GestureDetector(
              onTap: () => context.go('/account'),
              child: Tooltip(
                message: user.email,
                child: Container(
                  width: 32,
                  height: 32,
                  decoration: const BoxDecoration(
                    color: Color(0xFF3472A4),
                    shape: BoxShape.circle,
                  ),
                  child: Center(
                    child: Text(
                      initials,
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ),
              ),
            );
          }),

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

  String _short(String id) =>
      id.length > 8 ? id.substring(0, 8) : id;

  String _initials(String name, String email) {
    if (name.isNotEmpty) {
      final parts = name.trim().split(' ');
      if (parts.length >= 2) {
        return '${parts[0][0]}${parts[1][0]}'.toUpperCase();
      }
      return name[0].toUpperCase();
    }
    if (email.isNotEmpty) return email[0].toUpperCase();
    return '?';
  }
}

// ── Project breadcrumb dropdown ───────────────────────────────────────────────

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
      onSelected: (id) {
        context.go('/project/$id/${_section()}');
      },
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
          Text(
            projectName,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 13,
              fontWeight: FontWeight.w500,
            ),
          ),
          const SizedBox(width: 4),
          Icon(LucideIcons.chevronDown,
              size: 14, color: Colors.white.withOpacity(0.35)),
        ],
      ),
    );
  }
}

// ── Org nav button ────────────────────────────────────────────────────────────

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
          Text(
            orgName,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 13,
              fontWeight: FontWeight.w500,
            ),
          ),
          const SizedBox(width: 4),
          Icon(LucideIcons.chevronDown,
              size: 14, color: Colors.white.withOpacity(0.35)),
        ],
      ),
    );
  }
}

// ── Sidebar ───────────────────────────────────────────────────────────────────

class _Sidebar extends ConsumerWidget {
  final String currentPath;
  final String projectId;
  const _Sidebar({required this.currentPath, required this.projectId});

  String get _base => '/project/$projectId';

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final orgsAsync = ref.watch(orgsProvider);
    final currentOrg = ref.watch(currentOrgProvider);

    return Container(
      width: 216,
      decoration: BoxDecoration(
        color: const Color(0xFF0B0B0F),
        border: Border(
            right: BorderSide(color: Colors.white.withOpacity(0.06))),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SizedBox(height: 8),

          _OrgDropdown(
            orgs: orgsAsync.valueOrNull ?? [],
            currentOrg: currentOrg,
            onSelect: (id) =>
                ref.read(currentOrgProvider.notifier).state = id,
            onCreateOrg: () => _showCreateOrgDialog(context, ref),
          ),

          const SizedBox(height: 6),
          _divider(),

          _NavItem(
            icon: LucideIcons.checkCircle,
            label: 'Get started',
            path: '$_base/overview',
            currentPath: currentPath,
          ),

          const SizedBox(height: 4),

          _NavItem(
            icon: LucideIcons.barChart3,
            label: 'Overview',
            path: '$_base/overview',
            currentPath: currentPath,
            exactMatch: true,
          ),

          const SizedBox(height: 10),
          _sectionHeader('BUILD'),

          _NavItem(
            icon: LucideIcons.users,
            label: 'Auth',
            path: '$_base/auth',
            currentPath: currentPath,
          ),
          _NavItem(
            icon: LucideIcons.database,
            label: 'Databases',
            path: '$_base/databases',
            currentPath: currentPath,
          ),
          _NavItem(
            icon: LucideIcons.zap,
            label: 'Functions',
            path: '$_base/functions',
            currentPath: currentPath,
          ),
          _NavItem(
            icon: LucideIcons.messageSquare,
            label: 'Messaging',
            path: '$_base/messaging',
            currentPath: currentPath,
          ),
          _NavItem(
            icon: LucideIcons.folderClosed,
            label: 'Storage',
            path: '$_base/storage',
            currentPath: currentPath,
          ),

          const SizedBox(height: 10),
          _sectionHeader('DEPLOY'),

          _NavItem(
            icon: LucideIcons.rocket,
            label: 'Deploy',
            path: '$_base/deploy',
            currentPath: currentPath,
          ),

          const SizedBox(height: 10),
          _sectionHeader('WORKFLOWS'),

          _NavItem(
            icon: LucideIcons.gitBranch,
            label: 'Workflows',
            path: '$_base/workflows',
            currentPath: currentPath,
          ),

          const SizedBox(height: 10),
          _sectionHeader('FEATURE FLAGS'),

          _NavItem(
            icon: LucideIcons.toggleRight,
            label: 'Flags',
            path: '$_base/flags',
            currentPath: currentPath,
          ),

          const Spacer(),
          _divider(),

          _NavItem(
            icon: LucideIcons.settings,
            label: 'Settings',
            path: '$_base/settings',
            currentPath: currentPath,
          ),
          const SizedBox(height: 10),
        ],
      ),
    );
  }

  Widget _sectionHeader(String title) {
    return Padding(
      padding: const EdgeInsets.only(left: 20, bottom: 4, top: 2),
      child: Text(
        title,
        style: TextStyle(
          color: Colors.white.withOpacity(0.22),
          fontSize: 10,
          fontWeight: FontWeight.w600,
          letterSpacing: 1.2,
        ),
      ),
    );
  }

  Widget _divider() {
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 14, vertical: 4),
      height: 1,
      color: Colors.white.withOpacity(0.06),
    );
  }

  void _showCreateOrgDialog(BuildContext context, WidgetRef ref) {
    final nameCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create organization',
      subtitle: 'Organize your projects into teams',
      content: AppDialogField(controller: nameCtrl, label: 'Organization name', hint: 'My org', autofocus: true),
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
            if (context.mounted) Navigator.pop(context);
          },
        ),
      ],
    );
  }
}

// ── Organization dropdown (sidebar) ──────────────────────────────────────────

class _OrgDropdown extends StatelessWidget {
  final List<Map<String, dynamic>> orgs;
  final String? currentOrg;
  final ValueChanged<String> onSelect;
  final VoidCallback onCreateOrg;

  const _OrgDropdown({
    required this.orgs,
    required this.currentOrg,
    required this.onSelect,
    required this.onCreateOrg,
  });

  String get _displayName {
    if (currentOrg != null) {
      final org = orgs.where((o) => o['\$id'] == currentOrg).firstOrNull;
      if (org != null) return org['name'] ?? 'Organization';
    }
    if (orgs.isNotEmpty) return orgs.first['name'] ?? 'Organization';
    return 'Personal';
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 10),
      child: PopupMenuButton<String>(
        offset: const Offset(0, 40),
        color: const Color(0xFF1A1A22),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
          side: BorderSide(color: Colors.white.withOpacity(0.08)),
        ),
        onSelected: (value) {
          if (value == '__create__') {
            onCreateOrg();
          } else {
            onSelect(value);
          }
        },
        itemBuilder: (_) {
          final items = <PopupMenuEntry<String>>[];
          for (final org in orgs) {
            items.add(PopupMenuItem<String>(
              value: org['\$id'] as String,
              child: Text(org['name'] ?? 'Unnamed',
                  style:
                      const TextStyle(color: Colors.white70, fontSize: 13)),
            ));
          }
          if (orgs.isNotEmpty) items.add(const PopupMenuDivider());
          items.add(const PopupMenuItem<String>(
            value: '__create__',
            child: Row(
              children: [
                Icon(LucideIcons.plus, size: 15, color: Colors.white54),
                SizedBox(width: 8),
                Text('Create organization',
                    style:
                        TextStyle(color: Colors.white70, fontSize: 13)),
              ],
            ),
          ));
          return items;
        },
        child: Container(
          width: double.infinity,
          padding:
              const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
          decoration: BoxDecoration(
            color: Colors.white.withOpacity(0.04),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              Container(
                width: 26,
                height: 26,
                decoration: BoxDecoration(
                  color: const Color(0xFF3472A4).withOpacity(0.18),
                  borderRadius: BorderRadius.circular(5),
                ),
                child: Center(
                  child: Text(
                    _displayName.isNotEmpty
                        ? _displayName[0].toUpperCase()
                        : 'P',
                    style: const TextStyle(
                      color: Color(0xFF3472A4),
                      fontWeight: FontWeight.w600,
                      fontSize: 12,
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 9),
              Expanded(
                child: Text(
                  _displayName,
                  style: const TextStyle(
                    color: Colors.white70,
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              Icon(LucideIcons.chevronDown,
                  size: 14, color: Colors.white.withOpacity(0.28)),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Nav item ──────────────────────────────────────────────────────────────────

class _NavItem extends StatefulWidget {
  final IconData icon;
  final String label;
  final String path;
  final String currentPath;
  final bool exactMatch;

  const _NavItem({
    required this.icon,
    required this.label,
    required this.path,
    required this.currentPath,
    this.exactMatch = false,
  });

  @override
  State<_NavItem> createState() => _NavItemState();
}

class _NavItemState extends State<_NavItem> {
  bool _hovered = false;

  bool get _isActive {
    if (widget.exactMatch) return widget.currentPath == widget.path;
    return widget.currentPath.startsWith(widget.path);
  }

  @override
  Widget build(BuildContext context) {
    final active = _isActive;
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: () => context.go(widget.path),
        child: Container(
          width: double.infinity,
          margin: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
          padding:
              const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
          decoration: BoxDecoration(
            color: active
                ? Colors.white.withOpacity(0.08)
                : _hovered
                    ? Colors.white.withOpacity(0.04)
                    : Colors.transparent,
            borderRadius: BorderRadius.circular(7),
          ),
          child: Row(
            children: [
              Icon(
                widget.icon,
                size: 16,
                color: active
                    ? Colors.white.withOpacity(0.9)
                    : Colors.white.withOpacity(0.35),
              ),
              const SizedBox(width: 10),
              Text(
                widget.label,
                style: TextStyle(
                  color: active
                      ? Colors.white.withOpacity(0.9)
                      : Colors.white.withOpacity(0.5),
                  fontSize: 13,
                  fontWeight:
                      active ? FontWeight.w500 : FontWeight.w400,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
