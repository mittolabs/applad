import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/search_list.dart';

// ── Responsive padding ────────────────────────────────────────────────────────

double _hPad(BuildContext context) {
  final w = MediaQuery.sizeOf(context).width;
  if (w > 1400) return 80.0;
  if (w > 1100) return 60.0;
  return 40.0;
}

// ── Colors ────────────────────────────────────────────────────────────────────

const _bg = Color(0xFF0B0B0F);
const _surface = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _border = Color(0x14FFFFFF);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);

// ── Providers ─────────────────────────────────────────────────────────────────

final _authTabProvider = StateProvider<int>((ref) => 0);
final _userSearchProvider = StateProvider<String>((ref) => '');
final _userPerPageProvider = StateProvider<int>((ref) => 12);
final _userPageProvider = StateProvider<int>((ref) => 1);

final usersProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final projectId = ref.watch(currentProjectProvider);
  if (projectId == null) return {'users': [], 'total': 0};
  api.setProject(projectId);
  final search = ref.watch(_userSearchProvider);
  final limit = ref.watch(_userPerPageProvider);
  final page = ref.watch(_userPageProvider);
  final offset = (page - 1) * limit;
  final params = <String, dynamic>{'limit': limit, 'offset': offset};
  if (search.isNotEmpty) params['search'] = search;
  final res = await api.get('/users', params: params);
  return res.data as Map<String, dynamic>;
});

final teamsProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final projectId = ref.watch(currentProjectProvider);
  if (projectId == null) return {'teams': [], 'total': 0};
  api.setProject(projectId);
  final res = await api.get('/teams');
  return res.data as Map<String, dynamic>;
});

// ── Page ──────────────────────────────────────────────────────────────────────

class AuthPage extends ConsumerStatefulWidget {
  const AuthPage({super.key});

  @override
  ConsumerState<AuthPage> createState() => _AuthPageState();
}

class _AuthPageState extends ConsumerState<AuthPage> {
  final _searchCtrl = TextEditingController();

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  void _doSearch() {
    ref.read(_userSearchProvider.notifier).state = _searchCtrl.text.trim();
    ref.read(_userPageProvider.notifier).state = 1;
  }

  @override
  Widget build(BuildContext context) {
    final tab = ref.watch(_authTabProvider);

    return Scaffold(
      backgroundColor: _bg,
      body: Padding(
        padding: EdgeInsets.symmetric(horizontal: _hPad(context)),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            const Text('Auth',
                style: TextStyle(
                    color: Colors.white,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 24),
            PageTabs(
              tabs: const [
                'Users',
                'Teams',
                'Providers',
                'Security',
                'Templates',
                'Usage',
                'Settings',
              ],
              selected: tab,
              onChanged: (i) => ref.read(_authTabProvider.notifier).state = i,
            ),
            const SizedBox(height: 20),
            Expanded(child: _tabBody(tab)),
          ],
        ),
      ),
    );
  }

  Widget _tabBody(int tab) {
    switch (tab) {
      case 0:
        return _UsersTab(searchCtrl: _searchCtrl, onSearch: _doSearch);
      case 1:
        return const _TeamsTab();
      case 2:
        return const _ProvidersTab();
      case 3:
        return const _SecurityTab();
      case 4:
        return const _TemplatesTab();
      case 5:
        return const _UsageTab();
      case 6:
        return const _SettingsTab();
      default:
        return const SizedBox.shrink();
    }
  }
}

// ── Users tab ─────────────────────────────────────────────────────────────────

class _UsersTab extends ConsumerWidget {
  final TextEditingController searchCtrl;
  final VoidCallback onSearch;

  const _UsersTab({required this.searchCtrl, required this.onSearch});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final usersAsync = ref.watch(usersProvider);
    final perPage = ref.watch(_userPerPageProvider);
    final currentPage = ref.watch(_userPageProvider);

