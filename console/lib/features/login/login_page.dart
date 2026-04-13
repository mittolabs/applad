// ignore: avoid_web_libraries_in_flutter, deprecated_member_use
import 'dart:html' as html;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:package_info_plus/package_info_plus.dart';
import '../../core/api/client.dart';
import '../../core/providers/auth_provider.dart';
import '../../core/theme/console_colors.dart';

class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

enum _Mode { login, signup, forgot, reset }

class _LoginPageState extends ConsumerState<LoginPage> {
  final _emailCtrl    = TextEditingController();
  final _passwordCtrl = TextEditingController();
  final _newPassCtrl  = TextEditingController();
  final _confirmCtrl  = TextEditingController();
  final _tokenCtrl    = TextEditingController();
  final _nameCtrl     = TextEditingController();
  _Mode  _mode        = _Mode.login;
  bool   _loading     = false;
  bool   _oauthLoading = false;
  bool   _obscure     = true;
  bool   _obscureNew  = true;
  String? _error;
  String? _success;
  String? _surfacedToken; // token returned when SMTP not configured
  String  _version   = '';
  late ConsoleColors _cs;

  bool get _isSignup => _mode == _Mode.signup;

  @override
  void initState() {
    super.initState();
    _handleOAuthCallback();
    PackageInfo.fromPlatform().then((info) {
      if (mounted) setState(() => _version = info.version);
    });
  }

