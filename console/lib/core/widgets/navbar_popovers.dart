// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../providers/auth_provider.dart';
import '../providers/theme_provider.dart';

bool _isLight(BuildContext context) =>
  Theme.of(context).brightness == Brightness.light;

Color _popoverSurface(BuildContext context) =>
  _isLight(context) ? Colors.white : const Color(0xFF1C1C24);

Color _popoverBorder(BuildContext context) => _isLight(context)
  ? Colors.black.withOpacity(0.08)
  : Colors.white.withOpacity(0.08);

Color _popoverShadow(BuildContext context) => _isLight(context)
  ? Colors.black.withOpacity(0.08)
  : Colors.black.withOpacity(0.4);

Color _primaryText(BuildContext context) =>
  _isLight(context) ? const Color(0xFF1A1A2E) : Colors.white;

Color _secondaryText(BuildContext context) => _isLight(context)
  ? const Color(0xFF1A1A2E).withOpacity(0.8)
  : Colors.white.withOpacity(0.8);

Color _mutedText(BuildContext context) => _isLight(context)
  ? Colors.black.withOpacity(0.35)
  : Colors.white.withOpacity(0.35);

Color _subtleFill(BuildContext context) => _isLight(context)
  ? Colors.black.withOpacity(0.04)
  : Colors.white.withOpacity(0.06);

Color _hoverFill(BuildContext context) => _isLight(context)
  ? Colors.black.withOpacity(0.04)
  : Colors.white.withOpacity(0.04);

Color _dividerColor(BuildContext context) => _isLight(context)
  ? Colors.black.withOpacity(0.08)
  : Colors.white.withOpacity(0.06);

// ── Shared overlay helper ────────────────────────────────────────────────────

OverlayEntry _buildOverlay({
  required double top,
  required double right,
  required VoidCallback onClose,
  required Widget panel,
}) {
  return OverlayEntry(
    builder: (_) => Stack(
      children: [
        // Full-screen dismiss barrier
        Positioned.fill(
          child: GestureDetector(
            behavior: HitTestBehavior.translucent,
            onTap: onClose,
          ),
        ),
        // Panel
        Positioned(
          top: top,
          right: right,
          child: Material(color: Colors.transparent, child: panel),
        ),
      ],
    ),
  );
}

// ── Shared ghost button (with active state) ──────────────────────────────────

class NavGhostButton extends StatefulWidget {
  final String label;
  final VoidCallback onTap;
  final bool active;
  const NavGhostButton(
      {super.key,
      required this.label,
      required this.onTap,
      this.active = false});

  @override
  State<NavGhostButton> createState() => _NavGhostButtonState();
}

class _NavGhostButtonState extends State<NavGhostButton> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final highlight = _hovered || widget.active;
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          decoration: BoxDecoration(
            color: highlight
                ? _subtleFill(context)
                : Colors.transparent,
            borderRadius: BorderRadius.circular(7),
          ),
          child: Text(
            widget.label,
            style: TextStyle(
              color: (_isLight(context)
                      ? const Color(0xFF1A1A2E)
                      : Colors.white)
                  .withOpacity(highlight ? 0.8 : 0.45),
              fontSize: 13,
            ),
          ),
        ),
      ),
    );
  }
}

// ── Feedback button ───────────────────────────────────────────────────────────

class FeedbackButton extends StatefulWidget {
  const FeedbackButton({super.key});

  @override
  State<FeedbackButton> createState() => _FeedbackButtonState();
}

class _FeedbackButtonState extends State<FeedbackButton> {
  OverlayEntry? _overlay;

  bool get _open => _overlay != null;

  void _close() {
    _overlay?.remove();
    _overlay = null;
    if (mounted) setState(() {});
  }

  void _toggle() {
    if (_open) {
      _close();
      return;
    }
    _overlay = _buildOverlay(
      top: 56,
      right: 8,
      onClose: _close,
      panel: _FeedbackPanel(onClose: _close),
    );
    Overlay.of(context).insert(_overlay!);
    setState(() {});
  }

  @override
  void dispose() {
    _overlay?.remove();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return NavGhostButton(
      label: 'Feedback',
      active: _open,
      onTap: _toggle,
    );
  }
}

// ── Feedback panel ────────────────────────────────────────────────────────────

