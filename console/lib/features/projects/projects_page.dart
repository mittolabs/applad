import 'package:flutter/material.dart';
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

const _bgColor = Color(0xFF0B0B0F);
const _cardColor = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _border = Color(0x0DFFFFFF);

class ProjectsPage extends ConsumerStatefulWidget {
  const ProjectsPage({super.key});

  @override
  ConsumerState<ProjectsPage> createState() => _ProjectsPageState();
}

class _ProjectsPageState extends ConsumerState<ProjectsPage> {
  int _tabIndex = 0;
  final _searchCtrl = TextEditingController();

  // Pagination
  int _page = 1;
  int _perPage = 6;

  // Members
  List<Map<String, dynamic>> _members = [];
  bool _membersLoading = false;

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  // ---------------------------------------------------------------------------
  // Data fetching
  // ---------------------------------------------------------------------------

  Future<void> _loadMembers() async {
    final orgId = ref.read(currentOrgProvider);
    if (orgId == null) return;
    setState(() => _membersLoading = true);
    try {
      final api = ref.read(apiClientProvider);
      final res = await api.get('/organizations/$orgId/members');
      final data = res.data as Map<String, dynamic>;
      setState(() {
        _members = List<Map<String, dynamic>>.from(data['members'] ?? []);
      });
    } catch (_) {}
    if (mounted) setState(() => _membersLoading = false);
  }

  // ---------------------------------------------------------------------------
  // Actions
  // ---------------------------------------------------------------------------

  void _selectProject(Map<String, dynamic> project) {
    final id = project['\$id'] as String;
    ref.read(currentProjectProvider.notifier).state = id;
    ref.read(apiClientProvider).setProject(id);
    context.go('/project/$id/overview');
  }