    return usersAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _errorView(e.toString(), () => ref.invalidate(usersProvider)),
      data: (data) {
        final users = List<Map<String, dynamic>>.from(data['users'] ?? []);
        final total = data['total'] as int? ?? 0;

        return Column(
          children: [
            // Search + Create button row
            Row(
              children: [
                SizedBox(
                  width: 280,
                  child: TextField(
                    controller: searchCtrl,
                    onSubmitted: (_) => onSearch(),
                    style: const TextStyle(fontSize: 13, color: Colors.white),
                    decoration: InputDecoration(
                      hintText: 'Search by name, email, or ID',
                      hintStyle: const TextStyle(
                          color: _subtleText, fontSize: 13),
                      prefixIcon: const Padding(
                        padding: EdgeInsets.only(left: 10, right: 6),
                        child: Icon(Icons.search,
                            size: 16, color: _subtleText),
                      ),
                      prefixIconConstraints:
                          const BoxConstraints(minWidth: 32, minHeight: 0),
                      filled: true,
                      fillColor: const Color(0x0AFFFFFF),
                      isDense: true,
                      contentPadding: const EdgeInsets.symmetric(
                          vertical: 10, horizontal: 12),
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(
                            color: Colors.white.withOpacity(0.08)),
                      ),
                      enabledBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(
                            color: Colors.white.withOpacity(0.08)),
                      ),
                      focusedBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide:
                            const BorderSide(color: _accent),
                      ),
                    ),
                  ),
                ),
                const Spacer(),
                FilledButton.icon(
                  style: FilledButton.styleFrom(
                    backgroundColor: _accent,
                    padding: const EdgeInsets.symmetric(
                        horizontal: 14, vertical: 8),
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                  ),
                  onPressed: () => _showCreateUserDialog(context, ref),
                  icon: const Icon(LucideIcons.plus, size: 14),
                  label: const Text('Create user',
                      style: TextStyle(fontSize: 12)),
                ),
              ],
            ),
            const SizedBox(height: 16),

            // Table
            Expanded(
              child: Column(
                children: [
                  _TableHeader(),
                  Expanded(
                    child: users.isEmpty
                        ? _UsersEmptyState(
                            onCreate: () =>
                                _showCreateUserDialog(context, ref))
                        : ListView.builder(
                            itemCount: users.length,
                            itemBuilder: (context, i) =>
                                _UserRow(user: users[i], ref: ref),
                          ),
                  ),
                ],
              ),
            ),

            // Footer
            SearchListFooter(
              total: total,
              perPage: perPage,
              currentPage: currentPage,
              onPrev: () =>
                  ref.read(_userPageProvider.notifier).update((s) => s - 1),
              onNext: () =>
                  ref.read(_userPageProvider.notifier).update((s) => s + 1),
              onPerPageChanged: (v) {
                ref.read(_userPerPageProvider.notifier).state = v;
                ref.read(_userPageProvider.notifier).state = 1;
              },
            ),
            const SizedBox(height: 8),
          ],
        );
      },
    );
  }

  void _showCreateUserDialog(BuildContext context, WidgetRef ref) {
    final emailCtrl = TextEditingController();
    final passCtrl = TextEditingController();
    final nameCtrl = TextEditingController();

    showAppDialog(
      context: context,
      title: 'Create user',
      subtitle: 'Add a new user to your project',
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          AppDialogField(
              controller: nameCtrl,
              label: 'Name',
              hint: 'Full name',
              autofocus: true),
          const SizedBox(height: 12),
          AppDialogField(
              controller: emailCtrl,
              label: 'Email',
              hint: 'user@example.com'),
          const SizedBox(height: 12),
          AppDialogField(
              controller: passCtrl,
              label: 'Password',
              hint: 'At least 8 characters'),
        ],
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            final api = ref.read(apiClientProvider);
            await api.post('/users', data: {
              'userId': 'unique()',
              'email': emailCtrl.text,
              'password': passCtrl.text,
              'name': nameCtrl.text,
            });
            if (context.mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
            ref.invalidate(usersProvider);
          },
        ),
      ],
    );
  }
}

