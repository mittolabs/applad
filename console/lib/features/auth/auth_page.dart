import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/utils/url_utils.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/app_error_state.dart';
import '../../core/widgets/status_chip.dart';


// ── Colors ────────────────────────────────────────────────────────────────────

const _accent = Color(0xFF3472A4);

// ── Providers ─────────────────────────────────────────────────────────────────

final _userSearchProvider = StateProvider<String>((ref) => '');
final _userPerPageProvider = StateProvider<int>((ref) => 12);
final _userPageProvider = StateProvider<int>((ref) => 1);
final _teamSearchProvider = StateProvider<String>((ref) => '');
final _teamPerPageProvider = StateProvider<int>((ref) => 12);
final _teamPageProvider = StateProvider<int>((ref) => 1);

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
  final search = ref.watch(_teamSearchProvider);
  final limit = ref.watch(_teamPerPageProvider);
  final page = ref.watch(_teamPageProvider);
  final offset = (page - 1) * limit;
  final params = <String, dynamic>{'limit': limit, 'offset': offset};
  if (search.isNotEmpty) params['search'] = search;
  final res = await api.get('/teams', params: params);
  return res.data as Map<String, dynamic>;
});

// ── Page ──────────────────────────────────────────────────────────────────────

class AuthPage extends ConsumerStatefulWidget {
  const AuthPage({super.key});

  @override
  ConsumerState<AuthPage> createState() => _AuthPageState();
}

class _AuthPageState extends ConsumerState<AuthPage> {
  static const _tabNames = [
    'users', 'teams', 'security', 'templates', 'usage', 'settings',
  ];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    // Sync ?page= from URL → StateProvider so FutureProviders refetch.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final tabName = tabFromQuery(context, defaultTab: 'users');
      final page = pageFromQuery(context);
      if (tabName == 'users' && ref.read(_userPageProvider) != page) {
        ref.read(_userPageProvider.notifier).state = page;
      } else if (tabName == 'teams' && ref.read(_teamPageProvider) != page) {
        ref.read(_teamPageProvider.notifier).state = page;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final tabName = tabFromQuery(context, defaultTab: 'users');
    final tab = _tabNames.indexOf(tabName).clamp(0, _tabNames.length - 1);

    return Scaffold(
      backgroundColor: colors.background,
      body: Padding(
        padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Text('Auth',
                style: TextStyle(
                    color: colors.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            Text('Manage users, sessions, OAuth providers and access control',
                style: TextStyle(color: colors.textSecondary, fontSize: 13)),
            const SizedBox(height: 20),
            PageTabs(
              tabs: const [
                'Users',
                'Teams',
                'Security',
                'Templates',
                'Usage',
                'Settings',
              ],
              selected: tab,
              onChanged: (i) => context.go(
                withQuery(context, {'tab': _tabNames[i], 'page': null}),
              ),
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
      case 0: return const _UsersTab();
      case 1: return const _TeamsTab();
      case 2: return const _SecurityTab();
      case 3: return const _TemplatesTab();
      case 4: return const _UsageTab();
      case 5: return const _AuthSettingsTab();
      default: return const SizedBox.shrink();
    }
  }
}

// ── Users tab ─────────────────────────────────────────────────────────────────

class _UsersTab extends ConsumerStatefulWidget {
  const _UsersTab();

  @override
  ConsumerState<_UsersTab> createState() => _UsersTabState();
}

class _UsersTabState extends ConsumerState<_UsersTab> {
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
    final usersAsync = ref.watch(usersProvider);
    final perPage = ref.watch(_userPerPageProvider);
    final currentPage = ref.watch(_userPageProvider);

    return usersAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e, onRetry: () => ref.invalidate(usersProvider)),
      data: (data) {
        final users = List<Map<String, dynamic>>.from(data['users'] ?? []);
        final total = data['total'] as int? ?? 0;
        return AppDataTable(
          columns: const [
            AppTableColumn(key: r'$id',        label: 'User ID', flex: 3),
            AppTableColumn(key: 'name',         label: 'Name',    flex: 3),
            AppTableColumn(key: 'email',        label: 'Email',   flex: 4),
            AppTableColumn(key: 'status',       label: 'Status',  flex: 2, sortable: false),
            AppTableColumn(key: r'$createdAt',  label: 'Joined',  flex: 2),
          ],
          rows: users,
          getCellValue: (row, key) => switch (key) {
            r'$id'       => row[r'$id'] as String? ?? '',
            'name'       => (row['name'] as String? ?? '').isEmpty
                                ? 'Anonymous'
                                : row['name'] as String,
            'email'      => row['email'] as String? ?? '',
            'status'     => (row['status'] as bool? ?? false) ? 'Active' : 'Disabled',
            r'$createdAt' => _relativeTime(row[r'$createdAt'] as String? ?? ''),
            _            => '',
          },
          cellBuilder: (row, key) {
            if (key == 'status') {
              final active = row['status'] as bool? ?? false;
              final verified = row['emailVerification'] as bool? ?? false;
              if (!active) return StatusChip.fromStatus('disabled');
              return StatusChip.fromStatus(verified ? 'verified' : 'unverified');
            }
            return null;
          },
          getRowIcon: (_) => LucideIcons.user,
          onDeleteRow: (row) => _deleteUser(row[r'$id'] as String? ?? ''),
          createLabel: 'Create user',
          onCreateTap: () => _showCreateUserDialog(context, ref),
          total: total,
          perPage: perPage,
          currentPage: currentPage,
          onPrev: () {
            final p = ref.read(_userPageProvider) - 1;
            ref.read(_userPageProvider.notifier).state = p;
            context.go(withQuery(context, {'page': p == 1 ? null : '$p'}));
          },
          onNext: () {
            final p = ref.read(_userPageProvider) + 1;
            ref.read(_userPageProvider.notifier).state = p;
            context.go(withQuery(context, {'page': '$p'}));
          },
          onPerPageChanged: (v) {
            ref.read(_userPerPageProvider.notifier).state = v;
            ref.read(_userPageProvider.notifier).state = 1;
          },
          itemLabel: 'users',
          searchController: _searchCtrl,
          onSearch: _doSearch,
          searchHint: 'Search by name, email, or ID',
          emptyIcon: LucideIcons.users,
          emptyTitle: 'No users yet',
          emptySubtitle: 'Add users manually or let them sign up through your app.',
        );
      },
    );
  }

  Future<void> _deleteUser(String userId) async {
    final colors = consoleColors(context);
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete user',
      content: Text(
        'Are you sure you want to delete this user? This action cannot be undone.',
        style: TextStyle(color: colors.textSecondary),
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


// ── Teams tab ─────────────────────────────────────────────────────────────────

class _TeamsTab extends ConsumerStatefulWidget {
  const _TeamsTab();

  @override
  ConsumerState<_TeamsTab> createState() => _TeamsTabState();
}

class _TeamsTabState extends ConsumerState<_TeamsTab> {
  final _searchCtrl = TextEditingController();

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  void _doSearch() {
    ref.read(_teamSearchProvider.notifier).state = _searchCtrl.text.trim();
    ref.read(_teamPageProvider.notifier).state = 1;
  }

  @override
  Widget build(BuildContext context) {
    final teamsAsync = ref.watch(teamsProvider);
    final perPage = ref.watch(_teamPerPageProvider);
    final currentPage = ref.watch(_teamPageProvider);

    return teamsAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e, onRetry: () => ref.invalidate(teamsProvider)),
      data: (data) {
        final teams = List<Map<String, dynamic>>.from(data['teams'] ?? []);
        final total = data['total'] as int? ?? 0;
        return AppDataTable(
          columns: const [
            AppTableColumn(key: r'$id',       label: 'Team ID', flex: 3),
            AppTableColumn(key: 'name',        label: 'Name',    flex: 3),
            AppTableColumn(key: 'total',       label: 'Members', flex: 2),
            AppTableColumn(key: r'$createdAt', label: 'Created', flex: 2),
          ],
          rows: teams,
          getCellValue: (row, key) => switch (key) {
            r'$id'        => row[r'$id'] as String? ?? '',
            'name'        => row['name'] as String? ?? 'Unnamed',
            'total'       => '${row['total'] as int? ?? 0}',
            r'$createdAt' => _relativeTime(row[r'$createdAt'] as String? ?? ''),
            _             => '',
          },
          getRowIcon: (_) => LucideIcons.users,
          createLabel: 'Create team',
          total: total,
          perPage: perPage,
          currentPage: currentPage,
          onPrev: () {
            final p = ref.read(_teamPageProvider) - 1;
            ref.read(_teamPageProvider.notifier).state = p;
            context.go(withQuery(context, {'page': p == 1 ? null : '$p'}));
          },
          onNext: () {
            final p = ref.read(_teamPageProvider) + 1;
            ref.read(_teamPageProvider.notifier).state = p;
            context.go(withQuery(context, {'page': '$p'}));
          },
          onPerPageChanged: (v) {
            ref.read(_teamPerPageProvider.notifier).state = v;
            ref.read(_teamPageProvider.notifier).state = 1;
          },
          itemLabel: 'teams',
          searchController: _searchCtrl,
          onSearch: _doSearch,
          searchHint: 'Search by name or ID',
          emptyIcon: LucideIcons.users,
          emptyTitle: 'No teams yet',
          emptySubtitle: 'Create a team to group users together.',
        );
      },
    );
  }
}

// ── Providers tab ─────────────────────────────────────────────────────────────

// ── Security tab ──────────────────────────────────────────────────────────────

final _authSecurityProvider =
    FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
  final projectId = ref.watch(currentProjectProvider);
  if (projectId == null) return {};
  final res = await api.get('/projects/$projectId/auth/security');
  return res.data as Map<String, dynamic>;
});

