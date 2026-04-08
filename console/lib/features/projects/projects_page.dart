import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/auth_provider.dart';
import '../../core/providers/org_provider.dart';
import '../../core/providers/project_provider.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/search_list.dart';
import '../../core/widgets/search_modal.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/navbar_popovers.dart';
import '../../core/widgets/console_footer.dart';

// --- Constants ---------------------------------------------------------------

const _bgColor = Color(0xFF0B0B0F);
const _cardColor = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _border = Color(0x0DFFFFFF);
const _green = Color(0xFF10B981);
const _red = Color(0xFFEF4444);

// Deterministic card accent colors
const _cardColors = [
  Color(0xFF3472A4), Color(0xFF7C3AED), Color(0xFF059669),
  Color(0xFFD97706), Color(0xFFDC2626), Color(0xFF0891B2),
  Color(0xFF7C3AED), Color(0xFF6D28D9),
];

Color _projectColor(String seed) {
  var hash = 0;
  for (final c in seed.codeUnits) {
    hash = (hash * 31 + c) & 0xFFFFFFFF;
  }
  return _cardColors[hash % _cardColors.length];
}

// --- Providers ---------------------------------------------------------------

final _orgStatsProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, orgId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/organizations/$orgId/stats');
  return res.data as Map<String, dynamic>;
});

final _orgActivityProvider =
    FutureProvider.family<Map<String, dynamic>, String>((ref, orgId) async {
  final api = ref.read(apiClientProvider);
  final res = await api.get('/organizations/$orgId/activity',
      params: {'limit': 50});
  return res.data as Map<String, dynamic>;
});

// --- Page --------------------------------------------------------------------

class ProjectsPage extends ConsumerStatefulWidget {
  const ProjectsPage({super.key});

  @override
  ConsumerState<ProjectsPage> createState() => _ProjectsPageState();
}

class _ProjectsPageState extends ConsumerState<ProjectsPage> {
  int _tabIndex = 0;
  final _searchCtrl = TextEditingController();
  int _page = 1;
  int _perPage = 6;

  List<Map<String, dynamic>> _members = [];
  bool _membersLoading = false;

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  Future<void> _loadMembers() async {
    final orgId = ref.read(currentOrgProvider);
    if (orgId == null) return;
    setState(() => _membersLoading = true);
    try {
      final api = ref.read(apiClientProvider);
      final res = await api.get('/organizations/$orgId/members');
      setState(() {
        _members = List<Map<String, dynamic>>.from(
            (res.data as Map)['members'] ?? []);
      });
    } catch (_) {}
    if (mounted) setState(() => _membersLoading = false);
  }

  void _selectProject(Map<String, dynamic> project) {
    final id = project['\$id'] as String;
    ref.read(currentProjectProvider.notifier).state = id;
    ref.read(apiClientProvider).setProject(id);
    context.go('/project/$id/overview');
  }