// ── Empty state ───────────────────────────────────────────────────────────────

class _UsersEmptyState extends StatelessWidget {
  final VoidCallback onCreate;
  const _UsersEmptyState({required this.onCreate});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.04),
              borderRadius: BorderRadius.circular(12),
            ),
            child: const Icon(LucideIcons.users, size: 22, color: _subtleText),
          ),
          const SizedBox(height: 16),
          const Text('No users yet',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          const Text(
              'Add users manually or let them sign up through your app.',
              style: TextStyle(color: _dimText, fontSize: 13)),
          const SizedBox(height: 16),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              padding:
                  const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            onPressed: onCreate,
            child:
                const Text('Create user', style: TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }
}

// ── Table header row ──────────────────────────────────────────────────────────

class _TableHeader extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    const style = TextStyle(
        color: _dimText, fontSize: 12, fontWeight: FontWeight.w500);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      decoration: BoxDecoration(
        border: Border(
            bottom: BorderSide(color: Colors.white.withOpacity(0.06))),
      ),
      child: const Row(
        children: [
          Expanded(flex: 3, child: Text('User ID', style: style)),
          Expanded(flex: 3, child: Text('Name', style: style)),
          Expanded(flex: 4, child: Text('Email', style: style)),
          Expanded(flex: 2, child: Text('Status', style: style)),
          Expanded(flex: 2, child: Text('Joined', style: style)),
          SizedBox(width: 40),
        ],
      ),
    );
  }
}

// ── Single user row ───────────────────────────────────────────────────────────

class _UserRow extends StatefulWidget {
  final Map<String, dynamic> user;
  final WidgetRef ref;

  const _UserRow({required this.user, required this.ref});

  @override
  State<_UserRow> createState() => _UserRowState();
}

class _UserRowState extends State<_UserRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final u = widget.user;
    final id = (u['\$id'] as String? ?? '');
    final name = (u['name'] as String? ?? '').trim();
    final email = u['email'] as String? ?? '';
    final status = u['status'] as bool? ?? false;
    final emailVerification = u['emailVerification'] as bool? ?? false;
    final createdAt = u['\$createdAt'] as String? ?? '';

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: _hovered ? Colors.white.withOpacity(0.02) : null,
          border: Border(
              bottom: BorderSide(color: Colors.white.withOpacity(0.04))),
        ),
        child: Row(
          children: [
            // User ID
            Expanded(
              flex: 3,
              child: Row(
                children: [
                  Icon(LucideIcons.user, size: 14, color: _accent),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      id,
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 13,
                          fontFamily: 'monospace'),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
              ),
            ),

            // Name
            Expanded(
              flex: 3,
              child: Text(
                name.isNotEmpty ? name : 'Anonymous',
                style: const TextStyle(color: Colors.white, fontSize: 13),
                overflow: TextOverflow.ellipsis,
              ),
            ),

            // Email
            Expanded(
              flex: 4,
              child: Text(
                email.isNotEmpty ? email : '—',
                style:
                    const TextStyle(color: _dimText, fontSize: 12),
                overflow: TextOverflow.ellipsis,
              ),
            ),

            // Status badge
            Expanded(
              flex: 2,
              child: _StatusBadge(
                  active: status, verified: emailVerification),
            ),

            // Joined
            Expanded(
              flex: 2,
              child: Text(
                _relativeTime(createdAt),
                style: const TextStyle(color: _dimText, fontSize: 12),
              ),
            ),

            // Actions
            SizedBox(
              width: 40,
              child: _hovered
                  ? GestureDetector(
                      onTap: () =>
                          _deleteUser(context, widget.ref, id),
                      child: const Icon(LucideIcons.trash2,
                          size: 14, color: _subtleText),
                    )
                  : const SizedBox.shrink(),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _deleteUser(
      BuildContext context, WidgetRef ref, String userId) async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete user',
      content: Text(
        'Are you sure you want to delete this user? This action cannot be undone.',
        style: TextStyle(color: Colors.white.withOpacity(0.7)),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Delete',
          destructive: true,
          onTap: () => Navigator.of(context, rootNavigator: true).pop(true),
        ),
      ],
    );
    if (confirmed == true) {
      final api = ref.read(apiClientProvider);
      await api.delete('/users/$userId');
      ref.invalidate(usersProvider);
    }
  }
}

