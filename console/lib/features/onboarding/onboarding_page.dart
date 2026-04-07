import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/auth_provider.dart';
import '../../core/providers/org_provider.dart';

const _bg = Color(0xFF0B0B0F);
const _surface = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _border = Color(0x14FFFFFF);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);

class OnboardingPage extends ConsumerStatefulWidget {
  const OnboardingPage({super.key});

  @override
  ConsumerState<OnboardingPage> createState() => _OnboardingPageState();
}

class _OnboardingPageState extends ConsumerState<OnboardingPage> {
  final _orgNameCtrl = TextEditingController();
  bool _loading = false;
  bool _nameSet = false;

  @override
  void dispose() {
    _orgNameCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(consoleAuthProvider);
    final user = auth.valueOrNull;

    // Set default org name from user's name (once)
    if (user != null && !_nameSet) {
      _nameSet = true;
      final name = user.name.trim();
      if (name.isNotEmpty) {
        final firstName = name.split(' ').first;
        _orgNameCtrl.text = "$firstName's Workspace";
      } else {
        _orgNameCtrl.text = 'My Workspace';
      }
    }

    if (user == null && !auth.isLoading) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) context.go('/login');
      });
      return const SizedBox.shrink();
    }

    // If orgs already exist, skip onboarding
    final orgsAsync = ref.watch(orgsProvider);
    final orgs = orgsAsync.valueOrNull ?? [];
    if (orgs.isNotEmpty) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) context.go('/projects');
      });
      return const SizedBox.shrink();
    }

    final isWide = MediaQuery.of(context).size.width > 800;
    final userName = user?.name ?? '';

    return Scaffold(
      backgroundColor: _bg,
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(32),
          child: Container(
            width: isWide ? 480 : double.infinity,
            padding: const EdgeInsets.all(40),
            decoration: BoxDecoration(
              color: _surface,
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: _border),
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withOpacity(0.3),
                  blurRadius: 40,
                  offset: const Offset(0, 16),
                ),
              ],
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                // Logo
                ClipRRect(
                  borderRadius: BorderRadius.circular(12),
                  child: Image.asset(
                    'assets/icon.png',
                    width: 56,
                    height: 56,
                    fit: BoxFit.cover,
                  ),
                ),
                const SizedBox(height: 24),

                // Welcome
                Text(
                  userName.isNotEmpty
                      ? 'Welcome, $userName'
                      : 'Welcome to Applad',
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 24,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Create your organization to get started.\nOrganizations help you manage projects and team members.',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    color: Colors.white.withOpacity(0.45),
                    fontSize: 14,
                    height: 1.5,
                  ),
                ),
                const SizedBox(height: 32),

                // Org name field
                Align(
                  alignment: Alignment.centerLeft,
                  child: Text(
                    'Organization name',
                    style: TextStyle(
                      color: Colors.white.withOpacity(0.5),
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
                const SizedBox(height: 8),
                TextField(
                  controller: _orgNameCtrl,
                  autofocus: true,
                  style: const TextStyle(color: Colors.white, fontSize: 14),
                  decoration: InputDecoration(
                    hintText: 'e.g. Acme Inc',
                    hintStyle: TextStyle(
                        color: Colors.white.withOpacity(0.22), fontSize: 14),
                    filled: true,
                    fillColor: Colors.white.withOpacity(0.04),
                    contentPadding: const EdgeInsets.symmetric(
                        horizontal: 14, vertical: 12),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide:
                          BorderSide(color: Colors.white.withOpacity(0.1)),
                    ),
                    enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide:
                          BorderSide(color: Colors.white.withOpacity(0.1)),
                    ),
                    focusedBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: const BorderSide(color: _accent),
                    ),
                  ),
                  onSubmitted: (_) => _create(),
                ),
                const SizedBox(height: 24),

                // Create button
                SizedBox(
                  width: double.infinity,
                  height: 44,
                  child: FilledButton(
                    onPressed: _loading ? null : _create,
                    style: FilledButton.styleFrom(
                      backgroundColor: _accent,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                    child: _loading
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.white,
                            ),
                          )
                        : const Text(
                            'Get Started',
                            style: TextStyle(
                              fontSize: 15,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                  ),
                ),

                const SizedBox(height: 20),

                // Skip link
                GestureDetector(
                  onTap: () => context.go('/projects'),
                  child: Text(
                    'Skip for now',
                    style: TextStyle(
                      color: Colors.white.withOpacity(0.3),
                      fontSize: 13,
                      decoration: TextDecoration.underline,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _create() async {
    if (_orgNameCtrl.text.trim().isEmpty) return;
    setState(() => _loading = true);
    try {
      final api = ref.read(apiClientProvider);
      final res = await api.post('/organizations', data: {
        'name': _orgNameCtrl.text.trim(),
      });
      final data = res.data as Map<String, dynamic>;
      final orgId = data['\$id'] as String?;
      ref.invalidate(orgsProvider);
      if (orgId != null) {
        ref.read(currentOrgProvider.notifier).state = orgId;
      }
      if (mounted) context.go('/projects');
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }
}
