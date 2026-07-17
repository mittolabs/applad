import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_empty_state.dart';
import 'observe_providers.dart';
import 'observe_shared.dart';

class ObOverviewTab extends ConsumerWidget {
  final String projectId;
  const ObOverviewTab({super.key, required this.projectId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors    = consoleColors(context);
    final overview  = ref.watch(overviewProvider);
    final errAsync  = ref.watch(errorsProvider);

    return overview.when(
      loading: () =>
          const Center(child: CircularProgressIndicator(color: obAccent)),
      error: (_, __) => const SizedBox(),
      data: (data) {
        final stats    = data['stats'] as Map<String, dynamic>? ?? {};
        final services = List<Map<String, dynamic>>.from(data['services'] ?? []);
        final vitals   = data['vitals'] as Map<String, dynamic>? ?? {};

        return SingleChildScrollView(
          padding: EdgeInsets.symmetric(
              horizontal: pageHPad(context), vertical: 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // ── Stat cards ──────────────────────────────────────────────────
              LayoutBuilder(builder: (_, bc) {
                final cols = bc.maxWidth > 900 ? 5 : bc.maxWidth > 600 ? 3 : 2;
                return GridView.count(
                  crossAxisCount: cols,
                  mainAxisSpacing: 12,
                  crossAxisSpacing: 12,
                  shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  childAspectRatio: 3.0,
                  children: [
                    _StatCard('Errors (24h)',
                        obFmtNum(stats['errorsToday'] ?? 0), obRed,
                        LucideIcons.alertTriangle),
                    _StatCard('P95 Latency',
                        '${stats['p95Ms'] ?? 0}ms', obAccent,
                        LucideIcons.timer),
                    _StatCard('Uptime',
                        '${stats['uptimePct'] ?? 100}%', obGreen,
                        LucideIcons.heartPulse),
                    _StatCard('Apdex',
                        '${(stats['apdex'] ?? 1.0).toStringAsFixed(2)}',
                        _apdexColor(stats['apdex']),
                        LucideIcons.gauge),
                    _StatCard('Log volume (1h)',
                        obFmtNum(stats['logsLastHour'] ?? 0), obPurple,
                        LucideIcons.terminal),
                  ],
                );
              }),
              const SizedBox(height: 28),

              // ── Web Vitals ──────────────────────────────────────────────────
              if (vitals.isNotEmpty) ...[
                obSectionTitle('Web Vitals', colors),
                const SizedBox(height: 12),
                _WebVitalsRow(vitals: vitals, colors: colors),
                const SizedBox(height: 28),
              ],

              // ── Service health ──────────────────────────────────────────────
              obSectionTitle('Service Health', colors),
              const SizedBox(height: 12),
              services.isEmpty
                  ? const AppEmptyState(
                      icon: LucideIcons.heartPulse,
                      title: 'No services configured yet',
                      subtitle: 'Register uptime monitors to track your service health here.',
                    )
                  : Wrap(
                      spacing: 12,
                      runSpacing: 12,
                      children: services
                          .map((s) => _ServiceCard(service: s))
                          .toList(),
                    ),
              const SizedBox(height: 28),

              // ── Recent errors ───────────────────────────────────────────────
              Row(children: [
                Text('Recent Errors',
                    style: TextStyle(
                        color: colors.textPrimary,
                        fontSize: 14,
                        fontWeight: FontWeight.w600)),
                const Spacer(),
                TextButton(
                  onPressed: () =>
                      context.go('/project/$projectId/errors'),
                  child: const Text('View all',
                      style: TextStyle(color: obAccent, fontSize: 12)),
                ),
              ]),
              const SizedBox(height: 12),
              errAsync.when(
                loading: () => const SizedBox(),
                error: (_, __) => const SizedBox(),
                data: (d) {
                  final errs = List<Map<String, dynamic>>.from(
                          d['errors'] ?? [])
                      .take(5)
                      .toList();
                  if (errs.isEmpty) {
                    return const AppEmptyState(
                      icon: LucideIcons.checkCircle,
                      title: 'No errors — great job!',
                      subtitle: 'Errors captured by the Applad SDK will appear here.',
                    );
                  }
                  return Column(
                    children: errs
                        .map((e) => Padding(
                              padding: const EdgeInsets.only(bottom: 6),
                              child: _CompactErrorRow(err: e, colors: colors),
                            ))
                        .toList(),
                  );
                },
              ),
            ],
          ),
        );
      },
    );
  }

  Color _apdexColor(dynamic v) {
    final d = (v is num) ? v.toDouble() : 1.0;
    if (d >= 0.9) return obGreen;
    if (d >= 0.7) return obOrange;
    return obRed;
  }
}

class _StatCard extends StatelessWidget {
  final String label, value;
  final Color color;
  final IconData icon;
  const _StatCard(this.label, this.value, this.color, this.icon);

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border)),
      child: Row(children: [
        Container(
          width: 34,
          height: 34,
          decoration: BoxDecoration(
              color: color.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(8)),
          child: Icon(icon, size: 15, color: color),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text(value,
                    style: TextStyle(
                        color: colors.textPrimary,
                        fontSize: 18,
                        fontWeight: FontWeight.w700)),
                Text(label,
                    style: TextStyle(
                        color: colors.textSecondary, fontSize: 11)),
              ]),
        ),
      ]),
    );
  }
}