// ── Avatar widget ─────────────────────────────────────────────────────────────

class _Avatar extends StatelessWidget {
  final String name;
  final String email;
  final double size;

  const _Avatar({required this.name, required this.email, this.size = 32});

  @override
  Widget build(BuildContext context) {
    final initials = _initials(name, email);
    final color = _colorFor(name.isNotEmpty ? name : email);
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
      child: Center(
        child: Text(initials,
            style: TextStyle(
                color: Colors.white,
                fontSize: size * 0.4,
                fontWeight: FontWeight.w600)),
      ),
    );
  }

  String _initials(String name, String email) {
    if (name.trim().isNotEmpty) {
      final parts = name.trim().split(RegExp(r'\s+'));
      if (parts.length >= 2) return '${parts[0][0]}${parts[1][0]}'.toUpperCase();
      return parts[0][0].toUpperCase();
    }
    return email.isNotEmpty ? email[0].toUpperCase() : '?';
  }

  Color _colorFor(String s) {
    const colors = [
      Color(0xFF3472A4),
      Color(0xFF7C3AED),
      Color(0xFF059669),
      Color(0xFFD97706),
      Color(0xFFDC2626),
      Color(0xFF0891B2),
      Color(0xFF7C2D12),
      Color(0xFF1D4ED8),
    ];
    if (s.isEmpty) return colors[0];
    int hash = 0;
    for (final c in s.codeUnits) {
      hash = (hash * 31 + c) & 0x7FFFFFFF;
    }
    return colors[hash % colors.length];
  }
}

// ── Status badge ──────────────────────────────────────────────────────────────

class _StatusBadge extends StatelessWidget {
  final bool active;
  final bool verified;

  const _StatusBadge({required this.active, required this.verified});

  @override
  Widget build(BuildContext context) {
    if (!active) {
      return _badge('Disabled', const Color(0xFF374151), const Color(0xFF6B7280));
    }
    if (verified) {
      return _badge('Verified', const Color(0xFF064E3B), const Color(0xFF34D399));
    }
    return _badge('Unverified', const Color(0xFF1F2937), const Color(0xFF9CA3AF));
  }

  Widget _badge(String label, Color bg, Color fg) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
          decoration: BoxDecoration(
            color: bg,
            borderRadius: BorderRadius.circular(4),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 5,
                height: 5,
                decoration: BoxDecoration(color: fg, shape: BoxShape.circle),
              ),
              const SizedBox(width: 5),
              Text(label,
                  style: TextStyle(
                      color: fg, fontSize: 11, fontWeight: FontWeight.w500)),
            ],
          ),
        ),
      ],
    );
  }
}

// ── Teams tab ─────────────────────────────────────────────────────────────────

class _TeamsTab extends ConsumerWidget {
  const _TeamsTab();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final teamsAsync = ref.watch(teamsProvider);