  void _showCreateProjectDialog() {
    final nameCtrl = TextEditingController();
    final descCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Create project',
      subtitle: 'Start building something new',
      content: Column(mainAxisSize: MainAxisSize.min, children: [
        AppDialogField(controller: nameCtrl, label: 'Project name', hint: 'My project', autofocus: true),
        const SizedBox(height: 16),
        AppDialogField(controller: descCtrl, label: 'Description', hint: 'Optional description'),
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
                orgId != null ? '/organizations/$orgId/projects' : '/projects',
                data: {'name': nameCtrl.text.trim(), 'description': descCtrl.text.trim()},
              );
              ref.invalidate(projectsProvider);
            } catch (_) {}
            if (mounted) Navigator.pop(context);
          },
        ),
      ],
    );
  }

  void _showInviteMemberDialog() {
    final emailCtrl = TextEditingController();
    final nameCtrl = TextEditingController();
    showAppDialog(
      context: context,
      title: 'Invite member',
      subtitle: 'Add a team member to this organization',
      content: Column(mainAxisSize: MainAxisSize.min, children: [
        AppDialogField(controller: emailCtrl, label: 'Email address', hint: 'name@example.com', autofocus: true),
        const SizedBox(height: 16),
        AppDialogField(controller: nameCtrl, label: 'Name', hint: 'Optional'),
      ]),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Invite',
          onTap: () async {
            if (emailCtrl.text.trim().isEmpty) return;
            try {
              final orgId = ref.read(currentOrgProvider);
              if (orgId == null) return;
              final api = ref.read(apiClientProvider);
              await api.post('/organizations/$orgId/members',
                  data: {'email': emailCtrl.text.trim(), 'name': nameCtrl.text.trim()});
              _loadMembers();
            } catch (_) {}
            if (mounted) Navigator.pop(context);
          },
        ),
      ],
    );
  }

  void _showUpdateOrgDialog(String currentName) {
    final nameCtrl = TextEditingController(text: currentName);
    showAppDialog(
      context: context,
      title: 'Update organization',
      subtitle: 'Change the display name',
      content: AppDialogField(controller: nameCtrl, label: 'Organization name', hint: 'Name', autofocus: true),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Update',
          onTap: () async {
            try {
              final orgId = ref.read(currentOrgProvider);
              if (orgId == null) return;
              final api = ref.read(apiClientProvider);
              await api.patch('/organizations/$orgId', data: {'name': nameCtrl.text.trim()});
              ref.invalidate(orgsProvider);
            } catch (_) {}
            if (mounted) Navigator.pop(context);
          },
        ),
      ],
    );
  }

  void _deleteOrg() async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete organization',
      subtitle: 'This action is irreversible',
      content: Text(
        'All projects and data within this organization will be permanently deleted.',
        style: TextStyle(color: Colors.white.withOpacity(0.5), fontSize: 13),
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
        final api = ref.read(apiClientProvider);
        await api.delete('/organizations/$orgId');
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
      content: AppDialogField(controller: nameCtrl, label: 'Organization name', hint: 'My org', autofocus: true),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            if (nameCtrl.text.trim().isEmpty) return;
            try {
              final api = ref.read(apiClientProvider);
              final res = await api.post('/organizations', data: {'name': nameCtrl.text.trim()});
              final data = res.data as Map<String, dynamic>;
              final newId = data['\$id'] as String?;
              ref.invalidate(orgsProvider);
              if (newId != null) ref.read(currentOrgProvider.notifier).state = newId;
            } catch (_) {}
            if (mounted) Navigator.pop(context);
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

    String orgName = orgs.isNotEmpty ? (orgs.first['name'] ?? 'Organization') : 'Organization';
    if (currentOrgId != null) {
      final org = orgs.where((o) => o['\$id'] == currentOrgId).firstOrNull;
      if (org != null) orgName = org['name'] ?? 'Organization';
    }

    final userEmail = authUser?.email ?? '';
    final userName = authUser?.name ?? '';
    final initials = _buildInitials(userName, userEmail);

    return Scaffold(
      backgroundColor: _bgColor,
      body: Column(
        children: [
          // Top bar
          _topBar(orgs, currentOrgId, orgName, initials, userEmail,
              projectsAsync.valueOrNull ?? []),

          // Page body
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
                        // Org heading + member avatars + invite
                        Row(
                          children: [
                            Expanded(
                              child: Text(
                                orgName,
                                style: const TextStyle(
                                  color: Colors.white,
                                  fontSize: 28,
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
                            ),
                            ..._buildMemberAvatars(orgs, currentOrgId),
                            const SizedBox(width: 8),
                            OutlinedButton.icon(
                              style: OutlinedButton.styleFrom(
                                foregroundColor: Colors.white70,
                                side: BorderSide(
                                    color: Colors.white.withOpacity(0.12)),
                                shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(8)),
                                padding: const EdgeInsets.symmetric(
                                    horizontal: 14, vertical: 8),
                              ),
                              icon:
                                  const Icon(LucideIcons.userPlus, size: 14),
                              label: const Text('Invite',
                                  style: TextStyle(fontSize: 12)),
                              onPressed: _showInviteMemberDialog,
                            ),
                          ],
                        ),
                        const SizedBox(height: 24),

                        // Tabs
                        PageTabs(
                          tabs: const ['Projects', 'Members', 'Settings'],
                          selected: _tabIndex,
                          onChanged: (i) {
                            setState(() => _tabIndex = i);
                            if (i == 1) _loadMembers();
                          },
                        ),
                        const SizedBox(height: 16),

                        // Tab content
                        if (_tabIndex == 0)
                          _buildProjectsTab(projectsAsync),
                        if (_tabIndex == 1) _buildMembersTab(),
                        if (_tabIndex == 2) _buildSettingsTab(orgName),
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
  // Top bar (Appwrite-style)
  // ---------------------------------------------------------------------------

  void _openSearch(
      List<Map<String, dynamic>> projects, List<Map<String, dynamic>> orgs) {
    showDialog(
      context: context,
      barrierColor: Colors.black.withOpacity(0.65),
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
        border: Border(
          bottom: BorderSide(color: Colors.white.withOpacity(0.06)),
        ),
      ),
      child: Row(
        children: [
          // App logo mark
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

          // Separator
          Text(
            '/',
            style: TextStyle(
              color: Colors.white.withOpacity(0.18),
              fontSize: 18,
              fontWeight: FontWeight.w300,
            ),
          ),

          const SizedBox(width: 10),

          // Org dropdown
          PopupMenuButton<String>(
            offset: const Offset(0, 36),
            color: const Color(0xFF1A1A22),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
              side: BorderSide(color: Colors.white.withOpacity(0.08)),
            ),
            onSelected: (value) {
              if (value == '__create__') {
                _showCreateOrgDialog();
              } else {
                ref.read(currentOrgProvider.notifier).state =
                    value == '__personal__' ? null : value;
                ref.invalidate(projectsProvider);
              }
            },
            itemBuilder: (_) {
              final items = <PopupMenuEntry<String>>[];
              items.add(PopupMenuItem(
                value: '__personal__',
                child: Row(children: [
                  if (currentOrgId == null)
                    const Icon(LucideIcons.check, size: 12, color: Colors.white70)
                  else
                    const SizedBox(width: 12),
                  const SizedBox(width: 8),
                  const Text('Organization',
                      style: TextStyle(color: Colors.white70, fontSize: 13)),
                ]),
              ));
              for (final org in orgs) {
                final id = org['\$id'] as String? ?? '';
                final name = org['name'] as String? ?? 'Organization';
                items.add(PopupMenuItem(
                  value: id,
                  child: Row(children: [
                    if (currentOrgId == id)
                      const Icon(LucideIcons.check, size: 12, color: Colors.white70)
                    else
                      const SizedBox(width: 12),
                    const SizedBox(width: 8),
                    Text(name,
                        style:
                            const TextStyle(color: Colors.white70, fontSize: 13)),
                  ]),
                ));
              }
              items.add(const PopupMenuDivider());
              items.add(const PopupMenuItem(
                value: '__create__',
                child: Row(
                  children: [
                    Icon(LucideIcons.plus, size: 14, color: _dimText),
                    SizedBox(width: 8),
                    Text('Create organization',
                        style: TextStyle(color: _dimText, fontSize: 13)),
                  ],
                ),
              ));
              return items;
            },
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
          ),

          const Spacer(),

          // Feedback
          const FeedbackButton(),
          const SizedBox(width: 2),

          // Support
          const SupportButton(),
          const SizedBox(width: 2),

          // Theme toggle
          const ThemeToggleButton(),
          const SizedBox(width: 4),

          // Search icon
          Tooltip(
            message: '⌘K',
            child: InkWell(
              onTap: () => _openSearch(projects, orgs),
              borderRadius: BorderRadius.circular(8),
              child: Container(
                width: 34,
                height: 34,
                child: Icon(
                  LucideIcons.search,
                  size: 17,
                  color: Colors.white.withOpacity(0.45),
                ),
              ),
            ),
          ),

          const SizedBox(width: 8),

          // User avatar
          Tooltip(
            message: userEmail,
            child: GestureDetector(
              onTap: () => context.go('/account'),
              child: Container(
                width: 32,
                height: 32,
                decoration: const BoxDecoration(
                  color: _accent,
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
          ),

          const SizedBox(width: 4),
        ],
      ),
    );
  }

  String _buildInitials(String name, String email) {
    if (name.isNotEmpty) {
      final parts = name.trim().split(RegExp(r'\s+'));
      if (parts.length >= 2) {
        return '${parts[0][0]}${parts[1][0]}'.toUpperCase();
      }
      return name[0].toUpperCase();
    }
    if (email.isNotEmpty) return email[0].toUpperCase();
    return '?';
  }

  // ---------------------------------------------------------------------------
  // Member avatars
  // ---------------------------------------------------------------------------

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
          backgroundColor: _accent.withOpacity(0.25),
          child: Text(
            name.isNotEmpty ? name[0].toUpperCase() : '?',
            style: const TextStyle(
              color: _accent,
              fontSize: 11,
              fontWeight: FontWeight.w600,
            ),
          ),
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
          child: CircularProgressIndicator(color: _accent),
        ),
      ),
      error: (e, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(64),
          child: Text('Error loading projects: $e',
              style: const TextStyle(color: Colors.white70)),
        ),
      ),
      data: (allProjects) {
        // Filter by search
        final query = _searchCtrl.text.trim().toLowerCase();
        final filtered = query.isEmpty
            ? allProjects
            : allProjects.where((p) {
                final name = (p['name'] ?? '').toString().toLowerCase();
                final id = (p['\$id'] ?? '').toString().toLowerCase();
                return name.contains(query) || id.contains(query);
              }).toList();

        final total = filtered.length;
        final totalPages = (total / _perPage).ceil().clamp(1, 999999);
        final pageItems =
            filtered.skip((_page - 1) * _perPage).take(_perPage).toList();

        return Column(
          children: [
            // Search bar + create button
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
                  padding: const EdgeInsets.symmetric(
                      horizontal: 16, vertical: 10),
                ),
                icon: const Icon(LucideIcons.plus, size: 16),
                label: const Text('Create project',
                    style: TextStyle(fontSize: 13)),
                onPressed: _showCreateProjectDialog,
              ),
            ),
            const SizedBox(height: 16),

            // 2-column grid of project cards
            LayoutBuilder(builder: (context, constraints) {
              final cardWidth = (constraints.maxWidth - 16) / 2;
              return Wrap(
                spacing: 16,
                runSpacing: 16,
                children: [
                  ...pageItems.map((p) => _ProjectCard(
                        project: p,
                        width: cardWidth,
                        onTap: () => _selectProject(p),
                      )),
                  _CreateProjectPlaceholder(
                    width: cardWidth,
                    onTap: _showCreateProjectDialog,
                  ),
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
          ],
        );
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
          child: CircularProgressIndicator(color: _accent),
        ),
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            const Text('Members',
                style: TextStyle(
                    color: Colors.white,
                    fontSize: 16,
                    fontWeight: FontWeight.w600)),
            const Spacer(),
            OutlinedButton.icon(
              style: OutlinedButton.styleFrom(
                foregroundColor: Colors.white,
                backgroundColor: _accent.withOpacity(0.15),
                side: const BorderSide(color: _accent, width: 1),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
                padding: const EdgeInsets.symmetric(
                    horizontal: 16, vertical: 10),
              ),
              icon: const Icon(LucideIcons.userPlus, size: 16),
              label: const Text('Invite member',
                  style: TextStyle(fontSize: 13)),
              onPressed: _showInviteMemberDialog,
            ),
          ],
        ),
        const SizedBox(height: 16),
        if (_members.isEmpty)
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(32),
            decoration: BoxDecoration(
              color: _cardColor,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: _border),
            ),
            child: const Center(
              child: Text('No members yet. Invite someone to get started.',
                  style: TextStyle(color: _dimText, fontSize: 14)),
            ),
          )
        else
          ..._members.map((m) => Container(
                width: double.infinity,
                margin: const EdgeInsets.only(bottom: 8),
                padding: const EdgeInsets.symmetric(
                    horizontal: 20, vertical: 14),
                decoration: BoxDecoration(
                  color: _cardColor,
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(color: _border),
                ),
                child: Row(
                  children: [
                    CircleAvatar(
                      radius: 16,
                      backgroundColor: _accent.withOpacity(0.25),
                      child: Text(
                        _initial(m['name'] ?? m['email'] ?? '?'),
                        style: const TextStyle(
                          color: _accent,
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
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
                    Text(m['role'] ?? 'member',
                        style: const TextStyle(
                            color: _subtleText, fontSize: 12)),
                  ],
                ),
              )),
      ],
    );
  }

  String _initial(String s) => s.isNotEmpty ? s[0].toUpperCase() : '?';

  // ---------------------------------------------------------------------------
  // Settings tab
  // ---------------------------------------------------------------------------

  Widget _buildSettingsTab(String orgName) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Update org name
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(24),
          decoration: BoxDecoration(
            color: _cardColor,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: _border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('Organization name',
                  style: TextStyle(
                      color: Colors.white,
                      fontSize: 16,
                      fontWeight: FontWeight.w600)),
              const SizedBox(height: 4),
              Text('Update your organization display name.',
                  style: TextStyle(
                      color: Colors.white.withOpacity(0.5), fontSize: 13)),
              const SizedBox(height: 16),
              FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: Colors.white.withOpacity(0.08),
                  foregroundColor: Colors.white,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 20, vertical: 12),
                ),
                onPressed: () => _showUpdateOrgDialog(orgName),
                child:
                    const Text('Update name', style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
        ),
        const SizedBox(height: 24),

        // Danger zone
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(24),
          decoration: BoxDecoration(
            color: _cardColor,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: Colors.red.withOpacity(0.3)),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('Danger zone',
                  style: TextStyle(
                      color: Colors.red,
                      fontSize: 16,
                      fontWeight: FontWeight.w600)),
              const SizedBox(height: 8),
              Text(
                'Permanently delete this organization and all its projects. '
                'This action cannot be undone.',
                style: TextStyle(
                    color: Colors.white.withOpacity(0.5), fontSize: 13),
              ),
              const SizedBox(height: 16),
              FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: Colors.red.withOpacity(0.15),
                  foregroundColor: Colors.red,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8),
                    side: const BorderSide(color: Colors.red, width: 0.5),
                  ),
                ),
                onPressed: _deleteOrg,
                child: const Text('Delete organization'),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

