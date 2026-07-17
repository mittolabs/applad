import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_empty_state.dart';
import '../../core/widgets/app_error_state.dart';
import 'observe_providers.dart';
import 'observe_shared.dart';

class ObPerformanceTab extends ConsumerWidget {
  final String projectId;
  const ObPerformanceTab({super.key, required this.projectId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final async  = ref.watch(performanceProvider);

    return async.when(
      loading: () =>
          const Center(child: CircularProgressIndicator(color: obAccent)),
      error: (e, _) => AppErrorState(
          error: e, onRetry: () => ref.invalidate(performanceProvider)),
      data: (data) {
        final metrics   = data['metrics'] as Map<String, dynamic>? ?? {};
        final vitals    = data['vitals'] as Map<String, dynamic>? ?? {};
        final endpoints = List<Map<String, dynamic>>.from(data['endpoints'] ?? []);
        final chart     = List<dynamic>.from(data['chart'] ?? []);
        final traces    = List<Map<String, dynamic>>.from(data['traces'] ?? []);

        final hasData = endpoints.isNotEmpty ||
            chart.isNotEmpty ||
            vitals.isNotEmpty ||
            (metrics['p95Ms'] ?? 0) != 0;

        if (!hasData) {
          return const AppEmptyState(
            icon: LucideIcons.activity,
            title: 'No performance data yet',
            subtitle: 'Instrument your backend with the Applad SDK to start\ntracking latency, Apdex scores and web vitals.',
          );
        }

        return SingleChildScrollView(
          padding: EdgeInsets.symmetric(
              horizontal: pageHPad(context), vertical: 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // ── Latency + Apdex ──────────────────────────────────────────
              LayoutBuilder(builder: (_, bc) {
                final cols = bc.maxWidth > 800 ? 6 : 3;
                return GridView.count(
                  crossAxisCount: cols,
                  mainAxisSpacing: 12,
                  crossAxisSpacing: 12,
                  shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  childAspectRatio: 2.6,
                  children: [
                    _PerfCard('P50', '${metrics['p50Ms'] ?? 0}ms', obGreen),
                    _PerfCard('P75', '${metrics['p75Ms'] ?? 0}ms', obAccent),
                    _PerfCard('P95', '${metrics['p95Ms'] ?? 0}ms', obOrange),
                    _PerfCard('P99', '${metrics['p99Ms'] ?? 0}ms', obRed),
                    _PerfCard('Req/s', '${metrics['rps'] ?? 0}', obPurple),
                    _PerfCard('Apdex', _apdexStr(metrics['apdex']),
                        _apdexColor(metrics['apdex'])),
                  ],
                );
              }),
              const SizedBox(height: 24),

              // ── Web Vitals ────────────────────────────────────────────────
              if (vitals.isNotEmpty) ...[
                obSectionTitle('Web Vitals', colors),
                const SizedBox(height: 12),
                _WebVitalsGrid(vitals: vitals),
                const SizedBox(height: 24),
              ],

              // ── Response time chart ───────────────────────────────────────
              obSectionTitle('Response Time (last 24h)', colors),
              const SizedBox(height: 12),
              Container(
                height: 140,
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                    color: colors.surface,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: colors.border)),
                child: chart.isEmpty
                    ? Center(
                        child: Text('No data yet',
                            style: TextStyle(
                                color: colors.textSubtle, fontSize: 12)))
                    : ObLineChart(
                        points: chart
                            .map((p) =>
                                ((p as Map)['p95'] as num?)?.toDouble() ??
                                0.0)
                            .toList(),
                        color: obAccent),
              ),
              const SizedBox(height: 24),

              // ── Slowest endpoints ─────────────────────────────────────────
              obSectionTitle('Slowest Endpoints', colors),
              const SizedBox(height: 12),
              endpoints.isEmpty
                  ? obEmptyCard('No endpoints tracked yet', colors)
                  : _EndpointsTable(endpoints: endpoints, colors: colors),
              const SizedBox(height: 24),

              // ── Recent traces ─────────────────────────────────────────────
              if (traces.isNotEmpty) ...[
                obSectionTitle('Recent Traces', colors),
                const SizedBox(height: 12),
                _TracesTable(traces: traces, colors: colors),
              ],
            ],
          ),
        );
      },
    );
  }

  String _apdexStr(dynamic v) =>
      (v is num) ? v.toStringAsFixed(2) : '—';

  Color _apdexColor(dynamic v) {
    final d = (v is num) ? v.toDouble() : 1.0;
    if (d >= 0.9) return obGreen;
    if (d >= 0.7) return obOrange;
    return obRed;
  }
}

// ── Perf card ─────────────────────────────────────────────────────────────────