class _SecurityTab extends ConsumerStatefulWidget {
  const _SecurityTab();

  @override
  ConsumerState<_SecurityTab> createState() => _SecurityTabState();
}

class _SecurityTabState extends ConsumerState<_SecurityTab> {
  Map<String, dynamic>? _sec;
  bool _saving = false;

  Future<void> _save(Map<String, dynamic> patch) async {
    setState(() => _saving = true);
    try {
      final api = ref.read(apiClientProvider);
      final projectId = ref.read(currentProjectProvider);
      final merged = {...?_sec, ...patch};
      final res =
          await api.put('/projects/$projectId/auth/security', data: merged);
      setState(() => _sec = res.data as Map<String, dynamic>);
      ref.invalidate(_authSecurityProvider);
    } catch (_) {
    } finally {
      setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final async = ref.watch(_authSecurityProvider);

    return async.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => AppErrorState(error: e),
      data: (data) {
        _sec ??= data;
        final sec = _sec!;

        bool b(String k, bool def) =>
            sec[k] is bool ? sec[k] as bool : def;
        int n(String k, int def) =>
            sec[k] is int ? sec[k] as int : (sec[k] != null ? int.tryParse('${sec[k]}') ?? def : def);

        return SingleChildScrollView(
          padding: const EdgeInsets.only(bottom: 32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Security',
                  style: TextStyle(
                      color: colors.textPrimary,
                      fontSize: 16,
                      fontWeight: FontWeight.w600)),
              const SizedBox(height: 4),
              Text('Configure security policies for your project.',
                  style:
                      TextStyle(color: colors.textSecondary, fontSize: 13)),
              const SizedBox(height: 20),

              // Users limit
              _NumberCard(
                label: 'Users limit',
                description:
                    'Maximum number of users that can sign up. Set to 0 for unlimited.',
                value: n('usersLimit', 0),
                onSave: (v) => _save({'usersLimit': v}),
                saving: _saving,
                colors: colors,
              ),

              // Session length
              _NumberCard(
                label: 'Session length (seconds)',
                description:
                    'How long sessions remain valid before expiring.',
                value: n('sessionLengthSeconds', 31536000),
                onSave: (v) => _save({'sessionLengthSeconds': v}),
                saving: _saving,
                colors: colors,
              ),

              // Sessions per user
              _NumberCard(
                label: 'Sessions per user',
                description:
                    'Maximum concurrent active sessions per user. Set to 0 for unlimited.',
                value: n('sessionsPerUser', 10),
                onSave: (v) => _save({'sessionsPerUser': v}),
                saving: _saving,
                colors: colors,
              ),

              // Password min length
              _NumberCard(
                label: 'Password minimum length',
                description:
                    'Minimum number of characters required for passwords.',
                value: n('passwordMinLength', 8),
                onSave: (v) => _save({'passwordMinLength': v}),
                saving: _saving,
                colors: colors,
              ),

              // Password history
              _NumberCard(
                label: 'Password history',
                description:
                    'Number of previous passwords to remember and disallow reuse. Set to 0 to disable.',
                value: n('passwordHistory', 0),
                onSave: (v) => _save({'passwordHistory': v}),
                saving: _saving,
                colors: colors,
              ),

              // Toggle rows
              _ToggleCard(
                label: 'Password dictionary check',
                description:
                    'Reject commonly used or compromised passwords.',
                value: b('passwordDictionary', false),
                onSave: (v) => _save({'passwordDictionary': v}),
                saving: _saving,
                colors: colors,
              ),
              _ToggleCard(
                label: 'Personal data check',
                description:
                    'Reject passwords that contain the user\'s name or email.',
                value: b('passwordPersonalData', false),
                onSave: (v) => _save({'passwordPersonalData': v}),
                saving: _saving,
                colors: colors,
              ),
              _ToggleCard(
                label: 'Require MFA',
                description:
                    'Require multi-factor authentication for all users.',
                value: b('mfaRequired', false),
                onSave: (v) => _save({'mfaRequired': v}),
                saving: _saving,
                colors: colors,
              ),
              _ToggleCard(
                label: 'Session alerts',
                description:
                    'Send an email to the user when a new session is created.',
                value: b('sessionAlerts', false),
                onSave: (v) => _save({'sessionAlerts': v}),
                saving: _saving,
                colors: colors,
              ),
              _ToggleCard(
                label: 'Invalidate sessions on password change',
                description:
                    'Immediately revoke all active sessions when a user changes their password.',
                value: b('invalidateOnPasswordChange', true),
                onSave: (v) => _save({'invalidateOnPasswordChange': v}),
                saving: _saving,
                colors: colors,
              ),
            ],
          ),
        );
      },
    );
  }
}

