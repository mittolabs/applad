import 'package:flutter/material.dart';

import '../applad_service.dart';
import 'channels_screen.dart';

/// Email/password sign in and sign up. Exercises audit item G1: the session
/// secret returned here is what makes auth work on mobile as well as web.
class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _name = TextEditingController();
  final _email = TextEditingController();
  final _password = TextEditingController();
  bool _signUp = false;
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _name.dispose();
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final svc = AppladService.instance;
      if (_signUp) {
        await svc.signUp(_name.text.trim(), _email.text.trim(), _password.text);
      } else {
        await svc.signIn(_email.text.trim(), _password.text);
      }
      if (!mounted) return;
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => const ChannelsScreen()),
      );
    } catch (e) {
      setState(() => _error = _humanError(e));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 380),
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Icon(Icons.forum_outlined, size: 44, color: Color(0xFF6C47FF)),
                const SizedBox(height: 12),
                Text('Applad Chat',
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.headlineSmall),
                const SizedBox(height: 24),
                if (_signUp) ...[
                  TextField(
                    controller: _name,
                    decoration: const InputDecoration(labelText: 'Name'),
                    textInputAction: TextInputAction.next,
                  ),
                  const SizedBox(height: 12),
                ],
                TextField(
                  controller: _email,
                  decoration: const InputDecoration(labelText: 'Email'),
                  keyboardType: TextInputType.emailAddress,
                  autofillHints: const [AutofillHints.email],
                  textInputAction: TextInputAction.next,
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _password,
                  decoration: const InputDecoration(labelText: 'Password'),
                  obscureText: true,
                  onSubmitted: (_) => _busy ? null : _submit(),
                ),
                if (_error != null) ...[
                  const SizedBox(height: 12),
                  Text(_error!, style: const TextStyle(color: Colors.redAccent)),
                ],
                const SizedBox(height: 20),
                FilledButton(
                  onPressed: _busy ? null : _submit,
                  child: _busy
                      ? const SizedBox(
                          height: 18, width: 18, child: CircularProgressIndicator(strokeWidth: 2))
                      : Text(_signUp ? 'Create account' : 'Sign in'),
                ),
                TextButton(
                  onPressed: _busy ? null : () => setState(() => _signUp = !_signUp),
                  child: Text(_signUp
                      ? 'Have an account? Sign in'
                      : 'New here? Create an account'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

String _humanError(Object e) {
  final s = e.toString();
  if (s.contains('invalid credentials') || s.contains('401')) {
    return 'Wrong email or password.';
  }
  if (s.contains('already in use') || s.contains('409')) {
    return 'That email is already registered.';
  }
  return 'Something went wrong. Please try again.';
}
