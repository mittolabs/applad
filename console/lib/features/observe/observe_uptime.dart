import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../core/api/client.dart';
import '../../core/utils/url_utils.dart';
import '../../core/widgets/app_data_table.dart';
import '../../core/widgets/app_dialog.dart';
import '../../core/widgets/app_error_state.dart';
import 'observe_providers.dart';
import 'observe_shared.dart';

class ObUptimeTab extends ConsumerStatefulWidget {
  final String projectId;
  const ObUptimeTab({super.key, required this.projectId});
  @override
  ConsumerState<ObUptimeTab> createState() => _ObUptimeTabState();
}

class _ObUptimeTabState extends ConsumerState<ObUptimeTab> {
  final _search = TextEditingController();

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(uptimeProvider);

    return Padding(
      padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
      child: async.when(
        loading: () =>
            const Center(child: CircularProgressIndicator(strokeWidth: 2)),
        error: (e, _) => AppErrorState(
            error: e, onRetry: () => ref.invalidate(uptimeProvider)),
        data: (data) {
          var monitors =
              List<Map<String, dynamic>>.from(data['monitors'] ?? []);
          final q = _search.text.trim().toLowerCase();
          if (q.isNotEmpty) {
            monitors = monitors
                .where((m) =>
                    (m['name'] as String? ?? '').toLowerCase().contains(q) ||
                    (m['url'] as String? ?? '').toLowerCase().contains(q))
                .toList();
          }
          return AppDataTable(
            persistKey: 'ob_uptime',
            columns: const [
              AppTableColumn(
                  key: 'status', label: 'Status', flex: 2, sortable: false),
              AppTableColumn(key: 'name', label: 'Name', flex: 3),
              AppTableColumn(key: 'url', label: 'URL', flex: 4),
              AppTableColumn(key: 'uptime', label: 'Uptime', flex: 2),
              AppTableColumn(key: 'latency', label: 'Latency', flex: 2),
              AppTableColumn(key: 'checked', label: 'Checked', flex: 2),
            ],
            rows: monitors,
            getCellValue: (row, key) => switch (key) {
              'status'  => row['status'] as String? ?? 'up',
              'name'    => row['name'] as String? ?? '',
              'url'     => row['url'] as String? ?? '',
              'uptime'  =>
                '${(row['uptimePct'] as num? ?? 100.0).toStringAsFixed(2)}%',
              'latency' => '${row['latencyMs'] ?? 0}ms',
              'checked' => obTimeAgo(row['lastChecked']),
              _         => '',
            },
            cellBuilder: (row, key) {
              if (key == 'status') {
                final s = row['status'] as String? ?? 'up';
                final c = switch (s) {
                  'up'       => obGreen,
                  'down'     => obRed,
                  'degraded' => obOrange,
                  _          => const Color(0xFF64748B),
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
              if (key == 'uptime') {
                final pct =
                    (row['uptimePct'] as num? ?? 100.0).toDouble();
                final c = pct >= 99.9
                    ? obGreen
                    : pct >= 99.0
                        ? obOrange
                        : obRed;
                return Text('${pct.toStringAsFixed(2)}%',
                    style: TextStyle(
                        color: c,
                        fontSize: 13,
                        fontWeight: FontWeight.w600));
              }
              return null;
            },
            getRowIcon: (_) => LucideIcons.heartPulse,
            getRowIconColor: (row) => switch (row['status'] as String? ?? 'up') {
              'up'       => obGreen,
              'down'     => obRed,
              'degraded' => obOrange,
              _          => const Color(0xFF64748B),
            },
            onDeleteRow: (row) async {
              await ref
                  .read(apiClientProvider)
                  .delete('/observe/uptime/${row[r'$id']}');
              ref.invalidate(uptimeProvider);
            },
            createLabel: 'Add monitor',
            onCreateTap: () => _showAddDialog(context),
            filters: const [
              AppTableFilter(
                  key: 'status',
                  label: 'Status',
                  options: ['up', 'down', 'degraded', 'paused']),
            ],
            total: monitors.length,
            perPage: monitors.isEmpty ? 12 : monitors.length,
            currentPage: 1,
            onPrev: () {},
            onNext: () {},
            onPerPageChanged: (_) {},
            itemLabel: 'monitors',
            searchController: _search,
            onSearch: () => setState(() {}),
            searchHint: 'Search monitors…',
            emptyIcon: LucideIcons.heartPulse,
            emptyTitle: 'No uptime monitors',
            emptySubtitle: 'Track availability of your services and endpoints',
          );
        },
      ),
    );
  }

  void _showAddDialog(BuildContext context) {
    final nameCtrl = TextEditingController();
    final urlCtrl  = TextEditingController();
    String checkType    = 'http';
    String intervalSecs = '60';

    showAppDialog(
      context: context,
      title: 'Add uptime monitor',
      width: 420,
      content: StatefulBuilder(
        builder: (ctx, ss) => Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AppDialogField(
                controller: nameCtrl,
                label: 'Name',
                hint: 'API health'),
            const SizedBox(height: 12),
            AppDialogField(
                controller: urlCtrl,
                label: 'URL or host',
                hint: 'https://api.example.com/health'),
            const SizedBox(height: 12),
            ObDialogDropdown(
              label: 'Check type',
              value: checkType,
              items: const ['http', 'tcp', 'ping', 'keyword'],
              display: (v) => switch (v) {
                'http'    => 'HTTP(S)',
                'tcp'     => 'TCP port',
                'ping'    => 'ICMP ping',
                'keyword' => 'Keyword match',
                _         => v,
              },
              onChanged: (v) => ss(() => checkType = v),
            ),
            const SizedBox(height: 12),
            ObDialogDropdown(
              label: 'Check interval',
              value: intervalSecs,
              items: const ['30', '60', '300', '600', '1800'],
              display: (v) => switch (v) {
                '30'   => '30 seconds',
                '60'   => '1 minute',
                '300'  => '5 minutes',
                '600'  => '10 minutes',
                '1800' => '30 minutes',
                _      => '$v seconds',
              },
              onChanged: (v) => ss(() => intervalSecs = v),
            ),
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Add',
          onTap: () async {
            await ref.read(apiClientProvider).post('/observe/uptime', data: {
              'name':         nameCtrl.text.trim(),
              'url':          urlCtrl.text.trim(),
              'checkType':    checkType,
              'intervalSecs': int.tryParse(intervalSecs) ?? 60,
            });
            if (context.mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
            ref.invalidate(uptimeProvider);
          },
        ),
      ],
    );
  }
}