class _ToggleCard extends StatefulWidget {
  final String label;
  final String description;
  final bool value;
  final void Function(bool) onSave;
  final bool saving;
  final dynamic colors;

  const _ToggleCard({
    required this.label,
    required this.description,
    required this.value,
    required this.onSave,
    required this.saving,
    required this.colors,
  });

  @override
  State<_ToggleCard> createState() => _ToggleCardState();
}

class _ToggleCardState extends State<_ToggleCard> {
  late bool _val;

  @override
  void initState() {
    super.initState();
    _val = widget.value;
  }

  @override
  void didUpdateWidget(_ToggleCard old) {
    super.didUpdateWidget(old);
    if (old.value != widget.value) _val = widget.value;
  }

  @override
  Widget build(BuildContext context) {
    final c = widget.colors;
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: c.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: c.border),
      ),
      child: Row(children: [
        Expanded(
            child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
              Text(widget.label,
                  style: TextStyle(
                      color: c.textPrimary,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
              const SizedBox(height: 2),
              Text(widget.description,
                  style: TextStyle(color: c.textSubtle, fontSize: 12)),
            ])),
        Switch(
          value: _val,
          onChanged: (v) {
            setState(() => _val = v);
            widget.onSave(v);
          },
          activeThumbColor: _accent,
        ),
      ]),
    );
  }
}

class _NumberCard extends StatefulWidget {
  final String label;
  final String description;
  final int value;
  final void Function(int) onSave;
  final bool saving;
  final dynamic colors;

  const _NumberCard({
    required this.label,
    required this.description,
    required this.value,
    required this.onSave,
    required this.saving,
    required this.colors,
  });

  @override
  State<_NumberCard> createState() => _NumberCardState();
}