  Future<void> _deleteProject(String id) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: _cardColor,
        title: const Text('Delete project',
            style: TextStyle(color: Colors.white)),
        content: const Text(
            'All data in this project will be permanently deleted.',
            style: TextStyle(color: _dimText)),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: const Text('Cancel')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: _red),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    try {
      await ref.read(apiClientProvider).delete('/projects/$id');
      ref.invalidate(projectsProvider);
    } catch (_) {}
  }

  void _showCreateProjectDialog() {
    final nameCtrl = TextEditingController();
    final descCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create project',
      subtitle: 'Start building something new',
      content: Column(mainAxisSize: MainAxisSize.min, children: [
        AppDialogField(
            controller: nameCtrl,
            label: 'Project name',
            hint: 'My project',
            autofocus: true),
        const SizedBox(height: 16),
        AppDialogField(
            controller: descCtrl,
            label: 'Description',
            hint: 'Optional description'),
      ]),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (nameCtrl.text.trim().isEmpty) return;
            try {
              final api = ref.read(apiClientProvider);
              final orgId = ref.read(currentOrgProvider);
              await api.post(
                orgId != null
                    ? '/organizations/$orgId/projects'
                    : '/projects',
                data: {
                  'name': nameCtrl.text.trim(),
                  'description': descCtrl.text.trim()
                },
              );
              ref.invalidate(projectsProvider);
            } catch (_) {}
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
          },
        ),
      ],
    );
  }

  void _showInviteMemberDialog() {
    final emailCtrl = TextEditingController();
    final nameCtrl = TextEditingController();
    String selectedRole = 'member';
    showAppDialog(
      context: context,
      title: 'Invite member',
      subtitle: 'Add a team member to this organization',
      content: StatefulBuilder(
        builder: (ctx, setDialogState) => Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AppDialogField(
                controller: emailCtrl,
                label: 'Email address',
                hint: 'name@example.com',
                autofocus: true),
            const SizedBox(height: 16),
            AppDialogField(
                controller: nameCtrl, label: 'Name', hint: 'Optional'),
            const SizedBox(height: 16),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Role',
                    style: TextStyle(
                        color: Colors.white.withValues(alpha: 0.5),
                        fontSize: 12,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 6),
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.symmetric(horizontal: 12),
                  decoration: BoxDecoration(
                    color: const Color(0x0AFFFFFF),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(
                        color: Colors.white.withValues(alpha: 0.1)),
                  ),
                  child: DropdownButtonHideUnderline(
                    child: DropdownButton<String>(
                      value: selectedRole,
                      isExpanded: true,
                      dropdownColor: const Color(0xFF1E1F24),
                      style: const TextStyle(
                          color: Colors.white, fontSize: 13),
                      icon: Icon(Icons.keyboard_arrow_down,
                          size: 16,
                          color: Colors.white.withValues(alpha: 0.4)),
                      items: const [
                        DropdownMenuItem(
                            value: 'owner', child: Text('Owner')),
                        DropdownMenuItem(
                            value: 'admin', child: Text('Admin')),
                        DropdownMenuItem(
                            value: 'member', child: Text('Member')),
                      ],
                      onChanged: (v) {
                        if (v != null) {
                          setDialogState(() => selectedRole = v);
                        }
                      },
                    ),
                  ),
                ),
                const SizedBox(height: 6),
                Text(
                  selectedRole == 'owner'
                      ? 'Full access to all resources and settings'
                      : selectedRole == 'admin'
                          ? 'Manage projects, members, and settings'
                          : 'Access assigned projects only',
                  style: TextStyle(
                      color: Colors.white.withValues(alpha: 0.3),
                      fontSize: 11),
                ),
              ],
            ),
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Invite',
          onTap: () async {
            if (emailCtrl.text.trim().isEmpty) return;
            try {
              final orgId = ref.read(currentOrgProvider);
              if (orgId == null) return;
              await ref.read(apiClientProvider).post(
                    '/organizations/$orgId/members',
                    data: {
                      'email': emailCtrl.text.trim(),
                      'name': nameCtrl.text.trim(),
                      'role': selectedRole,
                    },
                  );
              _loadMembers();
            } catch (_) {}
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
          },
        ),
      ],
    );
  }

  Future<void> _removeMember(String memberId) async {
    final orgId = ref.read(currentOrgProvider);
    if (orgId == null) return;
    try {
      await ref
          .read(apiClientProvider)
          .delete('/organizations/$orgId/members/$memberId');
      _loadMembers();
    } catch (_) {}
  }

  Future<void> _updateMemberRole(String memberId, String role) async {
    final orgId = ref.read(currentOrgProvider);
    if (orgId == null) return;
    try {
      await ref.read(apiClientProvider).patch(
            '/organizations/$orgId/members/$memberId',
            data: {'role': role},
          );
      _loadMembers();
    } catch (_) {}
  }

  void _showUpdateOrgDialog(String currentName) {
    final nameCtrl = TextEditingController(text: currentName);
    showAppDialog(
      context: context,
      title: 'Update organization',
      subtitle: 'Change the display name',
      content: AppDialogField(
          controller: nameCtrl,
          label: 'Organization name',
          hint: 'Name',
          autofocus: true),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Update',
          onTap: () async {
            try {
              final orgId = ref.read(currentOrgProvider);
              if (orgId == null) return;
              await ref.read(apiClientProvider).patch(
                    '/organizations/$orgId',
                    data: {'name': nameCtrl.text.trim()},
                  );
              ref.invalidate(orgsProvider);
            } catch (_) {}
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
          },
        ),
      ],
    );
  }

  Future<void> _deleteOrg() async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete organization',
      subtitle: 'This action is irreversible',
      content: Text(
        'All projects and data within this organization will be permanently deleted.',
        style: TextStyle(color: Colors.white.withValues(alpha: 0.5), fontSize: 13),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Delete',
          destructive: true,
          onTap: () => Navigator.pop(context, true),
        ),
      ],
    );
    if (confirmed == true) {
      try {
        final orgId = ref.read(currentOrgProvider);
        if (orgId == null) return;
        await ref.read(apiClientProvider).delete('/organizations/$orgId');
        ref.read(currentOrgProvider.notifier).state = null;
        ref.invalidate(orgsProvider);
      } catch (_) {}
    }
  }

  void _showCreateOrgDialog() {
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
            if (nameCtrl.text.trim().isEmpty) return;
            try {
              final api = ref.read(apiClientProvider);
              final res = await api.post('/organizations',
                  data: {'name': nameCtrl.text.trim()});
              final data = res.data as Map<String, dynamic>;
              final newId = data['\$id'] as String?;
              ref.invalidate(orgsProvider);
              if (newId != null) {
                ref.read(currentOrgProvider.notifier).state = newId;
              }
            } catch (_) {}
            if (mounted) Navigator.of(context, rootNavigator: true).pop();
          },
        ),
      ],
    );
  }

  // ---------------------------------------------------------------------------
  // Build
  // ---------------------------------------------------------------------------

  @override
  Widget build(BuildContext context) {
    final orgs = ref.watch(orgsProvider).valueOrNull ?? [];
    final currentOrgId = ref.watch(currentOrgProvider);
    final projectsAsync = ref.watch(projectsProvider);
    final authUser = ref.watch(consoleAuthProvider).valueOrNull;

    if (orgs.isEmpty && authUser != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) context.go('/onboarding');
      });
      return const SizedBox.shrink();
    }

    final routeOrgId = GoRouterState.of(context).pathParameters['orgId'];
    if (routeOrgId != null && routeOrgId != currentOrgId) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) ref.read(currentOrgProvider.notifier).state = routeOrgId;
      });
    }
    if (routeOrgId == null && currentOrgId != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) context.go('/org/$currentOrgId/projects');
      });
      return const SizedBox.shrink();
    }

    String orgName = orgs.isNotEmpty ? orgs.first['name'] as String : '';
    if (currentOrgId != null) {
      final org = orgs.where((o) => o['\$id'] == currentOrgId).firstOrNull;
      if (org != null) orgName = org['name'] as String;
    }

    final userEmail = authUser?.email ?? '';
    final userName = authUser?.name ?? '';
    final initials = _buildInitials(userName, userEmail);

    return Scaffold(
      backgroundColor: _bgColor,
      body: Column(
        children: [
          _topBar(orgs, currentOrgId, orgName, initials, userEmail,
              projectsAsync.valueOrNull ?? []),
          Expanded(
            child: SingleChildScrollView(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 1200),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 40, vertical: 32),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        // Heading row
                        Row(
                          children: [
                            Expanded(
                              child: Text(orgName,
                                  style: const TextStyle(
                                      color: Colors.white,
                                      fontSize: 28,
                                      fontWeight: FontWeight.w700)),
                            ),
                            ..._buildMemberAvatars(orgs, currentOrgId),
                            const SizedBox(width: 8),
                            OutlinedButton.icon(
                              style: OutlinedButton.styleFrom(
                                foregroundColor: Colors.white70,
                                side: BorderSide(
                                    color: Colors.white.withValues(alpha: 0.12)),
                                shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(8)),
                                padding: const EdgeInsets.symmetric(
                                    horizontal: 14, vertical: 8),
                              ),
                              icon: const Icon(LucideIcons.userPlus, size: 14),
                              label: const Text('Invite',
                                  style: TextStyle(fontSize: 12)),
                              onPressed: _showInviteMemberDialog,
                            ),
                          ],
                        ),
                        const SizedBox(height: 24),

                        // Tabs
                        PageTabs(
                          tabs: const [
                            'Projects',
                            'Members',
                            'Roles',
                            'Usage',
                            'Activity',
                            'Settings',
                          ],
                          selected: _tabIndex,
                          onChanged: (i) {
                            setState(() => _tabIndex = i);
                            if (i == 1) _loadMembers();
                          },
                        ),
                        const SizedBox(height: 20),

                        // Tab body
                        if (_tabIndex == 0) _buildProjectsTab(projectsAsync),
                        if (_tabIndex == 1) _buildMembersTab(),
                        if (_tabIndex == 2) _buildRolesTab(),
                        if (_tabIndex == 3)
                          _buildUsageTab(currentOrgId ?? ''),
                        if (_tabIndex == 4)
                          _buildActivityTab(currentOrgId ?? ''),
                        if (_tabIndex == 5)
                          _buildSettingsTab(orgName, currentOrgId ?? ''),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
          const ConsoleFooter(),
        ],
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Top bar
  // ---------------------------------------------------------------------------

  void _openSearch(
      List<Map<String, dynamic>> projects, List<Map<String, dynamic>> orgs) {
    showDialog(
      context: context,
      barrierColor: Colors.black.withValues(alpha: 0.65),
      builder: (ctx) => SearchModal(
        projects: projects,
        orgs: orgs,
        onNavigate: (path) {
          Navigator.of(ctx).pop();
          context.go(path);
        },
        onCreateProject: () {
          Navigator.of(ctx).pop();
          _showCreateProjectDialog();
        },
        onCreateOrg: () {
          Navigator.of(ctx).pop();
          _showCreateOrgDialog();
        },
      ),
    );
  }

  Widget _topBar(
    List<Map<String, dynamic>> orgs,
    String? currentOrgId,
    String orgName,
    String initials,
    String userEmail,
    List<Map<String, dynamic>> projects,
  ) {
    return Container(
      height: 52,
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      decoration: BoxDecoration(
        color: _bgColor,
        border:
            Border(bottom: BorderSide(color: Colors.white.withValues(alpha: 0.06))),
      ),
      child: Row(
        children: [
          GestureDetector(
            onTap: () => context.go('/projects'),
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: ClipRRect(
                borderRadius: BorderRadius.circular(6),
                child: Image.asset('assets/icon.png',
                    width: 42, height: 42, fit: BoxFit.cover),
              ),
            ),
          ),
          const SizedBox(width: 10),
          Text('/',
              style: TextStyle(
                  color: Colors.white.withValues(alpha: 0.18),
                  fontSize: 18,
                  fontWeight: FontWeight.w300)),
          const SizedBox(width: 10),
          PopupMenuButton<String>(
            offset: const Offset(0, 36),
            color: const Color(0xFF1A1A22),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
              side: BorderSide(color: Colors.white.withValues(alpha: 0.08)),
            ),
            onSelected: (value) {
              if (value == '__create__') {
                _showCreateOrgDialog();
              } else {
                ref.read(currentOrgProvider.notifier).state = value;
                context.go('/org/$value/projects');
              }
            },
            itemBuilder: (_) {
              final items = <PopupMenuEntry<String>>[];
              for (final org in orgs) {
                final id = org['\$id'] as String;
                final name = org['name'] as String;
                items.add(PopupMenuItem(
                  value: id,
                  child: Row(children: [
                    if (currentOrgId == id)
                      const Icon(LucideIcons.check,
                          size: 12, color: Colors.white70)
                    else
                      const SizedBox(width: 12),
                    const SizedBox(width: 8),
                    Text(name,
                        style: const TextStyle(
                            color: Colors.white70, fontSize: 13)),
                  ]),
                ));
              }
              items.add(const PopupMenuDivider());
              items.add(const PopupMenuItem(
                value: '__create__',
                child: Row(children: [
                  Icon(LucideIcons.plus, size: 14, color: _dimText),
                  SizedBox(width: 8),
                  Text('Create organization',
                      style: TextStyle(color: _dimText, fontSize: 13)),
                ]),
              ));
              return items;
            },
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
                    size: 14, color: Colors.white.withValues(alpha: 0.35)),
              ],
            ),
          ),
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
              onTap: () => _openSearch(projects, orgs),
              borderRadius: BorderRadius.circular(8),
              child: SizedBox(
                width: 34,
                height: 34,
                child: Icon(LucideIcons.search,
                    size: 17, color: Colors.white.withValues(alpha: 0.45)),
              ),
            ),
          ),
          const SizedBox(width: 8),
          Tooltip(
            message: userEmail,
            child: GestureDetector(
              onTap: () => context.go('/account'),
              child: Container(
                width: 32,
                height: 32,
                decoration:
                    const BoxDecoration(color: _accent, shape: BoxShape.circle),
                child: Center(
                  child: Text(initials,
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 12,
                          fontWeight: FontWeight.w600)),
                ),
              ),
            ),
          ),
          const SizedBox(width: 4),
        ],
      ),
    );
  }

  String _buildInitials(String name, String email) {
    if (name.isNotEmpty) {
      final parts = name.trim().split(RegExp(r'\s+'));
      if (parts.length >= 2) return '${parts[0][0]}${parts[1][0]}'.toUpperCase();
      return name[0].toUpperCase();
    }
    if (email.isNotEmpty) return email[0].toUpperCase();
    return '?';
  }

  List<Widget> _buildMemberAvatars(
      List<Map<String, dynamic>> orgs, String? currentOrgId) {
    final memberNames = <String>['You'];
    if (currentOrgId != null) {
      final org = orgs.where((o) => o['\$id'] == currentOrgId).firstOrNull;
      if (org != null) {
        final count = (org['totalMembers'] ?? 1) as int;
        for (var i = 1; i < count && i < 4; i++) {
          memberNames.add('M$i');
        }
      }
    }
    return memberNames.map((name) {
      return Padding(
        padding: const EdgeInsets.only(right: 4),
        child: CircleAvatar(
          radius: 14,
          backgroundColor: _accent.withValues(alpha: 0.25),
          child: Text(name.isNotEmpty ? name[0].toUpperCase() : '?',
              style: const TextStyle(
                  color: _accent, fontSize: 11, fontWeight: FontWeight.w600)),
        ),
      );
    }).toList();
  }

  // ---------------------------------------------------------------------------
  // Projects tab
  // ---------------------------------------------------------------------------

  Widget _buildProjectsTab(
      AsyncValue<List<Map<String, dynamic>>> projectsAsync) {
    return projectsAsync.when(
      loading: () => const Center(
          child: Padding(
              padding: EdgeInsets.all(64),
              child: CircularProgressIndicator(color: _accent))),
      error: (e, _) => Center(
          child: Padding(
              padding: const EdgeInsets.all(64),
              child: Text('Error: $e',
                  style: const TextStyle(color: Colors.white70)))),
      data: (allProjects) {
        final query = _searchCtrl.text.trim().toLowerCase();
        final filtered = query.isEmpty
            ? allProjects
            : allProjects.where((p) {
                final name = (p['name'] ?? '').toString().toLowerCase();
                final id = (p['\$id'] ?? '').toString().toLowerCase();
                return name.contains(query) || id.contains(query);
              }).toList();

        final total = filtered.length;
        final pageItems =
            filtered.skip((_page - 1) * _perPage).take(_perPage).toList();

        return Column(children: [
          SearchListHeader(
            searchController: _searchCtrl,
            total: total,
            perPage: _perPage,
            currentPage: _page,
            onPerPageChanged: (v) => setState(() {
              _perPage = v;
              _page = 1;
            }),
            onPrev: () => setState(() => _page--),
            onNext: () => setState(() => _page++),
            onSearch: () => setState(() => _page = 1),
            trailing: FilledButton.icon(
              style: FilledButton.styleFrom(
                backgroundColor: _accent,
                foregroundColor: Colors.white,
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              ),
              icon: const Icon(LucideIcons.plus, size: 16),
              label: const Text('Create project',
                  style: TextStyle(fontSize: 13)),
              onPressed: _showCreateProjectDialog,
            ),
          ),
          const SizedBox(height: 16),
          if (filtered.isEmpty && query.isNotEmpty)
            Padding(
              padding: const EdgeInsets.all(48),
              child: Column(mainAxisSize: MainAxisSize.min, children: [
                const Icon(LucideIcons.searchX, size: 36, color: _subtleText),
                const SizedBox(height: 12),
                Text('No projects matching "$query"',
                    style: const TextStyle(color: _dimText, fontSize: 14)),
              ]),
            )
          else
            LayoutBuilder(builder: (context, constraints) {
              final cardWidth = (constraints.maxWidth - 16) / 2;
              return Wrap(
                spacing: 16,
                runSpacing: 16,
                children: [
                  ...pageItems.map((p) {
                    final id = p['\$id'] as String;
                    return _ProjectCard(
                        project: p,
                        width: cardWidth,
                        onTap: () => _selectProject(p),
                        onDelete: () => _deleteProject(id),
                        onSettings: () {
                          ref.read(currentProjectProvider.notifier).state = id;
                          ref.read(apiClientProvider).setProject(id);
                          context.go('/project/$id/settings');
                        },
                      );
                  }),
                  _CreateProjectPlaceholder(
                      width: cardWidth,
                      onTap: _showCreateProjectDialog),
                ],
              );
            }),
          const SizedBox(height: 16),
          SearchListFooter(
            total: total,
            perPage: _perPage,
            currentPage: _page,
            onPrev: () => setState(() => _page--),
            onNext: () => setState(() => _page++),
            onPerPageChanged: (v) => setState(() {
              _perPage = v;
              _page = 1;
            }),
          ),
        ]);
      },
    );
  }

  // ---------------------------------------------------------------------------
  // Members tab
  // ---------------------------------------------------------------------------

  Widget _buildMembersTab() {
    if (_membersLoading) {
      return const Center(
          child: Padding(
              padding: EdgeInsets.all(64),
              child: CircularProgressIndicator(color: _accent)));
    }
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        const Text('Members',
            style: TextStyle(
                color: Colors.white,
                fontSize: 16,
                fontWeight: FontWeight.w600)),
        const Spacer(),
        OutlinedButton.icon(
          style: OutlinedButton.styleFrom(
            foregroundColor: Colors.white,
            backgroundColor: _accent.withValues(alpha: 0.15),
            side: const BorderSide(color: _accent),
            shape:
                RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            padding:
                const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          ),
          icon: const Icon(LucideIcons.userPlus, size: 16),
          label:
              const Text('Invite member', style: TextStyle(fontSize: 13)),
          onPressed: _showInviteMemberDialog,
        ),
      ]),
      const SizedBox(height: 16),
      if (_members.isEmpty)
        _emptyState(LucideIcons.users, 'No members yet',
            'Invite someone to collaborate on this organization.'),
      ..._members.map((m) {
        final memberId = m['\$id'] as String? ?? '';
        final role = m['role'] as String? ?? 'member';
        final isOwner = role == 'owner';
        return Container(
          margin: const EdgeInsets.only(bottom: 8),
          padding:
              const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
          decoration: BoxDecoration(
            color: _cardColor,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(color: _border),
          ),
          child: Row(children: [
            CircleAvatar(
              radius: 16,
              backgroundColor: _accent.withValues(alpha: 0.2),
              child: Text(_initial(m['name'] ?? m['email'] ?? '?'),
                  style: const TextStyle(
                      color: _accent,
                      fontSize: 12,
                      fontWeight: FontWeight.w600)),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(m['name'] ?? 'Unnamed',
                      style: const TextStyle(
                          color: Colors.white, fontSize: 14)),
                  if (m['email'] != null)
                    Text(m['email'],
                        style: const TextStyle(
                            color: _dimText, fontSize: 12)),
                ],
              ),
            ),
            // Role dropdown
            DropdownButton<String>(
              value: role,
              dropdownColor: const Color(0xFF1A1A22),
              underline: const SizedBox.shrink(),
              style: const TextStyle(color: Colors.white70, fontSize: 12),
              items: const [
                DropdownMenuItem(value: 'owner', child: Text('Owner')),
                DropdownMenuItem(value: 'admin', child: Text('Admin')),
                DropdownMenuItem(
                    value: 'developer', child: Text('Developer')),
                DropdownMenuItem(value: 'viewer', child: Text('Viewer')),
              ],
              onChanged: isOwner
                  ? null
                  : (v) {
                      if (v != null) _updateMemberRole(memberId, v);
                    },
            ),
            const SizedBox(width: 8),
            if (!isOwner)
              IconButton(
                icon: const Icon(LucideIcons.userMinus, size: 15),
                color: Colors.red.withValues(alpha: 0.7),
                tooltip: 'Remove member',
                onPressed: () => _removeMember(memberId),
              )
            else
              const SizedBox(width: 40),
          ]),
        );
      }),
    ]);
  }

  String _initial(String s) => s.isNotEmpty ? s[0].toUpperCase() : '?';

  // ---------------------------------------------------------------------------
  // Roles tab
  // ---------------------------------------------------------------------------

  Widget _buildRolesTab() {
    const roles = ['Owner', 'Admin', 'Developer', 'Viewer'];
    const permissions = <(String, List<bool>)>[
      ('Manage organization settings', [true, false, false, false]),
      ('Delete organization', [true, false, false, false]),
      ('Invite & remove members', [true, true, false, false]),
      ('Change member roles', [true, true, false, false]),
      ('Create & delete projects', [true, true, false, false]),
      ('View all projects', [true, true, true, true]),
      ('Manage project settings', [true, true, true, false]),
      ('View API keys', [true, true, true, false]),
      ('Create & delete API keys', [true, true, false, false]),
      ('Access databases & storage', [true, true, true, false]),
      ('View usage & activity', [true, true, true, true]),
      ('Manage billing', [true, false, false, false]),
    ];

    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      const Text('Roles & Permissions',
          style: TextStyle(
              color: Colors.white,
              fontSize: 16,
              fontWeight: FontWeight.w600)),
      const SizedBox(height: 4),
      const Text('Roles are assigned per member in the Members tab.',
          style: TextStyle(color: _dimText, fontSize: 13)),
      const SizedBox(height: 20),
      Container(
        decoration: BoxDecoration(
          color: _cardColor,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: _border),
        ),
        child: Column(children: [
          // Header row
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
            decoration: BoxDecoration(
              border: Border(
                  bottom: BorderSide(color: Colors.white.withValues(alpha: 0.06))),
            ),
            child: Row(children: [
              const Expanded(
                  flex: 3,
                  child: Text('Permission',
                      style: TextStyle(
                          color: _dimText,
                          fontSize: 11,
                          fontWeight: FontWeight.w600,
                          letterSpacing: 0.5))),
              ...roles.map((r) => Expanded(
                    child: Center(
                      child: Text(r,
                          style: const TextStyle(
                              color: _dimText,
                              fontSize: 11,
                              fontWeight: FontWeight.w600,
                              letterSpacing: 0.5)),
                    ),
                  )),
            ]),
          ),
          // Permission rows
          ...permissions.asMap().entries.map((entry) {
            final i = entry.key;
            final perm = entry.value;
            final isLast = i == permissions.length - 1;
            return Container(
              padding:
                  const EdgeInsets.symmetric(horizontal: 20, vertical: 13),
              decoration: isLast
                  ? null
                  : BoxDecoration(
                      border: Border(
                          bottom: BorderSide(
                              color: Colors.white.withValues(alpha: 0.04)))),
              child: Row(children: [
                Expanded(
                    flex: 3,
                    child: Text(perm.$1,
                        style: const TextStyle(
                            color: Colors.white, fontSize: 13))),
                ...perm.$2.map((allowed) => Expanded(
                      child: Center(
                        child: Icon(
                          allowed
                              ? LucideIcons.checkCircle2
                              : LucideIcons.minusCircle,
                          size: 16,
                          color: allowed
                              ? _green
                              : Colors.white.withValues(alpha: 0.15),
                        ),
                      ),
                    )),
              ]),
            );
          }),
        ]),
      ),
    ]);
  }

  // ---------------------------------------------------------------------------
  // Usage tab
  // ---------------------------------------------------------------------------

  Widget _buildUsageTab(String orgId) {
    if (orgId.isEmpty) {
      return _emptyState(
          LucideIcons.barChart3, 'No organization selected', '');
    }
    final statsAsync = ref.watch(_orgStatsProvider(orgId));
    return statsAsync.when(
      loading: () => const Center(
          child: Padding(
              padding: EdgeInsets.all(48),
              child: CircularProgressIndicator(color: _accent))),
      error: (e, _) => _emptyState(LucideIcons.alertCircle,
          'Could not load usage', e.toString()),
      data: (stats) => Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Usage overview',
                style: TextStyle(
                    color: Colors.white,
                    fontSize: 16,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            const Text(
                'Aggregated across all projects in this organization.',
                style: TextStyle(color: _dimText, fontSize: 13)),
            const SizedBox(height: 20),
            Row(children: [
              _UsageStat(
                  icon: LucideIcons.layers,
                  label: 'Projects',
                  value: '${stats['totalProjects'] ?? 0}'),
              const SizedBox(width: 12),
              _UsageStat(
                  icon: LucideIcons.users,
                  label: 'Total users',
                  value: '${stats['totalUsers'] ?? 0}'),
              const SizedBox(width: 12),
              _UsageStat(
                  icon: LucideIcons.hardDrive,
                  label: 'Storage used',
                  value: _formatBytes(stats['totalStorage'])),
              const SizedBox(width: 12),
              _UsageStat(
                  icon: LucideIcons.zap,
                  label: 'Executions',
                  value: '${stats['totalExecutions'] ?? 0}'),
            ]),
            const SizedBox(height: 24),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: _cardColor,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: _border),
              ),
              child: const Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Members',
                        style: TextStyle(
                            color: _dimText,
                            fontSize: 11,
                            fontWeight: FontWeight.w600,
                            letterSpacing: 0.5)),
                    SizedBox(height: 8),
                    Text('Detailed per-project breakdown is available in '
                        'each project\'s Overview page.',
                        style: TextStyle(color: _dimText, fontSize: 13)),
                  ]),
            ),
          ]),
    );
  }

  String _formatBytes(dynamic raw) {
    final bytes = raw is int ? raw : (raw is double ? raw.toInt() : 0);
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    if (bytes < 1024 * 1024 * 1024) {
      return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
    }
    return '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(2)} GB';
  }

  // ---------------------------------------------------------------------------
  // Activity tab
  // ---------------------------------------------------------------------------

  Widget _buildActivityTab(String orgId) {
    if (orgId.isEmpty) {
      return _emptyState(
          LucideIcons.activity, 'No organization selected', '');
    }
    final activityAsync = ref.watch(_orgActivityProvider(orgId));
    return activityAsync.when(
      loading: () => const Center(
          child: Padding(
              padding: EdgeInsets.all(48),
              child: CircularProgressIndicator(color: _accent))),
      error: (e, _) => _emptyState(
          LucideIcons.alertCircle, 'Could not load activity', e.toString()),
      data: (data) {
        final entries = List<Map<String, dynamic>>.from(
            data['activity'] ?? []);
        return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(children: [
                const Text('Activity log',
                    style: TextStyle(
                        color: Colors.white,
                        fontSize: 16,
                        fontWeight: FontWeight.w600)),
                const Spacer(),
                OutlinedButton.icon(
                  style: OutlinedButton.styleFrom(
                    foregroundColor: Colors.white70,
                    side: BorderSide(
                        color: Colors.white.withValues(alpha: 0.12)),
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                    padding: const EdgeInsets.symmetric(
                        horizontal: 12, vertical: 8),
                  ),
                  onPressed: () => ref.invalidate(_orgActivityProvider(orgId)),
                  icon: const Icon(LucideIcons.refreshCw, size: 14),
                  label: const Text('Refresh',
                      style: TextStyle(fontSize: 13)),
                ),
              ]),
              const SizedBox(height: 16),
              if (entries.isEmpty)
                _emptyState(LucideIcons.scrollText, 'No activity yet',
                    'Actions taken across all projects will appear here.')
              else
                Container(
                  decoration: BoxDecoration(
                    color: _cardColor,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: _border),
                  ),
                  child: Column(children: [
                    // Header
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 20, vertical: 10),
                      decoration: BoxDecoration(
                          border: Border(
                              bottom: BorderSide(
                                  color:
                                      Colors.white.withValues(alpha: 0.06)))),
                      child: const Row(children: [
                        Expanded(
                            flex: 2,
                            child: Text('Action',
                                style: TextStyle(
                                    color: _dimText,
                                    fontSize: 11,
                                    fontWeight: FontWeight.w600,
                                    letterSpacing: 0.5))),
                        Expanded(
                            flex: 2,
                            child: Text('Project',
                                style: TextStyle(
                                    color: _dimText,
                                    fontSize: 11,
                                    fontWeight: FontWeight.w600,
                                    letterSpacing: 0.5))),
                        Expanded(
                            flex: 2,
                            child: Text('Path',
                                style: TextStyle(
                                    color: _dimText,
                                    fontSize: 11,
                                    fontWeight: FontWeight.w600,
                                    letterSpacing: 0.5))),
                        Expanded(
                            child: Text('Status',
                                style: TextStyle(
                                    color: _dimText,
                                    fontSize: 11,
                                    fontWeight: FontWeight.w600,
                                    letterSpacing: 0.5))),
                        Expanded(
                            flex: 2,
                            child: Text('Time',
                                style: TextStyle(
                                    color: _dimText,
                                    fontSize: 11,
                                    fontWeight: FontWeight.w600,
                                    letterSpacing: 0.5))),
                      ]),
                    ),
                    ...entries.asMap().entries.map((e) {
                      final i = e.key;
                      final entry = e.value;
                      final isLast = i == entries.length - 1;
                      final status = (entry['statusCode'] as num?)?.toInt() ?? 0;
                      final statusOk = status >= 200 && status < 300;
                      return Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 20, vertical: 11),
                        decoration: isLast
                            ? null
                            : BoxDecoration(
                                border: Border(
                                    bottom: BorderSide(
                                        color: Colors.white
                                            .withValues(alpha: 0.04)))),
                        child: Row(children: [
                          Expanded(
                            flex: 2,
                            child: Text(
                              entry['action'] ?? entry['method'] ?? '—',
                              style: const TextStyle(
                                  color: Colors.white, fontSize: 13),
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                          Expanded(
                            flex: 2,
                            child: Text(
                              entry['projectName'] ?? entry['projectId'] ?? '—',
                              style: const TextStyle(
                                  color: _dimText, fontSize: 12),
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                          Expanded(
                            flex: 2,
                            child: Text(
                              entry['path'] ?? '—',
                              style: const TextStyle(
                                  fontFamily: 'monospace',
                                  color: _dimText,
                                  fontSize: 11),
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                          Expanded(
                            child: Container(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 6, vertical: 2),
                              decoration: BoxDecoration(
                                color: statusOk
                                    ? _green.withValues(alpha: 0.12)
                                    : _red.withValues(alpha: 0.12),
                                borderRadius: BorderRadius.circular(4),
                              ),
                              child: Text(
                                '$status',
                                style: TextStyle(
                                    color: statusOk ? _green : _red,
                                    fontSize: 11,
                                    fontWeight: FontWeight.w600),
                              ),
                            ),
                          ),
                          Expanded(
                            flex: 2,
                            child: Text(
                              _relativeTime(entry['\$createdAt'] ??
                                  entry['createdAt'] ?? ''),
                              style: const TextStyle(
                                  color: _dimText, fontSize: 12),
                            ),
                          ),
                        ]),
                      );
                    }),
                  ]),
                ),
            ]);
      },
    );
  }

  String _relativeTime(dynamic raw) {
    if (raw == null) return '—';
    try {
      final dt = raw is DateTime ? raw : DateTime.parse(raw.toString());
      final diff = DateTime.now().difference(dt);
      if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
      if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
      if (diff.inHours < 24) return '${diff.inHours}h ago';
      return '${diff.inDays}d ago';
    } catch (_) {
      return raw.toString();
    }
  }

  // ---------------------------------------------------------------------------
  // Settings tab
  // ---------------------------------------------------------------------------

  Widget _buildSettingsTab(String orgName, String orgId) {
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      // Org name
      _settingsCard(
        title: 'Organization name',
        subtitle: 'Update your organization display name.',
        child: FilledButton(
          style: FilledButton.styleFrom(
            backgroundColor: Colors.white.withValues(alpha: 0.08),
            foregroundColor: Colors.white,
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8)),
            padding:
                const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
          ),
          onPressed: () => _showUpdateOrgDialog(orgName),
          child: const Text('Update name', style: TextStyle(fontSize: 13)),
        ),
      ),
      const SizedBox(height: 16),

      // Org ID
      _settingsCard(
        title: 'Organization ID',
        subtitle: 'Use this ID when referencing this organization via the API.',
        child: Row(children: [
          Container(
            padding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: 0.05),
              borderRadius: BorderRadius.circular(6),
              border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
            ),
            child: Text(orgId,
                style: const TextStyle(
                    fontFamily: 'monospace',
                    color: _dimText,
                    fontSize: 12)),
          ),
          const SizedBox(width: 8),
          IconButton(
            icon: const Icon(LucideIcons.copy, size: 15),
            color: _dimText,
            tooltip: 'Copy ID',
            onPressed: () {
              Clipboard.setData(ClipboardData(text: orgId));
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                    content: Text('Organization ID copied'),
                    duration: Duration(seconds: 2)),
              );
            },
          ),
        ]),
      ),
      const SizedBox(height: 24),

      // Danger zone
      Container(
        width: double.infinity,
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          color: _cardColor,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: Colors.red.withValues(alpha: 0.25)),
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Text('Danger zone',
              style: TextStyle(
                  color: Colors.red,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 8),
          Text(
            'Permanently delete this organization and all its projects. '
            'This action cannot be undone.',
            style:
                TextStyle(color: Colors.white.withValues(alpha: 0.5), fontSize: 13),
          ),
          const SizedBox(height: 16),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Colors.red.withValues(alpha: 0.15),
              foregroundColor: Colors.red,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8),
                side: const BorderSide(color: Colors.red, width: 0.5),
              ),
            ),
            onPressed: _deleteOrg,
            child: const Text('Delete organization'),
          ),
        ]),
      ),
    ]);
  }

  Widget _settingsCard(
      {required String title,
      required String subtitle,
      required Widget child}) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: _border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title,
            style: const TextStyle(
                color: Colors.white,
                fontSize: 15,
                fontWeight: FontWeight.w600)),
        const SizedBox(height: 4),
        Text(subtitle,
            style: TextStyle(
                color: Colors.white.withValues(alpha: 0.45), fontSize: 13)),
        const SizedBox(height: 16),
        child,
      ]),
    );
  }

  // ---------------------------------------------------------------------------
  // Shared helpers
  // ---------------------------------------------------------------------------

  Widget _emptyState(IconData icon, String title, String subtitle) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(48),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Icon(icon, size: 36, color: _subtleText),
          const SizedBox(height: 12),
          Text(title,
              style: const TextStyle(color: _dimText, fontSize: 14)),
          if (subtitle.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(subtitle,
                style: const TextStyle(color: _subtleText, fontSize: 12),
                textAlign: TextAlign.center),
          ],
        ]),
      ),
    );
  }
}

