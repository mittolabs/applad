import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import 'observe_overview.dart';
import 'observe_errors.dart';
import 'observe_performance.dart';
import 'observe_releases.dart';
import 'observe_logs.dart';
import 'observe_replays.dart';
import 'observe_uptime.dart';
import 'observe_crons.dart';
import 'observe_alerts.dart';

// ─────────────────────────────────────────────────────────────────────────────
// Observe page — sidebar-driven, no tab bar
// ─────────────────────────────────────────────────────────────────────────────

class ObservePage extends ConsumerWidget {
  const ObservePage({super.key});

  static const _sections = {
    'observe':     ('Overview',    'Errors, performance, releases, logs and uptime for your project'),
    'errors':      ('Errors',      'Track, triage and resolve errors in your project'),
    'performance': ('Performance', 'P95 latency, Apdex scores and web vitals'),
    'releases':    ('Releases',    'Tag deployments and correlate them with errors'),
    'logs':        ('Logs',        'Search and tail structured logs from your project'),
    'replays':     ('Replays',     'Session replays for debugging user-reported issues'),
    'uptime':      ('Uptime',      'HTTP monitors with alerts for downtime'),
    'crons':       ('Crons',       'Monitor scheduled jobs and detect missed runs'),
    'alerts':      ('Alerts',      'Rules that notify you when metrics exceed thresholds'),
  };

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors    = consoleColors(context);
    final projectId = ref.watch(currentProjectProvider) ?? '';
    final seg       = GoRouterState.of(context).uri.path.split('/').last;
    final section   = _sections[seg] ?? _sections['observe']!;

    return Scaffold(
      backgroundColor: colors.background,
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // ── Page header ───────────────────────────────────────────────────
          Padding(
            padding: EdgeInsets.fromLTRB(
                pageHPad(context), 32, pageHPad(context), 20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  section.$1,
                  style: TextStyle(
                    color: colors.textPrimary,
                    fontSize: 22,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  section.$2,
                  style: TextStyle(color: colors.textSecondary, fontSize: 13),
                ),
              ],
            ),
          ),

          // ── Section content ───────────────────────────────────────────────
          Expanded(
            child: switch (seg) {
              'errors'      => ObErrorsTab(projectId: projectId),
              'performance' => ObPerformanceTab(projectId: projectId),
              'releases'    => ObReleasesTab(projectId: projectId),
              'logs'        => ObLogsTab(projectId: projectId),
              'replays'     => ObReplaysTab(projectId: projectId),
              'uptime'      => ObUptimeTab(projectId: projectId),
              'crons'       => ObCronsTab(projectId: projectId),
              'alerts'      => ObAlertsTab(projectId: projectId),
              _             => ObOverviewTab(projectId: projectId),
            },
          ),
        ],
      ),
    );
  }
}
