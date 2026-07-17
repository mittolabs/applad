import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/theme/console_colors.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/app_error_state.dart';
import 'observe_providers.dart';
import 'observe_shared.dart';

class ObAlertsTab extends ConsumerStatefulWidget {
  final String projectId;
  const ObAlertsTab({super.key, required this.projectId});
  @override
  ConsumerState<ObAlertsTab> createState() => _ObAlertsTabState();
}

class _ObAlertsTabState extends ConsumerState<ObAlertsTab> {
  final _search = TextEditingController();

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(alertsProvider);

    return async.when(
      loading: () =>
          const Center(child: CircularProgressIndicator(strokeWidth: 2)),
      error: (e, _) => AppErrorState(
          error: e, onRetry: () => ref.invalidate(alertsProvider)),
      data: (data) {
        var rules =
            List<Map<String, dynamic>>.from(data['rules'] ?? []);
        final incidents =
            List<Map<String, dynamic>>.from(data['incidents'] ?? []);

        final q = _search.text.trim().toLowerCase();
        if (q.isNotEmpty) {
          rules = rules
              .where((r) =>
                  (r['name'] as String? ?? '').toLowerCase().contains(q) ||
                  (r['metric'] as String? ?? '').toLowerCase().contains(q))
              .toList();
        }

        return Column(children: [
          // Firing incidents banner (above table, full width)
          if (incidents.isNotEmpty)
            Padding(
              padding: EdgeInsets.fromLTRB(
                  pageHPad(context), 0, pageHPad(context), 12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(children: [
                    obSectionTitle('Firing Now', consoleColors(context)),
                    const SizedBox(width: 8),
                    _FiringBadge(incidents.length),
                  ]),
                  const SizedBox(height: 8),
                  ...incidents.map((inc) => Padding(
                        padding: const EdgeInsets.only(bottom: 6),
                        child: _IncidentCard(incident: inc),
                      )),
                ],
              ),
            ),

          Expanded(
            child: Padding(
              padding:
                  EdgeInsets.symmetric(horizontal: pageHPad(context)),
              child: AppDataTable(
                persistKey: 'ob_alerts',
                columns: const [
                  AppTableColumn(
                      key: 'severity',
                      label: 'Severity',
                      flex: 2,
                      sortable: false),
                  AppTableColumn(key: 'name', label: 'Name', flex: 3),
                  AppTableColumn(
                      key: 'condition', label: 'Condition', flex: 4, sortable: false),
                  AppTableColumn(
                      key: 'time_window', label: 'Window', flex: 2),
                  AppTableColumn(
                      key: 'channel', label: 'Channel', flex: 2),
                  AppTableColumn(
                      key: 'enabled',
                      label: 'Enabled',
                      flex: 1,
                      sortable: false),
                  AppTableColumn(
                      key: 'lastFired', label: 'Last fired', flex: 2),
                ],
                rows: rules,
                getCellValue: (row, key) => switch (key) {
                  'severity'    => row['severity'] as String? ?? 'warning',
                  'name'        => row['name'] as String? ?? '',
                  'condition'   => _conditionStr(row),
                  'time_window' => row['time_window'] as String? ?? row['window'] as String? ?? '',
                  'channel'     => row['channel'] as String? ?? '',
                  'enabled'     => (row['enabled'] != false).toString(),
                  'lastFired'   => obTimeAgo(row['lastFired']),
                  _             => '',
                },
                cellBuilder: (row, key) {
                  if (key == 'severity') {
                    final s = row['severity'] as String? ?? 'warning';
                    final c = switch (s) {
                      'critical' => obRed,
                      'warning'  => obOrange,
                      _          => obAccent,
                    };
                    return Row(mainAxisSize: MainAxisSize.min, children: [
                      Container(
                          width: 8,
                          height: 8,
                          decoration:
                              BoxDecoration(color: c, shape: BoxShape.circle)),
                      const SizedBox(width: 6),
                      Text(s[0].toUpperCase() + s.substring(1),
                          style: TextStyle(
                              color: c,
                              fontSize: 12,
                              fontWeight: FontWeight.w500)),
                    ]);
                  }
                  if (key == 'enabled') {
                    return Switch(
                      value: row['enabled'] != false,
                      onChanged: (_) async {
                        await ref
                            .read(apiClientProvider)
                            .patch('/observe/alerts/${row[r'$id']}/toggle');
                        ref.invalidate(alertsProvider);
                      },
                      activeThumbColor: obAccent,
                    );
                  }
                  return null;
                },
                getRowIcon: (row) => LucideIcons.bell,
                getRowIconColor: (row) => switch (
                    row['severity'] as String? ?? 'warning') {
                  'critical' => obRed,
                  'warning'  => obOrange,
                  _          => obAccent,
                },
                onDeleteRow: (row) async {
                  await ref
                      .read(apiClientProvider)
                      .delete('/observe/alerts/${row[r'$id']}');
                  ref.invalidate(alertsProvider);
                },
                createLabel: 'Create rule',
                onCreateTap: () => _showCreateDialog(context),
                filters: const [
                  AppTableFilter(
                      key: 'severity',
                      label: 'Severity',
                      options: ['info', 'warning', 'critical']),
                ],
                total: rules.length,
                perPage: rules.isEmpty ? 12 : rules.length,
                currentPage: 1,
                onPrev: () {},
                onNext: () {},
                onPerPageChanged: (_) {},
                itemLabel: 'rules',
                searchController: _search,
                onSearch: () => setState(() {}),
                searchHint: 'Search rules…',
                emptyIcon: LucideIcons.bell,
                emptyTitle: 'No alert rules',
                emptySubtitle:
                    'Create rules to get notified when metrics exceed thresholds',
              ),
            ),
          ),
        ]);
      },
    );
  }

  static String _conditionStr(Map<String, dynamic> row) {
    final metric = row['metric'] as String? ?? '';
    final op     = row['operator'] as String? ?? 'gt';
    final thresh = row['threshold'];
    final opStr  = switch (op) {
      'gt' => '>', 'lt' => '<', 'gte' => '≥', 'lte' => '≤', _ => op,
    };
    return '$metric $opStr $thresh';
  }

  void _showCreateDialog(BuildContext context) {
    final nameCtrl      = TextEditingController();
    final thresholdCtrl = TextEditingController(text: '5');
    String metric   = 'error_rate';
    String operator = 'gt';
    String window   = '5m';
    String severity = 'warning';
    String channel  = 'email';

    showAppDialog(
      context: context,
      title: 'Create alert rule',
      width: 440,
      content: StatefulBuilder(
        builder: (ctx, ss) => Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AppDialogField(
                controller: nameCtrl,
                label: 'Rule name',
                hint: 'High error rate'),
            const SizedBox(height: 12),
            ObDialogDropdown(
              label: 'Metric',
              value: metric,
              items: const [
                'error_rate', 'p95_latency', 'p99_latency',
                'request_rate', 'uptime', 'log_errors',
                'lcp', 'fid', 'cls', 'apdex',
              ],
              display: (v) => switch (v) {
                'error_rate'   => 'Error rate (%)',
                'p95_latency'  => 'P95 latency (ms)',
                'p99_latency'  => 'P99 latency (ms)',
                'request_rate' => 'Request rate (req/s)',
                'uptime'       => 'Uptime (%)',
                'log_errors'   => 'Log errors / min',
                'lcp'          => 'LCP (ms)',
                'fid'          => 'FID (ms)',
                'cls'          => 'CLS score',
                'apdex'        => 'Apdex score',
                _              => v,
              },
              onChanged: (v) => ss(() => metric = v),
            ),
            const SizedBox(height: 12),
            Row(children: [
              Expanded(
                child: ObDialogDropdown(
                  label: 'Condition',
                  value: operator,
                  items: const ['gt', 'lt', 'gte', 'lte'],
                  display: (v) => switch (v) {
                    'gt'  => 'Above',
                    'lt'  => 'Below',
                    'gte' => 'At or above',
                    'lte' => 'At or below',
                    _     => v,
                  },
                  onChanged: (v) => ss(() => operator = v),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: AppDialogField(
                    controller: thresholdCtrl,
                    label: 'Threshold',
                    hint: '5',
                    keyboardType: TextInputType.number),
              ),
            ]),
            const SizedBox(height: 12),
            ObDialogDropdown(
              label: 'Time window',
              value: window,
              items: const ['1m', '5m', '15m', '30m', '1h', '24h'],
              display: (v) => switch (v) {
                '1m'  => '1 minute',
                '5m'  => '5 minutes',
                '15m' => '15 minutes',
                '30m' => '30 minutes',
                '1h'  => '1 hour',
                '24h' => '24 hours',
                _     => v,
              },
              onChanged: (v) => ss(() => window = v),
            ),
            const SizedBox(height: 12),
            ObDialogDropdown(
              label: 'Severity',
              value: severity,
              items: const ['info', 'warning', 'critical'],
              display: (v) => v[0].toUpperCase() + v.substring(1),
              onChanged: (v) => ss(() => severity = v),
            ),
            const SizedBox(height: 12),
            ObDialogDropdown(
              label: 'Notify via',
              value: channel,
              items: const [
                'email', 'slack', 'webhook', 'pagerduty', 'opsgenie'
              ],
              display: (v) => switch (v) {
                'email'     => 'Email',
                'slack'     => 'Slack',
                'webhook'   => 'Webhook',
                'pagerduty' => 'PagerDuty',
                'opsgenie'  => 'Opsgenie',
                _           => v,
              },
              onChanged: (v) => ss(() => channel = v),
            ),
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Create',
          onTap: () async {
            await ref.read(apiClientProvider).post('/observe/alerts', data: {
              'name':      nameCtrl.text.trim(),
              'metric':    metric,
              'operator':  operator,
              'threshold': double.tryParse(thresholdCtrl.text) ?? 5.0,
              'window':    window,
              'severity':  severity,
              'channel':   channel,
              'enabled':   true,
            });
            if (context.mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
            ref.invalidate(alertsProvider);
          },
        ),
      ],
    );
  }
}