// =============================================================================
// Usage stat card
// =============================================================================

class _UsageStat extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;

  const _UsageStat(
      {required this.icon, required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: _cardColor,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: _border),
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Icon(icon, size: 18, color: _dimText),
          const SizedBox(height: 10),
          Text(value,
              style: const TextStyle(
                  color: Colors.white,
                  fontSize: 22,
                  fontWeight: FontWeight.w700)),
          const SizedBox(height: 2),
          Text(label,
              style: const TextStyle(color: _subtleText, fontSize: 11)),
        ]),
      ),
    );
  }
}

// =============================================================================
// Project card (stateful for hover)
// =============================================================================

class _ProjectCard extends StatefulWidget {
  final Map<String, dynamic> project;
  final double width;
  final VoidCallback onTap;
  final VoidCallback onDelete;
  final VoidCallback onSettings;

  const _ProjectCard({
    required this.project,
    required this.width,
    required this.onTap,
    required this.onDelete,
    required this.onSettings,
  });

  @override
  State<_ProjectCard> createState() => _ProjectCardState();
}

class _ProjectCardState extends State<_ProjectCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final name = widget.project['name'] as String? ?? 'Untitled';
    final id = widget.project['\$id'] as String? ?? '';
    final desc = widget.project['description'] as String? ?? '';
    final createdAt = widget.project['\$createdAt'] ??
        widget.project['createdAt'];
    final color = _projectColor(id.isNotEmpty ? id : name);

    return GestureDetector(
      onTap: widget.onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        onEnter: (_) => setState(() => _hovered = true),
        onExit: (_) => setState(() => _hovered = false),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          width: widget.width,
          height: 180,
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: _hovered
                ? const Color(0xFF1C1D22)
                : _cardColor,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: _hovered
                  ? Colors.white.withValues(alpha: 0.1)
                  : _border,
            ),
          ),
          child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(children: [
                  // Project icon
                  Container(
                    width: 40,
                    height: 40,
                    decoration: BoxDecoration(
                      color: color.withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Center(
                      child: Text(
                        name.isNotEmpty ? name[0].toUpperCase() : '?',
                        style: TextStyle(
                            color: color,
                            fontSize: 18,
                            fontWeight: FontWeight.w700),
                      ),
                    ),
                  ),
                  const Spacer(),
                  // 3-dot menu
                  PopupMenuButton<String>(
                    color: const Color(0xFF1A1A22),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                      side: BorderSide(
                          color: Colors.white.withValues(alpha: 0.08)),
                    ),
                    onSelected: (v) {
                      if (v == 'settings') {
                        widget.onSettings();
                      } else if (v == 'delete') {
                        widget.onDelete();
                      }
                    },
                    itemBuilder: (_) => [
                      const PopupMenuItem(
                        value: 'settings',
                        child: Row(children: [
                          Icon(LucideIcons.settings, size: 14, color: _dimText),
                          SizedBox(width: 8),
                          Text('Settings',
                              style: TextStyle(color: Colors.white70, fontSize: 13)),
                        ]),
                      ),
                      const PopupMenuDivider(),
                      const PopupMenuItem(
                        value: 'delete',
                        child: Row(children: [
                          Icon(LucideIcons.trash2, size: 14, color: Colors.red),
                          SizedBox(width: 8),
                          Text('Delete',
                              style: TextStyle(color: Colors.red, fontSize: 13)),
                        ]),
                      ),
                    ],
                    child: Icon(LucideIcons.moreVertical,
                        size: 16,
                        color: _hovered
                            ? Colors.white.withValues(alpha: 0.5)
                            : Colors.transparent),
                  ),
                ]),
                const SizedBox(height: 14),
                Text(name,
                    style: const TextStyle(
                        color: Colors.white,
                        fontSize: 16,
                        fontWeight: FontWeight.w600),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis),
                if (desc.isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text(desc,
                      style: const TextStyle(color: _dimText, fontSize: 12),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis),
                ] else ...[
                  const SizedBox(height: 2),
                  Text(id,
                      style: const TextStyle(color: _subtleText, fontSize: 11),
                      overflow: TextOverflow.ellipsis),
                ],
                const Spacer(),
                Text(_formatDate(createdAt),
                    style:
                        const TextStyle(color: _subtleText, fontSize: 11)),
              ]),
        ),
      ),
    );
  }

  String _formatDate(dynamic raw) {
    if (raw == null) return '';
    try {
      final dt = raw is DateTime ? raw : DateTime.parse(raw.toString());
      final diff = DateTime.now().difference(dt);
      if (diff.inDays == 0) return 'Today';
      if (diff.inDays == 1) return 'Yesterday';
      if (diff.inDays < 30) return '${diff.inDays}d ago';
      if (diff.inDays < 365) return '${(diff.inDays / 30).floor()}mo ago';
      return '${(diff.inDays / 365).floor()}y ago';
    } catch (_) {
      return '';
    }
  }
}

