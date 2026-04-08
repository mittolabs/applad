import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../providers/org_provider.dart';
import '../providers/project_provider.dart';
import 'navbar_popovers.dart';

const _bg = Color(0xFF0B0B0F);

/// Shared top navigation bar used across all pages (project shell, account, projects).
class AppNavBar extends ConsumerWidget {
  /// If null, only org breadcrumb is shown (no project segment).
  final String? projectId;

  const AppNavBar({super.key, this.projectId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final orgs = ref.watch(orgsProvider).valueOrNull ?? [];
    final currentOrgId = ref.watch(currentOrgProvider);
    final projects = ref.watch(projectsProvider).valueOrNull ?? [];

    String orgName = '';
    if (currentOrgId != null) {
      final org = orgs.where((o) => o['\$id'] == currentOrgId).firstOrNull;
      if (org != null) orgName = org['name'] as String;
    } else if (orgs.isNotEmpty) {
      orgName = orgs.first['name'] as String;
    }

    String? projectName;
    if (projectId != null) {
      final proj =
          projects.where((p) => p['\$id'] == projectId).firstOrNull;
      projectName = proj?['name'] as String?;
    }

    return Container(
      height: 52,
      decoration: BoxDecoration(
        color: _bg,
        border: Border(
            bottom: BorderSide(color: Colors.white.withOpacity(0.06))),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          // Logo
          GestureDetector(
            onTap: () => context.go('/projects'),
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: ClipRRect(
                borderRadius: BorderRadius.circular(6),
                child: Image.asset(
                  'assets/icon.png',
                  width: 42,
                  height: 42,
                  fit: BoxFit.cover,
                ),
              ),
            ),
          ),

          const SizedBox(width: 10),
          _sep(),
          const SizedBox(width: 10),

          // Org name
          GestureDetector(
            onTap: () => context.go('/projects'),
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    orgName.isNotEmpty ? orgName : 'Organization',
                    style: TextStyle(
                      color: Colors.white.withOpacity(0.7),
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  const SizedBox(width: 4),
                  Icon(LucideIcons.chevronDown,
                      size: 14, color: Colors.white.withOpacity(0.3)),
                ],
              ),
            ),
          ),

          // Project segment
          if (projectId != null) ...[
            const SizedBox(width: 10),
            _sep(),
            const SizedBox(width: 10),
            GestureDetector(
              onTap: () => context.go('/project/$projectId/overview'),
              child: MouseRegion(
                cursor: SystemMouseCursors.click,
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      projectName ?? _short(projectId!),
                      style: TextStyle(
                        color: Colors.white.withOpacity(0.7),
                        fontSize: 14,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(width: 4),
                    Icon(LucideIcons.chevronDown,
                        size: 14, color: Colors.white.withOpacity(0.3)),
                  ],
                ),
              ),
            ),
          ],

          const Spacer(),

          // Right side buttons
          const FeedbackButton(),
          const SizedBox(width: 2),
          const SupportButton(),
          const SizedBox(width: 2),
          const ThemeToggleButton(),
          const SizedBox(width: 4),

          // Search
          Tooltip(
            message: '⌘K',
            child: InkWell(
              onTap: () {},
              borderRadius: BorderRadius.circular(8),
              child: SizedBox(
                width: 34,
                height: 34,
                child: Icon(LucideIcons.search,
                    size: 17, color: Colors.white.withOpacity(0.45)),
              ),
            ),
          ),

          const SizedBox(width: 8),
          const UserMenuButton(),
          const SizedBox(width: 4),
        ],
      ),
    );
  }

  Widget _sep() => Text(
        '/',
        style: TextStyle(
          color: Colors.white.withOpacity(0.18),
          fontSize: 18,
          fontWeight: FontWeight.w300,
        ),
      );

  String _short(String id) => id.length > 8 ? id.substring(0, 8) : id;
}