class _FeedbackPanel extends StatefulWidget {
  final VoidCallback onClose;
  const _FeedbackPanel({required this.onClose});

  @override
  State<_FeedbackPanel> createState() => _FeedbackPanelState();
}

class _FeedbackPanelState extends State<_FeedbackPanel> {
  final _ctrl = TextEditingController();
  String _category = 'General';
  bool _loading = false;
  bool _submitted = false;

  static const _categories = ['Bug report', 'Feature request', 'General'];

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_ctrl.text.trim().isEmpty) return;
    setState(() => _loading = true);
    // Simulate async submission
    await Future.delayed(const Duration(milliseconds: 800));
    if (mounted) setState(() {
      _loading = false;
      _submitted = true;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 400,
      decoration: BoxDecoration(
        color: const Color(0xFF16171B),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.white.withOpacity(0.08)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.5),
            blurRadius: 32,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: _submitted ? _buildSuccess() : _buildForm(),
    );
  }

  Widget _buildSuccess() {
    return Padding(
      padding: const EdgeInsets.all(28),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: const Color(0xFF3472A4).withOpacity(0.15),
              shape: BoxShape.circle,
            ),
            child: const Icon(LucideIcons.check,
                size: 22, color: Color(0xFF3472A4)),
          ),
          const SizedBox(height: 16),
          const Text(
            'Feedback received',
            style: TextStyle(
              color: Colors.white,
              fontSize: 16,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Thank you for helping us improve Applad. We read every submission.',
            textAlign: TextAlign.center,
            style:
                TextStyle(color: Colors.white.withOpacity(0.5), fontSize: 13),
          ),
          const SizedBox(height: 20),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              style: FilledButton.styleFrom(
                backgroundColor: Colors.white.withOpacity(0.08),
                foregroundColor: Colors.white70,
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
              ),
              onPressed: widget.onClose,
              child: const Text('Close'),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildForm() {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Header
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      'Feedback',
                      style: TextStyle(
                        color: Colors.white,
                        fontSize: 16,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      'Applad evolves with your input. Share your thoughts and help us improve.',
                      style: TextStyle(
                          color: Colors.white.withOpacity(0.45), fontSize: 13),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 12),
              GestureDetector(
                onTap: widget.onClose,
                child: Icon(LucideIcons.x,
                    size: 16, color: Colors.white.withOpacity(0.3)),
              ),
            ],
          ),
        ),

        const SizedBox(height: 16),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20),
          child: Container(
              height: 1, color: Colors.white.withOpacity(0.06)),
        ),
        const SizedBox(height: 16),

        // Category chips
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Category',
                style: TextStyle(
                    color: Colors.white.withOpacity(0.5),
                    fontSize: 12,
                    fontWeight: FontWeight.w500),
              ),
              const SizedBox(height: 8),
              Row(
                children: _categories.map((cat) {
                  final selected = _category == cat;
                  return Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: GestureDetector(
                      onTap: () => setState(() => _category = cat),
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 6),
                        decoration: BoxDecoration(
                          color: selected
                              ? const Color(0xFF3472A4).withOpacity(0.15)
                              : Colors.white.withOpacity(0.05),
                          borderRadius: BorderRadius.circular(20),
                          border: Border.all(
                            color: selected
                                ? const Color(0xFF3472A4).withOpacity(0.6)
                                : Colors.white.withOpacity(0.1),
                          ),
                        ),
                        child: Text(
                          cat,
                          style: TextStyle(
                            color: selected
                                ? const Color(0xFF3472A4)
                                : Colors.white.withOpacity(0.5),
                            fontSize: 12,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ),
                    ),
                  );
                }).toList(),
              ),
            ],
          ),
        ),

        const SizedBox(height: 16),

        // Text area
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Tell us more about your experience',
                style: TextStyle(
                    color: Colors.white.withOpacity(0.5),
                    fontSize: 12,
                    fontWeight: FontWeight.w500),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: _ctrl,
                minLines: 4,
                maxLines: 6,
                style: const TextStyle(color: Colors.white, fontSize: 13),
                decoration: InputDecoration(
                  hintText: 'Share your suggestions and feature requests...',
                  hintStyle: TextStyle(
                      color: Colors.white.withOpacity(0.22), fontSize: 13),
                  filled: true,
                  fillColor: Colors.white.withOpacity(0.04),
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
                    borderSide:
                        const BorderSide(color: Color(0xFF3472A4)),
                  ),
                  contentPadding: const EdgeInsets.all(12),
                ),
              ),
            ],
          ),
        ),

        const SizedBox(height: 16),

        // Actions
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              TextButton(
                onPressed: widget.onClose,
                style: TextButton.styleFrom(
                  foregroundColor: Colors.white54,
                  padding: const EdgeInsets.symmetric(
                      horizontal: 16, vertical: 10),
                ),
                child: const Text('Cancel', style: TextStyle(fontSize: 13)),
              ),
              const SizedBox(width: 8),
              FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: const Color(0xFF3472A4),
                  padding: const EdgeInsets.symmetric(
                      horizontal: 20, vertical: 10),
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8)),
                ),
                onPressed: _loading ? null : _submit,
                child: _loading
                    ? const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(
                            strokeWidth: 2, color: Colors.white),
                      )
                    : const Text('Submit',
                        style: TextStyle(fontSize: 13)),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

