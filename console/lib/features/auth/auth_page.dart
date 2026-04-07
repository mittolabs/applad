import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/search_list.dart';

// --- Providers -----------------------------------------------------------

final _authTabProvider = StateProvider<int>((ref) => 0);
final _userSearchProvider = StateProvider<String>((ref) => '');
final _userPerPageProvider = StateProvider<int>((ref) => 12);
final _userPageProvider = StateProvider<int>((ref) => 1);

final usersProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);
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
  final res = await api.get('/teams');
  return res.data as Map<String, dynamic>;
});

// --- Page ----------------------------------------------------------------

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
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Title row
          Padding(
            padding: const EdgeInsets.fromLTRB(24, 20, 24, 0),
            child: Text('Auth',
                style: Theme.of(context)
                    .textTheme
                    .headlineSmall
                    ?.copyWith(color: Colors.white)),
          ),
          // Tabs
          PageTabs(
            tabs: const ['Users', 'Teams', 'Settings'],
            selected: tab,
            onChanged: (i) => ref.read(_authTabProvider.notifier).state = i,
          ),
          const Divider(height: 1, color: Color(0xFF2A2B30)),
          // Tab body
          Expanded(child: _tabBody(tab)),
        ],
      ),
    );
  }

  Widget _tabBody(int tab) {
    switch (tab) {
      case 0:
        return _UsersTab(
          searchCtrl: _searchCtrl,
          onSearch: _doSearch,
        );
      case 1:
        return const _TeamsTab();
      case 2:
        return const _SettingsTab();
      default:
        return const SizedBox.shrink();
    }
  }
}

// --- Users tab -----------------------------------------------------------

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
      error: (e, _) => Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, size: 48),
            const SizedBox(height: 16),
            Text('Failed to load users: $e'),
            const SizedBox(height: 8),
            FilledButton(
              onPressed: () => ref.invalidate(usersProvider),
              child: const Text('Retry'),
            ),
          ],
        ),
      ),
      data: (data) {
        final users = List<Map<String, dynamic>>.from(data['users'] ?? []);
        final total = data['total'] as int? ?? 0;

        return Column(
          children: [
            SearchListHeader(
              searchController: searchCtrl,
              total: total,
              perPage: perPage,
              currentPage: currentPage,
              onPerPageChanged: (v) {
                ref.read(_userPerPageProvider.notifier).state = v;
                ref.read(_userPageProvider.notifier).state = 1;
              },
              onPrev: () => ref.read(_userPageProvider.notifier).update((s) => s - 1),
              onNext: () => ref.read(_userPageProvider.notifier).update((s) => s + 1),
              onSearch: onSearch,
              trailing: FilledButton.icon(
                onPressed: () => _showCreateUserDialog(context, ref),
                icon: const Icon(Icons.add, size: 16),
                label: const Text('Create User'),
              ),
            ),
            Expanded(
              child: users.isEmpty
                  ? const Center(child: Text('No users yet'))
                  : ListView.builder(
                      itemCount: users.length,
                      itemBuilder: (context, i) {
                        final u = users[i];
                        return ListTile(
                          leading: CircleAvatar(
                            child: Text((u['name'] ?? u['email'] ?? '?')
                                .toString()
                                .substring(0, 1)
                                .toUpperCase()),
                          ),
                          title: Text(u['name'] ?? 'Anonymous'),
                          subtitle: Text(u['email'] ?? 'No email'),
                          trailing: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Chip(
                                label: Text(
                                    u['status'] == true ? 'Active' : 'Disabled'),
                                backgroundColor: u['status'] == true
                                    ? Colors.green.shade100
                                    : Colors.red.shade100,
                              ),
                              const SizedBox(width: 8),
                              IconButton(
                                icon: const Icon(Icons.delete_outline),
                                onPressed: () =>
                                    _deleteUser(context, ref, u['\$id']),
                              ),
                            ],
                          ),
                        );
                      },
                    ),
            ),
            SearchListFooter(
              total: total,
              perPage: perPage,
              currentPage: currentPage,
              onPrev: () => ref.read(_userPageProvider.notifier).update((s) => s - 1),
              onNext: () => ref.read(_userPageProvider.notifier).update((s) => s + 1),
              onPerPageChanged: (v) {
                ref.read(_userPerPageProvider.notifier).state = v;
                ref.read(_userPageProvider.notifier).state = 1;
              },
            ),
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
      title: 'Create User',
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          AppDialogField(
            controller: nameCtrl,
            label: 'Name',
            hint: 'Full name',
            autofocus: true,
          ),
          const SizedBox(height: 12),
          AppDialogField(
            controller: emailCtrl,
            label: 'Email',
            hint: 'user@example.com',
          ),
          const SizedBox(height: 12),
          AppDialogField(
            controller: passCtrl,
            label: 'Password',
            hint: 'Password',
          ),
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
            if (context.mounted) Navigator.pop(context);
            ref.invalidate(usersProvider);
          },
        ),
      ],
    );
  }

  Future<void> _deleteUser(
      BuildContext context, WidgetRef ref, String userId) async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete User',
      content: Text(
        'Are you sure you want to delete this user?',
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
      final api = ref.read(apiClientProvider);
      await api.delete('/users/$userId');
      ref.invalidate(usersProvider);
    }
  }
}

