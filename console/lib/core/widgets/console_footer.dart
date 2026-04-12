// ignore: avoid_web_libraries_in_flutter, deprecated_member_use
import 'dart:html' as html;

import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';

const _version = '1.0.0';

bool _isLight(BuildContext context) =>
  Theme.of(context).brightness == Brightness.light;

Color _footerBg(BuildContext context) => Theme.of(context).scaffoldBackgroundColor;

Color _footerBorder(BuildContext context) => _isLight(context)
  ? Colors.black.withValues(alpha: 0.08)
  : Colors.white.withValues(alpha: 0.06);

Color _footerText(BuildContext context, double alpha) => _isLight(context)
  ? const Color(0xFF1A1A2E).withValues(alpha: alpha)
  : Colors.white.withValues(alpha: alpha);

class ConsoleFooter extends StatelessWidget {
  const ConsoleFooter({super.key});

  void _open(String url) => html.window.open(url, '_blank');

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 44,
      padding: const EdgeInsets.symmetric(horizontal: 20),
      decoration: BoxDecoration(
        color: _footerBg(context),
        border: Border(
          top: BorderSide(color: _footerBorder(context)),
        ),
      ),
      child: Row(
        children: [
          // Copyright
          Text(
            '© ${DateTime.now().year} Mittolabs LTD. All rights reserved.',
            style: TextStyle(
              color: _footerText(context, 0.32),
              fontSize: 12,
            ),
          ),

          // Vertical separator
          Container(
            height: 14,
            width: 1,
            margin: const EdgeInsets.symmetric(horizontal: 12),
            color: _footerBorder(context),
          ),

          // GitHub icon
          _FooterIconButton(
            icon: LucideIcons.github,
            tooltip: 'GitHub',
            onTap: () => _open('https://github.com/mittolabs/applad'),
          ),

          const SizedBox(width: 8),

          // Discord icon
          _FooterIconButton(
            icon: LucideIcons.messageCircle,
            tooltip: 'Discord',
            onTap: () => _open('https://discord.gg/applad'),
          ),

          const Spacer(),

          // Version
          Text(
            'Version $_version',
            style: TextStyle(
              color: _footerText(context, 0.32),
              fontSize: 12,
            ),
          ),

          const SizedBox(width: 16),

          _FooterTextLink(
            label: 'Docs',
            onTap: () =>
                _open('https://github.com/mittolabs/applad#readme'),
          ),

          const SizedBox(width: 14),

          _FooterTextLink(
            label: 'Terms',
            onTap: () =>
                _open('https://github.com/mittolabs/applad'),
          ),

          const SizedBox(width: 14),

          _FooterTextLink(
            label: 'Privacy',
            onTap: () =>
                _open('https://github.com/mittolabs/applad'),
          ),
        ],
      ),
    );
  }
}

class _FooterIconButton extends StatefulWidget {
  final IconData icon;
  final String tooltip;
  final VoidCallback onTap;

  const _FooterIconButton({
    required this.icon,
    required this.tooltip,
    required this.onTap,
  });

  @override
  State<_FooterIconButton> createState() => _FooterIconButtonState();
}

class _FooterIconButtonState extends State<_FooterIconButton> {
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
          child: Icon(
            widget.icon,
            size: 15,
            color: _footerText(context, _hovered ? 0.6 : 0.32),
          ),
        ),
      ),
    );
  }
}

class _FooterTextLink extends StatefulWidget {
  final String label;
  final VoidCallback onTap;

  const _FooterTextLink({required this.label, required this.onTap});

  @override
  State<_FooterTextLink> createState() => _FooterTextLinkState();
}

class _FooterTextLinkState extends State<_FooterTextLink> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Text(
          widget.label,
          style: TextStyle(
            color: _footerText(context, _hovered ? 0.68 : 0.38),
            fontSize: 12,
          ),
        ),
      ),
    );
  }
}
