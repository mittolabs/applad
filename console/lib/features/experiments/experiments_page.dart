import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/providers/experiments_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';

const _green = Color(0xFF10B981);
const _orange = Color(0xFFF59E0B);

class _Experiment {
  final String key;
  final String name;
  final String description;
  final IconData icon;
  final String category;

  const _Experiment(this.key, this.name, this.description, this.icon,
      this.category);
}

const _experiments = <_Experiment>[
  _Experiment('aiChat', 'AI Assistant', 'Chat interface to build and manage with natural language',
      LucideIcons.sparkles, 'Coming soon'),
  _Experiment('search', 'Full-text Search', 'Search service with relevance ranking and facets',
      LucideIcons.search, 'Coming soon'),
  _Experiment('analytics', 'Event Analytics', 'First-party event tracking, funnels, and retention',
      LucideIcons.barChart3, 'Coming soon'),
  _Experiment('cache', 'Managed Cache', 'Cache API with TTL and event-driven invalidation',
      LucideIcons.database, 'Coming soon'),
  _Experiment('billing', 'Billing & Metering', 'Usage metering, plans, and Stripe integration',
      LucideIcons.creditCard, 'Coming soon'),
  _Experiment('edgeFunctions', 'Edge Functions', 'Serverless at the edge for low-latency middleware',
      LucideIcons.globe, 'Coming soon'),
  _Experiment('vectors', 'AI / Vectors', 'Vector store and embedding pipeline for RAG',
      LucideIcons.brain, 'Coming soon'),
  _Experiment('regions', 'Multi-Region', 'Data residency and region-pinned deployments',
      LucideIcons.mapPin, 'Coming soon'),
];

class ExperimentsPage extends ConsumerWidget {
  const ExperimentsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final experiments = ref.watch(experimentsProvider);
    final map = experiments.toMap();
    final enabledCount = map.values.where((v) => v).length;

    return Container(
      color: colors.background,
      child: Column(
        children: [
          // Header
          Padding(
            padding: EdgeInsets.fromLTRB(pageHPad(context), 28, pageHPad(context), 0),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          const Icon(LucideIcons.flaskConical,
                              size: 20, color: _orange),
                          const SizedBox(width: 8),
                          Text('Experiments',
                              style: TextStyle(
                                  color: colors.textPrimary,
                                  fontSize: 24,
                                  fontWeight: FontWeight.w700)),
                        ],
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'Enable upcoming features that are still in development. '
                        '$enabledCount of ${_experiments.length} enabled.',
                        style: TextStyle(
                          color: colors.textSecondary, fontSize: 14),
                      ),
                    ],
                  ),
                ),
                OutlinedButton(
                  style: OutlinedButton.styleFrom(
                    foregroundColor: colors.textSecondary,
                    side: BorderSide(color: colors.border),
                    shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8)),
                    padding: const EdgeInsets.symmetric(
                        horizontal: 14, vertical: 8),
                  ),
                  onPressed: () {
                    if (enabledCount == _experiments.length) {
                      ref.read(experimentsProvider.notifier).disableAll();
                    } else {
                      ref.read(experimentsProvider.notifier).enableAll();
                    }
                  },
                  child: Text(
                    enabledCount == _experiments.length
                        ? 'Disable all'
                        : 'Enable all',
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),

          // Warning banner
          Padding(
            padding: EdgeInsets.symmetric(horizontal: pageHPad(context), vertical: 8),
            child: Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: _orange.withValues(alpha: 0.08),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: _orange.withValues(alpha: 0.2)),
              ),
              child: Row(
                children: [
                  const Icon(LucideIcons.alertTriangle,
                      size: 16, color: _orange),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      'Experimental features may be incomplete, unstable, or change without notice. '
                      'They are not recommended for production use.',
                      style: TextStyle(
                          color: _orange.withValues(alpha: 0.8),
                          fontSize: 12),
                    ),
                  ),
                ],
              ),
            ),
          ),

          const SizedBox(height: 8),

          // Experiment list
          Expanded(
            child: ListView(
              padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
              children: [
                ..._buildCategory(context, 'Console Sections', map, ref),
                const SizedBox(height: 16),
                ..._buildCategory(context, 'Backend Services', map, ref),
                const SizedBox(height: 32),
              ],
            ),
          ),
        ],
      ),
    );
  }

  List<Widget> _buildCategory(
      BuildContext context, String category, Map<String, bool> map, WidgetRef ref) {
    final colors = consoleColors(context);
    final items =
        _experiments.where((e) => e.category == category).toList();
    return [
      Padding(
        padding: const EdgeInsets.only(top: 8, bottom: 8),
        child: Text(
          category.toUpperCase(),
          style: TextStyle(
            color: colors.textSubtle,
            fontSize: 11,
            fontWeight: FontWeight.w600,
            letterSpacing: 1.0,
          ),
        ),
      ),
      ...items.map((exp) {
        final enabled = map[exp.key] == true;
        return Container(
          margin: const EdgeInsets.only(bottom: 8),
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: colors.surface,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(
              color: enabled
                  ? _green.withValues(alpha: 0.2)
                  : colors.border,
            ),
          ),
          child: Row(
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: enabled
                      ? _green.withValues(alpha: 0.1)
                      : colors.fillHover,
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Icon(exp.icon,
                    size: 20,
                    color: enabled
                        ? _green
                        : colors.textSubtle),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Text(exp.name,
                          style: TextStyle(
                            color: colors.textPrimary,
                                fontSize: 14,
                                fontWeight: FontWeight.w500)),
                        if (enabled) ...[
                          const SizedBox(width: 8),
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 6, vertical: 1),
                            decoration: BoxDecoration(
                              color: _green.withValues(alpha: 0.15),
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: const Text('ON',
                                style: TextStyle(
                                    color: _green,
                                    fontSize: 9,
                                    fontWeight: FontWeight.w700)),
                          ),
                        ],
                      ],
                    ),
                    const SizedBox(height: 2),
                    Text(exp.description,
                        style: TextStyle(
                            color: colors.textSubtle, fontSize: 12)),
                  ],
                ),
              ),
              Switch(
                value: enabled,
                activeThumbColor: _green,
                onChanged: (_) =>
                    ref.read(experimentsProvider.notifier).toggle(exp.key),
              ),
            ],
          ),
        );
      }),
    ];
  }
}