    return teamsAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) =>
          _errorView(e.toString(), () => ref.invalidate(teamsProvider)),
      data: (data) {
        final teams = List<Map<String, dynamic>>.from(data['teams'] ?? []);
        final total = data['total'] as int? ?? 0;

        if (teams.isEmpty) {
          return _emptyState(
            icon: LucideIcons.users,
            title: 'No teams yet',
            description: 'Create a team to group users together.',
          );
        }

        return Column(
          children: [
            // Search + Create
            Row(
              children: [
                Text('$total team${total == 1 ? '' : 's'}',
                    style: const TextStyle(color: _dimText, fontSize: 13)),
                const Spacer(),
                FilledButton.icon(
                  style: FilledButton.styleFrom(
                    backgroundColor: _accent,
                    padding: const EdgeInsets.symmetric(
                        horizontal: 14, vertical: 8),
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                  ),
                  onPressed: () {},
                  icon: const Icon(LucideIcons.plus, size: 14),
                  label: const Text('Create team',
                      style: TextStyle(fontSize: 12)),
                ),
              ],
            ),
            const SizedBox(height: 16),
            // Table header
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              decoration: BoxDecoration(
                border: Border(
                    bottom: BorderSide(color: Colors.white.withOpacity(0.06))),
              ),
              child: const Row(
                children: [
                  Expanded(flex: 3, child: Text('Team ID', style: TextStyle(color: _dimText, fontSize: 12, fontWeight: FontWeight.w500))),
                  Expanded(flex: 3, child: Text('Name', style: TextStyle(color: _dimText, fontSize: 12, fontWeight: FontWeight.w500))),
                  Expanded(flex: 2, child: Text('Members', style: TextStyle(color: _dimText, fontSize: 12, fontWeight: FontWeight.w500))),
                  Expanded(flex: 2, child: Text('Created', style: TextStyle(color: _dimText, fontSize: 12, fontWeight: FontWeight.w500))),
                ],
              ),
            ),
            Expanded(
              child: ListView.builder(
                itemCount: teams.length,
                itemBuilder: (context, i) {
                  final t = teams[i];
                  final id = t['\$id'] as String? ?? '';
                  final name = t['name'] as String? ?? 'Unnamed';
                  final members = t['total'] as int? ?? 0;
                  final createdAt = t['\$createdAt'] as String? ?? '';
                  return _TeamRow(
                    id: id,
                    name: name,
                    members: members,
                    createdAt: createdAt,
                  );
                },
              ),
            ),
          ],
        );
      },
    );
  }
}

class _TeamRow extends StatefulWidget {
  final String id;
  final String name;
  final int members;
  final String createdAt;

  const _TeamRow({
    required this.id,
    required this.name,
    required this.members,
    required this.createdAt,
  });

  @override
  State<_TeamRow> createState() => _TeamRowState();
}

class _TeamRowState extends State<_TeamRow> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: _hovered ? Colors.white.withOpacity(0.02) : null,
          border: Border(
              bottom: BorderSide(color: Colors.white.withOpacity(0.04))),
        ),
        child: Row(
          children: [
            Expanded(
              flex: 3,
              child: Row(
                children: [
                  Icon(LucideIcons.users, size: 14, color: _accent),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(widget.id,
                        style: const TextStyle(
                            color: Colors.white,
                            fontSize: 13,
                            fontFamily: 'monospace'),
                        overflow: TextOverflow.ellipsis),
                  ),
                ],
              ),
            ),
            Expanded(
              flex: 3,
              child: Text(widget.name,
                  style:
                      const TextStyle(color: Colors.white, fontSize: 13)),
            ),
            Expanded(
              flex: 2,
              child: Text('${widget.members}',
                  style:
                      const TextStyle(color: _dimText, fontSize: 13)),
            ),
            Expanded(
              flex: 2,
              child: Text(_relativeTime(widget.createdAt),
                  style:
                      const TextStyle(color: _dimText, fontSize: 12)),
            ),
          ],
        ),
      ),
    );
  }
}

// ── Providers tab ─────────────────────────────────────────────────────────────

class _ProvidersTab extends StatelessWidget {
  const _ProvidersTab();