// =============================================================================
// Project card
// =============================================================================

class _ProjectCard extends StatelessWidget {
  final Map<String, dynamic> project;
  final double width;
  final VoidCallback onTap;

  const _ProjectCard({
    required this.project,
    required this.width,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final name = project['name'] ?? 'Untitled';
    final id = project['\$id'] ?? '';
    final platforms = List<String>.from(project['platforms'] ?? []);

    return GestureDetector(
      onTap: onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          width: width,
          height: 180,
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: _cardColor,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: _border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // App icon placeholder
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: _accent.withOpacity(0.15),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: const Center(
                  child: Icon(LucideIcons.layers, size: 20, color: _accent),
                ),
              ),
              const SizedBox(height: 14),
              Text(
                name,
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 16,
                  fontWeight: FontWeight.w600,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 4),
              Text(
                id,
                style: const TextStyle(color: _subtleText, fontSize: 12),
                overflow: TextOverflow.ellipsis,
              ),
              if (platforms.isNotEmpty) ...[
                const SizedBox(height: 12),
                Wrap(
                  spacing: 6,
                  runSpacing: 6,
                  children: platforms
                      .map((p) => Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 8, vertical: 3),
                            decoration: BoxDecoration(
                              color: Colors.white.withOpacity(0.06),
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: Text(p,
                                style: const TextStyle(
                                    color: _dimText, fontSize: 11)),
                          ))
                      .toList(),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

// =============================================================================
// Create project placeholder card
// =============================================================================

class _CreateProjectPlaceholder extends StatelessWidget {
  final double width;
  final VoidCallback onTap;

  const _CreateProjectPlaceholder({
    required this.width,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: CustomPaint(
          painter: _DashedBorderPainter(
            color: Colors.white.withOpacity(0.12),
            borderRadius: 12,
            dashWidth: 6,
            dashSpace: 4,
            strokeWidth: 1,
          ),
          child: Container(
            width: width,
            height: 180,
            padding: const EdgeInsets.all(20),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: Colors.white.withOpacity(0.06),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: const Center(
                    child: Icon(LucideIcons.plus, size: 20, color: _dimText),
                  ),
                ),
                const SizedBox(height: 14),
                const Text(
                  'Create a new project',
                  style: TextStyle(
                    color: _dimText,
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
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
      Radius.circular(borderRadius),
    );
    final path = Path()..addRRect(rrect);
    final metrics = path.computeMetrics();

    for (final metric in metrics) {
      double distance = 0;
      while (distance < metric.length) {
        final end = (distance + dashWidth).clamp(0.0, metric.length);
        final segment = metric.extractPath(distance, end);
        canvas.drawPath(segment, paint);
        distance += dashWidth + dashSpace;
      }
    }
  }

  @override
  bool shouldRepaint(covariant _DashedBorderPainter old) =>
      color != old.color || borderRadius != old.borderRadius;
}