class _PerfCard extends StatelessWidget {
  final String label, value;
  final Color color;
  const _PerfCard(this.label, this.value, this.color);

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border)),
      child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(value,
                style: TextStyle(
                    color: color,
                    fontSize: 20,
                    fontWeight: FontWeight.w700)),
            Text(label,
                style: TextStyle(
                    color: colors.textSecondary, fontSize: 11)),
          ]),
    );
  }
}

// ── Web Vitals grid ───────────────────────────────────────────────────────────

class _WebVitalsGrid extends StatelessWidget {
  final Map<String, dynamic> vitals;
  const _WebVitalsGrid({required this.vitals});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final items = [
      _Vital('LCP',  vitals['lcp'],  'ms',  2500, 4000, 'Largest Contentful Paint'),
      _Vital('FID',  vitals['fid'],  'ms',  100,  300,  'First Input Delay'),
      _Vital('CLS',  vitals['cls'],  '',    0.1,  0.25, 'Cumulative Layout Shift'),
      _Vital('TTFB', vitals['ttfb'], 'ms',  800,  1800, 'Time to First Byte'),
      _Vital('FCP',  vitals['fcp'],  'ms',  1800, 3000, 'First Contentful Paint'),
      _Vital('INP',  vitals['inp'],  'ms',  200,  500,  'Interaction to Next Paint'),
    ];
    return LayoutBuilder(builder: (_, bc) {
      final cols = bc.maxWidth > 900 ? 6 : bc.maxWidth > 600 ? 3 : 2;
      return GridView.count(
        crossAxisCount: cols,
        mainAxisSpacing: 12,
        crossAxisSpacing: 12,
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        childAspectRatio: 1.4,
        children: items
            .map((v) => _VitalCard(item: v, colors: colors))
            .toList(),
      );
    });
  }
}

class _Vital {
  final String name, unit, description;
  final dynamic value;
  final num goodThreshold, poorThreshold;
  const _Vital(this.name, this.value, this.unit, this.goodThreshold,
      this.poorThreshold, this.description);

  Color get color {
    final v = (value is num) ? (value as num).toDouble() : 0.0;
    if (v <= goodThreshold) return obGreen;
    if (v <= poorThreshold) return obOrange;
    return obRed;
  }

  String get rating {
    final v = (value is num) ? (value as num).toDouble() : 0.0;
    if (v <= goodThreshold) return 'Good';
    if (v <= poorThreshold) return 'Needs work';
    return 'Poor';
  }

  String get display {
    if (value == null) return '—';
    final v = (value is num) ? value : num.tryParse('$value');
    if (v == null) return '—';
    if (unit.isEmpty) return (v as num).toStringAsFixed(3);
    return '${(v as num).round()}$unit';
  }
}

class _VitalCard extends StatelessWidget {
  final _Vital item;
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
                        fontSize: 12,
                        fontWeight: FontWeight.w600)),
                const Spacer(),
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 5, vertical: 2),
                  decoration: BoxDecoration(
                      color: item.color.withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(4)),
                  child: Text(item.rating,
                      style: TextStyle(
                          color: item.color,
                          fontSize: 9,
                          fontWeight: FontWeight.w600)),
                ),
              ]),
              const SizedBox(height: 6),
              Text(item.display,
                  style: TextStyle(
                      color: item.color,
                      fontSize: 20,
                      fontWeight: FontWeight.w700,
                      fontFamily: 'monospace')),
              const Spacer(),
              Text(item.description,
                  style: TextStyle(
                      color: colors.textSubtle, fontSize: 9)),
            ]),
      );
}

// ── Endpoints table ───────────────────────────────────────────────────────────

class _EndpointsTable extends StatelessWidget {
  final List<Map<String, dynamic>> endpoints;
  final ConsoleColors colors;
  const _EndpointsTable(
      {required this.endpoints, required this.colors});

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border)),
      child: Column(children: [
        _Header(colors),
        ...endpoints.take(20).map((ep) => _EndpointRow(ep: ep, colors: colors)),
      ]),
    );
  }
}

class _Header extends StatelessWidget {
  final ConsoleColors colors;
  const _Header(this.colors);

  @override
  Widget build(BuildContext context) => Container(
        padding:
            const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        decoration:
            BoxDecoration(border: Border(bottom: BorderSide(color: colors.border))),
        child: Row(children: [
          SizedBox(
              width: 70,
              child: Text('Method',
                  style: TextStyle(
                      color: colors.textSubtle, fontSize: 11))),
          Expanded(
              child: Text('Endpoint',
                  style: TextStyle(
                      color: colors.textSubtle, fontSize: 11))),
          for (final h in ['P50', 'P95', 'P99', 'Req/s', 'Err%'])
            SizedBox(
                width: 70,
                child: Text(h,
                    style: TextStyle(
                        color: colors.textSubtle, fontSize: 11))),
        ]),
      );
}

