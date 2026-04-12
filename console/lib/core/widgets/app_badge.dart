import 'package:flutter/material.dart';

/// A compact, self-sizing label badge — for categories, types, and tags.
///
/// Unlike [StatusChip] (which carries semantic colour + a dot indicator),
/// [AppBadge] is neutral by default and accepts an optional [icon].
///
/// Usage:
///   AppBadge(label: 'Android')
///   AppBadge(label: 'Web', icon: LucideIcons.globe)
///   AppBadge(label: 'iOS', color: Color(0xFF6C47FF))
class AppBadge extends StatelessWidget {
  final String label;
  final IconData? icon;

  /// Foreground (text + icon) colour. Defaults to the theme's subtle text.
  final Color? color;

  /// Background colour. Defaults to a low-opacity tint of [color] (or grey).
  final Color? backgroundColor;

  const AppBadge({
    super.key,
    required this.label,
    this.icon,
    this.color,
    this.backgroundColor,
  });

  @override
  Widget build(BuildContext context) {
    final brightness = Theme.of(context).brightness;
    final isDark = brightness == Brightness.dark;

    final fg = color ??
        (isDark ? const Color(0xFF9CA3AF) : const Color(0xFF6B7280));
    final bg = backgroundColor ??
        fg.withValues(alpha: isDark ? 0.12 : 0.10);

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
          decoration: BoxDecoration(
            color: bg,
            borderRadius: BorderRadius.circular(4),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (icon != null) ...[
                Icon(icon, size: 10, color: fg),
                const SizedBox(width: 4),
              ],
              Text(
                label,
                style: TextStyle(
                  color: fg,
                  fontSize: 11,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}