class _NumberCardState extends State<_NumberCard> {
  late TextEditingController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: '${widget.value}');
  }

  @override
  void didUpdateWidget(_NumberCard old) {
    super.didUpdateWidget(old);
    if (old.value != widget.value) {
      _ctrl.text = '${widget.value}';
    }
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final c = widget.colors;
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: c.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: c.border),
      ),
      child: Row(children: [
        Expanded(
            child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
              Text(widget.label,
                  style: TextStyle(
                      color: c.textPrimary,
                      fontSize: 14,
                      fontWeight: FontWeight.w500)),
              const SizedBox(height: 2),
              Text(widget.description,
                  style: TextStyle(color: c.textSubtle, fontSize: 12)),
            ])),
        const SizedBox(width: 16),
        SizedBox(
          width: 100,
          child: TextField(
            controller: _ctrl,
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            style: TextStyle(color: c.textPrimary, fontSize: 13),
            decoration: InputDecoration(
              isDense: true,
              contentPadding:
                  const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              filled: true,
              fillColor: c.fill,
              border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(6),
                  borderSide: BorderSide(color: c.border)),
              enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(6),
                  borderSide: BorderSide(color: c.border)),
            ),
          ),
        ),
        const SizedBox(width: 8),
        TextButton(
          onPressed: widget.saving
              ? null
              : () {
                  final v = int.tryParse(_ctrl.text) ?? widget.value;
                  widget.onSave(v);
                },
          style: TextButton.styleFrom(
            foregroundColor: _accent,
            padding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          ),
          child: const Text('Save'),
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
    final colors = consoleColors(context);
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Email & SMS templates',
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
          Text(
            'Customize the messages sent to your users during authentication flows.',
            style: TextStyle(color: colors.textSecondary, fontSize: 13),
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
    final colors = consoleColors(context);
    final isEmail = type == 'email';
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colors.border),
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: colors.fill,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Icon(
              isEmail ? LucideIcons.mail : LucideIcons.smartphone,
              size: 14,
              color: colors.textSecondary,
            ),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(name,
                    style: TextStyle(
                        color: colors.textPrimary,
                        fontSize: 14,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 2),
                Text(description,
                    style: TextStyle(color: colors.textSubtle, fontSize: 12)),
              ],
            ),
          ),
          OutlinedButton(
            style: OutlinedButton.styleFrom(
              foregroundColor: colors.textSecondary,
              side: BorderSide(color: colors.border),
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
    final colors = consoleColors(context);
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
            Text('Usage',
              style: TextStyle(
                color: colors.textPrimary,
                  fontSize: 16,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 4),
            Text('Authentication activity for the past 30 days.',
              style: TextStyle(color: colors.textSecondary, fontSize: 13)),
          const SizedBox(height: 24),
          Row(
            children: [
              _statCard(context, 'Total users', '—', LucideIcons.users),
              const SizedBox(width: 12),
              _statCard(context, 'Active sessions', '—', LucideIcons.activity),
              const SizedBox(width: 12),
              _statCard(context, 'New signups (30d)', '—', LucideIcons.userPlus),
            ],
          ),
          const SizedBox(height: 24),
          Container(
            height: 200,
            decoration: BoxDecoration(
              color: colors.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: colors.border),
            ),
            child: Center(
              child: Text('Usage charts coming soon',
                  style: TextStyle(color: colors.textSubtle, fontSize: 13)),
            ),
          ),
        ],
      ),
    );
  }

  Widget _statCard(
      BuildContext context, String label, String value, IconData icon) {
    final colors = consoleColors(context);
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(icon, size: 16, color: colors.textSecondary),
            const SizedBox(height: 12),
            Text(value,
                style: TextStyle(
                    color: colors.textPrimary,
                    fontSize: 24,
                    fontWeight: FontWeight.w700)),
            const SizedBox(height: 4),
            Text(label,
                style: TextStyle(color: colors.textSecondary, fontSize: 12)),
          ],
        ),
      ),
    );
  }
}

// ── Auth Settings tab (merged: auth methods + OAuth providers) ────────────────

// Auth method toggles
class _AuthMethod {
  final String id;
  final String label;
  final String description;
  final IconData icon;
  final bool defaultOn;

  const _AuthMethod({
    required this.id,
    required this.label,
    required this.description,
    required this.icon,
    this.defaultOn = false,
  });
}

const _authMethods = [
  _AuthMethod(
    id: 'email',
    label: 'Email / Password',
    description: 'Sign in with email and password',
    icon: LucideIcons.mail,
    defaultOn: true,
  ),
  _AuthMethod(
    id: 'phone',
    label: 'Phone',
    description: 'Sign in with phone number + OTP',
    icon: LucideIcons.phone,
  ),
  _AuthMethod(
    id: 'magic',
    label: 'Magic URL',
    description: 'Passwordless email link sign-in',
    icon: LucideIcons.link,
  ),
  _AuthMethod(
    id: 'emailOtp',
    label: 'Email OTP',
    description: 'Sign in with one-time code via email',
    icon: LucideIcons.keyRound,
  ),
  _AuthMethod(
    id: 'anonymous',
    label: 'Anonymous',
    description: 'Sessions without credentials',
    icon: LucideIcons.userX,
    defaultOn: true,
  ),
  _AuthMethod(
    id: 'teamInvites',
    label: 'Team Invites',
    description: 'Accept team invitations to sign up',
    icon: LucideIcons.userPlus,
    defaultOn: true,
  ),
  _AuthMethod(
    id: 'jwt',
    label: 'JWT',
    description: 'Accept externally issued JWT tokens',
    icon: LucideIcons.shield,
  ),
];

// OAuth provider definitions
// ── Provider field model ──────────────────────────────────────────────────────

enum _FieldType { text, secret, multiline }

class _ProviderField {
  final String key;
  final String label;
  final String hint;
  final _FieldType type;

  const _ProviderField({
    required this.key,
    required this.label,
    required this.hint,
    this.type = _FieldType.text,
  });
}

// ── OAuth provider model ──────────────────────────────────────────────────────

class _OAuthProvider {
  final String id;
  final String name;
  final Color color;
  final String letter;
  final List<_ProviderField> fields;
  final String? setupNote; // null = use generic note

  const _OAuthProvider({
    required this.id,
    required this.name,
    required this.color,
    required this.letter,
    required this.fields,
    this.setupNote,
  });
}

