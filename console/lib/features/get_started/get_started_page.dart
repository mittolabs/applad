import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/providers/auth_provider.dart' show consoleAuthProvider;

// --- Constants ---------------------------------------------------------------

const _bgColor = Color(0xFF0B0B0F);
const _cardColor = Color(0xFF16171B);
const _accent = Color(0xFF3472A4);
const _dimText = Color(0x80FFFFFF);
const _subtleText = Color(0x40FFFFFF);
const _green = Color(0xFF10B981);

// --- Providers ---------------------------------------------------------------

final _getStartedDataProvider =
    FutureProvider.family<Map<String, int>, String>((ref, projectId) async {
  final api = ref.read(apiClientProvider);

  Future<int> count(String path, String key) async {
    try {
      final res = await api.get(path);
      final data = res.data;
      if (data is Map && data[key] is List) return (data[key] as List).length;
      if (data is Map && data['total'] is int) return data['total'] as int;
    } catch (_) {}
    return 0;
  }

  final results = await Future.wait([
    count('/databases', 'databases'),
    count('/account/users', 'users'),
    count('/storage/buckets', 'buckets'),
    count('/functions', 'functions'),
    count('/deploy/targets', 'targets'),
    count('/workflows', 'workflows'),
    count('/projects/$projectId/keys', 'keys'),
    count('/projects/$projectId/platforms', 'platforms'),
  ]);

  return {
    'databases': results[0],
    'users': results[1],
    'buckets': results[2],
    'functions': results[3],
    'deployments': results[4],
    'workflows': results[5],
    'apiKeys': results[6],
    'platforms': results[7],
  };
});

// --- Page --------------------------------------------------------------------

class GetStartedPage extends ConsumerWidget {
  const GetStartedPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final routerState = GoRouterState.of(context);
    final projectId = routerState.pathParameters['projectId'];
    final userName = ref.watch(consoleAuthProvider).valueOrNull?.name ?? '';

    if (projectId == null) {
      return const Scaffold(
        backgroundColor: _bgColor,
        body: Center(
            child: Text('No project selected',
                style: TextStyle(color: _dimText))),
      );
    }

    final dataAsync = ref.watch(_getStartedDataProvider(projectId));

