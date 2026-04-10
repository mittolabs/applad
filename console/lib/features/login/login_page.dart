import 'dart:html' as html;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/providers/auth_provider.dart';
import '../../core/theme/console_colors.dart';

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
  bool _oauthLoading = false;
  bool _obscure = true;
  String? _error;
  late ConsoleColors _cs;

  @override
  void initState() {
    super.initState();
    _handleOAuthCallback();
  }

  /// If the backend redirected back with ?console_token=..., complete the login.
  void _handleOAuthCallback() {
    final uri = Uri.base;
    final token = uri.queryParameters['console_token'];
    final error = uri.queryParameters['error'];

    if (token != null && token.isNotEmpty) {
      // Strip the token from the URL immediately so it doesn't sit in history.
      html.window.history.replaceState(null, '', '/login');
      WidgetsBinding.instance.addPostFrameCallback((_) async {
        if (!mounted) return;
        setState(() => _oauthLoading = true);
        try {
          await ref.read(consoleAuthProvider.notifier).loginWithToken(token);
          if (mounted) context.go('/onboarding');
        } catch (e) {
          if (mounted) {
            setState(() {
              _error = 'OAuth sign-in failed. Please try again.';
              _oauthLoading = false;
            });
          }
        }
      });
    } else if (error != null) {
      html.window.history.replaceState(null, '', '/login');
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        setState(() {
          _error = switch (error) {
            'signup_disabled' => 'Account creation is disabled. Contact your administrator.',
            'oauth_cancelled' => 'Sign-in was cancelled.',
            _ => 'OAuth sign-in failed. Please try again.',
          };
        });
      });
    }
  }

  @override
  void dispose() {
    _emailCtrl.dispose();
    _passwordCtrl.dispose();
    _nameCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
    final signupAsync = ref.watch(signupEnabledProvider);
    final signupEnabled = signupAsync.whenOrNull(data: (v) => v) ?? true;

    final auth = ref.watch(consoleAuthProvider);
    if (auth.valueOrNull != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) context.go('/onboarding');
      });
    }

    if (_isSignup && !signupEnabled) _isSignup = false;

    final isWide = MediaQuery.of(context).size.width > 900;

    if (!isWide) {
      return Scaffold(
        backgroundColor: _cs.background,
        body: _formPanel(signupEnabled),
      );
    }

    return Scaffold(
      backgroundColor: _cs.background,
      body: Row(
        children: [
          Expanded(flex: 5, child: _brandingPanel()),
          Expanded(flex: 4, child: _formPanel(signupEnabled)),
        ],
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Left branding panel (wide screens only)
  // ---------------------------------------------------------------------------

  Widget _brandingPanel() {
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [Color(0xFF0F1925), Color(0xFF0B0B0F)],
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.all(48),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Logo
            Image.asset(
              'assets/applad-logo.png',
              height: 40,
            ),
            const Spacer(),

            // Tagline
            Text(
              _isSignup
                  ? 'Your backend,\nyour rules.'
                  : 'Ship faster\nwith Applad_',
              style: TextStyle(
                color: Colors.white.withOpacity(0.85),
                fontSize: 48,
                fontWeight: FontWeight.w700,
                height: 1.15,
              ),
            ),
            const SizedBox(height: 24),

            // Subtitle
            Text(
              'Auth, Databases, Storage, Functions, Workflows\n'
              'and Messaging \u2014 one docker compose up.',
              style: TextStyle(
                color: Colors.white.withOpacity(0.4),
                fontSize: 16,
                height: 1.5,
              ),
            ),
            const SizedBox(height: 48),

            // Testimonial card
            Container(
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: Colors.white.withOpacity(0.05),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.white.withOpacity(0.08)),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '\u201cSelf-hosted BaaS with a workflow engine built in \u2014 '
                    'exactly what we needed.\u201d',
                    style: TextStyle(
                      color: Colors.white.withOpacity(0.6),
                      fontSize: 14,
                      fontStyle: FontStyle.italic,
                      height: 1.5,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      CircleAvatar(
                        radius: 16,
                        backgroundColor:
                            const Color(0xFF3472A4).withOpacity(0.3),
                        child: const Text(
                          'OS',
                          style: TextStyle(
                            color: Colors.white,
                            fontSize: 11,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
                      const SizedBox(width: 10),
                      Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'Open Source',
                            style: TextStyle(
                              color: Colors.white.withOpacity(0.7),
                              fontSize: 13,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                          Text(
                            'Community',
                            style: TextStyle(
                              color: Colors.white.withOpacity(0.35),
                              fontSize: 12,
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                ],
              ),
            ),
            const Spacer(flex: 2),
          ],
        ),
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Right form panel
  // ---------------------------------------------------------------------------

  Widget _formPanel(bool signupEnabled) {
    final isWide = MediaQuery.of(context).size.width > 900;
    final oauthAsync = ref.watch(consoleOAuthProvidersProvider);
    final oauthProviders = oauthAsync.whenOrNull(data: (v) => v) ?? [];

    return Container(
      color: _cs.background,
      child: Center(
        child: SingleChildScrollView(
          padding: EdgeInsets.symmetric(
            horizontal: isWide ? 48 : 24,
            vertical: 48,
          ),
          child: SizedBox(
            width: 380,
            child: _oauthLoading
                ? _loadingOverlay()
                : Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      // Heading
                      Text(
                        _isSignup ? 'Sign up' : 'Sign in',
                        style: TextStyle(
                          color: _cs.textPrimary,
                          fontSize: 28,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const SizedBox(height: 32),

                      // Social login buttons (only shown when providers are configured)
                      if (oauthProviders.isNotEmpty && !_isSignup) ...[
                        ...oauthProviders.map((p) => _socialButton(p)),
                        const SizedBox(height: 24),
                        _divider('or'),
                        const SizedBox(height: 24),
                      ],

                      // Name field (signup only)
                      if (_isSignup) ...[
                        _label('Name'),
                        const SizedBox(height: 6),
                        _field(_nameCtrl, 'Your name'),
                        const SizedBox(height: 20),
                      ],

                      // Email
                      _label('Email'),
                      const SizedBox(height: 6),
                      _field(_emailCtrl, 'Your email',
                          type: TextInputType.emailAddress),
                      const SizedBox(height: 20),

                      // Password
                      _label('Password'),
                      const SizedBox(height: 6),
                      _passwordField(),

                      // Password hint on signup
                      if (_isSignup) ...[
                        const SizedBox(height: 8),
                        Row(
                          children: [
                            Icon(LucideIcons.info,
                                size: 14, color: _cs.textSubtle),
                            const SizedBox(width: 6),
                            Text(
                              'Password must be at least 8 characters',
                              style: TextStyle(
                                color: _cs.textSubtle,
                                fontSize: 12,
                              ),
                            ),
                          ],
                        ),
                      ],

                      // Error
                      if (_error != null) ...[
                        const SizedBox(height: 16),
                        Container(
                          padding: const EdgeInsets.all(12),
                          decoration: BoxDecoration(
                            color: Colors.red.withOpacity(0.1),
                            borderRadius: BorderRadius.circular(8),
                            border:
                                Border.all(color: Colors.red.withOpacity(0.3)),
                          ),
                          child: Row(
                            children: [
                              const Icon(LucideIcons.alertCircle,
                                  size: 16, color: Colors.redAccent),
                              const SizedBox(width: 8),
                              Expanded(
                                child: Text(
                                  _error!,
                                  style: const TextStyle(
                                    color: Colors.redAccent,
                                    fontSize: 13,
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ],

                      const SizedBox(height: 24),

                      // Submit button
                      SizedBox(
                        width: double.infinity,
                        height: 36,
                        child: FilledButton(
                          onPressed: _loading ? null : _submit,
                          style: FilledButton.styleFrom(
                            backgroundColor: const Color(0xFF3472A4),
                            padding: const EdgeInsets.symmetric(vertical: 0),
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(8),
                            ),
                          ),
                          child: _loading
                              ? const SizedBox(
                                  width: 16,
                                  height: 16,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                    color: Colors.white,
                                  ),
                                )
                              : Text(
                                  _isSignup ? 'Sign up' : 'Sign in',
                                  style: const TextStyle(
                                    fontSize: 13,
                                    fontWeight: FontWeight.w500,
                                  ),
                                ),
                        ),
                      ),

                      const SizedBox(height: 20),

                      // Toggle links
                      if (signupEnabled)
                        Center(
                          child: _isSignup
                              ? Row(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    Text(
                                      'Already have an account? ',
                                      style: TextStyle(
                                        color: _cs.textMuted,
                                        fontSize: 13,
                                      ),
                                    ),
                                    GestureDetector(
                                      onTap: () => setState(() {
                                        _isSignup = false;
                                        _error = null;
                                      }),
                                      child: const Text(
                                        'Sign in',
                                        style: TextStyle(
                                          color: Color(0xFF3472A4),
                                          fontSize: 13,
                                          fontWeight: FontWeight.w600,
                                          decoration: TextDecoration.underline,
                                        ),
                                      ),
                                    ),
                                  ],
                                )
                              : Row(
                                  mainAxisAlignment: MainAxisAlignment.center,
                                  children: [
                                    GestureDetector(
                                      onTap: () {},
                                      child: Text(
                                        'Forgot password?',
                                        style: TextStyle(
                                          color: _cs.textMuted,
                                          fontSize: 13,
                                        ),
                                      ),
                                    ),
                                    Padding(
                                      padding: const EdgeInsets.symmetric(
                                          horizontal: 12),
                                      child: Text(
                                        '|',
                                        style: TextStyle(
                                          color: _cs.textSubtle,
                                        ),
                                      ),
                                    ),
                                    GestureDetector(
                                      onTap: () => setState(() {
                                        _isSignup = true;
                                        _error = null;
                                      }),
                                      child: const Text(
                                        'Sign up',
                                        style: TextStyle(
                                          color: Color(0xFF3472A4),
                                          fontSize: 13,
                                          fontWeight: FontWeight.w600,
                                          decoration: TextDecoration.underline,
                                        ),
                                      ),
                                    ),
                                  ],
                                ),
                        ),
                    ],
                  ),
          ),
        ),
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Social login button
  // ---------------------------------------------------------------------------

  Widget _socialButton(String provider) {
    final (label, icon) = switch (provider) {
      'github' => ('Continue with GitHub', Icon(LucideIcons.github, size: 18, color: _cs.textPrimary)),
      'google' => ('Continue with Google', _googleIcon()),
      'sso'    => ('Continue with SSO', Icon(LucideIcons.building2, size: 18, color: _cs.textSecondary)),
      _        => ('Continue with ${provider[0].toUpperCase()}${provider.substring(1)}',
                   Icon(LucideIcons.logIn, size: 18, color: _cs.textSecondary)),
    };

    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: SizedBox(
        width: double.infinity,
        height: 40,
        child: OutlinedButton(
          onPressed: () => _continueWithProvider(provider),
          style: OutlinedButton.styleFrom(
            foregroundColor: _cs.textPrimary,
            backgroundColor: _cs.surface,
            side: BorderSide(color: _cs.fieldBorder),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
            ),
            padding: const EdgeInsets.symmetric(horizontal: 16),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              icon,
              const SizedBox(width: 10),
              Text(
                label,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                  color: _cs.textPrimary,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _googleIcon() {
    return const Text('G', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w700, color: Color(0xFF4285F4)));
  }

  Widget _divider(String label) {
    return Row(
      children: [
        Expanded(child: Divider(color: _cs.fieldBorder, thickness: 1)),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12),
          child: Text(label, style: TextStyle(color: _cs.textSubtle, fontSize: 12)),
        ),
        Expanded(child: Divider(color: _cs.fieldBorder, thickness: 1)),
      ],
    );
  }

  Widget _loadingOverlay() {
    return SizedBox(
      height: 300,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const CircularProgressIndicator(
              strokeWidth: 2,
              color: Color(0xFF3472A4),
            ),
            const SizedBox(height: 16),
            Text('Signing you in…',
                style: TextStyle(color: _cs.textMuted, fontSize: 14)),
          ],
        ),
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------------

  Widget _label(String text) {
    return Text(
      text,
      style: TextStyle(
        color: _cs.textSecondary,
        fontSize: 13,
        fontWeight: FontWeight.w500,
      ),
    );
  }

  Widget _field(TextEditingController ctrl, String hint,
      {TextInputType type = TextInputType.text}) {
    return TextField(
      controller: ctrl,
      keyboardType: type,
      textInputAction: TextInputAction.next,
      style: TextStyle(color: _cs.textPrimary, fontSize: 13),
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: TextStyle(color: _cs.textSubtle, fontSize: 13),
        filled: true,
        fillColor: _cs.fieldFill,
        isDense: true,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: _cs.fieldBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: _cs.fieldBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: Color(0xFF3472A4)),
        ),
      ),
    );
  }

  Widget _passwordField() {
    return TextField(
      controller: _passwordCtrl,
      obscureText: _obscure,
      onSubmitted: (_) => _submit(),
      style: TextStyle(color: _cs.textPrimary, fontSize: 13),
      decoration: InputDecoration(
        hintText: 'Your password',
        hintStyle: TextStyle(color: _cs.textSubtle, fontSize: 13),
        filled: true,
        fillColor: _cs.fieldFill,
        isDense: true,
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: _cs.fieldBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: _cs.fieldBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: Color(0xFF3472A4)),
        ),
        suffixIcon: GestureDetector(
          onTap: () => setState(() => _obscure = !_obscure),
          child: Padding(
            padding: const EdgeInsets.only(right: 12),
            child: Icon(
              _obscure ? LucideIcons.eyeOff : LucideIcons.eye,
              size: 16,
              color: _cs.textSubtle,
            ),
          ),
        ),
        suffixIconConstraints: const BoxConstraints(minHeight: 0, minWidth: 0),
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Actions
  // ---------------------------------------------------------------------------

  void _continueWithProvider(String provider) {
    // Full-page redirect — the backend handles the OAuth dance and redirects
    // back to /login?console_token=<jwt> which _handleOAuthCallback picks up.
    html.window.location.href = '/v1/console/auth/$provider';
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
      var msg = e.toString();
      if (msg.contains('DioException') || msg.contains('status code')) {
        if (msg.contains('401') || msg.contains('403')) {
          msg = 'Invalid email or password';
        } else if (msg.contains('409')) {
          msg = 'An account with this email already exists';
        } else if (msg.contains('400')) {
          msg = 'Please check your input and try again';
        } else if (msg.contains('404') || msg.contains('405')) {
          msg = 'Could not reach the API. Make sure the server is running.';
        } else {
          msg = 'Something went wrong. Please try again.';
        }
      }
      setState(() => _error = msg);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }
}