// Need access to ref inside _ProjectCard — use extension on context
extension _RefContext on BuildContext {
  // ignore: unused_element
  WidgetRef? get _ref => null;
}

// We need ref in _ProjectCard for navigation. Use a workaround via InheritedWidget
// or just access via context.go which doesn't need ref. The settings navigation
// needs ref — solved by keeping it simple: pass callbacks from parent.

// =============================================================================
// Create project placeholder
// =============================================================================

class _CreateProjectPlaceholder extends StatefulWidget {
  final double width;
  final VoidCallback onTap;

  const _CreateProjectPlaceholder(
      {required this.width, required this.onTap});

  @override
  State<_CreateProjectPlaceholder> createState() =>
      _CreateProjectPlaceholderState();
}

class _CreateProjectPlaceholderState
    extends State<_CreateProjectPlaceholder> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: widget.onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        onEnter: (_) => setState(() => _hovered = true),
        onExit: (_) => setState(() => _hovered = false),
        child: CustomPaint(
          painter: _DashedBorderPainter(
            color: _hovered
                ? Colors.white.withValues(alpha: 0.2)
                : Colors.white.withValues(alpha: 0.1),
            borderRadius: 12,
            dashWidth: 6,
            dashSpace: 4,
            strokeWidth: 1,
          ),
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 120),
            width: widget.width,
            height: 180,
            padding: const EdgeInsets.all(20),
            color: Colors.transparent,
            child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Container(
                    width: 40,
                    height: 40,
                    decoration: BoxDecoration(
                      color: _hovered
                          ? Colors.white.withValues(alpha: 0.08)
                          : Colors.white.withValues(alpha: 0.04),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: const Center(
                      child:
                          Icon(LucideIcons.plus, size: 20, color: _dimText),
                    ),
                  ),
                  const SizedBox(height: 14),
                  Text('Create a new project',
                      style: TextStyle(
                          color: _hovered ? _dimText : _subtleText,
                          fontSize: 14,
                          fontWeight: FontWeight.w500)),
                ]),
          ),
        ),
      ),
    );
  }
}

class _DashedBorderPainter extends CustomPainter {
  final Color color;
  final double borderRadius;
  final double dashWidth;
  final double dashSpace;
  final double strokeWidth;

  _DashedBorderPainter({
    required this.color,
    required this.borderRadius,
    required this.dashWidth,
    required this.dashSpace,
    required this.strokeWidth,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..strokeWidth = strokeWidth
      ..style = PaintingStyle.stroke;
    final rrect = RRect.fromRectAndRadius(
        Rect.fromLTWH(0, 0, size.width, size.height),
        Radius.circular(borderRadius));
    final path = Path()..addRRect(rrect);
    for (final metric in path.computeMetrics()) {
      double distance = 0;
      while (distance < metric.length) {
        final end = math.min(distance + dashWidth, metric.length);
        canvas.drawPath(metric.extractPath(distance, end), paint);
        distance += dashWidth + dashSpace;
      }
    }
  }

  @override
  bool shouldRepaint(covariant _DashedBorderPainter old) =>
      color != old.color;
}
