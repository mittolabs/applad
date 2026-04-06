// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html;

import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';

const _version = '1.0.0';

class ConsoleFooter extends StatelessWidget {
  const ConsoleFooter({super.key});

  void _open(String url) => html.window.open(url, '_blank');

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 44,
      padding: const EdgeInsets.symmetric(horizontal: 20),
      decoration: BoxDecoration(
        color: const Color(0xFF0B0B0F),
        border: Border(
          top: BorderSide(color: Colors.white.withOpacity(0.06)),
        ),
      ),
      child: Row(
        children: [
          // Copyright
          Text(
            '© ${DateTime.now().year} Applad. All rights reserved.',
            style: TextStyle(
              color: Colors.white.withOpacity(0.25),
              fontSize: 12,
            ),
          ),

          // Vertical separator
          Container(
            height: 14,
            width: 1,
            margin: const EdgeInsets.symmetric(horizontal: 12),
            color: Colors.white.withOpacity(0.1),
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
              color: Colors.white.withOpacity(0.25),
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
            color: Colors.white
                .withOpacity(_hovered ? 0.5 : 0.25),
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
            color: Colors.white.withOpacity(_hovered ? 0.6 : 0.3),
            fontSize: 12,
          ),
        ),
      ),
    );
  }
}