  static const _providers = [
    {'id': 'email', 'name': 'Email / Password', 'icon': LucideIcons.mail, 'enabled': true},
    {'id': 'magic', 'name': 'Magic URL', 'icon': LucideIcons.link, 'enabled': false},
    {'id': 'anonymous', 'name': 'Anonymous', 'icon': LucideIcons.userX, 'enabled': true},
    {'id': 'google', 'name': 'Google', 'icon': LucideIcons.chrome, 'enabled': false},
    {'id': 'github', 'name': 'GitHub', 'icon': LucideIcons.github, 'enabled': false},
    {'id': 'apple', 'name': 'Apple', 'icon': LucideIcons.smartphone, 'enabled': false},
    {'id': 'facebook', 'name': 'Facebook', 'icon': LucideIcons.facebook, 'enabled': false},
    {'id': 'discord', 'name': 'Discord', 'icon': LucideIcons.messageSquare, 'enabled': false},
    {'id': 'twitter', 'name': 'Twitter / X', 'icon': LucideIcons.twitter, 'enabled': false},
    {'id': 'microsoft', 'name': 'Microsoft', 'icon': LucideIcons.grid, 'enabled': false},
    {'id': 'slack', 'name': 'Slack', 'icon': LucideIcons.hash, 'enabled': false},
    {'id': 'spotify', 'name': 'Spotify', 'icon': LucideIcons.music, 'enabled': false},
  ];

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Auth providers',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
          const Text(
            'Enable sign-in methods for your project users.',
            style: TextStyle(color: _dimText, fontSize: 13),
          ),
          const SizedBox(height: 20),
          ...List.generate(_providers.length, (i) {
            final p = _providers[i];
            final enabled = p['enabled'] as bool;
            return _ProviderRow(
              icon: p['icon'] as IconData,
              name: p['name'] as String,
              enabled: enabled,
            );
          }),
        ],
      ),
    );
  }
}

class _ProviderRow extends StatefulWidget {
  final IconData icon;
  final String name;
  final bool enabled;

  const _ProviderRow(
      {required this.icon, required this.name, required this.enabled});

  @override
  State<_ProviderRow> createState() => _ProviderRowState();
}

class _ProviderRowState extends State<_ProviderRow> {
  late bool _enabled;

  @override
  void initState() {
    super.initState();
    _enabled = widget.enabled;
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      decoration: BoxDecoration(
        color: _surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
      ),
      child: Row(
        children: [
          Icon(widget.icon, size: 16, color: _enabled ? Colors.white : _dimText),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              widget.name,
              style: TextStyle(
                  color: _enabled ? Colors.white : _dimText,
                  fontSize: 14),
            ),
          ),
          if (_enabled)
            Container(
              margin: const EdgeInsets.only(right: 12),
              padding:
                  const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
              decoration: BoxDecoration(
                color: const Color(0xFF064E3B),
                borderRadius: BorderRadius.circular(4),
              ),
              child: const Text('Enabled',
                  style: TextStyle(
                      color: Color(0xFF34D399),
                      fontSize: 11,
                      fontWeight: FontWeight.w500)),
            ),
          Switch(
            value: _enabled,
            onChanged: (v) => setState(() => _enabled = v),
            activeColor: _accent,
          ),
        ],
      ),
    );
  }
}

// ── Security tab ──────────────────────────────────────────────────────────────

class _SecurityTab extends StatelessWidget {
  const _SecurityTab();

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Security',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
          const Text(
            'Configure security policies for your project.',
            style: TextStyle(color: _dimText, fontSize: 13),
          ),
          const SizedBox(height: 20),
          _settingRow('Session duration', '365 days',
              'How long sessions remain valid before expiring'),
          _settingRow('Max sessions per user', '10',
              'Maximum number of concurrent active sessions'),
          _settingRow('Password minimum length', '8 characters',
              'Minimum number of characters required for passwords'),
          _settingRow('Password history', 'Disabled',
              'Prevent reuse of previously used passwords'),
          _settingRow('MFA enforcement', 'Optional',
              'Require multi-factor authentication for all users'),
          _settingRow('Personal data check', 'Disabled',
              'Reject passwords that contain personal information'),
          _settingRow('Dictionary attack check', 'Disabled',
              'Reject commonly used or compromised passwords'),
        ],
      ),
    );
  }

  Widget _settingRow(String label, String value, String description) {
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: _surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
      ),
      child: Row(children: [
        Expanded(
            child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
              Text(label,
                  style: const TextStyle(
                      color: Colors.white,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
              const SizedBox(height: 2),
              Text(description,
                  style: const TextStyle(color: _subtleText, fontSize: 12)),
            ])),
        Container(
          padding:
              const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          decoration: BoxDecoration(
            color: Colors.white.withOpacity(0.06),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Text(value,
              style: const TextStyle(color: _dimText, fontSize: 13)),
        ),
      ]),
    );
  }
}

