import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/auth_provider.dart';
import '../../core/providers/project_provider.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/app_navbar.dart';
import '../../core/widgets/page_tabs.dart';

const _bg = Color(0xFF0B0B0F);
const _cardColor = Color(0xFF16171B);
const _border = Color(0x0FFFFFFF);
const _accent = Color(0xFF3472A4);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _red = Color(0xFFEF4444);

class AccountPage extends ConsumerStatefulWidget {
  const AccountPage({super.key});

  @override
  ConsumerState<AccountPage> createState() => _AccountPageState();
}

class _AccountPageState extends ConsumerState<AccountPage> {
  final _nameCtrl = TextEditingController();
  final _emailCtrl = TextEditingController();
  final _oldPasswordCtrl = TextEditingController();
  final _newPasswordCtrl = TextEditingController();
  bool _initialized = false;
  bool _savingName = false;
  bool _savingEmail = false;
  bool _savingPassword = false;
  int _tabIndex = 0;

  @override
  void dispose() {
    _nameCtrl.dispose();
    _emailCtrl.dispose();
    _oldPasswordCtrl.dispose();
    _newPasswordCtrl.dispose();
    super.dispose();
  }

  void _initFields(ConsoleUser user) {
    if (!_initialized) {
      _nameCtrl.text = user.name;
      _emailCtrl.text = user.email;
      _initialized = true;
    }
  }

  Future<void> _updateName() async {
    setState(() => _savingName = true);
    try {
      await ref.read(apiClientProvider).patch('/console/me',
          data: {'name': _nameCtrl.text.trim()});
      ref.invalidate(consoleAuthProvider);
      if (mounted) _snack('Name updated');
    } catch (e) {
      if (mounted) _snack('Failed: $e', error: true);
    } finally {
      if (mounted) setState(() => _savingName = false);
    }
  }

  Future<void> _updateEmail() async {
    setState(() => _savingEmail = true);
    try {
      await ref.read(apiClientProvider).patch('/console/me',
          data: {'email': _emailCtrl.text.trim()});
      ref.invalidate(consoleAuthProvider);
      if (mounted) _snack('Email updated');
    } catch (e) {
      if (mounted) _snack('Failed: $e', error: true);
    } finally {
      if (mounted) setState(() => _savingEmail = false);
    }
  }

  Future<void> _updatePassword() async {
    if (_newPasswordCtrl.text.isEmpty || _oldPasswordCtrl.text.isEmpty) return;
    setState(() => _savingPassword = true);
    try {
      await ref.read(apiClientProvider).patch('/console/me/password', data: {
        'oldPassword': _oldPasswordCtrl.text,
        'password': _newPasswordCtrl.text,
      });
      _oldPasswordCtrl.clear();
      _newPasswordCtrl.clear();
      if (mounted) _snack('Password updated');
    } catch (e) {
      if (mounted) _snack('Failed: $e', error: true);
    } finally {
      if (mounted) setState(() => _savingPassword = false);
    }
  }