  /// If the backend redirected back with ?console_token=..., complete the login.
  void _handleOAuthCallback() {
    final uri = Uri.base;

    // Password-reset link: /login?reset_token=xxx
    final resetToken = uri.queryParameters['reset_token'];
    if (resetToken != null && resetToken.isNotEmpty) {
      html.window.history.replaceState(null, '', '/login');
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        setState(() {
          _mode = _Mode.reset;
          _tokenCtrl.text = resetToken;
        });
      });
      return;
    }

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
    _newPassCtrl.dispose();
    _confirmCtrl.dispose();
    _tokenCtrl.dispose();
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

    if (_mode == _Mode.signup && !signupEnabled) _mode = _Mode.login;

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
          Expanded(flex: 6, child: _brandingPanel()),
          Container(width: 1, color: Colors.white.withValues(alpha: 0.06)),
          Expanded(flex: 3, child: _formPanel(signupEnabled)),
        ],
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Left branding panel (wide screens only)
  // ---------------------------------------------------------------------------

  Widget _brandingPanel() {
    return Stack(
      children: [
        // Base background
        Positioned.fill(child: ColoredBox(color: const Color(0xFF0B0B0F))),
        // Abstract background shapes
        Positioned.fill(child: CustomPaint(painter: _PanelShapes())),
        Center(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 56, vertical: 48),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Logo row — mascot + wordmark together, centered
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Container(
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        boxShadow: [
                          BoxShadow(
                            color: const Color(0xFF3472A4).withValues(alpha: 0.7),
                            blurRadius: 10,
                            spreadRadius: 1,
                          ),
                          BoxShadow(
                            color: const Color(0xFF3472A4).withValues(alpha: 0.35),
                            blurRadius: 32,
                            spreadRadius: 4,
                          ),
                        ],
                      ),
                      child: ClipOval(
                        child: Image.asset(
                          'assets/applad-mascot-head.png',
                          width: 52,
                          height: 52,
                          fit: BoxFit.cover,
                        ),
                      ),
                    ),
                    const SizedBox(width: 14),
                    const Text(
                      'applad',
                      style: TextStyle(
                        color: Colors.white,
                        fontSize: 28,
                        fontWeight: FontWeight.w700,
                        letterSpacing: -0.4,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 48),

                // Tagline
                Text(
                  _isSignup ? 'Your backend,' : 'Go from idea',
                  textAlign: TextAlign.start,
                  style: TextStyle(
                    color: Colors.white.withValues(alpha: 0.88),
                    fontSize: 52,
                    fontWeight: FontWeight.w700,
                    height: 1.15,
                    letterSpacing: -0.5,
                  ),
                ),
                Text(
                  _isSignup ? 'your rules.' : 'to production today.',
                  textAlign: TextAlign.start,
                  style: TextStyle(
                    color: Colors.white.withValues(alpha: 0.88),
                    fontSize: 52,
                    fontWeight: FontWeight.w700,
                    height: 1.15,
                    letterSpacing: -0.5,
                  ),
                ),
                const SizedBox(height: 16),
                Text(
                  'Everything your app needs, without compromise.',
                  textAlign: TextAlign.start,
                  style: TextStyle(
                    color: Colors.white.withValues(alpha: 0.35),
                    fontSize: 15,
                    height: 1.5,
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  // ---------------------------------------------------------------------------
  // Right form panel
  // ---------------------------------------------------------------------------

  Widget _formPanel(bool signupEnabled) {
    final isWide = MediaQuery.of(context).size.width > 900;
    return Container(
      color: _cs.background,
      child: Column(
        children: [
          // Logo — shown only on narrow screens (branding panel is hidden)
          if (!isWide)
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 32, 24, 0),
              child: Row(
                children: [
                  Container(
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      boxShadow: [
                        BoxShadow(
                          color: const Color(0xFF3472A4).withValues(alpha: 0.7),
                          blurRadius: 10,
                          spreadRadius: 1,
                        ),
                        BoxShadow(
                          color: const Color(0xFF3472A4).withValues(alpha: 0.35),
                          blurRadius: 32,
                          spreadRadius: 4,
                        ),
                      ],
                    ),
                    child: ClipOval(
                      child: Image.asset(
                        'assets/applad-mascot-head.png',
                        width: 36,
                        height: 36,
                        fit: BoxFit.cover,
                      ),
                    ),
                  ),
                  const SizedBox(width: 10),
                  Text(
                    'applad',
                    style: TextStyle(
                      color: _cs.textPrimary,
                      fontSize: 20,
                      fontWeight: FontWeight.w700,
                      letterSpacing: -0.3,
                    ),
                  ),
                ],
              ),
            ),
          Expanded(
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
                      : switch (_mode) {
                          _Mode.forgot => _forgotForm(),
                          _Mode.reset  => _resetForm(),
                          _            => _loginSignupForm(signupEnabled),
                        },
                ),
              ),
            ),
          ),
          // Version footer
          if (_version.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 16),
              child: Text(
                'v$_version',
                style: TextStyle(color: _cs.textSubtle, fontSize: 11),
              ),
            ),
        ],
      ),
    );
  }

  // ── Login / Sign-up form ───────────────────────────────────────────────────

  Widget _loginSignupForm(bool signupEnabled) {
    final oauthAsync = ref.watch(consoleOAuthProvidersProvider);
    final oauthProviders = oauthAsync.whenOrNull(data: (v) => v) ?? [];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          _isSignup ? 'Sign up' : 'Sign in',
          style: TextStyle(
            color: _cs.textPrimary, fontSize: 28, fontWeight: FontWeight.w700,
          ),
        ),
        const SizedBox(height: 32),

        if (oauthProviders.isNotEmpty && !_isSignup) ...[
          ...oauthProviders.map((p) => _socialButton(p)),
          const SizedBox(height: 24),
          _divider('or'),
          const SizedBox(height: 24),
        ],

        if (_isSignup) ...[
          _label('Name'),
          const SizedBox(height: 6),
          _field(_nameCtrl, 'Your name'),
          const SizedBox(height: 20),
        ],

        _label('Email'),
        const SizedBox(height: 6),
        _field(_emailCtrl, 'Your email', type: TextInputType.emailAddress),
        const SizedBox(height: 20),

        _label('Password'),
        const SizedBox(height: 6),
        _passwordField(_passwordCtrl, onSubmit: _submit),

        if (_isSignup) ...[
          const SizedBox(height: 8),
          Row(children: [
            Icon(LucideIcons.info, size: 14, color: _cs.textSubtle),
            const SizedBox(width: 6),
            Text('Password must be at least 8 characters',
                style: TextStyle(color: _cs.textSubtle, fontSize: 12)),
          ]),
        ],

        _errorBanner(),
        const SizedBox(height: 24),

        _submitBtn(_isSignup ? 'Sign up' : 'Sign in', _submit),
        const SizedBox(height: 16),

        // Links row — forgot + sign up/in toggle
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            _link('Forgot password?', () => setState(() {
              _mode = _Mode.forgot; _error = null; _success = null;
            })),
            if (signupEnabled) ...[
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12),
                child: Text('|', style: TextStyle(color: _cs.textSubtle)),
              ),
              _isSignup
                  ? _link('Sign in', () => setState(() {
                      _mode = _Mode.login; _error = null;
                    }))
                  : _link('Sign up', () => setState(() {
                      _mode = _Mode.signup; _error = null;
                    }),
                      primary: true),
            ],
          ],
        ),
        const SizedBox(height: 16),

        // Legal footer
        Center(
          child: Text.rich(
            TextSpan(
              style: TextStyle(color: _cs.textSubtle, fontSize: 12, height: 1.5),
              children: const [
                TextSpan(text: 'By signing in, you agree to our '),
                TextSpan(
                  text: 'Terms',
                  style: TextStyle(decoration: TextDecoration.underline),
                ),
                TextSpan(text: ' and '),
                TextSpan(
                  text: 'Privacy Policy',
                  style: TextStyle(decoration: TextDecoration.underline),
                ),
                TextSpan(text: '.'),
              ],
            ),
          ),
        ),
      ],
    );
  }

  // ── Forgot password form ───────────────────────────────────────────────────

  Widget _forgotForm() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text('Reset password',
            style: TextStyle(
              color: _cs.textPrimary, fontSize: 28, fontWeight: FontWeight.w700,
            )),
        const SizedBox(height: 8),
        Text(
          'Enter your email and we\'ll send a reset link. '
          'If SMTP is not configured, a one-time token will be shown here.',
          style: TextStyle(color: _cs.textMuted, fontSize: 13, height: 1.5),
        ),
        const SizedBox(height: 28),

        _label('Email'),
        const SizedBox(height: 6),
        _field(_emailCtrl, 'Your email', type: TextInputType.emailAddress),

        if (_surfacedToken != null) ...[
          const SizedBox(height: 20),
          Container(
            padding: const EdgeInsets.all(14),
            decoration: BoxDecoration(
              color: const Color(0xFF3472A4).withValues(alpha: 0.08),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: const Color(0xFF3472A4).withValues(alpha: 0.25)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(children: [
                  const Icon(LucideIcons.info, size: 14, color: Color(0xFF3472A4)),
                  const SizedBox(width: 6),
                  Text('SMTP not configured — use this token:',
                      style: TextStyle(color: _cs.textSecondary, fontSize: 12,
                          fontWeight: FontWeight.w600)),
                ]),
                const SizedBox(height: 8),
                SelectableText(
                  _surfacedToken!,
                  style: const TextStyle(
                    fontFamily: 'monospace', fontSize: 12,
                    color: Color(0xFF3472A4),
                  ),
                ),
                const SizedBox(height: 8),
                GestureDetector(
                  onTap: () => setState(() {
                    _mode = _Mode.reset;
                    _tokenCtrl.text = _surfacedToken!;
                    _surfacedToken = null;
                    _error = null;
                    _success = null;
                  }),
                  child: const Text('Use this token →',
                      style: TextStyle(
                        color: Color(0xFF3472A4), fontSize: 12,
                        fontWeight: FontWeight.w600,
                        decoration: TextDecoration.underline,
                      )),
                ),
              ],
            ),
          ),
        ],

        if (_success != null) ...[
          const SizedBox(height: 16),
          _successBanner(_success!),
        ],

        _errorBanner(),
        const SizedBox(height: 24),

        _submitBtn('Send reset link', _submitForgot),
        const SizedBox(height: 20),

        Center(child: _link('Back to sign in', () => setState(() {
          _mode = _Mode.login; _error = null; _success = null;
          _surfacedToken = null;
        }))),
      ],
    );
  }

  // ── Reset password form ────────────────────────────────────────────────────

  Widget _resetForm() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text('Set new password',
            style: TextStyle(
              color: _cs.textPrimary, fontSize: 28, fontWeight: FontWeight.w700,
            )),
        const SizedBox(height: 8),
        Text('Enter your reset token and choose a new password.',
            style: TextStyle(color: _cs.textMuted, fontSize: 13)),
        const SizedBox(height: 28),

        _label('Reset token'),
        const SizedBox(height: 6),
        _field(_tokenCtrl, 'Paste your token here'),
        const SizedBox(height: 20),

        _label('New password'),
        const SizedBox(height: 6),
        _passwordField(_newPassCtrl, obscureState: _obscureNew,
            onToggle: () => setState(() => _obscureNew = !_obscureNew)),
        const SizedBox(height: 20),

        _label('Confirm password'),
        const SizedBox(height: 6),
        _field(_confirmCtrl, 'Repeat new password'),

        if (_success != null) ...[
          const SizedBox(height: 16),
          _successBanner(_success!),
        ],

        _errorBanner(),
        const SizedBox(height: 24),

        _submitBtn('Set new password', _submitReset),
        const SizedBox(height: 20),

        Center(child: _link('Back to sign in', () => setState(() {
          _mode = _Mode.login; _error = null; _success = null;
        }))),
      ],
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
        height: 38,
        child: OutlinedButton(
          onPressed: () => _continueWithProvider(provider),
          style: OutlinedButton.styleFrom(
            foregroundColor: _cs.textPrimary,
            backgroundColor: _cs.surface,
            side: BorderSide(color: _cs.fieldBorder),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
            ),
            minimumSize: Size.zero,
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
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

  Widget _passwordField(
    TextEditingController ctrl, {
    bool? obscureState,
    VoidCallback? onToggle,
    VoidCallback? onSubmit,
  }) {
    final isObscure = obscureState ?? _obscure;
    final toggle    = onToggle   ?? () => setState(() => _obscure = !_obscure);
    return TextField(
      controller: ctrl,
      obscureText: isObscure,
      onSubmitted: onSubmit != null ? (_) => onSubmit() : null,
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
          onTap: toggle,
          child: Padding(
            padding: const EdgeInsets.only(right: 12),
            child: Icon(
              isObscure ? LucideIcons.eyeOff : LucideIcons.eye,
              size: 16,
              color: _cs.textSubtle,
            ),
          ),
        ),
        suffixIconConstraints: const BoxConstraints(minHeight: 0, minWidth: 0),
      ),
    );
  }

  Widget _link(String label, VoidCallback fn, {bool primary = false}) {
    return GestureDetector(
      onTap: fn,
      child: Text(
        label,
        style: TextStyle(
          color: primary ? const Color(0xFF3472A4) : _cs.textSecondary,
          fontSize: 13,
          fontWeight: FontWeight.w500,
          decoration: TextDecoration.underline,
          decorationColor: primary
              ? const Color(0xFF3472A4)
              : _cs.textSecondary,
        ),
      ),
    );
  }

  Widget _errorBanner() {
    if (_error == null) return const SizedBox.shrink();
    return Padding(
      padding: const EdgeInsets.only(top: 16),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: Colors.red.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: Colors.red.withValues(alpha: 0.2)),
        ),
        child: Row(
          children: [
            const Icon(LucideIcons.alertCircle, size: 14, color: Colors.red),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                _error!,
                style: const TextStyle(color: Colors.red, fontSize: 12),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _successBanner(String msg) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: const Color(0xFF22C55E).withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFF22C55E).withValues(alpha: 0.25)),
      ),
      child: Row(
        children: [
          const Icon(LucideIcons.checkCircle, size: 14, color: Color(0xFF22C55E)),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              msg,
              style: const TextStyle(color: Color(0xFF22C55E), fontSize: 12),
            ),
          ),
        ],
      ),
    );
  }

  Widget _submitBtn(String label, VoidCallback fn) {
    return SizedBox(
      width: double.infinity,
      height: 34,
      child: ElevatedButton(
        onPressed: _loading ? null : fn,
        style: ElevatedButton.styleFrom(
          backgroundColor: const Color(0xFF3472A4),
          foregroundColor: Colors.white,
          disabledBackgroundColor: const Color(0xFF3472A4).withValues(alpha: 0.4),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
          minimumSize: Size.zero,
          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
          padding: EdgeInsets.zero,
          elevation: 0,
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
            : Text(label,
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                )),
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

  Future<void> _submitForgot() async {
    final email = _emailCtrl.text.trim();
    if (email.isEmpty) {
      setState(() => _error = 'Please enter your email address.');
      return;
    }
    setState(() { _loading = true; _error = null; _success = null; _surfacedToken = null; });
    try {
      final api = ref.read(apiClientProvider);
      final result = await requestPasswordReset(api, email);
      final emailSent = result['emailSent'] == true;
      final token    = result['token'] as String?;
      if (mounted) {
        setState(() {
          if (emailSent) {
            _success = 'Reset link sent — check your inbox.';
          } else if (token != null) {
            _surfacedToken = token;
          } else {
            _success = 'If that email is registered, a reset link has been sent.';
          }
        });
      }
    } catch (e) {
      if (mounted) setState(() => _error = 'Something went wrong. Please try again.');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _submitReset() async {
    final token   = _tokenCtrl.text.trim();
    final pass    = _newPassCtrl.text;
    final confirm = _confirmCtrl.text;
    if (token.isEmpty) {
      setState(() => _error = 'Please enter the reset token.');
      return;
    }
    if (pass.length < 8) {
      setState(() => _error = 'Password must be at least 8 characters.');
      return;
    }
    if (pass != confirm) {
      setState(() => _error = 'Passwords do not match.');
      return;
    }
    setState(() { _loading = true; _error = null; _success = null; });
    try {
      final api = ref.read(apiClientProvider);
      await confirmPasswordReset(api, token, pass);
      if (mounted) {
        setState(() {
          _success = 'Password updated. You can now sign in.';
          _newPassCtrl.clear();
          _confirmCtrl.clear();
          _tokenCtrl.clear();
        });
        await Future.delayed(const Duration(seconds: 2));
        if (mounted) setState(() { _mode = _Mode.login; _success = null; });
      }
    } catch (e) {
      var msg = e.toString();
      if (msg.contains('invalid') || msg.contains('expired')) {
        msg = 'Invalid or expired token. Please request a new one.';
      } else {
        msg = 'Something went wrong. Please try again.';
      }
      if (mounted) setState(() => _error = msg);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }
}

// ---------------------------------------------------------------------------
// Abstract panel background — angular swept shapes like folded material
// ---------------------------------------------------------------------------

class _PanelShapes extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final w = size.width;
    final h = size.height;

    // Shape 1 — large sweep from top-right corner, fading toward bottom-left
    _drawShape(
      canvas,
      size,
      path: Path()
        ..moveTo(w * 0.25, 0)
        ..lineTo(w, 0)
        ..lineTo(w, h * 0.55)
        ..cubicTo(w * 0.85, h * 0.42, w * 0.55, h * 0.28, w * 0.05, h * 0.18)
        ..close(),
      begin: Alignment.topRight,
      end: Alignment.bottomLeft,
      colors: [
        Colors.white.withValues(alpha: 0.055),
        Colors.white.withValues(alpha: 0.0),
      ],
    );

    // Shape 2 — tighter inner fold on top-right, slightly lighter
    _drawShape(
      canvas,
      size,
      path: Path()
        ..moveTo(w * 0.55, 0)
        ..lineTo(w, 0)
        ..lineTo(w, h * 0.32)
        ..cubicTo(w * 0.9, h * 0.22, w * 0.75, h * 0.14, w * 0.48, h * 0.07)
        ..close(),
      begin: Alignment.topRight,
      end: Alignment.centerLeft,
      colors: [
        Colors.white.withValues(alpha: 0.07),
        Colors.white.withValues(alpha: 0.01),
      ],
    );

    // Shape 3 — subtle bottom-left ambient shape
    _drawShape(
      canvas,
      size,
      path: Path()
        ..moveTo(0, h * 0.6)
        ..cubicTo(w * 0.15, h * 0.55, w * 0.3, h * 0.7, w * 0.1, h)
        ..lineTo(0, h)
        ..close(),
      begin: Alignment.centerLeft,
      end: Alignment.bottomRight,
      colors: [
        Colors.white.withValues(alpha: 0.025),
        Colors.white.withValues(alpha: 0.0),
      ],
    );

    // Edge highlight — thin bright line along the right edge of shape 1
    _drawShape(
      canvas,
      size,
      path: Path()
        ..moveTo(w * 0.24, 0)
        ..lineTo(w * 0.30, 0)
        ..cubicTo(w * 0.18, h * 0.08, w * 0.08, h * 0.14, w * 0.04, h * 0.19)
        ..cubicTo(w * 0.00, h * 0.14, w * 0.06, h * 0.09, w * 0.19, 0)
        ..close(),
      begin: Alignment.topCenter,
      end: Alignment.bottomCenter,
      colors: [
        Colors.white.withValues(alpha: 0.10),
        Colors.white.withValues(alpha: 0.0),
      ],
    );
  }

  void _drawShape(
    Canvas canvas,
    Size size, {
    required Path path,
    required AlignmentGeometry begin,
    required AlignmentGeometry end,
    required List<Color> colors,
  }) {
    final rect = Offset.zero & size;
    final b = (begin as Alignment);
    final e = (end as Alignment);
    final paint = Paint()
      ..shader = LinearGradient(begin: b, end: e, colors: colors)
          .createShader(rect);
    canvas.drawPath(path, paint);
  }

  @override
  bool shouldRepaint(_PanelShapes old) => false;
}