    return Scaffold(
      backgroundColor: _bgColor,
      body: dataAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
            child: Text('Error: $e',
                style: const TextStyle(color: Colors.red))),
        data: (counts) {
          final steps = _buildSteps(projectId, counts);
          final completed = steps.where((s) => s.done).length;
          final progress = steps.isEmpty ? 0.0 : completed / steps.length;

          final currentStepIndex = steps.indexWhere((s) => !s.done);

          return SingleChildScrollView(
            padding: EdgeInsets.symmetric(
              horizontal:
                  MediaQuery.of(context).size.width > 1400 ? 80 : 40,
              vertical: 32,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Header
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            userName.isNotEmpty
                                ? 'Welcome, $userName'
                                : 'Welcome',
                            style: const TextStyle(
                                color: Colors.white,
                                fontSize: 24,
                                fontWeight: FontWeight.w700),
                          ),
                          const SizedBox(height: 4),
                          Text(
                              'Follow a few quick steps to get started with Applad',
                              style: TextStyle(
                                  color: Colors.white.withOpacity(0.4),
                                  fontSize: 14)),
                        ],
                      ),
                    ),
                    TextButton(
                      onPressed: () =>
                          context.go('/project/$projectId/overview'),
                      style: TextButton.styleFrom(
                          foregroundColor: Colors.white54),
                      child: const Text('Dismiss this page',
                          style: TextStyle(fontSize: 13)),
                    ),
                  ],
                ),
                const SizedBox(height: 32),

                // Timeline steps with inline content
                _buildTimeline(context, steps),
                const SizedBox(height: 8),

                // Contextual cards grid
                _buildContextCards(context, projectId, currentStepIndex),

                const SizedBox(height: 40),
              ],
            ),
          );
        },
      ),
    );
  }

  List<_Step> _buildSteps(String projectId, Map<String, int> counts) {
    // Three realistic steps like Appwrite — not everything is required
    final hasAnyResource = (counts['databases'] ?? 0) > 0 ||
        (counts['buckets'] ?? 0) > 0 ||
        (counts['functions'] ?? 0) > 0 ||
        (counts['deployments'] ?? 0) > 0 ||
        (counts['workflows'] ?? 0) > 0;

    return [
      _Step(
        title: 'Create project',
        done: true, // Always done — they're on this page
        route: '/project/$projectId/overview',
      ),
      _Step(
        title: 'Connect your platform',
        subtitle: 'Register web, iOS, or Android apps',
        done: (counts['platforms'] ?? 0) > 0,
        route: '/project/$projectId/settings',
      ),
      _Step(
        title: 'Build your app',
        subtitle: 'Set up services like Auth, Databases, Storage and Functions',
        done: hasAnyResource,
        route: '/project/$projectId/databases',
      ),
    ];
  }

  Widget _buildTimeline(BuildContext context, List<_Step> steps) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: List.generate(steps.length, (i) {
        final step = steps[i];
        final isLast = i == steps.length - 1;
        final isCurrent =
            !step.done && (i == 0 || steps[i - 1].done);

        return _TimelineStep(
          step: step,
          isLast: isLast,
          isCurrent: isCurrent,
          onTap: () => context.go(step.route),
        );
      }),
    );
  }

  Widget _buildContextCards(
      BuildContext context, String projectId, int currentStepIndex) {
    final cards = <Widget>[];

    if (currentStepIndex == 1) {
      // Connect platform step
      cards.addAll([
        _FeatureCard(
          icon: LucideIcons.smartphone,
          title: 'Connect your platform',
          description:
              'Register your app to allow API access from web, iOS, Android, or Flutter.',
          onTap: () => context.go('/project/$projectId/settings'),
          actionLabel: 'Go to Settings',
        ),
        _FeatureCard(
          icon: LucideIcons.bookOpen,
          title: 'Discover our docs',
          description:
              'API references, tutorials, quick start guides, and more.',
          links: const [
            'API references',
            'Tutorials',
            'Storage quick start',
            'Functions quick start',
          ],
        ),
      ]);
    } else if (currentStepIndex == 2 || currentStepIndex == -1) {
      // Build your app / all done
      cards.addAll([
        _FeatureCard(
          icon: LucideIcons.database,
          title: 'Set up your database',
          description:
              'Create collections and documents to store your app data.',
          onTap: () => context.go('/project/$projectId/databases'),
          actionLabel: 'Set up your database',
        ),
        _FeatureCard(
          icon: LucideIcons.bookOpen,
          title: 'Discover our docs',
          description:
              'API references, tutorials, quick start guides, and more.',
          links: const [
            'API references',
            'Tutorials',
            'Storage quick start',
            'Functions quick start',
          ],
        ),
        _FeatureCard(
          icon: LucideIcons.users,
          title: 'Set up Auth',
          description:
              'Email & password, OAuth 2, magic links, and more.',
          onTap: () => context.go('/project/$projectId/auth'),
          actionLabel: 'View all methods',
        ),
        _FeatureCard(
          icon: LucideIcons.zap,
          title: 'Deploy a function',
          description:
              'Run server-side code triggered by events or HTTP requests.',
          onTap: () => context.go('/project/$projectId/functions'),
          actionLabel: 'Deploy a function',
        ),
      ]);
    }

    if (cards.isEmpty) return const SizedBox();

    // 2-column grid
    return LayoutBuilder(
      builder: (context, constraints) {
        final wide = constraints.maxWidth > 600;
        if (!wide) {
          return Column(
            children: cards
                .map((c) => Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: c))
                .toList(),
          );
        }
        final cardWidth = (constraints.maxWidth - 16) / 2;
        return Wrap(
          spacing: 16,
          runSpacing: 16,
          children: cards
              .map((c) => SizedBox(width: cardWidth, child: c))
              .toList(),
        );
      },
    );
  }
}

// --- Models ------------------------------------------------------------------

class _Step {
  final String title;
  final String? subtitle;
  final bool done;
  final String route;

  const _Step({
    required this.title,
    this.subtitle,
    required this.done,
    required this.route,
  });
}

// --- Widgets -----------------------------------------------------------------

class _TimelineStep extends StatefulWidget {
  final _Step step;
  final bool isLast;
  final bool isCurrent;
  final VoidCallback onTap;

  const _TimelineStep({
    required this.step,
    required this.isLast,
    required this.isCurrent,
    required this.onTap,
  });

  @override
  State<_TimelineStep> createState() => _TimelineStepState();
}

