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

class ObCronsTab extends ConsumerStatefulWidget {
  final String projectId;
  const ObCronsTab({super.key, required this.projectId});
  @override
  ConsumerState<ObCronsTab> createState() => _ObCronsTabState();
}

class _ObCronsTabState extends ConsumerState<ObCronsTab> {
  final _search = TextEditingController();

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(cronsProvider);

    return Padding(
      padding: EdgeInsets.symmetric(horizontal: pageHPad(context)),
      child: async.when(
        loading: () =>
            const Center(child: CircularProgressIndicator(strokeWidth: 2)),
        error: (e, _) => AppErrorState(
            error: e, onRetry: () => ref.invalidate(cronsProvider)),
        data: (data) {
          var monitors =
              List<Map<String, dynamic>>.from(data['monitors'] ?? []);
          final q = _search.text.trim().toLowerCase();
          if (q.isNotEmpty) {
            monitors = monitors
                .where((m) =>
                    (m['name'] as String? ?? '').toLowerCase().contains(q) ||
                    (m['schedule'] as String? ?? '').toLowerCase().contains(q))
                .toList();
          }
          return AppDataTable(
            persistKey: 'ob_crons',
            columns: const [
              AppTableColumn(
                  key: 'status', label: 'Status', flex: 2, sortable: false),
              AppTableColumn(key: 'name', label: 'Name', flex: 3),
              AppTableColumn(
                  key: 'schedule', label: 'Schedule', flex: 3, sortable: false),
              AppTableColumn(key: 'timezone', label: 'Timezone', flex: 2),
              AppTableColumn(key: 'lastRun', label: 'Last run', flex: 2),
              AppTableColumn(key: 'nextRun', label: 'Next run', flex: 2),
              AppTableColumn(
                  key: 'enabled', label: 'Enabled', flex: 1, sortable: false),
            ],
            rows: monitors,
            getCellValue: (row, key) => switch (key) {
              'status'   => row['status'] as String? ?? 'waiting',
              'name'     => row['name'] as String? ?? '',
              'schedule' => row['schedule'] as String? ?? '',
              'timezone' => row['timezone'] as String? ?? 'UTC',
              'lastRun'  => obTimeAgo(row['lastRunAt']),
              'nextRun'  => obTimeAgo(row['nextRunAt']),
              'enabled'  => (row['enabled'] != false).toString(),
              _          => '',
            },
            cellBuilder: (row, key) {
              if (key == 'status') {
                final s = row['status'] as String? ?? 'waiting';
                final (c, icon) = switch (s) {
                  'ok'      => (obGreen,  LucideIcons.checkCircle2),
                  'missed'  => (obOrange, LucideIcons.clock),
                  'failed'  => (obRed,    LucideIcons.xCircle),
                  'running' => (obAccent, LucideIcons.loader),
                  _         => (const Color(0xFF64748B), LucideIcons.timer),
                };
                return Row(mainAxisSize: MainAxisSize.min, children: [
                  Icon(icon, size: 12, color: c),
                  const SizedBox(width: 6),
                  Text(s[0].toUpperCase() + s.substring(1),
                      style: TextStyle(
                          color: c,
                          fontSize: 12,
                          fontWeight: FontWeight.w500)),
                ]);
              }
              if (key == 'schedule') {
                final s = row['schedule'] as String? ?? '';
                return Text(s,
                    style: const TextStyle(
                        color: obAccent,
                        fontSize: 12,
                        fontFamily: 'monospace',
                        fontWeight: FontWeight.w500));
              }
              if (key == 'enabled') {
                return Switch(
                  value: row['enabled'] != false,
                  onChanged: (_) async {
                    await ref
                        .read(apiClientProvider)
                        .patch('/observe/crons/${row[r'$id']}/toggle');
                    ref.invalidate(cronsProvider);
                  },
                  activeThumbColor: obAccent,
                );
              }
              return null;
            },
            getRowIcon: (_) => LucideIcons.clock,
            getRowIconColor: (row) => switch (row['status'] as String? ?? 'waiting') {
              'ok'      => obGreen,
              'missed'  => obOrange,
              'failed'  => obRed,
              'running' => obAccent,
              _         => const Color(0xFF64748B),
            },
            onDeleteRow: (row) async {
              await ref
                  .read(apiClientProvider)
                  .delete('/observe/crons/${row[r'$id']}');
              ref.invalidate(cronsProvider);
            },
            createLabel: 'Add monitor',
            onCreateTap: () => _showAddDialog(context),
            filters: const [
              AppTableFilter(
                  key: 'status',
                  label: 'Status',
                  options: ['ok', 'missed', 'failed', 'running', 'waiting']),
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
            emptyIcon: LucideIcons.clock,
            emptyTitle: 'No cron monitors',
            emptySubtitle:
                'Get alerted when scheduled jobs miss their execution window',
          );
        },
      ),
    );
  }

  void _showAddDialog(BuildContext context) {
    final nameCtrl     = TextEditingController();
    final scheduleCtrl = TextEditingController(text: '0 * * * *');
    String timezone  = 'UTC';
    int gracePeriod  = 5;

    showAppDialog(
      context: context,
      title: 'Add cron monitor',
      width: 440,
      content: StatefulBuilder(
        builder: (ctx, ss) => Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AppDialogField(
                controller: nameCtrl,
                label: 'Name',
                hint: 'Daily backup job'),
            const SizedBox(height: 12),
            AppDialogField(
                controller: scheduleCtrl,
                label: 'Schedule (cron expression)',
                hint: '0 * * * *'),
            const SizedBox(height: 4),
            const Text(
                'Use standard cron format — e.g. "0 9 * * 1-5" for weekdays at 9am',
                style: TextStyle(color: Color(0xFF64748B), fontSize: 11)),
            const SizedBox(height: 12),
            ObDialogDropdown(
              label: 'Timezone',
              value: timezone,
              items: const [
                'UTC', 'America/New_York', 'America/Los_Angeles',
                'Europe/London', 'Europe/Berlin', 'Asia/Tokyo',
                'Asia/Singapore', 'Australia/Sydney',
              ],
              onChanged: (v) => ss(() => timezone = v),
            ),
            const SizedBox(height: 12),
            ObDialogDropdown(
              label: 'Grace period (minutes)',
              value: '$gracePeriod',
              items: const ['1', '5', '10', '30', '60'],
              display: (v) => '$v minutes',
              onChanged: (v) => ss(() => gracePeriod = int.tryParse(v) ?? 5),
            ),
          ],
        ),
      ),
      actions: [
        const AppDialogCancel(),
        AppDialogAction(
          label: 'Add',
          onTap: () async {
            await ref.read(apiClientProvider).post('/observe/crons', data: {
              'name':        nameCtrl.text.trim(),
              'schedule':    scheduleCtrl.text.trim(),
              'timezone':    timezone,
              'gracePeriod': gracePeriod,
            });
            if (context.mounted) {
              Navigator.of(context, rootNavigator: true).pop();
            }
            ref.invalidate(cronsProvider);
          },
        ),
      ],
    );
  }
}