// ── Support button ────────────────────────────────────────────────────────────

class SupportButton extends StatefulWidget {
  const SupportButton({super.key});

  @override
  State<SupportButton> createState() => _SupportButtonState();
}

class _SupportButtonState extends State<SupportButton> {
  OverlayEntry? _overlay;

  bool get _open => _overlay != null;

  void _close() {
    _overlay?.remove();
    _overlay = null;
    if (mounted) setState(() {});
  }

  void _toggle() {
    if (_open) {
      _close();
      return;
    }
    _overlay = _buildOverlay(
      top: 56,
      right: 8,
      onClose: _close,
      panel: const _SupportPanel(),
    );
    Overlay.of(context).insert(_overlay!);
    setState(() {});
  }

  @override
  void dispose() {
    _overlay?.remove();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return NavGhostButton(
      label: 'Support',
      active: _open,
      onTap: _toggle,
    );
  }
}

// ── Support panel ────────────────────────────────────────────────────────────

class _SupportPanel extends StatelessWidget {
  const _SupportPanel();

  void _openUrl(String url) =>
      html.window.open(url, '_blank');

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 360,
      decoration: BoxDecoration(
        color: const Color(0xFF16171B),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.white.withOpacity(0.08)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.5),
            blurRadius: 32,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Title
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 20, 20, 16),
            child: const Text(
              'Support',
              style: TextStyle(
                color: Colors.white,
                fontSize: 16,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),

          // Community support card
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: _SupportCard(
              title: 'Community support',
              subtitle: 'Get help from our community on Discord',
              buttonLabel: 'Discord',
              buttonIcon: LucideIcons.messageCircle,
              onTap: () => _openUrl('https://discord.gg/applad'),
            ),
          ),

          const SizedBox(height: 8),

          // GitHub issues card
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: _SupportCard(
              title: 'Open GitHub issue',
              subtitle: 'Report a bug or pitch a new feature',
              buttonLabel: 'Open issue',
              buttonIcon: LucideIcons.github,
              onTap: () => _openUrl(
                  'https://github.com/mittolabs/applad/issues/new'),
            ),
          ),

          const SizedBox(height: 8),

          // Documentation card
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: _SupportCard(
              title: 'Documentation',
              subtitle: 'Browse API references and guides',
              buttonLabel: 'View docs',
              buttonIcon: LucideIcons.bookOpen,
              onTap: () =>
                  _openUrl('https://github.com/mittolabs/applad#readme'),
            ),
          ),

          const SizedBox(height: 12),
        ],
      ),
    );
  }
}

class _SupportCard extends StatefulWidget {
  final String title;
  final String subtitle;
  final String buttonLabel;
  final IconData buttonIcon;
  final VoidCallback onTap;

  const _SupportCard({
    required this.title,
    required this.subtitle,
    required this.buttonLabel,
    required this.buttonIcon,
    required this.onTap,
  });

  @override
  State<_SupportCard> createState() => _SupportCardState();
}

