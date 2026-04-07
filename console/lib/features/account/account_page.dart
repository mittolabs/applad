import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/api/client.dart';
import '../../core/providers/auth_provider.dart';
import '../../core/widgets/app_dialog.dart';

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
  bool _mfaEnabled = false;
  bool _initialized = false;

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
    try {
      final api = ref.read(apiClientProvider);
      await api.patch('/console/me', data: {'name': _nameCtrl.text});
      ref.invalidate(consoleAuthProvider);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Name updated')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to update name: $e')),
        );
      }
    }
  }

  Future<void> _updateEmail() async {
    try {
      final api = ref.read(apiClientProvider);
      await api.patch('/console/me', data: {'email': _emailCtrl.text});
      ref.invalidate(consoleAuthProvider);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Email updated')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to update email: $e')),
        );
      }
    }
  }

  Future<void> _updatePassword() async {
    if (_newPasswordCtrl.text.isEmpty || _oldPasswordCtrl.text.isEmpty) return;
    try {
      final api = ref.read(apiClientProvider);
      await api.patch('/console/me/password', data: {
        'oldPassword': _oldPasswordCtrl.text,
        'password': _newPasswordCtrl.text,
      });
      _oldPasswordCtrl.clear();
      _newPasswordCtrl.clear();
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Password updated')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to update password: $e')),
        );
      }
    }
  }

  Future<void> _deleteAccount() async {
    final confirmed = await showAppDialog<bool>(
      context: context,
      title: 'Delete account',
      content: Text(
        'This action is irreversible. All your data will be permanently deleted.',
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
      try {
        final api = ref.read(apiClientProvider);
        await api.delete('/console/me');
        ref.read(consoleAuthProvider.notifier).logout();
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Failed to delete account: $e')),
          );
        }
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final authAsync = ref.watch(consoleAuthProvider);
    final user = authAsync.valueOrNull;

    if (user != null) _initFields(user);

    return Scaffold(
      backgroundColor: const Color(0xFF0B0B0F),
      body: authAsync.when(
        loading: () => const Center(
          child: CircularProgressIndicator(color: Colors.white24),
        ),
        error: (e, _) => Center(
          child: Text('Error: $e',
              style: const TextStyle(color: Colors.white70)),
        ),
        data: (user) {
          if (user == null) {
            return const Center(
              child:
                  Text('Not logged in', style: TextStyle(color: Colors.white70)),
            );
          }
          return SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 40, vertical: 32),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 640),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Page header
                  const Text(
                    'Account',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: 24,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    'Manage your account settings and preferences.',
                    style: TextStyle(color: Colors.white.withOpacity(0.5), fontSize: 14),
                  ),
                  const SizedBox(height: 32),

                  // Name section
                  _buildSection(
                    title: 'Name',
                    description: 'Your display name across the platform.',
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _buildTextField(_nameCtrl, 'Full name'),
                        const SizedBox(height: 16),
                        _buildUpdateButton(onPressed: _updateName),
                      ],
                    ),
                  ),

                  // Email section
                  _buildSection(
                    title: 'Email',
                    description: 'The email address associated with your account.',
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _buildTextField(_emailCtrl, 'Email address'),
                        const SizedBox(height: 16),
                        _buildUpdateButton(onPressed: _updateEmail),
                      ],
                    ),
                  ),

                  // Password section
                  _buildSection(
                    title: 'Password',
                    description: 'Update your password to keep your account secure.',
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _buildTextField(_oldPasswordCtrl, 'Current password',
                            obscure: true),
                        const SizedBox(height: 12),
                        _buildTextField(_newPasswordCtrl, 'New password',
                            obscure: true),
                        const SizedBox(height: 16),
                        _buildUpdateButton(onPressed: _updatePassword),
                      ],
                    ),
                  ),

                  // MFA section
                  _buildSection(
                    title: 'Multi-factor authentication',
                    description:
                        'Add an extra layer of security to your account.',
                    child: Row(
                      children: [
                        Switch(
                          value: _mfaEnabled,
                          onChanged: (val) => setState(() => _mfaEnabled = val),
                          activeColor: const Color(0xFF3472A4),
                        ),
                        const SizedBox(width: 12),
                        Text(
                          _mfaEnabled ? 'Enabled' : 'Disabled',
                          style: TextStyle(
                            color: Colors.white.withOpacity(0.7),
                            fontSize: 14,
                          ),
                        ),
                      ],
                    ),
                  ),

                  // Danger zone
                  const SizedBox(height: 16),
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.all(24),
                    decoration: BoxDecoration(
                      color: const Color(0xFF1A1A22),
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(color: Colors.red.withOpacity(0.3)),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text(
                          'Danger zone',
                          style: TextStyle(
                            color: Colors.red,
                            fontSize: 16,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 8),
                        Text(
                          'Permanently delete your account and all associated data. This action cannot be undone.',
                          style: TextStyle(
                              color: Colors.white.withOpacity(0.5),
                              fontSize: 13),
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
                          onPressed: _deleteAccount,
                          child: const Text('Delete account'),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 48),
                ],
              ),
            ),
          );
        },
      ),
    );
  }

  Widget _buildSection({
    required String title,
    required String description,
    required Widget child,
  }) {
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: 24),
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: const Color(0xFF1A1A22),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.white.withOpacity(0.06)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 16,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            description,
            style: TextStyle(color: Colors.white.withOpacity(0.5), fontSize: 13),
          ),
          const SizedBox(height: 16),
          child,
        ],
      ),
    );
  }

  Widget _buildTextField(TextEditingController ctrl, String hint,
      {bool obscure = false}) {
    return TextField(
      controller: ctrl,
      obscureText: obscure,
      style: const TextStyle(color: Colors.white, fontSize: 14),
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: TextStyle(color: Colors.white.withOpacity(0.25)),
        filled: true,
        fillColor: const Color(0xFF0B0B0F),
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: Color(0xFF3472A4)),
        ),
      ),
    );
  }

  Widget _buildUpdateButton({required VoidCallback onPressed}) {
    return FilledButton(
      style: FilledButton.styleFrom(
        backgroundColor: Colors.white.withOpacity(0.08),
        foregroundColor: Colors.white,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
      ),
      onPressed: onPressed,
      child: const Text('Update', style: TextStyle(fontSize: 13)),
    );
  }
}