const _oauthProviders = [
  _OAuthProvider(
    id: 'google',
    name: 'Google',
    color: Color(0xFF4285F4),
    letter: 'G',
    fields: [
      _ProviderField(key: 'clientId', label: 'App ID', hint: 'Enter ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'App Secret',
          hint: 'Enter Secret',
          type: _FieldType.secret),
    ],
    setupNote:
        "To complete the setup, create an OAuth2 client ID with 'Web application' as the application type, then add this redirect URI to your Google configuration.",
  ),
  _OAuthProvider(
    id: 'github',
    name: 'GitHub',
    color: Color(0xFF24292E),
    letter: '',
    fields: [
      _ProviderField(key: 'clientId', label: 'App ID', hint: 'Enter ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'App Secret',
          hint: 'Enter App Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'apple',
    name: 'Apple',
    color: Color(0xFF555555),
    letter: '',
    fields: [
      _ProviderField(
          key: 'serviceId',
          label: 'Services ID',
          hint: 'com.company.appname'),
      _ProviderField(key: 'keyId', label: 'Key ID', hint: 'SHAB13ROFN'),
      _ProviderField(key: 'teamId', label: 'Team ID', hint: 'ELA2CD3AED'),
      _ProviderField(
          key: 'p8',
          label: 'P8 File',
          hint: '-----BEGIN PRIVATE KEY-----\n...',
          type: _FieldType.multiline),
    ],
  ),
  _OAuthProvider(
    id: 'facebook',
    name: 'Facebook',
    color: Color(0xFF1877F2),
    letter: 'f',
    fields: [
      _ProviderField(key: 'clientId', label: 'App ID', hint: 'Enter ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'App Secret',
          hint: 'Enter App Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'discord',
    name: 'Discord',
    color: Color(0xFF5865F2),
    letter: 'D',
    fields: [
      _ProviderField(
          key: 'clientId', label: 'Client ID', hint: 'Enter Client ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'Client Secret',
          hint: 'Enter Client Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'twitter',
    name: 'Twitter / X',
    color: Color(0xFF000000),
    letter: 'X',
    fields: [
      _ProviderField(
          key: 'clientId', label: 'Consumer Key', hint: 'Enter Consumer Key'),
      _ProviderField(
          key: 'clientSecret',
          label: 'Consumer Secret',
          hint: 'Enter Consumer Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'microsoft',
    name: 'Microsoft',
    color: Color(0xFF00A1F1),
    letter: 'M',
    fields: [
      _ProviderField(
          key: 'clientId', label: 'App (client) ID', hint: 'Enter Client ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'Client Secret Value',
          hint: 'Enter Client Secret',
          type: _FieldType.secret),
      _ProviderField(
          key: 'tenantId',
          label: 'Tenant ID (optional)',
          hint: 'common'),
    ],
  ),
  _OAuthProvider(
    id: 'slack',
    name: 'Slack',
    color: Color(0xFF4A154B),
    letter: 'S',
    fields: [
      _ProviderField(key: 'clientId', label: 'Client ID', hint: 'Enter ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'Client Secret',
          hint: 'Enter Client Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'spotify',
    name: 'Spotify',
    color: Color(0xFF1DB954),
    letter: '',
    fields: [
      _ProviderField(key: 'clientId', label: 'Client ID', hint: 'Enter ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'Client Secret',
          hint: 'Enter Client Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'linkedin',
    name: 'LinkedIn',
    color: Color(0xFF0A66C2),
    letter: 'in',
    fields: [
      _ProviderField(key: 'clientId', label: 'Client ID', hint: 'Enter ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'Client Secret',
          hint: 'Enter Client Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'gitlab',
    name: 'GitLab',
    color: Color(0xFFFC6D26),
    letter: '',
    fields: [
      _ProviderField(key: 'clientId', label: 'App ID', hint: 'Enter ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'App Secret',
          hint: 'Enter App Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'bitbucket',
    name: 'Bitbucket',
    color: Color(0xFF0052CC),
    letter: 'B',
    fields: [
      _ProviderField(
          key: 'clientId', label: 'Key (Client ID)', hint: 'Enter Key'),
      _ProviderField(
          key: 'clientSecret',
          label: 'Secret (Client Secret)',
          hint: 'Enter Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'twitch',
    name: 'Twitch',
    color: Color(0xFF9146FF),
    letter: 'T',
    fields: [
      _ProviderField(key: 'clientId', label: 'Client ID', hint: 'Enter ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'Client Secret',
          hint: 'Enter Client Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'notion',
    name: 'Notion',
    color: Color(0xFF191919),
    letter: 'N',
    fields: [
      _ProviderField(
          key: 'clientId', label: 'OAuth Client ID', hint: 'Enter ID'),
      _ProviderField(
          key: 'clientSecret',
          label: 'OAuth Client Secret',
          hint: 'Enter Secret',
          type: _FieldType.secret),
    ],
  ),
  _OAuthProvider(
    id: 'stripe',
    name: 'Stripe',
    color: Color(0xFF635BFF),
    letter: 'S',
    fields: [
      _ProviderField(
          key: 'clientId', label: 'Client ID', hint: 'ca_xxxxxxxxxxxx'),
      _ProviderField(
          key: 'clientSecret',
          label: 'Secret Key',
          hint: 'sk_live_xxxxxxxxxxxx',
          type: _FieldType.secret),
    ],
  ),
];

class _AuthSettingsTab extends StatefulWidget {
  const _AuthSettingsTab();

  @override
  State<_AuthSettingsTab> createState() => _AuthSettingsTabState();
}

class _AuthSettingsTabState extends State<_AuthSettingsTab> {
  // Local toggle state (method id → enabled)
  late final Map<String, bool> _methodState;
  // OAuth enabled state (provider id → enabled)
  late final Map<String, bool> _oauthState;

  @override
  void initState() {
    super.initState();
    _methodState = {for (final m in _authMethods) m.id: m.defaultOn};
    _oauthState = {for (final p in _oauthProviders) p.id: false};
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return SingleChildScrollView(
      padding: const EdgeInsets.only(bottom: 40),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // ── Auth methods ───────────────────────────────────────────────
          _SectionHeader(
            title: 'Auth methods',
            subtitle: 'Enable the authentication methods you wish to use.',
          ),
          const SizedBox(height: 16),
          Container(
            decoration: BoxDecoration(
              color: cs.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: cs.border),
            ),
            child: LayoutBuilder(
              builder: (_, constraints) {
                final cols = constraints.maxWidth > 680 ? 2 : 1;
                final items = _authMethods;
                if (cols == 1) {
                  return Column(
                    children: items.asMap().entries.map((e) {
                      final isLast = e.key == items.length - 1;
                      return _AuthMethodRow(
                        method: e.value,
                        enabled: _methodState[e.value.id] ?? false,
                        showDivider: !isLast,
                        onChanged: (v) =>
                            setState(() => _methodState[e.value.id] = v),
                      );
                    }).toList(),
                  );
                }
                // 2-column grid
                final left = items.where((m) => items.indexOf(m) % 2 == 0).toList();
                final right = items.where((m) => items.indexOf(m) % 2 == 1).toList();
                return IntrinsicHeight(
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: Column(
                          children: left.map((m) => _AuthMethodRow(
                                method: m,
                                enabled: _methodState[m.id] ?? false,
                                showDivider: left.indexOf(m) < left.length - 1 ||
                                    right.isNotEmpty,
                                onChanged: (v) =>
                                    setState(() => _methodState[m.id] = v),
                              )).toList(),
                        ),
                      ),
                      VerticalDivider(
                          width: 1, thickness: 1, color: cs.border),
                      Expanded(
                        child: Column(
                          children: right.map((m) => _AuthMethodRow(
                                method: m,
                                enabled: _methodState[m.id] ?? false,
                                showDivider:
                                    right.indexOf(m) < right.length - 1,
                                onChanged: (v) =>
                                    setState(() => _methodState[m.id] = v),
                              )).toList(),
                        ),
                      ),
                    ],
                  ),
                );
              },
            ),
          ),

          const SizedBox(height: 32),

          // ── OAuth providers ────────────────────────────────────────────
          _SectionHeader(
            title: 'OAuth2 Providers',
            subtitle:
                'Allow users to sign in with their existing third-party accounts.',
          ),
          const SizedBox(height: 16),
          LayoutBuilder(
            builder: (_, constraints) {
              final width = constraints.maxWidth;
              final cols = width > 1100
                  ? 5
                  : width > 800
                      ? 4
                      : width > 580
                          ? 3
                          : 2;
              final cardWidth = (width - (cols - 1) * 10) / cols;
              return Wrap(
                spacing: 10,
                runSpacing: 10,
                children: _oauthProviders.map((p) {
                  return SizedBox(
                    width: cardWidth,
                    child: _OAuthProviderCard(
                      provider: p,
                      enabled: _oauthState[p.id] ?? false,
                      onToggle: (v) =>
                          setState(() => _oauthState[p.id] = v),
                      onConfigure: () =>
                          _showConfigureDialog(context, p),
                    ),
                  );
                }).toList(),
              );
            },
          ),
        ],
      ),
    );
  }

  void _showConfigureDialog(BuildContext context, _OAuthProvider provider) {
    final projectId =
        GoRouterState.of(context).pathParameters['projectId'] ?? '';
    final currentEnabled = _oauthState[provider.id] ?? false;

    showDialog(
      context: context,
      builder: (_) => _OAuthConfigDialog(
        provider: provider,
        projectId: projectId,
        initialEnabled: currentEnabled,
        onSave: (enabled) {
          setState(() => _oauthState[provider.id] = enabled);
        },
      ),
    );
  }
}

// ── Section header ────────────────────────────────────────────────────────────

class _SectionHeader extends StatelessWidget {
  final String title;
  final String subtitle;

  const _SectionHeader({required this.title, required this.subtitle});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(title,
            style: TextStyle(
                color: cs.textPrimary,
                fontSize: 15,
                fontWeight: FontWeight.w600)),
        const SizedBox(height: 3),
        Text(subtitle,
            style: TextStyle(color: cs.textMuted, fontSize: 13)),
      ],
    );
  }
}

// ── Auth method row ───────────────────────────────────────────────────────────

class _AuthMethodRow extends StatelessWidget {
  final _AuthMethod method;
  final bool enabled;
  final bool showDivider;
  final ValueChanged<bool> onChanged;

  const _AuthMethodRow({
    required this.method,
    required this.enabled,
    required this.showDivider,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
      children: [
        Padding(
          padding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 13),
          child: Row(
            children: [
              Container(
                width: 30,
                height: 30,
                decoration: BoxDecoration(
                  color: enabled
                      ? _accent.withValues(alpha: 0.12)
                      : cs.fill,
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Icon(
                  method.icon,
                  size: 14,
                  color: enabled ? _accent : cs.textSubtle,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(method.label,
                        style: TextStyle(
                            color: cs.textPrimary,
                            fontSize: 13,
                            fontWeight: FontWeight.w500)),
                    Text(method.description,
                        style: TextStyle(
                            color: cs.textSubtle, fontSize: 11)),
                  ],
                ),
              ),
              Switch(
                value: enabled,
                onChanged: onChanged,
                activeThumbColor: Colors.white,
                activeTrackColor: _accent,
                inactiveThumbColor: cs.textSubtle,
                inactiveTrackColor: cs.fill,
                trackOutlineColor: WidgetStatePropertyAll(cs.border),
                materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
            ],
          ),
        ),
        if (showDivider) Divider(height: 1, color: cs.border),
      ],
    );
  }
}

// ── OAuth provider card ───────────────────────────────────────────────────────

class _OAuthProviderCard extends StatefulWidget {
  final _OAuthProvider provider;
  final bool enabled;
  final ValueChanged<bool> onToggle;
  final VoidCallback onConfigure;

  const _OAuthProviderCard({
    required this.provider,
    required this.enabled,
    required this.onToggle,
    required this.onConfigure,
  });

  @override
  State<_OAuthProviderCard> createState() => _OAuthProviderCardState();
}

class _OAuthProviderCardState extends State<_OAuthProviderCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    const green = Color(0xFF10B981);

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onConfigure,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 130),
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: _hovered ? cs.fillHover : cs.surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
                color: widget.enabled
                    ? green.withValues(alpha: 0.35)
                    : cs.border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  _ProviderBadge(provider: widget.provider, size: 28),
                  const Spacer(),
                  // Toggle pill
                  GestureDetector(
                    onTap: () => widget.onToggle(!widget.enabled),
                    behavior: HitTestBehavior.opaque,
                    child: AnimatedContainer(
                      duration: const Duration(milliseconds: 150),
                      width: 32,
                      height: 18,
                      decoration: BoxDecoration(
                        color: widget.enabled
                            ? _accent
                            : cs.fill,
                        borderRadius: BorderRadius.circular(9),
                        border: Border.all(color: cs.border),
                      ),
                      child: AnimatedAlign(
                        duration: const Duration(milliseconds: 150),
                        alignment: widget.enabled
                            ? Alignment.centerRight
                            : Alignment.centerLeft,
                        child: Container(
                          width: 12,
                          height: 12,
                          margin: const EdgeInsets.symmetric(horizontal: 3),
                          decoration: const BoxDecoration(
                            color: Colors.white,
                            shape: BoxShape.circle,
                          ),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 10),
              Text(
                widget.provider.name,
                style: TextStyle(
                  color: cs.textPrimary,
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                ),
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 4),
              Text(
                widget.enabled ? 'enabled' : 'disabled',
                style: TextStyle(
                  color: widget.enabled ? green : cs.textSubtle,
                  fontSize: 11,
                  fontWeight:
                      widget.enabled ? FontWeight.w500 : FontWeight.w400,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Provider badge (colored letter avatar) ────────────────────────────────────

class _ProviderBadge extends StatelessWidget {
  final _OAuthProvider provider;
  final double size;

  const _ProviderBadge({required this.provider, required this.size});

  static IconData? _iconFor(String id) {
    switch (id) {
      case 'github':
        return LucideIcons.github;
      case 'apple':
        return LucideIcons.apple;
      case 'spotify':
        return LucideIcons.music;
      case 'gitlab':
        return LucideIcons.gitBranch;
      default:
        return null;
    }
  }

  @override
  Widget build(BuildContext context) {
    final icon = _iconFor(provider.id);
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: provider.color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(size * 0.25),
      ),
      child: icon != null
          ? Icon(icon, size: size * 0.5, color: provider.color)
          : Center(
              child: Text(
                provider.letter.isNotEmpty
                    ? provider.letter
                    : provider.name[0],
                style: TextStyle(
                  color: provider.color,
                  fontSize: size * 0.4,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
    );
  }
}

// ── OAuth config dialog ───────────────────────────────────────────────────────

class _OAuthConfigDialog extends StatefulWidget {
  final _OAuthProvider provider;
  final String projectId;
  final bool initialEnabled;
  final ValueChanged<bool> onSave;

  const _OAuthConfigDialog({
    required this.provider,
    required this.projectId,
    required this.initialEnabled,
    required this.onSave,
  });

  @override
  State<_OAuthConfigDialog> createState() => _OAuthConfigDialogState();
}

class _OAuthConfigDialogState extends State<_OAuthConfigDialog> {
  late bool _enabled;
  late final Map<String, TextEditingController> _ctrls;
  final Map<String, bool> _obscured = {};

  @override
  void initState() {
    super.initState();
    _enabled = widget.initialEnabled;
    _ctrls = {
      for (final f in widget.provider.fields)
        f.key: TextEditingController()
    };
    for (final f in widget.provider.fields) {
      if (f.type == _FieldType.secret) _obscured[f.key] = true;
    }
  }

  @override
  void dispose() {
    for (final c in _ctrls.values) c.dispose();
    super.dispose();
  }

  String get _redirectUri =>
      'https://your-domain.com/v1/account/sessions/oauth2/callback/${widget.provider.id}/${widget.projectId}';

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    final provider = widget.provider;
    final note = provider.setupNote ??
        'To complete set up, add this OAuth2 redirect URI to your ${provider.name} app configuration.';

    return Dialog(
      backgroundColor: cs.surface,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
      child: SizedBox(
        width: 540,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ── Header ────────────────────────────────────────────────
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 24, 24, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: Text(
                          '${provider.name} OAuth2 settings',
                          style: TextStyle(
                            color: cs.textPrimary,
                            fontSize: 18,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      const SizedBox(width: 12),
                      GestureDetector(
                        onTap: () => Navigator.of(context).pop(),
                        child: MouseRegion(
                          cursor: SystemMouseCursors.click,
                          child: Container(
                            width: 28,
                            height: 28,
                            decoration: BoxDecoration(
                              color: cs.fill,
                              borderRadius: BorderRadius.circular(6),
                              border: Border.all(color: cs.border),
                            ),
                            child: Icon(LucideIcons.x,
                                size: 13, color: cs.textMuted),
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 6),
                  RichText(
                    text: TextSpan(
                      style: TextStyle(
                          color: cs.textMuted, fontSize: 13, height: 1.5),
                      children: const [
                        TextSpan(
                            text:
                                'To use this authentication provider in your application, first fill in this form. For more info you can '),
                        TextSpan(
                          text: 'visit the docs.',
                          style: TextStyle(
                            color: _accent,
                            decoration: TextDecoration.underline,
                            decorationColor: _accent,
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            Divider(height: 1, color: cs.border),

            // ── Scrollable body ───────────────────────────────────────
            Flexible(
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(24, 20, 24, 0),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Enabled toggle
                    GestureDetector(
                      onTap: () => setState(() => _enabled = !_enabled),
                      child: MouseRegion(
                        cursor: SystemMouseCursors.click,
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            _MiniToggle(value: _enabled),
                            const SizedBox(width: 10),
                            Text(
                              _enabled ? 'Enabled' : 'Disabled',
                              style: TextStyle(
                                color: _enabled
                                    ? cs.textPrimary
                                    : cs.textMuted,
                                fontSize: 14,
                                fontWeight: FontWeight.w500,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 20),

                    // Provider fields
                    ...provider.fields.map((f) {
                      final ctrl = _ctrls[f.key]!;
                      final isObscured = _obscured[f.key] ?? false;
                      return Padding(
                        padding: const EdgeInsets.only(bottom: 16),
                        child: _DialogField(
                          ctrl: ctrl,
                          label: f.label,
                          hint: f.hint,
                          type: f.type,
                          obscured: isObscured,
                          onToggleObscure: f.type == _FieldType.secret
                              ? () => setState(
                                  () => _obscured[f.key] = !isObscured)
                              : null,
                        ),
                      );
                    }),

                    // Info box
                    Container(
                      padding: const EdgeInsets.all(14),
                      decoration: BoxDecoration(
                        color: _accent.withValues(alpha: 0.07),
                        borderRadius: BorderRadius.circular(8),
                        border:
                            Border.all(color: _accent.withValues(alpha: 0.2)),
                      ),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Icon(LucideIcons.info,
                              size: 14,
                              color: _accent),
                          const SizedBox(width: 10),
                          Expanded(
                            child: Text(
                              note,
                              style: TextStyle(
                                  color: cs.textSecondary,
                                  fontSize: 12,
                                  height: 1.5),
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 16),

                    // Redirect URI
                    _DialogField(
                      ctrl: TextEditingController(text: _redirectUri),
                      label: 'URI',
                      hint: '',
                      type: _FieldType.text,
                      readOnly: true,
                      trailing: _CopyIconButton(text: _redirectUri),
                    ),
                    const SizedBox(height: 20),
                  ],
                ),
              ),
            ),

            // ── Footer ────────────────────────────────────────────────
            Divider(height: 1, color: cs.border),
            Padding(
              padding:
                  const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    onPressed: () => Navigator.of(context).pop(),
                    style: TextButton.styleFrom(
                        foregroundColor: cs.textMuted,
                        padding: const EdgeInsets.symmetric(
                            horizontal: 16, vertical: 10)),
                    child: const Text('Cancel', style: TextStyle(fontSize: 13)),
                  ),
                  const SizedBox(width: 8),
                  FilledButton(
                    style: FilledButton.styleFrom(
                      backgroundColor: _accent,
                      shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(8)),
                      padding: const EdgeInsets.symmetric(
                          horizontal: 20, vertical: 10),
                    ),
                    onPressed: () {
                      widget.onSave(_enabled);
                      Navigator.of(context).pop();
                    },
                    child:
                        const Text('Update', style: TextStyle(fontSize: 13)),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// Small toggle pill used inside the dialog
class _MiniToggle extends StatelessWidget {
  final bool value;
  const _MiniToggle({required this.value});

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return AnimatedContainer(
      duration: const Duration(milliseconds: 150),
      width: 38,
      height: 22,
      decoration: BoxDecoration(
        color: value ? _accent : cs.fill,
        borderRadius: BorderRadius.circular(11),
        border: Border.all(color: value ? _accent : cs.border),
      ),
      child: AnimatedAlign(
        duration: const Duration(milliseconds: 150),
        alignment: value ? Alignment.centerRight : Alignment.centerLeft,
        child: Container(
          width: 16,
          height: 16,
          margin: const EdgeInsets.symmetric(horizontal: 3),
          decoration: const BoxDecoration(
              color: Colors.white, shape: BoxShape.circle),
        ),
      ),
    );
  }
}

// ── Dialog field ──────────────────────────────────────────────────────────────

class _DialogField extends StatelessWidget {
  final TextEditingController ctrl;
  final String label;
  final String hint;
  final _FieldType type;
  final bool obscured;
  final VoidCallback? onToggleObscure;
  final bool readOnly;
  final Widget? trailing;

  const _DialogField({
    required this.ctrl,
    required this.label,
    required this.hint,
    required this.type,
    this.obscured = false,
    this.onToggleObscure,
    this.readOnly = false,
    this.trailing,
  });

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label,
            style: TextStyle(
                color: cs.textPrimary,
                fontSize: 13,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        TextField(
          controller: ctrl,
          obscureText: obscured,
          readOnly: readOnly,
          maxLines: type == _FieldType.multiline ? 5 : 1,
          minLines: type == _FieldType.multiline ? 5 : null,
          style: TextStyle(
              color: readOnly ? cs.textMuted : cs.textPrimary, fontSize: 13),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: TextStyle(color: cs.textSubtle, fontSize: 13),
            filled: true,
            fillColor: readOnly ? cs.surfaceAlt : cs.fieldFill,
            isDense: true,
            contentPadding: const EdgeInsets.symmetric(
                horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: BorderSide(color: cs.fieldBorder)),
            enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: BorderSide(color: cs.fieldBorder)),
            focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: _accent)),
            suffixIcon: onToggleObscure != null
                ? GestureDetector(
                    onTap: onToggleObscure,
                    child: MouseRegion(
                      cursor: SystemMouseCursors.click,
                      child: Icon(
                        obscured ? LucideIcons.eye : LucideIcons.eyeOff,
                        size: 15,
                        color: cs.textSubtle,
                      ),
                    ),
                  )
                : trailing != null
                    ? Padding(
                        padding: const EdgeInsets.only(right: 4),
                        child: trailing)
                    : null,
          ),
        ),
      ],
    );
  }
}

// Small copy icon button used in read-only URI field
class _CopyIconButton extends StatefulWidget {
  final String text;
  const _CopyIconButton({required this.text});

  @override
  State<_CopyIconButton> createState() => _CopyIconButtonState();
}

class _CopyIconButtonState extends State<_CopyIconButton> {
  bool _copied = false;

  void _copy() async {
    await Clipboard.setData(ClipboardData(text: widget.text));
    setState(() => _copied = true);
    await Future.delayed(const Duration(seconds: 2));
    if (mounted) setState(() => _copied = false);
  }

  @override
  Widget build(BuildContext context) {
    final cs = consoleColors(context);
    return GestureDetector(
      onTap: _copy,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: AnimatedSwitcher(
          duration: const Duration(milliseconds: 180),
          child: _copied
              ? Icon(LucideIcons.check,
                  key: const ValueKey('ck'),
                  size: 14,
                  color: const Color(0xFF10B981))
              : Icon(LucideIcons.copy,
                  key: const ValueKey('cp'),
                  size: 14,
                  color: cs.textSubtle),
        ),
      ),
    );
  }
}


Widget _emptyState(
    BuildContext context,
    {required IconData icon,
    required String title,
    required String description}) {
  final colors = consoleColors(context);
  return Center(
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 48,
          height: 48,
          decoration: BoxDecoration(
            color: colors.fill,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Icon(icon, size: 22, color: colors.textSubtle),
        ),
        const SizedBox(height: 16),
        Text(title,
            style: TextStyle(
                color: colors.textPrimary,
                fontSize: 15,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        Text(description,
            style: TextStyle(color: colors.textSecondary, fontSize: 13)),
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