class _TimelineStepState extends State<_TimelineStep> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final step = widget.step;

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor:
          step.done ? SystemMouseCursors.basic : SystemMouseCursors.click,
      child: GestureDetector(
        onTap: step.done ? null : widget.onTap,
        child: IntrinsicHeight(
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Timeline column
              SizedBox(
                width: 40,
                child: Column(
                  children: [
                    // Dot / check
                    Container(
                      width: 24,
                      height: 24,
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: step.done
                            ? _green
                            : widget.isCurrent
                                ? _accent
                                : Colors.white.withOpacity(0.08),
                        border: step.done || widget.isCurrent
                            ? null
                            : Border.all(
                                color: Colors.white.withOpacity(0.15)),
                      ),
                      child: step.done
                          ? const Icon(LucideIcons.check,
                              size: 12, color: Colors.white)
                          : widget.isCurrent
                              ? Center(
                                  child: Text('Now',
                                      style: TextStyle(
                                          color: Colors.white,
                                          fontSize: 8,
                                          fontWeight: FontWeight.w700)))
                              : null,
                    ),
                    // Vertical line
                    if (!widget.isLast)
                      Expanded(
                        child: Container(
                          width: 1.5,
                          margin: const EdgeInsets.symmetric(vertical: 4),
                          color: step.done
                              ? _green.withOpacity(0.3)
                              : Colors.white.withOpacity(0.06),
                        ),
                      ),
                  ],
                ),
              ),
              const SizedBox(width: 12),
              // Content
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.only(bottom: 24),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // Badge
                      if (step.done)
                        Container(
                          margin: const EdgeInsets.only(bottom: 6),
                          padding: const EdgeInsets.symmetric(
                              horizontal: 8, vertical: 3),
                          decoration: BoxDecoration(
                            color: _green.withOpacity(0.15),
                            borderRadius: BorderRadius.circular(4),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              const Icon(LucideIcons.check,
                                  size: 10, color: _green),
                              const SizedBox(width: 4),
                              const Text('Done',
                                  style: TextStyle(
                                      color: _green,
                                      fontSize: 11,
                                      fontWeight: FontWeight.w600)),
                            ],
                          ),
                        )
                      else if (widget.isCurrent)
                        Container(
                          margin: const EdgeInsets.only(bottom: 6),
                          padding: const EdgeInsets.symmetric(
                              horizontal: 8, vertical: 3),
                          decoration: BoxDecoration(
                            color: _accent.withOpacity(0.15),
                            borderRadius: BorderRadius.circular(4),
                          ),
                          child: const Text('Now',
                              style: TextStyle(
                                  color: Color(0xFF60A5FA),
                                  fontSize: 11,
                                  fontWeight: FontWeight.w600)),
                        ),
                      // Title
                      Text(
                        step.title,
                        style: TextStyle(
                          color: step.done ? _dimText : Colors.white,
                          fontSize: 15,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                      if (step.subtitle != null && !step.done) ...[
                        const SizedBox(height: 3),
                        Text(step.subtitle!,
                            style: const TextStyle(
                                color: _subtleText, fontSize: 12)),
                      ],
                    ],
                  ),
                ),
              ),
              // Arrow on hover (non-done steps)
              if (!step.done && _hovered)
                Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Icon(LucideIcons.arrowRight,
                      size: 16,
                      color: Colors.white.withOpacity(0.4)),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _FeatureCard extends StatefulWidget {
  final IconData icon;
  final String title;
  final String description;
  final VoidCallback? onTap;
  final String? actionLabel;
  final List<String>? links;

  const _FeatureCard({
    required this.icon,
    required this.title,
    required this.description,
    this.onTap,
    this.actionLabel,
    this.links,
  });

  @override
  State<_FeatureCard> createState() => _FeatureCardState();
}

class _FeatureCardState extends State<_FeatureCard> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: widget.onTap != null
          ? SystemMouseCursors.click
          : SystemMouseCursors.basic,
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          width: double.infinity,
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: _hovered
                ? Colors.white.withOpacity(0.04)
                : _cardColor,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
                color: Colors.white.withOpacity(0.06)),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(widget.icon, size: 20, color: _accent),
              const SizedBox(height: 12),
              Text(widget.title,
                  style: const TextStyle(
                      color: Colors.white,
                      fontSize: 15,
                      fontWeight: FontWeight.w600)),
              const SizedBox(height: 6),
              Text(widget.description,
                  style: const TextStyle(
                      color: _dimText, fontSize: 13)),
              if (widget.links != null) ...[
                const SizedBox(height: 12),
                ...widget.links!.map((link) => Padding(
                      padding: const EdgeInsets.only(bottom: 4),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(LucideIcons.arrowRight,
                              size: 12, color: _accent),
                          const SizedBox(width: 6),
                          Text(link,
                              style: const TextStyle(
                                  color: _dimText, fontSize: 12)),
                        ],
                      ),
                    )),
              ],
              if (widget.onTap != null) ...[
                const SizedBox(height: 12),
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(widget.actionLabel ?? widget.title,
                        style: const TextStyle(
                            color: _accent, fontSize: 13)),
                    const SizedBox(width: 4),
                    const Icon(LucideIcons.arrowRight,
                        size: 14, color: _accent),
                  ],
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