class _SupportCardState extends State<_SupportCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.03),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: Colors.white.withOpacity(0.06)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            widget.title,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 14,
              fontWeight: FontWeight.w500,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            widget.subtitle,
            style: TextStyle(
                color: Colors.white.withOpacity(0.4), fontSize: 12),
          ),
          const SizedBox(height: 12),
          MouseRegion(
            onEnter: (_) => setState(() => _hovered = true),
            onExit: (_) => setState(() => _hovered = false),
            child: GestureDetector(
              onTap: widget.onTap,
              child: Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(
                    horizontal: 14, vertical: 9),
                decoration: BoxDecoration(
                  color: _hovered
                      ? Colors.white.withOpacity(0.1)
                      : Colors.white.withOpacity(0.06),
                  borderRadius: BorderRadius.circular(8),
                  border:
                      Border.all(color: Colors.white.withOpacity(0.08)),
                ),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(widget.buttonIcon,
                        size: 14,
                        color: Colors.white.withOpacity(0.65)),
                    const SizedBox(width: 8),
                    Text(
                      widget.buttonLabel,
                      style: TextStyle(
                        color: Colors.white.withOpacity(0.75),
                        fontSize: 13,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ── Theme toggle ─────────────────────────────────────────────────────────────

// ── User avatar + dropdown menu ──────────────────────────────────────────────

class UserMenuButton extends ConsumerStatefulWidget {
  const UserMenuButton({super.key});

  @override
  ConsumerState<UserMenuButton> createState() => _UserMenuButtonState();
}

class _UserMenuButtonState extends ConsumerState<UserMenuButton> {
  OverlayEntry? _overlay;

  void _open(BuildContext context) {
    if (_overlay != null) {
      _close();
      return;
    }
    final box = context.findRenderObject() as RenderBox;
    final pos = box.localToGlobal(Offset.zero);
    final screenW = MediaQuery.of(context).size.width;
    final right = screenW - pos.dx - box.size.width;

    _overlay = _buildOverlay(
      top: pos.dy + box.size.height + 6,
      right: right,
      onClose: _close,
      panel: _UserMenuPanel(onClose: _close),
    );
    Overlay.of(context).insert(_overlay!);
    setState(() {});
  }

  void _close() {
    _overlay?.remove();
    _overlay = null;
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _overlay?.remove();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final user = ref.watch(consoleAuthProvider).valueOrNull;
    if (user == null) return const SizedBox.shrink();

    final initials = _userInitials(user.name, user.email);
    final isOpen = _overlay != null;

    return GestureDetector(
      onTap: () => _open(context),
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          width: 32,
          height: 32,
          decoration: BoxDecoration(
            color: isOpen
                ? const Color(0xFF3472A4)
                : const Color(0xFF3472A4).withOpacity(0.85),
            shape: BoxShape.circle,
            border: isOpen
                ? Border.all(color: Colors.white.withOpacity(0.2), width: 2)
                : null,
          ),
          child: Center(
            child: Text(
              initials,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 12,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

String _userInitials(String name, String email) {
  if (name.trim().isNotEmpty) {
    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.length >= 2) return '${parts[0][0]}${parts[1][0]}'.toUpperCase();
    return parts[0][0].toUpperCase();
  }
  return email.isNotEmpty ? email[0].toUpperCase() : '?';
}

class _UserMenuPanel extends ConsumerWidget {
  final VoidCallback onClose;
  const _UserMenuPanel({required this.onClose});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(consoleAuthProvider).valueOrNull;
    final isLight = ref.watch(themeModeProvider);

    return Container(
      width: 280,
      decoration: BoxDecoration(
        color: _popoverSurface(context),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: _popoverBorder(context)),
        boxShadow: [
          BoxShadow(
            color: _popoverShadow(context),
            blurRadius: 20,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Email
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 16, 20, 14),
            child: Text(
              user?.email ?? '',
              style: TextStyle(
                color: _primaryText(context),
                fontSize: 14,
                fontWeight: FontWeight.w500,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),

          _divider(context),

          // Account
          _MenuItemTrailing(
            label: 'Account',
            icon: LucideIcons.user,
            onTap: () {
              onClose();
              context.go('/account');
            },
          ),

          _divider(context),

          // Sign out
          _MenuItemTrailing(
            label: 'Sign out',
            icon: LucideIcons.logOut,
            onTap: () {
              onClose();
              ref.read(consoleAuthProvider.notifier).logout();
              context.go('/login');
            },
          ),

          _divider(context),

          // Theme row
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
            child: Row(
              children: [
                Text(
                  'Theme',
                  style: TextStyle(color: _secondaryText(context), fontSize: 14),
                ),
                const Spacer(),
                _ThemeToggle(isLight: isLight),
              ],
            ),
          ),

          const SizedBox(height: 2),
        ],
      ),
    );
  }

    Widget _divider(BuildContext context) => Container(
        height: 1,
      color: _dividerColor(context),
      );
}

class _MenuItemTrailing extends StatefulWidget {
  final String label;
  final IconData icon;
  final VoidCallback onTap;
  const _MenuItemTrailing({required this.label, required this.icon, required this.onTap});

  @override
  State<_MenuItemTrailing> createState() => _MenuItemTrailingState();
}

class _MenuItemTrailingState extends State<_MenuItemTrailing> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          width: double.infinity,
          color: _hovered
              ? _hoverFill(context)
              : Colors.transparent,
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
          child: Row(
            children: [
              Text(widget.label,
                  style: TextStyle(
                      color: _secondaryText(context), fontSize: 14)),
              const Spacer(),
              Icon(widget.icon, size: 16, color: _mutedText(context)),
            ],
          ),
        ),
      ),
    );
  }
}

class _ThemeToggle extends ConsumerWidget {
  final bool isLight;
  const _ThemeToggle({required this.isLight});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Container(
      decoration: BoxDecoration(
        color: _subtleFill(context),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          _ThemeOption(
            icon: LucideIcons.sun,
            active: isLight,
            tooltip: 'Light',
            onTap: () => ref.read(themeModeProvider.notifier).setLight(),
          ),
          _ThemeOption(
            icon: LucideIcons.moon,
            active: !isLight,
            tooltip: 'Dark',
            onTap: () => ref.read(themeModeProvider.notifier).setDark(),
          ),
          _ThemeOption(
            icon: LucideIcons.contrast,
            active: false,
            tooltip: 'System',
            onTap: () => ref.read(themeModeProvider.notifier).setDark(),
          ),
        ],
      ),
    );
  }
}