  Future<void> _deleteAccount() async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete account',
      content: Text(
        'Your account will be permanently deleted and access will be lost to all your teams and data. This action is irreversible.',
        style: TextStyle(color: Colors.white.withOpacity(0.6)),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Delete',
          destructive: true,
          onTap: () =>
              Navigator.of(context, rootNavigator: true).pop(true),
        ),
      ],
    );
    if (confirmed == true) {
      try {
        await ref.read(apiClientProvider).delete('/console/me');
        ref.read(consoleAuthProvider.notifier).logout();
        if (mounted) context.go('/login');
      } catch (e) {
        if (mounted) _snack('Failed: $e', error: true);
      }
    }
  }

  void _snack(String msg, {bool error = false}) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: error ? Colors.red.shade800 : null,
    ));
  }

  @override
  Widget build(BuildContext context) {
    final authAsync = ref.watch(consoleAuthProvider);
    final user = authAsync.valueOrNull;
    if (user != null) _initFields(user);

    return Scaffold(
      backgroundColor: _bg,
      body: authAsync.when(
        loading: () => const Center(
            child: CircularProgressIndicator(color: Colors.white24)),
        error: (e, _) => Center(
            child: Text('Error: $e',
                style: const TextStyle(color: Colors.white70))),
        data: (user) {
          if (user == null) {
            return const Center(
                child: Text('Not logged in',
                    style: TextStyle(color: Colors.white70)));
          }
          return Column(
            children: [
              const AppNavBar(),
              Expanded(
                child: Padding(
            padding: EdgeInsets.symmetric(
              horizontal:
                  MediaQuery.of(context).size.width > 1400 ? 80 : 40,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const SizedBox(height: 32),
                // Title + Logout
                Row(
                  children: [
                    Text(
                      user.name.isNotEmpty ? user.name : 'Account',
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 22,
                          fontWeight: FontWeight.w600),
                    ),
                    const Spacer(),
                    TextButton(
                      onPressed: () {
                        ref.read(consoleAuthProvider.notifier).logout();
                        context.go('/login');
                      },
                      style: TextButton.styleFrom(
                          foregroundColor: Colors.white54),
                      child: const Text('Logout',
                          style: TextStyle(fontSize: 13)),
                    ),
                  ],
                ),
                const SizedBox(height: 24),
                PageTabs(
                  tabs: const [
                    'Overview',
                    'Sessions',
                    'Activity',
                    'Organizations',
                  ],
                  selected: _tabIndex,
                  onChanged: (i) => setState(() => _tabIndex = i),
                ),
                const SizedBox(height: 24),
                Expanded(
                  child: _tabIndex == 0
                      ? _buildOverviewTab(user)
                      : _tabIndex == 1
                          ? _buildSessionsTab()
                          : _tabIndex == 2
                              ? _buildActivityTab()
                              : _buildOrgsTab(),
                ),
              ],
            ),
          ),
              ),
            ],
          );
        },
      ),
    );
  }

  // ===========================================================================
  // Overview Tab
  // ===========================================================================

  Widget _buildOverviewTab(ConsoleUser user) {
    final initials = _initials(user.name, user.email);

    return SingleChildScrollView(
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 620),
          child: Column(
            children: [
              // Name
              _AccountSection(
                title: 'Name',
                children: [
                  _LabeledField(
                      label: 'Name',
                      controller: _nameCtrl,
                      hint: 'Full name'),
                ],
                onUpdate: _savingName ? null : _updateName,
                loading: _savingName,
              ),

              // Email
              _AccountSection(
                title: 'Email',
                children: [
                  _LabeledField(
                      label: 'Email',
                      controller: _emailCtrl,
                      hint: 'Email address'),
                ],
                onUpdate: _savingEmail ? null : _updateEmail,
                loading: _savingEmail,
              ),

              // Password
              _AccountSection(
                title: 'Password',
                subtitle: 'Choose a strong password you don\'t use elsewhere.',
                children: [
                  _LabeledField(
                      label: 'Old password',
                      controller: _oldPasswordCtrl,
                      hint: 'Enter password',
                      obscure: true),
                  const SizedBox(height: 12),
                  _LabeledField(
                      label: 'New password',
                      controller: _newPasswordCtrl,
                      hint: 'Enter password',
                      obscure: true),
                ],
                onUpdate: _savingPassword ? null : _updatePassword,
                loading: _savingPassword,
              ),

              // MFA
              _AccountSection(
                title: 'Multi-factor authentication',
                subtitle:
                    'Enhance your account\'s security by requiring a second sign-in method.',
                children: [
                  Row(
                    children: [
                      Switch(
                        value: false,
                        onChanged: (_) {},
                        activeColor: _accent,
                      ),
                      const SizedBox(width: 8),
                      const Text('Multi-factor authentication',
                          style: TextStyle(
                              color: Colors.white, fontSize: 13)),
                    ],
                  ),
                ],
              ),

              // Delete account
              Container(
                width: double.infinity,
                margin: const EdgeInsets.only(bottom: 40),
                padding: const EdgeInsets.all(20),
                decoration: BoxDecoration(
                  color: _cardColor,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _red.withOpacity(0.3)),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Expanded(
                          child: Column(
                            crossAxisAlignment:
                                CrossAxisAlignment.start,
                            children: [
                              const Text('Delete account',
                                  style: TextStyle(
                                      color: _red,
                                      fontSize: 14,
                                      fontWeight: FontWeight.w500)),
                              const SizedBox(height: 4),
                              Text(
                                'Your account will be permanently deleted and access will be lost to all your teams and data. This action is irreversible.',
                                style: TextStyle(
                                    color:
                                        Colors.white.withOpacity(0.4),
                                    fontSize: 13),
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(width: 16),
                        // Avatar chip
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 12, vertical: 8),
                          decoration: BoxDecoration(
                            color: Colors.white.withOpacity(0.03),
                            borderRadius: BorderRadius.circular(8),
                            border: Border.all(color: _border),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Container(
                                width: 28,
                                height: 28,
                                decoration: const BoxDecoration(
                                  color: _accent,
                                  shape: BoxShape.circle,
                                ),
                                child: Center(
                                  child: Text(initials,
                                      style: const TextStyle(
                                          color: Colors.white,
                                          fontSize: 11,
                                          fontWeight:
                                              FontWeight.w600)),
                                ),
                              ),
                              const SizedBox(width: 8),
                              Text(user.name,
                                  style: const TextStyle(
                                      color: Colors.white,
                                      fontSize: 13)),
                            ],
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Align(
                      alignment: Alignment.centerRight,
                      child: OutlinedButton(
                        style: OutlinedButton.styleFrom(
                          foregroundColor: _red,
                          side: const BorderSide(color: _red),
                          padding: const EdgeInsets.symmetric(
                              horizontal: 20, vertical: 10),
                          shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(8)),
                        ),
                        onPressed: _deleteAccount,
                        child: const Text('Delete',
                            style: TextStyle(fontSize: 13)),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  // ===========================================================================
  // Placeholder tabs
  // ===========================================================================

  Widget _buildSessionsTab() {
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
            child: const Icon(LucideIcons.monitor,
                size: 22, color: _subtleText),
          ),
          const SizedBox(height: 16),
          const Text('Active sessions',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          const Text('Your active sign-in sessions will appear here.',
              style: TextStyle(color: _dimText, fontSize: 13)),
        ],
      ),
    );
  }

  Widget _buildActivityTab() {
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
            child: const Icon(LucideIcons.activity,
                size: 22, color: _subtleText),
          ),
          const SizedBox(height: 16),
          const Text('Account activity',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          const Text('Recent account activity will appear here.',
              style: TextStyle(color: _dimText, fontSize: 13)),
        ],
      ),
    );
  }

  Widget _buildOrgsTab() {
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
            child: const Icon(LucideIcons.building,
                size: 22, color: _subtleText),
          ),
          const SizedBox(height: 16),
          const Text('Organizations',
              style: TextStyle(
                  color: Colors.white,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          const Text(
              'Organizations you belong to will appear here.',
              style: TextStyle(color: _dimText, fontSize: 13)),
          const SizedBox(height: 16),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: _accent,
              padding: const EdgeInsets.symmetric(
                  horizontal: 20, vertical: 10),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            onPressed: () => context.go('/projects'),
            child: const Text('View organizations',
                style: TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }

  String _initials(String name, String email) {
    if (name.trim().isNotEmpty) {
      final parts = name.trim().split(RegExp(r'\s+'));
      if (parts.length >= 2) {
        return '${parts[0][0]}${parts[1][0]}'.toUpperCase();
      }
      return parts[0][0].toUpperCase();
    }
    return email.isNotEmpty ? email[0].toUpperCase() : '?';
  }
}

// =============================================================================
// Account Section Card (Appwrite-style: title left, fields right, Update btn)
// =============================================================================

class _AccountSection extends StatelessWidget {
  final String title;
  final String? subtitle;
  final List<Widget> children;
  final VoidCallback? onUpdate;
  final bool loading;

  const _AccountSection({
    required this.title,
    this.subtitle,
    required this.children,
    this.onUpdate,
    this.loading = false,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: _cardColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: _border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Left: title + subtitle
              SizedBox(
                width: 200,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(title,
                        style: const TextStyle(
                            color: Colors.white,
                            fontSize: 15,
                            fontWeight: FontWeight.w600)),
                    if (subtitle != null) ...[
                      const SizedBox(height: 4),
                      Text(subtitle!,
                          style: TextStyle(
                              color: Colors.white.withOpacity(0.4),
                              fontSize: 12)),
                    ],
                  ],
                ),
              ),
              const SizedBox(width: 24),
              // Right: fields
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: children,
                ),
              ),
            ],
          ),
          if (onUpdate != null) ...[
            const SizedBox(height: 16),
            Align(
              alignment: Alignment.centerRight,
              child: FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: Colors.white.withOpacity(0.08),
                  foregroundColor: Colors.white,
                  padding: const EdgeInsets.symmetric(
                      horizontal: 20, vertical: 10),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                ),
                onPressed: loading ? null : onUpdate,
                child: loading
                    ? const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(
                            strokeWidth: 2, color: Colors.white))
                    : const Text('Update',
                        style: TextStyle(fontSize: 13)),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _LabeledField extends StatelessWidget {
  final String label;
  final TextEditingController controller;
  final String hint;
  final bool obscure;

  const _LabeledField({
    required this.label,
    required this.controller,
    required this.hint,
    this.obscure = false,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label,
            style: TextStyle(
                color: Colors.white.withOpacity(0.5),
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        TextField(
          controller: controller,
          obscureText: obscure,
          style: const TextStyle(color: Colors.white, fontSize: 13),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: TextStyle(
                color: Colors.white.withOpacity(0.22), fontSize: 13),
            filled: true,
            fillColor: const Color(0x0AFFFFFF),
            isDense: true,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: Color(0x1AFFFFFF))),
            enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: Color(0x1AFFFFFF))),
            focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: _accent)),
            suffixIcon: obscure
                ? const Icon(LucideIcons.eye,
                    size: 16, color: _subtleText)
                : null,
          ),
        ),
      ],
    );
  }
}