// ── Templates tab ─────────────────────────────────────────────────────────────

class _TemplatesTab extends StatelessWidget {
  const _TemplatesTab();

  static const _templates = [
    {
      'name': 'Email verification',
      'type': 'email',
      'description': 'Sent when a user signs up to verify their email address.',
    },
    {
      'name': 'Magic URL',
      'type': 'email',
      'description': 'Passwordless sign-in link sent to the user\'s email.',
    },
    {
      'name': 'Password recovery',
      'type': 'email',
      'description': 'Sent when a user requests a password reset.',
    },
    {
      'name': 'Invitation',
      'type': 'email',
      'description': 'Sent when a user is invited to join a team.',
    },
    {
      'name': 'OTP verification',
      'type': 'sms',
      'description': 'SMS code sent for phone number verification.',
    },
  ];

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Email & SMS templates',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
          const Text(
            'Customize the messages sent to your users during authentication flows.',
            style: TextStyle(color: _dimText, fontSize: 13),
          ),
          const SizedBox(height: 20),
          ..._templates.map((t) => _TemplateRow(
                name: t['name']!,
                type: t['type']!,
                description: t['description']!,
              )),
        ],
      ),
    );
  }
}

class _TemplateRow extends StatelessWidget {
  final String name;
  final String type;
  final String description;

  const _TemplateRow(
      {required this.name, required this.type, required this.description});

  @override
  Widget build(BuildContext context) {
    final isEmail = type == 'email';
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: _surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.05),
              borderRadius: BorderRadius.circular(6),
            ),
            child: Icon(
              isEmail ? LucideIcons.mail : LucideIcons.smartphone,
              size: 14,
              color: _dimText,
            ),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(name,
                    style: const TextStyle(
                        color: Colors.white,
                        fontSize: 14,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 2),
                Text(description,
                    style:
                        const TextStyle(color: _subtleText, fontSize: 12)),
              ],
            ),
          ),
          OutlinedButton(
            style: OutlinedButton.styleFrom(
              foregroundColor: Colors.white70,
              side: BorderSide(color: Colors.white.withOpacity(0.12)),
              padding:
                  const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(6)),
            ),
            onPressed: () {},
            child: const Text('Edit', style: TextStyle(fontSize: 12)),
          ),
        ],
      ),
    );
  }
}

// ── Usage tab ─────────────────────────────────────────────────────────────────

class _UsageTab extends StatelessWidget {
  const _UsageTab();

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Usage',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
          const Text('Authentication activity for the past 30 days.',
              style: TextStyle(color: _dimText, fontSize: 13)),
          const SizedBox(height: 24),
          Row(
            children: [
              _statCard('Total users', '—', LucideIcons.users),
              const SizedBox(width: 12),
              _statCard('Active sessions', '—', LucideIcons.activity),
              const SizedBox(width: 12),
              _statCard('New signups (30d)', '—', LucideIcons.userPlus),
            ],
          ),
          const SizedBox(height: 24),
          Container(
            height: 200,
            decoration: BoxDecoration(
              color: _surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: _border),
            ),
            child: const Center(
              child: Text('Usage charts coming soon',
                  style: TextStyle(color: _subtleText, fontSize: 13)),
            ),
          ),
        ],
      ),
    );
  }

  Widget _statCard(String label, String value, IconData icon) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: _surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: _border),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(icon, size: 16, color: _dimText),
            const SizedBox(height: 12),
            Text(value,
                style: const TextStyle(
                    color: Colors.white,
                    fontSize: 24,
                    fontWeight: FontWeight.w700)),
            const SizedBox(height: 4),
            Text(label,
                style: const TextStyle(color: _dimText, fontSize: 12)),
          ],
        ),
      ),
    );
  }
}

