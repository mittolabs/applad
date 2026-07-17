import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/auth_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/app_navbar.dart';
import '../../core/widgets/page_tabs.dart';
import '../../core/widgets/app_error_state.dart';

const _accent = Color(0xFF3472A4);
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
    final colors = consoleColors(context);
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete account',
      content: Text(
        'Your account will be permanently deleted and access will be lost to all your teams and data. This action is irreversible.',
        style: TextStyle(color: colors.textSecondary),
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
    final colors = consoleColors(context);
    final authAsync = ref.watch(consoleAuthProvider);
    final user = authAsync.valueOrNull;
    if (user != null) _initFields(user);

    return Scaffold(
      backgroundColor: colors.background,
      body: authAsync.when(
        loading: () => Center(
            child: CircularProgressIndicator(color: colors.textSubtle)),
        error: (e, _) => AppErrorState(error: e),
        data: (user) {
          if (user == null) {
            return Center(
                child: Text('Not logged in',
                    style: TextStyle(color: colors.textSecondary)));
          }
          return Column(
            children: [
              const AppNavBar(),
              Expanded(
                child: Padding(
            padding: EdgeInsets.symmetric(
              horizontal:
                  pageHPad(context),
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
                      style: TextStyle(
                          color: colors.textPrimary,
                          fontSize: 22,
                          fontWeight: FontWeight.w600),
                    ),
                    const Spacer(),
                    TextButton(
                      onPressed: () async {
                        final confirmed = await showAppDialog<bool>(
                          context: context,
                          title: 'Sign out',
                          content: Text(
                            'Are you sure you want to sign out?',
                            style: TextStyle(color: colors.textSecondary),
                          ),
                          actions: [
                            const AppDialogCancel(),
                            AppDialogAction(
                              label: 'Sign out',
                              destructive: true,
                              onTap: () => Navigator.of(context, rootNavigator: true).pop(true),
                            ),
                          ],
                        );
                        if (confirmed == true && context.mounted) {
                          ref.read(consoleAuthProvider.notifier).logout();
                          context.go('/login');
                        }
                      },
                      style: TextButton.styleFrom(
                          foregroundColor: colors.textSecondary),
                      child: const Text('Sign out',
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
    final colors = consoleColors(context);
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
                onUpdate: _savingName ? null : _updateName,
                loading: _savingName,
                children: [
                  _LabeledField(
                      label: 'Name',
                      controller: _nameCtrl,
                      hint: 'Full name'),
                ],
              ),

              // Email
              _AccountSection(
                title: 'Email',
                onUpdate: _savingEmail ? null : _updateEmail,
                loading: _savingEmail,
                children: [
                  _LabeledField(
                      label: 'Email',
                      controller: _emailCtrl,
                      hint: 'Email address'),
                ],
              ),

              // Password
              _AccountSection(
                title: 'Password',
                subtitle: 'Choose a strong password you don\'t use elsewhere.',
                onUpdate: _savingPassword ? null : _updatePassword,
                loading: _savingPassword,
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
                        activeThumbColor: _accent,
                      ),
                      const SizedBox(width: 8),
                      Text('Multi-factor authentication',
                          style: TextStyle(
                              color: colors.textPrimary, fontSize: 13)),
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
                  color: colors.surface,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: _red.withValues(alpha: 0.3)),
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
                                  color: colors.textSecondary,
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
                            color: colors.fill,
                            borderRadius: BorderRadius.circular(8),
                            border: Border.all(color: colors.border),
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
                                  style: TextStyle(
                                    color: colors.textPrimary,
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
            child: Icon(LucideIcons.monitor,
                size: 22, color: colors.textSubtle),
          ),
          const SizedBox(height: 16),
          Text('Active sessions',
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          Text('Your active sign-in sessions will appear here.',
              style: TextStyle(color: colors.textSecondary, fontSize: 13)),
        ],
      ),
    );
  }

  Widget _buildActivityTab() {
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
            child: Icon(LucideIcons.activity,
                size: 22, color: colors.textSubtle),
          ),
          const SizedBox(height: 16),
          Text('Account activity',
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          Text('Recent account activity will appear here.',
              style: TextStyle(color: colors.textSecondary, fontSize: 13)),
        ],
      ),
    );
  }

  Widget _buildOrgsTab() {
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
            child: Icon(LucideIcons.building,
                size: 22, color: colors.textSubtle),
          ),
          const SizedBox(height: 16),
          Text('Organizations',
              style: TextStyle(
                  color: colors.textPrimary,
                  fontSize: 15,
                  fontWeight: FontWeight.w500)),
          const SizedBox(height: 6),
          Text(
              'Organizations you belong to will appear here.',
              style: TextStyle(color: colors.textSecondary, fontSize: 13)),
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
    final colors = consoleColors(context);
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colors.border),
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
                      style: TextStyle(
                        color: colors.textPrimary,
                            fontSize: 15,
                            fontWeight: FontWeight.w600)),
                    if (subtitle != null) ...[
                      const SizedBox(height: 4),
                      Text(subtitle!,
                          style: TextStyle(
                            color: colors.textSecondary,
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
                  backgroundColor: colors.fillActive,
                  foregroundColor: colors.textPrimary,
                  padding: const EdgeInsets.symmetric(
                      horizontal: 20, vertical: 10),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                ),
                onPressed: loading ? null : onUpdate,
                child: loading
                    ? SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(
                      strokeWidth: 2, color: colors.textPrimary))
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
    final colors = consoleColors(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label,
            style: TextStyle(
                color: colors.textSecondary,
                fontSize: 12,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 6),
        TextField(
          controller: controller,
          obscureText: obscure,
          style: TextStyle(color: colors.textPrimary, fontSize: 13),
          decoration: InputDecoration(
            hintText: hint,
            hintStyle: TextStyle(
            color: colors.textSubtle, fontSize: 13),
            filled: true,
          fillColor: colors.fieldFill,
            isDense: true,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: colors.fieldBorder)),
            enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: colors.fieldBorder)),
            focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: _accent)),
            suffixIcon: obscure
            ? Icon(LucideIcons.eye,
              size: 16, color: colors.textSubtle)
                : null,
          ),
        ),
      ],
    );
  }
}
