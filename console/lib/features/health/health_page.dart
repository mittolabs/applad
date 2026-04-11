import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';

import '../../core/api/client.dart';
import '../../core/providers/project_provider.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';

const _green = Color(0xFF10B981);
const _red = Color(0xFFEF4444);
const _amber = Color(0xFFF59E0B);

final _healthProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  final api = ref.read(apiClientProvider);

  Future<Map<String, dynamic>> fetchCheck(String path, String label) async {
    try {
      final response = await api.get(path);
      final payload = Map<String, dynamic>.from(response.data as Map);
      payload['label'] = label;
      payload['path'] = path;
      return payload;
    } catch (error) {
      return {
        'label': label,
        'path': path,
        'status': 'fail',
        'ping': 0,
        'error': error.toString(),
      };
    }
  }

  final checks = await Future.wait([
    fetchCheck('/health', 'Gateway'),
    fetchCheck('/health/db', 'PostgreSQL'),
    fetchCheck('/health/cache', 'Redis'),
  ]);

  final overall = checks.every((item) => item['status'] == 'pass')
      ? 'pass'
      : checks.any((item) => item['status'] == 'pass')
          ? 'warn'
          : 'fail';

  return {
    'status': overall,
    'checks': checks,
    'timestamp': DateTime.now().toIso8601String(),
  };
});

class HealthPage extends ConsumerWidget {
  const HealthPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors = consoleColors(context);
    final projectId = ref.watch(currentProjectProvider);
    final healthAsync = ref.watch(_healthProvider);

    return Scaffold(
      backgroundColor: colors.background,
      body: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: pageHPad(context),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 32),
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Health',
                        style: TextStyle(
                          color: colors.textPrimary,
                          fontSize: 24,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        projectId == null
                            ? 'Infrastructure status for the current workspace.'
                            : 'Infrastructure status for project $projectId.',
                        style: TextStyle(color: colors.textSecondary, fontSize: 13),
                      ),
                    ],
                  ),
                ),
                OutlinedButton.icon(
                  onPressed: () => ref.invalidate(_healthProvider),
                  icon: const Icon(LucideIcons.refreshCw, size: 14),
                  label: const Text('Refresh'),
                ),
              ],
            ),
            const SizedBox(height: 24),
            Expanded(
              child: healthAsync.when(
                loading: () => const Center(child: CircularProgressIndicator()),
                error: (error, _) => Center(
                  child: Text(
                    'Failed to load health data: $error',
                    style: const TextStyle(color: _red, fontSize: 13),
                  ),
                ),
                data: (data) {
                  final checks = List<Map<String, dynamic>>.from(data['checks'] ?? const []);
                  final overall = data['status']?.toString() ?? 'fail';
                  return SingleChildScrollView(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _HealthOverviewCard(
                          status: overall,
                          checks: checks,
                          timestamp: data['timestamp']?.toString() ?? '',
                        ),
                        const SizedBox(height: 16),
                        LayoutBuilder(
                          builder: (context, constraints) {
                            final compact = constraints.maxWidth < 1100;
                            return Wrap(
                              spacing: 16,
                              runSpacing: 16,
                              children: checks
                                  .map(
                                    (check) => SizedBox(
                                      width: compact
                                          ? constraints.maxWidth
                                          : (constraints.maxWidth - 16) / 2,
                                      child: _HealthCheckCard(check: check),
                                    ),
                                  )
                                  .toList(),
                            );
                          },
                        ),
                      ],
                    ),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _HealthOverviewCard extends StatelessWidget {
  final String status;
  final List<Map<String, dynamic>> checks;
  final String timestamp;

  const _HealthOverviewCard({
    required this.status,
    required this.checks,
    required this.timestamp,
  });

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final passCount = checks.where((item) => item['status'] == 'pass').length;
    final statusColor = _statusColor(status);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colors.border),
      ),
      child: Row(
        children: [
          Container(
            width: 52,
            height: 52,
            decoration: BoxDecoration(
              color: statusColor.withValues(alpha: 0.14),
              borderRadius: BorderRadius.circular(16),
            ),
            child: Icon(LucideIcons.heartPulse, color: statusColor, size: 24),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  status == 'pass'
                      ? 'All critical services are healthy'
                      : status == 'warn'
                          ? 'Some services are degraded'
                          : 'Health checks are failing',
                  style: TextStyle(
                    color: colors.textPrimary,
                    fontSize: 16,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 6),
                Text(
                  '$passCount of ${checks.length} checks passed. Last refresh: $timestamp',
                  style: TextStyle(color: colors.textSecondary, fontSize: 12),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _HealthCheckCard extends StatelessWidget {
  final Map<String, dynamic> check;

  const _HealthCheckCard({required this.check});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    final status = check['status']?.toString() ?? 'fail';
    final ping = check['ping'];
    final error = check['error']?.toString();
    final statusColor = _statusColor(status);
    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: colors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 10,
                height: 10,
                decoration: BoxDecoration(
                  color: statusColor,
                  shape: BoxShape.circle,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  check['label']?.toString() ?? 'Unknown',
                  style: TextStyle(
                    color: colors.textPrimary,
                    fontSize: 15,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
              _HealthStatusPill(status: status),
            ],
          ),
          const SizedBox(height: 12),
          _HealthMetric(label: 'Endpoint', value: check['path']?.toString() ?? '-'),
          _HealthMetric(label: 'Ping', value: '${ping ?? 0} ms'),
          if (error != null && error.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: _red.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  error,
                  style: const TextStyle(
                    color: Color(0xFFFFB4B4),
                    fontSize: 12,
                    height: 1.45,
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _HealthMetric extends StatelessWidget {
  final String label;
  final String value;

  const _HealthMetric({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    final colors = consoleColors(context);
    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Row(
        children: [
          SizedBox(
            width: 72,
            child: Text(
              label,
              style: TextStyle(color: colors.textSubtle, fontSize: 12),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: TextStyle(color: colors.textSecondary, fontSize: 12),
            ),
          ),
        ],
      ),
    );
  }
}

class _HealthStatusPill extends StatelessWidget {
  final String status;

  const _HealthStatusPill({required this.status});

  @override
  Widget build(BuildContext context) {
    final color = _statusColor(status);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        status.toUpperCase(),
        style: TextStyle(color: color, fontSize: 11, fontWeight: FontWeight.w700),
      ),
    );
  }
}

Color _statusColor(String status) {
  switch (status) {
    case 'pass':
      return _green;
    case 'warn':
      return _amber;
    default:
      return _red;
  }
}