class _EndpointRow extends StatelessWidget {
  final Map<String, dynamic> ep;
  final ConsoleColors colors;
  const _EndpointRow({required this.ep, required this.colors});

  @override
  Widget build(BuildContext context) {
    final method   = ep['method'] as String? ?? 'GET';
    final errPct   = (ep['errorPct'] ?? 0) as num;
    final mc = switch (method.toUpperCase()) {
      'GET'           => obGreen,
      'POST'          => obAccent,
      'PUT' || 'PATCH'=> obOrange,
      'DELETE'        => obRed,
      _               => const Color(0xFF64748B),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      decoration: BoxDecoration(
          border: Border(bottom: BorderSide(color: colors.border))),
      child: Row(children: [
        SizedBox(
          width: 70,
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
            decoration: BoxDecoration(
                color: mc.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(3)),
            child: Text(method,
                style: TextStyle(
                    color: mc,
                    fontSize: 10,
                    fontWeight: FontWeight.w700,
                    fontFamily: 'monospace')),
          ),
        ),
        Expanded(
          child: Text(ep['path'] as String? ?? '/',
              style: TextStyle(color: colors.textPrimary, fontSize: 12),
              overflow: TextOverflow.ellipsis),
        ),
        SizedBox(
            width: 70,
            child: Text('${ep['p50Ms'] ?? 0}ms',
                style: TextStyle(
                    color: colors.textSecondary, fontSize: 12))),
        SizedBox(
            width: 70,
            child: Text('${ep['p95Ms'] ?? 0}ms',
                style: const TextStyle(color: obOrange, fontSize: 12))),
        SizedBox(
            width: 70,
            child: Text('${ep['p99Ms'] ?? 0}ms',
                style: const TextStyle(color: obRed, fontSize: 12))),
        SizedBox(
            width: 70,
            child: Text('${ep['rps'] ?? 0}',
                style: TextStyle(
                    color: colors.textSecondary, fontSize: 12))),
        SizedBox(
            width: 70,
            child: Text('${errPct.toStringAsFixed(1)}%',
                style: TextStyle(
                    color: errPct > 5 ? obRed : colors.textSecondary,
                    fontSize: 12))),
      ]),
    );
  }
}

// ── Traces table ──────────────────────────────────────────────────────────────

class _TracesTable extends StatelessWidget {
  final List<Map<String, dynamic>> traces;
  final ConsoleColors colors;
  const _TracesTable({required this.traces, required this.colors});

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: colors.border)),
      child: Column(children: [
        Container(
          padding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          decoration: BoxDecoration(
              border:
                  Border(bottom: BorderSide(color: colors.border))),
          child: Row(children: [
            Expanded(child: Text('Trace',
                style: TextStyle(color: colors.textSubtle, fontSize: 11))),
            for (final h in ['Duration', 'Status', 'When'])
              SizedBox(width: 90,
                  child: Text(h,
                      style: TextStyle(
                          color: colors.textSubtle, fontSize: 11))),
          ]),
        ),
        ...traces.take(10).map((t) {
          final name   = t['name'] as String? ?? 'Trace';
          final dur    = t['durationMs'] ?? 0;
          final status = t['status'] as String? ?? 'ok';
          final ts     = t['timestamp'];
          final sc     = status == 'ok' ? obGreen : obRed;
          return Container(
            padding: const EdgeInsets.symmetric(
                horizontal: 16, vertical: 10),
            decoration: BoxDecoration(
                border:
                    Border(bottom: BorderSide(color: colors.border))),
            child: Row(children: [
              Expanded(
                child: Row(children: [
                  Icon(LucideIcons.activity,
                      size: 12, color: colors.textSubtle),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(name,
                        style: TextStyle(
                            color: colors.textPrimary, fontSize: 12),
                        overflow: TextOverflow.ellipsis),
                  ),
                ]),
              ),
              SizedBox(
                  width: 90,
                  child: Text('${dur}ms',
                      style: TextStyle(
                          color: dur > 1000 ? obOrange : colors.textSecondary,
                          fontSize: 12))),
              SizedBox(
                  width: 90,
                  child: Text(status.toUpperCase(),
                      style: TextStyle(
                          color: sc,
                          fontSize: 10,
                          fontWeight: FontWeight.w700))),
              SizedBox(
                  width: 90,
                  child: Text(obTimeAgo(ts),
                      style: TextStyle(
                          color: colors.textSubtle, fontSize: 11))),
            ]),
          );
        }),
      ]),
    );
  }
}