// ── Settings tab ──────────────────────────────────────────────────────────────

class _SettingsTab extends StatelessWidget {
  const _SettingsTab();

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Auth settings',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
          const Text(
            'Control global authentication behavior for your project.',
            style: TextStyle(color: _dimText, fontSize: 13),
          ),
          const SizedBox(height: 20),
          _toggleRow('Email / Password sign-in', true,
              'Allow users to sign in with email and password'),
          _toggleRow('Anonymous sessions', true,
              'Allow users to create anonymous sessions without credentials'),
          _toggleRow('Email verification required', false,
              'Block access until the user has verified their email'),
          _toggleRow('Phone number sign-in', false,
              'Allow users to sign in with a phone number and OTP'),
          _toggleRow('Limit login attempts', true,
              'Temporarily block users after multiple failed sign-in attempts'),
        ],
      ),
    );
  }

  Widget _toggleRow(String label, bool value, String description) {
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: _surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
      ),
      child: Row(children: [
        Expanded(
            child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
              Text(label,
                  style: const TextStyle(
                      color: Colors.white,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
              const SizedBox(height: 2),
              Text(description,
                  style: const TextStyle(color: _subtleText, fontSize: 12)),
            ])),
        Switch(value: value, onChanged: (_) {}, activeColor: _accent),
      ]),
    );
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────────

Widget _errorView(String message, VoidCallback onRetry) {
  return Center(
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 48,
          height: 48,
          decoration: BoxDecoration(
            color: Colors.white.withOpacity(0.04),
            borderRadius: BorderRadius.circular(12),
          ),
          child: const Icon(LucideIcons.alertTriangle,
              size: 22, color: _subtleText),
        ),
        const SizedBox(height: 16),
        Text('Error: $message',
            style: const TextStyle(
                color: Colors.white,
                fontSize: 15,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 16),
        FilledButton(
          style: FilledButton.styleFrom(
            backgroundColor: _accent,
            padding:
                const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(8)),
          ),
          onPressed: onRetry,
          child: const Text('Retry', style: TextStyle(fontSize: 13)),
        ),
      ],
    ),
  );
}

Widget _emptyState(
    {required IconData icon,
    required String title,
    required String description}) {
  return Center(
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 48,
          height: 48,
          decoration: BoxDecoration(
            color: Colors.white.withOpacity(0.04),
            borderRadius: BorderRadius.circular(12),
          ),
          child: Icon(icon, size: 22, color: _subtleText),
        ),
        const SizedBox(height: 16),
        Text(title,
            style: const TextStyle(
                color: Colors.white,
                fontSize: 15,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        Text(description,
            style: const TextStyle(color: _dimText, fontSize: 13)),
      ],
    ),
  );
}

String _relativeTime(String iso) {
  if (iso.isEmpty) return '—';
  try {
    final dt = DateTime.parse(iso);
    final diff = DateTime.now().difference(dt);
    if (diff.inDays > 365) return '${(diff.inDays / 365).floor()}y ago';
    if (diff.inDays > 30) return '${(diff.inDays / 30).floor()}mo ago';
    if (diff.inDays > 0) return '${diff.inDays}d ago';
    if (diff.inHours > 0) return '${diff.inHours}h ago';
    if (diff.inMinutes > 0) return '${diff.inMinutes}m ago';
    return 'just now';
  } catch (_) {
    return '—';
  }
}
