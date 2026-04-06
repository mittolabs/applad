// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../providers/theme_provider.dart';

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
                ? Colors.white.withOpacity(0.06)
                : Colors.transparent,
            borderRadius: BorderRadius.circular(7),
          ),
          child: Text(
            widget.label,
            style: TextStyle(
              color: Colors.white.withOpacity(highlight ? 0.8 : 0.45),
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