// ── Incident card ─────────────────────────────────────────────────────────────

class _IncidentCard extends StatelessWidget {
  final Map<String, dynamic> incident;
  const _IncidentCard({required this.incident});

  @override
  Widget build(BuildContext context) {
    final colors   = consoleColors(context);
    final severity = incident['severity'] as String? ?? 'warning';
    final name     = incident['ruleName'] as String? ?? 'Alert';
    final value    = incident['value'];
    final firedAt  = obTimeAgo(incident['firedAt']);
    final sc       = switch (severity) {
      'critical' => obRed,
      'warning'  => obOrange,
      _          => obAccent,
    };

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
          color: sc.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: sc.withValues(alpha: 0.3))),
      child: Row(children: [
        Icon(LucideIcons.alertTriangle, size: 16, color: sc),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(name,
                    style: TextStyle(
                        color: colors.textPrimary,
                        fontSize: 13,
                        fontWeight: FontWeight.w600)),
                Text('Value: $value  •  Fired $firedAt',
                    style: TextStyle(
                        color: colors.textSecondary, fontSize: 11)),
              ]),
        ),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
          decoration: BoxDecoration(
              color: sc.withValues(alpha: 0.15),
              borderRadius: BorderRadius.circular(4)),
          child: Text(severity.toUpperCase(),
              style: TextStyle(
                  color: sc,
                  fontSize: 10,
                  fontWeight: FontWeight.w700)),
        ),
      ]),
    );
  }
}

// ── Firing badge ──────────────────────────────────────────────────────────────

class _FiringBadge extends StatelessWidget {
  final int count;
  const _FiringBadge(this.count);

  @override
  Widget build(BuildContext context) => Container(
        padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
        decoration: BoxDecoration(
            color: obRed, borderRadius: BorderRadius.circular(10)),
        child: Text('$count',
            style: const TextStyle(
                color: Colors.white,
                fontSize: 11,
                fontWeight: FontWeight.w700)),
      );
}
