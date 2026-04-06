import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/providers/auth_provider.dart';

class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage> {
  final _emailCtrl = TextEditingController();
  final _passwordCtrl = TextEditingController();
  final _nameCtrl = TextEditingController();
  bool _isSignup = false;
  bool _loading = false;
  String? _error;

  @override
  Widget build(BuildContext context) {
    final signupAsync = ref.watch(signupEnabledProvider);
    final signupEnabled =
        signupAsync.whenOrNull(data: (v) => v) ?? false;

    // If user is already logged in, redirect
    final auth = ref.watch(consoleAuthProvider);
    if (auth.valueOrNull != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) context.go('/onboarding');
      });
    }

    // If signup is not possible and we're in signup mode, switch to login
    if (_isSignup && !signupEnabled) {
      _isSignup = false;
    }

    return Scaffold(
      body: Center(
        child: SingleChildScrollView(
          child: SizedBox(
            width: 400,
            child: Card(
              elevation: 4,
              child: Padding(
                padding: const EdgeInsets.all(32),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(Icons.cloud, size: 56),
                    const SizedBox(height: 8),
                    Text(
                      'Applad',
                      style:
                          Theme.of(context).textTheme.headlineMedium,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      _isSignup
                          ? 'Create your admin account'
                          : 'Sign in to your console',
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                    const SizedBox(height: 32),
                    if (_isSignup)
                      TextField(
                        controller: _nameCtrl,
                        decoration:
                            const InputDecoration(labelText: 'Name'),
                        textInputAction: TextInputAction.next,
                      ),
                    if (_isSignup) const SizedBox(height: 16),
                    TextField(
                      controller: _emailCtrl,
                      decoration:
                          const InputDecoration(labelText: 'Email'),
                      keyboardType: TextInputType.emailAddress,
                      textInputAction: TextInputAction.next,
                    ),
                    const SizedBox(height: 16),
                    TextField(
                      controller: _passwordCtrl,
                      decoration:
                          const InputDecoration(labelText: 'Password'),
                      obscureText: true,
                      onSubmitted: (_) => _submit(),
                    ),
                    if (_error != null) ...[
                      const SizedBox(height: 12),
                      Text(_error!,
                          style: const TextStyle(color: Colors.red)),
                    ],
                    const SizedBox(height: 24),
                    SizedBox(
                      width: double.infinity,
                      child: FilledButton(
                        onPressed: _loading ? null : _submit,
                        child: _loading
                            ? const SizedBox(
                                width: 20,
                                height: 20,
                                child: CircularProgressIndicator(
                                    strokeWidth: 2),
                              )
                            : Text(_isSignup ? 'Sign Up' : 'Sign In'),
                      ),
                    ),
                    if (signupEnabled) ...[
                      const SizedBox(height: 12),
                      TextButton(
                        onPressed: () => setState(() {
                          _isSignup = !_isSignup;
                          _error = null;
                        }),
                        child: Text(_isSignup
                            ? 'Already have an account? Sign in'
                            : 'Create an account'),
                      ),
                    ],
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _submit() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final notifier = ref.read(consoleAuthProvider.notifier);
      if (_isSignup) {
        await notifier.signup(
            _emailCtrl.text, _passwordCtrl.text, _nameCtrl.text);
      } else {
        await notifier.login(_emailCtrl.text, _passwordCtrl.text);
      }
      if (mounted) context.go('/onboarding');
    } catch (e) {
      setState(() => _error = e.toString().replaceAll('Exception: ', ''));
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }
}
