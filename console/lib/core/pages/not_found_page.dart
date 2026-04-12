import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../theme/console_colors.dart';

class NotFoundPage extends StatelessWidget {
  final String? path;

  const NotFoundPage({super.key, this.path});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);

    return Scaffold(
      backgroundColor: colors.background,
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Icon container
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                color: colors.surface,
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: colors.border),
              ),
              child: Icon(
                LucideIcons.fileQuestion,
                size: 32,
                color: colors.textMuted,
              ),
            ),
            const SizedBox(height: 24),

            // 404 label
            Text(
              '404',
              style: TextStyle(
                color: colors.textMuted,
                fontSize: 12,
                fontWeight: FontWeight.w600,
                letterSpacing: 1.5,
              ),
            ),
            const SizedBox(height: 8),

            // Title
            Text(
              'Page not found',
              style: TextStyle(
                color: colors.textPrimary,
                fontSize: 22,
                fontWeight: FontWeight.w600,
                letterSpacing: -0.3,
              ),
            ),
            const SizedBox(height: 8),

            // Subtitle
            Text(
              path != null
                  ? 'The path "$path" doesn\'t exist.'
                  : 'This page doesn\'t exist or has been moved.',
              style: TextStyle(
                color: colors.textMuted,
                fontSize: 14,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 32),

            // Actions
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                _OutlineButton(
                  colors: colors,
                  icon: LucideIcons.arrowLeft,
                  label: 'Go back',
                  onTap: () {
                    if (context.canPop()) {
                      context.pop();
                    } else {
                      context.go('/projects');
                    }
                  },
                ),
                const SizedBox(width: 10),
                _OutlineButton(
                  colors: colors,
                  icon: LucideIcons.layoutGrid,
                  label: 'Projects',
                  onTap: () => context.go('/projects'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _OutlineButton extends StatelessWidget {
  final ConsoleColors colors;
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _OutlineButton({
    required this.colors,
    required this.icon,
    required this.label,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return OutlinedButton.icon(
      style: OutlinedButton.styleFrom(
        foregroundColor: colors.textSecondary,
        side: BorderSide(color: colors.border),
        backgroundColor: colors.fill,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
        textStyle: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
      ),
      icon: Icon(icon, size: 15),
      label: Text(label),
      onPressed: onTap,
    );
  }
}