class _WebVitalsRow extends StatelessWidget {
  final Map<String, dynamic> vitals;
  final ConsoleColors colors;
  const _WebVitalsRow({required this.vitals, required this.colors});

  @override
  Widget build(BuildContext context) {
    final items = [
      _VitalItem('LCP', vitals['lcp'], 'ms', 2500, 4000, 'Largest Contentful Paint'),
      _VitalItem('FID', vitals['fid'], 'ms', 100, 300, 'First Input Delay'),
      _VitalItem('CLS', vitals['cls'], '', 0.1, 0.25, 'Cumulative Layout Shift'),
      _VitalItem('TTFB', vitals['ttfb'], 'ms', 800, 1800, 'Time to First Byte'),
      _VitalItem('FCP', vitals['fcp'], 'ms', 1800, 3000, 'First Contentful Paint'),
    ];
    return Row(
      children: items
          .map((v) => Expanded(child: Padding(
                padding: const EdgeInsets.only(right: 10),
                child: _VitalCard(item: v, colors: colors),
              )))
          .toList(),
    );
  }
}

class _VitalItem {
  final String name, unit, description;
  final dynamic value;
  final num good, poor;
  const _VitalItem(this.name, this.value, this.unit, this.good, this.poor,
      this.description);

  Color get color {
    final v = (value is num) ? (value as num).toDouble() : 0.0;
    if (v <= good) return obGreen;
    if (v <= poor) return obOrange;
    return obRed;
  }

  String get rating {
    final v = (value is num) ? (value as num).toDouble() : 0.0;
    if (v <= good) return 'Good';
    if (v <= poor) return 'Needs improvement';
    return 'Poor';
  }

  String get display {
    if (value == null) return '—';
    final v = (value is num) ? value : num.tryParse('$value');
    if (v == null) return '—';
    if (unit.isEmpty) return v.toStringAsFixed(3);
    return '${v.round()}$unit';
  }
}

class _VitalCard extends StatelessWidget {
  final _VitalItem item;
  final ConsoleColors colors;
  const _VitalCard({required this.item, required this.colors});

  @override
  Widget build(BuildContext context) => Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
            color: colors.surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: colors.border)),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Text(item.name,
                  style: TextStyle(
                      color: colors.textPrimary,
                      fontSize: 13,
                      fontWeight: FontWeight.w600)),
              const Spacer(),
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                    color: item.color.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(4)),
                child: Text(item.rating,
                    style: TextStyle(
                        color: item.color,
                        fontSize: 10,
                        fontWeight: FontWeight.w600)),
              ),
            ]),
            const SizedBox(height: 6),
            Text(item.display,
                style: TextStyle(
                    color: item.color,
                    fontSize: 22,
                    fontWeight: FontWeight.w700,
                    fontFamily: 'monospace')),
            const SizedBox(height: 2),
            Text(item.description,
                style: TextStyle(color: colors.textSubtle, fontSize: 10)),
          ],
        ),
      );
}

class _ServiceCard extends StatelessWidget {
  final Map<String, dynamic> service;
  const _ServiceCard({required this.service});

  @override
  Widget build(BuildContext context) {
    final colors  = consoleColors(context);
    final status  = service['status'] as String? ?? 'healthy';
    final (dot, label) = switch (status) {
      'healthy'  => (obGreen,  'Healthy'),
      'degraded' => (obOrange, 'Degraded'),
      'down'     => (obRed,    'Down'),
      _          => (obGreen,  'Unknown'),
    };
    return Container(
      width: 180,
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border)),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Container(
              width: 8,
              height: 8,
              decoration: BoxDecoration(color: dot, shape: BoxShape.circle)),
          const SizedBox(width: 6),
          Text(label,
              style: TextStyle(
                  color: dot, fontSize: 11, fontWeight: FontWeight.w500)),
        ]),
        const SizedBox(height: 8),
        Text(service['name'] as String? ?? 'Service',
            style: TextStyle(
                color: colors.textPrimary,
                fontSize: 13,
                fontWeight: FontWeight.w500)),
        const SizedBox(height: 2),
        Text(
            '${service['latencyMs'] ?? 0}ms  •  ${service['uptime'] ?? 100}% uptime',
            style: TextStyle(color: colors.textSubtle, fontSize: 11)),
      ]),
    );
  }
}

class _CompactErrorRow extends StatelessWidget {
  final Map<String, dynamic> err;
  final ConsoleColors colors;
  const _CompactErrorRow({required this.err, required this.colors});

  @override
  Widget build(BuildContext context) {
    final level = err['level'] as String? ?? 'error';
    final c = ObLevelChip.colorFor(level);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border)),
      child: Row(children: [
        Container(
            width: 7,
            height: 7,
            decoration: BoxDecoration(color: c, shape: BoxShape.circle)),
        const SizedBox(width: 10),
        Expanded(
          child: Text(err['title'] as String? ?? 'Unknown error',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: colors.textPrimary, fontSize: 13)),
        ),
        const SizedBox(width: 12),
        Text(obFmtNum(err['count'] ?? 0),
            style: TextStyle(color: colors.textSubtle, fontSize: 12)),
        const SizedBox(width: 8),
        Text(obTimeAgo(err['lastSeen']),
            style: TextStyle(color: colors.textSubtle, fontSize: 11)),
      ]),
    );
  }
}
