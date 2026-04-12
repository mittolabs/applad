import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';

class NotFoundPage extends StatelessWidget {
  final String? path;

  const NotFoundPage({super.key, this.path});

  @override
  Widget build(BuildContext context) {
    const accent = Color(0xFF6C47FF);
    const bg = Color(0xFF0B0B0F);
    const surface = Color(0xFF16171B);
    const border = Color(0xFF1F2025);
    const textPrimary = Color(0xFFEEEEEE);
    const textMuted = Color(0xFF6B7280);

    return Scaffold(
      backgroundColor: bg,
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Icon container
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                color: surface,
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: border),
              ),
              child: const Icon(
                LucideIcons.fileQuestion,
                size: 32,
                color: textMuted,
              ),
            ),
            const SizedBox(height: 24),

            // 404
            const Text(
              '404',
              style: TextStyle(
                color: accent,
                fontSize: 13,
                fontWeight: FontWeight.w600,
                letterSpacing: 1.5,
              ),
            ),
            const SizedBox(height: 8),

            // Title
            const Text(
              'Page not found',
              style: TextStyle(
                color: textPrimary,
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
              style: const TextStyle(
                color: textMuted,
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
                _FilledButton(
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

class _FilledButton extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _FilledButton({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return FilledButton.icon(
      style: FilledButton.styleFrom(
        backgroundColor: const Color(0xFF6C47FF),
        foregroundColor: Colors.white,
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

class _OutlineButton extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _OutlineButton({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return OutlinedButton.icon(
      style: OutlinedButton.styleFrom(
        foregroundColor: const Color(0xFFAAAAAA),
        side: const BorderSide(color: Color(0xFF1F2025)),
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