class _ThemeOption extends StatefulWidget {
  final IconData icon;
  final bool active;
  final String tooltip;
  final VoidCallback onTap;
  const _ThemeOption({required this.icon, required this.active, required this.tooltip, required this.onTap});

  @override
  State<_ThemeOption> createState() => _ThemeOptionState();
}

class _ThemeOptionState extends State<_ThemeOption> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: widget.tooltip,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        onEnter: (_) => setState(() => _hovered = true),
        onExit: (_) => setState(() => _hovered = false),
        child: GestureDetector(
          onTap: widget.onTap,
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 150),
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
            decoration: BoxDecoration(
              color: widget.active
                  ? (_isLight(context)
                      ? const Color(0xFF3472A4).withOpacity(0.12)
                      : Colors.white.withOpacity(0.12))
                  : _hovered
                      ? _subtleFill(context)
                      : Colors.transparent,
              borderRadius: BorderRadius.circular(5),
            ),
            child: Icon(
              widget.icon,
              size: 13,
              color: widget.active
                  ? (_isLight(context)
                      ? const Color(0xFF3472A4)
                      : Colors.white)
                  : (_isLight(context)
                      ? const Color(0xFF1A1A2E).withOpacity(0.45)
                      : Colors.white.withOpacity(0.4)),
            ),
          ),
        ),
      ),
    );
  }
}

// ── Theme toggle (standalone navbar button) ──────────────────────────────────

class ThemeToggleButton extends ConsumerWidget {
  const ThemeToggleButton({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isLight = ref.watch(themeModeProvider);
    return Tooltip(
      message: isLight ? 'Switch to dark mode' : 'Switch to light mode',
      child: InkWell(
        onTap: () => ref.read(themeModeProvider.notifier).toggle(),
        borderRadius: BorderRadius.circular(8),
        child: SizedBox(
          width: 34,
          height: 34,
          child: Icon(
            isLight ? LucideIcons.moon : LucideIcons.sun,
            size: 17,
            color: isLight
                ? Colors.black.withOpacity(0.45)
                : Colors.white.withOpacity(0.45),
          ),
        ),
      ),
    );
  }
}