// --- Teams tab -----------------------------------------------------------

class _TeamsTab extends ConsumerWidget {
  const _TeamsTab();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final teamsAsync = ref.watch(teamsProvider);

    return teamsAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, size: 48),
            const SizedBox(height: 16),
            Text('Failed to load teams: $e'),
            const SizedBox(height: 8),
            FilledButton(
              onPressed: () => ref.invalidate(teamsProvider),
              child: const Text('Retry'),
            ),
          ],
        ),
      ),
      data: (data) {
        final teams = List<Map<String, dynamic>>.from(data['teams'] ?? []);
        final total = data['total'] as int? ?? 0;

        if (teams.isEmpty) {
          return const Center(
            child: Text('No teams yet', style: TextStyle(color: Color(0x40FFFFFF))),
          );
        }

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text('$total teams',
                  style: Theme.of(context).textTheme.titleMedium),
            ),
            Expanded(
              child: ListView.builder(
                itemCount: teams.length,
                itemBuilder: (context, i) {
                  final t = teams[i];
                  return ListTile(
                    leading: const Icon(Icons.group),
                    title: Text(t['name'] ?? 'Unnamed'),
                    subtitle: Text(t['\$id'] ?? ''),
                    trailing: Text('${t['total'] ?? 0} members',
                        style: const TextStyle(
                            fontSize: 12, color: Color(0x40FFFFFF))),
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

// --- Settings tab --------------------------------------------------------

class _SettingsTab extends ConsumerWidget {
  const _SettingsTab();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Authentication Settings',
              style: TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.w600)),
          const SizedBox(height: 24),
          _settingRow('Session duration', '365 days', 'How long sessions remain valid'),
          _settingRow('Max sessions per user', '10', 'Maximum concurrent sessions'),
          _settingRow('Password minimum length', '8 characters', 'Minimum password requirement'),
          _settingRow('Password history', 'Disabled', 'Prevent reuse of previous passwords'),
          _settingRow('MFA enforcement', 'Optional', 'Require multi-factor authentication'),
          _settingRow('Email verification', 'Disabled', 'Require email verification on signup'),
          _settingRow('Anonymous sessions', 'Enabled', 'Allow anonymous session creation'),
        ],
      ),
    );
  }

  Widget _settingRow(String label, String value, String description) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 20),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: const Color(0xFF16171B),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: const Color(0x14FFFFFF)),
        ),
        child: Row(children: [
          Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(label, style: const TextStyle(color: Colors.white, fontSize: 14)),
            const SizedBox(height: 2),
            Text(description, style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 12)),
          ])),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.06),
              borderRadius: BorderRadius.circular(6),
            ),
            child: Text(value, style: TextStyle(color: Colors.white.withOpacity(0.7), fontSize: 13)),
          ),
        ]),
      ),
    );
  }
}
