// ignore: avoid_web_libraries_in_flutter, deprecated_member_use
import 'dart:html' as html;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../providers/auth_provider.dart';
import '../providers/theme_provider.dart';
import '../theme/console_colors.dart';
import 'app_dialog.dart';

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
    final cs = consoleColors(context);
    final highlight = _hovered || widget.active;
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          decoration: BoxDecoration(
            color: highlight ? cs.fill : Colors.transparent,
            borderRadius: BorderRadius.circular(7),
          ),
          child: Text(
            widget.label,
            style: TextStyle(
              color: highlight ? cs.textSecondary : cs.textMuted,
              fontSize: 13,
            ),
          ),
        ),
      ),
    );
  }
}

/// Opens the feedback panel from anywhere (e.g. the overflow nav menu).
void showFeedbackPanel(BuildContext context) {
  OverlayEntry? entry;
  void close() {
    entry?.remove();
    entry = null;
  }
  entry = _buildOverlay(
    top: 56,
    right: 8,
    onClose: close,
    panel: _FeedbackPanel(onClose: close),
  );
  Overlay.of(context).insert(entry!);
}

/// Opens the support panel from anywhere (e.g. the overflow nav menu).
void showSupportPanel(BuildContext context) {
  OverlayEntry? entry;
  void close() {
    entry?.remove();
    entry = null;
  }
  entry = _buildOverlay(
    top: 56,
    right: 8,
    onClose: close,
    panel: const _SupportPanel(),
  );
  Overlay.of(context).insert(entry!);
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
  late ConsoleColors _cs;
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
    if (mounted) {
      setState(() {
        _loading = false;
        _submitted = true;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    _cs = consoleColors(context);
    return Container(
      width: 400,
      decoration: BoxDecoration(
        color: _cs.popupSurface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: _cs.border),
        boxShadow: [
          BoxShadow(
            color: _cs.shadow,
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
              color: const Color(0xFF3472A4).withValues(alpha: 0.15),
              shape: BoxShape.circle,
            ),
            child: const Icon(LucideIcons.check,
                size: 22, color: Color(0xFF3472A4)),
          ),
          const SizedBox(height: 16),
          Text(
            'Feedback received',
            style: TextStyle(
              color: _cs.textPrimary,
              fontSize: 16,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Thank you for helping us improve Applad. We read every submission.',
            textAlign: TextAlign.center,
            style: TextStyle(color: _cs.textMuted, fontSize: 13),
          ),
          const SizedBox(height: 20),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              style: FilledButton.styleFrom(
                backgroundColor: _cs.fill,
                foregroundColor: _cs.textSecondary,
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
                    Text(
                      'Feedback',
                      style: TextStyle(
                        color: _cs.textPrimary,
                        fontSize: 16,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      'Applad evolves with your input. Share your thoughts and help us improve.',
                      style: TextStyle(
                          color: _cs.textMuted, fontSize: 13),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 12),
              GestureDetector(
                onTap: widget.onClose,
                child: Icon(LucideIcons.x,
                    size: 16, color: _cs.textSubtle),
              ),
            ],
          ),
        ),

        const SizedBox(height: 16),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20),
          child: Container(height: 1, color: _cs.border),
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
                    color: _cs.textMuted,
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
                              ? const Color(0xFF3472A4).withValues(alpha: 0.15)
                              : _cs.fill,
                          borderRadius: BorderRadius.circular(20),
                          border: Border.all(
                            color: selected
                                ? const Color(0xFF3472A4).withValues(alpha: 0.6)
                                : _cs.border,
                          ),
                        ),
                        child: Text(
                          cat,
                          style: TextStyle(
                            color: selected
                                ? const Color(0xFF3472A4)
                                : _cs.textMuted,
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
                    color: _cs.textMuted,
                    fontSize: 12,
                    fontWeight: FontWeight.w500),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: _ctrl,
                minLines: 4,
                maxLines: 6,
                style: TextStyle(color: _cs.textPrimary, fontSize: 13),
                decoration: InputDecoration(
                  hintText: 'Share your suggestions and feature requests...',
                  hintStyle: TextStyle(
                      color: _cs.textSubtle, fontSize: 13),
                  filled: true,
                  fillColor: _cs.fieldFill,
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
                  foregroundColor: _cs.textMuted,
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
    final cs = consoleColors(context);
    return Container(
      width: 360,
      decoration: BoxDecoration(
        color: cs.popupSurface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: cs.border),
        boxShadow: [
          BoxShadow(
            color: cs.shadow,
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
            child: Text(
              'Support',
              style: TextStyle(
                color: cs.textPrimary,
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
              buttonIcon: LucideIcons.gitBranch,
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
    final cs = consoleColors(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.fill,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: cs.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            widget.title,
            style: TextStyle(
              color: cs.textPrimary,
              fontSize: 14,
              fontWeight: FontWeight.w500,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            widget.subtitle,
            style: TextStyle(color: cs.textMuted, fontSize: 12),
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
                  color: _hovered ? cs.fillHover : cs.fill,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: cs.border),
                ),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(widget.buttonIcon,
                        size: 14,
                        color: cs.textSecondary),
                    const SizedBox(width: 8),
                    Text(
                      widget.buttonLabel,
                      style: TextStyle(
                        color: cs.textSecondary,
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
                : const Color(0xFF3472A4).withValues(alpha: 0.85),
            shape: BoxShape.circle,
            border: isOpen
                ? Border.all(color: Colors.white.withValues(alpha: 0.2), width: 2)
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
    final cs = consoleColors(context);
    final user = ref.watch(consoleAuthProvider).valueOrNull;
    final isLight = ref.watch(themeModeProvider);

    return Container(
      width: 230,
      decoration: BoxDecoration(
        color: cs.popupSurface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: cs.border),
        boxShadow: [
          BoxShadow(
            color: cs.shadow,
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
            padding: const EdgeInsets.fromLTRB(12, 12, 12, 6),
            child: Text(
              user?.email ?? '',
              style: TextStyle(
                color: cs.textMuted,
                fontSize: 11,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),

          // Items
          Padding(
            padding: const EdgeInsets.fromLTRB(4, 0, 4, 4),
            child: Column(
              children: [
                _MenuItemTrailing(
                  label: 'Account',
                  icon: LucideIcons.user,
                  onTap: () {
                    onClose();
                    context.go('/account');
                  },
                ),
                _MenuItemTrailing(
                  label: 'Sign out',
                  icon: LucideIcons.logOut,
                  onTap: () async {
                    // Capture notifier before the dialog opens — the popover
                    // widget may be disposed by the time the confirm button is
                    // tapped, making ref invalid.
                    final notifier = ref.read(consoleAuthProvider.notifier);
                    bool signingOut = false;
                    await showAppDialog<void>(
                      context: context,
                      title: 'Sign out',
                      content: StatefulBuilder(
                        builder: (ctx, ss) => Text(
                          'Are you sure you want to sign out?',
                          style: TextStyle(
                              color: consoleColors(ctx).textSecondary),
                        ),
                      ),
                      actions: [
                        StatefulBuilder(
                          builder: (ctx, ss) => Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              const AppDialogCancel(),
                              AppDialogAction(
                                label: 'Sign out',
                                destructive: true,
                                loading: signingOut,
                                onTap: signingOut
                                    ? null
                                    : () async {
                                        ss(() => signingOut = true);
                                        await notifier.logout();
                                        if (ctx.mounted) {
                                          Navigator.of(ctx,
                                                  rootNavigator: true)
                                              .pop();
                                        }
                                      },
                              ),
                            ],
                          ),
                        ),
                      ],
                    );
                  },
                ),
              ],
            ),
          ),

          Container(height: 1, color: cs.border, margin: const EdgeInsets.symmetric(horizontal: 8)),

          // Theme row
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 10),
            child: Row(
              children: [
                Text(
                  'Theme',
                  style: TextStyle(color: cs.textMuted, fontSize: 11),
                ),
                const Spacer(),
                _ThemeToggle(isLight: isLight),
              ],
            ),
          ),
        ],
      ),
    );
  }
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
    final cs = consoleColors(context);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          width: double.infinity,
          decoration: BoxDecoration(
            color: _hovered ? cs.fillHover : Colors.transparent,
            borderRadius: BorderRadius.circular(6),
          ),
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 7),
          child: Row(
            children: [
              Text(widget.label,
                  style: TextStyle(color: cs.textSecondary, fontSize: 12)),
              const Spacer(),
              Icon(widget.icon, size: 13, color: cs.textMuted),
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
    final cs = consoleColors(context);
    return Container(
      decoration: BoxDecoration(
        color: cs.fill,
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
    final cs = consoleColors(context);
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
                  ? cs.fillActive
                  : _hovered
                      ? cs.fill
                      : Colors.transparent,
              borderRadius: BorderRadius.circular(5),
            ),
            child: Icon(
              widget.icon,
              size: 13,
              color: widget.active
                  ? const Color(0xFF3472A4)
                  : cs.textMuted,
            ),
          ),
        ),
      ),
    );
  }
}

